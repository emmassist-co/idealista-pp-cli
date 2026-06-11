package cli

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"idealista-pp-cli/internal/store"
)

func TestSearchCommand_LiveGenericSearchRefuses(t *testing.T) {
	_, stderr, err := executeCookieCommand(t, "--json", "--data-source", "live", "search", "lisboa")
	if err == nil {
		t.Fatalf("expected generic live search to fail")
	}
	if got := ExitCode(err); got != 2 {
		t.Fatalf("ExitCode = %d, want 2", got)
	}
	if stderr == "" {
		t.Fatalf("expected usage guidance on stderr")
	}
}

func TestSearchLocalSubcommandUsesSQLiteFTS(t *testing.T) {
	path := filepath.Join(t.TempDir(), "search.db")
	db, err := store.OpenWithContext(context.Background(), path)
	if err != nil {
		t.Fatalf("OpenWithContext: %v", err)
	}
	defer db.Close()
	if err := db.Upsert("homes", "listing-1", json.RawMessage(`{"id":"listing-1","name":"Lisboa apartment","status":"active"}`)); err != nil {
		t.Fatalf("Upsert: %v", err)
	}

	stdout, _, err := executeCookieCommand(t, "--json", "search", "local", "Lisboa", "--db", path)
	if err != nil {
		t.Fatalf("search local: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}
	meta, _ := got["meta"].(map[string]any)
	if meta["source"] != "local" {
		t.Fatalf("meta.source = %v, want local", meta["source"])
	}
	results, _ := got["results"].([]any)
	if len(results) != 1 {
		t.Fatalf("results len = %d, want 1", len(results))
	}
}

func TestSummarizeSavedSearchPayload(t *testing.T) {
	raw := json.RawMessage(`{
		"body": {
			"lastSearchName": "Casas e apartamentos em Lisboa",
			"searchUrl": "/comprar-casas/lisboa/",
			"mapUrl": "/mapa/comprar-casas/lisboa/",
			"totalAds": "10.103",
			"canBeSaved": true
		}
	}`)
	got, err := summarizeSavedSearchPayload(raw)
	if err != nil {
		t.Fatalf("summarizeSavedSearchPayload: %v", err)
	}
	if got["last_search_name"] != "Casas e apartamentos em Lisboa" {
		t.Fatalf("last_search_name = %v", got["last_search_name"])
	}
	if got["search_url"] != "/comprar-casas/lisboa/" {
		t.Fatalf("search_url = %v", got["search_url"])
	}
	if got["total_ads"] != "10.103" {
		t.Fatalf("total_ads = %v", got["total_ads"])
	}
	if got["can_be_saved"] != true {
		t.Fatalf("can_be_saved = %v", got["can_be_saved"])
	}
}

func TestBuildSearchResultsRelativeURL_FromFixtures(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("testdata", "site_search_filters.json"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	var fixtures []struct {
		Name    string            `json:"name"`
		Spec    searchResultsSpec `json:"spec"`
		WantURL string            `json:"want_url"`
	}
	if err := json.Unmarshal(data, &fixtures); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}
	for _, fixture := range fixtures {
		fixture := fixture
		t.Run(fixture.Name, func(t *testing.T) {
			got, err := buildSearchResultsRelativeURL(fixture.Spec)
			if err != nil {
				t.Fatalf("buildSearchResultsRelativeURL: %v", err)
			}
			if got != fixture.WantURL {
				t.Fatalf("URL = %q, want %q", got, fixture.WantURL)
			}
		})
	}
}

func TestBuildSearchResultsState_CombinedRangesPopulateGeoReach(t *testing.T) {
	got, err := buildSearchResultsState(searchResultsSpec{
		LocationPath: "comprar-casas/lisboa/arroios",
		MinPrice:     220000,
		MaxPrice:     750000,
		MinSize:      60,
		MaxSize:      120,
		Bedrooms:     []string{"t2", "t4-t5"},
	}, "https://www.idealista.pt")
	if err != nil {
		t.Fatalf("buildSearchResultsState: %v", err)
	}
	if got.GeoReachPath != "/pt/ajax/listing/georeach/lisboa/arroios" {
		t.Fatalf("GeoReachPath = %q", got.GeoReachPath)
	}
	if got.GeoReachQuery["minPrice"] != "220000" || got.GeoReachQuery["maxPrice"] != "750000" {
		t.Fatalf("georeach price query = %#v", got.GeoReachQuery)
	}
	if got.GeoReachQuery["minArea"] != "60" || got.GeoReachQuery["maxArea"] != "120" {
		t.Fatalf("georeach area query = %#v", got.GeoReachQuery)
	}
	if got.GeoReachQuery["roomsTwo"] != "true" || got.GeoReachQuery["roomsFourOrMore"] != "true" {
		t.Fatalf("georeach room query = %#v", got.GeoReachQuery)
	}
	if got.ListingAjaxPath != "/ajax/listingcontroller/listingajax.ajax" {
		t.Fatalf("ListingAjaxPath = %q", got.ListingAjaxPath)
	}
	if got.TotalsAjaxPath != "/ajax/listingcontroller/totals/listingajax.ajax" {
		t.Fatalf("TotalsAjaxPath = %q", got.TotalsAjaxPath)
	}
	if got.ListingAjaxQuery["locationUri"] != "lisboa/arroios" {
		t.Fatalf("locationUri = %q", got.ListingAjaxQuery["locationUri"])
	}
	if got.ListingAjaxQuery["adfilter_pricemin"] != "220000" || got.ListingAjaxQuery["adfilter_price"] != "750000" {
		t.Fatalf("listingajax price query = %#v", got.ListingAjaxQuery)
	}
	if got.ListingAjaxQuery["adfilter_area"] != "60" || got.ListingAjaxQuery["adfilter_areamax"] != "120" {
		t.Fatalf("listingajax area query = %#v", got.ListingAjaxQuery)
	}
	if got.ListingAjaxQuery["adfilter_rooms_2"] != "2" || got.ListingAjaxQuery["adfilter_rooms_4_more"] != "4" {
		t.Fatalf("listingajax room query = %#v", got.ListingAjaxQuery)
	}
}

func TestBuildSearchResultsState_ListingAjaxMapsFilters(t *testing.T) {
	got, err := buildSearchResultsState(searchResultsSpec{
		LocationPath:    "comprar-casas/lisboa/arroios",
		MaxPrice:        300000,
		Bedrooms:        []string{"t1", "t2", "t3"},
		Bathrooms:       []int{1, 2},
		Amenities:       []string{"elevador", "garagem", "arrecadacao", "arcondicionado", "roupeiros-embutidos", "vista-mar"},
		EnergyClass:     "alta",
		PublishedWithin: "month",
		Sort:            "precos-asc",
	}, "https://www.idealista.pt")
	if err != nil {
		t.Fatalf("buildSearchResultsState: %v", err)
	}
	if got.ListingAjaxQuery["adfilter_price"] != "300000" {
		t.Fatalf("adfilter_price = %q", got.ListingAjaxQuery["adfilter_price"])
	}
	if got.ListingAjaxQuery["adfilter_rooms_1"] != "1" || got.ListingAjaxQuery["adfilter_rooms_2"] != "2" || got.ListingAjaxQuery["adfilter_rooms_3"] != "3" {
		t.Fatalf("listingajax rooms = %#v", got.ListingAjaxQuery)
	}
	if got.ListingAjaxQuery["adfilter_baths_1"] != "1" || got.ListingAjaxQuery["adfilter_baths_2"] != "2" {
		t.Fatalf("listingajax baths = %#v", got.ListingAjaxQuery)
	}
	if got.ListingAjaxQuery["adfilter_lift"] != "1" ||
		got.ListingAjaxQuery["adfilter_parkingspace"] != "1" ||
		got.ListingAjaxQuery["adfilter_boxroom"] != "1" ||
		got.ListingAjaxQuery["adfilter_hasairconditioning"] != "1" ||
		got.ListingAjaxQuery["adfilter_wardrobes"] != "1" ||
		got.ListingAjaxQuery["adfilter_seaviews"] != "1" {
		t.Fatalf("listingajax amenities = %#v", got.ListingAjaxQuery)
	}
	if got.ListingAjaxQuery["adfilter_energyCertificateHigh"] != "1" {
		t.Fatalf("energy mapping = %#v", got.ListingAjaxQuery)
	}
	if got.ListingAjaxQuery["adfilter_published"] != "3" {
		t.Fatalf("published mapping = %#v", got.ListingAjaxQuery)
	}
	if got.ListingAjaxQuery["ordem"] != "precos-asc" {
		t.Fatalf("ordem = %q", got.ListingAjaxQuery["ordem"])
	}
	if got.TotalsAjaxQuery["adfilter_price"] != "300000" {
		t.Fatalf("totals query mismatch = %#v", got.TotalsAjaxQuery)
	}
}

func TestParseListingCards(t *testing.T) {
	body := []byte(`
<article class="item">
  <a class="item-link" href="/imovel/11111111/" title="Apartamento T2 em Arroios">Apartamento T2 em Arroios</a>
  <span class="item-price h2-simulated">295.000 €</span>
  <span class="item-price-down">T2 67 m²</span>
  <span class="item-detail-location">Rua do Forno do Tijolo</span>
  <p class="item-description">Bom estado, perto do metro.</p>
  <span class="item-detail">T2</span>
  <span class="item-detail">67 m²</span>
  <img src="https://img4.idealista.pt/a.jpg" />
</article>
<article class="item">
  <a class="item-link" href="/imovel/22222222/">Apartamento T3 em Penha de Franca</a>
  <span class="item-price">300.000 €</span>
</article>`)

	got := parseListingCards(body)
	if len(got) != 2 {
		t.Fatalf("len(parseListingCards) = %d", len(got))
	}
	if got[0].ListingID != "11111111" {
		t.Fatalf("ListingID = %q", got[0].ListingID)
	}
	if got[0].Title != "Apartamento T2 em Arroios" {
		t.Fatalf("Title = %q", got[0].Title)
	}
	if got[0].Price != "295.000 €" {
		t.Fatalf("Price = %q", got[0].Price)
	}
	if got[0].PrimaryImageURL != "https://img4.idealista.pt/a.jpg" {
		t.Fatalf("PrimaryImageURL = %q", got[0].PrimaryImageURL)
	}
	if got[1].ListingID != "22222222" {
		t.Fatalf("second ListingID = %q", got[1].ListingID)
	}
}

func TestWhichSearchResultsEnrichedQueryMatches(t *testing.T) {
	matches := rankWhich(whichIndex, "results-enriched", 3)
	if len(matches) == 0 {
		t.Fatalf("expected enriched search match")
	}
	if matches[0].Entry.Command != "search results-enriched" {
		t.Fatalf("top match = %q, want search results-enriched", matches[0].Entry.Command)
	}
}

func TestBuildSearchResultsRelativeURL_RejectsInvalidRangeOrder(t *testing.T) {
	_, err := buildSearchResultsRelativeURL(searchResultsSpec{
		LocationPath: "comprar-casas/lisboa/arroios",
		MinPrice:     750000,
		MaxPrice:     220000,
	})
	if err == nil {
		t.Fatalf("expected invalid price range to fail")
	}
}

func TestBuildSearchResultsRelativeURL_RejectsUnsupportedAmenity(t *testing.T) {
	_, err := buildSearchResultsRelativeURL(searchResultsSpec{
		LocationPath: "comprar-casas/lisboa/arroios",
		Amenities:    []string{"piscina"},
	})
	if err == nil {
		t.Fatalf("expected unsupported amenity to fail")
	}
}

func TestWhichListingPhotosQueryMatches(t *testing.T) {
	matches := rankWhich(whichIndex, "listing photos", 3)
	if len(matches) == 0 {
		t.Fatalf("expected listing photos match")
	}
	if matches[0].Entry.Command != "listing photos" {
		t.Fatalf("top match = %q, want listing photos", matches[0].Entry.Command)
	}
}

func TestPtListPagerRequiresAdID(t *testing.T) {
	_, _, err := executeCookieCommand(t, "pt", "list-pager")
	if err == nil {
		t.Fatalf("expected missing --ad-id to fail")
	}
	if got := ExitCode(err); got != 2 {
		t.Fatalf("ExitCode = %d, want 2", got)
	}
}

func TestPtListAdContactRequiresAdID(t *testing.T) {
	_, _, err := executeCookieCommand(t, "pt", "list-ad-contact-info-for-detail.ajax")
	if err == nil {
		t.Fatalf("expected missing --ad-id to fail")
	}
	if got := ExitCode(err); got != 2 {
		t.Fatalf("ExitCode = %d, want 2", got)
	}
}

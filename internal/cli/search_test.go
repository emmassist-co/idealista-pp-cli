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

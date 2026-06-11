package cli

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"idealista-pp-cli/internal/client"
	"idealista-pp-cli/internal/config"
)

var supportedBedroomBands = []string{"t0", "t1", "t2", "t3", "t4-t5"}
var supportedBathroomCounts = []int{1, 2, 3}
var supportedSortOrders = []string{"preco_medio-asc", "precos-asc", "atualizado-desc"}

var bedroomToGeoReachParam = map[string]string{
	"t0":    "roomsZero",
	"t1":    "roomsOne",
	"t2":    "roomsTwo",
	"t3":    "roomsThree",
	"t4-t5": "roomsFourOrMore",
}

var supportedAmenityTokens = map[string]string{
	"elevador":            "elevador",
	"garagem":             "garagem",
	"arrecadacao":         "arrecadacao",
	"arcondicionado":      "arcondicionado",
	"roupeiros-embutidos": "roupeiros-embutidos",
	"vista-mar":           "vista-mar",
}

var supportedEnergyClasses = map[string]string{
	"alta":  "eficiencia-energetica-alta",
	"media": "eficiencia-energetica-media",
	"baixa": "eficiencia-energetica-baixa",
}

var supportedPublishedWindows = map[string]string{
	"48h":   "publicado_ultimas-48-horas",
	"week":  "publicado_ultima-semana",
	"month": "publicado_ultimo-mes",
}

type searchResultsSpec struct {
	LocationPath    string   `json:"location_path"`
	MinPrice        int      `json:"min_price,omitempty"`
	MaxPrice        int      `json:"max_price,omitempty"`
	MinSize         int      `json:"min_size,omitempty"`
	MaxSize         int      `json:"max_size,omitempty"`
	Bedrooms        []string `json:"bedrooms,omitempty"`
	Bathrooms       []int    `json:"bathrooms,omitempty"`
	Amenities       []string `json:"amenities,omitempty"`
	EnergyClass     string   `json:"energy_class,omitempty"`
	PublishedWithin string   `json:"published_within,omitempty"`
	Sort            string   `json:"sort,omitempty"`
}

type derivedSearchState struct {
	RelativeURL       string            `json:"relative_url"`
	GeoReachPath      string            `json:"georeach_path"`
	GeoReachURL       string            `json:"georeach_url"`
	GeoReachQuery     map[string]string `json:"georeach_query,omitempty"`
	ListingAjaxPath   string            `json:"listingajax_path,omitempty"`
	ListingAjaxURL    string            `json:"listingajax_url,omitempty"`
	ListingAjaxQuery  map[string]string `json:"listingajax_query,omitempty"`
	TotalsAjaxPath    string            `json:"totalsajax_path,omitempty"`
	TotalsAjaxURL     string            `json:"totalsajax_url,omitempty"`
	TotalsAjaxQuery   map[string]string `json:"totalsajax_query,omitempty"`
	FilterTokens      []string          `json:"filter_tokens,omitempty"`
	OmittedFromURL    []string          `json:"omitted_from_url,omitempty"`
	URLFilterCoverage string            `json:"url_filter_coverage,omitempty"`
}

type liveCheckResult struct {
	ResultsURLStatusCode int    `json:"results_url_status_code"`
	ResultsURLStatus     string `json:"results_url_status"`
	GeoReachStatusCode   int    `json:"georeach_status_code"`
	GeoReachStatus       string `json:"georeach_status"`
}

type normalizedSearchResultsSpec struct {
	LocationPath    string
	LocationURI     string
	MinPrice        int
	MaxPrice        int
	MinSize         int
	MaxSize         int
	Bedrooms        []string
	Bathrooms       []int
	Amenities       []string
	EnergyClass     string
	PublishedWithin string
	Sort            string
}

func newSearchResultsURLCmd(flags *rootFlags) *cobra.Command {
	var locationPath string
	var minPrice int
	var maxPrice int
	var minSize int
	var maxSize int
	var bedrooms []string
	var bathrooms []int
	var amenities []string
	var energyClass string
	var publishedWithin string
	var sortOrder string
	var checkLive bool

	cmd := &cobra.Command{
		Use:   "results-url",
		Short: "Build and validate a website results state for common house filters",
		Long: `Build a canonical Idealista website results URL from the validated filter
subset observed in real browser traffic, and derive the matching georeach
query contract for the parts the website exposes through AJAX.

Supported filters in this round:
  - location path (required)
  - min/max price, including combined ranges
  - min/max size, including combined ranges
  - room bands: t0, t1, t2, t3, t4-t5
  - bathroom counts: 1, 2, 3
  - amenities: elevador, garagem, arrecadacao, arcondicionado,
    roupeiros-embutidos, vista-mar
  - energy class: alta, media, baixa
  - published within: 48h, week, month
  - sort: preco_medio-asc, precos-asc, atualizado-desc

The output includes both the canonical browse URL and the georeach query state
used for live validation of the numeric and room-based filters.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			spec := searchResultsSpec{
				LocationPath:    locationPath,
				MinPrice:        minPrice,
				MaxPrice:        maxPrice,
				MinSize:         minSize,
				MaxSize:         maxSize,
				Bedrooms:        bedrooms,
				Bathrooms:       bathrooms,
				Amenities:       amenities,
				EnergyClass:     energyClass,
				PublishedWithin: publishedWithin,
				Sort:            sortOrder,
			}
			derived, err := buildSearchResultsState(spec, currentBaseURL(flags))
			if err != nil {
				return usageErr(err)
			}
			payload := map[string]any{
				"url":                 strings.TrimRight(currentBaseURL(flags), "/") + derived.RelativeURL,
				"relative_url":        derived.RelativeURL,
				"georeach_url":        derived.GeoReachURL,
				"georeach_path":       derived.GeoReachPath,
				"georeach_query":      derived.GeoReachQuery,
				"filter_tokens":       derived.FilterTokens,
				"omitted_from_url":    derived.OmittedFromURL,
				"url_filter_coverage": derived.URLFilterCoverage,
				"validated_spec":      spec,
			}
			if dryRunOK(flags) {
				payload["dry_run"] = true
				return printJSONFiltered(cmd.OutOrStdout(), payload, flags)
			}
			if checkLive {
				c, err := flags.newClient()
				if err != nil {
					return err
				}
				resultsStatus, probeErr := c.ProbeGet(cmd.Context(), derived.RelativeURL)
				if probeErr != nil {
					return classifySearchValidationError(probeErr, derived.RelativeURL, flags)
				}
				_, geoErr := c.Get(cmd.Context(), derived.GeoReachPath, derived.GeoReachQuery)
				if geoErr != nil {
					return classifySearchValidationError(geoErr, derived.GeoReachURL, flags)
				}
				payload["live_check"] = liveCheckResult{
					ResultsURLStatusCode: resultsStatus,
					ResultsURLStatus:     http.StatusText(resultsStatus),
					GeoReachStatusCode:   http.StatusOK,
					GeoReachStatus:       http.StatusText(http.StatusOK),
				}
			}
			data, err := json.Marshal(payload)
			if err != nil {
				return err
			}
			if flags.asJSON || (!isTerminal(cmd.OutOrStdout()) && !flags.csv && !flags.quiet && !flags.plain) {
				filtered := data
				if flags.selectFields != "" {
					filtered = filterFields(filtered, flags.selectFields)
				} else if flags.compact {
					filtered = compactFields(filtered)
				}
				wrapped, wrapErr := wrapWithProvenance(filtered, DataProvenance{Source: "constructed"})
				if wrapErr != nil {
					return wrapErr
				}
				return printOutput(cmd.OutOrStdout(), wrapped, true)
			}
			return printOutputWithFlags(cmd.OutOrStdout(), data, flags)
		},
	}
	cmd.Flags().StringVar(&locationPath, "location-path", "", "Required location path like comprar-casas/lisboa/arroios")
	cmd.Flags().IntVar(&minPrice, "min-price", 0, "Validated minimum price filter")
	cmd.Flags().IntVar(&maxPrice, "max-price", 0, "Validated maximum price filter")
	cmd.Flags().IntVar(&minSize, "min-size", 0, "Validated minimum area filter")
	cmd.Flags().IntVar(&maxSize, "max-size", 0, "Validated maximum area filter")
	cmd.Flags().StringSliceVar(&bedrooms, "bedrooms", nil, "Validated room bands (t0,t1,t2,t3,t4-t5)")
	cmd.Flags().IntSliceVar(&bathrooms, "bathrooms", nil, "Validated bathroom counts (1,2,3)")
	cmd.Flags().StringSliceVar(&amenities, "amenities", nil, "Validated amenity filters (elevador,garagem,arrecadacao,arcondicionado,roupeiros-embutidos,vista-mar)")
	cmd.Flags().StringVar(&energyClass, "energy-class", "", "Validated energy class (alta,media,baixa)")
	cmd.Flags().StringVar(&publishedWithin, "published-within", "", "Validated recency window (48h,week,month)")
	cmd.Flags().StringVar(&sortOrder, "sort", "", "Validated sort order (preco_medio-asc, precos-asc, atualizado-desc)")
	cmd.Flags().BoolVar(&checkLive, "check-live", false, "Probe both the browse URL and georeach query against the live website session")
	_ = cmd.MarkFlagRequired("location-path")
	return cmd
}

func currentBaseURL(flags *rootFlags) string {
	if flags == nil {
		return "https://www.idealista.pt"
	}
	cfg, err := config.Load(flags.configPath)
	if err != nil || cfg == nil || cfg.BaseURL == "" {
		return "https://www.idealista.pt"
	}
	return cfg.BaseURL
}

func buildSearchResultsRelativeURL(spec searchResultsSpec) (string, error) {
	derived, err := buildSearchResultsState(spec, "https://www.idealista.pt")
	if err != nil {
		return "", err
	}
	return derived.RelativeURL, nil
}

func buildSearchResultsState(spec searchResultsSpec, baseURL string) (derivedSearchState, error) {
	normalized, err := normalizeSearchResultsSpec(spec)
	if err != nil {
		return derivedSearchState{}, err
	}
	energyToken, err := canonicalEnergyClass(normalized.EnergyClass)
	if err != nil {
		return derivedSearchState{}, err
	}
	publishedToken, err := canonicalPublishedWindow(normalized.PublishedWithin)
	if err != nil {
		return derivedSearchState{}, err
	}

	filterTokens := make([]string, 0, 4+len(normalized.Bedrooms)+len(normalized.Bathrooms)+len(normalized.Amenities)+2)
	if normalized.MaxPrice > 0 {
		filterTokens = append(filterTokens, "preco-max_"+strconv.Itoa(normalized.MaxPrice))
	}
	if normalized.MinPrice > 0 {
		filterTokens = append(filterTokens, "preco-min_"+strconv.Itoa(normalized.MinPrice))
	}
	if normalized.MinSize > 0 {
		filterTokens = append(filterTokens, "tamanho-min_"+strconv.Itoa(normalized.MinSize))
	}
	if normalized.MaxSize > 0 {
		filterTokens = append(filterTokens, "tamanho-max_"+strconv.Itoa(normalized.MaxSize))
	}
	filterTokens = append(filterTokens, normalized.Bedrooms...)
	for _, bathroom := range normalized.Bathrooms {
		filterTokens = append(filterTokens, "banho-"+strconv.Itoa(bathroom))
	}
	filterTokens = append(filterTokens, normalized.Amenities...)
	if energyToken != "" {
		filterTokens = append(filterTokens, energyToken)
	}
	if publishedToken != "" {
		filterTokens = append(filterTokens, publishedToken)
	}

	relativePath := "/" + normalized.LocationPath + "/"
	if len(filterTokens) > 0 {
		relativePath += "com-" + strings.Join(filterTokens, ",") + "/"
	}
	u := &url.URL{Path: path.Clean(relativePath)}
	if strings.HasSuffix(relativePath, "/") && !strings.HasSuffix(u.Path, "/") {
		u.Path += "/"
	}
	query := url.Values{}
	if normalized.Sort != "" {
		query.Set("ordem", normalized.Sort)
	}
	u.RawQuery = query.Encode()

	geoReachPath, geoReachParams, err := buildGeoReachState(normalized.LocationPath, spec, normalized.Bedrooms)
	if err != nil {
		return derivedSearchState{}, err
	}
	listingAjaxPath, listingAjaxParams, err := buildListingAjaxState(normalized)
	if err != nil {
		return derivedSearchState{}, err
	}
	totalsAjaxPath := "/ajax/listingcontroller/totals/listingajax.ajax"
	totalsAjaxParams := cloneStringMap(listingAjaxParams)
	geoURL := strings.TrimRight(baseURL, "/") + geoReachPath
	if len(geoReachParams) > 0 {
		geoQuery := url.Values{}
		keys := make([]string, 0, len(geoReachParams))
		for k := range geoReachParams {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, key := range keys {
			geoQuery.Set(key, geoReachParams[key])
		}
		geoURL += "?" + geoQuery.Encode()
	}
	listingAjaxURL := strings.TrimRight(baseURL, "/") + listingAjaxPath
	if encoded := encodeSortedQuery(listingAjaxParams); encoded != "" {
		listingAjaxURL += "?" + encoded
	}
	totalsAjaxURL := strings.TrimRight(baseURL, "/") + totalsAjaxPath
	if encoded := encodeSortedQuery(totalsAjaxParams); encoded != "" {
		totalsAjaxURL += "?" + encoded
	}

	return derivedSearchState{
		RelativeURL:       u.String(),
		GeoReachPath:      geoReachPath,
		GeoReachURL:       geoURL,
		GeoReachQuery:     geoReachParams,
		ListingAjaxPath:   listingAjaxPath,
		ListingAjaxURL:    listingAjaxURL,
		ListingAjaxQuery:  listingAjaxParams,
		TotalsAjaxPath:    totalsAjaxPath,
		TotalsAjaxURL:     totalsAjaxURL,
		TotalsAjaxQuery:   totalsAjaxParams,
		FilterTokens:      filterTokens,
		OmittedFromURL:    nil,
		URLFilterCoverage: "full",
	}, nil
}

func buildGeoReachState(locationPath string, spec searchResultsSpec, bedrooms []string) (string, map[string]string, error) {
	parts := strings.Split(strings.TrimPrefix(locationPath, "comprar-casas/"), "/")
	if len(parts) < 2 {
		return "", nil, fmt.Errorf("--location-path must include at least province and zone after comprar-casas/")
	}
	geoReachPath := "/pt/ajax/listing/georeach/" + strings.Join(parts, "/")
	params := map[string]string{}
	if spec.MinPrice > 0 {
		params["minPrice"] = strconv.Itoa(spec.MinPrice)
	}
	if spec.MaxPrice > 0 {
		params["maxPrice"] = strconv.Itoa(spec.MaxPrice)
	}
	if spec.MinSize > 0 {
		params["minArea"] = strconv.Itoa(spec.MinSize)
	}
	if spec.MaxSize > 0 {
		params["maxArea"] = strconv.Itoa(spec.MaxSize)
	}
	for _, bedroom := range bedrooms {
		paramName := bedroomToGeoReachParam[bedroom]
		if paramName == "" {
			return "", nil, fmt.Errorf("no georeach room mapping for %q", bedroom)
		}
		params[paramName] = "true"
	}
	return geoReachPath, params, nil
}

func normalizeSearchResultsSpec(spec searchResultsSpec) (normalizedSearchResultsSpec, error) {
	locationPath := normalizeLocationPath(spec.LocationPath)
	if locationPath == "" {
		return normalizedSearchResultsSpec{}, fmt.Errorf("--location-path is required")
	}
	if !strings.HasPrefix(locationPath, "comprar-casas/") {
		return normalizedSearchResultsSpec{}, fmt.Errorf("--location-path must start with comprar-casas/")
	}
	if spec.MinPrice < 0 || spec.MaxPrice < 0 || spec.MinSize < 0 || spec.MaxSize < 0 {
		return normalizedSearchResultsSpec{}, fmt.Errorf("numeric bounds must be positive")
	}
	if spec.MinPrice > 0 && spec.MaxPrice > 0 && spec.MinPrice > spec.MaxPrice {
		return normalizedSearchResultsSpec{}, fmt.Errorf("--min-price cannot be greater than --max-price")
	}
	if spec.MinSize > 0 && spec.MaxSize > 0 && spec.MinSize > spec.MaxSize {
		return normalizedSearchResultsSpec{}, fmt.Errorf("--min-size cannot be greater than --max-size")
	}

	bedrooms := canonicalBedrooms(spec.Bedrooms)
	for _, bedroom := range bedrooms {
		if !slices.Contains(supportedBedroomBands, bedroom) {
			return normalizedSearchResultsSpec{}, fmt.Errorf("unsupported bedroom band %q; supported: %s", bedroom, strings.Join(supportedBedroomBands, ", "))
		}
	}
	bathrooms := canonicalBathrooms(spec.Bathrooms)
	for _, bathroom := range bathrooms {
		if !slices.Contains(supportedBathroomCounts, bathroom) {
			return normalizedSearchResultsSpec{}, fmt.Errorf("unsupported bathroom count %d; supported: 1, 2, 3", bathroom)
		}
	}
	amenities, err := canonicalAmenities(spec.Amenities)
	if err != nil {
		return normalizedSearchResultsSpec{}, err
	}
	if spec.Sort != "" && !slices.Contains(supportedSortOrders, spec.Sort) {
		return normalizedSearchResultsSpec{}, fmt.Errorf("unsupported sort %q; supported: %s", spec.Sort, strings.Join(supportedSortOrders, ", "))
	}

	return normalizedSearchResultsSpec{
		LocationPath:    locationPath,
		LocationURI:     strings.TrimPrefix(locationPath, "comprar-casas/"),
		MinPrice:        spec.MinPrice,
		MaxPrice:        spec.MaxPrice,
		MinSize:         spec.MinSize,
		MaxSize:         spec.MaxSize,
		Bedrooms:        bedrooms,
		Bathrooms:       bathrooms,
		Amenities:       amenities,
		EnergyClass:     strings.TrimSpace(strings.ToLower(spec.EnergyClass)),
		PublishedWithin: strings.TrimSpace(strings.ToLower(spec.PublishedWithin)),
		Sort:            strings.TrimSpace(spec.Sort),
	}, nil
}

func buildListingAjaxState(spec normalizedSearchResultsSpec) (string, map[string]string, error) {
	query := map[string]string{
		"typology":                              "1",
		"operation":                             "1",
		"freeText":                              "",
		"locationUri":                           spec.LocationURI,
		"adfilter_pricemin":                     "default",
		"adfilter_price":                        "default",
		"adfilter_area":                         "default",
		"adfilter_areamax":                      "default",
		"adfilter_tenanted":                     "",
		"adfilter_free":                         "",
		"adfilter_rooms_0":                      "",
		"adfilter_rooms_1":                      "",
		"adfilter_rooms_2":                      "",
		"adfilter_rooms_3":                      "",
		"adfilter_rooms_4_more":                 "",
		"adfilter_baths_1":                      "",
		"adfilter_baths_2":                      "",
		"adfilter_baths_3":                      "",
		"adfilter_newconstruction":              "",
		"adfilter_goodcondition":                "",
		"adfilter_toberestored":                 "",
		"adfilter_hasairconditioning":           "",
		"adfilter_wardrobes":                    "",
		"adfilter_lift":                         "",
		"adfilter_parkingspace":                 "",
		"adfilter_garden":                       "",
		"adfilter_swimmingpool":                 "",
		"adfilter_boxroom":                      "",
		"adfilter_accessibleHousing":            "",
		"adfilter_luxury":                       "",
		"adfilter_seaviews":                     "",
		"adfilter_top_floor":                    "",
		"adfilter_intermediate_floor":           "",
		"adfilter_ground_floor":                 "",
		"adfilter_energyCertificateHigh":        "",
		"adfilter_energyCertificateMedium":      "",
		"adfilter_energyCertificateLow":         "",
		"adfilter_hasplan":                      "",
		"adfilter_digitalvisit":                 "",
		"adfilter_agencyisabank":                "",
		"adfilter_published":                    "default",
		"ordem":                                 "",
		"adfilter_onlyflats":                    "",
		"adfilter_penthouse":                    "",
		"adfilter_duplex":                       "",
		"adfilter_homes":                        "",
		"adfilter_independent":                  "",
		"adfilter_semidetached":                 "",
		"adfilter_terraced":                     "",
		"adfilter_countryhouses":                "",
		"adfilter_chalets":                      "",
		"adfilter_balcony":                      "",
		"adfilter_hasterrace":                   "",
		"adfilter_exterior_domestic_space_type": "",
		"device":                                "desktop",
	}
	if spec.MinPrice > 0 {
		query["adfilter_pricemin"] = strconv.Itoa(spec.MinPrice)
	}
	if spec.MaxPrice > 0 {
		query["adfilter_price"] = strconv.Itoa(spec.MaxPrice)
	}
	if spec.MinSize > 0 {
		query["adfilter_area"] = strconv.Itoa(spec.MinSize)
	}
	if spec.MaxSize > 0 {
		query["adfilter_areamax"] = strconv.Itoa(spec.MaxSize)
	}
	for _, bedroom := range spec.Bedrooms {
		switch bedroom {
		case "t0":
			query["adfilter_rooms_0"] = "0"
		case "t1":
			query["adfilter_rooms_1"] = "1"
		case "t2":
			query["adfilter_rooms_2"] = "2"
		case "t3":
			query["adfilter_rooms_3"] = "3"
		case "t4-t5":
			query["adfilter_rooms_4_more"] = "4"
		}
	}
	for _, bathroom := range spec.Bathrooms {
		query["adfilter_baths_"+strconv.Itoa(bathroom)] = strconv.Itoa(bathroom)
	}
	for _, amenity := range spec.Amenities {
		switch amenity {
		case "elevador":
			query["adfilter_lift"] = "1"
		case "garagem":
			query["adfilter_parkingspace"] = "1"
		case "arrecadacao":
			query["adfilter_boxroom"] = "1"
		case "arcondicionado":
			query["adfilter_hasairconditioning"] = "1"
		case "roupeiros-embutidos":
			query["adfilter_wardrobes"] = "1"
		case "vista-mar":
			query["adfilter_seaviews"] = "1"
		}
	}
	switch spec.EnergyClass {
	case "alta":
		query["adfilter_energyCertificateHigh"] = "1"
	case "media":
		query["adfilter_energyCertificateMedium"] = "1"
	case "baixa":
		query["adfilter_energyCertificateLow"] = "1"
	}
	switch spec.PublishedWithin {
	case "48h":
		query["adfilter_published"] = "1"
	case "week":
		query["adfilter_published"] = "2"
	case "month":
		query["adfilter_published"] = "3"
	}
	if spec.Sort != "" {
		query["ordem"] = spec.Sort
	}
	return "/ajax/listingcontroller/listingajax.ajax", query, nil
}

func encodeSortedQuery(params map[string]string) string {
	query := url.Values{}
	keys := make([]string, 0, len(params))
	for key := range params {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		query.Set(key, params[key])
	}
	return query.Encode()
}

func cloneStringMap(in map[string]string) map[string]string {
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func canonicalBedrooms(raw []string) []string {
	values := splitCSVValues(raw)
	return orderUniqueStrings(values, supportedBedroomBands)
}

func canonicalBathrooms(raw []int) []int {
	seen := map[int]bool{}
	out := make([]int, 0, len(raw))
	for _, value := range raw {
		if seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	sort.Ints(out)
	return out
}

func canonicalAmenities(raw []string) ([]string, error) {
	values := splitCSVValues(raw)
	keys := make([]string, 0, len(supportedAmenityTokens))
	for key := range supportedAmenityTokens {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	orderedInputs := orderUniqueStrings(values, keys)
	out := make([]string, 0, len(orderedInputs))
	for _, value := range orderedInputs {
		token, ok := supportedAmenityTokens[value]
		if !ok {
			return nil, fmt.Errorf("unsupported amenity %q; supported: %s", value, strings.Join(keys, ", "))
		}
		out = append(out, token)
	}
	return out, nil
}

func canonicalEnergyClass(raw string) (string, error) {
	value := strings.TrimSpace(strings.ToLower(raw))
	if value == "" {
		return "", nil
	}
	token, ok := supportedEnergyClasses[value]
	if !ok {
		keys := make([]string, 0, len(supportedEnergyClasses))
		for key := range supportedEnergyClasses {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		return "", fmt.Errorf("unsupported energy class %q; supported: %s", value, strings.Join(keys, ", "))
	}
	return token, nil
}

func canonicalPublishedWindow(raw string) (string, error) {
	value := strings.TrimSpace(strings.ToLower(raw))
	if value == "" {
		return "", nil
	}
	token, ok := supportedPublishedWindows[value]
	if !ok {
		keys := make([]string, 0, len(supportedPublishedWindows))
		for key := range supportedPublishedWindows {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		return "", fmt.Errorf("unsupported published-within value %q; supported: %s", value, strings.Join(keys, ", "))
	}
	return token, nil
}

func splitCSVValues(raw []string) []string {
	out := make([]string, 0, len(raw))
	for _, item := range raw {
		for _, part := range strings.Split(item, ",") {
			value := strings.TrimSpace(strings.ToLower(part))
			if value == "" {
				continue
			}
			out = append(out, value)
		}
	}
	return out
}

func orderUniqueStrings(values []string, order []string) []string {
	orderIndex := map[string]int{}
	for i, value := range order {
		orderIndex[value] = i
	}
	seen := map[string]bool{}
	filtered := make([]string, 0, len(values))
	for _, value := range values {
		if seen[value] {
			continue
		}
		seen[value] = true
		filtered = append(filtered, value)
	}
	sort.SliceStable(filtered, func(i, j int) bool {
		oi, iok := orderIndex[filtered[i]]
		oj, jok := orderIndex[filtered[j]]
		switch {
		case iok && jok:
			return oi < oj
		case iok:
			return true
		case jok:
			return false
		default:
			return filtered[i] < filtered[j]
		}
	})
	return filtered
}

func normalizeLocationPath(raw string) string {
	trimmed := strings.TrimSpace(raw)
	return strings.Trim(trimmed, "/")
}

func classifySearchValidationError(err error, target string, flags *rootFlags) error {
	var apiErr *client.APIError
	if As(err, &apiErr) {
		if apiErr.StatusCode == http.StatusUnauthorized || apiErr.StatusCode == http.StatusForbidden {
			return authErr(fmt.Errorf("website session cookie refresh required to validate %s", target))
		}
	}
	if strings.Contains(err.Error(), "HTTP 401") || strings.Contains(err.Error(), "HTTP 403") {
		return authErr(fmt.Errorf("website session cookie refresh required to validate %s", target))
	}
	return classifyAPIError(err, flags)
}

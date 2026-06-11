package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"
	"idealista-pp-cli/internal/client"
)

type listingContactSummary struct {
	CommercialName   string `json:"commercial_name,omitempty"`
	ReferenceMessage string `json:"reference_message,omitempty"`
	FormattedPhone   string `json:"formatted_phone,omitempty"`
	ShowPhoneContact bool   `json:"show_phone_contact,omitempty"`
	ShowEmailContact bool   `json:"show_email_contact,omitempty"`
	ShowProfessional bool   `json:"show_professional_name,omitempty"`
}

type listingMortgageSummary struct {
	InitialPrice           any `json:"initial_price,omitempty"`
	SavedMoney             any `json:"saved_money,omitempty"`
	DefaultPercentSavings  any `json:"default_percent_savings,omitempty"`
	YearsSupported         any `json:"years_supported,omitempty"`
	MaxYears               any `json:"max_years,omitempty"`
	TaxRateFixed           any `json:"tax_rate_fixed,omitempty"`
	RateInit               any `json:"rate_init,omitempty"`
	SimulationWithExpenses any `json:"simulation_with_expenses,omitempty"`
}

type listingInspectSummary struct {
	ListingID       string                 `json:"listing_id"`
	ListingURL      string                 `json:"listing_url,omitempty"`
	Title           string                 `json:"title,omitempty"`
	Price           any                    `json:"price,omitempty"`
	Operation       any                    `json:"operation,omitempty"`
	Typology        any                    `json:"typology,omitempty"`
	GeoLocationID   any                    `json:"geo_location_id,omitempty"`
	Address         string                 `json:"address,omitempty"`
	PrimaryImageURL string                 `json:"primary_image_url,omitempty"`
	ImageCount      int                    `json:"image_count"`
	Images          []listingPhoto         `json:"images,omitempty"`
	FeaturesSummary []string               `json:"features_summary,omitempty"`
	AgencyName      []string               `json:"agency_name,omitempty"`
	Contact         listingContactSummary  `json:"contact,omitempty"`
	Mortgage        listingMortgageSummary `json:"mortgage,omitempty"`
	Warnings        []string               `json:"warnings,omitempty"`
}

func newListingInspectCmd(flags *rootFlags) *cobra.Command {
	var typologyID string
	var isVacational bool

	cmd := &cobra.Command{
		Use:   "inspect <listing_id>",
		Short: "Get a shaped listing summary across detail endpoints",
		Args:  cobra.ExactArgs(1),
		Example: `  idealista-pp-cli listing inspect 34998327
  idealista-pp-cli listing inspect 34998327 --json --select listing_id,title,price,contact`,
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := flags.newClient()
			if err != nil {
				return err
			}
			summary, err := inspectListing(cmd.Context(), c, flags, args[0], typologyID, isVacational)
			if err != nil {
				return classifyAPIError(err, flags)
			}
			return outputWebsiteSearchPayload(cmd, flags, summary, DataProvenance{Source: "live"})
		},
	}
	cmd.Flags().StringVar(&typologyID, "typology-id", "", "Optional listing typology for the datalayer request")
	cmd.Flags().BoolVar(&isVacational, "is-vacational", false, "Pass isVacational=true to the gallery endpoint")
	return cmd
}

func inspectListing(ctx context.Context, c *client.Client, flags *rootFlags, listingID, typologyID string, isVacational bool) (listingInspectSummary, error) {
	detailPath := replacePathParam("/detail/{detail_id}/datalayer", "detail_id", listingID)
	detailParams := map[string]string{}
	if strings.TrimSpace(typologyID) != "" {
		detailParams["typologyId"] = typologyID
	}
	detailData, _, err := resolveReadWithStrategy(ctx, c, flags, "live", "detail", false, detailPath, detailParams, nil, io.Discard)
	if err != nil {
		return listingInspectSummary{}, err
	}

	galleryPath := replacePathParam("/pt/openDetailGallery/{opendetailgallery_id}", "opendetailgallery_id", listingID)
	galleryParams := map[string]string{}
	if isVacational {
		galleryParams["isVacational"] = "true"
	}
	galleryData, _, err := resolveReadWithStrategy(ctx, c, flags, "live", "pt", false, galleryPath, galleryParams, nil, io.Discard)
	if err != nil {
		return listingInspectSummary{}, err
	}

	configurationPath := replacePathParam("/pt/detail/{detail_id}/configuration", "detail_id", listingID)
	configurationData, _, configurationErr := resolveReadWithStrategy(ctx, c, flags, "live", "pt", false, configurationPath, nil, nil, io.Discard)

	contactData, _, contactErr := resolveReadWithStrategy(ctx, c, flags, "live", "pt", false, endpointPath("pt", "ajax", "listingController", "adContactInfoForDetail.ajax"), map[string]string{
		"adId": listingID,
	}, nil, io.Discard)

	return summarizeListingInspect(listingID, detailData, galleryData, configurationData, configurationErr, contactData, contactErr)
}

func summarizeListingInspect(listingID string, detailData, galleryData, configurationData json.RawMessage, configurationErr error, contactData json.RawMessage, contactErr error) (listingInspectSummary, error) {
	photosSummary, err := summarizeListingPhotos(listingID, detailData, galleryData)
	if err != nil {
		return listingInspectSummary{}, err
	}
	summary := listingInspectSummary{
		ListingID:       photosSummary.ListingID,
		ListingURL:      photosSummary.ListingURL,
		Price:           photosSummary.Price,
		Operation:       photosSummary.Operation,
		Typology:        photosSummary.Typology,
		GeoLocationID:   photosSummary.GeoLocationID,
		PrimaryImageURL: photosSummary.PrimaryImageURL,
		ImageCount:      photosSummary.ImageCount,
		Images:          photosSummary.Images,
		FeaturesSummary: photosSummary.FeaturesSummary,
		AgencyName:      photosSummary.AgencyName,
	}

	var detail map[string]any
	if len(detailData) > 0 && json.Unmarshal(detailData, &detail) == nil {
		if value, ok := findPreferredScalar(detail, []string{"adTitle", "title", "subtitle", "address"}); ok {
			summary.Title = fmt.Sprint(value)
		}
		if value, ok := findPreferredScalar(detail, []string{"address", "streetName", "location"}); ok {
			summary.Address = fmt.Sprint(value)
		}
	}

	if configurationErr != nil {
		summary.Warnings = append(summary.Warnings, "configuration_unavailable")
	} else if len(configurationData) > 0 {
		summary.Mortgage = summarizeMortgageConfiguration(configurationData)
	}

	if contactErr != nil {
		summary.Warnings = append(summary.Warnings, "contact_unavailable")
	} else if len(contactData) > 0 {
		summary.Contact = summarizeContactInfo(contactData)
	}

	return summary, nil
}

func summarizeMortgageConfiguration(raw json.RawMessage) listingMortgageSummary {
	var root map[string]any
	if json.Unmarshal(raw, &root) != nil {
		return listingMortgageSummary{}
	}
	mortgages, _ := root["mortgagesConfiguration"].(map[string]any)
	if len(mortgages) == 0 {
		return listingMortgageSummary{}
	}
	return listingMortgageSummary{
		InitialPrice:           mortgages["initialPrice"],
		SavedMoney:             mortgages["savedMoney"],
		DefaultPercentSavings:  mortgages["defaultPercentSavings"],
		YearsSupported:         mortgages["yearsSupported"],
		MaxYears:               mortgages["maxYears"],
		TaxRateFixed:           mortgages["taxRateFixed"],
		RateInit:               mortgages["rateInit"],
		SimulationWithExpenses: mortgages["simulationWithExpenses"],
	}
}

func summarizeContactInfo(raw json.RawMessage) listingContactSummary {
	var root map[string]any
	if json.Unmarshal(raw, &root) != nil {
		return listingContactSummary{}
	}
	data, _ := root["data"].(map[string]any)
	if len(data) == 0 {
		return listingContactSummary{}
	}
	return listingContactSummary{
		CommercialName:   stringifyValue(data["commercialName"]),
		ReferenceMessage: stringifyValue(data["referenceMessage"]),
		FormattedPhone:   firstNonEmpty(stringifyValue(data["formattedContactPhoneWithPrefix"]), stringifyValue(data["formattedContactPhone1"])),
		ShowPhoneContact: boolValue(data["showPhoneContactMethod"]),
		ShowEmailContact: boolValue(data["showEmailContactMethod"]),
		ShowProfessional: boolValue(data["showProfessionalName"]),
	}
}

func stringifyValue(v any) string {
	value := strings.TrimSpace(fmt.Sprint(v))
	if value == "" || value == "<nil>" {
		return ""
	}
	return value
}

func boolValue(v any) bool {
	value, _ := v.(bool)
	return value
}

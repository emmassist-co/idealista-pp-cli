package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"
)

type listingPhoto struct {
	ID          any    `json:"id,omitempty"`
	URL         string `json:"url,omitempty"`
	WebPURL     string `json:"webp_url,omitempty"`
	Description string `json:"description,omitempty"`
	Width       any    `json:"width,omitempty"`
	Height      any    `json:"height,omitempty"`
}

type listingPhotosSummary struct {
	ListingID       string         `json:"listing_id"`
	ListingURL      string         `json:"listing_url,omitempty"`
	PrimaryImageURL string         `json:"primary_image_url,omitempty"`
	ImageCount      int            `json:"image_count"`
	Images          []listingPhoto `json:"images,omitempty"`
	FeaturesSummary []string       `json:"features_summary,omitempty"`
	AgencyName      []string       `json:"agency_name,omitempty"`
	Price           any            `json:"price,omitempty"`
	Typology        any            `json:"typology,omitempty"`
	Operation       any            `json:"operation,omitempty"`
	GeoLocationID   any            `json:"geo_location_id,omitempty"`
}

func newListingCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "listing",
		Short: "High-level listing workflows",
		Long: `Listing commands compose the generated website-detail endpoints into
operator-friendly read-only workflows.`,
		RunE: parentNoSubcommandRunE(flags),
	}
	cmd.AddCommand(newListingPhotosCmd(flags))
	return cmd
}

func newListingPhotosCmd(flags *rootFlags) *cobra.Command {
	var typologyID string
	var isVacational bool

	cmd := &cobra.Command{
		Use:   "photos <listing_id>",
		Short: "Get listing photos and gallery metadata",
		Args:  cobra.ExactArgs(1),
		Example: `  idealista-pp-cli listing photos 34998327
  idealista-pp-cli listing photos 34998327 --json --select listing_id,primary_image_url,image_count`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runListingPhotos(cmd, flags, args[0], typologyID, isVacational)
		},
	}
	cmd.Flags().StringVar(&typologyID, "typology-id", "", "Optional listing typology for the datalayer request")
	cmd.Flags().BoolVar(&isVacational, "is-vacational", false, "Pass isVacational=true to the gallery endpoint")
	return cmd
}

func runListingPhotos(cmd *cobra.Command, flags *rootFlags, listingID, typologyID string, isVacational bool) error {
	c, err := flags.newClient()
	if err != nil {
		return err
	}

	detailPath := replacePathParam("/detail/{detail_id}/datalayer", "detail_id", listingID)
	detailParams := map[string]string{}
	if strings.TrimSpace(typologyID) != "" {
		detailParams["typologyId"] = typologyID
	}
	detailData, _, err := resolveReadWithStrategy(cmd.Context(), c, flags, "live", "detail", false, detailPath, detailParams, nil, io.Discard)
	if err != nil {
		return classifyAPIError(err, flags)
	}

	galleryPath := replacePathParam("/pt/openDetailGallery/{opendetailgallery_id}", "opendetailgallery_id", listingID)
	galleryParams := map[string]string{}
	if isVacational {
		galleryParams["isVacational"] = "true"
	}
	galleryData, _, err := resolveReadWithStrategy(cmd.Context(), c, flags, "live", "pt", false, galleryPath, galleryParams, nil, io.Discard)
	if err != nil {
		return classifyAPIError(err, flags)
	}

	payload, err := summarizeListingPhotos(listingID, detailData, galleryData)
	if err != nil {
		return err
	}
	return outputWebsiteSearchPayload(cmd, flags, payload, DataProvenance{Source: "live"})
}

func summarizeListingPhotos(listingID string, detailData, galleryData json.RawMessage) (listingPhotosSummary, error) {
	var detail map[string]any
	if len(detailData) > 0 {
		if err := json.Unmarshal(detailData, &detail); err != nil {
			return listingPhotosSummary{}, fmt.Errorf("parse detail datalayer: %w", err)
		}
	}
	var galleryRoot map[string]any
	if err := json.Unmarshal(galleryData, &galleryRoot); err != nil {
		return listingPhotosSummary{}, fmt.Errorf("parse gallery payload: %w", err)
	}

	summary := listingPhotosSummary{
		ListingID:  listingID,
		ListingURL: "https://www.idealista.pt/imovel/" + listingID + "/",
	}

	if value, ok := findScalarByKey(detail, "adId"); ok {
		summary.ListingID = fmt.Sprint(value)
		summary.ListingURL = "https://www.idealista.pt/imovel/" + summary.ListingID + "/"
	}
	if value, ok := findScalarByKey(detail, "adPrice"); ok {
		summary.Price = value
	}
	if value, ok := findScalarByKey(detail, "typology"); ok {
		summary.Typology = value
	}
	if value, ok := findScalarByKey(detail, "operation"); ok {
		summary.Operation = value
	}
	if value, ok := findScalarByKey(detail, "geoLocationId"); ok {
		summary.GeoLocationID = value
	}
	if agencyRaw, ok := detail["agencyName"].([]any); ok {
		summary.AgencyName = stringifySlice(agencyRaw)
	}

	dataNode, _ := galleryRoot["data"].(map[string]any)
	if featuresRaw, ok := dataNode["featuresSummary"].([]any); ok {
		summary.FeaturesSummary = summarizeFeatureTexts(featuresRaw)
	}
	multimedias, _ := dataNode["multimedias"].(map[string]any)
	if pictureRaw, ok := multimedias["PICTURE"].([]any); ok {
		summary.Images = summarizePictures(pictureRaw)
		summary.ImageCount = len(summary.Images)
		if len(summary.Images) > 0 {
			summary.PrimaryImageURL = summary.Images[0].URL
		}
	}

	return summary, nil
}

func summarizeFeatureTexts(raw []any) []string {
	out := make([]string, 0, len(raw))
	for _, item := range raw {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		text, _ := m["text"].(string)
		if strings.TrimSpace(text) == "" {
			continue
		}
		out = append(out, text)
	}
	return out
}

func summarizePictures(raw []any) []listingPhoto {
	out := make([]listingPhoto, 0, len(raw))
	for _, item := range raw {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		photo := listingPhoto{}
		if value, ok := m["id"]; ok {
			photo.ID = value
		}
		if value, ok := m["src"].(string); ok {
			photo.URL = value
		}
		if value, ok := m["srcWebp"].(string); ok {
			photo.WebPURL = value
		}
		if value, ok := m["description"].(string); ok {
			photo.Description = value
		}
		if value, ok := m["width"]; ok {
			photo.Width = value
		}
		if value, ok := m["height"]; ok {
			photo.Height = value
		}
		out = append(out, photo)
	}
	return out
}

func stringifySlice(raw []any) []string {
	out := make([]string, 0, len(raw))
	for _, item := range raw {
		value := strings.TrimSpace(fmt.Sprint(item))
		if value == "" || value == "<nil>" {
			continue
		}
		out = append(out, value)
	}
	return out
}

package cli

import (
	"encoding/json"
	"testing"
)

func TestListingPhotosSummary(t *testing.T) {
	detail := json.RawMessage(`{
		"responseStatusCode":"200",
		"adId":"34998327",
		"operation":"1",
		"typology":"1",
		"geoLocationId":"0-EU-PT-11-06-056-56-03-001",
		"adPrice":"460000.0",
		"agencyName":["RE/MAX Vantagem Platina"]
	}`)
	gallery := json.RawMessage(`{
		"result":"ok",
		"data":{
			"featuresSummary":[
				{"type":"CONSTRUCTED_AREA","text":"84 m² área bruta"},
				{"type":"ROOM_NUMBER","text":"T3"}
			],
			"multimedias":{
				"PICTURE":[
					{"id":316068131,"src":"https://img4.idealista.pt/a.jpg","srcWebp":"https://img4.idealista.pt/a.webp","description":"Cozinha","width":1200,"height":800},
					{"id":316068178,"src":"https://img4.idealista.pt/b.jpg","srcWebp":"https://img4.idealista.pt/b.webp","description":"Quarto","width":1200,"height":800}
				]
			}
		}
	}`)
	got, err := summarizeListingPhotos("34998327", detail, gallery)
	if err != nil {
		t.Fatalf("summarizeListingPhotos: %v", err)
	}
	if got.ListingID != "34998327" {
		t.Fatalf("ListingID = %q", got.ListingID)
	}
	if got.PrimaryImageURL != "https://img4.idealista.pt/a.jpg" {
		t.Fatalf("PrimaryImageURL = %q", got.PrimaryImageURL)
	}
	if got.ImageCount != 2 {
		t.Fatalf("ImageCount = %d", got.ImageCount)
	}
	if len(got.FeaturesSummary) != 2 {
		t.Fatalf("FeaturesSummary len = %d", len(got.FeaturesSummary))
	}
	if len(got.AgencyName) != 1 || got.AgencyName[0] != "RE/MAX Vantagem Platina" {
		t.Fatalf("AgencyName = %#v", got.AgencyName)
	}
}

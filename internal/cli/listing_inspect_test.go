package cli

import (
	"encoding/json"
	"testing"
)

func TestListingInspectSummary(t *testing.T) {
	detail := json.RawMessage(`{
		"responseStatusCode":"200",
		"adId":"34998327",
		"adTitle":"Apartamento T3 em Arroios",
		"address":"Rua Carlos Mardel",
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
					{"id":316068131,"src":"https://img4.idealista.pt/a.jpg","srcWebp":"https://img4.idealista.pt/a.webp","description":"Cozinha","width":1200,"height":800}
				]
			}
		}
	}`)
	configuration := json.RawMessage(`{
		"rolConfiguration":{"rolSupportPhone":null},
		"mortgagesConfiguration":{
			"initialPrice":420000,
			"savedMoney":"126000.00",
			"defaultPercentSavings":"30",
			"yearsSupported":40,
			"maxYears":40,
			"taxRateFixed":3.45,
			"rateInit":3.45,
			"simulationWithExpenses":true
		}
	}`)
	contact := json.RawMessage(`{
		"message":null,
		"result":"OK",
		"errorCode":null,
		"data":{
			"commercialName":"ERA Praça de Espanha",
			"referenceMessage":"431250053",
			"formattedContactPhoneWithPrefix":"+351 210 000 000",
			"showPhoneContactMethod":true,
			"showEmailContactMethod":true,
			"showProfessionalName":true
		}
	}`)

	got, err := summarizeListingInspect("34998327", detail, gallery, configuration, nil, contact, nil)
	if err != nil {
		t.Fatalf("summarizeListingInspect: %v", err)
	}
	if got.ListingID != "34998327" {
		t.Fatalf("ListingID = %q", got.ListingID)
	}
	if got.Title != "Apartamento T3 em Arroios" {
		t.Fatalf("Title = %q", got.Title)
	}
	if got.Address != "Rua Carlos Mardel" {
		t.Fatalf("Address = %q", got.Address)
	}
	if got.PrimaryImageURL != "https://img4.idealista.pt/a.jpg" {
		t.Fatalf("PrimaryImageURL = %q", got.PrimaryImageURL)
	}
	if got.Contact.CommercialName != "ERA Praça de Espanha" {
		t.Fatalf("CommercialName = %q", got.Contact.CommercialName)
	}
	if got.Mortgage.InitialPrice != float64(420000) {
		t.Fatalf("InitialPrice = %#v", got.Mortgage.InitialPrice)
	}
	if len(got.Warnings) != 0 {
		t.Fatalf("Warnings = %#v", got.Warnings)
	}
}

func TestListingInspectSummary_ToleratesOptionalFailures(t *testing.T) {
	detail := json.RawMessage(`{"adId":"34998327","adTitle":"Apartamento T3 em Arroios","adPrice":"460000.0"}`)
	gallery := json.RawMessage(`{"data":{"multimedias":{"PICTURE":[]}}}`)

	got, err := summarizeListingInspect("34998327", detail, gallery, nil, assertiveErr("config failed"), nil, assertiveErr("contact failed"))
	if err != nil {
		t.Fatalf("summarizeListingInspect: %v", err)
	}
	if len(got.Warnings) != 2 {
		t.Fatalf("Warnings = %#v", got.Warnings)
	}
}

type assertiveErr string

func (e assertiveErr) Error() string { return string(e) }

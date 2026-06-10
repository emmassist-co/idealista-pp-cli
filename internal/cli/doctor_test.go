package cli

import (
	"strings"
	"testing"

	"idealista-pp-cli/internal/client"
)

func TestLooksLikeDoctorInterstitial_DataDome(t *testing.T) {
	body := []byte("<html><title>DataDome blocked</title>datadome captcha challenge</html>")
	if got := looksLikeDoctorInterstitial(body); got != "DataDome" {
		t.Fatalf("looksLikeDoctorInterstitial = %q, want DataDome", got)
	}
}

func TestClassifySiteProbe_ForbiddenWithoutVendorNeedsRefresh(t *testing.T) {
	got := classifySiteProbe(nil, &client.APIError{StatusCode: 403, Body: "forbidden"})
	if got.Status != "refresh-required" {
		t.Fatalf("Status = %q, want refresh-required", got.Status)
	}
	if !strings.Contains(got.APIReport, "HTTP 403") {
		t.Fatalf("APIReport = %q, want HTTP 403 detail", got.APIReport)
	}
}

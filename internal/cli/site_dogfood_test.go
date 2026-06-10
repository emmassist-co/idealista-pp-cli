package cli

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestSiteDogfood(t *testing.T) {
	if strings.TrimSpace(os.Getenv("IDEALISTA_COOKIE")) == "" {
		t.Skip("IDEALISTA_COOKIE not set; skipping live website dogfood checks")
	}

	t.Run("cookie check", func(t *testing.T) {
		stdout, _, err := executeCookieCommand(t, "--json", "cookie", "check")
		if err != nil {
			t.Fatalf("cookie check: %v\nstdout=%s", err, stdout)
		}
		var got map[string]any
		if err := json.Unmarshal([]byte(stdout), &got); err != nil {
			t.Fatalf("cookie check json: %v", err)
		}
		if got["status"] != "usable" {
			t.Fatalf("cookie check output = %s", stdout)
		}
	})

	t.Run("location suggestions", func(t *testing.T) {
		stdout, _, err := executeCookieCommand(t, "--json", "search", "locations", "lisboa")
		if err != nil {
			t.Fatalf("search locations: %v\nstdout=%s", err, stdout)
		}
		if !strings.Contains(strings.ToLower(stdout), "lisboa") {
			t.Fatalf("search locations output = %s", stdout)
		}
	})

	t.Run("saved search summary", func(t *testing.T) {
		stdout, _, err := executeCookieCommand(t, "--json", "search", "saved")
		if err != nil {
			t.Fatalf("search saved: %v\nstdout=%s", err, stdout)
		}
		if !strings.Contains(stdout, `"meta"`) {
			t.Fatalf("saved search output = %s", stdout)
		}
	})

	t.Run("validated result url", func(t *testing.T) {
		stdout, _, err := executeCookieCommand(t, "--json", "search", "results-url", "--location-path", "comprar-casas/lisboa/arroios", "--min-price", "220000", "--max-price", "750000", "--min-size", "60", "--max-size", "120", "--check-live")
		if err != nil {
			t.Fatalf("results-url --check-live: %v\nstdout=%s", err, stdout)
		}
		var got map[string]any
		if err := json.Unmarshal([]byte(stdout), &got); err != nil {
			t.Fatalf("results-url json: %v", err)
		}
		results, _ := got["results"].(map[string]any)
		liveCheck, _ := results["live_check"].(map[string]any)
		if liveCheck["results_url_status_code"] != float64(200) || liveCheck["georeach_status_code"] != float64(200) {
			t.Fatalf("results-url output = %s", stdout)
		}
	})

	t.Run("listing photos", func(t *testing.T) {
		stdout, _, err := executeCookieCommand(t, "--json", "listing", "photos", "34998327")
		if err != nil {
			t.Fatalf("listing photos: %v\nstdout=%s", err, stdout)
		}
		var got map[string]any
		if err := json.Unmarshal([]byte(stdout), &got); err != nil {
			t.Fatalf("listing photos json: %v", err)
		}
		results, _ := got["results"].(map[string]any)
		if results["listing_id"] != "34998327" || results["primary_image_url"] == nil {
			t.Fatalf("listing photos output = %s", stdout)
		}
	})
}

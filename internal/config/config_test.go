package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadCookieSourcePrefersEnv(t *testing.T) {
	t.Setenv("IDEALISTA_COOKIE", "env-cookie=xyz")
	path := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(path, []byte("[headers]\nCookie = \"config-cookie=abc\"\nX-Test = \"1\"\n"), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got := cfg.CookieHeader(); got != "env-cookie=xyz" {
		t.Fatalf("CookieHeader = %q, want env cookie", got)
	}
	if got := cfg.CookieSource(); got != "env:IDEALISTA_COOKIE" {
		t.Fatalf("CookieSource = %q, want env:IDEALISTA_COOKIE", got)
	}
}

func TestSaveCookieAndClearCookie(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	cfg.Headers = map[string]string{"X-Test": "1"}

	if err := cfg.SaveCookie("datadome=abc; other=def"); err != nil {
		t.Fatalf("SaveCookie: %v", err)
	}

	reloaded, err := Load(path)
	if err != nil {
		t.Fatalf("Reload after save: %v", err)
	}
	if got := reloaded.CookieHeader(); got != "datadome=abc; other=def" {
		t.Fatalf("CookieHeader = %q, want saved cookie", got)
	}
	if got := reloaded.CookieSource(); got != "config" {
		t.Fatalf("CookieSource = %q, want config", got)
	}
	if got := reloaded.Headers["X-Test"]; got != "1" {
		t.Fatalf("X-Test header lost after save, got %q", got)
	}

	if err := reloaded.ClearCookie(); err != nil {
		t.Fatalf("ClearCookie: %v", err)
	}

	cleared, err := Load(path)
	if err != nil {
		t.Fatalf("Reload after clear: %v", err)
	}
	if got := cleared.CookieHeader(); got != "" {
		t.Fatalf("CookieHeader = %q, want empty", got)
	}
	if got := cleared.Headers["X-Test"]; got != "1" {
		t.Fatalf("X-Test header lost after clear, got %q", got)
	}
}

func TestClearCookieRemovesHeadersTableWhenEmpty(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if err := cfg.SaveCookie("datadome=abc"); err != nil {
		t.Fatalf("SaveCookie: %v", err)
	}
	if err := cfg.ClearCookie(); err != nil {
		t.Fatalf("ClearCookie: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if strings.Contains(string(data), "[headers]") {
		t.Fatalf("config still contains [headers] after clearing sole cookie header: %s", string(data))
	}
}

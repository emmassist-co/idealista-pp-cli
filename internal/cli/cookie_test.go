package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"idealista-pp-cli/internal/client"
)

func executeCookieCommand(t *testing.T, args ...string) (stdout string, stderr string, err error) {
	return executeCookieCommandWithInput(t, "", args...)
}

func executeCookieCommandWithInput(t *testing.T, input string, args ...string) (stdout string, stderr string, err error) {
	t.Helper()
	var flags rootFlags
	cmd := newRootCmd(&flags)
	var outBuf bytes.Buffer
	var errBuf bytes.Buffer
	cmd.SetOut(&outBuf)
	cmd.SetErr(&errBuf)
	cmd.SetIn(strings.NewReader(input))
	cmd.SetArgs(args)
	err = cmd.Execute()
	if err != nil && isCobraUsageError(err) {
		err = usageErr(err)
	}
	return outBuf.String(), errBuf.String(), err
}

func TestCookieSetAndSourceJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	if _, _, err := executeCookieCommand(t, "--config", path, "--json", "cookie", "set", "datadome=abc; other=def"); err != nil {
		t.Fatalf("cookie set: %v", err)
	}

	stdout, _, err := executeCookieCommand(t, "--config", path, "--json", "cookie", "source")
	if err != nil {
		t.Fatalf("cookie source: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("source output JSON: %v", err)
	}
	if got["source"] != "config" {
		t.Fatalf("source = %v, want config", got["source"])
	}
	if got["configured"] != true {
		t.Fatalf("configured = %v, want true", got["configured"])
	}
}

func TestCookieClearJSONNotesEnvOverride(t *testing.T) {
	t.Setenv("IDEALISTA_COOKIE", "env-cookie=xyz")
	path := filepath.Join(t.TempDir(), "config.toml")
	if _, _, err := executeCookieCommand(t, "--config", path, "--json", "cookie", "set", "datadome=abc"); err != nil {
		t.Fatalf("cookie set: %v", err)
	}
	stdout, _, err := executeCookieCommand(t, "--config", path, "--json", "cookie", "clear")
	if err != nil {
		t.Fatalf("cookie clear: %v", err)
	}
	if !strings.Contains(stdout, "IDEALISTA_COOKIE env var is still set") {
		t.Fatalf("clear output = %q, want env note", stdout)
	}
}

func TestCookieSourceMissingReturnsAuthExitCode(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	_, _, err := executeCookieCommand(t, "--config", path, "--json", "cookie", "source")
	if err == nil {
		t.Fatalf("cookie source without cookie should fail")
	}
	if got := ExitCode(err); got != 4 {
		t.Fatalf("ExitCode = %d, want 4", got)
	}
}

func TestClassifySiteProbe(t *testing.T) {
	t.Run("usable", func(t *testing.T) {
		got := classifySiteProbe([]byte(`{"ok":true}`), nil)
		if got.Status != "usable" {
			t.Fatalf("Status = %q, want usable", got.Status)
		}
	})

	t.Run("datadome refresh required", func(t *testing.T) {
		err := &client.APIError{StatusCode: 403, Body: "<html><title>DataDome blocked</title>datadome blocked</html>"}
		got := classifySiteProbe(nil, err)
		if got.Status != "refresh-required" {
			t.Fatalf("Status = %q, want refresh-required", got.Status)
		}
		if got.Vendor != "DataDome" {
			t.Fatalf("Vendor = %q, want DataDome", got.Vendor)
		}
	})

	t.Run("transport failure unreachable", func(t *testing.T) {
		got := classifySiteProbe(nil, errors.New("dial tcp: no route to host"))
		if got.Status != "unreachable" {
			t.Fatalf("Status = %q, want unreachable", got.Status)
		}
	})
}

func TestCookieCheckMissingDoesNotNeedNetwork(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	stdout, _, err := executeCookieCommand(t, "--config", path, "--json", "cookie", "check")
	if err == nil {
		t.Fatalf("cookie check without cookie should fail")
	}
	if got := ExitCode(err); got != 4 {
		t.Fatalf("ExitCode = %d, want 4", got)
	}
	var got map[string]any
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("cookie check output JSON: %v", err)
	}
	if got["status"] != "missing" {
		t.Fatalf("status = %v, want missing", got["status"])
	}
}

func TestCookieSetDoesNotEchoSecret(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	secret := "datadome=super-secret-cookie"
	stdout, stderr, err := executeCookieCommand(t, "--config", path, "cookie", "set", secret)
	if err != nil {
		t.Fatalf("cookie set: %v", err)
	}
	if strings.Contains(stdout, secret) || strings.Contains(stderr, secret) {
		t.Fatalf("secret leaked in output stdout=%q stderr=%q", stdout, stderr)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if !strings.Contains(string(data), secret) {
		t.Fatalf("config file did not persist secret")
	}
}

func TestCookieSetFromStdinStripsHeaderPrefix(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	input := "Cookie: datadome=abc; other=def\n"
	if _, _, err := executeCookieCommandWithInput(t, input, "--config", path, "--json", "cookie", "set", "--stdin"); err != nil {
		t.Fatalf("cookie set --stdin: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if !strings.Contains(string(data), "datadome=abc; other=def") {
		t.Fatalf("config file did not persist stdin cookie: %s", string(data))
	}
	if strings.Contains(string(data), "Cookie: datadome=abc") {
		t.Fatalf("config file preserved Cookie header prefix: %s", string(data))
	}
}

func TestCookieSetupJSON(t *testing.T) {
	stdout, _, err := executeCookieCommand(t, "--json", "cookie", "setup")
	if err != nil {
		t.Fatalf("cookie setup: %v", err)
	}
	if !strings.Contains(stdout, "\"steps\"") {
		t.Fatalf("cookie setup missing steps: %s", stdout)
	}
	if !strings.Contains(stdout, "cookie set --stdin") {
		t.Fatalf("cookie setup missing stdin guidance: %s", stdout)
	}
}

func TestCookieSetupDryRunLaunch(t *testing.T) {
	stdout, _, err := executeCookieCommand(t, "--json", "--dry-run", "cookie", "setup", "--launch")
	if err != nil {
		t.Fatalf("cookie setup --launch --dry-run: %v", err)
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if payload["launch"] != true {
		t.Fatalf("cookie setup dry-run did not preserve launch flag: %#v", payload)
	}
}

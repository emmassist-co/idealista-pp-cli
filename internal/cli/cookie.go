package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
	"idealista-pp-cli/internal/client"
	"idealista-pp-cli/internal/config"
)

type cookieCheckResult struct {
	Source     string `json:"source"`
	Configured bool   `json:"configured"`
	Status     string `json:"status"`
	Message    string `json:"message,omitempty"`
	HTTPStatus int    `json:"http_status,omitempty"`
	Vendor     string `json:"vendor,omitempty"`
}

func newCookieCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cookie",
		Short: "Manage the website session cookie",
		Long: `Cookie commands manage the locally supplied website session used by
the Idealista.pt website workflow.

  cookie set <value>      saves a cookie string into the local config file
  cookie setup            prints the guided browser workflow for capturing a cookie
  cookie clear            removes the config-backed cookie
  cookie source           shows where the active cookie comes from
  cookie check            validates the current cookie against the live site`,
		RunE: parentNoSubcommandRunE(flags),
	}
	cmd.AddCommand(newCookieSetupCmd(flags))
	cmd.AddCommand(newCookieSetCmd(flags))
	cmd.AddCommand(newCookieClearCmd(flags))
	cmd.AddCommand(newCookieSourceCmd(flags))
	cmd.AddCommand(newCookieCheckCmd(flags))
	return cmd
}

func newCookieSetupCmd(flags *rootFlags) *cobra.Command {
	var launch bool
	cmd := &cobra.Command{
		Use:   "setup",
		Short: "Print the browser workflow for obtaining a website session cookie",
		Example: "  idealista-pp-cli cookie setup\n" +
			"  idealista-pp-cli cookie setup --launch",
		RunE: func(cmd *cobra.Command, args []string) error {
			steps := []string{
				"Open https://www.idealista.pt and navigate to the results page you want to inspect.",
				"Open browser DevTools and switch to Network.",
				"Refresh the page and click a first-party Idealista request.",
				"Copy the full Cookie request header value.",
				"Paste it back with: pbpaste | idealista-pp-cli cookie set --stdin",
				"Validate it with: idealista-pp-cli cookie check",
			}
			if dryRunOK(flags) {
				return printJSONFiltered(cmd.OutOrStdout(), map[string]any{
					"dry_run": true,
					"launch":  launch,
					"steps":   steps,
				}, flags)
			}
			if launch {
				if err := openCookieSetupURL("https://www.idealista.pt"); err != nil {
					return err
				}
			}
			if flags.asJSON {
				return printJSONFiltered(cmd.OutOrStdout(), map[string]any{
					"launch": launch,
					"steps":  steps,
				}, flags)
			}
			if launch {
				fmt.Fprintln(cmd.OutOrStdout(), "Opened https://www.idealista.pt")
				fmt.Fprintln(cmd.OutOrStdout())
			}
			for i, step := range steps {
				fmt.Fprintf(cmd.OutOrStdout(), "%d. %s\n", i+1, step)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&launch, "launch", false, "Open Idealista.pt in the default browser before printing the setup steps")
	return cmd
}

func openCookieSetupURL(target string) error {
	var command []string
	switch runtime.GOOS {
	case "darwin":
		command = []string{"open", target}
	case "linux":
		command = []string{"xdg-open", target}
	default:
		return fmt.Errorf("cookie setup --launch is not supported on %s", runtime.GOOS)
	}
	if err := exec.Command(command[0], command[1:]...).Start(); err != nil {
		return fmt.Errorf("launching browser: %w", err)
	}
	return nil
}

func newCookieSetCmd(flags *rootFlags) *cobra.Command {
	var stdin bool
	cmd := &cobra.Command{
		Use:   "set <cookie>",
		Short: "Save a website session cookie to the config file",
		Args: func(cmd *cobra.Command, args []string) error {
			if stdin {
				if len(args) != 0 {
					return cobra.ExactArgs(0)(cmd, args)
				}
				return nil
			}
			return cobra.ExactArgs(1)(cmd, args)
		},
		Example: "  idealista-pp-cli cookie set 'datadome=...; other_cookie=...'\n  pbpaste | idealista-pp-cli cookie set --stdin",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(flags.configPath)
			if err != nil {
				return configErr(err)
			}
			cookieValue, err := cookieValueFromInput(cmd, stdin, args)
			if err != nil {
				return usageErr(err)
			}
			if dryRunOK(flags) {
				return printJSONFiltered(cmd.OutOrStdout(), map[string]any{
					"dry_run":     true,
					"would_save":  true,
					"config_path": cfg.Path,
				}, flags)
			}
			if err := cfg.SaveCookie(cookieValue); err != nil {
				return configErr(fmt.Errorf("saving cookie: %w", err))
			}
			if flags.asJSON {
				return printJSONFiltered(cmd.OutOrStdout(), map[string]any{
					"saved":       true,
					"source":      "config",
					"config_path": cfg.Path,
				}, flags)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Cookie saved to %s\n", cfg.Path)
			return nil
		},
	}
	cmd.Flags().BoolVar(&stdin, "stdin", false, "Read the cookie value from stdin and strip an optional leading 'Cookie: ' prefix")
	return cmd
}

func cookieValueFromInput(cmd *cobra.Command, readStdin bool, args []string) (string, error) {
	if !readStdin {
		return strings.TrimSpace(args[0]), nil
	}
	body, err := io.ReadAll(cmd.InOrStdin())
	if err != nil {
		return "", fmt.Errorf("reading stdin: %w", err)
	}
	value := strings.TrimSpace(string(body))
	value = strings.TrimPrefix(value, "Cookie:")
	value = strings.TrimPrefix(value, "cookie:")
	value = strings.TrimSpace(value)
	if value == "" {
		return "", fmt.Errorf("no cookie value provided on stdin")
	}
	return value, nil
}

func newCookieClearCmd(flags *rootFlags) *cobra.Command {
	return &cobra.Command{
		Use:     "clear",
		Short:   "Remove the config-backed website session cookie",
		Example: "  idealista-pp-cli cookie clear",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(flags.configPath)
			if err != nil {
				return configErr(err)
			}
			if dryRunOK(flags) {
				return printJSONFiltered(cmd.OutOrStdout(), map[string]any{
					"dry_run":     true,
					"would_clear": cfg.CookieHeader() != "",
					"config_path": cfg.Path,
				}, flags)
			}
			if err := cfg.ClearCookie(); err != nil {
				return configErr(fmt.Errorf("clearing cookie: %w", err))
			}
			envStillSet := os.Getenv("IDEALISTA_COOKIE") != ""
			if flags.asJSON {
				out := map[string]any{
					"cleared":     true,
					"config_path": cfg.Path,
				}
				if envStillSet {
					out["note"] = "IDEALISTA_COOKIE env var is still set"
				}
				return printJSONFiltered(cmd.OutOrStdout(), out, flags)
			}
			if envStillSet {
				fmt.Fprintf(cmd.OutOrStdout(), "Config cookie cleared from %s. Note: IDEALISTA_COOKIE env var is still set.\n", cfg.Path)
				return nil
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Cookie cleared from %s\n", cfg.Path)
			return nil
		},
	}
}

func newCookieSourceCmd(flags *rootFlags) *cobra.Command {
	return &cobra.Command{
		Use:     "source",
		Short:   "Show the active website session cookie source",
		Example: "  idealista-pp-cli cookie source",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(flags.configPath)
			if err != nil {
				return configErr(err)
			}
			source := cfg.CookieSource()
			configured := source != "none"
			out := map[string]any{
				"configured":           configured,
				"source":               source,
				"config_path":          cfg.Path,
				"env_overrides_config": source == "env:IDEALISTA_COOKIE",
			}
			if flags.asJSON {
				if err := printJSONFiltered(cmd.OutOrStdout(), out, flags); err != nil {
					return err
				}
				if !configured {
					return authErr(fmt.Errorf("no website session cookie configured"))
				}
				return nil
			}
			if !configured {
				fmt.Fprintln(cmd.OutOrStdout(), "No website session cookie configured")
				fmt.Fprintf(cmd.OutOrStdout(), "Set IDEALISTA_COOKIE or run %s cookie set <cookie>\n", cmd.Root().Name())
				return authErr(fmt.Errorf("no website session cookie configured"))
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Active cookie source: %s\n", source)
			fmt.Fprintf(cmd.OutOrStdout(), "Config: %s\n", cfg.Path)
			if source == "env:IDEALISTA_COOKIE" {
				fmt.Fprintln(cmd.OutOrStdout(), "Env cookie currently overrides any config-backed cookie")
			}
			return nil
		},
	}
}

func newCookieCheckCmd(flags *rootFlags) *cobra.Command {
	return &cobra.Command{
		Use:     "check",
		Short:   "Validate the current website session cookie",
		Example: "  idealista-pp-cli cookie check\n  idealista-pp-cli cookie check --json",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(flags.configPath)
			if err != nil {
				return configErr(err)
			}
			result := cookieCheckResult{
				Source:     cfg.CookieSource(),
				Configured: cfg.CookieSource() != "none",
			}
			if !result.Configured {
				result.Status = "missing"
				result.Message = "No website session cookie configured"
				if flags.asJSON {
					if err := printJSONFiltered(cmd.OutOrStdout(), result, flags); err != nil {
						return err
					}
				} else {
					fmt.Fprintln(cmd.OutOrStdout(), "No website session cookie configured")
					fmt.Fprintf(cmd.OutOrStdout(), "Set IDEALISTA_COOKIE or run %s cookie set <cookie>\n", cmd.Root().Name())
				}
				return authErr(fmt.Errorf("no website session cookie configured"))
			}
			if dryRunOK(flags) {
				return printJSONFiltered(cmd.OutOrStdout(), map[string]any{
					"dry_run":    true,
					"source":     result.Source,
					"configured": true,
					"action":     "cookie check",
				}, flags)
			}

			c, err := flags.newClient()
			if err != nil {
				return configErr(err)
			}
			assessment := assessCookieSession(cmd.Context(), c)
			result.Status = assessment.Status
			result.Message = assessment.Message
			result.HTTPStatus = assessment.HTTPStatus
			result.Vendor = assessment.Vendor

			if flags.asJSON {
				if err := printJSONFiltered(cmd.OutOrStdout(), result, flags); err != nil {
					return err
				}
			} else {
				fmt.Fprintf(cmd.OutOrStdout(), "Cookie status: %s\n", result.Status)
				fmt.Fprintf(cmd.OutOrStdout(), "Source: %s\n", result.Source)
				if result.Message != "" {
					fmt.Fprintf(cmd.OutOrStdout(), "Detail: %s\n", result.Message)
				}
			}

			switch result.Status {
			case "usable":
				return nil
			case "refresh-required":
				return authErr(fmt.Errorf("website session cookie refresh required"))
			case "missing":
				return authErr(fmt.Errorf("no website session cookie configured"))
			default:
				return apiErr(errors.New(result.Message))
			}
		},
	}
}

type siteProbeAssessment struct {
	APIReport   string
	Status      string
	Message     string
	HTTPStatus  int
	Vendor      string
	IsReachable bool
}

func assessCookieSession(ctx context.Context, c *client.Client) siteProbeAssessment {
	body, err := c.Get(ctx, "/", nil)
	return classifySiteProbe(body, err)
}

func classifySiteProbe(body []byte, err error) siteProbeAssessment {
	if err == nil {
		if vendor := looksLikeDoctorInterstitial(body); vendor != "" {
			return siteProbeAssessment{
				APIReport:   fmt.Sprintf("blocked by %s interstitial — the configured transport reached the wall. Try a different network, wait for the IP-level rate limit to clear, or check that the browser-chrome transport is bound correctly.", vendor),
				Status:      "refresh-required",
				Message:     fmt.Sprintf("%s interstitial reached at /", vendor),
				HTTPStatus:  http.StatusOK,
				Vendor:      vendor,
				IsReachable: true,
			}
		}
		return siteProbeAssessment{
			APIReport:   "reachable",
			Status:      "usable",
			Message:     "Site reachable and current session accepted",
			HTTPStatus:  http.StatusOK,
			IsReachable: true,
		}
	}

	var apiErr *client.APIError
	if errors.As(err, &apiErr) {
		if vendor := looksLikeDoctorInterstitial([]byte(apiErr.Body)); vendor != "" {
			return siteProbeAssessment{
				APIReport:   fmt.Sprintf("blocked by %s interstitial (HTTP %d) — the configured transport reached the wall.", vendor, apiErr.StatusCode),
				Status:      "refresh-required",
				Message:     fmt.Sprintf("%s interstitial rejected the session (HTTP %d)", vendor, apiErr.StatusCode),
				HTTPStatus:  apiErr.StatusCode,
				Vendor:      vendor,
				IsReachable: true,
			}
		}
		if apiErr.StatusCode == http.StatusUnauthorized || apiErr.StatusCode == http.StatusForbidden {
			return siteProbeAssessment{
				APIReport:   fmt.Sprintf("reachable (HTTP %d at /)", apiErr.StatusCode),
				Status:      "refresh-required",
				Message:     fmt.Sprintf("Site rejected the current session with HTTP %d", apiErr.StatusCode),
				HTTPStatus:  apiErr.StatusCode,
				IsReachable: true,
			}
		}
		return siteProbeAssessment{
			APIReport:   fmt.Sprintf("reachable (HTTP %d at /)", apiErr.StatusCode),
			Status:      "indeterminate",
			Message:     fmt.Sprintf("Site reachable, but session could not be verified from HTTP %d at /", apiErr.StatusCode),
			HTTPStatus:  apiErr.StatusCode,
			IsReachable: true,
		}
	}

	return siteProbeAssessment{
		APIReport:   fmt.Sprintf("unreachable: %s", err),
		Status:      "unreachable",
		Message:     fmt.Sprintf("Site unreachable: %s", err),
		IsReachable: false,
	}
}

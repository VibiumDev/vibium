package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/vibium/clicker/internal/verifier"
)

type setupCheck struct {
	Name    string `json:"name"`
	Status  string `json:"status"` // passed, failed, or skipped
	Message string `json:"message"`
	Fix     string `json:"fix,omitempty"`
}

type setupResult struct {
	Ready  bool         `json:"ready"`
	Checks []setupCheck `json:"checks"`
	Notes  []string     `json:"notes"`
}

func newSoundcheckCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use: "soundcheck", Short: "Check verifier configuration and provider tool support",
		Long:    "Check verifier configuration, then test the provider with a small synthetic tool round-trip.\nMakes up to two model requests (API charges may apply). Does not launch or change a browser.\nEnvironment files are not loaded automatically; run this in the same environment as Verify.",
		Example: "  vibium soundcheck\n  # Reports setup problems and fixes, or READY when configuration and provider checks pass.\n  vibium soundcheck --json\n  # Structured checks; exit 0 when ready, 1 when a check fails.",
		Args:    cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			config, _ := verifier.ConfigFromEnv() // Report every problem, not just the first.
			if config.Validate() == nil && !jsonOutput {
				fmt.Fprintln(cmd.ErrOrStderr(), "Checking the verifier provider (up to two model requests)...")
			}
			result := checkVerifierSetup(cmd.Context(), config, (&verifier.OpenAI{}).Probe)
			if !result.Ready && config.Validate() != nil {
				if home, err := os.UserHomeDir(); err == nil {
					if info, err := os.Stat(filepath.Join(home, ".config", "vibium", "verifier.env")); err == nil && info.Mode().IsRegular() {
						result.Notes = append(result.Notes, "Found ~/.config/vibium/verifier.env; Vibium does not load it automatically. Use export NAME=value assignments in that file. In Bash/Zsh, run: source ~/.config/vibium/verifier.env; then rerun this command in the same shell.")
					}
				}
			}
			if jsonOutput {
				envelope := jsonEnvelope{OK: result.Ready, Result: result}
				if !result.Ready {
					envelope.Error = "Verifier setup needs attention"
				}
				_ = json.NewEncoder(cmd.OutOrStdout()).Encode(envelope)
			} else {
				for _, check := range result.Checks {
					fmt.Fprintf(cmd.OutOrStdout(), "[%s] %s: %s\n", strings.ToUpper(check.Status), check.Name, check.Message)
					if check.Fix != "" {
						fmt.Fprintf(cmd.OutOrStdout(), "  Fix: %s\n", check.Fix)
					}
				}
				if result.Ready {
					fmt.Fprintln(cmd.OutOrStdout(), "\nREADY: verifier configuration and provider tool round-trip passed.")
				} else {
					fmt.Fprintln(cmd.OutOrStdout(), "\nNOT READY: fix the failed checks and rerun vibium soundcheck.")
				}
				for _, note := range result.Notes {
					fmt.Fprintln(cmd.OutOrStdout(), note)
				}
			}
			if !result.Ready {
				os.Exit(1)
			}
		},
	}
	return cmd
}

func checkVerifierSetup(ctx context.Context, config verifier.Config, probe func(context.Context, verifier.Config) error) setupResult {
	result := setupResult{Ready: true, Notes: []string{"Browser access, screenshot support, and application behavior are not tested by this soundcheck."}}
	fixes := map[string]string{
		"VIBIUM_VERIFIER_PROVIDER":         "Export VIBIUM_VERIFIER_PROVIDER=openai, or openai-compatible for a compatible server.",
		"VIBIUM_VERIFIER_MODEL":            "Export VIBIUM_VERIFIER_MODEL with a tool-capable model available to your API account.",
		"OPENAI_API_KEY":                   "Export OPENAI_API_KEY in the shell running Verify; keep its value out of chat and logs.",
		"VIBIUM_VERIFIER_BASE_URL":         "Export VIBIUM_VERIFIER_BASE_URL as the server's API base URL, such as http://localhost:1234/v1.",
		"VIBIUM_VERIFIER_REASONING_EFFORT": "Unset VIBIUM_VERIFIER_REASONING_EFFORT or choose none, minimal, low, medium, high, xhigh, or max as supported by your model.",
	}
	for _, check := range config.Checks() {
		item := setupCheck{Name: check.Variable, Status: "passed", Message: "Configuration valid (value not displayed)."}
		if check.Variable == "OPENAI_API_KEY" && config.APIKey == "" {
			item.Message = "Not set; required only for OpenAI."
		}
		if check.Variable == "VIBIUM_VERIFIER_REASONING_EFFORT" && config.ReasoningEffort == "" {
			item.Message = "Not set; using the model's default."
		}
		if check.Variable == "VIBIUM_VERIFIER_BASE_URL" && config.BaseURL == "" {
			item.Message = "No custom endpoint; OpenAI uses its default, compatible servers require one."
		}
		if check.Error != "" {
			item.Status, item.Message, item.Fix = "failed", check.Error, fixes[check.Variable]
			result.Ready = false
		}
		result.Checks = append(result.Checks, item)
	}
	provider := setupCheck{Name: "provider", Status: "skipped", Message: "Fix configuration before testing the provider."}
	if result.Ready {
		provider.Status, provider.Message = "passed", "Authentication, model access, tool call, and structured response succeeded."
		if err := probe(ctx, config); err != nil {
			result.Ready = false
			provider.Status, provider.Message, provider.Fix = "failed", err.Error(), providerSetupFix(err)
		}
	}
	result.Checks = append(result.Checks, provider)
	return result
}

func providerSetupFix(err error) string {
	switch text := err.Error(); {
	case strings.Contains(text, "HTTP 401"), strings.Contains(text, "HTTP 403"):
		return "Check the API key and its permissions for the configured endpoint and model."
	case strings.Contains(text, "model_not_found"), strings.Contains(text, "HTTP 404"):
		return "Check VIBIUM_VERIFIER_MODEL, model access, and the API base URL."
	case strings.Contains(text, "HTTP 429"):
		return "Check API quota, billing, and rate limits before retrying."
	case strings.Contains(text, "reasoning_effort"):
		return "Choose a reasoning effort supported by the model's function tools; gpt-5.6-sol requires VIBIUM_VERIFIER_REASONING_EFFORT=none."
	case strings.Contains(text, "connectivity"), strings.Contains(text, "timeout"):
		return "Check endpoint connectivity and retry when the provider is available."
	default:
		return "Check that the endpoint and model support Chat Completions function tools, max_completion_tokens, and your reasoning-effort setting."
	}
}

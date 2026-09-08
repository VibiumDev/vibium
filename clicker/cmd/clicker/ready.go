package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/vibium/clicker/internal/verifier"
)

type setupCheck struct {
	Name    string `json:"name"`
	Status  string `json:"status"`
	Message string `json:"message"`
	Fix     string `json:"fix,omitempty"`
}

type setupResult struct {
	Ready    bool           `json:"ready"`
	Scope    string         `json:"scope,omitempty"`
	Summary  string         `json:"summary,omitempty"`
	Checks   []setupCheck   `json:"checks"`
	Notes    []string       `json:"notes"`
	Browsers []readyBrowser `json:"browsers,omitempty"`
}

// Ready owns its validation: AI-only checks must work even with invalid or
// unavailable browser settings. It never addresses a daemon session.
func isReadyCommand(cmd *cobra.Command) bool {
	for c := cmd; c != nil; c = c.Parent() {
		if c.Name() == "ready" {
			return true
		}
	}
	return false
}

func newReadyCmd() *cobra.Command {
	root := &cobra.Command{
		Annotations: map[string]string{"standalone": "true"},
		Use:         "ready", Short: "Check browser installation and AI setup",
		Long:    "Check the selected local browser executable files, then test AI when configured. No browser or driver is launched.\nDoes not install browsers or change existing sessions. Missing AI is optional here; ready ai requires it.\nAI checks make up to two model requests (API charges may apply). Environment files are not loaded automatically.",
		Example: "  vibium ready\n  # Checks browser installation and configured AI; reports fixes or READY.\n  vibium ready --json\n  # Structured readiness results; exit 0 when requested checks pass, otherwise 1.",
		Args:    cobra.NoArgs,
	}
	ai := &cobra.Command{
		Use: "ai [provider]", Short: "Test AI configuration and a provider tool round-trip without a browser",
		Long:      "Require valid AI configuration and test authentication, model access, tool calling, and a structured response.\nMakes up to two model requests (API charges may apply). Does not launch a browser or load env files.\nChanging provider requires --model; per-call options do not change defaults.",
		Example:   "  vibium ready ai\n  # Tests the configured provider and model.\n  vibium ready ai anthropic --model your-model\n  # Tests Anthropic with the supplied model and ANTHROPIC_API_KEY.\n  vibium ready ai --json\n  # Prints the provider checks as JSON.",
		Args:      cobra.MaximumNArgs(1),
		ValidArgs: []string{"openai", "anthropic", "google", "openai-compatible", "local"},
	}
	browserCmd := &cobra.Command{
		Use: "browser [engine]", Short: "Check installed browser executable files without launching them",
		Long:    "List discovered Vibium browser installations and check the selected Chrome or Firefox executable files.\nDoes not launch browsers or drivers, test BiDi connectivity, install browsers, contact AI, or touch the active daemon session.\nUse --engine and --channel for the same selection as live commands.",
		Example: "  vibium ready browser\n  # Checks the default/selected installation; other discovered installations are informational.\n  vibium ready browser firefox --channel beta\n  # Checks Firefox beta installation without launching it or changing defaults.\n  vibium ready browser chrome --json\n  # Structured installation results; browser connection is reported as skipped.",
		Args:    cobra.MaximumNArgs(1), ValidArgs: []string{"chrome", "firefox"},
	}
	root.AddCommand(ai, browserCmd)
	addModelFlags(root)
	addModelFlags(ai)
	for _, cmd := range []*cobra.Command{root, ai, browserCmd} {
		cmd.Run = func(cmd *cobra.Command, args []string) {
			ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
			defer stop()
			cmd.SetContext(ctx)
			result := runReadiness(cmd, args, (&verifier.Model{}).Probe)
			writeReadiness(cmd, result)
			if !result.Ready {
				os.Exit(1)
			}
		}
	}
	return root
}

func runReadiness(cmd *cobra.Command, args []string, aiProbe func(context.Context, verifier.Config) error) setupResult {
	scope := cmd.Name()
	if scope == "ready" {
		scope = "all"
	}
	result := setupResult{Ready: true, Scope: scope, Checks: []setupCheck{}, Notes: []string{}}
	if scope != "ai" {
		part := checkBrowserSetup(cmd, args)
		result.Ready = part.Ready
		result.Checks = append(result.Checks, part.Checks...)
		result.Notes = append(result.Notes, part.Notes...)
		result.Browsers = part.Browsers
	}
	aiRequested := scope == "ai"
	if scope != "browser" {
		overrides := modelOverrides(cmd)
		if scope == "ai" && len(args) == 1 {
			if overrides.Provider != nil && *overrides.Provider != args[0] {
				result.Ready = false
				result.Summary = "Choose one provider and rerun vibium ready ai."
				result.Checks = append(result.Checks, setupCheck{"provider", "failed", "Provider argument conflicts with --provider.", "Choose one provider."})
				return result
			}
			overrides.Provider = &args[0]
		}
		aiRequested = aiRequested || overrides.Provider != nil || overrides.Model != nil || overrides.BaseURL != nil || overrides.ReasoningEffort != nil
		for _, key := range []string{"PROVIDER", "MODEL", "BASE_URL", "REASONING_EFFORT"} {
			aiRequested = aiRequested || os.Getenv("VIBIUM_AI_"+key) != ""
		}
		if aiRequested {
			config, _ := verifier.ResolveConfig("check", overrides)
			if config.Validate() == nil && !jsonOutput {
				fmt.Fprintln(cmd.ErrOrStderr(), "Testing AI provider (up to two model requests)...")
			}
			part := checkVerifierSetup(cmd.Context(), config, aiProbe)
			result.Ready = result.Ready && part.Ready
			result.Checks = append(result.Checks, part.Checks...)
			result.Notes = append(result.Notes, part.Notes...)
			if config.Validate() != nil {
				result.Notes = append(result.Notes, readyEnvNote()...)
			}
		} else {
			result.Checks = append(result.Checks, setupCheck{"ai", "skipped", "AI is not configured; optional for direct browser automation.", "For Run or Check, configure VIBIUM_AI_PROVIDER and VIBIUM_AI_MODEL, then run vibium ready ai."})
			result.Notes = append(result.Notes, readyEnvNote()...)
		}
	}
	retry := "vibium ready"
	if scope != "all" {
		retry += " " + scope
	}
	result.Summary = "Fix the failed checks and rerun " + retry + "."
	if result.Ready {
		switch {
		case scope == "ai":
			result.Summary = "AI configuration and provider tool round-trip passed."
		case scope == "browser" || !aiRequested:
			result.Summary = "Browser installation checks passed; launch and BiDi connectivity were not tested."
		default:
			result.Summary = "Browser installation and AI checks passed; browser launch and BiDi connectivity were not tested."
		}
	}
	return result
}

func readyEnvNote() []string {
	if home, err := os.UserHomeDir(); err == nil {
		if info, err := os.Stat(filepath.Join(home, ".config", "vibium", "ai.env")); err == nil && info.Mode().IsRegular() {
			return []string{"Found ~/.config/vibium/ai.env; Vibium does not load it automatically. Use export NAME=value assignments in that file. In Bash/Zsh, run: source ~/.config/vibium/ai.env; then rerun readiness in the same shell."}
		}
	}
	return nil
}

func writeReadiness(cmd *cobra.Command, result setupResult) {
	if jsonOutput {
		envelope := jsonEnvelope{OK: result.Ready, Result: result}
		if !result.Ready {
			envelope.Error = "Requested setup checks failed"
		}
		_ = json.NewEncoder(cmd.OutOrStdout()).Encode(envelope)
		return
	}
	for _, b := range result.Browsers {
		if b.Installed {
			fmt.Fprintf(cmd.OutOrStdout(), "Installed: %s (%s) — %s\n", b.Engine, b.Channel, b.Path)
		}
	}
	for _, check := range result.Checks {
		fmt.Fprintf(cmd.OutOrStdout(), "[%s] %s: %s\n", strings.ToUpper(check.Status), check.Name, check.Message)
		if check.Fix != "" {
			fmt.Fprintf(cmd.OutOrStdout(), "  Fix: %s\n", check.Fix)
		}
	}
	label := "READY"
	if !result.Ready {
		label = "NOT READY"
	}
	fmt.Fprintf(cmd.OutOrStdout(), "\n%s: %s\n", label, result.Summary)
	for _, note := range result.Notes {
		fmt.Fprintln(cmd.OutOrStdout(), note)
	}
}

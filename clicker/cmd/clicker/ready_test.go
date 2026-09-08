package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/vibium/clicker/internal/verifier"
)

func readyTestCommand(t *testing.T, scope string) *cobra.Command {
	t.Helper()
	for _, key := range []string{"VIBIUM_AI_PROVIDER", "VIBIUM_AI_MODEL", "VIBIUM_AI_BASE_URL", "VIBIUM_AI_REASONING_EFFORT", "VIBIUM_ENGINE_CHANNEL", "VIBIUM_ENGINE_VERSION", "VIBIUM_ENGINE_PATH"} {
		t.Setenv(key, "")
	}
	t.Setenv("VIBIUM_CACHE_DIR", t.TempDir())
	oldEngine, oldChannel, oldJSON := engineName, engineChannel, jsonOutput
	t.Cleanup(func() { engineName, engineChannel, jsonOutput = oldEngine, oldChannel, oldJSON })
	engineName, engineChannel, jsonOutput = "firefox", "beta", true
	exe := filepath.Join(t.TempDir(), "firefox")
	if err := os.WriteFile(exe, []byte("fixture"), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("VIBIUM_ENGINE_PATH", exe)
	root := &cobra.Command{Use: "vibium"}
	root.PersistentFlags().StringVar(&engineName, "engine", engineName, "")
	root.PersistentFlags().StringVar(&engineChannel, "channel", engineChannel, "")
	ready := newReadyCmd()
	root.AddCommand(ready)
	if scope == "all" {
		return ready
	}
	cmd, _, err := ready.Find([]string{scope})
	if err != nil {
		t.Fatal(err)
	}
	return cmd
}

func TestReadyOptionalAIAndRequiredScopes(t *testing.T) {
	for _, scope := range []string{"all", "browser", "ai"} {
		t.Run(scope, func(t *testing.T) {
			cmd := readyTestCommand(t, scope)
			result := runReadiness(cmd, nil, func(context.Context, verifier.Config) error { t.Fatal("unconfigured AI was contacted"); return nil })
			if result.Ready != (scope != "ai") {
				t.Fatalf("unexpected readiness: %+v", result)
			}
			if (len(result.Browsers) == 0) != (scope == "ai") {
				t.Fatal("wrong browser scope")
			}
			for _, check := range result.Checks {
				if check.Name == "browser.connection" && check.Status != "skipped" {
					t.Fatal("readiness must not claim to test a browser connection")
				}
			}
			if os.Getenv("VIBIUM_ENGINE_CHANNEL") != "" {
				t.Fatal("channel defaults changed")
			}
		})
	}
}

func TestReadyReportsBrowserFailureAndStillProbesConfiguredAI(t *testing.T) {
	cmd := readyTestCommand(t, "all")
	t.Setenv("VIBIUM_AI_PROVIDER", "local")
	t.Setenv("VIBIUM_AI_MODEL", "fixture")
	t.Setenv("VIBIUM_ENGINE_PATH", filepath.Join(t.TempDir(), "missing-firefox"))
	called := false
	result := runReadiness(cmd, nil, func(context.Context, verifier.Config) error { called = true; return nil })
	if result.Ready || !called {
		t.Fatalf("unexpected readiness: %+v", result)
	}
	found := false
	for _, c := range result.Checks {
		if c.Name == "browser.installation" {
			found = c.Status == "failed" && c.Fix != ""
		}
	}
	if !found {
		t.Fatal("browser failure was not explained")
	}
}

func TestReadyProviderSelectionDoesNotMutateDefaults(t *testing.T) {
	cmd := readyTestCommand(t, "ai")
	t.Setenv("VIBIUM_AI_PROVIDER", "google")
	t.Setenv("VIBIUM_AI_MODEL", "old-model")
	if err := cmd.ParseFlags([]string{"--model", "selected-model"}); err != nil {
		t.Fatal(err)
	}
	result := runReadiness(cmd, []string{"local"}, func(_ context.Context, c verifier.Config) error {
		if c.Provider != "local" || c.Model != "selected-model" {
			t.Fatal("provider override lost")
		}
		return nil
	})
	if !result.Ready || os.Getenv("VIBIUM_AI_PROVIDER") != "google" || os.Getenv("VIBIUM_AI_MODEL") != "old-model" {
		t.Fatal("override mutated defaults")
	}
}

package main

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/vibium/clicker/internal/verifier"
)

func TestReadyAIReportsAllConfigurationProblemsWithoutProbing(t *testing.T) {
	secret := "secret-do-not-print"
	config := verifier.Config{Provider: "openai", BaseURL: "https://user:" + secret + "@example.test", ReasoningEffort: secret}
	result := checkVerifierSetup(context.Background(), config, func(context.Context, verifier.Config) error {
		t.Fatal("contacted provider with invalid configuration")
		return nil
	})
	failed := map[string]bool{}
	for _, check := range result.Checks {
		if check.Status == "failed" {
			failed[check.Name] = true
			if check.Fix == "" {
				t.Error("missing fix")
			}
		}
	}
	for _, name := range []string{"VIBIUM_AI_MODEL", "OPENAI_API_KEY", "VIBIUM_AI_BASE_URL", "VIBIUM_AI_REASONING_EFFORT"} {
		if !failed[name] {
			t.Errorf("did not report %s", name)
		}
	}
	data, _ := json.Marshal(result)
	if result.Ready || strings.Contains(string(data), secret) || result.Checks[len(result.Checks)-1].Status != "skipped" {
		t.Fatalf("bad report: %s", data)
	}
}

func TestReadyAIProviderOutcomeAndFixes(t *testing.T) {
	for _, tc := range []struct{ problem, fix string }{
		{"", ""}, {"verifier provider returned HTTP 401", "API key"},
		{"verifier provider returned HTTP 429 (insufficient_quota)", "billing"},
		{"verifier provider returned HTTP 404 (model_not_found)", "model access"},
		{"verifier provider returned HTTP 400 in reasoning_effort", "=none"},
		{"verification timeout", "connectivity"}, {"invalid JSON verdict", "function tools"},
	} {
		t.Run(tc.problem, func(t *testing.T) {
			config := verifier.Config{Provider: "openai", Model: "test-model", APIKey: "secret"}
			called := false
			result := checkVerifierSetup(context.Background(), config, func(_ context.Context, received verifier.Config) error {
				called = true
				if received != config {
					t.Error("configuration changed")
				}
				if tc.problem != "" {
					return errors.New(tc.problem)
				}
				return nil
			})
			check := result.Checks[len(result.Checks)-1]
			if !called || result.Ready != (tc.problem == "") || !strings.Contains(check.Fix, tc.fix) {
				t.Fatalf("unexpected report: %+v", result)
			}
		})
	}
}

func TestReadyAIRequiresProviderBeforeAssessingDependentSettings(t *testing.T) {
	for _, provider := range []string{"", "unsupported-secret-provider"} {
		for _, key := range []string{"", "secret-key"} {
			config := verifier.Config{Provider: provider, APIKey: key}
			result := checkVerifierSetup(context.Background(), config, func(context.Context, verifier.Config) error {
				t.Fatal("contacted provider without prerequisites")
				return nil
			})
			if result.Ready {
				t.Fatal("missing configuration reported ready")
			}
			expected := []struct{ name, status string }{
				{"VIBIUM_AI_PROVIDER", "failed"}, {"VIBIUM_AI_MODEL", "failed"},
				{"credentials", "skipped"}, {"VIBIUM_AI_BASE_URL", "skipped"},
				{"VIBIUM_AI_REASONING_EFFORT", "skipped"}, {"provider", "skipped"},
			}
			if len(result.Checks) != len(expected) {
				t.Fatalf("unexpected checks: %+v", result.Checks)
			}
			for i, want := range expected {
				if result.Checks[i].Name != want.name || result.Checks[i].Status != want.status {
					t.Fatalf("unexpected check: %+v; want %+v", result.Checks[i], want)
				}
			}
			data, _ := json.Marshal(result)
			for _, secret := range []string{"unsupported-secret-provider", "secret-key", "OPENAI_API_KEY"} {
				if strings.Contains(string(data), secret) {
					t.Fatalf("unexpected value or assumed provider in output: %s", data)
				}
			}
		}
	}
}

func TestReadyAIXAIOauthCheckNameAndAuthFix(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("VIBIUM_CONFIG_DIR", t.TempDir())
	t.Setenv("XAI_API_KEY", "")
	missing := checkVerifierSetup(context.Background(), verifier.Config{Provider: "xai", Model: "grok-4"}, func(context.Context, verifier.Config) error {
		t.Fatal("probed without credentials")
		return nil
	})
	found := false
	for _, check := range missing.Checks {
		if check.Name == "XAI_API_KEY" {
			t.Fatal("missing xAI oauth still named XAI_API_KEY")
		}
		if check.Name == "Grok login" {
			found = check.Status == "failed" && strings.Contains(check.Fix, "vibium login xai") && strings.Contains(check.Fix, "XAI_API_KEY")
		}
	}
	if missing.Ready || !found {
		t.Fatalf("missing oauth report: %+v", missing)
	}

	oauth := verifier.Config{Provider: "xai", Model: "grok-4", APIKey: "oauth-token", CredentialSource: verifier.CredentialOAuth}
	passed := checkVerifierSetup(context.Background(), oauth, func(context.Context, verifier.Config) error { return nil })
	namedKey := false
	namedLogin := false
	for _, check := range passed.Checks {
		if check.Name == "XAI_API_KEY" {
			namedKey = true
		}
		if check.Name == "Grok login" && check.Status == "passed" && strings.Contains(check.Message, "not displayed") {
			namedLogin = true
		}
	}
	if namedKey || !namedLogin || !passed.Ready {
		t.Fatalf("oauth ready report: %+v", passed)
	}

	denied := checkVerifierSetup(context.Background(), oauth, func(context.Context, verifier.Config) error {
		return errors.New("verifier provider returned HTTP 401")
	})
	fix := denied.Checks[len(denied.Checks)-1].Fix
	if denied.Ready || !strings.Contains(fix, "vibium login xai") || !strings.Contains(fix, "XAI_API_KEY") {
		t.Fatalf("401 fix: %s", fix)
	}
}

func TestReadyAIStillReportsMalformedSettingsWithoutProvider(t *testing.T) {
	result := checkVerifierSetup(context.Background(), verifier.Config{BaseURL: "not-a-url", ReasoningEffort: "invalid"}, func(context.Context, verifier.Config) error {
		t.Fatal("contacted provider with malformed settings")
		return nil
	})
	for _, name := range []string{"VIBIUM_AI_BASE_URL", "VIBIUM_AI_REASONING_EFFORT"} {
		found := false
		for _, check := range result.Checks {
			if check.Name == name {
				found = check.Status == "failed" && check.Fix != ""
			}
		}
		if !found {
			t.Fatalf("did not report invalid %s: %+v", name, result)
		}
	}
}

package main

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/vibium/clicker/internal/ai"
)

func TestReadyAIReportsAllConfigurationProblemsWithoutProbing(t *testing.T) {
	secret := "secret-do-not-print"
	config := ai.Config{Provider: "openai", BaseURL: "https://user:" + secret + "@example.test", ReasoningEffort: secret}
	result := checkVerifierSetup(context.Background(), config, func(context.Context, ai.Config) error {
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
		// The hint names the configured model, not the wizard default (#621).
		{"verifier provider returned HTTP 400 in reasoning_effort", "test-model"},
		{"verifier provider returned HTTP 400 (unsupported_value) in reasoning_effort", "none"},
		{"verification timeout", "connectivity"}, {"invalid JSON verdict", "function tools"},
	} {
		t.Run(tc.problem, func(t *testing.T) {
			config := ai.Config{Provider: "openai", Model: "test-model", APIKey: "secret"}
			called := false
			result := checkVerifierSetup(context.Background(), config, func(_ context.Context, received ai.Config) error {
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

// The reasoning-effort fix names the configured model and a remedy that is
// valid for it; gpt-6.1 models get a plain statement instead of effort
// advice that cannot work (#621).
func TestReadyAIReasoningEffortFixMatchesModel(t *testing.T) {
	effortErr := errors.New("verifier provider returned HTTP 400 in reasoning_effort")
	for _, tc := range []struct{ provider, model, effort, want string }{
		{"openai", "gpt-6.1-sol", "", "Responses API"},
		{"openai", "gpt-6.1-sol", "low", "gpt-6.1-sol"},
		{"openai", "gpt-5.6-sol", "", "VIBIUM_AI_REASONING_EFFORT=none"},
		{"openai", "gpt-6-sol", "low", "VIBIUM_AI_REASONING_EFFORT=none"},
		// Already at the known effort: generic advice, naming the model.
		{"openai", "gpt-5.6-sol", "none", "reasoning effort gpt-5.6-sol supports"},
		{"openai", "other-model", "", "reasoning effort other-model supports"},
		// Model knowledge applies to the native provider only.
		{"openai-compatible", "gpt-6.1-sol", "", "reasoning effort gpt-6.1-sol supports"},
	} {
		fix := providerSetupFixForConfig(effortErr, ai.Config{Provider: tc.provider, Model: tc.model, ReasoningEffort: tc.effort})
		if !strings.Contains(fix, tc.want) {
			t.Errorf("%s %s effort=%q: fix %q does not contain %q", tc.provider, tc.model, tc.effort, fix, tc.want)
		}
	}
}

func TestReadyAIRequiresProviderBeforeAssessingDependentSettings(t *testing.T) {
	for _, provider := range []string{"", "unsupported-secret-provider"} {
		for _, key := range []string{"", "secret-key"} {
			config := ai.Config{Provider: provider, APIKey: key}
			result := checkVerifierSetup(context.Background(), config, func(context.Context, ai.Config) error {
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

func TestReadyAIStillReportsMalformedSettingsWithoutProvider(t *testing.T) {
	result := checkVerifierSetup(context.Background(), ai.Config{BaseURL: "not-a-url", ReasoningEffort: "invalid"}, func(context.Context, ai.Config) error {
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

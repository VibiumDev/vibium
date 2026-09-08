package main

import (
	"context"
	"slices"
	"strings"

	"github.com/vibium/clicker/internal/verifier"
)

func checkVerifierSetup(ctx context.Context, config verifier.Config, probe func(context.Context, verifier.Config) error) setupResult {
	prefix := config.Prefix()
	result := setupResult{Ready: true, Notes: []string{"AI readiness does not test browser access, screenshot support, or application behavior."}}
	fixes := map[string]string{
		prefix + "PROVIDER":         "Export " + prefix + "PROVIDER as openai, anthropic, google, openai-compatible, or local.",
		prefix + "MODEL":            "Set a tool-capable model with --model or export " + prefix + "MODEL. When changing provider, supply --model explicitly.",
		config.CredentialVariable(): "Export " + config.CredentialVariable() + " in the shell running the command; keep its value out of chat and logs.",
		prefix + "BASE_URL":         "Export " + prefix + "BASE_URL as the server's API base URL, such as http://localhost:1234/v1.",
		prefix + "REASONING_EFFORT": "Unset " + prefix + "REASONING_EFFORT for Anthropic/Google, or choose an effort supported by your OpenAI-compatible model.",
	}
	checks := config.Checks()
	providerValid := false
	for _, check := range checks {
		if check.Variable == prefix+"PROVIDER" {
			providerValid = check.Error == ""
		}
	}
	// Present prerequisites before settings that depend on them. Keep shared
	// validation unchanged for Run and Check.
	order := map[string]int{prefix + "PROVIDER": 0, prefix + "MODEL": 1, config.CredentialVariable(): 2, prefix + "BASE_URL": 3, prefix + "REASONING_EFFORT": 4}
	slices.SortStableFunc(checks, func(a, b verifier.ConfigCheck) int { return order[a.Variable] - order[b.Variable] })
	for _, check := range checks {
		item := setupCheck{Name: check.Variable, Status: "passed", Message: "Configuration valid (value not displayed)."}
		if check.Variable == config.CredentialVariable() && config.APIKey == "" {
			item.Message = "Not set; optional for openai-compatible and local."
		}
		if check.Variable == prefix+"REASONING_EFFORT" && config.ReasoningEffort == "" {
			item.Message = "Not set; no reasoning-effort override."
		}
		if check.Variable == prefix+"BASE_URL" && config.BaseURL == "" {
			item.Message = "No custom endpoint; using the provider default when available."
		}
		if !providerValid {
			switch check.Variable {
			case config.CredentialVariable():
				item.Name, item.Status, item.Message = "credentials", "skipped", "Select a supported provider before checking its API key requirements."
			case prefix + "BASE_URL", prefix + "REASONING_EFFORT":
				item.Status, item.Message = "skipped", "Select a supported provider before checking this setting."
			}
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
			provider.Status, provider.Message, provider.Fix = "failed", err.Error(), providerSetupFixForConfig(err, config)
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
		return "Check VIBIUM_AI_MODEL, model access, and the API base URL."
	case strings.Contains(text, "HTTP 429"):
		return "Check API quota, billing, and rate limits before retrying."
	case strings.Contains(text, "reasoning_effort"):
		return "Choose a reasoning effort supported by the model's function tools; gpt-5.6-sol requires VIBIUM_AI_REASONING_EFFORT=none."
	case strings.Contains(text, "connectivity"), strings.Contains(text, "timeout"):
		return "Check endpoint connectivity and retry when the provider is available."
	default:
		return "Check that the endpoint and model support Chat Completions function tools, max_completion_tokens, and your reasoning-effort setting."
	}
}

func providerSetupFixForConfig(err error, config verifier.Config) string {
	fix := strings.ReplaceAll(providerSetupFix(err), "VIBIUM_AI_", config.Prefix())
	if config.Provider == "anthropic" || config.Provider == "google" {
		if strings.Contains(fix, "Chat Completions") {
			return "Check that the endpoint and model support native tool calling and image input. Leave " + config.Prefix() + "REASONING_EFFORT unset."
		}
	}
	return fix
}

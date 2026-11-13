package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSharedConfigurationAndLocalAlias(t *testing.T) {
	for _, key := range []string{"VIBIUM_AI_PROVIDER", "VIBIUM_AI_MODEL", "VIBIUM_AI_BASE_URL", "VIBIUM_AI_REASONING_EFFORT", "OPENAI_API_KEY"} {
		t.Setenv(key, "")
	}
	t.Setenv("VIBIUM_AI_PROVIDER", "openai")
	t.Setenv("VIBIUM_AI_MODEL", "check-model")
	t.Setenv("OPENAI_API_KEY", "shared-key")
	for _, role := range []string{"run", "check"} {
		c, err := ConfigForRole(role)
		if err != nil || c.Provider != "openai" || c.Model != "check-model" || c.APIKey != "shared-key" {
			t.Fatal("operations did not share AI defaults")
		}
	}
	t.Setenv("OPENAI_API_KEY", "")
	t.Setenv("VIBIUM_AI_PROVIDER", "local")
	t.Setenv("VIBIUM_AI_MODEL", "local-model")
	config, err := ConfigForRole("run")
	if err != nil || config.Endpoint() != "http://127.0.0.1:8080/v1" || config.APIKey != "" || config.Model != "local-model" {
		t.Fatalf("local config: %+v %v", config, err)
	}
	for _, row := range []struct{ provider, key string }{{"anthropic", "ANTHROPIC_API_KEY"}, {"google", "GOOGLE_API_KEY"}, {"xai", "XAI_API_KEY"}} {
		t.Setenv("VIBIUM_AI_PROVIDER", row.provider)
		t.Setenv(row.key, "native-secret")
		t.Setenv("OPENAI_API_KEY", "wrong-key")
		config, err := ConfigForRole("run")
		if err != nil || config.APIKey != "native-secret" || config.CredentialVariable() != row.key {
			t.Fatal("wrong native credential selected")
		}
	}
}

func TestGoogleGeminiKeyFallback(t *testing.T) {
	for _, key := range []string{"VIBIUM_AI_PROVIDER", "VIBIUM_AI_MODEL", "VIBIUM_AI_BASE_URL", "VIBIUM_AI_REASONING_EFFORT", "GOOGLE_API_KEY", "GEMINI_API_KEY"} {
		t.Setenv(key, "")
	}
	t.Setenv("VIBIUM_AI_PROVIDER", "google")
	t.Setenv("VIBIUM_AI_MODEL", "gemini-2.5-pro")
	_, err := ConfigForRole("run")
	if err == nil || !strings.Contains(err.Error(), "GOOGLE_API_KEY") || !strings.Contains(err.Error(), "GEMINI_API_KEY") {
		t.Fatalf("missing google key should name both variables: %v", err)
	}
	t.Setenv("GEMINI_API_KEY", "gemini-secret")
	c, err := ConfigForRole("run")
	if err != nil || c.APIKey != "gemini-secret" {
		t.Fatalf("GEMINI_API_KEY fallback not applied: %v", err)
	}
	if name, problem := c.credentialCheck(); name != "GEMINI_API_KEY" || problem != "" {
		t.Fatalf("fallback should be reported as GEMINI_API_KEY: %q %q", name, problem)
	}
	t.Setenv("GOOGLE_API_KEY", "google-secret")
	c, err = ConfigForRole("run")
	if err != nil || c.APIKey != "google-secret" {
		t.Fatalf("GOOGLE_API_KEY should win over GEMINI_API_KEY: %v", err)
	}
	if name, problem := c.credentialCheck(); name != "GOOGLE_API_KEY" || problem != "" {
		t.Fatalf("canonical key should be reported as GOOGLE_API_KEY: %q %q", name, problem)
	}
}

func TestXAINativeDefaults(t *testing.T) {
	c := Config{Provider: "xai", Model: "grok-4", APIKey: "k"}
	if c.Endpoint() != "https://api.x.ai/v1" || c.CredentialVariable() != "XAI_API_KEY" {
		t.Fatalf("xai defaults: %+v", c)
	}
	if err := c.Validate(); err != nil {
		t.Fatal(err)
	}
	c.APIKey = ""
	if err := c.Validate(); err == nil || !strings.Contains(err.Error(), "XAI_API_KEY") {
		t.Fatalf("missing xai key: %v", err)
	}
}

func TestNativeProviderErrorsAreBoundedAndSecretSafe(t *testing.T) {
	for _, provider := range []string{"anthropic", "google"} {
		for _, scenario := range []string{"auth", "refused", "malformed", "large"} {
			t.Run(provider+"/"+scenario, func(t *testing.T) {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					switch scenario {
					case "auth":
						w.WriteHeader(401)
						fmt.Fprint(w, `{"error":{"message":"PRIVATE-SECRET"}}`)
					case "refused":
						fmt.Fprint(w, `{"stop_reason":"refusal","candidates":[{"finishReason":"SAFETY"}]}`)
					case "large":
						fmt.Fprint(w, strings.Repeat("PRIVATE-SECRET", 100000))
					default:
						fmt.Fprint(w, "PRIVATE-SECRET")
					}
				}))
				defer server.Close()
				err := (&Model{}).Probe(context.Background(), Config{Provider: provider, Model: "fixture", BaseURL: server.URL, APIKey: "key"})
				if err == nil || strings.Contains(err.Error(), "PRIVATE-SECRET") {
					t.Fatalf("unsafe provider result: %v", err)
				}
			})
		}
	}
}

func TestNativeImageAndToolResultTranslation(t *testing.T) {
	call := toolCall{ID: "one", Type: "function", Signature: "opaque"}
	call.Function.Name = "browser_screenshot"
	call.Function.Arguments = "{}"
	messages := []message{{Role: "system", Content: "instructions"}, {Role: "assistant", ToolCalls: []toolCall{call}}, {Role: "tool", ToolCallID: "one", Content: "image captured"}, {Role: "user", Content: []interface{}{map[string]interface{}{"type": "image_url", "image_url": map[string]string{"url": "data:image/png;base64,aGVsbG8="}}}}}
	for _, google := range []bool{false, true} {
		_, history, err := nativeHistory(messages, google)
		if err != nil {
			t.Fatal(err)
		}
		data, _ := json.Marshal(history)
		if !strings.Contains(string(data), "aGVsbG8=") || !strings.Contains(string(data), "image/png") {
			t.Fatal("native provider lost requested screenshot")
		}
	}
}

func TestNativeParallelResultsPrecedeScreenshot(t *testing.T) {
	first := toolCall{ID: "first", Type: "function"}
	first.Function.Name, first.Function.Arguments = "browser_screenshot", "{}"
	second := first
	second.ID, second.Function.Name = "second", "browser_map"
	messages := []message{
		{Role: "assistant", ToolCalls: []toolCall{first, second}},
		{Role: "tool", ToolCallID: "first", Content: "screenshot"},
		{Role: "user", Content: []interface{}{map[string]interface{}{"type": "image_url", "image_url": map[string]string{"url": "data:image/png;base64,aGVsbG8="}}}},
		{Role: "tool", ToolCallID: "second", Content: "map"},
	}
	for _, google := range []bool{false, true} {
		_, history, err := nativeHistory(messages, google)
		if err != nil || len(history) != 2 {
			t.Fatalf("native history: %+v, %v", history, err)
		}
		parts := history[1].Content
		if google {
			parts = history[1].Parts
		}
		if len(parts) != 3 {
			t.Fatalf("lost parallel results or image: %+v", parts)
		}
		if google {
			if parts[0]["functionResponse"] == nil || parts[1]["functionResponse"] == nil || parts[2]["inlineData"] == nil {
				t.Fatalf("wrong Google result order: %+v", parts)
			}
		} else if parts[0]["tool_use_id"] != "first" || parts[1]["tool_use_id"] != "second" || parts[2]["type"] != "image" {
			t.Fatalf("wrong Anthropic result order: %+v", parts)
		}
	}
}

// The --ai-base-url hint names the setting only when a base URL is in play;
// a native provider error must not suggest it.
func TestInvalidProviderResponseHintsBaseURLOnlyWhenSet(t *testing.T) {
	if err := invalidProviderResponse(Config{Provider: "openai"}, "text/html", "a chat-completions message"); strings.Contains(err.Error(), "--ai-base-url") {
		t.Fatal("hint should name the override only when one is in effect")
	}
}

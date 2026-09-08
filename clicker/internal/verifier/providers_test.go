package verifier

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
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
	for _, row := range []struct{ provider, key string }{{"anthropic", "ANTHROPIC_API_KEY"}, {"google", "GOOGLE_API_KEY"}} {
		t.Setenv("VIBIUM_AI_PROVIDER", row.provider)
		t.Setenv(row.key, "native-secret")
		t.Setenv("OPENAI_API_KEY", "wrong-key")
		config, err := ConfigForRole("run")
		if err != nil || config.APIKey != "native-secret" || config.CredentialVariable() != row.key {
			t.Fatal("wrong native credential selected")
		}
	}
}

func TestNativeProviderProbeAndFreshCheck(t *testing.T) {
	for _, provider := range []string{"anthropic", "google", "local", "openai-compatible"} {
		t.Run(provider, func(t *testing.T) {
			calls := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				raw, _ := io.ReadAll(r.Body)
				if strings.Contains(string(raw), "PRIVATE-THINKING") || strings.Contains(string(raw), "provider-secret") {
					t.Error("leaked private content into provider history")
				}
				var body map[string]interface{}
				if json.Unmarshal(raw, &body) != nil {
					t.Error("bad request")
				}
				second := false
				var toolName, text string
				switch provider {
				case "anthropic":
					if r.URL.Path != "/v1/messages" || r.Header.Get("x-api-key") != "provider-secret" || r.Header.Get("anthropic-version") != "2023-06-01" {
						t.Error("wrong Anthropic transport")
					}
					if _, ok := body["system"].(string); !ok {
						t.Error("missing system instructions")
					}
					tools := body["tools"].([]interface{})
					toolName = tools[0].(map[string]interface{})["name"].(string)
					if tools[0].(map[string]interface{})["input_schema"] == nil {
						t.Error("missing native schema")
					}
					turns := body["messages"].([]interface{})
					for _, turn := range turns {
						for _, item := range turn.(map[string]interface{})["content"].([]interface{}) {
							p := item.(map[string]interface{})
							if p["type"] == "tool_result" {
								second = true
								text = p["content"].(string)
								if p["tool_use_id"] != "native-call" {
									t.Error("unmatched tool result")
								}
							}
						}
					}
					if !second {
						json.NewEncoder(w).Encode(map[string]interface{}{"stop_reason": "tool_use", "content": []interface{}{map[string]string{"type": "thinking", "thinking": "PRIVATE-THINKING"}, map[string]interface{}{"type": "tool_use", "id": "native-call", "name": toolName, "input": map[string]interface{}{}}}})
						return
					}
				case "google":
					if r.URL.Path != "/v1/models/fixture:generateContent" || r.Header.Get("x-goog-api-key") != "provider-secret" || r.URL.RawQuery != "" {
						t.Error("wrong Google transport")
					}
					defs := body["tools"].([]interface{})[0].(map[string]interface{})["functionDeclarations"].([]interface{})
					toolName = defs[0].(map[string]interface{})["name"].(string)
					if defs[0].(map[string]interface{})["parametersJsonSchema"] == nil {
						t.Error("missing JSON schema")
					}
					signed := false
					for _, turn := range body["contents"].([]interface{}) {
						for _, item := range turn.(map[string]interface{})["parts"].([]interface{}) {
							p := item.(map[string]interface{})
							if p["thoughtSignature"] == "opaque-signature" {
								signed = true
							}
							if f, ok := p["functionResponse"].(map[string]interface{}); ok {
								second = true
								text = f["response"].(map[string]interface{})["output"].(string)
								if f["name"] != toolName || f["id"] != "google-native" {
									t.Error("unmatched function result")
								}
							}
						}
					}
					if second && !signed {
						t.Error("dropped Gemini function-call signature")
					}
					if !second {
						json.NewEncoder(w).Encode(map[string]interface{}{"candidates": []interface{}{map[string]interface{}{"finishReason": "STOP", "content": map[string]interface{}{"role": "model", "parts": []interface{}{map[string]interface{}{"thought": true, "text": "PRIVATE-THINKING"}, map[string]interface{}{"functionCall": map[string]interface{}{"name": toolName, "id": "google-native", "args": map[string]interface{}{}}, "thoughtSignature": "opaque-signature"}}}}}})
						return
					}
				default:
					if r.URL.Path != "/v1/chat/completions" {
						t.Error("compatible adapter path changed")
					}
					if provider == "local" && r.Header.Get("Authorization") != "" {
						t.Error("local required a key")
					}
					toolName = body["tools"].([]interface{})[0].(map[string]interface{})["function"].(map[string]interface{})["name"].(string)
					for _, m := range body["messages"].([]interface{}) {
						msg := m.(map[string]interface{})
						if msg["role"] == "tool" {
							second = true
							text = msg["content"].(string)
						}
					}
					if !second {
						answer(w, "PRIVATE-THINKING", callsFor(toolName))
						return
					}
				}
				if toolName != "verifier_ping" {
					text = verdict
				}
				switch provider {
				case "anthropic":
					json.NewEncoder(w).Encode(map[string]interface{}{"stop_reason": "end_turn", "content": []interface{}{map[string]string{"type": "text", "text": text}}})
				case "google":
					json.NewEncoder(w).Encode(map[string]interface{}{"candidates": []interface{}{map[string]interface{}{"finishReason": "STOP", "content": map[string]interface{}{"parts": []interface{}{map[string]string{"text": text}}}}}})
				default:
					answer(w, text, nil)
				}
			}))
			defer server.Close()
			config := Config{Provider: provider, Model: "fixture", BaseURL: server.URL + "/v1", APIKey: "provider-secret"}
			if provider == "local" {
				config.APIKey = ""
			}
			for i := 0; i < 2; i++ {
				if err := (&Model{}).Probe(context.Background(), config); err != nil {
					t.Fatal(err)
				}
			}
			if calls != 4 {
				t.Fatalf("probe calls: %d", calls)
			}
			for i := 0; i < 2; i++ {
				tools := &fakeTools{}
				result, err := (&Model{}).Check(context.Background(), Request{Claim: "fresh native claim", Config: config}, tools)
				if err != nil || result.Status != "passed" || len(tools.calls) != 4 {
					t.Fatalf("native Check: %+v %v", result, err)
				}
			}
			if calls != 8 {
				t.Fatal("native verifier inherited previous history")
			}

		})
	}
}
func callsFor(name string) []interface{} { return calls(name, 1) }

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

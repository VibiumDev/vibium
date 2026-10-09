package check

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/vibium/clicker/internal/ai"
)

func TestNativeProviderProbeAndFreshCheck(t *testing.T) {
	for _, provider := range []string{"anthropic", "google", "local", "openai-compatible", "xai"} {
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
					if blocks, ok := body["system"].([]interface{}); !ok || len(blocks) != 1 || blocks[0].(map[string]interface{})["text"] == "" {
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
					if provider == "xai" && r.Header.Get("Authorization") != "Bearer provider-secret" {
						t.Error("wrong xAI transport")
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
			config := ai.Config{Provider: provider, Model: "fixture", BaseURL: server.URL + "/v1", APIKey: "provider-secret"}
			if provider == "local" {
				config.APIKey = ""
			}
			for i := 0; i < 2; i++ {
				if err := (&ai.Model{}).Probe(context.Background(), config); err != nil {
					t.Fatal(err)
				}
			}
			if calls != 4 {
				t.Fatalf("probe calls: %d", calls)
			}
			for i := 0; i < 2; i++ {
				tools := &fakeTools{}
				result, err := Check(context.Background(), Request{Claim: "fresh native claim", Config: config}, tools)
				if err != nil || result.Status != "passed" || len(tools.calls) != 5 {
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

// Auto turns leave Anthropic parallel tool use enabled; the forced verdict
// turn disables it to get exactly one result call (#594).
func TestAnthropicParallelToolUsePolicy(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		var body struct {
			ToolChoice struct {
				Type            string `json:"type"`
				Name            string `json:"name"`
				DisableParallel *bool  `json:"disable_parallel_tool_use"`
			} `json:"tool_choice"`
		}
		json.NewDecoder(r.Body).Decode(&body)
		switch requests {
		case 1:
			if body.ToolChoice.Type != "auto" || body.ToolChoice.DisableParallel != nil {
				t.Errorf("auto turn tool_choice = %+v, want auto without disable_parallel_tool_use", body.ToolChoice)
			}
			fmt.Fprint(w, `{"stop_reason":"end_turn","content":[{"type":"text","text":"BROKEN"}]}`)
		default:
			if body.ToolChoice.Type != "tool" || body.ToolChoice.Name != "return_verdict" || body.ToolChoice.DisableParallel == nil || !*body.ToolChoice.DisableParallel {
				t.Errorf("forced turn tool_choice = %+v, want forced single return_verdict", body.ToolChoice)
			}
			fmt.Fprint(w, `{"stop_reason":"tool_use","content":[{"type":"tool_use","id":"v1","name":"return_verdict","input":`+verdict+`}]}`)
		}
	}))
	defer server.Close()
	req := testRequest(server.URL)
	req.Config.Provider = "anthropic"
	result, err := Check(context.Background(), req, &fakeTools{})
	if err != nil || result.Status != "passed" || requests != 2 {
		t.Fatalf("result=%+v err=%v requests=%d", result, err, requests)
	}
}

func TestAnthropicPromptCaching(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		body, _ := io.ReadAll(r.Body)
		if got := strings.Count(string(body), `"cache_control"`); got != 2 {
			t.Errorf("request %d carries %d cache_control markers, want 2", requests, got)
		}
		var payload struct {
			System []struct {
				CacheControl *struct{ Type string } `json:"cache_control"`
			} `json:"system"`
			Messages []struct {
				Content []map[string]interface{} `json:"content"`
			} `json:"messages"`
		}
		if json.Unmarshal(body, &payload) != nil || len(payload.System) != 1 || payload.System[0].CacheControl == nil {
			t.Errorf("request %d: system block not cache marked", requests)
		}
		last := payload.Messages[len(payload.Messages)-1].Content
		if len(last) == 0 || last[len(last)-1]["cache_control"] == nil {
			t.Errorf("request %d: final conversation block not cache marked", requests)
		}
		if requests == 1 {
			fmt.Fprint(w, `{"stop_reason":"tool_use","content":[{"type":"tool_use","id":"t1","name":"browser_map","input":{}}]}`)
			return
		}
		fmt.Fprint(w, `{"stop_reason":"tool_use","content":[{"type":"tool_use","id":"v1","name":"return_verdict","input":`+verdict+`}]}`)
	}))
	defer server.Close()
	req := testRequest(server.URL)
	req.Config.Provider = "anthropic"
	result, err := Check(context.Background(), req, &fakeTools{})
	if err != nil || result.Status != "passed" || requests != 2 {
		t.Fatalf("result=%+v err=%v requests=%d", result, err, requests)
	}
}

func TestUnparseableProviderResponseNamesContentAndSetting(t *testing.T) {
	for _, tc := range []struct{ name, contentType, body, want string }{
		{name: "html page", contentType: "text/html; charset=utf-8", body: "<!doctype html><title>welcome</title>", want: "AI provider returned text/html, not JSON; check --ai-base-url / VIBIUM_AI_BASE_URL"},
		{name: "unrecognized type", contentType: "application/x-mystery", body: "junk-body", want: "AI provider returned a non-JSON content type"},
		{name: "json wrong shape", contentType: "application/json", body: `{"ok":true}`, want: "AI provider returned JSON that is not a chat-completions message"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", tc.contentType)
				fmt.Fprint(w, tc.body)
			}))
			defer server.Close()
			_, err := Check(context.Background(), testRequest(server.URL), &fakeTools{})
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("want %q in error, got: %v", tc.want, err)
			}
			for _, leak := range []string{"welcome", "junk-body", `"ok"`} {
				if strings.Contains(err.Error(), leak) {
					t.Fatal("leaked provider body")
				}
			}
		})
	}
}

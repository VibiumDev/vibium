package verifier

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestProbeFreshToolRoundTrip(t *testing.T) {
	requests := 0
	lastEvidence := ""
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if r.Header.Get("Authorization") != "Bearer test-secret" {
			t.Error("missing configured authentication")
		}
		var body struct {
			Messages []message `json:"messages"`
			Tools    []struct {
				Function Tool `json:"function"`
			} `json:"tools"`
			Model     string `json:"model"`
			Reasoning string `json:"reasoning_effort"`
		}
		json.NewDecoder(r.Body).Decode(&body)
		if body.Model != "test-model" || body.Reasoning != "none" || len(body.Tools) != 1 || body.Tools[0].Function.Name != "verifier_ping" {
			t.Error("probe did not use configured adapter with its one constrained tool")
		}
		data, _ := json.Marshal(body.Messages)
		if strings.Contains(string(data), "test-secret") || strings.Contains(string(data), "PRIVATE REASONING") {
			t.Error("probe exposed credentials or inherited assistant content")
		}
		if requests%2 == 1 {
			if len(body.Messages) != 2 || body.Messages[0].Role != "system" {
				t.Error("probe inherited conversation")
			}
			answer(w, "PRIVATE REASONING", calls("verifier_ping", 1))
		} else {
			if len(body.Messages) != 4 || body.Messages[2].Content != nil || body.Messages[3].Role != "tool" || body.Messages[3].ToolCallID != "tool0" {
				t.Error("invalid tool result conversation")
			}
			content := body.Messages[3].Content.(string)
			if content == lastEvidence {
				t.Error("reused diagnostic evidence")
			}
			lastEvidence = content
			answer(w, content, nil)
		}
	}))
	defer server.Close()
	config := testRequest(server.URL).Config
	config.ReasoningEffort = "none"
	for i := 0; i < 2; i++ {
		if err := (&OpenAI{}).Probe(context.Background(), config); err != nil {
			t.Fatal(err)
		}
	}
	if requests != 4 {
		t.Fatalf("got %d requests", requests)
	}
}

func TestProbeRejectsFalseReadiness(t *testing.T) {
	for _, scenario := range []string{"no tool", "wrong tool", "extra tools", "invalid args", "extra args", "invented evidence", "malformed verdict", "continued tools", "auth", "unsupported reasoning"} {
		t.Run(scenario, func(t *testing.T) {
			requests := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests++
				if scenario == "auth" {
					w.WriteHeader(401)
					w.Write([]byte(`{"error":{"message":"test-secret"}}`))
					return
				}
				if scenario == "unsupported reasoning" {
					w.WriteHeader(400)
					w.Write([]byte(`{"error":{"code":"unsupported_value","param":"reasoning_effort","message":"test-secret"}}`))
					return
				}
				if requests == 1 {
					switch scenario {
					case "no tool":
						answer(w, verdict, nil)
					case "wrong tool":
						answer(w, nil, calls("browser_click", 1))
					case "extra tools":
						answer(w, nil, calls("verifier_ping", 2))
					case "invalid args", "extra args":
						args := "null"
						if scenario == "extra args" {
							args = `{"secret":"test-secret"}`
						}
						answer(w, nil, []interface{}{map[string]interface{}{"id": "call1", "type": "function", "function": map[string]string{"name": "verifier_ping", "arguments": args}}})
					default:
						answer(w, nil, calls("verifier_ping", 1))
					}
				} else {
					switch scenario {
					case "malformed verdict":
						answer(w, "test-secret", nil)
					case "continued tools":
						answer(w, nil, calls("verifier_ping", 1))
					default:
						answer(w, verdict, nil)
					}
				}
			}))
			defer server.Close()
			err := (&OpenAI{}).Probe(context.Background(), testRequest(server.URL).Config)
			if err == nil || strings.Contains(err.Error(), "test-secret") {
				t.Fatalf("unsafe or absent error: %v", err)
			}
			if requests > 2 {
				t.Fatal("unbounded probe")
			}
		})
	}
}

func TestProbeCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { io.Copy(io.Discard, r.Body); <-r.Context().Done() }))
	defer server.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	if err := (&OpenAI{}).Probe(ctx, testRequest(server.URL).Config); err == nil {
		t.Fatal("accepted timeout")
	}
}

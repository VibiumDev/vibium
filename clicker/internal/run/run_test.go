package run

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/vibium/clicker/internal/verifier"
)

type fixtureTools struct{ calls int }

func (f *fixtureTools) Tools() []verifier.Tool {
	return []verifier.Tool{{Name: "browser_click", Parameters: map[string]interface{}{"type": "object", "properties": map[string]interface{}{}}}}
}
func (f *fixtureTools) Execute(_ context.Context, name string, args map[string]interface{}) (verifier.Observation, error) {
	f.calls++
	return verifier.Observation{Text: "goal observed"}, nil
}
// The result arrives as a return_result tool call and its arguments are the
// result; no free-text JSON parse is involved.
func TestRunResultViaToolCall(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		result, _ := json.Marshal(Result{Status: "completed", Summary: "Fixture result", Evidence: []verifier.Evidence{{Type: "observation", Summary: "Goal observed"}}})
		call := map[string]interface{}{"id": "r1", "type": "function", "function": map[string]interface{}{"name": "return_result", "arguments": string(result)}}
		json.NewEncoder(w).Encode(map[string]interface{}{"choices": []interface{}{map[string]interface{}{"finish_reason": "tool_calls", "message": map[string]interface{}{"role": "assistant", "tool_calls": []interface{}{call}}}}})
	}))
	defer server.Close()
	req := Request{Goal: "the real goal", Config: verifier.Config{Role: "run", Provider: "local", Model: "fixture", BaseURL: server.URL}}
	result, err := Run(context.Background(), req, &fixtureTools{})
	if err != nil || result.Status != "completed" || result.Goal != req.Goal || requests != 1 {
		t.Fatalf("result=%+v err=%v requests=%d", result, err, requests)
	}
}

// The repair turn is wired for Run too: a result wrapped in prose gets one
// corrective turn and the retried JSON parses.
func TestRunRepairsInvalidResult(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		result, _ := json.Marshal(Result{Status: "completed", Summary: "Fixture result", Evidence: []verifier.Evidence{{Type: "observation", Summary: "Goal observed"}}})
		content := string(result)
		if requests == 1 {
			content = "Here is the result: " + content
		}
		json.NewEncoder(w).Encode(map[string]interface{}{"choices": []interface{}{map[string]interface{}{"finish_reason": "stop", "message": map[string]interface{}{"role": "assistant", "content": content}}}})
	}))
	defer server.Close()
	req := Request{Goal: "the real goal", Config: verifier.Config{Role: "run", Provider: "local", Model: "fixture", BaseURL: server.URL}}
	result, err := Run(context.Background(), req, &fixtureTools{})
	if err != nil || result.Status != "completed" || requests != 2 {
		t.Fatalf("result=%+v err=%v requests=%d", result, err, requests)
	}
}
func TestRunContractAndLimits(t *testing.T) {
	for _, scenario := range []string{"completed", "not_completed", "wrong verdict", "no evidence", "limit", "fresh"} {
		t.Run(scenario, func(t *testing.T) {
			requests := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests++
				var body map[string]interface{}
				json.NewDecoder(r.Body).Decode(&body)
				messages := body["messages"].([]interface{})
				system := messages[0].(map[string]interface{})["content"].(string)
				if !strings.Contains(system, "accomplish") || strings.Contains(system, "independent software verifier") {
					t.Error("Run received Check role")
				}
				if scenario == "fresh" && len(messages) != 5 {
					t.Error("Run inherited context")
				}
				message := map[string]interface{}{"role": "assistant"}
				reason := "stop"
				if scenario == "limit" {
					reason = "tool_calls"
					message["tool_calls"] = []interface{}{map[string]interface{}{"id": "call", "type": "function", "function": map[string]string{"name": "browser_click", "arguments": "{}"}}}
				} else {
					status := scenario
					if scenario == "fresh" || scenario == "no evidence" {
						status = "completed"
					}
					if scenario == "wrong verdict" {
						status = "passed"
					}
					evidence := []verifier.Evidence{}
					if scenario != "no evidence" {
						evidence = append(evidence, verifier.Evidence{Type: "observation", Summary: "Goal observed"})
					}
					result, _ := json.Marshal(Result{Status: status, Goal: "invented", Summary: "Fixture result", Evidence: evidence})
					message["content"] = string(result)
				}
				json.NewEncoder(w).Encode(map[string]interface{}{"choices": []interface{}{map[string]interface{}{"finish_reason": reason, "message": message}}})
			}))
			defer server.Close()
			req := Request{Goal: "the real goal", Config: verifier.Config{Role: "run", Provider: "local", Model: "fixture", BaseURL: server.URL}}
			tools := &fixtureTools{}
			result, err := Run(context.Background(), req, tools)
			if scenario == "wrong verdict" || scenario == "no evidence" {
				if err == nil {
					t.Fatal("accepted invalid completion")
				}
				return
			}
			if err != nil || result.Goal != req.Goal {
				t.Fatalf("result=%+v err=%v", result, err)
			}
			if scenario == "limit" && (result.Status != "not_completed" || tools.calls != verifier.MaxActions+3) {
				t.Fatal("action limit not enforced")
			}
			if scenario == "fresh" {
				if _, err := Run(context.Background(), req, tools); err != nil || requests != 2 {
					t.Fatal("second run failed")
				}
			}
		})
	}
}

type siteTools struct{ calls []map[string]interface{} }

func (f *siteTools) Tools() []verifier.Tool {
	return []verifier.Tool{{Name: "browser_navigate", Parameters: map[string]interface{}{"type": "object", "properties": map[string]interface{}{"url": map[string]interface{}{"type": "string"}}}}}
}
func (f *siteTools) Execute(_ context.Context, name string, args map[string]interface{}) (verifier.Observation, error) {
	call := map[string]interface{}{"name": name}
	for k, v := range args {
		call[k] = v
	}
	f.calls = append(f.calls, call)
	if name == "browser_get_url" {
		return verifier.Observation{Text: "https://elsewhere.example/"}, nil
	}
	return verifier.Observation{Text: "ok"}, nil
}

// A declared site under test is opened first, carried as a trusted input
// line, and resolves the model's relative navigation targets.
func TestRunUsesTheSiteUnderTest(t *testing.T) {
	requests := 0
	sawSiteLine := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		var body struct {
			Messages []struct {
				Role    string      `json:"role"`
				Content interface{} `json:"content"`
			} `json:"messages"`
		}
		json.NewDecoder(r.Body).Decode(&body)
		for _, m := range body.Messages {
			if text, ok := m.Content.(string); ok && m.Role == "user" && strings.Contains(text, "Site under test: http://site.example:3000") {
				sawSiteLine = true
			}
		}
		if requests == 1 {
			call := map[string]interface{}{"id": "n1", "type": "function", "function": map[string]interface{}{"name": "browser_navigate", "arguments": `{"url":"/cart"}`}}
			json.NewEncoder(w).Encode(map[string]interface{}{"choices": []interface{}{map[string]interface{}{"finish_reason": "tool_calls", "message": map[string]interface{}{"role": "assistant", "tool_calls": []interface{}{call}}}}})
			return
		}
		result, _ := json.Marshal(Result{Status: "completed", Summary: "Fixture result", Evidence: []verifier.Evidence{{Type: "observation", Summary: "Cart reached"}}})
		call := map[string]interface{}{"id": "r1", "type": "function", "function": map[string]interface{}{"name": "return_result", "arguments": string(result)}}
		json.NewEncoder(w).Encode(map[string]interface{}{"choices": []interface{}{map[string]interface{}{"finish_reason": "tool_calls", "message": map[string]interface{}{"role": "assistant", "tool_calls": []interface{}{call}}}}})
	}))
	defer server.Close()
	tools := &siteTools{}
	req := Request{Goal: "reach the cart", BaseSite: "http://site.example:3000", Config: verifier.Config{Role: "run", Provider: "local", Model: "fixture", BaseURL: server.URL}}
	result, err := Run(context.Background(), req, tools)
	if err != nil || result.Status != "completed" {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	if !sawSiteLine {
		t.Fatal("model input carried no trusted site line")
	}
	var opened, resolved bool
	for i, call := range tools.calls {
		if call["name"] == "browser_navigate" && call["url"] == "http://site.example:3000" && i == 1 {
			opened = true // right after the browser_get_url origin probe
		}
		if call["name"] == "browser_navigate" && call["url"] == "http://site.example:3000/cart" {
			resolved = true
		}
	}
	if !opened || !resolved {
		t.Fatalf("opened=%v resolved=%v calls=%v", opened, resolved, tools.calls)
	}
	if err := (Request{Goal: "g", BaseSite: "not a url", Config: req.Config}).Validate(); err == nil {
		t.Fatal("invalid site URL was accepted")
	}
}

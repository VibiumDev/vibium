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

package verifier

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type fakeTools struct {
	calls    []string
	image    bool
	err      error
	errAfter int // return err only once this many calls have completed
}

func (f *fakeTools) Tools() []Tool {
	return []Tool{{Name: "browser_map", Parameters: map[string]interface{}{"type": "object"}}}
}
func (f *fakeTools) Execute(ctx context.Context, name string, args map[string]interface{}) (Observation, error) {
	f.calls = append(f.calls, name)
	obs := Observation{Text: "observed value: America/Chicago"}
	if f.image && len(f.calls) > 3 {
		obs.Image = "cG5n"
	}
	if f.err != nil && len(f.calls) > f.errAfter {
		return Observation{}, f.err
	}
	return obs, nil
}
func testRequest(base string) Request {
	return Request{Claim: "name persists", Config: Config{Provider: "openai-compatible", Model: "test-model", BaseURL: base, APIKey: "test-secret"}}
}
func answer(w http.ResponseWriter, content interface{}, calls interface{}) {
	reason := "stop"
	if calls != nil {
		reason = "tool_calls"
	}
	json.NewEncoder(w).Encode(map[string]interface{}{"choices": []interface{}{map[string]interface{}{"finish_reason": reason, "message": map[string]interface{}{"role": "assistant", "content": content, "tool_calls": calls, "reasoning_content": "DO NOT PERSIST"}}}})
}
func calls(name string, count int) []interface{} {
	result := []interface{}{}
	for i := 0; i < count; i++ {
		result = append(result, map[string]interface{}{"id": fmt.Sprint("tool", i), "type": "function", "function": map[string]string{"name": name, "arguments": "{}"}})
	}
	return result
}

const verdict = `{"status":"passed","summary":"Name survived reload.","evidence":[{"type":"observation","summary":"America/Chicago after reload."}]}`

func TestFreshContextAndToolLoop(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if r.URL.Path != "/chat/completions" || r.Header.Get("Authorization") != "Bearer test-secret" {
			t.Error("wrong endpoint or auth")
		}
		var body struct {
			Model           string    `json:"model"`
			ReasoningEffort string    `json:"reasoning_effort"`
			Messages        []message `json:"messages"`
		}
		json.NewDecoder(r.Body).Decode(&body)
		data, _ := json.Marshal(body.Messages)
		if strings.Contains(string(data), "test-secret") || strings.Contains(string(data), "DO NOT PERSIST") {
			t.Error("secret or reasoning inherited")
		}
		if body.Model != "test-model" || body.ReasoningEffort != "none" || body.Messages[0].Content != systemInstruction {
			t.Error("model/instructions not applied")
		}
		if requests%2 == 1 {
			if len(body.Messages) != 5 {
				t.Errorf("new verification inherited messages: %d", len(body.Messages))
			}
			answer(w, "DO NOT PERSIST", calls("browser_map", 1))
		} else {
			if body.Messages[5].Content != nil || body.Messages[6].Role != "tool" {
				t.Error("tool conversation malformed")
			}
			answer(w, verdict, nil)
		}
	}))
	defer server.Close()
	adapter := &OpenAI{}
	for i := 0; i < 2; i++ {
		tools := &fakeTools{}
		req := testRequest(server.URL)
		req.Config.ReasoningEffort = "none"
		result, err := adapter.Check(context.Background(), req, tools)
		if err != nil || result.Status != "passed" || result.Claim != "name persists" || len(tools.calls) != 4 {
			t.Fatalf("result=%+v err=%v calls=%v", result, err, tools.calls)
		}
	}
}
// A final message that fails the strict parse gets one corrective turn
// instead of discarding the completed verification; a repeated failure
// keeps the original error.
func TestInvalidVerdictGetsOneRepairTurn(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		var body struct {
			Messages []message `json:"messages"`
		}
		json.NewDecoder(r.Body).Decode(&body)
		if requests == 1 {
			answer(w, "The evidence is clear: "+verdict, nil)
			return
		}
		last := body.Messages[len(body.Messages)-1]
		content, _ := last.Content.(string)
		prior := body.Messages[len(body.Messages)-2]
		priorContent, _ := prior.Content.(string)
		if last.Role != "user" || !strings.Contains(content, "JSON object") || prior.Role != "assistant" || !strings.Contains(priorContent, "The evidence is clear") {
			t.Errorf("corrective turn malformed: prior=%+v last=%+v", prior, last)
		}
		answer(w, verdict, nil)
	}))
	defer server.Close()
	result, err := (&OpenAI{}).Check(context.Background(), testRequest(server.URL), &fakeTools{})
	if err != nil || result.Status != "passed" || requests != 2 {
		t.Fatalf("result=%+v err=%v requests=%d", result, err, requests)
	}
}
func TestPersistentInvalidVerdictStillErrors(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		answer(w, "still not the JSON you asked for", nil)
	}))
	defer server.Close()
	_, err := (&OpenAI{}).Check(context.Background(), testRequest(server.URL), &fakeTools{})
	if err == nil || !strings.Contains(err.Error(), "invalid JSON verdict") || requests != 2 {
		t.Fatalf("err=%v requests=%d", err, requests)
	}
}
func TestProviderErrorsAndVerdicts(t *testing.T) {
	for _, tc := range []struct {
		name, content string
		status        int
		tool          string
		wantError     bool
	}{
		{name: "fail", content: `{"status":"failed","summary":"Reverted","evidence":[{"type":"observation","summary":"Old value after reload"}]}`},
		{name: "inconclusive", content: `{"status":"inconclusive","summary":"No save control"}`},
		{name: "malformed", content: `oops`, wantError: true},
		{name: "no evidence", content: `{"status":"passed","summary":"Trust me"}`, wantError: true},
		{name: "invalid status", content: `{"status":"PASS","summary":"ok"}`, wantError: true},
		{name: "provider error", status: 401, content: "test-secret DO NOT PERSIST", wantError: true},
		{name: "disallowed tool", tool: "browser_evaluate", wantError: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tc.status != 0 {
					w.WriteHeader(tc.status)
					fmt.Fprint(w, tc.content)
					return
				}
				if tc.tool != "" {
					answer(w, nil, calls(tc.tool, 1))
					return
				}
				answer(w, tc.content, nil)
			}))
			defer server.Close()
			tools := &fakeTools{}
			_, err := (&OpenAI{}).Check(context.Background(), testRequest(server.URL), tools)
			if (err != nil) != tc.wantError {
				t.Fatalf("err=%v", err)
			}
			if err != nil && (strings.Contains(err.Error(), "test-secret") || strings.Contains(err.Error(), "DO NOT PERSIST")) {
				t.Fatal("leaked provider body")
			}
			if tc.tool != "" && len(tools.calls) != 3 {
				t.Fatal("executed denied tool")
			}
		})
	}
}
func TestActionErrorReturnedToModel(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		var body struct {
			Messages []message `json:"messages"`
		}
		json.NewDecoder(r.Body).Decode(&body)
		if requests == 1 {
			answer(w, nil, calls("browser_map", 1))
			return
		}
		last := body.Messages[len(body.Messages)-1]
		content, _ := last.Content.(string)
		if last.Role != "tool" || !strings.Contains(content, "Error: failed to click: element not found") {
			t.Errorf("action failure not delivered as tool result: role=%q content=%q", last.Role, content)
		}
		answer(w, verdict, nil)
	}))
	defer server.Close()
	// The 3 initial observations succeed; the model's own call fails.
	tools := &fakeTools{err: &ActionError{Err: fmt.Errorf("failed to click: element not found")}, errAfter: 3}
	result, err := (&OpenAI{}).Check(context.Background(), testRequest(server.URL), tools)
	if err != nil || result.Status != "passed" || requests != 2 {
		t.Fatalf("result=%+v err=%v requests=%d", result, err, requests)
	}
}
func TestNonActionErrorStaysFatal(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		answer(w, nil, calls("browser_map", 1))
	}))
	defer server.Close()
	tools := &fakeTools{err: fmt.Errorf("browser connection lost"), errAfter: 3}
	_, err := (&OpenAI{}).Check(context.Background(), testRequest(server.URL), tools)
	if err == nil || !strings.Contains(err.Error(), "verifier browser action") {
		t.Fatalf("expected fatal browser action error, got: %v", err)
	}
}
func TestActionBudget(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { answer(w, nil, calls("browser_map", 2)) }))
	defer server.Close()
	tools := &fakeTools{}
	result, err := (&OpenAI{}).Check(context.Background(), testRequest(server.URL), tools)
	if err != nil || result.Status != "inconclusive" || len(tools.calls) != MaxActions+3 {
		t.Fatalf("%+v %v %d", result, err, len(tools.calls))
	}
}
func TestTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { time.Sleep(100 * time.Millisecond) }))
	defer server.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	_, err := (&OpenAI{}).Check(ctx, testRequest(server.URL), &fakeTools{})
	if err == nil || !strings.Contains(err.Error(), "timeout") {
		t.Fatalf("%v", err)
	}
}
func TestScreenshotMessages(t *testing.T) {
	n := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n++
		if n == 1 {
			answer(w, nil, calls("browser_map", 1))
			return
		}
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		data, _ := json.Marshal(body)
		if !strings.Contains(string(data), "data:image/png;base64,cG5n") {
			t.Error("missing image")
		}
		answer(w, verdict, nil)
	}))
	defer server.Close()
	if _, err := (&OpenAI{}).Check(context.Background(), testRequest(server.URL), &fakeTools{image: true}); err != nil {
		t.Fatal(err)
	}
}
func TestConfiguration(t *testing.T) {
	t.Setenv("VIBIUM_AI_PROVIDER", "openai")
	t.Setenv("VIBIUM_AI_MODEL", "")
	t.Setenv("OPENAI_API_KEY", "")
	if _, err := ConfigFromEnv(); err == nil {
		t.Fatal("accepted missing model")
	}
	t.Setenv("VIBIUM_AI_MODEL", "configured-model")
	if _, err := ConfigFromEnv(); err == nil {
		t.Fatal("accepted missing key")
	}
	t.Setenv("OPENAI_API_KEY", "secret")
	if _, err := ConfigFromEnv(); err != nil {
		t.Fatal(err)
	}
	for _, base := range []string{"file:///tmp/provider", "http://user:secret@localhost/v1", "http://localhost/v1?key=secret"} {
		c := testRequest(base).Config
		if c.Validate() == nil {
			t.Errorf("accepted %s", base)
		}
	}
}

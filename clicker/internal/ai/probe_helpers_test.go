package ai

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// answer and calls fake one openai-compatible completion response; the same
// helpers exist in internal/check's tests, which exercise the loop through
// Check instead of Probe.
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

func testConfig(base string) Config {
	return Config{Provider: "openai-compatible", Model: "test-model", BaseURL: base, APIKey: "test-secret"}
}

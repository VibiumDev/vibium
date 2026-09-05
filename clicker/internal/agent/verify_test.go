package agent

import (
	"context"
	"testing"
)

func TestVerifierToolBoundary(t *testing.T) {
	v := &verifyTools{h: &Handlers{}}
	for _, tc := range []struct {
		name string
		args map[string]interface{}
	}{
		{"browser_evaluate", map[string]interface{}{"expression": "1"}},
		{"browser_start", map[string]interface{}{}},
		{"browser_screenshot", map[string]interface{}{"filename": "/tmp/forbidden.png"}},
		{"browser_screenshot", map[string]interface{}{"annotate": true}},
		{"browser_map", map[string]interface{}{"page": "other-page"}},
		{"browser_navigate", map[string]interface{}{"url": "file:///tmp/secret"}},
		{"browser_navigate", map[string]interface{}{"url": "javascript:alert(1)"}},
		{"browser_fill", map[string]interface{}{"selector": "input", "text": false}},
		{"browser_click", map[string]interface{}{"selector": "button", "timeout": float64(-1)}},
		{"browser_click", map[string]interface{}{}},
		{"browser_scroll", map[string]interface{}{"amount": float64(100)}},
	} {
		if _, err := v.Execute(context.Background(), tc.name, tc.args); err == nil {
			t.Errorf("accepted %s %v", tc.name, tc.args)
		}
	}
	// Filtering must not mutate the schemas used by existing clients.
	for _, tool := range GetToolSchemas() {
		if tool.Name == "browser_screenshot" {
			if _, ok := tool.InputSchema["properties"].(map[string]interface{})["filename"]; !ok {
				t.Fatal("mutated shared schema")
			}
		}
	}
}

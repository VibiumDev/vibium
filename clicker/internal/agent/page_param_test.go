package agent

import "testing"

// Every page-scoped tool advertises the page argument, and the exclusion
// list holds only session lifecycle, page management, recording control, and
// plain waits (#383). Injection happens centrally in GetToolSchemas, so a
// new tool gets the argument unless it is deliberately listed.
func TestPageParamOnEveryPageScopedTool(t *testing.T) {
	tools := GetToolSchemas()
	names := make(map[string]bool, len(tools))

	for _, tool := range tools {
		names[tool.Name] = true
		props, ok := tool.InputSchema["properties"].(map[string]interface{})
		if !ok {
			t.Fatalf("%s: schema has no properties object", tool.Name)
		}
		_, hasPage := props["page"]
		if noPageParam[tool.Name] && hasPage {
			t.Errorf("%s is listed as not page-scoped but has a page argument", tool.Name)
		}
		if !noPageParam[tool.Name] && !hasPage {
			t.Errorf("%s should accept a page argument", tool.Name)
		}
	}

	// A stale exclusion entry would silently stop guarding anything.
	for name := range noPageParam {
		if !names[name] {
			t.Errorf("noPageParam lists %q, which is not a tool", name)
		}
	}
}

// A page argument without a running browser fails loudly instead of falling
// through to whatever the ambient page would have been.
func TestPageArgumentWithoutBrowserFails(t *testing.T) {
	h := NewHandlers("", "chrome", true, "", nil)
	_, err := h.Call("browser_get_url", map[string]interface{}{"page": "no-such-page"})
	if err == nil {
		t.Fatal("a page argument with no browser must error")
	}
}

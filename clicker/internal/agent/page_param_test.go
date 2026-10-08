package agent

import "testing"

// A page argument without a running browser fails loudly instead of falling
// through to whatever the ambient page would have been.
func TestPageArgumentWithoutBrowserFails(t *testing.T) {
	h := NewHandlers("", "chrome", true, "", nil, nil)
	_, err := h.Call("browser_get_url", map[string]interface{}{"page": "no-such-page"})
	if err == nil {
		t.Fatal("a page argument with no browser must error")
	}
}

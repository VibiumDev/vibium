package api

import (
	"encoding/json"
	"strings"
	"testing"
)

// Route dispatch must be decided in the binary: blocked request events get a
// vibiumMatchedPatterns list computed from the registered globs, so all
// clients share one matching dialect (#446).
func TestRouteRegistryAnnotation(t *testing.T) {
	rr := newRouteRegistry()

	if _, need, err := rr.add("ctx1", "**/api/**"); err != nil || !need {
		t.Fatalf("first add: need=%v err=%v, want intercept needed", need, err)
	}
	rr.setIntercept("ctx1", "icp-1")
	if id, need, err := rr.add("ctx1", "**/*.png"); err != nil || need || id != "icp-1" {
		t.Fatalf("second add: id=%q need=%v err=%v, want existing icp-1", id, need, err)
	}

	blocked := `{"method":"network.beforeRequestSent","params":{"context":"ctx1","isBlocked":true,"request":{"url":"https://x.test/api/data"}}}`
	out := rr.annotateBlockedRequest(blocked)
	var evt struct {
		Params struct {
			Matched []string `json:"vibiumMatchedPatterns"`
		} `json:"params"`
	}
	if err := json.Unmarshal([]byte(out), &evt); err != nil {
		t.Fatalf("annotated event unparseable: %v", err)
	}
	if len(evt.Params.Matched) != 1 || evt.Params.Matched[0] != "**/api/**" {
		t.Fatalf("matched = %v, want [**/api/**]", evt.Params.Matched)
	}

	t.Run("no match still carries an empty list", func(t *testing.T) {
		out := rr.annotateBlockedRequest(`{"method":"network.beforeRequestSent","params":{"context":"ctx1","isBlocked":true,"request":{"url":"https://x.test/other"}}}`)
		if !strings.Contains(out, `"vibiumMatchedPatterns":[]`) {
			t.Fatalf("want explicit empty list, got %s", out)
		}
	})

	t.Run("unblocked and foreign events pass through untouched", func(t *testing.T) {
		for _, msg := range []string{
			`{"method":"network.beforeRequestSent","params":{"context":"ctx1","isBlocked":false,"request":{"url":"https://x.test/api/data"}}}`,
			`{"method":"browsingContext.load","params":{"url":"https://x.test/api/data"}}`,
			`not json`,
		} {
			if got := rr.annotateBlockedRequest(msg); got != msg {
				t.Fatalf("message modified: %q -> %q", msg, got)
			}
		}
	})

	t.Run("other contexts do not see ctx1 patterns", func(t *testing.T) {
		out := rr.annotateBlockedRequest(`{"method":"network.beforeRequestSent","params":{"context":"ctx2","isBlocked":true,"request":{"url":"https://x.test/api/data"}}}`)
		if !strings.Contains(out, `"vibiumMatchedPatterns":[]`) {
			t.Fatalf("ctx2 must match nothing, got %s", out)
		}
	})

	t.Run("refcounted removal tears down at zero", func(t *testing.T) {
		rr.add("ctx1", "**/api/**") // second registration of the same pattern
		if _, empty := rr.remove("ctx1", "**/api/**"); empty {
			t.Fatal("one of two registrations removed, context should stay")
		}
		if _, empty := rr.remove("ctx1", "**/api/**"); empty {
			t.Fatal("png pattern still registered, context should stay")
		}
		id, empty := rr.remove("ctx1", "**/*.png")
		if !empty || id != "icp-1" {
			t.Fatalf("last removal: id=%q empty=%v, want icp-1 teardown", id, empty)
		}
	})

	t.Run("bad glob is rejected at registration", func(t *testing.T) {
		if _, _, err := rr.add("ctx1", "https://x.test/{a"); err == nil {
			t.Fatal("expected a compile error for an unmatched brace group")
		}
	})
}

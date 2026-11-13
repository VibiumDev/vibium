package agent

import (
	"context"
	"errors"
	"testing"

	"github.com/vibium/clicker/internal/bidi"
)

// A selector the browser rejects can never match, so the find polling loop
// must surface the syntax error instead of retrying it into a timeout (#616,
// from discussion #104).
func TestInvalidSelectorError(t *testing.T) {
	cases := map[string]struct {
		err  error
		want bool
	}{
		"css syntax":      {&bidi.ScriptException{Text: "SyntaxError: '@x' is not a valid selector."}, true},
		"xpath syntax":    {&bidi.ScriptException{Text: "SyntaxError: '///' is not a valid XPath expression."}, true},
		"other exception": {&bidi.ScriptException{Text: "TypeError: x is not a function"}, false},
		"context cancel":  {context.Canceled, false},
		"nil":             {nil, false},
	}
	for name, c := range cases {
		got := invalidSelectorError(c.err)
		if (got != nil) != c.want {
			t.Errorf("%s: invalidSelectorError = %v, want non-nil=%v", name, got, c.want)
		}
		if c.want && !isInvalidSelector(got) {
			t.Errorf("%s: isInvalidSelector should recognize its own error type", name)
		}
	}
}

// The error must remain detectable after wrapping, so callers up the stack can
// distinguish it from a genuine miss.
func TestInvalidSelectorErrorWraps(t *testing.T) {
	err := invalidSelectorError(&bidi.ScriptException{Text: "SyntaxError: 'x[' is not a valid selector."})
	if !isInvalidSelector(err) {
		t.Fatal("isInvalidSelector should match the direct error")
	}
	wrapped := errors.Join(errors.New("context"), err)
	if !isInvalidSelector(wrapped) {
		t.Fatal("isInvalidSelector should match through errors.Join")
	}
}

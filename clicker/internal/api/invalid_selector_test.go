package api

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/vibium/clicker/internal/bidi"
)

// A selector the browser rejected can never match, so the polling loops must
// surface the syntax error immediately instead of retrying it into a
// misleading "element not found" timeout (#616, from discussion #104).
func TestCheckInvalidSelectorDetectsSyntaxErrors(t *testing.T) {
	for name, text := range map[string]string{
		"chrome css":   `SyntaxError: Failed to execute 'querySelectorAll' on 'Document': '@Our approach' is not a valid selector.`,
		"firefox css":  `SyntaxError: Document.querySelectorAll: '@Our approach' is not a valid selector`,
		"chrome xpath": `SyntaxError: Failed to execute 'evaluate' on 'Document': The string '///' is not a valid XPath expression.`,
	} {
		resp, _ := json.Marshal(map[string]interface{}{
			"result": map[string]interface{}{
				"type":             "exception",
				"exceptionDetails": map[string]interface{}{"text": text},
			},
		})
		err := checkInvalidSelector(resp)
		var invErr *bidi.InvalidSelectorError
		if !errors.As(err, &invErr) {
			t.Errorf("%s: expected InvalidSelectorError, got %v", name, err)
		}
	}
}

// Anything that is not a selector-syntax exception keeps the old retry
// behavior: transient page states must still be polled through.
func TestCheckInvalidSelectorIgnoresOtherResults(t *testing.T) {
	for name, resp := range map[string]string{
		"success":         `{"result":{"type":"success","result":{"type":"string","value":"{}"}}}`,
		"other exception": `{"result":{"type":"exception","exceptionDetails":{"text":"TypeError: x is not a function"}}}`,
		"not json":        `nope`,
	} {
		if err := checkInvalidSelector(json.RawMessage(resp)); err != nil {
			t.Errorf("%s: expected nil, got %v", name, err)
		}
	}
}

package api

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/vibium/clicker/internal/verifier"
)

// A scroll past the cap is model-correctable, so it must come back as an
// ActionError the loop feeds to the model, not a fatal error that aborts the
// whole check (#619). This is the SDK-surface twin of the agent test.
func TestAPIScrollLimitIsRecoverable(t *testing.T) {
	v := &apiModelTools{}
	_, err := v.Execute(context.Background(), "browser_scroll", map[string]interface{}{"amount": float64(50)})
	var action *verifier.ActionError
	if !errors.As(err, &action) {
		t.Fatalf("got %v, want ActionError", err)
	}
	if !strings.Contains(err.Error(), "10") {
		t.Fatalf("error should name the cap: %v", err)
	}
}

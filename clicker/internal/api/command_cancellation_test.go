package api

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"
)

func TestElementPollingPreservesCancellation(t *testing.T) {
	for _, cause := range []error{context.Canceled, context.DeadlineExceeded} {
		for _, name := range []string{"reference", "element", "actionable", "eval"} {
			t.Run(name+cause.Error(), func(t *testing.T) {
				s := &scriptedSession{}
				s.step("", fmt.Errorf("browser command: %w", cause))
				var err error
				ep := ElementParams{Selector: "#missing", Timeout: time.Second}
				switch name {
				case "reference":
					_, err = ResolveElementRef(s, "page", ep)
				case "element":
					_, err = WaitForElementWithScript(s, "page", "() => null", nil, time.Second)
				case "actionable":
					_, err = WaitForActionable(s, "page", ep, ClickChecks)
				case "eval":
					_, err = EvalSimpleScript(s, "page", "() => null")
				}
				if !errors.Is(err, cause) || s.calls != 1 {
					t.Fatalf("err=%v calls=%d; cancellation must stop polling", err, s.calls)
				}
			})
		}
	}
}

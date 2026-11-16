package bidi

import (
	"strings"
	"time"
)

// staleRealmRetryBudget outlasts the gap between a navigation committing and the new document's realm existing.
const staleRealmRetryBudget = 2 * time.Second

// ScriptNeverRan reports a script command Chrome rejected without running it, because a navigation replaced its realm while it was queued (#811).
func ScriptNeverRan(method string, err error) bool {
	if err == nil || (method != "script.evaluate" && method != "script.callFunction") {
		return false
	}
	return strings.Contains(err.Error(), "Cannot find context with specified id")
}

// RetryStaleRealm resends a script command that never ran until it reaches the new document or the budget runs out.
func RetryStaleRealm[T any](method string, send func() (T, error)) (T, error) {
	out, err := send()
	deadline := time.Now().Add(staleRealmRetryBudget)
	for ScriptNeverRan(method, err) && time.Now().Before(deadline) {
		time.Sleep(25 * time.Millisecond)
		out, err = send()
	}
	return out, err
}

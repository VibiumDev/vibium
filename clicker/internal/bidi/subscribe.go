package bidi

import (
	"fmt"
	"os"
	"strings"
)

// SubscribeEvents subscribes to the given event names in one batch. An engine
// that does not recognise a single name rejects the whole batch (Firefox does
// this for browsingContext.navigationAborted), which would leave every event
// undelivered. On batch failure each event is subscribed individually so an
// unknown name costs only itself. Returns the names the browser rejected.
//
// The per-event pass is serial round trips, but it runs only when the batch
// was rejected; an engine that accepts the batch pays a single round trip.
// If the retries ever matter, cache the rejected set per engine after the
// first connect.
//
// subscribe sends one session.subscribe for the given events and returns an
// error for both transport failures and BiDi error responses.
func SubscribeEvents(subscribe func(events []string) error, events []string) []string {
	if len(events) == 0 {
		return nil
	}
	batchErr := subscribe(events)
	if batchErr == nil {
		return nil
	}

	var rejected []string
	for _, ev := range events {
		if subscribe([]string{ev}) != nil {
			rejected = append(rejected, ev)
		}
	}

	// Directly on stderr: the default log level discards Warn, and a feature
	// silently losing its events must stay visible (#348). When nothing
	// subscribed at all, the cause is whatever failed the batch (a dead
	// connection, a timeout), so report that error rather than blaming
	// every event name.
	if len(rejected) == len(events) {
		fmt.Fprintf(os.Stderr, "vibium: event subscription failed: %v\n", batchErr)
	} else if len(rejected) > 0 {
		fmt.Fprintf(os.Stderr, "vibium: browser rejected event subscription for %s; features relying on these events are degraded\n",
			strings.Join(rejected, ", "))
	}
	return rejected
}

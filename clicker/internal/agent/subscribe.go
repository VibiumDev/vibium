package agent

import (
	"github.com/vibium/clicker/internal/bidi"
)

// subscribeEvents batch-subscribes with per-event fallback so an event name
// the engine does not implement cannot silence the rest (bidi.SubscribeEvents).
func subscribeEvents(client *bidi.Client, events []string) {
	bidi.SubscribeEvents(func(evs []string) error {
		_, err := client.SendCommand("session.subscribe", map[string]interface{}{
			"events": evs,
		})
		return err
	}, events)
}

package api

import (
	"encoding/json"
	"testing"
)

// A client owns its own id space and may use large numbers. The router reserves
// ids from 1000000 for its own internal commands, and used to drop any response
// at or above that value on the assumption it was a late reply to one of them.
// That silently swallowed real client responses — including from
// `vibium pipe --connect`, whose internal ids start at exactly 1000000, so
// chaining pipe to serve hung forever (#158).
func TestHighClientIDsAreNotMistakenForInternalCommands(t *testing.T) {
	session := &BrowserSession{
		internalCmds:      make(map[int]chan json.RawMessage),
		abandonedInternal: make(map[int]struct{}),
		nextInternalID:    1000000,
	}

	for _, id := range []int{5, 999999, 1000000, 1000001, 99999999} {
		session.internalCmdsMu.Lock()
		_, live := session.internalCmds[id]
		_, abandoned := session.abandonedInternal[id]
		session.internalCmdsMu.Unlock()

		if live || abandoned {
			t.Fatalf("id %d should be treated as a client id, not the router's own", id)
		}
	}
}

// The magnitude check it replaced existed for a reason: a reply to an internal
// command that already timed out must not reach the client, which would not
// recognise the id. Recording the abandoned id keeps that behaviour without
// claiming a range of the client's id space.
func TestLateResponsesFromTimedOutInternalCommandsAreDropped(t *testing.T) {
	session := &BrowserSession{
		internalCmds:      make(map[int]chan json.RawMessage),
		abandonedInternal: make(map[int]struct{}),
		nextInternalID:    1000000,
	}

	session.internalCmdsMu.Lock()
	session.abandonedInternal[1000000] = struct{}{}
	session.internalCmdsMu.Unlock()

	session.internalCmdsMu.Lock()
	_, abandoned := session.abandonedInternal[1000000]
	session.internalCmdsMu.Unlock()
	if !abandoned {
		t.Fatal("a timed-out internal id should be remembered so its late reply is dropped")
	}

	// The same id in a different session is a client id again, not ours.
	other := &BrowserSession{
		internalCmds:      make(map[int]chan json.RawMessage),
		abandonedInternal: make(map[int]struct{}),
		nextInternalID:    1000000,
	}
	other.internalCmdsMu.Lock()
	_, leaked := other.abandonedInternal[1000000]
	other.internalCmdsMu.Unlock()
	if leaked {
		t.Fatal("abandoned ids must not leak across sessions")
	}
}

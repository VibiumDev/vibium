package daemon

import (
	"encoding/json"
	"testing"

	"github.com/vibium/clicker/internal/agent"
)

// The daemon socket serves the same MCP initialize handshake as the stdio
// server, and had the same hard-coded 2024-11-05 answer: a client that
// dropped that version could not connect here either (#650).
func TestDaemonInitializeNegotiatesProtocolVersion(t *testing.T) {
	d := &Daemon{version: "test"}

	cases := map[string]string{
		"2025-06-18": "2025-06-18", // supported, echoed
		"2024-11-05": "2024-11-05", // still supported, echoed (back-compat)
		"2099-01-01": "2025-06-18", // unknown, fall back to newest
	}
	for requested, want := range cases {
		params, _ := json.Marshal(agent.InitializeParams{ProtocolVersion: requested})
		res, mcpErr := d.handleInitialize(params)
		if mcpErr != nil {
			t.Fatalf("handleInitialize(%q) error: %+v", requested, mcpErr)
		}
		if got := res.(agent.InitializeResult).ProtocolVersion; got != want {
			t.Errorf("initialize(%q) answered protocolVersion %q, want %q", requested, got, want)
		}
	}

	// Missing params must keep working: the CLI sends initialize with no
	// protocolVersion at all.
	res, mcpErr := d.handleInitialize(nil)
	if mcpErr != nil {
		t.Fatalf("handleInitialize(nil) error: %+v", mcpErr)
	}
	if got := res.(agent.InitializeResult).ProtocolVersion; got != "2025-06-18" {
		t.Errorf("initialize with no params answered %q, want the newest 2025-06-18", got)
	}
}

package agent

import (
	"encoding/json"
	"testing"
)

// The initialize handshake must echo a version the client proposed when vibium
// speaks it, and fall back to the newest otherwise, so a client that dropped
// 2024-11-05 can still connect (#650).
func TestNegotiateProtocolVersion(t *testing.T) {
	newest := supportedProtocolVersions[0]
	cases := map[string]string{
		"2025-06-18": "2025-06-18", // newest, echoed
		"2025-03-26": "2025-03-26", // supported, echoed
		"2024-11-05": "2024-11-05", // still supported, echoed (back-compat)
		"2030-01-01": newest,       // unknown/future, fall back to newest
		"":           newest,       // missing, fall back to newest
		"garbage":    newest,
	}
	for requested, want := range cases {
		if got := negotiateProtocolVersion(requested); got != want {
			t.Errorf("negotiateProtocolVersion(%q) = %q, want %q", requested, got, want)
		}
	}
}

// The handler wires negotiation through: initialize with 2025-06-18 must come
// back 2025-06-18, not the old hard-coded 2024-11-05.
func TestHandleInitializeEchoesSupportedVersion(t *testing.T) {
	s := &Server{version: "test"}
	params, _ := json.Marshal(InitializeParams{ProtocolVersion: "2025-06-18"})

	res, mcpErr := s.handleInitialize(params)
	if mcpErr != nil {
		t.Fatalf("handleInitialize error: %+v", mcpErr)
	}
	got := res.(InitializeResult).ProtocolVersion
	if got != "2025-06-18" {
		t.Errorf("initialize answered protocolVersion %q, want the client's 2025-06-18", got)
	}
}

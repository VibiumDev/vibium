package main

import (
	"strings"
	"testing"

	"github.com/vibium/clicker/internal/agent"
)

// The help must name every tool the server serves. The old hand-written list
// froze at 22 of 85 tools (#393); generating it from the registry keeps the
// two in lockstep, and this locks that in.
func TestMCPHelpListsEveryServedTool(t *testing.T) {
	long := newMCPCmd().Long
	for _, tool := range agent.GetToolSchemas() {
		if !strings.Contains(long, "- "+tool.Name+":") {
			t.Errorf("mcp --help is missing tool %q", tool.Name)
		}
	}
}

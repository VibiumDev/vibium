package agent

import (
	"reflect"
	"testing"

	"github.com/vibium/clicker/internal/paths"
)

// browser_start's channel enum must stay the union of the shared per-engine
// table. It was last hand-written before Chrome channels existed and spent a
// year advertising a Firefox-only world (#525).
func TestBrowserStartChannelEnumMatchesTable(t *testing.T) {
	var got []string
	for _, tool := range GetToolSchemas() {
		if tool.Name != "browser_start" {
			continue
		}
		props := tool.InputSchema["properties"].(map[string]interface{})
		channel := props["channel"].(map[string]interface{})
		got = channel["enum"].([]string)
	}
	if got == nil {
		t.Fatal("browser_start has no channel enum")
	}

	seen := map[string]bool{}
	for _, channels := range paths.EngineChannels {
		for _, c := range channels {
			seen[c] = true
		}
	}
	if len(got) != len(seen) {
		t.Fatalf("channel enum = %v, want exactly the union of paths.EngineChannels %v", got, seen)
	}
	for _, c := range got {
		if !seen[c] {
			t.Errorf("channel enum lists %q, which no engine in paths.EngineChannels has", c)
		}
	}
	if !reflect.DeepEqual(got, channelEnum()) {
		t.Errorf("channel enum = %v, want channelEnum() = %v", got, channelEnum())
	}
}

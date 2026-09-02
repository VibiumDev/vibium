package browser

import (
	"strings"
	"testing"
)

// An unknown channel must fail before any network request, with the
// supported values in the message.
func TestFetchChannelVersionRejectsUnknownChannel(t *testing.T) {
	_, err := fetchChannelVersion("nightly")
	if err == nil {
		t.Fatal("fetchChannelVersion(nightly): want error, got nil")
	}
	if !strings.Contains(err.Error(), "stable, beta, dev, canary") {
		t.Errorf("error %q does not list the supported channels", err)
	}
}

func TestFindVersionInfoMatchesExactly(t *testing.T) {
	versions := []VersionInfo{
		{Version: "140.0.7000.10"},
		{Version: "140.0.7000.100"},
	}
	if got := findVersionInfo(versions, "140.0.7000.10"); got == nil || got.Version != "140.0.7000.10" {
		t.Errorf("findVersionInfo(140.0.7000.10) = %v, want exact match", got)
	}
	if got := findVersionInfo(versions, "140.0.7000.1"); got != nil {
		t.Errorf("findVersionInfo(140.0.7000.1) = %v, want nil for a prefix", got)
	}
}

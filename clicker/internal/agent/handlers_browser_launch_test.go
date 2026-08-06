package agent

import (
	"strings"
	"testing"

	"github.com/vibium/clicker/internal/bidi"
)

func TestBrowserLaunchRejectsEngineMismatch(t *testing.T) {
	h := &Handlers{client: &bidi.Client{}, launchedEngine: "chrome"}
	_, err := h.browserLaunch(map[string]interface{}{"engine": "firefox"})
	if err == nil || !strings.Contains(err.Error(), "chrome is already running; requested firefox") {
		t.Fatalf("browserLaunch() error = %v", err)
	}
}

func TestBrowserLaunchRejectsFirefoxChannelMismatch(t *testing.T) {
	h := &Handlers{
		client:          &bidi.Client{},
		launchedEngine:  "firefox",
		launchedChannel: "release",
	}
	_, err := h.browserLaunch(map[string]interface{}{"engine": "firefox", "channel": "beta"})
	if err == nil || !strings.Contains(err.Error(), "Firefox release is already running; requested beta") {
		t.Fatalf("browserLaunch() error = %v", err)
	}
}

func TestHandlersCaptureFirefoxChannelDefault(t *testing.T) {
	t.Setenv("VIBIUM_FIREFOX_CHANNEL", "release")
	h := NewHandlers("", "firefox", true, "", nil)

	// A per-launch override must not change the default remembered by this
	// daemon/session manager for a later browser session.
	t.Setenv("VIBIUM_FIREFOX_CHANNEL", "beta")
	if h.firefoxChannel != "release" {
		t.Fatalf("firefoxChannel = %q, want captured default release", h.firefoxChannel)
	}
}

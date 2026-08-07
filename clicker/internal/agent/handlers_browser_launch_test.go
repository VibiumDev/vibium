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

func TestBrowserLaunchAttachesToRemoteSessionDespiteModeArgs(t *testing.T) {
	// A session attached to a remote browser has no local engine or headless
	// mode to compare against. Client defaults such as VIBIUM_ENGINE=chrome
	// must reattach as a no-op rather than fail the mismatch checks.
	h := &Handlers{
		client:     &bidi.Client{},
		connectURL: "ws://127.0.0.1:9222/session",
	}
	result, err := h.browserLaunch(map[string]interface{}{"engine": "chrome", "headless": true})
	if err != nil {
		t.Fatalf("browserLaunch() error = %v, want reattach", err)
	}
	if len(result.Content) == 0 || !strings.Contains(result.Content[0].Text, "already running") {
		t.Fatalf("browserLaunch() = %+v, want already-running reattach", result)
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

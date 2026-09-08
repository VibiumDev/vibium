package browser

import (
	"regexp"
	"strings"
	"testing"
)

// With no override, stable installs the baked known-good version instead of
// whatever Google shipped today (#470). Version resolution is offline; only
// the download-URL lookup for that exact version touches the network.
func TestChromeStableChannelResolvesToBakedPin(t *testing.T) {
	t.Setenv("VIBIUM_ENGINE_VERSION", "")
	t.Setenv("VIBIUM_ENGINE_CHANNEL", "")
	if got := chromeInstallVersion(); got != pinnedChromeVersion {
		t.Errorf("chromeInstallVersion() = %q, want the baked %q", got, pinnedChromeVersion)
	}
}

func TestChromeVersionOverrideBeatsBakedPin(t *testing.T) {
	t.Setenv("VIBIUM_ENGINE_VERSION", "140.0.7000.10")
	if got := chromeInstallVersion(); got != "140.0.7000.10" {
		t.Errorf("chromeInstallVersion() = %q, want the override 140.0.7000.10", got)
	}
}

// Non-stable channels keep resolving their current version: a pinned beta
// would blind beta-watch.
func TestChromeBetaChannelSkipsBakedPin(t *testing.T) {
	t.Setenv("VIBIUM_ENGINE_VERSION", "")
	t.Setenv("VIBIUM_ENGINE_CHANNEL", "beta")
	if got := chromeInstallVersion(); got != "" {
		t.Errorf("chromeInstallVersion() = %q, want \"\" so beta fetches its current version", got)
	}
}

// The version-bump workflow rewrites the pins mechanically; this catches a
// bad write (a "null" from jq, a beta like 156.0b3) before it can ship.
func TestBakedPinsAreExactReleaseVersions(t *testing.T) {
	if !regexp.MustCompile(`^\d+\.\d+\.\d+\.\d+$`).MatchString(pinnedChromeVersion) {
		t.Errorf("pinnedChromeVersion = %q, want four dotted numbers", pinnedChromeVersion)
	}
	if !regexp.MustCompile(`^\d+\.\d+(\.\d+)?$`).MatchString(pinnedFirefoxVersion) {
		t.Errorf("pinnedFirefoxVersion = %q, want a release version, not a beta", pinnedFirefoxVersion)
	}
}

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

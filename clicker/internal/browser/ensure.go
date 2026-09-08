package browser

import (
	"os"
	"strings"
)

// SkipBrowserDownload reports whether VIBIUM_SKIP_BROWSER_DOWNLOAD disables
// automatic browser downloads. Accepts "1" and any casing of "true", the
// values the client libraries historically accepted.
func SkipBrowserDownload() bool {
	v := os.Getenv("VIBIUM_SKIP_BROWSER_DOWNLOAD")
	return v == "1" || strings.EqualFold(v, "true")
}

// EngineInstalled reports whether the selected engine is already installed.
// It only stats local paths — no network.
func EngineInstalled(engine string) bool {
	return EngineInstalledForChannel(engine, "")
}

// EngineInstalledForChannel reports whether the engine is installed for a
// specific Firefox channel. An empty channel means the VIBIUM_ENGINE_CHANNEL
// default. Chrome resolves its own channel from the environment and ignores
// the argument.
func EngineInstalledForChannel(engine, channel string) bool {
	if engine == "firefox" {
		return IsFirefoxInstalledForChannel(channel)
	}
	return IsInstalled()
}

// EnsureInstalled installs the selected engine if it is not already present.
// When VIBIUM_SKIP_BROWSER_DOWNLOAD is set it is a no-op, so a later launch
// fails (or finds a user-provided browser) exactly as it did before.
func EnsureInstalled(engine string) error {
	return EnsureInstalledForChannel(engine, "")
}

// EnsureInstalledForChannel installs the engine for a specific Firefox
// channel. The daemon resolves a per-call --channel that need not match the
// environment, so installing the environment's channel there would download
// a Firefox the launch then fails to find.
func EnsureInstalledForChannel(engine, channel string) error {
	if SkipBrowserDownload() || EngineInstalledForChannel(engine, channel) {
		return nil
	}
	if engine == "firefox" {
		_, err := InstallFirefoxForChannel(channel)
		return err
	}
	_, err := Install()
	return err
}

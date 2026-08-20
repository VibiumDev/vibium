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
	if engine == "firefox" {
		return IsFirefoxInstalled()
	}
	return IsInstalled()
}

// EnsureInstalled installs the selected engine if it is not already present.
// When VIBIUM_SKIP_BROWSER_DOWNLOAD is set it is a no-op, so a later launch
// fails (or finds a user-provided browser) exactly as it did before.
func EnsureInstalled(engine string) error {
	if SkipBrowserDownload() || EngineInstalled(engine) {
		return nil
	}
	if engine == "firefox" {
		_, err := InstallFirefox()
		return err
	}
	_, err := Install()
	return err
}

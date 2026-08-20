package browser

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/vibium/clicker/internal/paths"
)

const firefoxVersionsURL = "https://product-details.mozilla.org/1.0/firefox_versions.json"

// InstallFirefox downloads Firefox from Mozilla's release archive into the
// vibium cache and returns the executable path. Skips the download if the
// current version is already installed. The channel comes from
// VIBIUM_FIREFOX_CHANNEL (default "release"; "beta" for pre-release testing).
//
// Windows is unsupported: Mozilla ships only installer executables there, no
// archive build we can unpack into the cache. Install Firefox manually and
// set VIBIUM_FIREFOX_PATH instead.
func InstallFirefox() (string, error) {
	if os.Getenv("VIBIUM_SKIP_BROWSER_DOWNLOAD") == "1" {
		return "", fmt.Errorf("browser download skipped (VIBIUM_SKIP_BROWSER_DOWNLOAD=1)")
	}

	if p := os.Getenv("VIBIUM_FIREFOX_PATH"); p != "" {
		if _, err := os.Stat(p); err != nil {
			return "", fmt.Errorf("VIBIUM_FIREFOX_PATH is set but not usable: %w", err)
		}
		return p, nil
	}

	if runtime.GOOS == "windows" {
		return "", fmt.Errorf("Firefox auto-install is not supported on Windows: install Firefox and set VIBIUM_FIREFOX_PATH to firefox.exe")
	}

	channel := paths.FirefoxChannel()
	version, err := resolveFirefoxVersion(channel)
	if err != nil {
		return "", fmt.Errorf("failed to fetch Firefox version info: %w", err)
	}

	ffDir, err := paths.GetFirefoxDir()
	if err != nil {
		return "", fmt.Errorf("failed to get cache dir: %w", err)
	}
	versionDir := filepath.Join(ffDir, version)
	exePath := paths.FirefoxPathInVersion(versionDir)

	if _, err := os.Stat(exePath); err == nil {
		fmt.Printf("Firefox v%s already installed.\n", version)
		return exePath, nil
	}

	fmt.Printf("Installing Firefox v%s (%s channel)...\n", version, channel)

	downloadURL := firefoxDownloadURL(version)
	fmt.Printf("Downloading Firefox from %s...\n", downloadURL)

	if err := os.MkdirAll(versionDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create version dir: %w", err)
	}

	if runtime.GOOS == "darwin" {
		err = installFirefoxDMG(downloadURL, versionDir)
	} else {
		err = installFirefoxTarXZ(downloadURL, versionDir)
	}
	if err != nil {
		os.RemoveAll(versionDir)
		return "", fmt.Errorf("failed to install Firefox: %w", err)
	}

	if _, err := os.Stat(exePath); err != nil {
		os.RemoveAll(versionDir)
		return "", fmt.Errorf("Firefox installed but executable not found: %w", err)
	}

	// Remove quarantine attribute on macOS to avoid Gatekeeper prompts
	if runtime.GOOS == "darwin" {
		exec.Command("xattr", "-dr", "com.apple.quarantine", filepath.Join(versionDir, "Firefox.app")).Run()
	}

	return exePath, nil
}

// IsFirefoxInstalled checks if a usable Firefox executable is available.
func IsFirefoxInstalled() bool {
	p, err := paths.GetFirefoxExecutable()
	if err != nil {
		return false
	}
	_, err = os.Stat(p)
	return err == nil
}

// resolveFirefoxVersion returns the Firefox version to install:
// VIBIUM_FIREFOX_VERSION when set, otherwise the channel's current version
// from Mozilla's product-details JSON. The pin exists because "latest"
// changes out from under CI and fleets (a beta pin silently jumped 154 to
// 155 the day 154 reached release); it also skips the network round-trip.
func resolveFirefoxVersion(channel string) (string, error) {
	if v := os.Getenv("VIBIUM_FIREFOX_VERSION"); v != "" {
		return v, nil
	}
	return fetchLatestFirefoxVersion(channel)
}

// fetchLatestFirefoxVersion resolves the current version for a channel from
// Mozilla's product-details JSON.
func fetchLatestFirefoxVersion(channel string) (string, error) {
	resp, err := http.Get(firefoxVersionsURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	var data struct {
		Release string `json:"LATEST_FIREFOX_VERSION"`
		Beta    string `json:"LATEST_FIREFOX_DEVEL_VERSION"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return "", err
	}

	switch channel {
	case "release":
		if data.Release == "" {
			return "", fmt.Errorf("no release version in product details")
		}
		return data.Release, nil
	case "beta":
		if data.Beta == "" {
			return "", fmt.Errorf("no beta version in product details")
		}
		return data.Beta, nil
	default:
		return "", fmt.Errorf("unknown Firefox channel %q (supported: release, beta)", channel)
	}
}

// firefoxDownloadURL returns the Mozilla archive URL for this platform.
// Betas live under the same releases/ tree as stable versions.
func firefoxDownloadURL(version string) string {
	return firefoxDownloadURLFor(runtime.GOOS, runtime.GOARCH, version)
}

func firefoxDownloadURLFor(goos, goarch, version string) string {
	base := "https://ftp.mozilla.org/pub/firefox/releases/" + version
	switch goos {
	case "darwin":
		return base + "/mac/en-US/" + url.PathEscape("Firefox "+version+".dmg")
	default: // linux
		arch := "linux-x86_64"
		if goarch == "arm64" {
			arch = "linux-aarch64"
		}
		return base + "/" + arch + "/en-US/firefox-" + version + ".tar.xz"
	}
}

// downloadToTemp downloads a URL to a temp file and returns its path.
// The caller removes the file.
func downloadToTemp(downloadURL, pattern string) (string, error) {
	resp, err := http.Get(downloadURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	tmpFile, err := os.CreateTemp("", pattern)
	if err != nil {
		return "", err
	}
	tmpPath := tmpFile.Name()

	pw := &progressWriter{dst: tmpFile, total: resp.ContentLength, out: os.Stdout}
	if _, err := io.Copy(pw, resp.Body); err != nil {
		tmpFile.Close()
		os.Remove(tmpPath)
		return "", err
	}
	if err := tmpFile.Close(); err != nil {
		os.Remove(tmpPath)
		return "", err
	}
	return tmpPath, nil
}

// installFirefoxDMG mounts the DMG and copies Firefox.app into versionDir.
// cp -R (not Go file walking) preserves the app bundle's symlinks and
// permissions, which code signing validation depends on.
func installFirefoxDMG(downloadURL, versionDir string) error {
	dmgPath, err := downloadToTemp(downloadURL, "firefox-*.dmg")
	if err != nil {
		return err
	}
	defer os.Remove(dmgPath)

	mountPoint, err := os.MkdirTemp("", "firefox-dmg-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(mountPoint)

	if out, err := exec.Command("hdiutil", "attach", dmgPath, "-nobrowse", "-readonly", "-mountpoint", mountPoint).CombinedOutput(); err != nil {
		return fmt.Errorf("hdiutil attach failed: %w: %s", err, out)
	}
	defer exec.Command("hdiutil", "detach", mountPoint, "-force").Run()

	if out, err := exec.Command("cp", "-R", filepath.Join(mountPoint, "Firefox.app"), versionDir).CombinedOutput(); err != nil {
		return fmt.Errorf("copying Firefox.app failed: %w: %s", err, out)
	}
	return nil
}

// installFirefoxTarXZ extracts the Linux tar.xz (a firefox/ directory) into
// versionDir. The system tar handles xz; Go's stdlib does not.
func installFirefoxTarXZ(downloadURL, versionDir string) error {
	tarPath, err := downloadToTemp(downloadURL, "firefox-*.tar.xz")
	if err != nil {
		return err
	}
	defer os.Remove(tarPath)

	if out, err := exec.Command("tar", "-xJf", tarPath, "-C", versionDir).CombinedOutput(); err != nil {
		return fmt.Errorf("extracting Firefox archive failed: %w: %s", err, out)
	}
	return nil
}

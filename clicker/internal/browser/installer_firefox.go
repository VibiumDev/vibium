package browser

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

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

	pattern := "firefox-*.tar.xz"
	if runtime.GOOS == "darwin" {
		pattern = "firefox-*.dmg"
	}
	archivePath, err := downloadToTemp(downloadURL, pattern)
	if err != nil {
		return "", fmt.Errorf("failed to download Firefox: %w", err)
	}
	defer os.Remove(archivePath)

	if err := verifyFirefoxArchive(archivePath, version); err != nil {
		return "", fmt.Errorf("Firefox download failed verification: %w", err)
	}

	if err := os.MkdirAll(versionDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create version dir: %w", err)
	}

	if runtime.GOOS == "darwin" {
		err = installFirefoxDMG(archivePath, versionDir)
	} else {
		err = installFirefoxTarXZ(archivePath, versionDir)
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
	segments := strings.Split(firefoxArchiveRelPath(goos, goarch, version), "/")
	for i, s := range segments {
		segments[i] = url.PathEscape(s)
	}
	return "https://ftp.mozilla.org/pub/firefox/releases/" + version + "/" + strings.Join(segments, "/")
}

// firefoxArchiveRelPath returns the archive's path relative to the release
// directory, unescaped, exactly as it appears in Mozilla's SHA256SUMS.
func firefoxArchiveRelPath(goos, goarch, version string) string {
	switch goos {
	case "darwin":
		return "mac/en-US/Firefox " + version + ".dmg"
	default: // linux
		arch := "linux-x86_64"
		if goarch == "arm64" {
			arch = "linux-aarch64"
		}
		return arch + "/en-US/firefox-" + version + ".tar.xz"
	}
}

// verifyFirefoxArchive checks the downloaded archive against Mozilla's
// published SHA256SUMS for the release, before anything unpacks it.
func verifyFirefoxArchive(archivePath, version string) error {
	sums, err := fetchFirefoxChecksums(version)
	if err != nil {
		return fmt.Errorf("fetching SHA256SUMS: %w", err)
	}
	relPath := firefoxArchiveRelPath(runtime.GOOS, runtime.GOARCH, version)
	if err := verifyArchiveAgainstSums(archivePath, sums, relPath); err != nil {
		return err
	}
	fmt.Printf("Verified Firefox archive against Mozilla's SHA256SUMS.\n")
	return nil
}

// fetchFirefoxChecksums downloads the SHA256SUMS body for a release.
func fetchFirefoxChecksums(version string) (string, error) {
	resp, err := http.Get("https://ftp.mozilla.org/pub/firefox/releases/" + version + "/SHA256SUMS")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

// verifyArchiveAgainstSums compares the file's SHA256 with the entry for
// relPath in a SHA256SUMS body (lines of "<hex>  <path>").
func verifyArchiveAgainstSums(archivePath, sums, relPath string) error {
	want := findFirefoxChecksum(sums, relPath)
	if want == "" {
		return fmt.Errorf("no SHA256SUMS entry for %q", relPath)
	}
	got, err := sha256File(archivePath)
	if err != nil {
		return err
	}
	if !strings.EqualFold(got, want) {
		return fmt.Errorf("SHA256 mismatch for %q: got %s, want %s", relPath, got, want)
	}
	return nil
}

// findFirefoxChecksum returns the hash listed for relPath, or "" if absent.
func findFirefoxChecksum(sums, relPath string) string {
	for _, line := range strings.Split(sums, "\n") {
		hash, path, ok := strings.Cut(line, "  ")
		if ok && path == relPath {
			return hash
		}
	}
	return ""
}

func sha256File(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
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

	if _, err := io.Copy(tmpFile, resp.Body); err != nil {
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
func installFirefoxDMG(dmgPath, versionDir string) error {
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
func installFirefoxTarXZ(tarPath, versionDir string) error {
	if out, err := exec.Command("tar", "-xJf", tarPath, "-C", versionDir).CombinedOutput(); err != nil {
		return fmt.Errorf("extracting Firefox archive failed: %w: %s", err, out)
	}
	return nil
}

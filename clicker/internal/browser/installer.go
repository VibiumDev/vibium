package browser

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/vibium/clicker/internal/paths"
)

const (
	knownGoodVersionsURL = "https://googlechromelabs.github.io/chrome-for-testing/known-good-versions-with-downloads.json"
	lastKnownGoodURL     = "https://googlechromelabs.github.io/chrome-for-testing/last-known-good-versions-with-downloads.json"
)

// VersionInfo represents the Chrome for Testing version information.
type VersionInfo struct {
	Version   string                `json:"version"`
	Downloads map[string][]Download `json:"downloads"`
}

// Download represents a download URL for a specific platform.
type Download struct {
	Platform string `json:"platform"`
	URL      string `json:"url"`
}

// LastKnownGoodResponse represents the API response for last known good versions.
type LastKnownGoodResponse struct {
	Channels map[string]VersionInfo `json:"channels"`
}

// InstallResult contains the paths to installed binaries.
type InstallResult struct {
	ChromePath       string
	ChromedriverPath string
	Version          string
}

// Install downloads and installs Chrome for Testing and chromedriver.
// Returns paths to the installed binaries. Skips download if already installed.
func Install() (*InstallResult, error) {
	// Check for skip environment variable
	if os.Getenv("VIBIUM_SKIP_BROWSER_DOWNLOAD") == "1" {
		return nil, fmt.Errorf("browser download skipped (VIBIUM_SKIP_BROWSER_DOWNLOAD=1)")
	}

	// Check if already installed
	if IsInstalled() {
		chromePath, _ := paths.GetChromeExecutable()
		chromedriverPath, _ := paths.GetChromedriverPath()
		// Extract version from path (e.g., .../chrome-for-testing/143.0.7499.192/...)
		version := extractVersionFromPath(chromePath)
		fmt.Printf("Chrome for Testing v%s already installed.\n", version)
		return &InstallResult{
			ChromePath:       chromePath,
			ChromedriverPath: chromedriverPath,
			Version:          version,
		}, nil
	}

	platform := paths.GetPlatformString()

	// Fetch latest stable version info
	versionInfo, err := fetchLatestStableVersion()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch version info: %w", err)
	}

	fmt.Printf("Installing Chrome for Testing v%s...\n", versionInfo.Version)

	// Create version directory
	cftDir, err := paths.GetChromeForTestingDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get cache dir: %w", err)
	}

	versionDir := filepath.Join(cftDir, versionInfo.Version)
	if err := os.MkdirAll(versionDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create version dir: %w", err)
	}

	// Download and extract Chrome
	chromeURL := findDownloadURL(versionInfo.Downloads["chrome"], platform)
	if chromeURL == "" {
		return nil, fmt.Errorf("no Chrome download available for platform %s", platform)
	}

	fmt.Printf("Downloading Chrome from %s...\n", chromeURL)
	if err := downloadAndExtract(chromeURL, versionDir); err != nil {
		return nil, fmt.Errorf("failed to install Chrome: %w", err)
	}

	// Download and extract chromedriver
	chromedriverURL := findDownloadURL(versionInfo.Downloads["chromedriver"], platform)
	if chromedriverURL == "" {
		return nil, fmt.Errorf("no chromedriver download available for platform %s", platform)
	}

	fmt.Printf("Downloading chromedriver from %s...\n", chromedriverURL)
	if err := downloadAndExtract(chromedriverURL, versionDir); err != nil {
		return nil, fmt.Errorf("failed to install chromedriver: %w", err)
	}

	// Get paths to installed binaries
	chromePath, err := paths.GetChromeExecutable()
	if err != nil {
		return nil, fmt.Errorf("Chrome installed but not found: %w", err)
	}

	chromedriverPath, err := paths.GetChromedriverPath()
	if err != nil {
		return nil, fmt.Errorf("chromedriver installed but not found: %w", err)
	}

	// Make executable on Unix
	if runtime.GOOS != "windows" {
		os.Chmod(chromePath, 0755)
		os.Chmod(chromedriverPath, 0755)
	}

	// Remove quarantine attribute on macOS to avoid Gatekeeper prompts
	if runtime.GOOS == "darwin" {
		exec.Command("xattr", "-d", "com.apple.quarantine", chromePath).Run()
		exec.Command("xattr", "-d", "com.apple.quarantine", chromedriverPath).Run()
	}

	return &InstallResult{
		ChromePath:       chromePath,
		ChromedriverPath: chromedriverPath,
		Version:          versionInfo.Version,
	}, nil
}

// fetchLatestStableVersion fetches the latest stable Chrome for Testing version.
func fetchLatestStableVersion() (*VersionInfo, error) {
	resp, err := http.Get(lastKnownGoodURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	var data LastKnownGoodResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}

	stable, ok := data.Channels["Stable"]
	if !ok {
		return nil, fmt.Errorf("no Stable channel found")
	}

	return &stable, nil
}

// findDownloadURL finds the download URL for the given platform.
func findDownloadURL(downloads []Download, platform string) string {
	for _, d := range downloads {
		if d.Platform == platform {
			return d.URL
		}
	}
	return ""
}

// downloadAndExtract downloads a zip file and extracts it to the destination.
func downloadAndExtract(url, destDir string) error {
	// Download to temp file
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	tmpFile, err := os.CreateTemp("", "chrome-*.zip")
	if err != nil {
		return err
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	pw := &progressWriter{dst: tmpFile, total: resp.ContentLength, out: os.Stdout}
	if _, err := io.Copy(pw, resp.Body); err != nil {
		tmpFile.Close()
		return err
	}
	tmpFile.Close()

	// Extract zip
	return extractZip(tmpPath, destDir)
}

// progressWriter wraps a download destination and prints coarse progress
// lines while bytes flow: one line per 10% step when the total is known, one
// line per 25 MB when the server sends no Content-Length. Progress goes to
// out (os.Stdout for the install command; pipe mode redirects that to stderr
// so the protocol stream stays clean and clients see download liveness).
type progressWriter struct {
	dst     io.Writer
	out     io.Writer
	total   int64 // expected bytes; <= 0 when unknown
	written int64
	lastPct int   // last 10%-step printed
	lastMB  int64 // last 25MB-step printed
}

const progressStepBytes = 25 << 20

func (p *progressWriter) Write(b []byte) (int, error) {
	n, err := p.dst.Write(b)
	p.written += int64(n)
	if p.total > 0 {
		if pct := int(p.written * 100 / p.total); pct/10 > p.lastPct/10 {
			p.lastPct = pct
			fmt.Fprintf(p.out, "  %d%% of %.1f MB\n", pct-pct%10, float64(p.total)/(1<<20))
		}
	} else if step := p.written / progressStepBytes; step > p.lastMB {
		p.lastMB = step
		fmt.Fprintf(p.out, "  %d MB downloaded\n", step*25)
	}
	return n, err
}

// extractZip extracts a zip file to the destination directory.
func extractZip(zipPath, destDir string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		// Strip the top-level directory (e.g. "chrome-mac-arm64/..." → "...")
		name := f.Name
		if i := strings.IndexByte(name, '/'); i >= 0 {
			name = name[i+1:]
		}
		if name == "" {
			continue
		}

		fpath := filepath.Join(destDir, name)

		// Security check: prevent zip slip
		if !strings.HasPrefix(fpath, filepath.Clean(destDir)+string(os.PathSeparator)) {
			return fmt.Errorf("invalid file path: %s", fpath)
		}

		if f.FileInfo().IsDir() {
			os.MkdirAll(fpath, os.ModePerm)
			continue
		}

		if err := os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
			return err
		}

		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}

		rc, err := f.Open()
		if err != nil {
			outFile.Close()
			return err
		}

		_, err = io.Copy(outFile, rc)
		outFile.Close()
		rc.Close()

		if err != nil {
			return err
		}
	}

	return nil
}

// IsInstalled checks if Chrome for Testing and chromedriver are both installed.
func IsInstalled() bool {
	chromePath, err := paths.GetChromeExecutable()
	if err != nil {
		return false
	}
	if _, err = os.Stat(chromePath); err != nil {
		return false
	}

	chromedriverPath, err := paths.GetChromedriverPath()
	if err != nil {
		return false
	}
	_, err = os.Stat(chromedriverPath)
	return err == nil
}

// extractVersionFromPath extracts the version number from a Chrome path.
// e.g., ".../chrome-for-testing/143.0.7499.192/..." -> "143.0.7499.192"
func extractVersionFromPath(path string) string {
	parts := strings.Split(path, string(os.PathSeparator))
	for i, part := range parts {
		if part == "chrome-for-testing" && i+1 < len(parts) {
			return parts[i+1]
		}
	}
	return "unknown"
}

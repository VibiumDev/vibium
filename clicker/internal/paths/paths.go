package paths

import (
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
)

// GetCacheDir returns the platform-specific cache directory for Vibium.
// Override with VIBIUM_CACHE_DIR environment variable.
// Linux: ~/.cache/vibium/
// macOS: ~/Library/Caches/vibium/
// Windows: %LOCALAPPDATA%\vibium\
func GetCacheDir() (string, error) {
	if cacheDir := os.Getenv("VIBIUM_CACHE_DIR"); cacheDir != "" {
		return cacheDir, nil
	}

	var baseDir string

	switch runtime.GOOS {
	case "linux":
		if xdgCache := os.Getenv("XDG_CACHE_HOME"); xdgCache != "" {
			baseDir = xdgCache
		} else {
			home, err := os.UserHomeDir()
			if err != nil {
				return "", err
			}
			baseDir = filepath.Join(home, ".cache")
		}
	case "darwin":
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		baseDir = filepath.Join(home, "Library", "Caches")
	case "windows":
		if localAppData := os.Getenv("LOCALAPPDATA"); localAppData != "" {
			baseDir = localAppData
		} else {
			home, err := os.UserHomeDir()
			if err != nil {
				return "", err
			}
			baseDir = filepath.Join(home, "AppData", "Local")
		}
	default:
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		baseDir = filepath.Join(home, ".cache")
	}

	return filepath.Join(baseDir, "vibium"), nil
}

// GetChromeForTestingDir returns the directory where Chrome for Testing is cached.
func GetChromeForTestingDir() (string, error) {
	cacheDir, err := GetCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(cacheDir, "chrome-for-testing"), nil
}

// resolveVersionDir returns the newest cached version directory containing BOTH
// Chrome and chromedriver.
//
// They used to be resolved independently, each taking the first directory that
// held its own binary, so a cache with Chrome under 146.x and chromedriver under
// 147.x produced a mismatched pair that IsInstalled() certified as fine — the
// failure surfaced later as chromedriver's "only supports Chrome version N"
// (#265). Newest-first also replaces os.ReadDir's lexical order, under which
// "99.0" sorts above "100.0".
func resolveVersionDir() (string, error) {
	cftDir, err := GetChromeForTestingDir()
	if err != nil {
		return "", err
	}

	entries, err := os.ReadDir(cftDir)
	if err != nil {
		return "", err
	}

	var complete []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		dir := filepath.Join(cftDir, entry.Name())
		if _, err := os.Stat(getChromePathInVersion(dir)); err != nil {
			continue
		}
		if _, err := os.Stat(getChromedriverPathInVersion(dir)); err != nil {
			continue
		}
		complete = append(complete, entry.Name())
	}
	if len(complete) == 0 {
		return "", os.ErrNotExist
	}

	sort.Slice(complete, func(i, j int) bool {
		return compareVersions(complete[i], complete[j]) > 0
	})
	return filepath.Join(cftDir, complete[0]), nil
}

// compareVersions orders dotted numeric versions, falling back to string order
// for anything that does not parse.
func compareVersions(a, b string) int {
	as, bs := strings.Split(a, "."), strings.Split(b, ".")
	for i := 0; i < len(as) || i < len(bs); i++ {
		var ai, bi int
		if i < len(as) {
			ai, _ = strconv.Atoi(as[i])
		}
		if i < len(bs) {
			bi, _ = strconv.Atoi(bs[i])
		}
		if ai != bi {
			if ai > bi {
				return 1
			}
			return -1
		}
	}
	return strings.Compare(a, b)
}

// GetChromeExecutable returns the path to Chrome for Testing executable.
// Only checks Vibium cache - does not fall back to system Chrome.
func GetChromeExecutable() (string, error) {
	dir, err := resolveVersionDir()
	if err != nil {
		return "", err
	}
	return getChromePathInVersion(dir), nil
}

// GetChromedriverPath returns the path to the cached chromedriver. It resolves
// from the same version directory as GetChromeExecutable, so the two always
// match.
func GetChromedriverPath() (string, error) {
	dir, err := resolveVersionDir()
	if err != nil {
		return "", err
	}
	return getChromedriverPathInVersion(dir), nil
}

// getChromePathInVersion returns the Chrome executable path within a version directory.
func getChromePathInVersion(versionDir string) string {
	switch runtime.GOOS {
	case "darwin":
		return filepath.Join(versionDir, "Google Chrome for Testing.app", "Contents", "MacOS", "Google Chrome for Testing")
	case "windows":
		return filepath.Join(versionDir, "chrome.exe")
	default: // linux
		return filepath.Join(versionDir, "chrome")
	}
}

// getChromedriverPathInVersion returns the chromedriver path within a version directory.
func getChromedriverPathInVersion(versionDir string) string {
	switch runtime.GOOS {
	case "windows":
		return filepath.Join(versionDir, "chromedriver.exe")
	default:
		return filepath.Join(versionDir, "chromedriver")
	}
}

// getPlatformString returns the platform string used by Chrome for Testing.
func getPlatformString() string {
	switch runtime.GOOS {
	case "darwin":
		if runtime.GOARCH == "arm64" {
			return "mac-arm64"
		}
		return "mac-x64"
	case "windows":
		return "win64"
	default: // linux
		return "linux64"
	}
}

// GetPlatformString is exported for use by the installer.
func GetPlatformString() string {
	return getPlatformString()
}

// GetDaemonDir returns the directory for daemon files (socket, PID).
// This reuses the cache directory since daemon files are ephemeral.
func GetDaemonDir() (string, error) {
	return GetCacheDir()
}

// GetSocketPath returns the platform-specific socket path for the daemon.
// macOS/Linux: ~/Library/Caches/vibium/vibium.sock or ~/.cache/vibium/vibium.sock
// Windows: \\.\pipe\vibium (named pipe)
func GetSocketPath() (string, error) {
	if runtime.GOOS == "windows" {
		return `\\.\pipe\vibium`, nil
	}
	dir, err := GetDaemonDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "vibium.sock"), nil
}

// GetPIDPath returns the path to the daemon PID file.
func GetPIDPath() (string, error) {
	dir, err := GetDaemonDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "vibium.pid"), nil
}

// GetScreenshotDir returns the platform-specific default directory for screenshots.
// macOS: ~/Pictures/Vibium/
// Linux: ~/Pictures/Vibium/
// Windows: %USERPROFILE%\Pictures\Vibium\
func GetScreenshotDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	switch runtime.GOOS {
	case "windows":
		// Windows uses Pictures folder in user profile
		return filepath.Join(home, "Pictures", "Vibium"), nil
	default:
		// macOS and Linux use ~/Pictures/Vibium
		return filepath.Join(home, "Pictures", "Vibium"), nil
	}
}

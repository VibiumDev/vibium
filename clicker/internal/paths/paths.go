package paths

import (
	"os"
	"path/filepath"
	"runtime"
)

// GetCacheDir returns the platform-specific cache directory for Vibium.
// Linux: ~/.cache/vibium/
// macOS: ~/Library/Caches/vibium/
// Windows: %LOCALAPPDATA%\vibium\
func GetCacheDir() (string, error) {
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

// GetChromeExecutable returns the path to Chrome for Testing executable.
// Only checks Vibium cache - does not fall back to system Chrome.
func GetChromeExecutable() (string, error) {
	cftDir, err := GetChromeForTestingDir()
	if err != nil {
		return "", err
	}

	// Look for version directories
	entries, err := os.ReadDir(cftDir)
	if err != nil {
		return "", err
	}

	// Use the first (or latest) version found
	for _, entry := range entries {
		if entry.IsDir() {
			chromePath := getChromePathInVersion(filepath.Join(cftDir, entry.Name()))
			if _, err := os.Stat(chromePath); err == nil {
				return chromePath, nil
			}
		}
	}

	return "", os.ErrNotExist
}

// GetChromedriverPath returns the path to the cached chromedriver.
func GetChromedriverPath() (string, error) {
	cftDir, err := GetChromeForTestingDir()
	if err != nil {
		return "", err
	}

	entries, err := os.ReadDir(cftDir)
	if err != nil {
		return "", err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			driverPath := getChromedriverPathInVersion(filepath.Join(cftDir, entry.Name()))
			if _, err := os.Stat(driverPath); err == nil {
				return driverPath, nil
			}
		}
	}

	return "", os.ErrNotExist
}

// getChromePathInVersion returns the Chrome executable path within a version directory.
func getChromePathInVersion(versionDir string) string {
	platform := getPlatformString()

	switch runtime.GOOS {
	case "darwin":
		return filepath.Join(versionDir, "chrome-"+platform, "Google Chrome for Testing.app", "Contents", "MacOS", "Google Chrome for Testing")
	case "windows":
		return filepath.Join(versionDir, "chrome-"+platform, "chrome.exe")
	default: // linux
		return filepath.Join(versionDir, "chrome-"+platform, "chrome")
	}
}

// getChromedriverPathInVersion returns the chromedriver path within a version directory.
func getChromedriverPathInVersion(versionDir string) string {
	platform := getPlatformString()

	switch runtime.GOOS {
	case "windows":
		return filepath.Join(versionDir, "chromedriver-"+platform, "chromedriver.exe")
	default:
		return filepath.Join(versionDir, "chromedriver-"+platform, "chromedriver")
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

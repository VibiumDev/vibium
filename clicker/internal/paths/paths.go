package paths

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
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

// FirefoxChannel returns the Firefox release channel to install and run.
// Defaults to "release"; override with VIBIUM_FIREFOX_CHANNEL (e.g. "beta")
// to run features that have not reached stable yet.
func FirefoxChannel() string {
	if c := os.Getenv("VIBIUM_FIREFOX_CHANNEL"); c != "" {
		return c
	}
	return "release"
}

// GetFirefoxDir returns the directory where Firefox is cached. Channels get
// separate directories so an installed beta never shadows stable (version
// dirs like 154.0b6 would otherwise sort above the 153.x release).
func GetFirefoxDir() (string, error) {
	return GetFirefoxDirForChannel(FirefoxChannel())
}

// GetFirefoxDirForChannel returns the cache directory for a specific channel.
// Passing the channel explicitly lets long-lived daemons launch different
// named sessions without mutating process-wide environment state.
func GetFirefoxDirForChannel(channel string) (string, error) {
	cacheDir, err := GetCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(cacheDir, "firefox", channel), nil
}

// GetFirefoxExecutable returns the path to the cached Firefox executable.
// VIBIUM_FIREFOX_PATH overrides the cache lookup (e.g. a system Firefox).
func GetFirefoxExecutable() (string, error) {
	return GetFirefoxExecutableForChannel(FirefoxChannel())
}

// GetFirefoxExecutableForChannel returns the Firefox executable for channel.
func GetFirefoxExecutableForChannel(channel string) (string, error) {
	if p := os.Getenv("VIBIUM_FIREFOX_PATH"); p != "" {
		return p, nil
	}
	dir, err := resolveFirefoxVersionDir(channel)
	if err != nil {
		return "", err
	}
	return FirefoxPathInVersion(dir), nil
}

// resolveFirefoxVersionDir returns the newest cached Firefox version
// directory containing the executable, mirroring resolveVersionDir.
func resolveFirefoxVersionDir(channel string) (string, error) {
	ffDir, err := GetFirefoxDirForChannel(channel)
	if err != nil {
		return "", err
	}

	entries, err := os.ReadDir(ffDir)
	if err != nil {
		return "", err
	}

	var complete []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		if _, err := os.Stat(FirefoxPathInVersion(filepath.Join(ffDir, entry.Name()))); err != nil {
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
	return filepath.Join(ffDir, complete[0]), nil
}

// FirefoxPathInVersion returns the Firefox executable path within a version
// directory. Layout follows what Mozilla's archives unpack to: Firefox.app
// from the macOS DMG, a firefox/ directory from the Linux tar.xz.
func FirefoxPathInVersion(versionDir string) string {
	switch runtime.GOOS {
	case "darwin":
		return filepath.Join(versionDir, "Firefox.app", "Contents", "MacOS", "firefox")
	case "windows":
		return filepath.Join(versionDir, "firefox", "firefox.exe")
	default: // linux
		return filepath.Join(versionDir, "firefox", "firefox")
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

// SessionName returns the daemon session name from the VIBIUM_SESSION
// environment variable. Empty means the default (shared) session.
// Named sessions get their own daemon socket, PID file, and browser,
// so concurrent CLI users on one host stay isolated.
func SessionName() string {
	return os.Getenv("VIBIUM_SESSION")
}

var sessionNameRe = regexp.MustCompile(`^[A-Za-z0-9_-]{1,64}$`)

// ValidateSessionName checks that a session name is safe to embed in
// socket filenames and Windows named-pipe names.
func ValidateSessionName(name string) error {
	if name == "" {
		return nil
	}
	if !sessionNameRe.MatchString(name) {
		return fmt.Errorf("invalid session name %q: use only letters, digits, '-' and '_' (max 64 chars)", name)
	}
	return nil
}

// sessionSuffix returns "-<name>" for a named session, "" for the default.
func sessionSuffix() (string, error) {
	name := SessionName()
	if err := ValidateSessionName(name); err != nil {
		return "", err
	}
	if name == "" {
		return "", nil
	}
	return "-" + name, nil
}

// maxSocketPathLen returns the longest usable Unix socket path for this
// platform. sockaddr_un.sun_path holds 104 bytes on macOS/BSD and 108 on
// Linux, including the trailing NUL.
func maxSocketPathLen() int {
	if runtime.GOOS == "darwin" {
		return 103
	}
	return 107
}

// GetSocketPath returns the platform-specific socket path for the daemon.
// macOS/Linux: ~/Library/Caches/vibium/vibium[-<session>].sock or ~/.cache/vibium/vibium[-<session>].sock
// Windows: \\.\pipe\vibium[-<session>] (named pipe)
func GetSocketPath() (string, error) {
	suffix, err := sessionSuffix()
	if err != nil {
		return "", err
	}
	if runtime.GOOS == "windows" {
		return `\\.\pipe\vibium` + suffix, nil
	}
	dir, err := GetDaemonDir()
	if err != nil {
		return "", err
	}
	path := filepath.Join(dir, "vibium"+suffix+".sock")
	// Binding a socket beyond sun_path's capacity fails deep inside daemon
	// startup; reject it here with an actionable message instead.
	if len(path) > maxSocketPathLen() {
		return "", fmt.Errorf("socket path %q is %d bytes, over the %d-byte OS limit: use a shorter session name or point VIBIUM_CACHE_DIR at a shorter path", path, len(path), maxSocketPathLen())
	}
	return path, nil
}

// GetPIDPath returns the path to the daemon PID file.
func GetPIDPath() (string, error) {
	suffix, err := sessionSuffix()
	if err != nil {
		return "", err
	}
	dir, err := GetDaemonDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "vibium"+suffix+".pid"), nil
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

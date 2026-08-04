package browser

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/vibium/clicker/internal/bidi"
	"github.com/vibium/clicker/internal/log"
	"github.com/vibium/clicker/internal/paths"
	"github.com/vibium/clicker/internal/process"
)

// firefoxConnectTimeout bounds the wait for Firefox's BiDi WebSocket to come
// up after process start (cold launch on a busy machine can take a while).
const firefoxConnectTimeout = 30 * time.Second

// firefoxPrefs are written to user.js in the temp profile. Firefox's remote
// agent applies its own "recommended" automation prefs on top (focus,
// throttling, etc. — remote.prefs.recommended defaults to true); these cover
// the first-run, update, and UI noise the recommended set leaves alone.
var firefoxPrefs = map[string]interface{}{
	"browser.shell.checkDefaultBrowser":           false,
	"browser.aboutwelcome.enabled":                false,
	"browser.startup.homepage":                    "about:blank",
	"startup.homepage_welcome_url":                "about:blank",
	"startup.homepage_welcome_url.additional":     "",
	"browser.newtabpage.enabled":                  false,
	"app.update.disabledForTesting":               true,
	"app.update.auto":                             false,
	"datareporting.policy.dataSubmissionEnabled":  false,
	"datareporting.healthreport.uploadEnabled":    false,
	"toolkit.telemetry.reportingpolicy.firstRun":  false,
	"dom.disable_open_during_load":                false,
	"dom.webnotifications.enabled":                false,
	"signon.rememberSignons":                      false,
	"extensions.formautofill.creditCards.enabled": false,
	"browser.contentblocking.introCount":          99,
	"browser.tabs.warnOnClose":                    false,
	"browser.warnOnQuit":                          false,
}

// launchFirefox starts Firefox directly, no driver process: Firefox
// implements WebDriver BiDi natively and serves it on
// --remote-debugging-port at ws://.../session.
func launchFirefox(opts LaunchOptions) (*LaunchResult, error) {
	log.Debug("launching firefox", "headless", opts.Headless)

	firefoxPath, err := paths.GetFirefoxExecutable()
	if err != nil {
		return nil, fmt.Errorf("Firefox not found: run `vibium install --browser firefox` or set VIBIUM_FIREFOX_PATH")
	}
	log.Debug("found firefox", "path", firefoxPath)

	port := opts.Port
	if port == 0 {
		port, err = findAvailablePort()
		if err != nil {
			return nil, fmt.Errorf("failed to find available port: %w", err)
		}
	}
	log.Debug("using port", "port", port)

	profileDir, err := os.MkdirTemp("", "vibium-firefox-profile-")
	if err != nil {
		return nil, fmt.Errorf("failed to create profile dir: %w", err)
	}
	if err := writeFirefoxPrefs(profileDir); err != nil {
		os.RemoveAll(profileDir)
		return nil, fmt.Errorf("failed to write profile prefs: %w", err)
	}

	args := []string{
		fmt.Sprintf("--remote-debugging-port=%d", port),
		"--profile", profileDir,
		"--no-remote",
	}
	if opts.Headless {
		args = append(args, "--headless")
	}

	cmd := exec.Command(firefoxPath, args...)
	setProcGroup(cmd)
	if opts.Verbose {
		fmt.Println("       ------- firefox -------")
		pw := newPrefixWriter(os.Stdout, "       ")
		cmd.Stdout = pw
		cmd.Stderr = pw
	}
	if err := cmd.Start(); err != nil {
		os.RemoveAll(profileDir)
		return nil, fmt.Errorf("failed to start firefox: %w", err)
	}

	process.Track(cmd)

	conn, err := connectFirefoxBidi(fmt.Sprintf("ws://127.0.0.1:%d/session", port), firefoxConnectTimeout)
	if err != nil {
		killProcessTree(cmd.Process.Pid)
		cmd.Wait()
		process.Untrack(cmd)
		os.RemoveAll(profileDir)
		return nil, fmt.Errorf("firefox failed to start: %w", err)
	}

	if opts.Verbose {
		fmt.Println("       -----------------------")
	}

	result, err := bidi.SessionNewOnConn(conn, firefoxCapabilities())
	if err != nil {
		conn.Close()
		killProcessTree(cmd.Process.Pid)
		cmd.Wait()
		process.Untrack(cmd)
		os.RemoveAll(profileDir)
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	log.Info("firefox launched via BiDi session.new", "sessionId", result.SessionID)
	return &LaunchResult{
		BidiConn:    conn,
		SessionID:   result.SessionID,
		BrowserCmd:  cmd,
		Port:        port,
		UserDataDir: profileDir,
	}, nil
}

// firefoxCapabilities returns the capabilities map for session.new. Launch
// configuration (headless, profile) travels as process arguments, not
// capabilities, because we start the binary ourselves.
func firefoxCapabilities() map[string]interface{} {
	return map[string]interface{}{
		"alwaysMatch": map[string]interface{}{
			"browserName":  "firefox",
			"webSocketUrl": true,
			"unhandledPromptBehavior": map[string]interface{}{
				"default": "ignore",
			},
		},
	}
}

// connectFirefoxBidi polls the BiDi endpoint until Firefox is accepting
// WebSocket connections.
func connectFirefoxBidi(wsURL string, timeout time.Duration) (*bidi.Connection, error) {
	deadline := time.Now().Add(timeout)
	var lastErr error
	for time.Now().Before(deadline) {
		conn, err := bidi.Connect(wsURL)
		if err == nil {
			return conn, nil
		}
		lastErr = err
		time.Sleep(100 * time.Millisecond)
	}
	return nil, fmt.Errorf("timeout waiting for firefox BiDi endpoint: %w", lastErr)
}

// writeFirefoxPrefs writes user.js into the profile directory.
func writeFirefoxPrefs(profileDir string) error {
	var b []byte
	for k, v := range firefoxPrefs {
		b = append(b, []byte(fmt.Sprintf("user_pref(%q, %s);\n", k, prefValue(v)))...)
	}
	return os.WriteFile(filepath.Join(profileDir, "user.js"), b, 0644)
}

func prefValue(v interface{}) string {
	if s, ok := v.(string); ok {
		return fmt.Sprintf("%q", s)
	}
	return fmt.Sprintf("%v", v)
}

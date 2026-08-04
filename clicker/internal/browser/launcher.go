package browser

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/vibium/clicker/internal/bidi"
	"github.com/vibium/clicker/internal/log"
	"github.com/vibium/clicker/internal/paths"
	"github.com/vibium/clicker/internal/process"
)

// sessionCreateTimeout bounds the HTTP POST /session fallback (Chrome cold
// launch ~16s) so a wedged chromedriver can't hang it indefinitely.
const sessionCreateTimeout = 30 * time.Second

// prefixWriter wraps an io.Writer and prepends a prefix to each line.
type prefixWriter struct {
	w      io.Writer
	prefix string
	atBOL  bool // at beginning of line
}

func newPrefixWriter(w io.Writer, prefix string) *prefixWriter {
	return &prefixWriter{w: w, prefix: prefix, atBOL: true}
}

func (pw *prefixWriter) Write(p []byte) (n int, err error) {
	for _, b := range p {
		if pw.atBOL {
			if _, err := pw.w.Write([]byte(pw.prefix)); err != nil {
				return n, err
			}
			pw.atBOL = false
		}
		if _, err := pw.w.Write([]byte{b}); err != nil {
			return n, err
		}
		n++
		if b == '\n' {
			pw.atBOL = true
		}
	}
	return n, nil
}

// LaunchOptions contains options for launching the browser.
type LaunchOptions struct {
	Headless bool
	Port     int  // Chromedriver port, 0 = auto-select
	Verbose  bool // Show chromedriver output
}

// LaunchResult contains the result of launching the browser via chromedriver.
type LaunchResult struct {
	BidiConn        *bidi.Connection // non-nil when session created via BiDi (no HTTP)
	WebSocketURL    string           // set when session created via HTTP fallback
	SessionID       string
	ChromedriverCmd *exec.Cmd
	Port            int
	UserDataDir     string // Chrome temp profile dir — cleaned up on Close()
}

// sessionRequest is the payload for creating a new session.
type sessionRequest struct {
	Capabilities capabilities `json:"capabilities"`
}

type capabilities struct {
	AlwaysMatch alwaysMatch `json:"alwaysMatch"`
}

type alwaysMatch struct {
	BrowserName  string   `json:"browserName"`
	WebSocketURL bool     `json:"webSocketUrl"`
	Args         []string `json:"goog:chromeOptions,omitempty"`
}

type chromeOptions struct {
	Args   []string `json:"args,omitempty"`
	Binary string   `json:"binary,omitempty"`
}

// sessionResponse is the response from creating a new session.
type sessionResponse struct {
	Value sessionValue `json:"value"`
}

type sessionValue struct {
	SessionID    string                 `json:"sessionId"`
	Capabilities map[string]interface{} `json:"capabilities"`
}

// Launch starts chromedriver and creates a BiDi session.
func Launch(opts LaunchOptions) (*LaunchResult, error) {
	log.Debug("launching browser", "headless", opts.Headless)

	chromedriverPath, err := paths.GetChromedriverPath()
	if err != nil {
		return nil, fmt.Errorf("chromedriver not found")
	}
	log.Debug("found chromedriver", "path", chromedriverPath)

	chromePath, err := paths.GetChromeExecutable()
	if err != nil {
		return nil, fmt.Errorf("Chrome not found")
	}
	log.Debug("found chrome", "path", chromePath)

	// Find available port
	port := opts.Port
	if port == 0 {
		port, err = findAvailablePort()
		if err != nil {
			return nil, fmt.Errorf("failed to find available port: %w", err)
		}
	}
	log.Debug("using port", "port", port)

	// Start chromedriver as a process group leader so we can kill all children
	cdArgs := []string{fmt.Sprintf("--port=%d", port)}
	if dir := os.Getenv("VIBIUM_CHROMEDRIVER_LOG_DIR"); dir != "" {
		cdArgs = append(cdArgs, "--verbose", fmt.Sprintf("--log-path=%s/chromedriver-%d.log", dir, port))
	}
	cmd := exec.Command(chromedriverPath, cdArgs...)
	setProcGroup(cmd)
	// Set here, not around `make`: macOS strips DYLD_* across SIP-protected execs.
	if shim := vmFastLaunchShim(); shim != "" {
		cmd.Env = append(os.Environ(), "DYLD_INSERT_LIBRARIES="+shim)
	}
	if opts.Verbose {
		fmt.Println("       ------- chromedriver -------")
		pw := newPrefixWriter(os.Stdout, "       ")
		cmd.Stdout = pw
		cmd.Stderr = pw
	}
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start chromedriver: %w", err)
	}

	// Track for cleanup
	process.Track(cmd)

	// Wait for chromedriver to be ready
	baseURL := fmt.Sprintf("http://localhost:%d", port)
	if err := waitForChromedriver(baseURL, 10*time.Second); err != nil {
		cmd.Process.Kill()
		return nil, fmt.Errorf("chromedriver failed to start: %w", err)
	}

	if opts.Verbose {
		fmt.Println("       ----------------------------")
	}

	// Try BiDi session.new first (direct WebSocket, no HTTP round-trip)
	wsURL := fmt.Sprintf("ws://localhost:%d/session", port)
	conn, connErr := bidi.Connect(wsURL)
	if connErr == nil {
		caps := buildCapabilities(chromePath, opts.Headless)
		// Handshake without NewClient: a Client's reader would own this
		// connection forever, and the consumer of LaunchResult.BidiConn
		// (agent or api router) must be able to take over reads itself.
		result, sessionErr := bidi.SessionNewOnConn(conn, caps)
		if sessionErr == nil {
			userDataDir, _ := result.Capabilities["userDataDir"].(string)
			log.Info("browser launched via BiDi session.new", "sessionId", result.SessionID)
			return &LaunchResult{
				BidiConn:        conn,
				SessionID:       result.SessionID,
				ChromedriverCmd: cmd,
				Port:            port,
				UserDataDir:     userDataDir,
			}, nil
		}
		log.Debug("BiDi session.new failed, falling back to HTTP", "error", sessionErr)
		conn.Close()
	} else {
		log.Debug("BiDi WebSocket connect failed, falling back to HTTP", "error", connErr)
	}

	// Fallback: HTTP POST /session (original path)
	sessionID, httpWsURL, userDataDir, err := createSession(baseURL, chromePath, opts.Headless, opts.Verbose)
	if err != nil {
		cmd.Process.Kill()
		return nil, fmt.Errorf("failed to create session: %w", err)
	}
	log.Info("browser launched via HTTP", "sessionId", sessionID, "wsUrl", httpWsURL)

	return &LaunchResult{
		WebSocketURL:    httpWsURL,
		SessionID:       sessionID,
		ChromedriverCmd: cmd,
		Port:            port,
		UserDataDir:     userDataDir,
	}, nil
}

// findAvailablePort finds an available TCP port.
func findAvailablePort() (int, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer listener.Close()
	return listener.Addr().(*net.TCPAddr).Port, nil
}

// waitForChromedriver waits for chromedriver to be ready.
func waitForChromedriver(baseURL string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := http.Get(baseURL + "/status")
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
	return fmt.Errorf("timeout waiting for chromedriver")
}

// chromeArgs returns the standard Chrome launch arguments.
func chromeArgs(headless bool) []string {
	args := []string{
		"--no-first-run",
		"--no-default-browser-check",
		"--disable-infobars",
		"--disable-blink-features=AutomationControlled",
		"--disable-crash-reporter",
		"--disable-background-networking",
		"--disable-background-timer-throttling",
		"--disable-backgrounding-occluded-windows",
		"--disable-breakpad",
		"--disable-component-extensions-with-background-pages",
		"--disable-component-update",
		"--disable-default-apps",
		"--disable-dev-shm-usage",
		"--disable-extensions",
		"--disable-notifications",
		"--disable-features=TranslateUI,PasswordLeakDetection",
		"--disable-hang-monitor",
		"--disable-ipc-flooding-protection",
		"--disable-popup-blocking",
		"--disable-prompt-on-repost",
		"--disable-renderer-backgrounding",
		"--disable-sync",
		"--enable-features=NetworkService,NetworkServiceInProcess",
		"--force-color-profile=srgb",
		"--metrics-recording-only",
		"--password-store=basic",
		"--use-mock-keychain",
	}
	args = append(args, platformChromeArgs()...)
	if headless {
		args = append(args, "--headless=new")
	}
	// Append any custom flags from VIBIUM_CHROME_ARGS (space-separated) so users
	// can pass --no-sandbox etc. in root/CI/container environments where Chrome
	// refuses to start otherwise. Empty tokens (from extra whitespace) are skipped.
	args = append(args, customChromeArgs()...)
	return args
}

// vmFastLaunchShim returns a Metal-interposing dylib path for macOS VM guests
// whose dead virtual GPU adds ~15s to every Chrome launch. Opt-in, darwin-only.
// See docs/how-to-guides/slow-chrome-launch-in-macos-vm.md
func vmFastLaunchShim() string {
	if runtime.GOOS != "darwin" {
		return ""
	}
	return os.Getenv("VIBIUM_VM_FAST_LAUNCH")
}

// customChromeArgs reads extra Chrome flags from the VIBIUM_CHROME_ARGS
// environment variable, splitting on whitespace and dropping empty tokens.
// Returns nil when the variable is unset or contains only whitespace.
func customChromeArgs() []string {
	return strings.Fields(os.Getenv("VIBIUM_CHROME_ARGS"))
}

// buildCapabilities returns the capabilities map for BiDi session.new.
func buildCapabilities(chromePath string, headless bool) map[string]interface{} {
	return map[string]interface{}{
		"alwaysMatch": map[string]interface{}{
			"browserName":  "chrome",
			"webSocketUrl": true,
			"unhandledPromptBehavior": map[string]interface{}{
				"default": "ignore",
			},
			"goog:chromeOptions": map[string]interface{}{
				"binary":          chromePath,
				"args":            chromeArgs(headless),
				"excludeSwitches": []string{"enable-automation", "enable-logging"},
				"prefs": map[string]interface{}{
					"credentials_enable_service":                           false,
					"profile.password_manager_enabled":                     false,
					"profile.password_manager_leak_detection":              false,
					"profile.default_content_setting_values.notifications": 2,
				},
			},
		},
	}
}

// createSession creates a new WebDriver session with BiDi enabled via HTTP.
func createSession(baseURL, chromePath string, headless, verbose bool) (string, string, string, error) {
	reqBody := map[string]interface{}{
		"capabilities": buildCapabilities(chromePath, headless),
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", "", "", err
	}

	if verbose {
		fmt.Println("       ------- POST /session -------")
		fmt.Printf("       --> %s\n", string(jsonBody))
	}

	// Bounded so a wedged chromedriver can't hang the HTTP fallback forever.
	// session.new + Chrome cold-launch fit comfortably within this.
	httpClient := &http.Client{Timeout: sessionCreateTimeout}
	resp, err := httpClient.Post(baseURL+"/session", "application/json", bytes.NewReader(jsonBody))
	if err != nil {
		return "", "", "", err
	}
	defer resp.Body.Close()

	// Read response body for logging and parsing
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", "", fmt.Errorf("failed to read session response: %w", err)
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		// The body carries the real reason — a version mismatch, a bad binary
		// path, a sandbox failure. Returning only the status code (#107) left
		// users with "HTTP 500" and nothing to act on.
		return "", "", "", fmt.Errorf("failed to create session: HTTP %d: %s",
			resp.StatusCode, driverErrorMessage(respBody))
	}

	if verbose {
		fmt.Printf("       <-- %s\n", string(respBody))
		fmt.Println("       ------------------------------")
	}

	var sessResp sessionResponse
	if err := json.Unmarshal(respBody, &sessResp); err != nil {
		return "", "", "", fmt.Errorf("failed to decode session response: %w", err)
	}

	wsURL, ok := sessResp.Value.Capabilities["webSocketUrl"].(string)
	if !ok || wsURL == "" {
		return "", "", "", fmt.Errorf("webSocketUrl not found in session capabilities")
	}

	// Extract the Chrome user-data-dir so we can clean it up on Close()
	userDataDir, _ := sessResp.Value.Capabilities["userDataDir"].(string)

	return sessResp.Value.SessionID, wsURL, userDataDir, nil
}

// Close terminates a chromedriver session and process.
func (r *LaunchResult) Close() error {
	log.Debug("closing browser", "sessionId", r.SessionID)

	// Kill chromedriver and all its descendants
	if r.ChromedriverCmd != nil && r.ChromedriverCmd.Process != nil {
		pid := r.ChromedriverCmd.Process.Pid

		// Kill the entire process tree (chromedriver + Chrome + all helpers)
		killProcessTree(pid)

		// Wait for chromedriver to exit
		r.ChromedriverCmd.Wait()

		process.Untrack(r.ChromedriverCmd)
	}

	// Clean up the Chrome temp profile directory for THIS session only.
	// Do not glob-clean other Chrome temp dirs here: under parallel test runs
	// (e.g. node --test --test-concurrency=4) every test file spawns its own
	// vibium pipe + Chrome, and a glob would happily RemoveAll() sibling
	// processes' still-active user-data-dirs — Chrome then loses the ability
	// to create new files inside its profile and the BiDi connection hangs.
	// Run an orphan sweep explicitly (e.g. `make double-tap`) instead.
	if r.UserDataDir != "" {
		log.Debug("removing Chrome user data dir", "path", r.UserDataDir)
		os.RemoveAll(r.UserDataDir)
	}

	return nil
}

// killProcessTree kills a process and all its descendants using process group kill.
// Chromedriver is started as a process group leader (Setpgid: true), so killing
// the group atomically terminates all children — no racy pgrep walk needed.
func killProcessTree(pid int) {
	killProcessGroup(pid)
	killByPid(pid) // fallback: kill root directly if pgid lookup failed
	waitForProcessDead(pid, 2*time.Second)
}

// isVibiumProcess reports whether comm (from `ps -o comm=`: bare name on
// Linux, full path on macOS) names a vibium binary.
func isVibiumProcess(comm string) bool {
	name := filepath.Base(comm)
	return name == "vibium" || name == "clicker"
}

// KillOrphanedChromeProcesses kills processes running from vibium's Chrome
// for Testing cache dir that no live vibium process owns. Ownership is
// checked via the process tree rather than ppid==1 because systemd
// re-parents orphans to the session subreaper, not PID 1. Only call during
// shutdown, after CloseAll().
func KillOrphanedChromeProcesses() {
	cftDir, err := paths.GetChromeForTestingDir()
	if err != nil {
		return
	}

	// ^ anchor: only match processes running from the dir
	output, err := exec.Command("pgrep", "-f", "^"+cftDir).Output()
	if err != nil {
		// pgrep exits 1 when nothing matched, which is the normal case
		if exitErr, ok := err.(*exec.ExitError); !ok || exitErr.ExitCode() != 1 {
			log.Debug("orphan sweep: pgrep failed", "error", err)
		}
		return
	}

	// Collect matched PIDs first to tell tree roots from descendants.
	// (PID reuse between pgrep and ps is possible; the window is tiny.)
	matched := make(map[int]bool)
	for _, line := range bytes.Split(bytes.TrimSpace(output), []byte("\n")) {
		var pid int
		if _, err := fmt.Sscanf(string(line), "%d", &pid); err == nil {
			matched[pid] = true
		}
	}

	myPID := os.Getpid()
	for pid := range matched {
		ppidOut, err := exec.Command("ps", "-o", "ppid=", "-p", fmt.Sprintf("%d", pid)).Output()
		if err != nil {
			continue
		}
		var ppid int
		if _, err := fmt.Sscanf(string(bytes.TrimSpace(ppidOut)), "%d", &ppid); err != nil {
			continue
		}
		// Descendant of another matched process; handled via its root.
		if matched[ppid] {
			continue
		}
		if ppid == myPID {
			killProcessGroup(pid)
			killByPid(pid)
			continue
		}
		parentComm, err := exec.Command("ps", "-o", "comm=", "-p", fmt.Sprintf("%d", ppid)).Output()
		if err != nil || !isVibiumProcess(strings.TrimSpace(string(parentComm))) {
			killProcessGroup(pid)
			killByPid(pid)
		}
	}
}

// CleanupOrphanedChromeTempDirs removes Chrome temp directories left behind
// by previous crashed runs. Safe to call at process start or from explicit
// cleanup tooling (e.g. `make double-tap`). NEVER call this on a normal
// browser shutdown — sibling vibium processes' Chrome user-data-dirs match
// the same glob, and deleting them out from under a live Chrome breaks its
// BiDi connection.
//
// The minAge filter only deletes directories whose mtime is older than the
// given duration — anything fresher could belong to a currently-running
// sibling process. Pass time.Minute or longer for parallel-safe cleanup.
func CleanupOrphanedChromeTempDirs(minAge time.Duration) {
	tmpDir := os.TempDir()
	patterns := []string{
		filepath.Join(tmpDir, "com.google.chrome.for.testing.*"),
		filepath.Join(tmpDir, "org.chromium.Chromium.scoped_dir.*"),
	}
	cutoff := time.Now().Add(-minAge)
	var count int
	for _, pattern := range patterns {
		matches, _ := filepath.Glob(pattern)
		for _, m := range matches {
			info, err := os.Stat(m)
			if err != nil || info.ModTime().After(cutoff) {
				continue
			}
			if os.RemoveAll(m) == nil {
				count++
			}
		}
	}
	if count > 0 {
		log.Debug("cleaned up orphaned Chrome temp dirs", "count", count, "minAge", minAge)
	}
}

// driverErrorMessage pulls the human-readable part out of a WebDriver error
// response, falling back to the raw body when it is not the expected shape.
func driverErrorMessage(body []byte) string {
	var e struct {
		Value struct {
			Error   string `json:"error"`
			Message string `json:"message"`
		} `json:"value"`
	}
	if err := json.Unmarshal(body, &e); err == nil {
		switch {
		case e.Value.Message != "":
			return strings.TrimSpace(e.Value.Message)
		case e.Value.Error != "":
			return e.Value.Error
		}
	}
	trimmed := strings.TrimSpace(string(body))
	if trimmed == "" {
		return "(empty response body)"
	}
	if len(trimmed) > 500 {
		return trimmed[:500] + "…"
	}
	return trimmed
}

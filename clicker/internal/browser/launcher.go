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
	"time"

	"github.com/vibium/clicker/internal/log"
	"github.com/vibium/clicker/internal/paths"
	"github.com/vibium/clicker/internal/process"
)

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
	WebSocketURL    string
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
		return nil, fmt.Errorf("chromedriver not found: %w (run 'clicker install' first)", err)
	}
	log.Debug("found chromedriver", "path", chromedriverPath)

	chromePath, err := paths.GetChromeExecutable()
	if err != nil {
		return nil, fmt.Errorf("Chrome not found: %w (run 'clicker install' first)", err)
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
	cmd := exec.Command(chromedriverPath, fmt.Sprintf("--port=%d", port))
	setProcGroup(cmd)
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

	// Create session with BiDi enabled
	sessionID, wsURL, userDataDir, err := createSession(baseURL, chromePath, opts.Headless, opts.Verbose)
	if err != nil {
		cmd.Process.Kill()
		return nil, fmt.Errorf("failed to create session: %w", err)
	}
	log.Info("browser launched", "sessionId", sessionID, "wsUrl", wsURL)

	return &LaunchResult{
		WebSocketURL:    wsURL,
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

// createSession creates a new WebDriver session with BiDi enabled.
func createSession(baseURL, chromePath string, headless, verbose bool) (string, string, string, error) {
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

	if headless {
		args = append(args, "--headless=new")
	}

	reqBody := map[string]interface{}{
		"capabilities": map[string]interface{}{
			"alwaysMatch": map[string]interface{}{
				"browserName":              "chrome",
				"webSocketUrl":             true,
				"unhandledPromptBehavior": map[string]interface{}{
					"default": "ignore",
				},
				"goog:chromeOptions": map[string]interface{}{
					"binary":          chromePath,
					"args":            args,
					"excludeSwitches": []string{"enable-automation"},
					"prefs": map[string]interface{}{
						"credentials_enable_service":                          false,
						"profile.password_manager_enabled":                    false,
						"profile.password_manager_leak_detection":             false,
						"profile.default_content_setting_values.notifications": 2,
					},
				},
			},
		},
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", "", "", err
	}

	if verbose {
		fmt.Println("       ------- POST /session -------")
		fmt.Printf("       --> %s\n", string(jsonBody))
	}

	resp, err := http.Post(baseURL+"/session", "application/json", bytes.NewReader(jsonBody))
	if err != nil {
		return "", "", "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return "", "", "", fmt.Errorf("failed to create session: HTTP %d", resp.StatusCode)
	}

	// Read response body for logging and parsing
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", "", fmt.Errorf("failed to read session response: %w", err)
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

	// Delete session first (tells chromedriver to quit Chrome gracefully).
	// Skip on Windows: the DELETE can cause chromedriver to exit before
	// taskkill /T runs, orphaning Chrome children. taskkill /T handles cleanup.
	if !skipGracefulShutdown() && r.SessionID != "" && r.Port > 0 {
		req, _ := http.NewRequest(http.MethodDelete, fmt.Sprintf("http://localhost:%d/session/%s", r.Port, r.SessionID), nil)
		if req != nil {
			client := &http.Client{Timeout: 5 * time.Second}
			client.Do(req)
		}
		// Give Chrome a moment to quit gracefully
		time.Sleep(500 * time.Millisecond)
	}

	// Kill chromedriver and all its descendants
	if r.ChromedriverCmd != nil && r.ChromedriverCmd.Process != nil {
		pid := r.ChromedriverCmd.Process.Pid

		// Kill the entire process tree (chromedriver + Chrome + all helpers)
		killProcessTree(pid)

		// Wait for chromedriver to exit
		r.ChromedriverCmd.Wait()

		process.Untrack(r.ChromedriverCmd)
	}

	// Clean up the Chrome temp profile directory
	if r.UserDataDir != "" {
		log.Debug("removing Chrome user data dir", "path", r.UserDataDir)
		os.RemoveAll(r.UserDataDir)
	}

	// Clean up orphaned Chrome temp directories
	cleanupChromeTempDirs()

	return nil
}

// killProcessTree kills a process and all its descendants.
func killProcessTree(pid int) {
	// First, find all descendant PIDs while parent relationships still exist
	descendants := getDescendants(pid)

	// Kill descendants first (deepest children first)
	for i := len(descendants) - 1; i >= 0; i-- {
		killByPid(descendants[i])
	}

	// Kill the root process
	killByPid(pid)

	// Wait a moment for processes to die
	time.Sleep(100 * time.Millisecond)

	// Kill any orphaned Chrome processes that escaped
	// (Chrome helpers sometimes get reparented to init before we can kill them)
	killOrphanedChromeProcesses()
}

// getDescendants returns all descendant PIDs of a process (recursive).
func getDescendants(pid int) []int {
	var descendants []int

	// Use pgrep to find direct children
	cmd := exec.Command("pgrep", "-P", fmt.Sprintf("%d", pid))
	output, err := cmd.Output()
	if err != nil {
		return descendants
	}

	// Parse child PIDs
	lines := bytes.Split(bytes.TrimSpace(output), []byte("\n"))
	for _, line := range lines {
		if len(line) == 0 {
			continue
		}
		var childPid int
		if _, err := fmt.Sscanf(string(line), "%d", &childPid); err == nil {
			descendants = append(descendants, childPid)
			// Recursively get grandchildren
			descendants = append(descendants, getDescendants(childPid)...)
		}
	}

	return descendants
}

// killOrphanedChromeProcesses finds and kills Chrome/chromedriver processes
// that have been orphaned (reparented to init/launchd).
func killOrphanedChromeProcesses() {
	// Kill orphaned chromedriver and Chrome for Testing processes
	patterns := []string{"chromedriver", "Chrome for Testing"}

	for _, pattern := range patterns {
		cmd := exec.Command("pgrep", "-f", pattern)
		output, err := cmd.Output()
		if err != nil {
			continue
		}

		lines := bytes.Split(bytes.TrimSpace(output), []byte("\n"))
		for _, line := range lines {
			if len(line) == 0 {
				continue
			}
			var pid int
			if _, err := fmt.Sscanf(string(line), "%d", &pid); err == nil {
				// Check if this process's parent is 1 (orphaned)
				ppidCmd := exec.Command("ps", "-o", "ppid=", "-p", fmt.Sprintf("%d", pid))
				ppidOut, err := ppidCmd.Output()
				if err != nil {
					continue
				}
				var ppid int
				if _, err := fmt.Sscanf(string(bytes.TrimSpace(ppidOut)), "%d", &ppid); err == nil {
					if ppid == 1 {
						// This is an orphaned process - kill it and its children
						killProcessTreeByPid(pid)
					}
				}
			}
		}
	}
}

// cleanupChromeTempDirs removes orphaned temp directories created by Chrome for Testing.
// Chrome creates these in os.TempDir() and doesn't clean them up when force-killed.
func cleanupChromeTempDirs() {
	tmpDir := os.TempDir()
	patterns := []string{
		filepath.Join(tmpDir, "com.google.chrome.for.testing.*"),
		filepath.Join(tmpDir, "org.chromium.Chromium.scoped_dir.*"),
	}
	var count int
	for _, pattern := range patterns {
		matches, _ := filepath.Glob(pattern)
		for _, m := range matches {
			if os.RemoveAll(m) == nil {
				count++
			}
		}
	}
	if count > 0 {
		log.Debug("cleaned up Chrome temp dirs", "count", count)
	}
}

// killProcessTreeByPid kills a process and all its descendants by PID.
func killProcessTreeByPid(pid int) {
	descendants := getDescendants(pid)
	for i := len(descendants) - 1; i >= 0; i-- {
		killByPid(descendants[i])
	}
	killByPid(pid)
}

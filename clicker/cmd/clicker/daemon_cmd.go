package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/vibium/clicker/internal/browser"
	"github.com/vibium/clicker/internal/daemon"
	"github.com/vibium/clicker/internal/paths"
)

func newDaemonCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "daemon",
		Short: "Manage the vibium daemon (background browser process)",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
	}

	cmd.AddCommand(newDaemonStartCmd())
	cmd.AddCommand(newDaemonStopCmd())
	cmd.AddCommand(newDaemonStatusCmd())

	return cmd
}

func newDaemonStartCmd() *cobra.Command {
	var (
		foreground  bool
		detach      bool // kept for -d compatibility
		idleTimeout time.Duration
		internal    bool // hidden flag for auto-start
		connectFlag string
		headerFlags []string
		capsFlag    string
	)

	cmd := &cobra.Command{
		Use:   "start",
		Short: "Start the vibium daemon",
		Example: `  vibium daemon start
  # Starts daemon in background

  vibium daemon start --foreground
  # Starts daemon in foreground (for debugging)

  vibium daemon start --idle-timeout 30m
  # Auto-shutdown after 30 minutes of inactivity

  vibium daemon start --connect ws://remote:9515/session
  # Connect to a remote browser instead of launching a local one

  vibium daemon start --connect https://USER:KEY@hub-cloud.browserstack.com/wd/hub \
    --connect-caps '{"bstack:options":{"os":"OS X"}}'
  # Classic WebDriver endpoint (Selenium Grid, cloud grid): vibium creates
  # a session with webSocketUrl:true and connects to the BiDi URL it returns

  vibium --session projA daemon start
  # Isolated session "projA": own daemon, own browser, socket
  # vibium-projA.sock; VIBIUM_SESSION=projA does the same`,
		Run: func(cmd *cobra.Command, args []string) {
			if !foreground && !internal {
				// Daemonize: re-exec as detached child
				daemonize(idleTimeout, connectFlag, headerFlags, capsFlag)
				return
			}

			// Foreground mode (or internal detached child)
			runDaemonForeground(idleTimeout, connectFlag, headerFlags, capsFlag)
		},
	}

	cmd.Flags().BoolVar(&foreground, "foreground", false, "Run daemon in foreground (for debugging)")
	cmd.Flags().BoolVarP(&detach, "detach", "d", true, "Run daemon in background (default, kept for compatibility)")
	cmd.Flags().MarkHidden("detach")
	cmd.Flags().DurationVar(&idleTimeout, "idle-timeout", 30*time.Minute, "Shutdown after this duration of inactivity (0 to disable)")
	cmd.Flags().BoolVar(&internal, "_internal", false, "Internal flag for auto-start")
	cmd.Flags().MarkHidden("_internal")
	cmd.Flags().StringVar(&connectFlag, "connect", "", "Connect to a remote BiDi WebSocket URL instead of launching a local browser")
	cmd.Flags().StringArrayVar(&headerFlags, "connect-header", nil, "HTTP header for WebSocket connect (repeatable, format: \"Key: Value\")")
	cmd.Flags().StringVar(&capsFlag, "connect-caps", "", "Extra alwaysMatch capabilities for classic WebDriver endpoints (JSON object)")

	return cmd
}

func newDaemonStopCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "stop",
		Short: "Stop the vibium daemon",
		Run: func(cmd *cobra.Command, args []string) {
			if !daemon.IsRunning() {
				fmt.Println("Daemon is not running.")
				return
			}

			if err := shutdownDaemonAndWait(); err != nil {
				fmt.Fprintf(os.Stderr, "Error stopping daemon: %v\n", err)
				os.Exit(1)
			}

			fmt.Println("Daemon stopped.")
		},
	}
}

func newDaemonStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show daemon status",
		Run: func(cmd *cobra.Command, args []string) {
			if !daemon.IsRunning() {
				// Exit non-zero so `vibium daemon status` is usable in a
				// conditional, and keep the human line out of --json output.
				if jsonOutput {
					printJSON(map[string]interface{}{
						"running": false,
					})
				} else {
					fmt.Println("Daemon is not running.")
				}
				os.Exit(1)
			}

			status, err := daemon.Status()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error getting status: %v\n", err)
				os.Exit(1)
			}

			if jsonOutput {
				printJSON(map[string]interface{}{
					"running": true,
					"version": status.Version,
					"pid":     status.PID,
					"uptime":  status.Uptime,
					"socket":  status.Socket,
					"session": status.Session,
				})
				return
			}

			fmt.Printf("vibium daemon v%s\n", status.Version)
			fmt.Printf("status:   running\n")
			fmt.Printf("pid:      %d\n", status.PID)
			fmt.Printf("uptime:   %s\n", status.Uptime)
			fmt.Printf("socket:   %s\n", status.Socket)
			if status.Session != "" {
				fmt.Printf("session:  %s\n", status.Session)
			}
		},
	}
}

// shutdownDaemonAndWait asks the daemon to shut down and waits for the process
// to fully exit (including Chrome cleanup). No-op if it is not running.
func shutdownDaemonAndWait() error {
	if !daemon.IsRunning() {
		return nil
	}

	// Read PID before sending shutdown so we can wait for the process to exit
	pid, _ := daemon.ReadPID()

	if err := daemon.Shutdown(); err != nil {
		return err
	}

	if pid > 0 {
		deadline := time.Now().Add(10 * time.Second)
		for time.Now().Before(deadline) {
			if !daemon.ProcessExists(pid) {
				break
			}
			time.Sleep(100 * time.Millisecond)
		}
	}

	return nil
}

// resolveConnect merges CLI flags with env vars. Flags take precedence.
func resolveConnect(connectFlag string, headerFlags []string, capsFlag string) (string, http.Header, map[string]interface{}) {
	connectURL := connectFlag
	if connectURL == "" {
		connectURL, _ = connectFromEnv()
	}

	caps := parseConnectCaps(capsFlag)
	if caps == nil {
		caps = connectCapsFromEnv()
	}

	var headers http.Header

	// Start with env var API key
	_, envHeaders := connectFromEnv()
	if envHeaders != nil {
		headers = envHeaders.Clone()
	}

	// CLI --connect-header flags override / add
	if len(headerFlags) > 0 {
		if headers == nil {
			headers = make(http.Header)
		}
		for _, h := range headerFlags {
			parts := strings.SplitN(h, ":", 2)
			if len(parts) == 2 {
				headers.Set(strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]))
			}
		}
	}

	return connectURL, headers, caps
}

// runDaemonForeground starts the daemon in the current process.
func runDaemonForeground(idleTimeout time.Duration, connectFlag string, headerFlags []string, capsFlag string) {
	// Clean stale files from a previous crash
	daemon.CleanStale()

	// Reclaim Chrome profile dirs orphaned by earlier crashed/killed sessions
	// (parallel-safe: the minAge filter skips any live sibling's dir).
	browser.CleanupOrphanedBrowserTempDirs(time.Minute)

	if daemon.IsRunning() {
		fmt.Fprintln(os.Stderr, "Daemon is already running.")
		os.Exit(1)
	}

	screenshotDir := ""
	defaultDir, err := paths.GetScreenshotDir()
	if err == nil {
		screenshotDir = defaultDir
	}

	connectURL, connectHeaders, connectCaps := resolveConnect(connectFlag, headerFlags, capsFlag)

	d := daemon.New(daemon.Options{
		Version:        version,
		ScreenshotDir:  screenshotDir,
		Engine:         engineName,
		Headless:       headless,
		IdleTimeout:    idleTimeout,
		ConnectURL:     connectURL,
		ConnectHeaders: connectHeaders,
		ConnectCaps:    connectCaps,
	})

	// Install signal handler for clean shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigCh
		fmt.Fprintf(os.Stderr, "\nDaemon shutting down...\n")
		d.Shutdown()
	}()

	socketPath, _ := paths.GetSocketPath()
	fmt.Fprintf(os.Stderr, "Daemon starting (pid %d, socket %s)\n", os.Getpid(), socketPath)

	ctx := context.Background()
	if err := d.Run(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Daemon error: %v\n", err)
		os.Exit(1)
	}
}

// daemonize spawns the daemon as a detached background process.
func daemonize(idleTimeout time.Duration, connectFlag string, headerFlags []string, capsFlag string) {
	// Resolve the socket path first: an unusable path (bad session name,
	// over-long socket path) should fail here, not in the detached child.
	socketPath, err := paths.GetSocketPath()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Clean stale files first
	daemon.CleanStale()

	if daemon.IsRunning() {
		fmt.Println("Daemon is already running.")
		return
	}

	exe, err := os.Executable()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error finding executable: %v\n", err)
		os.Exit(1)
	}

	args := []string{"daemon", "start", "--_internal",
		fmt.Sprintf("--idle-timeout=%s", idleTimeout)}
	if headless {
		args = append(args, "--headless")
	}
	if engineName != "chrome" {
		args = append(args, "--engine="+engineName)
	}

	// Forward connect flags to child process
	if connectFlag != "" {
		args = append(args, fmt.Sprintf("--connect=%s", connectFlag))
	}
	for _, h := range headerFlags {
		args = append(args, fmt.Sprintf("--connect-header=%s", h))
	}
	if capsFlag != "" {
		args = append(args, fmt.Sprintf("--connect-caps=%s", capsFlag))
	}

	cmd := exec.Command(exe, args...)
	cmd.Stdout = nil
	cmd.Stderr = nil
	cmd.Stdin = nil

	// Detach the child process
	setSysProcAttr(cmd)

	if err := cmd.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "Error starting daemon: %v\n", err)
		os.Exit(1)
	}

	// Poll for socket availability
	if err := waitForSocket(socketPath, 5*time.Second); err != nil {
		fmt.Fprintf(os.Stderr, "Daemon failed to start: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Daemon started (pid %d)\n", cmd.Process.Pid)
}

// waitForSocket polls until the socket is connectable or timeout.
func waitForSocket(socketPath string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	interval := 50 * time.Millisecond

	for time.Now().Before(deadline) {
		conn, err := dialSocket(socketPath, 500*time.Millisecond)
		if err == nil {
			conn.Close()
			return nil
		}
		time.Sleep(interval)
		if interval < 500*time.Millisecond {
			interval *= 2
		}
	}

	return fmt.Errorf("socket not available after %s", timeout)
}

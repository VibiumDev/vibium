package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"
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
		detach      bool
		idleTimeout time.Duration
		internal    bool // hidden flag for auto-start
	)

	cmd := &cobra.Command{
		Use:   "start",
		Short: "Start the vibium daemon",
		Example: `  vibium daemon start
  # Starts daemon in foreground

  vibium daemon start -d
  # Starts daemon in background

  vibium daemon start --idle-timeout 30m
  # Auto-shutdown after 30 minutes of inactivity`,
		Run: func(cmd *cobra.Command, args []string) {
			if detach && !internal {
				// Daemonize: re-exec as detached child
				daemonize(idleTimeout)
				return
			}

			// Foreground mode (or internal detached child)
			runDaemonForeground(idleTimeout)
		},
	}

	cmd.Flags().BoolVarP(&detach, "detach", "d", false, "Run daemon in background")
	cmd.Flags().DurationVar(&idleTimeout, "idle-timeout", 30*time.Minute, "Shutdown after this duration of inactivity (0 to disable)")
	cmd.Flags().BoolVar(&internal, "_internal", false, "Internal flag for auto-start")
	cmd.Flags().MarkHidden("_internal")

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

			if err := daemon.Shutdown(); err != nil {
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
				fmt.Println("Daemon is not running.")
				if jsonOutput {
					printJSON(map[string]interface{}{
						"running": false,
					})
				}
				return
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
				})
				return
			}

			fmt.Printf("vibium daemon v%s\n", status.Version)
			fmt.Printf("status:   running\n")
			fmt.Printf("pid:      %d\n", status.PID)
			fmt.Printf("uptime:   %s\n", status.Uptime)
			fmt.Printf("socket:   %s\n", status.Socket)
		},
	}
}

// runDaemonForeground starts the daemon in the current process.
func runDaemonForeground(idleTimeout time.Duration) {
	// Clean stale files from a previous crash
	daemon.CleanStale()

	if daemon.IsRunning() {
		fmt.Fprintln(os.Stderr, "Daemon is already running.")
		os.Exit(1)
	}

	screenshotDir := ""
	defaultDir, err := paths.GetScreenshotDir()
	if err == nil {
		screenshotDir = defaultDir
	}

	d := daemon.New(daemon.Options{
		Version:       version,
		ScreenshotDir: screenshotDir,
		Headless:      headless,
		IdleTimeout:   idleTimeout,
	})

	// Install signal handler for clean shutdown
	// This overrides the global signal handler from process.SetupSignalHandler()
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
func daemonize(idleTimeout time.Duration) {
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
	socketPath, _ := paths.GetSocketPath()
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

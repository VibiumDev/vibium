package main

import (
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/spf13/cobra"
	"github.com/vibium/clicker/internal/daemon"
	"github.com/vibium/clicker/internal/paths"
)

func newStartCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "start [url]",
		Short: "Start a browser session",
		Long: `Start a browser session. Without arguments, launches a local browser.
With a URL argument, connects to a remote BiDi WebSocket endpoint.

If no URL is given, checks VIBIUM_CONNECT_URL env var before falling
back to a local browser launch.

Set VIBIUM_CONNECT_API_KEY to send an Authorization: Bearer header.`,
		Example: `  vibium start
  # Start with a local browser

  vibium start ws://remote:9515/session
  # Connect to a remote browser

  export VIBIUM_CONNECT_URL=wss://cloud.example.com/session
  export VIBIUM_CONNECT_API_KEY=my-api-key
  vibium start
  # Connect using env vars`,
		Args: cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			// Determine connect URL: arg > env > local
			var connectURL string
			if len(args) > 0 {
				connectURL = args[0]
			} else {
				connectURL, _ = connectFromEnv()
			}

			if connectURL == "" {
				// Local launch — just ensure daemon is running (lazy browser launch)
				result, err := daemonCall("browser_start", map[string]interface{}{})
				if err != nil {
					printError(err)
					return
				}
				printResult(result)
				return
			}

			// Remote connect — stop existing daemon and start fresh with --connect
			if err := shutdownDaemonAndWait(); err != nil {
				fmt.Fprintf(os.Stderr, "Error stopping existing daemon: %v\n", err)
				os.Exit(1)
			}

			daemon.CleanStale()

			exe, err := os.Executable()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error finding executable: %v\n", err)
				os.Exit(1)
			}

			daemonArgs := []string{"daemon", "start", "--_internal", "--idle-timeout=30m",
				fmt.Sprintf("--connect=%s", connectURL)}
			if headless {
				daemonArgs = append(daemonArgs, "--headless")
			}

			_, envHeaders := connectFromEnv()
			for key, vals := range envHeaders {
				for _, v := range vals {
					daemonArgs = append(daemonArgs, fmt.Sprintf("--connect-header=%s: %s", key, v))
				}
			}

			child := exec.Command(exe, daemonArgs...)
			child.Stdout = nil
			child.Stderr = nil
			child.Stdin = nil
			setSysProcAttr(child)

			if err := child.Start(); err != nil {
				fmt.Fprintf(os.Stderr, "Error starting daemon: %v\n", err)
				os.Exit(1)
			}

			socketPath, _ := paths.GetSocketPath()
			if err := waitForSocket(socketPath, 5*time.Second); err != nil {
				fmt.Fprintf(os.Stderr, "Daemon failed to start: %v\n", err)
				os.Exit(1)
			}

			// Connect now instead of leaving it to the first command that
			// needs a browser, so a bad URL or a dead endpoint fails here,
			// where the user typed it. A daemon that cannot reach its
			// endpoint is no use to the next command either — take it down
			// rather than leave it to auto-start the same failure.
			if _, err := daemonCall("browser_start", map[string]interface{}{}); err != nil {
				fmt.Fprintf(os.Stderr, "Failed to connect to %s: %v\n", connectURL, err)
				shutdownDaemonAndWait()
				os.Exit(1)
			}

			fmt.Printf("Connected to %s (daemon pid %d)\n", connectURL, child.Process.Pid)
		},
	}
}

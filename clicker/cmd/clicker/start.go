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

  vibium start --engine firefox
  # Start with Firefox instead of Chrome

  vibium start --engine firefox --channel beta
  # Start with the installed Firefox beta instead of the release build

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
				// Local launch — just ensure daemon is running (lazy browser launch).
				// Carry --headless on the call: the flag otherwise only reaches a
				// daemon we spawn ourselves, so it is silently dropped whenever one
				// is already up. Only when explicitly given, so a bare `start`
				// does not override a daemon deliberately started headless.
				startArgs := map[string]interface{}{}
				if cmd.Flags().Changed("headless") {
					startArgs["headless"] = headless
				}
				// Same reasoning for --engine; VIBIUM_ENGINE changes the
				// default without marking the flag Changed, so check both.
				if cmd.Flags().Changed("engine") || engineName != "chrome" {
					startArgs["engine"] = engineName
				}
				result, err := daemonCall("browser_start", startArgs)
				if err != nil {
					printError(err)
					return
				}
				printResult(result)
				return
			}

			// Remote connect — stop existing daemon and start fresh with --connect.
			// Failures go through printError: these paths wrote straight to
			// stderr, which is why --json never reached them (#451).
			if err := shutdownDaemonAndWait(); err != nil {
				printError(fmt.Errorf("stopping existing daemon: %w", err))
				return
			}

			daemon.CleanStale()

			exe, err := os.Executable()
			if err != nil {
				printError(fmt.Errorf("finding executable: %w", err))
				return
			}

			daemonArgs := []string{"daemon", "start", "--_internal", "--idle-timeout=30m",
				fmt.Sprintf("--connect=%s", connectURL)}
			if headless {
				daemonArgs = append(daemonArgs, "--headless")
			}
			if engineName != "chrome" {
				daemonArgs = append(daemonArgs, "--engine="+engineName)
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
				printError(fmt.Errorf("starting daemon: %w", err))
				return
			}

			socketPath, _ := paths.GetSocketPath()
			if err := waitForSocket(socketPath, 5*time.Second); err != nil {
				printError(fmt.Errorf("daemon failed to start: %w", err))
				return
			}

			// Connect now instead of leaving it to the first command that
			// needs a browser, so a bad URL or a dead endpoint fails here,
			// where the user typed it. A daemon that cannot reach its
			// endpoint is no use to the next command either — take it down
			// rather than leave it to auto-start the same failure.
			if _, err := daemonCall("browser_start", map[string]interface{}{}); err != nil {
				// Take the daemon down before reporting: it cannot reach the
				// endpoint either, so leaving it up only defers the failure.
				shutdownDaemonAndWait()
				printError(fmt.Errorf("failed to connect to %s: %w", connectURL, err))
				return
			}

			msg := fmt.Sprintf("Connected to %s (daemon pid %d)", connectURL, child.Process.Pid)
			if jsonOutput {
				printJSON(jsonEnvelope{OK: true, Result: msg})
				return
			}
			fmt.Println(msg)
		},
	}
}

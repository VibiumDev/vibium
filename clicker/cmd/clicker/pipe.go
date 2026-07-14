package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/vibium/clicker/internal/api"
	"github.com/vibium/clicker/internal/browser"
)

func newPipeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pipe",
		Short: "Run as a child process communicating via stdin/stdout pipes",
		Long: `Start vibium in pipe mode where protocol messages are exchanged
over stdin (commands) and stdout (responses/events) as newline-delimited JSON.
Diagnostic output goes to stderr. This mode is used by client libraries.

Use --connect to proxy to a remote BiDi endpoint instead of launching a local browser.`,
		Example: `  echo '{"id":1,"method":"vibium:browser.page","params":{}}' | vibium pipe --headless

  # Connect to a remote browser
  vibium pipe --connect ws://remote:9515

  # Connect with auth header
  vibium pipe --connect wss://cloud.example.com/bidi --connect-header "Authorization: Bearer token"`,
		Run: func(cmd *cobra.Command, args []string) {
			connectURL, _ := cmd.Flags().GetString("connect")
			headerStrs, _ := cmd.Flags().GetStringArray("connect-header")

			var connectHeaders http.Header
			if len(headerStrs) > 0 {
				connectHeaders = make(http.Header)
				for _, h := range headerStrs {
					parts := strings.SplitN(h, ":", 2)
					if len(parts) == 2 {
						connectHeaders.Add(strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]))
					}
				}
			}

			runPipe(connectURL, connectHeaders)
		},
	}
	cmd.Flags().String("connect", "", "Connect to a remote BiDi WebSocket URL instead of launching a local browser")
	cmd.Flags().StringArray("connect-header", nil, "HTTP header for WebSocket connect (repeatable, format: \"Key: Value\")")
	return cmd
}

func runPipe(connectURL string, connectHeaders http.Header) {
	// Save a reference to the real fd 1 for protocol output BEFORE redirecting.
	fd, err := dupFd(os.Stdout.Fd())
	if err != nil {
		fmt.Fprintf(os.Stderr, "[pipe] Failed to dup stdout: %v\n", err)
		os.Exit(1)
	}
	protocolOut := os.NewFile(fd, "protocolOut")

	// Redirect os.Stdout to stderr so any stray fmt.Print / log output
	// doesn't corrupt the protocol stream.
	os.Stdout = os.Stderr

	router := api.NewRouter(headless, connectURL, connectHeaders)
	client := api.NewPipeClientConn(protocolOut)

	// OnClientConnect blocks until Chrome is launched, BiDi connected,
	// and events subscribed — the client won't see messages until it's ready.
	router.OnClientConnect(client)

	// Send ready signal so the client knows it can start sending commands.
	ready := map[string]interface{}{
		"method": "vibium:lifecycle.ready",
		"params": map[string]interface{}{
			"version": version,
		},
	}
	readyJSON, _ := json.Marshal(ready)
	if err := client.Send(string(readyJSON)); err != nil {
		fmt.Fprintf(os.Stderr, "[pipe] Failed to send ready signal: %v\n", err)
		os.Exit(1)
	}

	// Handle signals for clean shutdown
	sigCh := make(chan os.Signal, 1)
	notifyShutdownSignals(sigCh)

	// Read commands from stdin line by line
	done := make(chan struct{})
	go func() {
		defer close(done)
		scanner := bufio.NewScanner(os.Stdin)
		// Allow large messages (10MB, matching WebSocket limit)
		scanner.Buffer(make([]byte, 0, 64*1024), 10*1024*1024)
		for scanner.Scan() {
			line := scanner.Text()
			if line == "" {
				continue
			}
			router.OnClientMessage(client, line)
		}
	}()

	// Wait for stdin EOF or signal
	select {
	case <-done:
		// stdin closed (parent process ended or sent EOF)
	case <-sigCh:
		// Received SIGTERM/SIGINT
	}

	// Clean up — kill THIS process's chromedriver process tree, remove its
	// user-data-dir, and sweep any Chrome subprocesses orphaned by the
	// chromedriver kill (PPID=1). KillOrphanedChromeProcesses is safe
	// under parallel runs because a sibling vibium's Chrome processes
	// have PPID = sibling's chromedriver (not 1) — they're not matched.
	router.OnClientDisconnect(client)
	router.CloseAll()
	browser.KillOrphanedChromeProcesses()

	protocolOut.Close()
}

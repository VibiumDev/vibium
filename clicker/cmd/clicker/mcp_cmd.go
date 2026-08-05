package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/vibium/clicker/internal/agent"
	"github.com/vibium/clicker/internal/paths"
	"github.com/vibium/clicker/internal/process"
)

func newMCPCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mcp",
		Short: "Start MCP server (stdio JSON-RPC for LLM agents)",
		Long: `Start the Model Context Protocol (MCP) server.

This runs a JSON-RPC 2.0 server over stdin/stdout, designed for integration
with LLM agents like Claude Code.

The server provides browser automation tools:
  - browser_start: Start a browser session
  - browser_navigate: Go to a URL
  - browser_click: Click an element
  - browser_type: Type into an element
  - browser_screenshot: Capture the page
  - browser_find: Find element info
  - browser_evaluate: Execute JavaScript
  - browser_stop: Stop the browser
  - browser_get_text: Get page/element text
  - browser_get_url: Get current URL
  - browser_get_title: Get page title
  - browser_get_html: Get page/element HTML
  - browser_find_all: Find all matching elements
  - browser_wait: Wait for element state
  - browser_hover: Hover over an element
  - browser_select: Select a dropdown option
  - browser_scroll: Scroll the page
  - browser_keys: Press keys
  - browser_new_page: Open a new page
  - browser_list_pages: List open pages
  - browser_switch_page: Switch pages
  - browser_close_page: Close a page`,
		Example: `  # Run directly (for testing)
  vibium mcp

  # Launch browsers with no visible window (servers, CI)
  vibium mcp --headless

  # Configure in Claude Code
  claude mcp add vibium -- vibium mcp

  # Custom screenshot directory
  vibium mcp --screenshot-dir ./screenshots

  # Disable screenshot file saving (inline only)
  vibium mcp --screenshot-dir ""

  # Test with echo
  echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"capabilities":{}}}' | vibium mcp`,
		Run: func(cmd *cobra.Command, args []string) {
			process.WithCleanup(func() {
				// If running in a terminal, print helpful info to stderr
				if stat, _ := os.Stdin.Stat(); (stat.Mode() & os.ModeCharDevice) != 0 {
					fmt.Fprintf(os.Stderr, "Vibium MCP server v%s\n", version)
					fmt.Fprintln(os.Stderr, "This server communicates via JSON-RPC over stdin/stdout.")
					fmt.Fprintln(os.Stderr, "It's meant to be run by an MCP client (e.g., Claude Desktop).")
					fmt.Fprintln(os.Stderr, "")

					// Show Chrome for Testing status
					chromePath, chromeErr := paths.GetChromeExecutable()
					chromedriverPath, driverErr := paths.GetChromedriverPath()

					if chromeErr != nil || driverErr != nil {
						fmt.Fprintln(os.Stderr, "Chrome for Testing: not installed")
						fmt.Fprintln(os.Stderr, "Run 'vibium install' to download Chrome for Testing and chromedriver.")
					} else {
						fmt.Fprintf(os.Stderr, "Chrome: %s\n", chromePath)
						fmt.Fprintf(os.Stderr, "Chromedriver: %s\n", chromedriverPath)
					}

					fmt.Fprintln(os.Stderr, "")
					fmt.Fprintln(os.Stderr, "Waiting for client connection on stdin...")
				}

				var screenshotDir string

				// Check if flag was explicitly set
				if cmd.Flags().Changed("screenshot-dir") {
					screenshotDir, _ = cmd.Flags().GetString("screenshot-dir")
					// Empty string means explicitly disabled
				} else {
					// Use platform-specific default
					defaultDir, err := paths.GetScreenshotDir()
					if err != nil {
						fmt.Fprintf(os.Stderr, "Warning: could not determine default screenshot directory: %v\n", err)
					} else {
						screenshotDir = defaultDir
					}
				}

				connectURL, connectHeaders := connectFromEnv()
				headless, _ := cmd.Flags().GetBool("headless")

				server := agent.NewServer(version, agent.ServerOptions{
					ScreenshotDir:  screenshotDir,
					Engine:         engineName,
					Headless:       headless,
					ConnectURL:     connectURL,
					ConnectHeaders: connectHeaders,
					ConnectCaps:    connectCapsFromEnv(),
				})
				defer server.Close()

				// Handle SIGTERM so Chrome is cleaned up even if stdin isn't closed
				sigCh := make(chan os.Signal, 1)
				signal.Notify(sigCh, syscall.SIGTERM)
				go func() {
					<-sigCh
					server.Close()
					os.Exit(0)
				}()

				if err := server.Run(); err != nil {
					fmt.Fprintf(os.Stderr, "MCP server error: %v\n", err)
					os.Exit(1)
				}
			})
		},
	}
	cmd.Flags().String("screenshot-dir", "", "Directory for saving screenshots (default: ~/Pictures/Vibium, use \"\" to disable)")
	cmd.Flags().Bool("headless", false, "Launch browsers in headless mode (no visible window)")
	return cmd
}

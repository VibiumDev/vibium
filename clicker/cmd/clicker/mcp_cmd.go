package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/vibium/clicker/internal/mcp"
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
  - browser_launch: Start a browser session
  - browser_navigate: Go to a URL
  - browser_click: Click an element
  - browser_type: Type into an element
  - browser_screenshot: Capture the page
  - browser_find: Find element info
  - browser_evaluate: Execute JavaScript
  - browser_quit: Close the browser
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
  - browser_new_tab: Open a new tab
  - browser_list_tabs: List open tabs
  - browser_switch_tab: Switch tabs
  - browser_close_tab: Close a tab`,
		Example: `  # Run directly (for testing)
  vibium mcp

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

				server := mcp.NewServer(version, mcp.ServerOptions{
					ScreenshotDir: screenshotDir,
				})
				defer server.Close()

				if err := server.Run(); err != nil {
					fmt.Fprintf(os.Stderr, "MCP server error: %v\n", err)
					os.Exit(1)
				}
			})
		},
	}
	cmd.Flags().String("screenshot-dir", "", "Directory for saving screenshots (default: ~/Pictures/Vibium, use \"\" to disable)")
	return cmd
}

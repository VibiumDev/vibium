package main

import (
	"bufio"
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/vibium/clicker/internal/bidi"
	"github.com/vibium/clicker/internal/browser"
	"github.com/vibium/clicker/internal/features"
	"github.com/vibium/clicker/internal/log"
	"github.com/vibium/clicker/internal/mcp"
	"github.com/vibium/clicker/internal/paths"
	"github.com/vibium/clicker/internal/process"
	"github.com/vibium/clicker/internal/proxy"
	"github.com/vibium/clicker/internal/recording"
)

var version = "0.1.0"

// Global flags
var (
	headless  bool
	waitOpen  int
	waitClose int
	verbose   bool
)

// doWaitOpen waits for page to load if --wait-open is set.
func doWaitOpen() {
	if waitOpen > 0 {
		fmt.Printf("Waiting %d seconds for page to load...\n", waitOpen)
		time.Sleep(time.Duration(waitOpen) * time.Second)
	}
}

// waitAndClose handles the --wait-close flag before closing the browser.
func waitAndClose(launchResult *browser.LaunchResult) {
	if waitClose > 0 {
		fmt.Printf("\nKeeping browser open for %d seconds...\n", waitClose)
		time.Sleep(time.Duration(waitClose) * time.Second)
	}
	launchResult.Close()
}

// printCheck prints an actionability check result with a checkmark or X.
func printCheck(name string, passed bool) {
	if passed {
		fmt.Printf("✓ %s: true\n", name)
	} else {
		fmt.Printf("✗ %s: false\n", name)
	}
}

func main() {
	// Setup signal handler to cleanup on Ctrl+C
	process.SetupSignalHandler()

	rootCmd := &cobra.Command{
		Use:   "clicker",
		Short: "Browser automation for AI agents and humans",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			// Enable logging only if --verbose is used
			if verbose {
				log.Setup(log.LevelVerbose)
			}
		},
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
	}

	// Add global flags for browser commands
	rootCmd.PersistentFlags().BoolVar(&headless, "headless", false, "Hide browser window (visible by default)")
	rootCmd.PersistentFlags().IntVar(&waitOpen, "wait-open", 0, "Seconds to wait after navigation for page to load")
	rootCmd.PersistentFlags().IntVar(&waitClose, "wait-close", 0, "Seconds to keep browser open before closing")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable debug logging")

	rootCmd.AddCommand(&cobra.Command{
		Use:   "version",
		Short: "Print the version number",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("Clicker v%s\n", version)
		},
	})

	rootCmd.AddCommand(&cobra.Command{
		Use:   "paths",
		Short: "Print browser and cache paths",
		Run: func(cmd *cobra.Command, args []string) {
			cacheDir, err := paths.GetCacheDir()
			if err != nil {
				fmt.Printf("Cache directory: error: %v\n", err)
			} else {
				fmt.Printf("Cache directory: %s\n", cacheDir)
			}

			chromePath, err := paths.GetChromeExecutable()
			if err != nil {
				fmt.Println("Chrome: not found")
			} else {
				fmt.Printf("Chrome: %s\n", chromePath)
			}

			chromedriverPath, err := paths.GetChromedriverPath()
			if err != nil {
				fmt.Println("Chromedriver: not found")
			} else {
				fmt.Printf("Chromedriver: %s\n", chromedriverPath)
			}
		},
	})

	rootCmd.AddCommand(&cobra.Command{
		Use:   "install",
		Short: "Download Chrome for Testing and chromedriver",
		Run: func(cmd *cobra.Command, args []string) {
			result, err := browser.Install()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}

			fmt.Println("Installation complete!")
			fmt.Printf("Chrome: %s\n", result.ChromePath)
			fmt.Printf("Chromedriver: %s\n", result.ChromedriverPath)
			fmt.Printf("Version: %s\n", result.Version)
		},
	})

	rootCmd.AddCommand(&cobra.Command{
		Use:   "launch-test",
		Short: "Launch browser via chromedriver and print BiDi WebSocket URL",
		Run: func(cmd *cobra.Command, args []string) {
			result, err := browser.Launch(browser.LaunchOptions{Headless: headless})
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}

			fmt.Printf("Session ID: %s\n", result.SessionID)
			fmt.Printf("BiDi WebSocket: %s\n", result.WebSocketURL)
			fmt.Println("Press Ctrl+C to stop...")

			// Wait for signal, then cleanup
			process.WaitForSignal()
			result.Close()
		},
	})

	rootCmd.AddCommand(&cobra.Command{
		Use:   "ws-test [url]",
		Short: "Test WebSocket connection (type messages, see echoes)",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			url := args[0]
			fmt.Printf("Connecting to %s...\n", url)

			conn, err := bidi.Connect(url)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			defer conn.Close()

			fmt.Println("Connected! Type messages (Ctrl+C to quit):")

			// Read responses in background
			go func() {
				for {
					msg, err := conn.Receive()
					if err != nil {
						return
					}
					fmt.Printf("< %s\n", msg)
				}
			}()

			// Read input and send
			scanner := bufio.NewScanner(os.Stdin)
			for scanner.Scan() {
				msg := scanner.Text()
				if err := conn.Send(msg); err != nil {
					fmt.Fprintf(os.Stderr, "Send error: %v\n", err)
					break
				}
				fmt.Printf("> %s\n", msg)
			}
		},
	})

	rootCmd.AddCommand(&cobra.Command{
		Use:   "bidi-test",
		Short: "Launch browser, connect via BiDi, send session.status",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("[1/5] Launching chromedriver...")
			launchResult, err := browser.Launch(browser.LaunchOptions{Headless: true, Verbose: true})
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error launching browser: %v\n", err)
				os.Exit(1)
			}
			defer waitAndClose(launchResult)
			fmt.Printf("       Chromedriver started on port %d\n", launchResult.Port)
			fmt.Printf("       Session ID: %s\n", launchResult.SessionID)

			fmt.Println("[2/5] WebDriver session created with BiDi enabled")
			fmt.Printf("       WebSocket URL: %s\n", launchResult.WebSocketURL)

			fmt.Println("[3/5] Connecting to BiDi WebSocket...")
			conn, err := bidi.Connect(launchResult.WebSocketURL)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error connecting: %v\n", err)
				os.Exit(1)
			}
			defer conn.Close()
			fmt.Println("       Connected!")

			fmt.Println("[4/5] Sending BiDi command: session.status")
			client := bidi.NewClient(conn)
			client.SetVerbose(true)

			status, err := client.SessionStatus()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}

			fmt.Println("[5/5] Parsed response:")
			fmt.Printf("       Ready: %v\n", status.Ready)
			fmt.Printf("       Message: %s\n", status.Message)

			fmt.Println("\nTest complete!")
		},
	})

	rootCmd.AddCommand(&cobra.Command{
		Use:   "navigate [url]",
		Short: "Navigate to a URL and print page info",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			process.WithCleanup(func() {
				url := args[0]

				fmt.Println("Launching browser...")
				launchResult, err := browser.Launch(browser.LaunchOptions{Headless: headless})
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error launching browser: %v\n", err)
					os.Exit(1)
				}
				defer waitAndClose(launchResult)

				fmt.Println("Connecting to BiDi...")
				conn, err := bidi.Connect(launchResult.WebSocketURL)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error connecting: %v\n", err)
					os.Exit(1)
				}
				defer conn.Close()

				client := bidi.NewClient(conn)

				fmt.Printf("Navigating to %s...\n", url)
				result, err := client.Navigate("", url)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error navigating: %v\n", err)
					os.Exit(1)
				}

				fmt.Printf("Navigation complete!\n")
				fmt.Printf("  URL: %s\n", result.URL)
				fmt.Printf("  Navigation ID: %s\n", result.Navigation)
			})
		},
	})

	screenshotCmd := &cobra.Command{
		Use:   "screenshot [url]",
		Short: "Navigate to a URL and capture a screenshot",
		Example: `  clicker screenshot https://example.com -o shot.png
  # Saves screenshot to shot.png`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			process.WithCleanup(func() {
				url := args[0]
				output, _ := cmd.Flags().GetString("output")

				fmt.Println("Launching browser...")
				launchResult, err := browser.Launch(browser.LaunchOptions{Headless: headless})
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error launching browser: %v\n", err)
					os.Exit(1)
				}
				defer waitAndClose(launchResult)

				fmt.Println("Connecting to BiDi...")
				conn, err := bidi.Connect(launchResult.WebSocketURL)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error connecting: %v\n", err)
					os.Exit(1)
				}
				defer conn.Close()

				client := bidi.NewClient(conn)

				fmt.Printf("Navigating to %s...\n", url)
				_, err = client.Navigate("", url)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error navigating: %v\n", err)
					os.Exit(1)
				}

				doWaitOpen()

				fmt.Println("Capturing screenshot...")
				base64Data, err := client.CaptureScreenshot("")
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error capturing screenshot: %v\n", err)
					os.Exit(1)
				}

				// Decode base64 to PNG bytes
				pngData, err := base64.StdEncoding.DecodeString(base64Data)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error decoding screenshot: %v\n", err)
					os.Exit(1)
				}

				// Save to file
				if err := os.WriteFile(output, pngData, 0644); err != nil {
					fmt.Fprintf(os.Stderr, "Error saving screenshot: %v\n", err)
					os.Exit(1)
				}

				fmt.Printf("Screenshot saved to %s (%d bytes)\n", output, len(pngData))
			})
		},
	}
	screenshotCmd.Flags().StringP("output", "o", "screenshot.png", "Output file path")
	rootCmd.AddCommand(screenshotCmd)

	rootCmd.AddCommand(&cobra.Command{
		Use:   "eval [url] [expression]",
		Short: "Navigate to a URL and evaluate a JavaScript expression",
		Example: `  clicker eval https://example.com "document.title"
  # Prints: Example Domain`,
		Args: cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			process.WithCleanup(func() {
				url := args[0]
				expression := args[1]

				fmt.Println("Launching browser...")
				launchResult, err := browser.Launch(browser.LaunchOptions{Headless: headless})
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error launching browser: %v\n", err)
					os.Exit(1)
				}
				defer waitAndClose(launchResult)

				fmt.Println("Connecting to BiDi...")
				conn, err := bidi.Connect(launchResult.WebSocketURL)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error connecting: %v\n", err)
					os.Exit(1)
				}
				defer conn.Close()

				client := bidi.NewClient(conn)

				fmt.Printf("Navigating to %s...\n", url)
				_, err = client.Navigate("", url)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error navigating: %v\n", err)
					os.Exit(1)
				}

				doWaitOpen()

				fmt.Printf("Evaluating: %s\n", expression)
				result, err := client.Evaluate("", expression)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error evaluating: %v\n", err)
					os.Exit(1)
				}

				fmt.Printf("Result: %v\n", result)
			})
		},
	})

	rootCmd.AddCommand(&cobra.Command{
		Use:   "find [url] [selector]",
		Short: "Navigate to a URL and find an element by CSS selector",
		Example: `  clicker find https://example.com "a"
  # Prints: tag=A, text="Learn more", box={x,y,w,h}`,
		Args: cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			process.WithCleanup(func() {
				url := args[0]
				selector := args[1]

				fmt.Println("Launching browser...")
				launchResult, err := browser.Launch(browser.LaunchOptions{Headless: headless})
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error launching browser: %v\n", err)
					os.Exit(1)
				}
				defer waitAndClose(launchResult)

				fmt.Println("Connecting to BiDi...")
				conn, err := bidi.Connect(launchResult.WebSocketURL)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error connecting: %v\n", err)
					os.Exit(1)
				}
				defer conn.Close()

				client := bidi.NewClient(conn)

				fmt.Printf("Navigating to %s...\n", url)
				_, err = client.Navigate("", url)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error navigating: %v\n", err)
					os.Exit(1)
				}

				doWaitOpen()

				fmt.Printf("Finding element: %s\n", selector)
				info, err := client.FindElement("", selector)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error finding element: %v\n", err)
					os.Exit(1)
				}

				fmt.Printf("Found: tag=%s, text=\"%s\", box={x:%.0f, y:%.0f, w:%.0f, h:%.0f}\n",
					info.Tag, info.Text, info.Box.X, info.Box.Y, info.Box.Width, info.Box.Height)
			})
		},
	})

	clickCmd := &cobra.Command{
		Use:   "click [url] [selector]",
		Short: "Navigate to a URL and click an element (with actionability checks)",
		Example: `  clicker click https://example.com "a"
  # Waits for element to be visible, stable, receive events, and enabled
  # Then clicks the link and navigates to the target page

  clicker click https://example.com "a" --timeout 5s
  # Custom timeout for actionability checks`,
		Args: cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			process.WithCleanup(func() {
				url := args[0]
				selector := args[1]
				timeout, _ := cmd.Flags().GetDuration("timeout")

				fmt.Println("Launching browser...")
				launchResult, err := browser.Launch(browser.LaunchOptions{Headless: headless})
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error launching browser: %v\n", err)
					os.Exit(1)
				}
				defer waitAndClose(launchResult)

				fmt.Println("Connecting to BiDi...")
				conn, err := bidi.Connect(launchResult.WebSocketURL)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error connecting: %v\n", err)
					os.Exit(1)
				}
				defer conn.Close()

				client := bidi.NewClient(conn)

				fmt.Printf("Navigating to %s...\n", url)
				_, err = client.Navigate("", url)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error navigating: %v\n", err)
					os.Exit(1)
				}

				doWaitOpen()

				// Wait for element to be actionable (Visible, Stable, ReceivesEvents, Enabled)
				fmt.Printf("Waiting for element to be actionable: %s\n", selector)
				opts := features.WaitOptions{Timeout: timeout}
				if err := features.WaitForClick(client, "", selector, opts); err != nil {
					fmt.Fprintf(os.Stderr, "Error: %v\n", err)
					os.Exit(1)
				}

				fmt.Printf("Clicking element: %s\n", selector)
				err = client.ClickElement("", selector)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error clicking: %v\n", err)
					os.Exit(1)
				}

				// TODO: Replace sleep with proper navigation wait (poll URL change or listen for BiDi events)
				fmt.Println("Waiting for navigation...")
				time.Sleep(1 * time.Second)

				// Get current URL after click
				currentURL, err := client.GetCurrentURL()
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error getting URL: %v\n", err)
					os.Exit(1)
				}

				fmt.Printf("Click complete! Current URL: %s\n", currentURL)
			})
		},
	}
	clickCmd.Flags().Duration("timeout", features.DefaultTimeout, "Timeout for actionability checks (e.g., 5s, 30s)")
	rootCmd.AddCommand(clickCmd)

	typeCmd := &cobra.Command{
		Use:   "type [url] [selector] [text]",
		Short: "Navigate to a URL, click an element, and type text (with actionability checks)",
		Example: `  clicker type https://the-internet.herokuapp.com/inputs "input" "12345"
  # Waits for element to be visible, stable, receive events, enabled, and editable
  # Then types "12345" into the input

  clicker type https://the-internet.herokuapp.com/inputs "input" "12345" --timeout 5s
  # Custom timeout for actionability checks`,
		Args: cobra.ExactArgs(3),
		Run: func(cmd *cobra.Command, args []string) {
			process.WithCleanup(func() {
				url := args[0]
				selector := args[1]
				text := args[2]
				timeout, _ := cmd.Flags().GetDuration("timeout")

				fmt.Println("Launching browser...")
				launchResult, err := browser.Launch(browser.LaunchOptions{Headless: headless})
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error launching browser: %v\n", err)
					os.Exit(1)
				}
				defer waitAndClose(launchResult)

				fmt.Println("Connecting to BiDi...")
				conn, err := bidi.Connect(launchResult.WebSocketURL)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error connecting: %v\n", err)
					os.Exit(1)
				}
				defer conn.Close()

				client := bidi.NewClient(conn)

				fmt.Printf("Navigating to %s...\n", url)
				_, err = client.Navigate("", url)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error navigating: %v\n", err)
					os.Exit(1)
				}

				doWaitOpen()

				// Wait for element to be actionable (Visible, Stable, ReceivesEvents, Enabled, Editable)
				fmt.Printf("Waiting for element to be actionable: %s\n", selector)
				opts := features.WaitOptions{Timeout: timeout}
				if err := features.WaitForType(client, "", selector, opts); err != nil {
					fmt.Fprintf(os.Stderr, "Error: %v\n", err)
					os.Exit(1)
				}

				fmt.Printf("Typing into element: %s\n", selector)
				err = client.TypeIntoElement("", selector, text)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error typing: %v\n", err)
					os.Exit(1)
				}

				// Get the resulting value
				value, err := client.GetElementValue("", selector)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error getting value: %v\n", err)
					os.Exit(1)
				}

				fmt.Printf("Typed \"%s\", value is now: %s\n", text, value)
			})
		},
	}
	typeCmd.Flags().Duration("timeout", features.DefaultTimeout, "Timeout for actionability checks (e.g., 5s, 30s)")
	rootCmd.AddCommand(typeCmd)

	rootCmd.AddCommand(&cobra.Command{
		Use:   "check-actionable [url] [selector]",
		Short: "Check actionability of an element (Visible, Stable, ReceivesEvents, Enabled, Editable)",
		Example: `  clicker check-actionable https://example.com "a"
  # Output:
  # Checking actionability for selector: a
  # ✓ Visible: true
  # ✓ Stable: true
  # ✓ ReceivesEvents: true
  # ✓ Enabled: true
  # ✗ Editable: false`,
		Args: cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			process.WithCleanup(func() {
				url := args[0]
				selector := args[1]

				fmt.Println("Launching browser...")
				launchResult, err := browser.Launch(browser.LaunchOptions{Headless: headless})
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error launching browser: %v\n", err)
					os.Exit(1)
				}
				defer waitAndClose(launchResult)

				fmt.Println("Connecting to BiDi...")
				conn, err := bidi.Connect(launchResult.WebSocketURL)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error connecting: %v\n", err)
					os.Exit(1)
				}
				defer conn.Close()

				client := bidi.NewClient(conn)

				fmt.Printf("Navigating to %s...\n", url)
				_, err = client.Navigate("", url)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error navigating: %v\n", err)
					os.Exit(1)
				}

				doWaitOpen()

				fmt.Printf("\nChecking actionability for selector: %s\n", selector)

				result, err := features.CheckAll(client, "", selector)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error: %v\n", err)
					os.Exit(1)
				}

				// Print results with checkmarks
				printCheck("Visible", result.Visible)
				printCheck("Stable", result.Stable)
				printCheck("ReceivesEvents", result.ReceivesEvents)
				printCheck("Enabled", result.Enabled)
				printCheck("Editable", result.Editable)
			})
		},
	})

	serveCmd := &cobra.Command{
		Use:   "serve",
		Short: "Start WebSocket proxy server for browser automation",
		Example: `  clicker serve
  # Starts server on default port 9515, visible browser

  clicker serve --port 8080
  # Starts server on port 8080

  clicker serve --headless
  # Starts server with headless browser`,
		Run: func(cmd *cobra.Command, args []string) {
			process.WithCleanup(func() {
				port, _ := cmd.Flags().GetInt("port")

				fmt.Printf("Starting Clicker proxy server on port %d...\n", port)

				// Create router to manage browser sessions
				router := proxy.NewRouter(headless)

				server := proxy.NewServer(
					proxy.WithPort(port),
					proxy.WithOnConnect(router.OnClientConnect),
					proxy.WithOnMessage(router.OnClientMessage),
					proxy.WithOnClose(router.OnClientDisconnect),
				)

				if err := server.Start(); err != nil {
					fmt.Fprintf(os.Stderr, "Error starting server: %v\n", err)
					os.Exit(1)
				}

				fmt.Printf("Server listening on ws://localhost:%d\n", port)
				fmt.Println("Press Ctrl+C to stop...")

				// Wait for signal
				process.WaitForSignal()

				fmt.Println("\nShutting down...")

				// Close all browser sessions
				router.CloseAll()

				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				server.Stop(ctx)
			})
		},
	}
	serveCmd.Flags().IntP("port", "p", 9515, "Port to listen on")
	rootCmd.AddCommand(serveCmd)

	mcpCmd := &cobra.Command{
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
  - browser_quit: Close the browser`,
		Example: `  # Run directly (for testing)
  clicker mcp

  # Configure in Claude Code
  claude mcp add vibium -- clicker mcp

  # Custom screenshot directory
  clicker mcp --screenshot-dir ./screenshots

  # Disable screenshot file saving (inline only)
  clicker mcp --screenshot-dir ""

  # Test with echo
  echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"capabilities":{}}}' | clicker mcp`,
		Run: func(cmd *cobra.Command, args []string) {
			process.WithCleanup(func() {
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
	mcpCmd.Flags().String("screenshot-dir", "", "Directory for saving screenshots (default: ~/Pictures/Vibium, use \"\" to disable)")
	rootCmd.AddCommand(mcpCmd)

	recordCmd := &cobra.Command{
		Use:   "record [url]",
		Short: "Navigate to a URL and record the browser session as a video",
		Long: `Record a browser session as a video file.

This command navigates to a URL, waits for a specified duration while
capturing screenshots, then encodes them into a video using FFmpeg.

Requires FFmpeg to be installed and available in PATH.`,
		Example: `  clicker record https://example.com -o recording.mp4
  # Records the page for 5 seconds (default)

  clicker record https://example.com -o recording.mp4 --duration 10
  # Records for 10 seconds

  clicker record https://example.com -o recording.webm --format webm --fps 15
  # Records as WebM at 15 FPS`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			process.WithCleanup(func() {
				url := args[0]
				output, _ := cmd.Flags().GetString("output")
				duration, _ := cmd.Flags().GetInt("duration")
				fps, _ := cmd.Flags().GetInt("fps")
				format, _ := cmd.Flags().GetString("format")

				// Check FFmpeg availability
				if !recording.IsFFmpegAvailable() {
					fmt.Fprintf(os.Stderr, "Error: FFmpeg is not installed or not in PATH\n")
					fmt.Fprintf(os.Stderr, "Please install FFmpeg: https://ffmpeg.org/download.html\n")
					os.Exit(1)
				}

				fmt.Println("Launching browser...")
				launchResult, err := browser.Launch(browser.LaunchOptions{Headless: headless})
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error launching browser: %v\n", err)
					os.Exit(1)
				}
				defer waitAndClose(launchResult)

				fmt.Println("Connecting to BiDi...")
				conn, err := bidi.Connect(launchResult.WebSocketURL)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error connecting: %v\n", err)
					os.Exit(1)
				}
				defer conn.Close()

				client := bidi.NewClient(conn)

				fmt.Printf("Navigating to %s...\n", url)
				_, err = client.Navigate("", url)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error navigating: %v\n", err)
					os.Exit(1)
				}

				doWaitOpen()

				// Create screenshot function
				screenshotFn := func() (string, error) {
					// Simple fix: Focus + scroll to top before screenshot
					_, err := client.Evaluate("", `
						window.focus();
						window.scrollTo(0, 0);
						document.body.style.overflow = 'visible';
					`)
					if err != nil {
						return "", err
					}
					
					data, err := client.CaptureScreenshot("")
					if err != nil {
						return "", err
					}
					
					fmt.Printf("[screenshot] Captured %d bytes\n", len(data))
					return data, nil
				}



				// Create recorder
				recorder := recording.New(screenshotFn, recording.Options{
					FPS:        fps,
					Format:     format,
					OutputPath: output,
				})

				fmt.Printf("Starting recording (duration: %ds, fps: %d, format: %s)...\n", duration, fps, format)
				if err := recorder.Start(); err != nil {
					fmt.Fprintf(os.Stderr, "Error starting recording: %v\n", err)
					os.Exit(1)
				}

				// Record for specified duration
				time.Sleep(time.Duration(duration) * time.Second)

				fmt.Println("Stopping recording and encoding video...")
				outputPath, err := recorder.Stop()
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error stopping recording: %v\n", err)
					os.Exit(1)
				}

				fmt.Printf("Recording saved to: %s\n", outputPath)
			})
		},
	}
	recordCmd.Flags().StringP("output", "o", "recording.mp4", "Output file path")
	recordCmd.Flags().IntP("duration", "d", 5, "Recording duration in seconds")
	recordCmd.Flags().Int("fps", 10, "Frames per second")
	recordCmd.Flags().String("format", "mp4", "Output format: mp4 or webm")
	rootCmd.AddCommand(recordCmd)

	rootCmd.Version = version
	rootCmd.SetVersionTemplate("Clicker v{{.Version}}\n")

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

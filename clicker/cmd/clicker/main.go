package main

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/vibium/clicker/internal/bidi"
	"github.com/vibium/clicker/internal/browser"
	"github.com/vibium/clicker/internal/paths"
	"github.com/vibium/clicker/internal/process"
)

var version = "0.1.0"

func main() {
	rootCmd := &cobra.Command{
		Use:   "clicker",
		Short: "Browser automation for AI agents and humans",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
	}

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
			result, err := browser.Launch(browser.LaunchOptions{Headless: true})
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
			defer launchResult.Close()
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
			url := args[0]

			fmt.Println("Launching browser...")
			launchResult, err := browser.Launch(browser.LaunchOptions{Headless: true})
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error launching browser: %v\n", err)
				os.Exit(1)
			}
			defer launchResult.Close()

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
		},
	})

	screenshotCmd := &cobra.Command{
		Use:   "screenshot [url]",
		Short: "Navigate to a URL and capture a screenshot",
		Example: `  clicker screenshot https://example.com -o shot.png
  # Saves screenshot to shot.png`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			url := args[0]
			output, _ := cmd.Flags().GetString("output")

			fmt.Println("Launching browser...")
			launchResult, err := browser.Launch(browser.LaunchOptions{Headless: true})
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error launching browser: %v\n", err)
				os.Exit(1)
			}
			defer launchResult.Close()

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
			url := args[0]
			expression := args[1]

			fmt.Println("Launching browser...")
			launchResult, err := browser.Launch(browser.LaunchOptions{Headless: true})
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error launching browser: %v\n", err)
				os.Exit(1)
			}
			defer launchResult.Close()

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

			fmt.Printf("Evaluating: %s\n", expression)
			result, err := client.Evaluate("", expression)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error evaluating: %v\n", err)
				os.Exit(1)
			}

			fmt.Printf("Result: %v\n", result)
		},
	})

	rootCmd.AddCommand(&cobra.Command{
		Use:   "find [url] [selector]",
		Short: "Navigate to a URL and find an element by CSS selector",
		Example: `  clicker find https://example.com "a"
  # Prints: tag=A, text="Learn more", box={x,y,w,h}`,
		Args: cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			url := args[0]
			selector := args[1]

			fmt.Println("Launching browser...")
			launchResult, err := browser.Launch(browser.LaunchOptions{Headless: true})
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error launching browser: %v\n", err)
				os.Exit(1)
			}
			defer launchResult.Close()

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

			fmt.Printf("Finding element: %s\n", selector)
			info, err := client.FindElement("", selector)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error finding element: %v\n", err)
				os.Exit(1)
			}

			fmt.Printf("Found: tag=%s, text=\"%s\", box={x:%.0f, y:%.0f, w:%.0f, h:%.0f}\n",
				info.Tag, info.Text, info.Box.X, info.Box.Y, info.Box.Width, info.Box.Height)
		},
	})

	rootCmd.AddCommand(&cobra.Command{
		Use:   "click [url] [selector]",
		Short: "Navigate to a URL and click an element",
		Example: `  clicker click https://example.com "a"
  # Clicks the link and navigates to the target page`,
		Args: cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			url := args[0]
			selector := args[1]

			fmt.Println("Launching browser...")
			launchResult, err := browser.Launch(browser.LaunchOptions{Headless: true})
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error launching browser: %v\n", err)
				os.Exit(1)
			}
			defer launchResult.Close()

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
		},
	})

	rootCmd.Version = version
	rootCmd.SetVersionTemplate("Clicker v{{.Version}}\n")

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

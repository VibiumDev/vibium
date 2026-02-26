package main

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/vibium/clicker/internal/bidi"
	"github.com/vibium/clicker/internal/browser"
	"github.com/vibium/clicker/internal/features"
	"github.com/vibium/clicker/internal/process"
)

func newClickCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "click [url] [selector]",
		Short: "Click an element (optionally navigate to URL first)",
		Example: `  vibium click "a"
  # Clicks on current page (daemon mode)

  vibium click https://example.com "a"
  # Navigates to URL first, then clicks

  vibium click https://example.com "a" --timeout 5s
  # Custom timeout for actionability checks`,
		Args: cobra.RangeArgs(1, 2),
		Run: func(cmd *cobra.Command, args []string) {
			// Daemon mode
			if !oneshot {
				var selector string
				if len(args) == 2 {
					// click <url> <selector> — navigate first
					_, err := daemonCall("browser_navigate", map[string]interface{}{"url": args[0]})
					if err != nil {
						printError(err)
						return
					}
					selector = args[1]
				} else {
					// click <selector> — current page
					selector = args[0]
				}

				// Click element
				result, err := daemonCall("browser_click", map[string]interface{}{"selector": selector})
				if err != nil {
					printError(err)
					return
				}
				printResult(result)
				return
			}

			// Oneshot mode (original behavior) — requires URL + selector
			if len(args) < 2 {
				fatalExit("Error: requires [url] [selector] in oneshot mode")
			}
			url := args[0]
			selector := args[1]
			process.WithCleanup(func() {
				timeout, _ := cmd.Flags().GetDuration("timeout")

				fmt.Println("Launching browser...")
				launchResult, err := browser.Launch(browser.LaunchOptions{Headless: headless})
				if err != nil {
					fatalExit("Error launching browser: %v", err)
				}
				defer waitAndClose(launchResult)

				fmt.Println("Connecting to BiDi...")
				conn, err := bidi.Connect(launchResult.WebSocketURL)
				if err != nil {
					fatalExit("Error connecting: %v", err)
				}
				defer conn.Close()

				client := bidi.NewClient(conn)

				fmt.Printf("Navigating to %s...\n", url)
				_, err = client.Navigate("", url)
				if err != nil {
					fatalExit("Error navigating: %v", err)
				}

				doWaitOpen()

				// Wait for element to be actionable (Visible, Stable, ReceivesEvents, Enabled)
				fmt.Printf("Waiting for element to be actionable: %s\n", selector)
				opts := features.WaitOptions{Timeout: timeout}
				if err := features.WaitForClick(client, "", selector, opts); err != nil {
					fatalExit("Error: %v", err)
				}

				// Get URL before click so we can detect navigation
				urlBefore, _ := client.GetCurrentURL()

				fmt.Printf("Clicking element: %s\n", selector)
				err = client.ClickElement("", selector)
				if err != nil {
					fatalExit("Error clicking: %v", err)
				}

				// Poll for URL change to detect click-triggered navigation.
				// Returns immediately when URL changes, or after 2s if no navigation.
				// Note: daemon mode (PR #29) uses BiDi events for proper navigation
				// waiting. This polling approach is a stopgap for oneshot mode.
				fmt.Println("Waiting for navigation...")
				currentURL := urlBefore
				deadline := time.Now().Add(2 * time.Second)
				for time.Now().Before(deadline) {
					time.Sleep(100 * time.Millisecond)
					u, err := client.GetCurrentURL()
					if err != nil {
						break
					}
					currentURL = u
					if currentURL != urlBefore {
						break
					}
				}

				fmt.Printf("Click complete! Current URL: %s\n", currentURL)
			})
		},
	}
	cmd.Flags().Duration("timeout", features.DefaultTimeout, "Timeout for actionability checks (e.g., 5s, 30s)")
	return cmd
}

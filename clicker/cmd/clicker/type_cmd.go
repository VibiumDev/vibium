package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/vibium/clicker/internal/bidi"
	"github.com/vibium/clicker/internal/browser"
	"github.com/vibium/clicker/internal/features"
	"github.com/vibium/clicker/internal/process"
)

func newTypeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "type [url] [selector] [text]",
		Short: "Type text into an element (optionally navigate to URL first)",
		Example: `  vibium type "input" "12345"
  # Types on current page (daemon mode)

  vibium type https://the-internet.herokuapp.com/inputs "input" "12345"
  # Navigates to URL first, then types

  vibium type https://the-internet.herokuapp.com/inputs "input" "12345" --timeout 5s
  # Custom timeout for actionability checks`,
		Args: cobra.RangeArgs(2, 3),
		Run: func(cmd *cobra.Command, args []string) {
			// Daemon mode
			if !oneshot {
				var selector, text string
				if len(args) == 3 {
					// type <url> <selector> <text> — navigate first
					_, err := daemonCall("browser_navigate", map[string]interface{}{"url": args[0]})
					if err != nil {
						printError(err)
						return
					}
					selector = args[1]
					text = args[2]
				} else {
					// type <selector> <text> — current page
					selector = args[0]
					text = args[1]
				}

				// Type into element
				result, err := daemonCall("browser_type", map[string]interface{}{
					"selector": selector,
					"text":     text,
				})
				if err != nil {
					printError(err)
					return
				}
				printResult(result)
				return
			}

			// Oneshot mode (original behavior) — requires URL + selector + text
			if len(args) < 3 {
				fmt.Fprintf(os.Stderr, "Error: requires [url] [selector] [text] in oneshot mode\n")
				os.Exit(1)
			}
			url := args[0]
			selector := args[1]
			text := args[2]
			process.WithCleanup(func() {
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
	cmd.Flags().Duration("timeout", features.DefaultTimeout, "Timeout for actionability checks (e.g., 5s, 30s)")
	return cmd
}

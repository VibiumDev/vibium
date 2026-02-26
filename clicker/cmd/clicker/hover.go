package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/vibium/clicker/internal/bidi"
	"github.com/vibium/clicker/internal/browser"
	"github.com/vibium/clicker/internal/process"
)

func newHoverCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "hover [selector]",
		Short: "Hover over an element by CSS selector",
		Example: `  vibium hover "a"
  # Hover over first link (daemon mode)

  vibium hover https://example.com "a"
  # Navigate then hover (oneshot mode)`,
		Args: cobra.RangeArgs(1, 2),
		Run: func(cmd *cobra.Command, args []string) {
			// Daemon mode
			if !oneshot {
				var selector string
				if len(args) == 2 {
					_, err := daemonCall("browser_navigate", map[string]interface{}{"url": args[0]})
					if err != nil {
						printError(err)
						return
					}
					selector = args[1]
				} else {
					selector = args[0]
				}

				result, err := daemonCall("browser_hover", map[string]interface{}{"selector": selector})
				if err != nil {
					printError(err)
					return
				}
				printResult(result)
				return
			}

			// Oneshot mode
			if len(args) < 2 {
				fmt.Fprintf(os.Stderr, "Error: requires [url] [selector] in oneshot mode\n")
				os.Exit(1)
			}
			url := args[0]
			selector := args[1]
			process.WithCleanup(func() {
				launchResult, err := browser.Launch(browser.LaunchOptions{Headless: headless})
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error launching browser: %v\n", err)
					os.Exit(1)
				}
				defer waitAndClose(launchResult)

				conn, err := bidi.Connect(launchResult.WebSocketURL)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error connecting: %v\n", err)
					os.Exit(1)
				}
				defer conn.Close()

				client := bidi.NewClient(conn)

				_, err = client.Navigate("", url)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error navigating: %v\n", err)
					os.Exit(1)
				}

				doWaitOpen()

				info, err := client.FindElement("", selector)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error finding element: %v\n", err)
					os.Exit(1)
				}

				x, y := info.GetCenter()
				if err := client.MoveMouse("", x, y); err != nil {
					fmt.Fprintf(os.Stderr, "Error hovering: %v\n", err)
					os.Exit(1)
				}

				fmt.Printf("Hovered over element: %s\n", selector)
			})
		},
	}
}

package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/vibium/clicker/internal/bidi"
	"github.com/vibium/clicker/internal/browser"
	"github.com/vibium/clicker/internal/process"
)

func newTextCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "text [selector]",
		Short: "Get text content of the page or an element",
		Example: `  vibium text
  # Get all page text (daemon mode)

  vibium text "h1"
  # Get text of a specific element

  vibium text https://example.com
  # Navigate then get page text (oneshot mode)`,
		Args: cobra.MaximumNArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			// Daemon mode
			if !oneshot {
				toolArgs := map[string]interface{}{}
				if len(args) == 2 {
					// text <url> <selector> — navigate first
					_, err := daemonCall("browser_navigate", map[string]interface{}{"url": args[0]})
					if err != nil {
						printError(err)
						return
					}
					toolArgs["selector"] = args[1]
				} else if len(args) == 1 {
					// Could be selector or URL
					toolArgs["selector"] = args[0]
				}

				result, err := daemonCall("browser_get_text", toolArgs)
				if err != nil {
					printError(err)
					return
				}
				printResult(result)
				return
			}

			// Oneshot mode — requires URL
			if len(args) < 1 {
				fmt.Fprintf(os.Stderr, "Error: requires URL in oneshot mode\n")
				os.Exit(1)
			}
			url := args[0]
			var selector string
			if len(args) >= 2 {
				selector = args[1]
			}
			process.WithCleanup(func() {
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

				var expr string
				if selector != "" {
					expr = fmt.Sprintf(`document.querySelector(%q)?.innerText || ''`, selector)
				} else {
					expr = `document.body.innerText`
				}

				result, err := client.Evaluate("", expr)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error getting text: %v\n", err)
					os.Exit(1)
				}

				fmt.Printf("%v\n", result)
			})
		},
	}
}

package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/vibium/clicker/internal/bidi"
	"github.com/vibium/clicker/internal/browser"
	"github.com/vibium/clicker/internal/process"
)

func newHTMLCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "html [selector]",
		Short: "Get HTML content of the page or an element",
		Example: `  vibium html
  # Get full page HTML (daemon mode)

  vibium html "div.content"
  # Get innerHTML of a specific element

  vibium html "div.content" --outer
  # Get outerHTML of a specific element`,
		Args: cobra.MaximumNArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			outer, _ := cmd.Flags().GetBool("outer")

			// Daemon mode
			if !oneshot {
				toolArgs := map[string]interface{}{}
				if outer {
					toolArgs["outer"] = true
				}
				if len(args) == 2 {
					// html <url> <selector> — navigate first
					_, err := daemonCall("browser_navigate", map[string]interface{}{"url": args[0]})
					if err != nil {
						printError(err)
						return
					}
					toolArgs["selector"] = args[1]
				} else if len(args) == 1 {
					toolArgs["selector"] = args[0]
				}

				result, err := daemonCall("browser_get_html", toolArgs)
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

				var expr string
				if selector != "" {
					if outer {
						expr = fmt.Sprintf(`document.querySelector(%q)?.outerHTML || ''`, selector)
					} else {
						expr = fmt.Sprintf(`document.querySelector(%q)?.innerHTML || ''`, selector)
					}
				} else {
					expr = `document.documentElement.outerHTML`
				}

				result, err := client.Evaluate("", expr)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error getting HTML: %v\n", err)
					os.Exit(1)
				}

				fmt.Printf("%v\n", result)
			})
		},
	}
	cmd.Flags().Bool("outer", false, "Return outerHTML instead of innerHTML")
	return cmd
}

package main

import (
	"time"

	"github.com/spf13/cobra"
)

func newClickCmd() *cobra.Command {
	var timeout time.Duration
	cmd := &cobra.Command{
		Use:   "click [url] [selector]",
		Short: "Click an element (optionally navigate to URL first)",
		Example: `  vibium click "a"
  # Clicks on current page

  vibium click https://example.com "a"
  # Navigates to URL first, then clicks

  vibium click https://example.com "a" --timeout 5s
  # Custom timeout (5s, or 5000 for milliseconds)`,
		Args: cobra.RangeArgs(1, 2),
		Run: func(cmd *cobra.Command, args []string) {
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
			result, err := daemonCall("browser_click", map[string]interface{}{
				"selector": selector,
				"timeout":  float64(timeout.Milliseconds()),
			})
			if err != nil {
				printError(err)
				return
			}
			printResult(result)
		},
	}
	addTimeoutFlag(cmd, &timeout)
	return cmd
}

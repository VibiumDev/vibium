package main

import (
	"time"

	"github.com/spf13/cobra"
)

func newTypeCmd() *cobra.Command {
	var timeout time.Duration
	cmd := &cobra.Command{
		Use:   "type [url] [selector] [text]",
		Short: "Type text into an element (optionally navigate to URL first)",
		Example: `  vibium type "input" "12345"
  # Types on current page

  vibium type https://the-internet.herokuapp.com/inputs "input" "12345"
  # Navigates to URL first, then types

  vibium type https://the-internet.herokuapp.com/inputs "input" "12345" --timeout 5s
  # Custom timeout (5s, or 5000 for milliseconds)`,
		Args: cobra.RangeArgs(2, 3),
		Run: func(cmd *cobra.Command, args []string) {
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
	cmd.Flags().SetInterspersed(false)
	return cmd
}

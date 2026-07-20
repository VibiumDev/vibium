package main

import (
	"time"

	"github.com/spf13/cobra"
)

func newHoverCmd() *cobra.Command {
	var timeout time.Duration
	cmd := &cobra.Command{
		Use:   "hover [selector]",
		Short: "Hover over an element by CSS selector",
		Example: `  vibium hover "a"
  # Hover over first link

  vibium hover https://example.com "a"
  # Navigate then hover

  vibium hover "a" --timeout 5s
  # Custom timeout (5s, or 5000 for milliseconds)`,
		Args: cobra.RangeArgs(1, 2),
		Run: func(cmd *cobra.Command, args []string) {
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

			result, err := daemonCall("browser_hover", map[string]interface{}{
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

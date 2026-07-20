package main

import (
	"time"

	"github.com/spf13/cobra"
)

func newDblClickCmd() *cobra.Command {
	var timeout time.Duration
	cmd := &cobra.Command{
		Use:   "dblclick [selector]",
		Short: "Double-click an element",
		Example: `  vibium dblclick "td.cell"
  # Double-click to edit a table cell

  vibium dblclick @e2
  # Double-click element from map

  vibium dblclick "td.cell" --timeout 5s
  # Custom timeout (5s, or 5000 for milliseconds)`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			selector := args[0]

			result, err := daemonCall("browser_dblclick", map[string]interface{}{
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

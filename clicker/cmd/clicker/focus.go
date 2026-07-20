package main

import (
	"time"

	"github.com/spf13/cobra"
)

func newFocusCmd() *cobra.Command {
	var timeout time.Duration
	cmd := &cobra.Command{
		Use:   "focus [selector]",
		Short: "Focus an element",
		Example: `  vibium focus "input[name=email]"
  # Focus the email input

  vibium focus @e1
  # Focus element from map

  vibium focus "input[name=email]" --timeout 5s
  # Custom timeout (5s, or 5000 for milliseconds)`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			selector := args[0]

			result, err := daemonCall("browser_focus", map[string]interface{}{
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

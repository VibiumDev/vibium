package main

import (
	"time"

	"github.com/spf13/cobra"
)

func newUncheckCmd() *cobra.Command {
	var timeout time.Duration
	cmd := &cobra.Command{
		Use:   "uncheck [selector]",
		Short: "Uncheck a checkbox",
		Example: `  vibium uncheck "input[name=agree]"
  # Uncheck the "agree" checkbox (idempotent)

  vibium uncheck "input[name=agree]" --timeout 5s
  # Custom timeout (5s, or 5000 for milliseconds)`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			selector := args[0]

			result, err := daemonCall("browser_uncheck", map[string]interface{}{
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

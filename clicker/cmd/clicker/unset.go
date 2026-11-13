package main

import (
	"time"

	"github.com/spf13/cobra"
)

func newUnsetCmd() *cobra.Command {
	var timeout time.Duration
	cmd := &cobra.Command{
		Use:   "unset [selector]",
		Short: "Uncheck a checkbox",
		Example: `  vibium unset "input[name=agree]"
  # Uncheck the "agree" checkbox (idempotent)

  vibium unset "input[name=agree]" --timeout 5s
  # Custom timeout (5s, or 5000 for milliseconds)`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			selector := args[0]

			result, err := daemonCall("browser_unset", map[string]interface{}{
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

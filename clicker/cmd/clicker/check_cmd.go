package main

import (
	"time"

	"github.com/spf13/cobra"
)

func newCheckCmd() *cobra.Command {
	var timeout time.Duration
	cmd := &cobra.Command{
		Use:   "check [selector]",
		Short: "Check a checkbox or radio button",
		Example: `  vibium check "input[name=agree]"
  # Check the "agree" checkbox (idempotent)

  vibium check "input[name=agree]" --timeout 5s
  # Custom timeout (5s, or 5000 for milliseconds)`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			selector := args[0]

			result, err := daemonCall("browser_check", map[string]interface{}{
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

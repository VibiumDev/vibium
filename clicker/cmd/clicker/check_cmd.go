package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func newCheckCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "check [selector]",
		Short: "Check a checkbox or radio button",
		Example: `  vibium check "input[name=agree]"
  # Check the "agree" checkbox (idempotent)`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			selector := args[0]

			if !oneshot {
				result, err := daemonCall("browser_check", map[string]interface{}{"selector": selector})
				if err != nil {
					printError(err)
					return
				}
				printResult(result)
				return
			}

			fmt.Fprintf(os.Stderr, "Error: check command requires daemon mode\n")
			os.Exit(1)
		},
	}
}

package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func newUncheckCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "uncheck [selector]",
		Short: "Uncheck a checkbox",
		Example: `  vibium uncheck "input[name=agree]"
  # Uncheck the "agree" checkbox (idempotent)`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			selector := args[0]

			if !oneshot {
				result, err := daemonCall("browser_uncheck", map[string]interface{}{"selector": selector})
				if err != nil {
					printError(err)
					return
				}
				printResult(result)
				return
			}

			fmt.Fprintf(os.Stderr, "Error: uncheck command requires daemon mode\n")
			os.Exit(1)
		},
	}
}

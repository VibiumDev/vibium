package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func newIsEnabledCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "is-enabled [selector]",
		Short: "Check if an element is enabled",
		Example: `  vibium is-enabled "button[type=submit]"
  # Prints true or false`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			selector := args[0]

			if !oneshot {
				result, err := daemonCall("browser_is_enabled", map[string]interface{}{"selector": selector})
				if err != nil {
					printError(err)
					return
				}
				printResult(result)
				return
			}

			fmt.Fprintf(os.Stderr, "Error: is-enabled command requires daemon mode\n")
			os.Exit(1)
		},
	}
}

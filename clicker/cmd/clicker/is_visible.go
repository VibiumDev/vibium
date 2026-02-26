package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func newIsVisibleCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "is-visible [selector]",
		Short: "Check if an element is visible on the page",
		Example: `  vibium is-visible "h1"
  # Prints true or false`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			selector := args[0]

			if !oneshot {
				result, err := daemonCall("browser_is_visible", map[string]interface{}{"selector": selector})
				if err != nil {
					printError(err)
					return
				}
				printResult(result)
				return
			}

			fmt.Fprintf(os.Stderr, "Error: is-visible command requires daemon mode\n")
			os.Exit(1)
		},
	}
}

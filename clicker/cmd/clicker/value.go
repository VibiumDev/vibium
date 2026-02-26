package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func newValueCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "value [selector]",
		Short: "Get the current value of a form element",
		Example: `  vibium value "input[name=email]"
  # Print the current value of the email input`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			selector := args[0]

			if !oneshot {
				result, err := daemonCall("browser_get_value", map[string]interface{}{"selector": selector})
				if err != nil {
					printError(err)
					return
				}
				printResult(result)
				return
			}

			fmt.Fprintf(os.Stderr, "Error: value command requires daemon mode\n")
			os.Exit(1)
		},
	}
}

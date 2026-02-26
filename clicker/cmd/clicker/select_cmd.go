package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func newSelectCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "select [selector] [value]",
		Short: "Select an option in a <select> element",
		Example: `  vibium select "select#color" "blue"
  # Select "blue" in the color dropdown (daemon mode)`,
		Args: cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			selector := args[0]
			value := args[1]

			if !oneshot {
				result, err := daemonCall("browser_select", map[string]interface{}{
					"selector": selector,
					"value":    value,
				})
				if err != nil {
					printError(err)
					return
				}
				printResult(result)
				return
			}

			fmt.Fprintf(os.Stderr, "Error: select command requires daemon mode\n")
			os.Exit(1)
		},
	}
}

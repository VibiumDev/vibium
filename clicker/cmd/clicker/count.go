package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func newCountCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "count [selector]",
		Short: "Count matching elements",
		Example: `  vibium count "a"
  # Print number of links on the page

  vibium count "li.item"
  # Count list items`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			selector := args[0]

			if !oneshot {
				result, err := daemonCall("browser_count", map[string]interface{}{"selector": selector})
				if err != nil {
					printError(err)
					return
				}
				printResult(result)
				return
			}

			fmt.Fprintf(os.Stderr, "Error: count command requires daemon mode\n")
			os.Exit(1)
		},
	}
}

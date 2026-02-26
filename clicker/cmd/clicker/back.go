package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func newBackCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "back",
		Short: "Navigate back in browser history",
		Example: `  vibium back
  # Go back one page (like clicking the back button)`,
		Args: cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			if !oneshot {
				result, err := daemonCall("browser_back", map[string]interface{}{})
				if err != nil {
					printError(err)
					return
				}
				printResult(result)
				return
			}

			fmt.Fprintf(os.Stderr, "Error: back command requires daemon mode\n")
			os.Exit(1)
		},
	}
}

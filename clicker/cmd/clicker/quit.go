package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func newQuitCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "quit",
		Aliases: []string{"close"},
		Short:   "Close the browser session",
		Example: `  vibium quit
  # Close the browser (daemon keeps running)`,
		Args: cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			if !oneshot {
				result, err := daemonCall("browser_quit", map[string]interface{}{})
				if err != nil {
					printError(err)
					return
				}
				printResult(result)
				return
			}

			fmt.Fprintf(os.Stderr, "Error: quit command requires daemon mode\n")
			os.Exit(1)
		},
	}
}

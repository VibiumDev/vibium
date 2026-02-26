package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func newReloadCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "reload",
		Short: "Reload the current page",
		Example: `  vibium reload
  # Reload the current page`,
		Args: cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			if !oneshot {
				result, err := daemonCall("browser_reload", map[string]interface{}{})
				if err != nil {
					printError(err)
					return
				}
				printResult(result)
				return
			}

			fmt.Fprintf(os.Stderr, "Error: reload command requires daemon mode\n")
			os.Exit(1)
		},
	}
}

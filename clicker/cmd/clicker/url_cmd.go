package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func newURLCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "url",
		Short: "Get the current page URL",
		Example: `  vibium url
  # Prints: https://example.com`,
		Args: cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			if !oneshot {
				result, err := daemonCall("browser_get_url", map[string]interface{}{})
				if err != nil {
					printError(err)
					return
				}
				printResult(result)
				return
			}

			fmt.Fprintf(os.Stderr, "Error: url command requires daemon mode (no URL to navigate to)\n")
			os.Exit(1)
		},
	}
}

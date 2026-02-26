package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func newTabsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "tabs",
		Short: "List all open browser tabs",
		Example: `  vibium tabs
  # [0] https://example.com
  # [1] https://google.com`,
		Args: cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			if !oneshot {
				result, err := daemonCall("browser_list_tabs", map[string]interface{}{})
				if err != nil {
					printError(err)
					return
				}
				printResult(result)
				return
			}

			fmt.Fprintf(os.Stderr, "Error: tabs command requires daemon mode\n")
			os.Exit(1)
		},
	}
}

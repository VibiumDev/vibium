package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func newTabNewCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "tab-new [url]",
		Short: "Open a new browser tab",
		Example: `  vibium tab-new
  # Open a blank new tab

  vibium tab-new https://example.com
  # Open a new tab and navigate to URL`,
		Args: cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			if !oneshot {
				toolArgs := map[string]interface{}{}
				if len(args) == 1 {
					toolArgs["url"] = args[0]
				}

				result, err := daemonCall("browser_new_tab", toolArgs)
				if err != nil {
					printError(err)
					return
				}
				printResult(result)
				return
			}

			fmt.Fprintf(os.Stderr, "Error: tab-new command requires daemon mode\n")
			os.Exit(1)
		},
	}
}

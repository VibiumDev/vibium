package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/spf13/cobra"
)

func newTabSwitchCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "tab-switch [index or url]",
		Short: "Switch to a browser tab by index or URL substring",
		Example: `  vibium tab-switch 1
  # Switch to tab at index 1

  vibium tab-switch google.com
  # Switch to tab containing "google.com" in URL`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			if !oneshot {
				toolArgs := map[string]interface{}{}

				// Try to parse as integer index
				if idx, err := strconv.Atoi(args[0]); err == nil {
					toolArgs["index"] = float64(idx)
				} else {
					toolArgs["url"] = args[0]
				}

				result, err := daemonCall("browser_switch_tab", toolArgs)
				if err != nil {
					printError(err)
					return
				}
				printResult(result)
				return
			}

			fmt.Fprintf(os.Stderr, "Error: tab-switch command requires daemon mode\n")
			os.Exit(1)
		},
	}
}

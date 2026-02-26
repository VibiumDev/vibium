package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/spf13/cobra"
)

func newTabCloseCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "tab-close [index]",
		Short: "Close a browser tab by index (default: current tab)",
		Example: `  vibium tab-close
  # Close current tab (index 0)

  vibium tab-close 1
  # Close tab at index 1`,
		Args: cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			if !oneshot {
				toolArgs := map[string]interface{}{}
				if len(args) == 1 {
					idx, err := strconv.Atoi(args[0])
					if err != nil {
						fmt.Fprintf(os.Stderr, "Error: invalid tab index: %s\n", args[0])
						os.Exit(1)
					}
					toolArgs["index"] = float64(idx)
				}

				result, err := daemonCall("browser_close_tab", toolArgs)
				if err != nil {
					printError(err)
					return
				}
				printResult(result)
				return
			}

			fmt.Fprintf(os.Stderr, "Error: tab-close command requires daemon mode\n")
			os.Exit(1)
		},
	}
}

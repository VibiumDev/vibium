package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func newWaitForLoadCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "wait-for-load",
		Short: "Wait until the page is fully loaded",
		Example: `  vibium wait-for-load
  # Wait until document.readyState is "complete"

  vibium wait-for-load --timeout 10000
  # Wait up to 10 seconds`,
		Args: cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			timeout, _ := cmd.Flags().GetInt("timeout")

			if !oneshot {
				toolArgs := map[string]interface{}{}
				if cmd.Flags().Changed("timeout") {
					toolArgs["timeout"] = float64(timeout)
				}

				result, err := daemonCall("browser_wait_for_load", toolArgs)
				if err != nil {
					printError(err)
					return
				}
				printResult(result)
				return
			}

			fmt.Fprintf(os.Stderr, "Error: wait-for-load command requires daemon mode\n")
			os.Exit(1)
		},
	}
	cmd.Flags().Int("timeout", 30000, "Timeout in milliseconds")
	return cmd
}

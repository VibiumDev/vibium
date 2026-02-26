package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func newWaitForFnCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "wait-for-fn [expression]",
		Short: "Wait until a JS expression returns truthy",
		Example: `  vibium wait-for-fn "document.readyState === 'complete'"
  # Wait for page to be fully loaded

  vibium wait-for-fn "window.ready === true" --timeout 10000
  # Wait for custom condition with timeout`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			expression := args[0]
			timeout, _ := cmd.Flags().GetFloat64("timeout")

			if !oneshot {
				callArgs := map[string]interface{}{"expression": expression}
				if timeout > 0 {
					callArgs["timeout"] = timeout
				}
				result, err := daemonCall("browser_wait_for_fn", callArgs)
				if err != nil {
					printError(err)
					return
				}
				printResult(result)
				return
			}

			fmt.Fprintf(os.Stderr, "Error: wait-for-fn command requires daemon mode\n")
			os.Exit(1)
		},
	}
	cmd.Flags().Float64("timeout", 30000, "Timeout in milliseconds")
	return cmd
}

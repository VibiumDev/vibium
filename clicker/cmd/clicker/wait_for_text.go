package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func newWaitForTextCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "wait-for-text [text]",
		Short: "Wait until text appears on the page",
		Example: `  vibium wait-for-text "Welcome"
  # Waits until "Welcome" appears on the page

  vibium wait-for-text "Success" --timeout 10000
  # Wait with custom timeout (10 seconds)`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			text := args[0]
			timeout, _ := cmd.Flags().GetFloat64("timeout")

			if !oneshot {
				callArgs := map[string]interface{}{"text": text}
				if timeout > 0 {
					callArgs["timeout"] = timeout
				}
				result, err := daemonCall("browser_wait_for_text", callArgs)
				if err != nil {
					printError(err)
					return
				}
				printResult(result)
				return
			}

			fmt.Fprintf(os.Stderr, "Error: wait-for-text command requires daemon mode\n")
			os.Exit(1)
		},
	}
	cmd.Flags().Float64("timeout", 30000, "Timeout in milliseconds")
	return cmd
}

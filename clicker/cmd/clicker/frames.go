package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func newFramesCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "frames",
		Short: "List all child frames (iframes) on the page",
		Example: `  vibium frames
  # [{"context":"abc","url":"https://example.com/frame","name":"myFrame"}]`,
		Args: cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			if !oneshot {
				result, err := daemonCall("browser_frames", map[string]interface{}{})
				if err != nil {
					printError(err)
					return
				}
				printResult(result)
				return
			}

			fmt.Fprintf(os.Stderr, "Error: frames command requires daemon mode\n")
			os.Exit(1)
		},
	}
}

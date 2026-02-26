package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func newScrollIntoViewCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "scroll-into-view [selector]",
		Short: "Scroll an element into view",
		Example: `  vibium scroll-into-view "#footer"
  # Scroll the footer element into view (centered on screen)`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			selector := args[0]

			if !oneshot {
				result, err := daemonCall("browser_scroll_into_view", map[string]interface{}{"selector": selector})
				if err != nil {
					printError(err)
					return
				}
				printResult(result)
				return
			}

			fmt.Fprintf(os.Stderr, "Error: scroll-into-view command requires daemon mode\n")
			os.Exit(1)
		},
	}
}

package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func newHighlightCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "highlight [selector]",
		Short: "Highlight an element with a red outline for 3 seconds",
		Example: `  vibium highlight "h1"
  # Highlights the first h1 element

  vibium highlight @e1
  # Highlights the element from map`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			selector := args[0]

			if !oneshot {
				result, err := daemonCall("browser_highlight", map[string]interface{}{"selector": selector})
				if err != nil {
					printError(err)
					return
				}
				printResult(result)
				return
			}

			fmt.Fprintf(os.Stderr, "Error: highlight command requires daemon mode\n")
			os.Exit(1)
		},
	}
}

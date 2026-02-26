package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func newFocusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "focus [selector]",
		Short: "Focus an element",
		Example: `  vibium focus "input[name=email]"
  # Focus the email input

  vibium focus @e1
  # Focus element from map`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			selector := args[0]

			if !oneshot {
				result, err := daemonCall("browser_focus", map[string]interface{}{"selector": selector})
				if err != nil {
					printError(err)
					return
				}
				printResult(result)
				return
			}

			fmt.Fprintf(os.Stderr, "Error: focus command requires daemon mode\n")
			os.Exit(1)
		},
	}
}

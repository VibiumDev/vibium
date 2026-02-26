package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func newPressCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "press [key] [selector]",
		Short: "Press a key on a specific element or the focused element",
		Example: `  vibium press Enter
  # Press Enter on the currently focused element

  vibium press Enter "input[name=search]"
  # Click to focus the input, then press Enter

  vibium press "Control+a"
  # Select all`,
		Args: cobra.RangeArgs(1, 2),
		Run: func(cmd *cobra.Command, args []string) {
			key := args[0]

			if !oneshot {
				toolArgs := map[string]interface{}{"key": key}
				if len(args) == 2 {
					toolArgs["selector"] = args[1]
				}

				result, err := daemonCall("browser_press", toolArgs)
				if err != nil {
					printError(err)
					return
				}
				printResult(result)
				return
			}

			fmt.Fprintf(os.Stderr, "Error: press command requires daemon mode\n")
			os.Exit(1)
		},
	}
}

package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func newDragCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "drag [source] [target]",
		Short: "Drag from one element to another",
		Example: `  vibium drag ".draggable" ".dropzone"
  # Drag element to drop target

  vibium drag @e1 @e3
  # Drag using map refs`,
		Args: cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			source := args[0]
			target := args[1]

			if !oneshot {
				result, err := daemonCall("browser_drag", map[string]interface{}{
					"source": source,
					"target": target,
				})
				if err != nil {
					printError(err)
					return
				}
				printResult(result)
				return
			}

			fmt.Fprintf(os.Stderr, "Error: drag command requires daemon mode\n")
			os.Exit(1)
		},
	}
}

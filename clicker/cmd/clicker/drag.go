package main

import (
	"time"

	"github.com/spf13/cobra"
)

func newDragCmd() *cobra.Command {
	var timeout time.Duration
	cmd := &cobra.Command{
		Use:   "drag [source] [target]",
		Short: "Drag from one element to another",
		Example: `  vibium drag ".draggable" ".dropzone"
  # Drag element to drop target

  vibium drag @e1 @e3
  # Drag using map refs

  vibium drag ".draggable" ".dropzone" --timeout 5s
  # Custom timeout (5s, or 5000 for milliseconds)`,
		Args: cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			source := args[0]
			target := args[1]

			result, err := daemonCall("browser_drag", map[string]interface{}{
				"source":  source,
				"target":  target,
				"timeout": float64(timeout.Milliseconds()),
			})
			if err != nil {
				printError(err)
				return
			}
			printResult(result)
		},
	}
	addTimeoutFlag(cmd, &timeout)
	return cmd
}

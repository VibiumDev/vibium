package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/spf13/cobra"
)

func newMouseMoveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "mouse-move [x] [y]",
		Short: "Move the mouse to coordinates",
		Example: `  vibium mouse-move 100 200
  # Move mouse to position (100, 200)`,
		Args: cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			x, err := strconv.ParseFloat(args[0], 64)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: invalid x coordinate: %s\n", args[0])
				os.Exit(1)
			}
			y, err := strconv.ParseFloat(args[1], 64)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: invalid y coordinate: %s\n", args[1])
				os.Exit(1)
			}

			if !oneshot {
				result, err := daemonCall("browser_mouse_move", map[string]interface{}{"x": x, "y": y})
				if err != nil {
					printError(err)
					return
				}
				printResult(result)
				return
			}

			fmt.Fprintf(os.Stderr, "Error: mouse-move command requires daemon mode\n")
			os.Exit(1)
		},
	}
}

package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/spf13/cobra"
)

func newWindowCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "window [width] [height] [x] [y]",
		Short: "Get or set the OS browser window size, position, or state",
		Example: `  vibium window
  # {"state":"normal","x":0,"y":25,"width":1280,"height":720}

  vibium window 1920 1080
  # Set window to 1920x1080

  vibium window 1920 1080 0 0
  # Set window to 1920x1080 at position (0, 0)

  vibium window 1920 1080 -1920 25
  # Move to a monitor left of the primary display

  vibium window --state maximized
  # Maximize the window`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) > 4 {
				fmt.Fprintf(os.Stderr, "Error: accepts between 0 and 4 arg(s), received %d\n", len(args))
				os.Exit(1)
			}
			state, _ := cmd.Flags().GetString("state")

			if len(args) == 0 && state == "" {
				// Get window
				result, err := daemonCall("browser_get_window", map[string]interface{}{})
				if err != nil {
					printError(err)
					return
				}
				printResult(result)
				return
			}

			if len(args) == 1 || len(args) == 3 {
				fmt.Fprintf(os.Stderr, "Error: provide both width and height\n")
				os.Exit(1)
			}

			callArgs := map[string]interface{}{}

			if len(args) >= 2 {
				width, err := strconv.Atoi(args[0])
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error: invalid width: %s\n", args[0])
					os.Exit(1)
				}
				height, err := strconv.Atoi(args[1])
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error: invalid height: %s\n", args[1])
					os.Exit(1)
				}
				callArgs["width"] = float64(width)
				callArgs["height"] = float64(height)
			}

			if len(args) == 4 {
				x, err := strconv.Atoi(args[2])
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error: invalid x: %s\n", args[2])
					os.Exit(1)
				}
				y, err := strconv.Atoi(args[3])
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error: invalid y: %s\n", args[3])
					os.Exit(1)
				}
				callArgs["x"] = float64(x)
				callArgs["y"] = float64(y)
			}

			if state != "" {
				callArgs["state"] = state
			}

			result, err := daemonCall("browser_set_window", callArgs)
			if err != nil {
				printError(err)
				return
			}
			printResult(result)
		},
	}
	cmd.Flags().String("state", "", "Window state: normal, maximized, minimized, fullscreen")
	// A window x/y left or above the primary display is negative:
	// `window 1920 1080 -1920 25`.
	return lateParse(cmd)
}

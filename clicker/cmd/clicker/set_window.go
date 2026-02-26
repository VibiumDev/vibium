package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/spf13/cobra"
)

func newSetWindowCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set-window [width] [height] [x] [y]",
		Short: "Set the OS browser window size, position, or state",
		Example: `  vibium set-window 1920 1080
  # Set window to 1920x1080

  vibium set-window 1920 1080 0 0
  # Set window to 1920x1080 at position (0, 0)

  vibium set-window --state maximized
  # Maximize the window`,
		Args: cobra.RangeArgs(0, 4),
		Run: func(cmd *cobra.Command, args []string) {
			state, _ := cmd.Flags().GetString("state")

			if len(args) == 0 && state == "" {
				fmt.Fprintf(os.Stderr, "Error: provide width and height, or use --state\n")
				os.Exit(1)
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

			if !oneshot {
				result, err := daemonCall("browser_set_window", callArgs)
				if err != nil {
					printError(err)
					return
				}
				printResult(result)
				return
			}

			fmt.Fprintf(os.Stderr, "Error: set-window command requires daemon mode\n")
			os.Exit(1)
		},
	}
	cmd.Flags().String("state", "", "Window state: normal, maximized, minimized, fullscreen")
	return cmd
}

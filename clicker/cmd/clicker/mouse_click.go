package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/spf13/cobra"
)

func newMouseClickCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mouse-click [x] [y]",
		Short: "Click at coordinates or current position",
		Example: `  vibium mouse-click 100 200
  # Left click at (100, 200)

  vibium mouse-click 100 200 --button 2
  # Right click at (100, 200)

  vibium mouse-click
  # Left click at current position

  vibium mouse-click --button 1
  # Middle click at current position`,
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) != 0 && len(args) != 2 {
				return fmt.Errorf("accepts 0 or 2 arg(s), received %d", len(args))
			}
			return nil
		},
		Run: func(cmd *cobra.Command, args []string) {
			button, _ := cmd.Flags().GetInt("button")

			params := map[string]interface{}{
				"button": float64(button),
			}

			if len(args) == 2 {
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
				params["x"] = x
				params["y"] = y
			}

			if !oneshot {
				result, err := daemonCall("browser_mouse_click", params)
				if err != nil {
					printError(err)
					return
				}
				printResult(result)
				return
			}

			fmt.Fprintf(os.Stderr, "Error: mouse-click command requires daemon mode\n")
			os.Exit(1)
		},
	}
	cmd.Flags().Int("button", 0, "Mouse button (0=left, 1=middle, 2=right)")
	return cmd
}

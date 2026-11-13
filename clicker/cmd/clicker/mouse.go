package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/spf13/cobra"
)

func newMouseCmd() *cobra.Command {
	mouseCmd := &cobra.Command{
		Use:   "mouse",
		Short: "Mouse control (click, move, down, up)",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
	}

	clickCmd := &cobra.Command{
		Use:   "click [x] [y]",
		Short: "Click at coordinates or current position",
		Example: `  vibium mouse click 100 200
  # Left click at (100, 200)

  vibium mouse click 100 200 --button 2
  # Right click at (100, 200)

  vibium mouse click
  # Left click at current position`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 0 && len(args) != 2 {
				fmt.Fprintf(os.Stderr, "Error: accepts 0 or 2 arg(s), received %d\n", len(args))
				os.Exit(1)
			}
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

			result, err := daemonCall("browser_mouse_click", params)
			if err != nil {
				printError(err)
				return
			}
			printResult(result)
		},
	}
	clickCmd.Flags().Int("button", 0, "Mouse button (0=left, 1=middle, 2=right)")

	moveCmd := &cobra.Command{
		Use:   "move [x] [y]",
		Short: "Move the mouse to coordinates",
		Example: `  vibium mouse move 100 200
  # Move mouse to position (100, 200)`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 2 {
				fmt.Fprintf(os.Stderr, "Error: accepts 2 arg(s), received %d\n", len(args))
				os.Exit(1)
			}
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

			result, err := daemonCall("browser_mouse_move", map[string]interface{}{"x": x, "y": y})
			if err != nil {
				printError(err)
				return
			}
			printResult(result)
		},
	}

	downCmd := &cobra.Command{
		Use:   "down",
		Short: "Press a mouse button down",
		Example: `  vibium mouse down
  # Press left mouse button

  vibium mouse down --button 2
  # Press right mouse button`,
		Args: cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			button, _ := cmd.Flags().GetInt("button")

			result, err := daemonCall("browser_mouse_down", map[string]interface{}{"button": float64(button)})
			if err != nil {
				printError(err)
				return
			}
			printResult(result)
		},
	}
	downCmd.Flags().Int("button", 0, "Mouse button (0=left, 1=middle, 2=right)")

	upCmd := &cobra.Command{
		Use:   "up",
		Short: "Release a mouse button",
		Example: `  vibium mouse up
  # Release left mouse button

  vibium mouse up --button 2
  # Release right mouse button`,
		Args: cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			button, _ := cmd.Flags().GetInt("button")

			result, err := daemonCall("browser_mouse_up", map[string]interface{}{"button": float64(button)})
			if err != nil {
				printError(err)
				return
			}
			printResult(result)
		},
	}
	upCmd.Flags().Int("button", 0, "Mouse button (0=left, 1=middle, 2=right)")

	// Coordinates are numeric positionals, so a negative one would otherwise
	// be read as an unknown flag: `mouse move 100 -50`.
	mouseCmd.AddCommand(lateParse(clickCmd))
	mouseCmd.AddCommand(lateParse(moveCmd))
	mouseCmd.AddCommand(downCmd)
	mouseCmd.AddCommand(upCmd)
	return mouseCmd
}

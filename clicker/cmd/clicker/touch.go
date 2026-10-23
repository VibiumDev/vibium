package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/spf13/cobra"
)

// parseCoord parses a coordinate positional, exiting with a clear error if it
// is not a number. Touch gestures take several coordinates, so this keeps the
// commands below readable.
func parseCoord(val, name string) float64 {
	f, err := strconv.ParseFloat(val, 64)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: invalid %s coordinate: %s\n", name, val)
		os.Exit(1)
	}
	return f
}

func newTapCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tap [x] [y]",
		Short: "Tap a touchscreen at coordinates",
		Example: `  vibium tap 100 200
  # Tap at (100, 200)`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 2 {
				fmt.Fprintf(os.Stderr, "Error: accepts 2 arg(s), received %d\n", len(args))
				os.Exit(1)
			}
			params := map[string]interface{}{
				"x": parseCoord(args[0], "x"),
				"y": parseCoord(args[1], "y"),
			}
			result, err := daemonCall("browser_tap", params)
			if err != nil {
				printError(err)
				return
			}
			printResult(result)
		},
	}
	// Coordinates are numeric positionals, so a negative one would otherwise
	// be read as an unknown flag: `tap 100 -50`.
	return lateParse(cmd)
}

func newSwipeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "swipe [x1] [y1] [x2] [y2]",
		Short: "Swipe a touchscreen from one point to another",
		Example: `  vibium swipe 200 400 200 100
  # Swipe up from (200, 400) to (200, 100)

  vibium swipe 50 300 350 300 --duration 600
  # Slower swipe right, over 600ms`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 4 {
				fmt.Fprintf(os.Stderr, "Error: accepts 4 arg(s), received %d\n", len(args))
				os.Exit(1)
			}
			duration, _ := cmd.Flags().GetInt("duration")
			params := map[string]interface{}{
				"x1":       parseCoord(args[0], "x1"),
				"y1":       parseCoord(args[1], "y1"),
				"x2":       parseCoord(args[2], "x2"),
				"y2":       parseCoord(args[3], "y2"),
				"duration": float64(duration),
			}
			result, err := daemonCall("browser_swipe", params)
			if err != nil {
				printError(err)
				return
			}
			printResult(result)
		},
	}
	cmd.Flags().Int("duration", 300, "How long the move takes in milliseconds")
	return lateParse(cmd)
}

func newLongPressCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "long-press [x] [y]",
		Short: "Press and hold a touchscreen at coordinates",
		Example: `  vibium long-press 100 200
  # Hold a press at (100, 200) for 500ms

  vibium long-press 100 200 --duration 1200
  # Hold for 1200ms`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 2 {
				fmt.Fprintf(os.Stderr, "Error: accepts 2 arg(s), received %d\n", len(args))
				os.Exit(1)
			}
			duration, _ := cmd.Flags().GetInt("duration")
			params := map[string]interface{}{
				"x":        parseCoord(args[0], "x"),
				"y":        parseCoord(args[1], "y"),
				"duration": float64(duration),
			}
			result, err := daemonCall("browser_long_press", params)
			if err != nil {
				printError(err)
				return
			}
			printResult(result)
		},
	}
	cmd.Flags().Int("duration", 500, "How long to hold the press in milliseconds")
	return lateParse(cmd)
}

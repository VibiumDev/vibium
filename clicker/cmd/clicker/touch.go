package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

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
		Use:   "swipe <up|down|left|right> | swipe [x1] [y1] [x2] [y2]",
		Short: "Swipe a touchscreen in a direction or from one point to another",
		Example: `  vibium swipe up
  # Swipe up from the viewport center, half the viewport height

  vibium swipe left --distance 200
  # Swipe left 200px from the center

  vibium swipe up --from bottom --speed fast
  # Phone (Roadie): go home

  vibium swipe up --from bottom --speed slow --hold 500
  # Phone (Roadie): open the app switcher

  vibium swipe down --from top
  # Phone (Roadie): Notification Center, or the Android notification shade

  vibium swipe down --from 400,0
  # iPhone (Roadie): Control Center, pulled from near the top-right corner

  vibium swipe up --from 200,600 --duration 600
  # Swipe up from (200, 600) over 600ms

  vibium swipe 200 400 200 100
  # Swipe from (200, 400) to (200, 100)`,
		Run: func(cmd *cobra.Command, args []string) {
			params := map[string]interface{}{}
			switch len(args) {
			case 1:
				params["direction"] = args[0]
				from, _ := cmd.Flags().GetString("from")
				if x, y, ok := strings.Cut(from, ","); ok {
					params["fromX"] = parseCoord(x, "from x")
					params["fromY"] = parseCoord(y, "from y")
				} else if from != "" {
					params["from"] = from
				}
				if cmd.Flags().Changed("distance") {
					d, _ := cmd.Flags().GetInt("distance")
					params["distance"] = float64(d)
				}
			case 4:
				for i, k := range []string{"x1", "y1", "x2", "y2"} {
					params[k] = parseCoord(args[i], k)
				}
				if cmd.Flags().Changed("from") || cmd.Flags().Changed("distance") {
					fmt.Fprintln(os.Stderr, "Error: --from and --distance go with a direction, not with x1 y1 x2 y2")
					os.Exit(1)
				}
			default:
				fmt.Fprintf(os.Stderr, "Error: accepts a direction or 4 coordinates, received %d arg(s)\n", len(args))
				os.Exit(1)
			}
			if speed, _ := cmd.Flags().GetString("speed"); speed != "" {
				params["speed"] = speed
			}
			if cmd.Flags().Changed("duration") {
				d, _ := cmd.Flags().GetInt("duration")
				params["duration"] = float64(d)
			}
			if hold, _ := cmd.Flags().GetInt("hold"); hold > 0 {
				params["hold"] = float64(hold)
			}
			result, err := daemonCall("browser_swipe", params)
			if err != nil {
				printError(err)
				return
			}
			printResult(result)
		},
	}
	cmd.Flags().String("from", "", "Where a direction swipe starts: top, bottom, left, right, center, or x,y")
	cmd.Flags().Int("distance", 0, "How far a direction swipe moves in pixels (default: half the viewport)")
	cmd.Flags().String("speed", "", "slow, normal, fast, or pixels per second (instead of --duration)")
	cmd.Flags().Int("duration", 300, "How long the move takes in milliseconds")
	cmd.Flags().Int("hold", 0, "Milliseconds to rest at the end before lifting")
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

package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/spf13/cobra"
)

func newSetViewportCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set-viewport [width] [height]",
		Short: "Set the browser viewport size",
		Example: `  vibium set-viewport 1280 720
  # Set viewport to 1280x720

  vibium set-viewport 375 812 --dpr 3
  # Simulate iPhone X viewport`,
		Args: cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
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

			dpr, _ := cmd.Flags().GetFloat64("dpr")

			if !oneshot {
				callArgs := map[string]interface{}{
					"width":  float64(width),
					"height": float64(height),
				}
				if dpr > 0 {
					callArgs["devicePixelRatio"] = dpr
				}
				result, err := daemonCall("browser_set_viewport", callArgs)
				if err != nil {
					printError(err)
					return
				}
				printResult(result)
				return
			}

			fmt.Fprintf(os.Stderr, "Error: set-viewport command requires daemon mode\n")
			os.Exit(1)
		},
	}
	cmd.Flags().Float64("dpr", 0, "Device pixel ratio (e.g., 2 for Retina)")
	return cmd
}

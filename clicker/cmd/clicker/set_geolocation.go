package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/spf13/cobra"
)

func newSetGeolocationCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set-geolocation [latitude] [longitude]",
		Short: "Override the browser geolocation",
		Example: `  vibium set-geolocation 40.7128 -74.006
  # Set location to New York City

  vibium set-geolocation 51.5074 -0.1278 --accuracy 10
  # Set location to London with 10m accuracy`,
		Args: cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			lat, err := strconv.ParseFloat(args[0], 64)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: invalid latitude: %s\n", args[0])
				os.Exit(1)
			}
			lng, err := strconv.ParseFloat(args[1], 64)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: invalid longitude: %s\n", args[1])
				os.Exit(1)
			}

			accuracy, _ := cmd.Flags().GetFloat64("accuracy")

			if !oneshot {
				callArgs := map[string]interface{}{
					"latitude":  lat,
					"longitude": lng,
				}
				if accuracy > 0 {
					callArgs["accuracy"] = accuracy
				}
				result, err := daemonCall("browser_set_geolocation", callArgs)
				if err != nil {
					printError(err)
					return
				}
				printResult(result)
				return
			}

			fmt.Fprintf(os.Stderr, "Error: set-geolocation command requires daemon mode\n")
			os.Exit(1)
		},
	}
	cmd.Flags().Float64("accuracy", 0, "Accuracy in meters (default: 1)")
	return cmd
}

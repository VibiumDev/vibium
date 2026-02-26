package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/spf13/cobra"
)

func newSleepCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "sleep [ms]",
		Short: "Pause execution for a number of milliseconds",
		Example: `  vibium sleep 1000
  # Wait 1 second

  vibium sleep 500
  # Wait 500ms`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			ms, err := strconv.ParseFloat(args[0], 64)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: invalid milliseconds value: %s\n", args[0])
				os.Exit(1)
			}

			if !oneshot {
				result, err := daemonCall("browser_sleep", map[string]interface{}{"ms": ms})
				if err != nil {
					printError(err)
					return
				}
				printResult(result)
				return
			}

			fmt.Fprintf(os.Stderr, "Error: sleep command requires daemon mode\n")
			os.Exit(1)
		},
	}
}

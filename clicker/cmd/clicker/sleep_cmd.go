package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/spf13/cobra"
)

func newSleepCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sleep [ms]",
		Short: "Pause execution for a number of milliseconds",
		Example: `  vibium sleep 1000
  # Wait 1 second

  vibium sleep 500
  # Wait 500ms`,
		DisableFlagParsing: true,
		Args:               cobra.ArbitraryArgs,
		Run: func(cmd *cobra.Command, args []string) {
			args, perr := parseFlagsAllowNegative(cmd, args)
			if perr != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", perr)
				os.Exit(1)
			}
			if len(args) != 1 {
				fmt.Fprintf(os.Stderr, "Error: accepts 1 arg(s), received %d\n", len(args))
				os.Exit(1)
			}
			ms, err := strconv.ParseFloat(args[0], 64)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: invalid milliseconds value: %s\n", args[0])
				os.Exit(1)
			}

			result, err := daemonCall("browser_sleep", map[string]interface{}{"ms": ms})
			if err != nil {
				printError(err)
				return
			}
			printResult(result)
		},
	}
	return cmd
}

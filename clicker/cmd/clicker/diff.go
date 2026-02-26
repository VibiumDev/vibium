package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func newDiffCmd() *cobra.Command {
	diffCmd := &cobra.Command{
		Use:   "diff",
		Short: "Compare current state vs previous",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
	}

	diffMapCmd := &cobra.Command{
		Use:   "map",
		Short: "Compare current page elements vs last map",
		Example: `  vibium map           # take initial snapshot
  vibium click @e3     # interact with page
  vibium diff map      # see what changed`,
		Args: cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			if !oneshot {
				result, err := daemonCall("browser_diff_map", map[string]interface{}{})
				if err != nil {
					printError(err)
					return
				}
				printResult(result)
				return
			}

			fmt.Fprintf(os.Stderr, "Error: diff command requires daemon mode\n")
			os.Exit(1)
		},
	}

	diffCmd.AddCommand(diffMapCmd)
	return diffCmd
}

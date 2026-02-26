package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func newMapCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "map",
		Short: "Map interactive page elements with @refs",
		Example: `  vibium map
  # Lists interactive elements with refs like @e1, @e2
  # Use refs with other commands: vibium click @e1

  vibium map --selector "nav"
  # Only map elements inside the <nav> element`,
		Args: cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			if !oneshot {
				toolArgs := map[string]interface{}{}
				if sel, _ := cmd.Flags().GetString("selector"); sel != "" {
					toolArgs["selector"] = sel
				}
				result, err := daemonCall("browser_map", toolArgs)
				if err != nil {
					printError(err)
					return
				}
				printResult(result)
				return
			}

			fmt.Fprintf(os.Stderr, "Error: map command requires daemon mode\n")
			os.Exit(1)
		},
	}

	cmd.Flags().String("selector", "", "Scope to elements within this CSS selector")

	return cmd
}

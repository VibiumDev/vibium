package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func newKeysCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "keys [keys]",
		Short: "Press a key or key combination",
		Example: `  vibium keys Enter
  # Press Enter (daemon mode)

  vibium keys "Control+a"
  # Select all

  vibium keys "Shift+Tab"
  # Shift+Tab to previous field`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			keys := args[0]

			if !oneshot {
				result, err := daemonCall("browser_keys", map[string]interface{}{"keys": keys})
				if err != nil {
					printError(err)
					return
				}
				printResult(result)
				return
			}

			fmt.Fprintf(os.Stderr, "Error: keys command requires daemon mode\n")
			os.Exit(1)
		},
	}
}

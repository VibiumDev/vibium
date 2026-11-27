package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func newOfflineCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "offline [on|off]",
		Short: "Cut the page off from the network, or reconnect it",
		Example: `  vibium offline on
  # → Network offline

  vibium offline off
  # → Network online`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			if args[0] != "on" && args[0] != "off" {
				fmt.Fprintf(os.Stderr, "Error: offline takes on or off, got %q\n", args[0])
				os.Exit(1)
			}
			result, err := daemonCall("browser_set_offline", map[string]interface{}{"offline": args[0] == "on"})
			if err != nil {
				printError(err)
				return
			}
			printResult(result)
		},
	}
}

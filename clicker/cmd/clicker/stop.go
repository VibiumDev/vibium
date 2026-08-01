package main

import (
	"github.com/spf13/cobra"
)

func newStopCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "stop",
		Short: "Stop the browser session",
		Example: `  vibium stop
  # Stop the browser and daemon`,
		Args: cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			result, err := daemonCall("browser_stop", map[string]interface{}{})
			if err != nil {
				printError(err)
				return
			}
			// Full teardown: a daemon left behind keeps holding the session's
			// connect settings, so a later command would silently reuse them.
			if err := shutdownDaemonAndWait(); err != nil {
				printError(err)
				return
			}
			printResult(result)
		},
	}
}

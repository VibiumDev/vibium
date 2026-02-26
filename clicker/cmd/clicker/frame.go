package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func newFrameCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "frame [nameOrUrl]",
		Short: "Find a frame by name or URL substring",
		Example: `  vibium frame "myIframe"
  # Find frame by name

  vibium frame "example.com"
  # Find frame by URL substring`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			nameOrURL := args[0]

			if !oneshot {
				result, err := daemonCall("browser_frame", map[string]interface{}{"nameOrUrl": nameOrURL})
				if err != nil {
					printError(err)
					return
				}
				printResult(result)
				return
			}

			fmt.Fprintf(os.Stderr, "Error: frame command requires daemon mode\n")
			os.Exit(1)
		},
	}
}

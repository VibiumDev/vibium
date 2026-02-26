package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func newViewportCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "viewport",
		Short: "Get the current viewport dimensions",
		Example: `  vibium viewport
  # {"width":1280,"height":720,"devicePixelRatio":1}`,
		Args: cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			if !oneshot {
				result, err := daemonCall("browser_get_viewport", map[string]interface{}{})
				if err != nil {
					printError(err)
					return
				}
				printResult(result)
				return
			}

			fmt.Fprintf(os.Stderr, "Error: viewport command requires daemon mode\n")
			os.Exit(1)
		},
	}
}

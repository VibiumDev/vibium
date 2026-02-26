package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func newMouseDownCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mouse-down",
		Short: "Press a mouse button down",
		Example: `  vibium mouse-down
  # Press left mouse button

  vibium mouse-down --button 2
  # Press right mouse button`,
		Args: cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			button, _ := cmd.Flags().GetInt("button")

			if !oneshot {
				result, err := daemonCall("browser_mouse_down", map[string]interface{}{"button": float64(button)})
				if err != nil {
					printError(err)
					return
				}
				printResult(result)
				return
			}

			fmt.Fprintf(os.Stderr, "Error: mouse-down command requires daemon mode\n")
			os.Exit(1)
		},
	}
	cmd.Flags().Int("button", 0, "Mouse button (0=left, 1=middle, 2=right)")
	return cmd
}

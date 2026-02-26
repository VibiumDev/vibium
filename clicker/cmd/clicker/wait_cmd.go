package main

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/vibium/clicker/internal/features"
)

func newWaitCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "wait [selector]",
		Short: "Wait for an element to reach a specified state",
		Example: `  vibium wait "div.loaded"
  # Wait for element to exist in DOM (daemon mode)

  vibium wait "div.loaded" --state visible
  # Wait for element to be visible

  vibium wait "div.spinner" --state hidden --timeout 5000
  # Wait for spinner to disappear`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			selector := args[0]
			state, _ := cmd.Flags().GetString("state")
			timeoutMs, _ := cmd.Flags().GetInt("timeout")

			if !oneshot {
				toolArgs := map[string]interface{}{
					"selector": selector,
					"state":    state,
					"timeout":  float64(timeoutMs),
				}

				result, err := daemonCall("browser_wait", toolArgs)
				if err != nil {
					printError(err)
					return
				}
				printResult(result)
				return
			}

			// Oneshot mode — not very useful without navigation, but support it
			fmt.Fprintf(os.Stderr, "Error: wait command requires daemon mode\n")
			os.Exit(1)
		},
	}
	cmd.Flags().String("state", "attached", "State to wait for: attached, visible, hidden")
	cmd.Flags().Int("timeout", int(features.DefaultTimeout/time.Millisecond), "Timeout in milliseconds")
	return cmd
}

package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func newScrollCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "scroll [direction]",
		Short: "Scroll the page or an element",
		Example: `  vibium scroll
  # Scroll down by default (daemon mode)

  vibium scroll up
  # Scroll up

  vibium scroll down --amount 5
  # Scroll down 5 increments

  vibium scroll down --selector "div.content"
  # Scroll within a specific element`,
		Args: cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			direction := "down"
			if len(args) == 1 {
				direction = args[0]
			}
			amount, _ := cmd.Flags().GetInt("amount")
			selector, _ := cmd.Flags().GetString("selector")

			if !oneshot {
				toolArgs := map[string]interface{}{
					"direction": direction,
					"amount":    float64(amount),
				}
				if selector != "" {
					toolArgs["selector"] = selector
				}

				result, err := daemonCall("browser_scroll", toolArgs)
				if err != nil {
					printError(err)
					return
				}
				printResult(result)
				return
			}

			fmt.Fprintf(os.Stderr, "Error: scroll command requires daemon mode\n")
			os.Exit(1)
		},
	}
	cmd.Flags().Int("amount", 3, "Number of scroll increments")
	cmd.Flags().String("selector", "", "CSS selector for element to scroll to")
	return cmd
}

package main

import (
	"time"

	"github.com/spf13/cobra"
)

func newScrollCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "scroll [direction]",
		Short: "Scroll the page or an element",
		Example: `  vibium scroll
  # Scroll down by default

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
		},
	}
	cmd.Flags().Int("amount", 3, "Number of scroll increments")
	cmd.Flags().String("selector", "", "CSS selector for element to scroll to")

	var intoViewTimeout time.Duration
	intoViewCmd := &cobra.Command{
		Use:   "into-view [selector]",
		Short: "Scroll an element into view",
		Example: `  vibium scroll into-view "#footer"
  # Scroll the footer element into view (centered on screen)

  vibium scroll into-view "#footer" --timeout 5s
  # Custom timeout (5s, or 5000 for milliseconds)`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			result, err := daemonCall("browser_scroll_into_view", map[string]interface{}{
				"selector": args[0],
				"timeout":  float64(intoViewTimeout.Milliseconds()),
			})
			if err != nil {
				printError(err)
				return
			}
			printResult(result)
		},
	}
	addTimeoutFlag(intoViewCmd, &intoViewTimeout)

	cmd.AddCommand(intoViewCmd)
	return cmd
}

package main

import (
	"time"

	"github.com/spf13/cobra"
)

func newFillCmd() *cobra.Command {
	var timeout time.Duration
	cmd := &cobra.Command{
		Use:   "fill [selector] [text]",
		Short: "Clear an input field and type new text",
		Example: `  vibium fill "input[name=email]" "user@example.com"
  # Clear the field and type new value
  vibium fill "#search" "vibium"
  # Replace search field contents

  vibium fill "#search" "vibium" --timeout 5s
  # Custom timeout (5s, or 5000 for milliseconds)`,
		Args: cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			selector := args[0]
			text := args[1]
			result, err := daemonCall("browser_fill", map[string]interface{}{
				"selector": selector,
				"value":    text,
				"timeout":  float64(timeout.Milliseconds()),
			})
			if err != nil {
				printError(err)
				return
			}
			printResult(result)
		},
	}
	addTimeoutFlag(cmd, &timeout)
	// fill now carries a --timeout flag, so it can't disable flag parsing the way
	// #179 did for the flagless case. SetInterspersed(false) instead keeps negative
	// positional values (e.g. fill "#x" "-2") from being parsed as shorthand flags.
	cmd.Flags().SetInterspersed(false)
	return cmd
}

package main

import (
	"fmt"
	"os"
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
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 2 {
				fmt.Fprintf(os.Stderr, "Error: accepts 2 arg(s), received %d\n", len(args))
				os.Exit(1)
			}
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
	// Values like fill "#x" "-2" must not be parsed as shorthand flags (#179).
	return lateParse(cmd)
}

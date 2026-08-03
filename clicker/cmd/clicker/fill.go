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
		DisableFlagParsing: true,
		Args:               cobra.ArbitraryArgs,
		Run: func(cmd *cobra.Command, args []string) {
			args, perr := parseFlagsAllowNegative(cmd, args)
			if perr != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", perr)
				os.Exit(1)
			}
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
	// fill now carries a --timeout flag, so it can't disable flag parsing the way
	// #179 did for the flagless case. SetInterspersed(false) instead keeps negative
	// positional values (e.g. fill "#x" "-2") from being parsed as shorthand flags.
	return cmd
}

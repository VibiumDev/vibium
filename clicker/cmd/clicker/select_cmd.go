package main

import (
	"time"

	"github.com/spf13/cobra"
)

func newSelectCmd() *cobra.Command {
	var timeout time.Duration
	cmd := &cobra.Command{
		Use:   "select [selector] [value]",
		Short: "Select an option in a <select> element",
		Example: `  vibium select "select#color" "blue"
  # Select "blue" in the color dropdown

  vibium select "select#color" "blue" --timeout 5s
  # Custom timeout (5s, or 5000 for milliseconds)`,
		Args: cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			selector := args[0]
			value := args[1]

			result, err := daemonCall("browser_select", map[string]interface{}{
				"selector": selector,
				"value":    value,
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
	return cmd
}

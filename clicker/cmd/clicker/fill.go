package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func newFillCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "fill [selector] [text]",
		Short: "Clear an input field and type new text",
		Example: `  vibium fill "input[name=email]" "user@example.com"
  # Clear the field and type new value

  vibium fill "#search" "vibium"
  # Replace search field contents`,
		Args: cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			selector := args[0]
			text := args[1]

			if !oneshot {
				result, err := daemonCall("browser_fill", map[string]interface{}{
					"selector": selector,
					"text":     text,
				})
				if err != nil {
					printError(err)
					return
				}
				printResult(result)
				return
			}

			fmt.Fprintf(os.Stderr, "Error: fill command requires daemon mode\n")
			os.Exit(1)
		},
	}
}

package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func newAttrCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "attr [selector] [attribute]",
		Short: "Get an HTML attribute value from an element",
		Example: `  vibium attr "a" "href"
  # Get the href of the first link

  vibium attr "img" "src"
  # Get the image source URL`,
		Args: cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			selector := args[0]
			attribute := args[1]

			if !oneshot {
				result, err := daemonCall("browser_get_attribute", map[string]interface{}{
					"selector":  selector,
					"attribute": attribute,
				})
				if err != nil {
					printError(err)
					return
				}
				printResult(result)
				return
			}

			fmt.Fprintf(os.Stderr, "Error: attr command requires daemon mode\n")
			os.Exit(1)
		},
	}
}

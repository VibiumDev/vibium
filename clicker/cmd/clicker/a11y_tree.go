package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func newA11yTreeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "a11y-tree",
		Short: "Get the accessibility tree of the current page",
		Example: `  vibium a11y-tree
  # Print the accessibility tree (interesting nodes only)

  vibium a11y-tree --everything
  # Include all nodes (generic containers, etc.)`,
		Args: cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			everything, _ := cmd.Flags().GetBool("everything")

			if !oneshot {
				toolArgs := map[string]interface{}{}
				if everything {
					toolArgs["everything"] = true
				}

				result, err := daemonCall("browser_a11y_tree", toolArgs)
				if err != nil {
					printError(err)
					return
				}
				printResult(result)
				return
			}

			fmt.Fprintf(os.Stderr, "Error: a11y-tree command requires daemon mode\n")
			os.Exit(1)
		},
	}
	cmd.Flags().Bool("everything", false, "Show all nodes including generic containers")
	return cmd
}

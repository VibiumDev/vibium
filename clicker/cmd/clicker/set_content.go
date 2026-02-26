package main

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
)

func newSetContentCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set-content [html]",
		Short: "Replace the page HTML content",
		Example: `  vibium set-content "<h1>Hello World</h1>"
  # Set page content directly

  echo "<h1>Hello</h1>" | vibium set-content --stdin
  # Set page content from stdin`,
		Args: cobra.RangeArgs(0, 1),
		Run: func(cmd *cobra.Command, args []string) {
			useStdin, _ := cmd.Flags().GetBool("stdin")

			var html string
			if useStdin {
				data, err := io.ReadAll(os.Stdin)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error reading stdin: %v\n", err)
					os.Exit(1)
				}
				html = string(data)
			} else if len(args) == 1 {
				html = args[0]
			} else {
				fmt.Fprintf(os.Stderr, "Error: html argument or --stdin flag is required\n")
				os.Exit(1)
			}

			if !oneshot {
				result, err := daemonCall("browser_set_content", map[string]interface{}{"html": html})
				if err != nil {
					printError(err)
					return
				}
				printResult(result)
				return
			}

			fmt.Fprintf(os.Stderr, "Error: set-content command requires daemon mode\n")
			os.Exit(1)
		},
	}
	cmd.Flags().Bool("stdin", false, "Read HTML from stdin")
	return cmd
}

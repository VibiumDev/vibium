package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func newStorageStateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "storage-state",
		Short: "Export browser state (cookies, localStorage, sessionStorage)",
		Example: `  vibium storage-state
  # Print state as JSON

  vibium storage-state -o state.json
  # Save state to file`,
		Args: cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			output, _ := cmd.Flags().GetString("output")

			if !oneshot {
				result, err := daemonCall("browser_storage_state", map[string]interface{}{})
				if err != nil {
					printError(err)
					return
				}
				if output != "" {
					// Save to file
					text := extractText(result)
					if err := os.WriteFile(output, []byte(text), 0644); err != nil {
						printError(fmt.Errorf("failed to write file: %w", err))
						return
					}
					fmt.Printf("State saved to %s\n", output)
					return
				}
				printResult(result)
				return
			}

			fmt.Fprintf(os.Stderr, "Error: storage-state command requires daemon mode\n")
			os.Exit(1)
		},
	}
	cmd.Flags().StringP("output", "o", "", "Output file path")
	return cmd
}

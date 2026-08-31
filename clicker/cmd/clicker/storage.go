package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

func newStorageCmd() *cobra.Command {
	storageCmd := &cobra.Command{
		Use:   "storage",
		Short: "Export or restore browser state (cookies, localStorage, sessionStorage)",
		// Unlike other subcommand parents, running this bare does real work
		// (prints the state); the annotation keeps it in `vibium commands`.
		Annotations: map[string]string{"standalone": "true"},
		Example: `  vibium storage
  # Print state as JSON

  vibium storage -o state.json
  # Save state to file`,
		Args: cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			output, _ := cmd.Flags().GetString("output")

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
				msg := fmt.Sprintf("State saved to %s", output)
				if jsonOutput {
					printJSON(jsonEnvelope{OK: true, Result: msg})
					return
				}
				fmt.Println(msg)
				return
			}
			printResult(result)
		},
	}
	storageCmd.Flags().StringP("output", "o", "", "Output file path")

	restoreCmd := &cobra.Command{
		Use:   "restore [path]",
		Short: "Restore browser state from a JSON file",
		Example: `  vibium storage restore state.json
  # Restore cookies and storage from saved state`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			path, err := filepath.Abs(args[0])
			if err != nil {
				printError(fmt.Errorf("invalid path: %w", err))
				return
			}

			result, err := daemonCall("browser_restore_storage", map[string]interface{}{"path": path})
			if err != nil {
				printError(err)
				return
			}
			printResult(result)
		},
	}

	storageCmd.AddCommand(restoreCmd)
	return storageCmd
}

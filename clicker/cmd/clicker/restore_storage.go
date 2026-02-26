package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

func newRestoreStorageCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "restore-storage [path]",
		Short: "Restore browser state from a JSON file",
		Example: `  vibium restore-storage state.json
  # Restore cookies and storage from saved state`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			path, err := filepath.Abs(args[0])
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: invalid path: %v\n", err)
				os.Exit(1)
			}

			if !oneshot {
				result, err := daemonCall("browser_restore_storage", map[string]interface{}{"path": path})
				if err != nil {
					printError(err)
					return
				}
				printResult(result)
				return
			}

			fmt.Fprintf(os.Stderr, "Error: restore-storage command requires daemon mode\n")
			os.Exit(1)
		},
	}
}

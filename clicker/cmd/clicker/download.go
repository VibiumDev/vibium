package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

func newDownloadCmd() *cobra.Command {
	downloadCmd := &cobra.Command{
		Use:   "download",
		Short: "Manage browser downloads",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
	}

	setDirCmd := &cobra.Command{
		Use:   "set-dir [path]",
		Short: "Set the download directory",
		Example: `  vibium download set-dir ./downloads
  # Set download directory to ./downloads`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			dir, err := filepath.Abs(args[0])
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: invalid path: %v\n", err)
				os.Exit(1)
			}

			if !oneshot {
				result, err := daemonCall("browser_download_set_dir", map[string]interface{}{"path": dir})
				if err != nil {
					printError(err)
					return
				}
				printResult(result)
				return
			}

			fmt.Fprintf(os.Stderr, "Error: download command requires daemon mode\n")
			os.Exit(1)
		},
	}

	downloadCmd.AddCommand(setDirCmd)
	return downloadCmd
}

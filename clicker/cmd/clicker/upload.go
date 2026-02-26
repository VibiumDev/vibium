package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

func newUploadCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "upload [selector] [files...]",
		Short: "Set files on an input[type=file] element",
		Example: `  vibium upload "input[type=file]" ./photo.jpg
  # Upload a single file

  vibium upload "#file-input" ./photo.jpg ./doc.pdf
  # Upload multiple files`,
		Args: cobra.MinimumNArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			selector := args[0]
			filePaths := args[1:]

			// Resolve to absolute paths
			absFiles := make([]interface{}, len(filePaths))
			for i, f := range filePaths {
				abs, err := filepath.Abs(f)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error: invalid file path %q: %v\n", f, err)
					os.Exit(1)
				}
				absFiles[i] = abs
			}

			if !oneshot {
				result, err := daemonCall("browser_upload", map[string]interface{}{
					"selector": selector,
					"files":    absFiles,
				})
				if err != nil {
					printError(err)
					return
				}
				printResult(result)
				return
			}

			fmt.Fprintf(os.Stderr, "Error: upload command requires daemon mode\n")
			os.Exit(1)
		},
	}
}

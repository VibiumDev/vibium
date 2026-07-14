package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
)

func newUploadCmd() *cobra.Command {
	var timeout time.Duration
	cmd := &cobra.Command{
		Use:   "upload [selector] [files...]",
		Short: "Set files on an input[type=file] element",
		Example: `  vibium upload "input[type=file]" ./photo.jpg
  # Upload a single file

  vibium upload "#file-input" ./photo.jpg ./doc.pdf
  # Upload multiple files

  vibium upload "input[type=file]" ./photo.jpg --timeout 5s
  # Custom timeout (5s, or 5000 for milliseconds)`,
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

			result, err := daemonCall("browser_upload", map[string]interface{}{
				"selector": selector,
				"files":    absFiles,
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

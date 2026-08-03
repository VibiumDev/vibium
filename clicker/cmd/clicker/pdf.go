package main

import (
	"path/filepath"

	"github.com/spf13/cobra"
)

func newPDFCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pdf [url]",
		Short: "Save page as PDF",
		Example: `  vibium pdf -o page.pdf
  # Save current page as PDF

  vibium pdf https://example.com -o page.pdf
  # Navigate to URL first, then save as PDF`,
		Args: cobra.RangeArgs(0, 1),
		Run: func(cmd *cobra.Command, args []string) {
			output, _ := cmd.Flags().GetString("output")
			// The daemon is a separate long-lived process whose working directory
			// is not the user's, so resolve against this shell before the path goes
			// over the socket (#119).
			if abs, err := filepath.Abs(output); err == nil {
				output = abs
			}

			// Navigate first if URL provided
			if len(args) == 1 {
				_, err := daemonCall("browser_navigate", map[string]interface{}{"url": args[0]})
				if err != nil {
					printError(err)
					return
				}
			}

			result, err := daemonCall("browser_pdf", map[string]interface{}{"filename": output})
			if err != nil {
				printError(err)
				return
			}
			printResult(result)
		},
	}
	cmd.Flags().StringP("output", "o", "page.pdf", "Output file path")
	return cmd
}

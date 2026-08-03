package main

import (
	"path/filepath"

	"github.com/spf13/cobra"
)

func newScreenshotCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "screenshot [url]",
		Short: "Capture a screenshot (optionally navigate to URL first)",
		Example: `  vibium screenshot
  # Saves ./screenshot.png in the current directory

  vibium screenshot -o shot.png
  # Screenshots the current page to ./shot.png

  vibium screenshot -o ~/Pictures/shot.png
  # Paths are honored as given

  vibium screenshot https://example.com -o shot.png
  # Navigates to URL first, then screenshots

  vibium screenshot -o full.png --full-page
  # Capture the entire page (not just the viewport)`,
		Args: cobra.RangeArgs(0, 1),
		Run: func(cmd *cobra.Command, args []string) {
			output, _ := cmd.Flags().GetString("output")
			// The daemon is a separate long-lived process whose working
			// directory is not the user's, so resolve against this shell
			// before the path goes over the socket. Typing a path into a
			// terminal means "here", not "wherever the daemon started" —
			// the ~/Pictures/Vibium default exists for MCP, where the
			// caller cannot see a working directory at all (#119).
			if abs, err := filepath.Abs(output); err == nil {
				output = abs
			}
			fullPage, _ := cmd.Flags().GetBool("full-page")
			annotate, _ := cmd.Flags().GetBool("annotate")

			// Navigate first if URL provided
			if len(args) == 1 {
				_, err := daemonCall("browser_navigate", map[string]interface{}{"url": args[0]})
				if err != nil {
					printError(err)
					return
				}
			}

			// Take screenshot with filename
			screenshotArgs := map[string]interface{}{"filename": output}
			if fullPage {
				screenshotArgs["fullPage"] = true
			}
			if annotate {
				screenshotArgs["annotate"] = true
			}
			result, err := daemonCall("browser_screenshot", screenshotArgs)
			if err != nil {
				printError(err)
				return
			}
			printResult(result)
		},
	}
	cmd.Flags().StringP("output", "o", "screenshot.png", "Output file path")
	cmd.Flags().Bool("full-page", false, "Capture the full page instead of just the viewport")
	cmd.Flags().Bool("annotate", false, "Annotate interactive elements with numbered labels")
	return cmd
}

package main

import (
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

func newScreenshotCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "screenshot [url]",
		Short: "Capture a screenshot (optionally navigate to URL first)",
		Example: `  vibium screenshot -o shot.png
  # default screenshot dir (~/Pictures/Vibium/shot.png)

  vibium screenshot -o ./shot.png
  # cwd-relative; path component honored

  vibium screenshot -o /tmp/shot.png
  # absolute path honored

  vibium screenshot https://example.com -o shot.png
  # navigate first, then screenshot

  vibium screenshot -o full.png --full-page
  # capture the entire page`,
		Args: cobra.RangeArgs(0, 1),
		Run: func(cmd *cobra.Command, args []string) {
			output, _ := cmd.Flags().GetString("output")
			fullPage, _ := cmd.Flags().GetBool("full-page")
			annotate, _ := cmd.Flags().GetBool("annotate")

			if len(args) == 1 {
				_, err := daemonCall("browser_navigate", map[string]interface{}{"url": args[0]})
				if err != nil {
					printError(err)
					return
				}
			}

			// Resolve relative paths against the CLI's cwd, not the daemon's.
			if strings.ContainsAny(output, "/"+string(filepath.Separator)) {
				if abs, err := filepath.Abs(output); err == nil {
					output = abs
				}
			}

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
	cmd.Flags().StringP("output", "o", "screenshot.png", "Output file path (bare filename → screenshot dir; path with separator → honored as written)")
	cmd.Flags().Bool("full-page", false, "Capture the full page instead of just the viewport")
	cmd.Flags().Bool("annotate", false, "Annotate interactive elements with numbered labels")
	return cmd
}

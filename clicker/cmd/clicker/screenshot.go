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
  # Saves to the default screenshot dir (~/Pictures/Vibium/shot.png)

  vibium screenshot -o ./shot.png
  # Saves to ./shot.png in the current directory (path component honored)

  vibium screenshot -o /tmp/shot.png
  # Saves to an absolute path (honored as-is)

  vibium screenshot https://example.com -o shot.png
  # Navigates to URL first, then screenshots

  vibium screenshot -o full.png --full-page
  # Capture the entire page (not just the viewport)`,
		Args: cobra.RangeArgs(0, 1),
		Run: func(cmd *cobra.Command, args []string) {
			output, _ := cmd.Flags().GetString("output")
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

			// If --output contains a directory component, resolve it against the
			// operator's CWD before sending — otherwise the daemon (which has its
			// own CWD) would interpret a relative path against the wrong root.
			// A bare basename like "shot.png" is left unchanged so the daemon's
			// default screenshot dir still applies.
			if strings.ContainsAny(output, "/"+string(filepath.Separator)) {
				if abs, err := filepath.Abs(output); err == nil {
					output = abs
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
	cmd.Flags().StringP("output", "o", "screenshot.png", "Output file path. Bare filenames go to the default screenshot dir; paths with a directory component (./foo, /abs/foo, sub/foo) are honored as written.")
	cmd.Flags().Bool("full-page", false, "Capture the full page instead of just the viewport")
	cmd.Flags().Bool("annotate", false, "Annotate interactive elements with numbered labels")
	return cmd
}

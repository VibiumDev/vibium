package main

import (
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

func newPDFCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pdf [url]",
		Short: "Save page as PDF",
		Example: `  vibium pdf -o page.pdf
  # Save current page as PDF

  vibium pdf https://example.com -o page.pdf
  # Navigate to URL first, then save as PDF

  vibium pdf https://example.com -o report.pdf --landscape --scale 0.8 --background
  # Landscape at 80% scale with background graphics
  # PDF saved to report.pdf (183940 bytes)

  vibium pdf -o excerpt.pdf --page-ranges 1,3-5 --margin 2
  # Pages 1 and 3-5 with 2cm margins on all sides`,
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

			callArgs := map[string]interface{}{"filename": output}
			if v, _ := cmd.Flags().GetBool("landscape"); v {
				callArgs["landscape"] = true
			}
			if cmd.Flags().Changed("scale") {
				v, _ := cmd.Flags().GetFloat64("scale")
				callArgs["scale"] = v
			}
			if v, _ := cmd.Flags().GetBool("background"); v {
				callArgs["background"] = true
			}
			if cmd.Flags().Changed("margin") {
				v, _ := cmd.Flags().GetFloat64("margin")
				for _, side := range []string{"marginTop", "marginBottom", "marginLeft", "marginRight"} {
					callArgs[side] = v
				}
			}
			if cmd.Flags().Changed("page-width") {
				v, _ := cmd.Flags().GetFloat64("page-width")
				callArgs["pageWidth"] = v
			}
			if cmd.Flags().Changed("page-height") {
				v, _ := cmd.Flags().GetFloat64("page-height")
				callArgs["pageHeight"] = v
			}
			if ranges, _ := cmd.Flags().GetString("page-ranges"); ranges != "" {
				var list []interface{}
				for _, r := range strings.Split(ranges, ",") {
					if r = strings.TrimSpace(r); r != "" {
						list = append(list, r)
					}
				}
				callArgs["pageRanges"] = list
			}

			result, err := daemonCall("browser_pdf", callArgs)
			if err != nil {
				printError(err)
				return
			}
			printResult(result)
		},
	}
	cmd.Flags().StringP("output", "o", "page.pdf", "Output file path")
	cmd.Flags().Bool("landscape", false, "Landscape orientation")
	cmd.Flags().Float64("scale", 1, "Print scale, 0.1-2")
	cmd.Flags().Bool("background", false, "Print background graphics")
	cmd.Flags().Float64("margin", 1, "Margin on all sides in cm")
	cmd.Flags().Float64("page-width", 21.59, "Page width in cm")
	cmd.Flags().Float64("page-height", 27.94, "Page height in cm")
	cmd.Flags().String("page-ranges", "", "Pages to print, e.g. 1,3-5 (default all)")
	return cmd
}

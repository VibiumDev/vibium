package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func newEmulateMediaCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "emulate-media",
		Short: "Override CSS media features",
		Example: `  vibium emulate-media --color-scheme dark
  # Enable dark mode

  vibium emulate-media --reduced-motion reduce
  # Reduce motion

  vibium emulate-media --color-scheme light --forced-colors active
  # Override multiple features`,
		Args: cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			colorScheme, _ := cmd.Flags().GetString("color-scheme")
			reducedMotion, _ := cmd.Flags().GetString("reduced-motion")
			forcedColors, _ := cmd.Flags().GetString("forced-colors")
			contrast, _ := cmd.Flags().GetString("contrast")
			media, _ := cmd.Flags().GetString("media")

			callArgs := map[string]interface{}{}
			if colorScheme != "" {
				callArgs["colorScheme"] = colorScheme
			}
			if reducedMotion != "" {
				callArgs["reducedMotion"] = reducedMotion
			}
			if forcedColors != "" {
				callArgs["forcedColors"] = forcedColors
			}
			if contrast != "" {
				callArgs["contrast"] = contrast
			}
			if media != "" {
				callArgs["media"] = media
			}

			if len(callArgs) == 0 {
				fmt.Fprintf(os.Stderr, "Error: at least one media feature flag is required\n")
				os.Exit(1)
			}

			if !oneshot {
				result, err := daemonCall("browser_emulate_media", callArgs)
				if err != nil {
					printError(err)
					return
				}
				printResult(result)
				return
			}

			fmt.Fprintf(os.Stderr, "Error: emulate-media command requires daemon mode\n")
			os.Exit(1)
		},
	}
	cmd.Flags().String("color-scheme", "", "Color scheme: light, dark, no-preference")
	cmd.Flags().String("reduced-motion", "", "Reduced motion: reduce, no-preference")
	cmd.Flags().String("forced-colors", "", "Forced colors: active, none")
	cmd.Flags().String("contrast", "", "Contrast: more, less, no-preference")
	cmd.Flags().String("media", "", "Media type: screen, print")
	return cmd
}

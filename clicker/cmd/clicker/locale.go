package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func newLocaleCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "locale [tag]",
		Short: "Make the page report a locale such as de-DE",
		Example: `  vibium locale de-DE
  # → Locale set to de-DE

  vibium locale --reset
  # → Locale override removed`,
		Args: cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			reset, _ := cmd.Flags().GetBool("reset")
			if reset == (len(args) == 1) {
				fmt.Fprintf(os.Stderr, "Error: locale takes a language tag such as de-DE, or --reset\n")
				os.Exit(1)
			}
			locale := ""
			if !reset {
				locale = args[0]
			}
			result, err := daemonCall("browser_set_locale", map[string]interface{}{"locale": locale})
			if err != nil {
				printError(err)
				return
			}
			printResult(result)
		},
	}
	cmd.Flags().Bool("reset", false, "Remove the locale override")
	return cmd
}

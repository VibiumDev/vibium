package main

import (
	"strconv"

	"github.com/spf13/cobra"
)

func newPageCmd() *cobra.Command {
	pageCmd := &cobra.Command{
		Use:   "page",
		Short: "Manage browser pages (new, close, switch)",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
	}

	var isolated bool
	newCmd := &cobra.Command{
		Use:   "new [url]",
		Short: "Open a new browser page",
		Example: `  vibium page new
  # Open a blank new page

  vibium page new https://example.com
  # Open a new page and navigate to URL

  vibium page new --isolated https://example.com
  # Open the page in its own isolated context (separate cookies/storage):
  #   New isolated page opened and navigated to https://example.com (page: A1B2...)`,
		Args: cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			toolArgs := map[string]interface{}{}
			if len(args) == 1 {
				toolArgs["url"] = args[0]
			}
			if isolated {
				toolArgs["isolated"] = true
			}

			result, err := daemonCall("browser_new_page", toolArgs)
			if err != nil {
				printError(err)
				return
			}
			printResult(result)
		},
	}
	newCmd.Flags().BoolVar(&isolated, "isolated", false, "Open the page in its own isolated context (separate cookies and storage)")

	closeCmd := &cobra.Command{
		Use:   "close [index or page id]",
		Short: "Close a browser page by index or id (default: current page)",
		Example: `  vibium page close
  # Close current page (index 0)

  vibium page close 1
  # Close page at index 1

  vibium page close A1B2C3D4
  # Close the page with that id (from "vibium page new")`,
		Args: cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			toolArgs := map[string]interface{}{}
			if len(args) == 1 {
				// An integer is an index; anything else is a page id.
				if idx, err := strconv.Atoi(args[0]); err == nil {
					toolArgs["index"] = float64(idx)
				} else {
					toolArgs["page"] = args[0]
				}
			}

			result, err := daemonCall("browser_close_page", toolArgs)
			if err != nil {
				printError(err)
				return
			}
			printResult(result)
		},
	}

	switchCmd := &cobra.Command{
		Use:   "switch [index or url]",
		Short: "Switch to a browser page by index or URL substring",
		Example: `  vibium page switch 1
  # Switch to page at index 1

  vibium page switch google.com
  # Switch to page containing "google.com" in URL`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			toolArgs := map[string]interface{}{}

			// Try to parse as integer index
			if idx, err := strconv.Atoi(args[0]); err == nil {
				toolArgs["index"] = float64(idx)
			} else {
				toolArgs["url"] = args[0]
			}

			result, err := daemonCall("browser_switch_page", toolArgs)
			if err != nil {
				printError(err)
				return
			}
			printResult(result)
		},
	}

	pageCmd.AddCommand(newCmd)
	pageCmd.AddCommand(closeCmd)
	pageCmd.AddCommand(switchCmd)
	return pageCmd
}

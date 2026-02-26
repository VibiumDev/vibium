package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func newDialogCmd() *cobra.Command {
	dialogCmd := &cobra.Command{
		Use:   "dialog",
		Short: "Handle browser dialogs (alert, confirm, prompt)",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
	}

	acceptCmd := &cobra.Command{
		Use:   "accept [text]",
		Short: "Accept a dialog (optionally with prompt text)",
		Example: `  vibium dialog accept
  # Accept an alert or confirm dialog

  vibium dialog accept "my input"
  # Accept a prompt dialog with text`,
		Args: cobra.RangeArgs(0, 1),
		Run: func(cmd *cobra.Command, args []string) {
			if !oneshot {
				callArgs := map[string]interface{}{}
				if len(args) == 1 {
					callArgs["text"] = args[0]
				}
				result, err := daemonCall("browser_dialog_accept", callArgs)
				if err != nil {
					printError(err)
					return
				}
				printResult(result)
				return
			}

			fmt.Fprintf(os.Stderr, "Error: dialog command requires daemon mode\n")
			os.Exit(1)
		},
	}

	dismissCmd := &cobra.Command{
		Use:   "dismiss",
		Short: "Dismiss a dialog",
		Example: `  vibium dialog dismiss
  # Dismiss/cancel a dialog`,
		Args: cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			if !oneshot {
				result, err := daemonCall("browser_dialog_dismiss", map[string]interface{}{})
				if err != nil {
					printError(err)
					return
				}
				printResult(result)
				return
			}

			fmt.Fprintf(os.Stderr, "Error: dialog command requires daemon mode\n")
			os.Exit(1)
		},
	}

	dialogCmd.AddCommand(acceptCmd)
	dialogCmd.AddCommand(dismissCmd)
	return dialogCmd
}

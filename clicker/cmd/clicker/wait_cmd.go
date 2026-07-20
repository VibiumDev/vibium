package main

import (
	"time"

	"github.com/spf13/cobra"
)

func newWaitCmd() *cobra.Command {
	var waitTimeout, urlTimeout, textTimeout, loadTimeout, fnTimeout time.Duration

	cmd := &cobra.Command{
		Use:   "wait [selector]",
		Short: "Wait for an element, URL, text, page load, or JS condition",
		Example: `  vibium wait "div.loaded"
  # Wait for element to exist in DOM

  vibium wait "div.loaded" --state visible
  # Wait for element to be visible

  vibium wait "div.spinner" --state hidden --timeout 5s
  # Wait for spinner to disappear (5s, or 5000 for milliseconds)`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			selector := args[0]
			state, _ := cmd.Flags().GetString("state")

			result, err := daemonCall("browser_wait", map[string]interface{}{
				"selector": selector,
				"state":    state,
				"timeout":  float64(waitTimeout.Milliseconds()),
			})
			if err != nil {
				printError(err)
				return
			}
			printResult(result)
		},
	}
	cmd.Flags().String("state", "attached", "State to wait for: attached, visible, hidden")
	addTimeoutFlag(cmd, &waitTimeout)

	urlCmd := &cobra.Command{
		Use:   "url [pattern]",
		Short: "Wait until the page URL contains a substring",
		Example: `  vibium wait url "/dashboard"
  # Wait until URL contains "/dashboard"

  vibium wait url "success" --timeout 10s
  # Wait up to 10 seconds (or 10000 for milliseconds)`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			result, err := daemonCall("browser_wait_for_url", map[string]interface{}{
				"pattern": args[0],
				"timeout": float64(urlTimeout.Milliseconds()),
			})
			if err != nil {
				printError(err)
				return
			}
			printResult(result)
		},
	}
	addTimeoutFlag(urlCmd, &urlTimeout)

	textCmd := &cobra.Command{
		Use:   "text [text]",
		Short: "Wait until text appears on the page",
		Example: `  vibium wait text "Welcome"
  # Waits until "Welcome" appears on the page

  vibium wait text "Success" --timeout 10s
  # Wait with custom timeout (10s, or 10000 for milliseconds)`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			result, err := daemonCall("browser_wait_for_text", map[string]interface{}{
				"text":    args[0],
				"timeout": float64(textTimeout.Milliseconds()),
			})
			if err != nil {
				printError(err)
				return
			}
			printResult(result)
		},
	}
	addTimeoutFlag(textCmd, &textTimeout)

	loadCmd := &cobra.Command{
		Use:   "load",
		Short: "Wait until the page is fully loaded",
		Example: `  vibium wait load
  # Wait until document.readyState is "complete"

  vibium wait load --timeout 10s
  # Wait up to 10 seconds (or 10000 for milliseconds)`,
		Args: cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			result, err := daemonCall("browser_wait_for_load", map[string]interface{}{
				"timeout": float64(loadTimeout.Milliseconds()),
			})
			if err != nil {
				printError(err)
				return
			}
			printResult(result)
		},
	}
	addTimeoutFlag(loadCmd, &loadTimeout)

	fnCmd := &cobra.Command{
		Use:   "fn [expression]",
		Short: "Wait until a JS expression returns truthy",
		Example: `  vibium wait fn "document.readyState === 'complete'"
  # Wait for page to be fully loaded

  vibium wait fn "window.ready === true" --timeout 10s
  # Wait for custom condition (10s, or 10000 for milliseconds)`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			result, err := daemonCall("browser_wait_for_fn", map[string]interface{}{
				"expression": args[0],
				"timeout":    float64(fnTimeout.Milliseconds()),
			})
			if err != nil {
				printError(err)
				return
			}
			printResult(result)
		},
	}
	addTimeoutFlag(fnCmd, &fnTimeout)

	cmd.AddCommand(urlCmd)
	cmd.AddCommand(textCmd)
	cmd.AddCommand(loadCmd)
	cmd.AddCommand(fnCmd)
	return cmd
}

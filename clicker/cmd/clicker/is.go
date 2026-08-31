package main

import (
	"encoding/json"
	"fmt"
	"github.com/vibium/clicker/internal/agent"
	"github.com/vibium/clicker/internal/api"
	"github.com/vibium/clicker/internal/process"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

// exitOnFalse makes `--fail` turn a "false" answer into a non-zero exit, so the
// is-commands can be used in a conditional. Without it the only signal is
// stdout, since a well-formed "false" is a successful query (#206).
func exitOnFalse(cmd *cobra.Command, result *agent.ToolsCallResult) {
	fail, _ := cmd.Flags().GetBool("fail")
	if fail && strings.TrimSpace(extractText(result)) == "false" {
		process.KillAll()
		os.Exit(1)
	}
}

func newIsCmd() *cobra.Command {
	isCmd := &cobra.Command{
		Use:   "is",
		Short: "Check element state (visible, enabled, checked, actionable)",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
	}

	visibleCmd := &cobra.Command{
		Use:   "visible [selector]",
		Short: "Check if an element is visible on the page",
		Example: `  vibium is visible "h1"
  # Prints true or false`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			result, err := daemonCall("browser_is_visible", map[string]interface{}{"selector": args[0]})
			if err != nil {
				printError(err)
				return
			}
			printResult(result)
			exitOnFalse(cmd, result)
		},
	}

	enabledCmd := &cobra.Command{
		Use:   "enabled [selector]",
		Short: "Check if an element is enabled",
		Example: `  vibium is enabled "button[type=submit]"
  # Prints true or false`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			result, err := daemonCall("browser_is_enabled", map[string]interface{}{"selector": args[0]})
			if err != nil {
				printError(err)
				return
			}
			printResult(result)
			exitOnFalse(cmd, result)
		},
	}

	checkedCmd := &cobra.Command{
		Use:   "checked [selector]",
		Short: "Check if a checkbox or radio is checked",
		Example: `  vibium is checked "input[type=checkbox]"
  # Prints true or false`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			result, err := daemonCall("browser_is_checked", map[string]interface{}{"selector": args[0]})
			if err != nil {
				printError(err)
				return
			}
			printResult(result)
			exitOnFalse(cmd, result)
		},
	}

	actionableCmd := &cobra.Command{
		Use:   "actionable [url] [selector]",
		Short: "Check actionability of an element (Visible, Stable, ReceivesEvents, Enabled, Editable)",
		Example: `  vibium is actionable "a"
  # Output:
  # Checking actionability for selector: a
  # ✓ Visible: true
  # ✓ Stable: true
  # ✓ ReceivesEvents: true
  # ✓ Enabled: true
  # ✗ Editable: false

  vibium is actionable https://example.com "a"
  # Navigate first, then check`,
		Args: cobra.RangeArgs(1, 2),
		Run: func(cmd *cobra.Command, args []string) {
			// The optional leading [url] matches the other is-commands and
			// `find`; requiring it here made this the odd one out (#199).
			selector := args[0]
			if len(args) == 2 {
				if _, err := daemonCall("browser_navigate", map[string]interface{}{"url": args[0]}); err != nil {
					printError(err)
					return
				}
				selector = args[1]
			}

			if !jsonOutput {
				fmt.Printf("\nChecking actionability for selector: %s\n", selector)
			}

			// Evaluate actionability script
			script := `(() => {
				const selector = ` + fmt.Sprintf("%q", selector) + `;
				const el = document.querySelector(selector);
				if (!el) return JSON.stringify({ error: 'element not found' });

				const rect = el.getBoundingClientRect();
				const style = window.getComputedStyle(el);
				const visible = rect.width > 0 && rect.height > 0 &&
					style.visibility !== 'hidden' && style.display !== 'none';

				` + api.InViewCenterJS + `
				const hit = document.elementFromPoint(px, py);
				const receivesEvents = hit && (el === hit || el.contains(hit));

				let enabled = true;
				if (el.disabled === true) enabled = false;
				else if (el.getAttribute('aria-disabled') === 'true') enabled = false;
				else {
					const fs = el.closest('fieldset[disabled]');
					if (fs) { const legend = fs.querySelector('legend'); if (!legend || !legend.contains(el)) enabled = false; }
				}

				let editable = enabled && !el.readOnly && el.getAttribute('aria-readonly') !== 'true';
				if (editable) {
					const tag = el.tagName.toLowerCase();
					if (tag === 'input') {
						const t = (el.type || 'text').toLowerCase();
						editable = ` + api.FillableInputTypesJS + `.includes(t);
					} else if (tag !== 'textarea' && !el.isContentEditable) {
						editable = false;
					}
				}

				return JSON.stringify({ visible, stable: true, receivesEvents, enabled, editable });
			})()`

			result, err := daemonCall("browser_evaluate", map[string]interface{}{"expression": script})
			if err != nil {
				printError(err)
				return
			}

			// Parse the result
			resultText := ""
			if result != nil {
				for _, c := range result.Content {
					if c.Type == "text" {
						resultText = c.Text
						break
					}
				}
			}

			var actionResult struct {
				Visible        bool   `json:"visible"`
				Stable         bool   `json:"stable"`
				ReceivesEvents bool   `json:"receivesEvents"`
				Enabled        bool   `json:"enabled"`
				Editable       bool   `json:"editable"`
				Error          string `json:"error,omitempty"`
			}
			if err := json.Unmarshal([]byte(resultText), &actionResult); err != nil {
				printError(fmt.Errorf("failed to parse actionability result: %w", err))
				return
			}
			if actionResult.Error != "" {
				printError(fmt.Errorf("%s", actionResult.Error))
				return
			}

			// The five booleans are the result; a string would lose them.
			// Struct-valued like paths --json (#392).
			if jsonOutput {
				printJSON(jsonEnvelope{OK: true, Result: actionResult})
				return
			}

			printCheck("Visible", actionResult.Visible)
			printCheck("Stable", actionResult.Stable)
			printCheck("ReceivesEvents", actionResult.ReceivesEvents)
			printCheck("Enabled", actionResult.Enabled)
			printCheck("Editable", actionResult.Editable)
		},
	}

	for _, c := range []*cobra.Command{visibleCmd, enabledCmd, checkedCmd} {
		c.Flags().Bool("fail", false, "Exit non-zero when the answer is false")
	}
	isCmd.AddCommand(visibleCmd)
	isCmd.AddCommand(enabledCmd)
	isCmd.AddCommand(checkedCmd)
	isCmd.AddCommand(actionableCmd)
	return isCmd
}

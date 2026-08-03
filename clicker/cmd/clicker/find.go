package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func newFindCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "find [selector]",
		Short: "Find elements by CSS selector or semantic locator",
		Example: `  vibium find "a"
  # → @e1 [a] "More information..."

  vibium find "a" --all
  # → @e1 [a] "Home"  @e2 [a] "About"  ...

  vibium find text "Sign In"
  # → @e1 [button] "Sign In"

  vibium find role button
  # → @e1 [button] "Submit"

  vibium find role heading --name "Example"
  # Find heading with accessible name "Example"`,
		Args: cobra.RangeArgs(0, 2),
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				fmt.Fprintf(os.Stderr, "Error: requires a CSS selector or use a subcommand (text, role, label, etc.)\n")
				os.Exit(1)
			}

			all, _ := cmd.Flags().GetBool("all")

			toolArgs := map[string]interface{}{}

			if len(args) == 2 && isURL(args[0]) {
				_, err := daemonCall("browser_navigate", map[string]interface{}{"url": args[0]})
				if err != nil {
					printError(err)
					return
				}
				toolArgs["selector"] = args[1]
			} else {
				toolArgs["selector"] = args[0]
			}

			if all {
				limit, _ := cmd.Flags().GetInt("limit")
				toolArgs["limit"] = float64(limit)
				result, err := daemonCall("browser_find_all", toolArgs)
				if err != nil {
					printError(err)
					return
				}
				printResult(result)
				return
			}

			result, err := daemonCall("browser_find", toolArgs)
			if err != nil {
				printError(err)
				return
			}
			printResult(result)
		},
	}

	cmd.Flags().Bool("all", false, "Find all matching elements")
	cmd.Flags().Int("limit", 10, "Maximum number of elements to return (with --all)")

	// Semantic locator subcommands
	textCmd := &cobra.Command{
		Use:   "text [text]",
		Short: "Find element by text content",
		Example: `  vibium find text "Sign In"
  # → @e1 [button] "Sign In"`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			result, err := daemonCall("browser_find", map[string]interface{}{"text": args[0]})
			if err != nil {
				printError(err)
				return
			}
			printResult(result)
		},
	}

	roleCmd := &cobra.Command{
		Use:   "role [role]",
		Short: "Find element by ARIA role",
		Example: `  vibium find role button
  # → @e1 [button] "Submit"

  vibium find role heading --name "Example"
  # Find heading with accessible name "Example"`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			toolArgs := map[string]interface{}{"role": args[0]}
			name, _ := cmd.Flags().GetString("name")
			if name != "" {
				// "label" is the accessible-name filter; "text" is a raw
				// textContent match, which is "" for a void element like
				// <input type="submit">.
				toolArgs["label"] = name
			}
			result, err := daemonCall("browser_find", toolArgs)
			if err != nil {
				printError(err)
				return
			}
			printResult(result)
		},
	}
	roleCmd.Flags().String("name", "", "Accessible name filter")

	labelCmd := &cobra.Command{
		Use:   "label [label]",
		Short: "Find input by associated label text",
		Example: `  vibium find label "Email"
  # → @e1 [input type="email"] placeholder="Email"`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			result, err := daemonCall("browser_find", map[string]interface{}{"label": args[0]})
			if err != nil {
				printError(err)
				return
			}
			printResult(result)
		},
	}

	placeholderCmd := &cobra.Command{
		Use:   "placeholder [placeholder]",
		Short: "Find element by placeholder attribute",
		Example: `  vibium find placeholder "Search..."
  # → @e1 [input] placeholder="Search..."`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			result, err := daemonCall("browser_find", map[string]interface{}{"placeholder": args[0]})
			if err != nil {
				printError(err)
				return
			}
			printResult(result)
		},
	}

	testidCmd := &cobra.Command{
		Use:   "testid [testid]",
		Short: "Find element by data-testid attribute",
		Example: `  vibium find testid "submit-btn"
  # → @e1 [button] data-testid="submit-btn"`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			result, err := daemonCall("browser_find", map[string]interface{}{"testid": args[0]})
			if err != nil {
				printError(err)
				return
			}
			printResult(result)
		},
	}

	xpathCmd := &cobra.Command{
		Use:   "xpath [expression]",
		Short: "Find element by XPath expression",
		Example: `  vibium find xpath "//div[@class='main']"
  # → @e1 [div.main] ...`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			result, err := daemonCall("browser_find", map[string]interface{}{"xpath": args[0]})
			if err != nil {
				printError(err)
				return
			}
			printResult(result)
		},
	}

	altCmd := &cobra.Command{
		Use:   "alt [alt]",
		Short: "Find element by alt attribute",
		Example: `  vibium find alt "Logo"`,
		Args:    cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			result, err := daemonCall("browser_find", map[string]interface{}{"alt": args[0]})
			if err != nil {
				printError(err)
				return
			}
			printResult(result)
		},
	}

	titleCmd := &cobra.Command{
		Use:   "title [title]",
		Short: "Find element by title attribute",
		Example: `  vibium find title "Close"`,
		Args:    cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			result, err := daemonCall("browser_find", map[string]interface{}{"title": args[0]})
			if err != nil {
				printError(err)
				return
			}
			printResult(result)
		},
	}

	cmd.AddCommand(textCmd)
	cmd.AddCommand(roleCmd)
	cmd.AddCommand(labelCmd)
	cmd.AddCommand(placeholderCmd)
	cmd.AddCommand(testidCmd)
	cmd.AddCommand(xpathCmd)
	cmd.AddCommand(altCmd)
	cmd.AddCommand(titleCmd)

	return cmd
}

// isURL returns true if the string looks like a URL (starts with http:// or https://).
func isURL(s string) bool {
	return len(s) > 8 && (s[:7] == "http://" || s[:8] == "https://")
}

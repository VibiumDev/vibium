package main

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
)

func newTypeCmd() *cobra.Command {
	var timeout time.Duration
	cmd := &cobra.Command{
		Use:   "type [url] [selector] [text]",
		Short: "Type text into an element, or into whatever has focus",
		Example: `  vibium type "calc"
  # Types into whatever has focus: a search box, or a phone's screen

  vibium type "input" "12345"
  # Types into the element on the current page

  vibium type https://the-internet.herokuapp.com/inputs "input" "12345"
  # Navigates to URL first, then types

  vibium type https://the-internet.herokuapp.com/inputs "input" "12345" --timeout 5s
  # Custom timeout (5s, or 5000 for milliseconds)`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) < 1 || len(args) > 3 {
				fmt.Fprintf(os.Stderr, "Error: accepts between 1 and 3 arg(s), received %d\n", len(args))
				os.Exit(1)
			}
			var selector, text string
			switch len(args) {
			case 3:
				// type <url> <selector> <text> — navigate first
				_, err := daemonCall("browser_navigate", map[string]interface{}{"url": args[0]})
				if err != nil {
					printError(err)
					return
				}
				selector = args[1]
				text = args[2]
			case 2:
				// type <selector> <text> — current page
				selector = args[0]
				text = args[1]
			default:
				// type <text> — whatever has focus
				text = args[0]
			}

			toolArgs := map[string]interface{}{
				"text":    text,
				"timeout": float64(timeout.Milliseconds()),
			}
			if selector != "" {
				toolArgs["selector"] = selector
			}
			result, err := daemonCall("browser_type", toolArgs)
			if err != nil {
				printError(err)
				return
			}
			printResult(result)
		},
	}
	addTimeoutFlag(cmd, &timeout)
	// Values like type "#x" "-2" must not be parsed as shorthand flags (#179).
	return lateParse(cmd)
}

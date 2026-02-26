package main

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/vibium/clicker/internal/bidi"
	"github.com/vibium/clicker/internal/browser"
	"github.com/vibium/clicker/internal/features"
	"github.com/vibium/clicker/internal/process"
)

func newCheckActionableCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "check-actionable [url] [selector]",
		Short: "Check actionability of an element (Visible, Stable, ReceivesEvents, Enabled, Editable)",
		Example: `  vibium check-actionable https://example.com "a"
  # Output:
  # Checking actionability for selector: a
  # ✓ Visible: true
  # ✓ Stable: true
  # ✓ ReceivesEvents: true
  # ✓ Enabled: true
  # ✗ Editable: false`,
		Args: cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			process.WithCleanup(func() {
				url := args[0]
				selector := args[1]

				fmt.Println("Launching browser...")
				launchResult, err := browser.Launch(browser.LaunchOptions{Headless: headless})
				if err != nil {
					fatalExit("Error launching browser: %v", err)
				}
				defer waitAndClose(launchResult)

				fmt.Println("Connecting to BiDi...")
				conn, err := bidi.Connect(launchResult.WebSocketURL)
				if err != nil {
					fatalExit("Error connecting: %v", err)
				}
				defer conn.Close()

				client := bidi.NewClient(conn)

				fmt.Printf("Navigating to %s...\n", url)
				_, err = client.Navigate("", url)
				if err != nil {
					fatalExit("Error navigating: %v", err)
				}

				doWaitOpen()

				fmt.Printf("\nChecking actionability for selector: %s\n", selector)

				result, err := features.CheckAll(client, "", selector)
				if err != nil {
					fatalExit("Error: %v", err)
				}

				// Print results with checkmarks
				printCheck("Visible", result.Visible)
				printCheck("Stable", result.Stable)
				printCheck("ReceivesEvents", result.ReceivesEvents)
				printCheck("Enabled", result.Enabled)
				printCheck("Editable", result.Editable)
			})
		},
	}
}

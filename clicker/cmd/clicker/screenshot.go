package main

import (
	"encoding/base64"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/vibium/clicker/internal/bidi"
	"github.com/vibium/clicker/internal/browser"
	"github.com/vibium/clicker/internal/process"
)

func newScreenshotCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "screenshot [url]",
		Short: "Capture a screenshot (optionally navigate to URL first)",
		Example: `  vibium screenshot -o shot.png
  # Screenshots the current page (daemon mode)

  vibium screenshot https://example.com -o shot.png
  # Navigates to URL first, then screenshots

  vibium screenshot -o full.png --full-page
  # Capture the entire page (not just the viewport)`,
		Args: cobra.RangeArgs(0, 1),
		Run: func(cmd *cobra.Command, args []string) {
			output, _ := cmd.Flags().GetString("output")
			fullPage, _ := cmd.Flags().GetBool("full-page")
			annotate, _ := cmd.Flags().GetBool("annotate")

			// Daemon mode
			if !oneshot {
				// Navigate first if URL provided
				if len(args) == 1 {
					_, err := daemonCall("browser_navigate", map[string]interface{}{"url": args[0]})
					if err != nil {
						printError(err)
						return
					}
				}

				// Take screenshot with filename
				screenshotArgs := map[string]interface{}{"filename": output}
				if fullPage {
					screenshotArgs["fullPage"] = true
				}
				if annotate {
					screenshotArgs["annotate"] = true
				}
				result, err := daemonCall("browser_screenshot", screenshotArgs)
				if err != nil {
					printError(err)
					return
				}
				printResult(result)
				return
			}

			// Oneshot mode (original behavior) — requires URL
			if len(args) < 1 {
				fmt.Fprintf(os.Stderr, "Error: requires [url] in oneshot mode\n")
				os.Exit(1)
			}
			url := args[0]
			process.WithCleanup(func() {
				fmt.Println("Launching browser...")
				launchResult, err := browser.Launch(browser.LaunchOptions{Headless: headless})
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error launching browser: %v\n", err)
					os.Exit(1)
				}
				defer waitAndClose(launchResult)

				fmt.Println("Connecting to BiDi...")
				conn, err := bidi.Connect(launchResult.WebSocketURL)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error connecting: %v\n", err)
					os.Exit(1)
				}
				defer conn.Close()

				client := bidi.NewClient(conn)

				fmt.Printf("Navigating to %s...\n", url)
				_, err = client.Navigate("", url)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error navigating: %v\n", err)
					os.Exit(1)
				}

				doWaitOpen()

				fmt.Println("Capturing screenshot...")
				var base64Data string
				var captureErr error
				if fullPage {
					base64Data, captureErr = client.CaptureFullPageScreenshot("")
				} else {
					base64Data, captureErr = client.CaptureScreenshot("")
				}
				err = captureErr
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error capturing screenshot: %v\n", err)
					os.Exit(1)
				}

				// Decode base64 to PNG bytes
				pngData, err := base64.StdEncoding.DecodeString(base64Data)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error decoding screenshot: %v\n", err)
					os.Exit(1)
				}

				// Save to file
				if err := os.WriteFile(output, pngData, 0644); err != nil {
					fmt.Fprintf(os.Stderr, "Error saving screenshot: %v\n", err)
					os.Exit(1)
				}

				fmt.Printf("Screenshot saved to %s (%d bytes)\n", output, len(pngData))
			})
		},
	}
	cmd.Flags().StringP("output", "o", "screenshot.png", "Output file path")
	cmd.Flags().Bool("full-page", false, "Capture the full page instead of just the viewport")
	cmd.Flags().Bool("annotate", false, "Annotate interactive elements with numbered labels")
	return cmd
}

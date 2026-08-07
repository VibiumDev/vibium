package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/vibium/clicker/internal/bidi"
	"github.com/vibium/clicker/internal/browser"
	"github.com/vibium/clicker/internal/paths"
	"github.com/vibium/clicker/internal/process"
)

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the version number",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("%s v%s\n", filepath.Base(os.Args[0]), version)
		},
	}
}

func newPathsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "paths",
		Short: "Print browser and cache paths",
		Run: func(cmd *cobra.Command, args []string) {
			cacheDir, err := paths.GetCacheDir()
			if err != nil {
				fmt.Printf("Cache directory: error: %v\n", err)
			} else {
				fmt.Printf("Cache directory: %s\n", cacheDir)
			}

			chromePath, err := paths.GetChromeExecutable()
			if err != nil {
				fmt.Println("Chrome: not found")
			} else {
				fmt.Printf("Chrome: %s\n", chromePath)
			}

			chromedriverPath, err := paths.GetChromedriverPath()
			if err != nil {
				fmt.Println("Chromedriver: not found")
			} else {
				fmt.Printf("Chromedriver: %s\n", chromedriverPath)
			}
		},
	}
}

func newIsInstalledCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "is-installed",
		Short: "Check if the selected browser is installed (exit 0 = yes, exit 1 = no)",
		Run: func(cmd *cobra.Command, args []string) {
			if engineName == "firefox" {
				if !browser.IsFirefoxInstalled() {
					os.Exit(1)
				}
				return
			}
			if !browser.IsInstalled() {
				os.Exit(1)
			}
		},
	}
}

func newInstallCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "install",
		Short: "Download the selected browser (Chrome for Testing by default)",
		Example: `  vibium install
  # Installing Chrome for Testing v139.0.7258.68...

  vibium install --engine firefox
  # Installing Firefox v153.0.3 (release channel)...

  vibium install --engine firefox --firefox-channel beta
  # Installing Firefox v154.0b6 (beta channel)...`,
		Run: func(cmd *cobra.Command, args []string) {
			if engineName == "firefox" {
				exePath, err := browser.InstallFirefox()
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error: %v\n", err)
					os.Exit(1)
				}
				fmt.Println("Installation complete!")
				fmt.Printf("Firefox: %s\n", exePath)
				return
			}

			result, err := browser.Install()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}

			fmt.Println("Installation complete!")
			fmt.Printf("Chrome: %s\n", result.ChromePath)
			fmt.Printf("Chromedriver: %s\n", result.ChromedriverPath)
			fmt.Printf("Version: %s\n", result.Version)
		},
	}
}

func newLaunchTestCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "launch-test",
		Short: "Launch the selected browser and print BiDi session info",
		Run: func(cmd *cobra.Command, args []string) {
			result, err := browser.Launch(browser.LaunchOptions{Engine: engineName, Headless: headless})
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}

			fmt.Printf("Session ID: %s\n", result.SessionID)
			fmt.Printf("BiDi WebSocket: %s\n", result.WebSocketURL)
			fmt.Println("Press Ctrl+C to stop...")

			// Wait for signal, then cleanup
			process.WaitForSignal()
			result.Close()
		},
	}
}

func newWSTestCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "ws-test [url]",
		Short: "Test WebSocket connection (type messages, see echoes)",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			url := args[0]
			if strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://") {
				ws := strings.Replace(strings.Replace(url, "https://", "wss://", 1), "http://", "ws://", 1)
				fmt.Fprintf(os.Stderr, "Error: %s is not a WebSocket URL. Try: %s\n", url, ws)
				os.Exit(1)
			}
			fmt.Printf("Connecting to %s...\n", url)

			conn, err := bidi.Connect(url)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			defer conn.Close()

			fmt.Println("Connected! Type messages (Ctrl+C to quit):")

			// Read responses in background
			go func() {
				for {
					msg, err := conn.Receive()
					if err != nil {
						return
					}
					fmt.Printf("< %s\n", msg)
				}
			}()

			// Read input and send
			scanner := bufio.NewScanner(os.Stdin)
			for scanner.Scan() {
				msg := scanner.Text()
				if err := conn.Send(msg); err != nil {
					fmt.Fprintf(os.Stderr, "Send error: %v\n", err)
					break
				}
				fmt.Printf("> %s\n", msg)
			}
		},
	}
}

func newBiDiTestCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "bidi-test",
		Short: "Launch browser, connect via BiDi, send session.status",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("[1/5] Launching chromedriver...")
			launchResult, err := browser.Launch(browser.LaunchOptions{Headless: true, Verbose: true})
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error launching browser: %v\n", err)
				os.Exit(1)
			}
			defer launchResult.Close()
			fmt.Printf("       Chromedriver started on port %d\n", launchResult.Port)
			fmt.Printf("       Session ID: %s\n", launchResult.SessionID)

			fmt.Println("[2/5] WebDriver session created with BiDi enabled")

			// The BiDi-first launch path hands back the open connection and
			// leaves WebSocketURL empty; only the HTTP fallback needs a dial.
			var conn *bidi.Connection
			if launchResult.BidiConn != nil {
				fmt.Println("[3/5] Reusing BiDi WebSocket from launch...")
				conn = launchResult.BidiConn
			} else {
				fmt.Printf("       WebSocket URL: %s\n", launchResult.WebSocketURL)
				fmt.Println("[3/5] Connecting to BiDi WebSocket...")
				var err error
				conn, err = bidi.Connect(launchResult.WebSocketURL)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error connecting: %v\n", err)
					os.Exit(1)
				}
			}
			defer conn.Close()
			fmt.Println("       Connected!")

			fmt.Println("[4/5] Sending BiDi command: session.status")
			client := bidi.NewClient(conn)
			client.SetVerbose(true)

			status, err := client.SessionStatus()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}

			fmt.Println("[5/5] Parsed response:")
			fmt.Printf("       Ready: %v\n", status.Ready)
			fmt.Printf("       Message: %s\n", status.Message)

			fmt.Println("\nTest complete!")
		},
	}
}

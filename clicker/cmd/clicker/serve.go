package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/vibium/clicker/internal/api"
	"github.com/vibium/clicker/internal/browser"
)

func newServeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Start WebSocket proxy server for browser automation",
		Example: `  vibium serve
  # Starts server on default port 9515, visible browser

  vibium serve --port 8080
  # Starts server on port 8080

  vibium serve --headless
  # Starts server with headless browser`,
		Run: func(cmd *cobra.Command, args []string) {
			port, _ := cmd.Flags().GetInt("port")

			fmt.Printf("Starting Vibium proxy server on port %d...\n", port)

			// Reclaim Chrome profile dirs orphaned by earlier crashed/killed
			// sessions (parallel-safe: the minAge filter skips live siblings).
			browser.CleanupOrphanedBrowserTempDirs(time.Minute)

			// Create router to manage browser sessions
			router := api.NewRouter(engineName, headless, "", nil)

			server := api.NewServer(
				api.WithPort(port),
				api.WithOnConnect(router.OnClientConnect),
				api.WithOnMessage(router.OnClientMessage),
				api.WithOnClose(router.OnClientDisconnect),
			)

			if err := server.Start(); err != nil {
				fmt.Fprintf(os.Stderr, "Error starting server: %v\n", err)
				os.Exit(1)
			}

			fmt.Printf("Server listening on ws://localhost:%d\n", server.Port())
			fmt.Println("Press Ctrl+C to stop...")

			// Wait for signal
			sigCh := make(chan os.Signal, 1)
			signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
			<-sigCh

			fmt.Println("\nShutting down...")

			// Close all browser sessions (thorough kill of chromedriver + Chrome)
			router.CloseAll()

			// Safety net: kill any Chrome/chromedriver processes orphaned by races
			browser.KillOrphanedChromeProcesses()
			browser.KillOrphanedFirefoxProcesses()

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			server.Stop(ctx)
		},
	}
	cmd.Flags().IntP("port", "p", 9515, "Port to listen on")
	return cmd
}

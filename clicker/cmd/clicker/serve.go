package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/vibium/clicker/internal/process"
	"github.com/vibium/clicker/internal/proxy"
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
			process.WithCleanup(func() {
				port, _ := cmd.Flags().GetInt("port")

				fmt.Printf("Starting Vibium proxy server on port %d...\n", port)

				// Create router to manage browser sessions
				router := proxy.NewRouter(headless)

				server := proxy.NewServer(
					proxy.WithPort(port),
					proxy.WithOnConnect(router.OnClientConnect),
					proxy.WithOnMessage(router.OnClientMessage),
					proxy.WithOnClose(router.OnClientDisconnect),
				)

				if err := server.Start(); err != nil {
					fmt.Fprintf(os.Stderr, "Error starting server: %v\n", err)
					os.Exit(1)
				}

				fmt.Printf("Server listening on ws://localhost:%d\n", server.Port())
				fmt.Println("Press Ctrl+C to stop...")

				// Wait for signal
				process.WaitForSignal()

				fmt.Println("\nShutting down...")

				// Close all browser sessions
				router.CloseAll()

				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				server.Stop(ctx)
			})
		},
	}
	cmd.Flags().IntP("port", "p", 9515, "Port to listen on")
	return cmd
}

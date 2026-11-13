package main

import (
	"crypto/rand"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/vibium/clicker/internal/webbotauth"
)

// newWebBotAuthCmd groups the Web Bot Auth key helpers. Signing itself is
// turned on at launch with VIBIUM_WEB_BOT_AUTH_KEY; these subcommands create
// the key and print the directory a site fetches to verify it (#28).
func newWebBotAuthCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "webbotauth",
		Short: "Web Bot Auth key and directory helpers for Cloudflare Verified Bots",
		Long: `Create and inspect the Ed25519 key used for Web Bot Auth request signing.

Signing is enabled at launch, not here: point VIBIUM_WEB_BOT_AUTH_KEY at the
key file (and optionally VIBIUM_WEB_BOT_AUTH_DIRECTORY at the https URL that
serves the directory below), then run vibium normally.`,
	}
	cmd.AddCommand(newWebBotAuthKeygenCmd())
	cmd.AddCommand(newWebBotAuthDirectoryCmd())
	return cmd
}

func newWebBotAuthKeygenCmd() *cobra.Command {
	var out string
	cmd := &cobra.Command{
		Use:   "keygen",
		Short: "Generate an Ed25519 signing key (PKCS#8 PEM)",
		Example: `  vibium webbotauth keygen -o web-bot-auth.pem
  # Wrote web-bot-auth.pem (keyid: poqkLGiymh_W0uP6PZFw-dvez3QJT5SolqXBCW38r0U)`,
		Run: func(cmd *cobra.Command, args []string) {
			pemBytes, err := webbotauth.GenerateKeyPEM(rand.Reader)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			if out == "" {
				os.Stdout.Write(pemBytes)
				return
			}
			if err := os.WriteFile(out, pemBytes, 0o600); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			// Load it back so the printed keyid is the real one.
			signer, err := webbotauth.LoadSigner(out, "")
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("Wrote %s (keyid: %s)\n", out, signer.KeyID())
		},
	}
	cmd.Flags().StringVarP(&out, "output", "o", "", "Write the key to this file (default: stdout)")
	return cmd
}

func newWebBotAuthDirectoryCmd() *cobra.Command {
	var keyPath string
	cmd := &cobra.Command{
		Use:   "directory",
		Short: "Print the JWKS directory to publish at /.well-known/http-message-signatures-directory",
		Example: `  vibium webbotauth directory --key web-bot-auth.pem
  # {"keys":[{"kty":"OKP","crv":"Ed25519","x":"...","kid":"...","nbf":...}]}`,
		Run: func(cmd *cobra.Command, args []string) {
			if keyPath == "" {
				fmt.Fprintln(os.Stderr, "Error: --key is required")
				os.Exit(1)
			}
			signer, err := webbotauth.LoadSigner(keyPath, "")
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			doc, err := signer.Directory(time.Now())
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			fmt.Println(string(doc))
		},
	}
	cmd.Flags().StringVar(&keyPath, "key", "", "Path to the Ed25519 key (PKCS#8 PEM)")
	return cmd
}

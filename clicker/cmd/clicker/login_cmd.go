package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
	"github.com/vibium/clicker/internal/verifier"
)

func newLoginCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "login",
		Short: "Sign in to a model provider",
		Long:  "Sign in so Run and Check can use a provider without an API key in the environment.",
	}
	cmd.AddCommand(newLoginStatusCmd())
	cmd.AddCommand(newLoginXAICmd())
	return cmd
}

func newLoginStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show which xAI credential would be used",
		Long:  "Report the xAI credential Run and Check would use, without printing secrets or refreshing tokens.\nDoes not contact the network or write xai-auth.json / ~/.grok/auth.json.\nXAI_API_KEY wins when set. Otherwise a Vibium or Grok CLI login is reported.",
		Example: `  vibium login status
  # xai: Grok login (Vibium ~/.config/vibium/xai-auth.json)
  vibium login status --json
  # {"ok":true,"result":{"providers":[{"provider":"xai","active":"oauth","api_key":false,"oauth":{"source":"vibium","expired":false}}]}}`,
		Args: cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			code, err := emitLoginStatus(cmd.OutOrStdout())
			if err != nil {
				printError(err)
			}
			if code != 0 {
				os.Exit(code)
			}
		},
	}
}

type loginStatusResult struct {
	Providers []verifier.XAILoginStatus `json:"providers"`
}

func emitLoginStatus(w io.Writer) (int, error) {
	status, err := verifier.ReadXAILoginStatus()
	if err != nil {
		return 1, err
	}
	if jsonOutput {
		data, err := json.Marshal(jsonEnvelope{OK: true, Result: loginStatusResult{Providers: []verifier.XAILoginStatus{status}}})
		if err != nil {
			return 1, err
		}
		fmt.Fprintln(w, string(data))
	} else {
		fmt.Fprint(w, formatXAILoginStatus(status))
	}
	if status.Active == verifier.CredentialNone {
		return 1, nil
	}
	return 0, nil
}

func formatXAILoginStatus(s verifier.XAILoginStatus) string {
	loc := oauthLocation(s)
	switch {
	case s.Active == verifier.CredentialAPIKey:
		return "xai: XAI_API_KEY (active; wins over Grok login)\n"
	case s.Active == verifier.CredentialOAuth:
		extra := ""
		if s.TokenExpiring {
			extra = "; token expiring"
		}
		return fmt.Sprintf("xai: Grok login (%s%s)\n", loc, extra)
	case s.OAuth.Expired:
		return fmt.Sprintf("xai: Grok login expired (%s)\nExport XAI_API_KEY or run: vibium login xai\n", loc)
	default:
		return "xai: not signed in\nExport XAI_API_KEY or run: vibium login xai\n"
	}
}

func oauthLocation(s verifier.XAILoginStatus) string {
	path := tildePath(s.Path)
	switch s.OAuth.Source {
	case "vibium":
		return "Vibium " + path
	case "grok":
		return "Grok CLI " + path
	default:
		return path
	}
}

func newLoginXAICmd() *cobra.Command {
	return &cobra.Command{
		Use:   "xai",
		Short: "Sign in to xAI with a Grok subscription",
		Long:  "Start a device-code login at auth.x.ai. Open the printed URL, enter the code, and Vibium stores tokens in ~/.config/vibium/xai-auth.json.\nDoes not open a browser. Does not read or write ~/.grok/auth.json.\nXAI_API_KEY still wins when set.",
		Example: `  vibium login xai
  # Prints a URL and user code; writes ~/.config/vibium/xai-auth.json (0600) after approval.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			prompt := cmd.OutOrStdout()
			if jsonOutput {
				prompt = cmd.ErrOrStderr()
			}
			ctx := cmd.Context()
			if ctx == nil {
				ctx = context.Background()
			}
			path, err := verifier.LoginXAI(ctx, prompt)
			if err != nil {
				return err
			}
			if jsonOutput {
				printJSON(jsonEnvelope{OK: true, Result: map[string]interface{}{
					"path": path,
					"mode": "0600",
				}})
				return nil
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Wrote %s (0600)\n", tildePath(path))
			return nil
		},
	}
}

func newLogoutCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "logout",
		Short: "Sign out of a model provider",
		Long:  "Remove a Vibium provider login. Does not delete API keys from the environment or ~/.grok/auth.json.",
	}
	cmd.AddCommand(newLogoutXAICmd())
	return cmd
}

func newLogoutXAICmd() *cobra.Command {
	return &cobra.Command{
		Use:     "xai",
		Short:   "Remove the Vibium xAI login",
		Long:    "Delete ~/.config/vibium/xai-auth.json. Leaves ~/.grok/auth.json and XAI_API_KEY alone.",
		Example: "  vibium logout xai\n  # Removes only the Vibium xAI login file.",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := verifier.LogoutXAI(); err != nil {
				return err
			}
			if jsonOutput {
				printJSON(jsonEnvelope{OK: true, Result: map[string]string{"removed": "xai-auth.json"}})
				return nil
			}
			fmt.Fprintln(cmd.OutOrStdout(), "Removed Vibium xAI login.")
			return nil
		},
	}
}

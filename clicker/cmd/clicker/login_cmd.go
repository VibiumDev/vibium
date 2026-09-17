package main

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/vibium/clicker/internal/verifier"
)

func newLoginCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "login",
		Short: "Sign in to a model provider",
		Long:  "Sign in so Run and Check can use a provider without an API key in the environment.",
	}
	cmd.AddCommand(newLoginXAICmd())
	return cmd
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

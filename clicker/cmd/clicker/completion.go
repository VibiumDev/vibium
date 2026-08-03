package main

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

// compinitGuard makes the generated zsh script safe to `source` directly.
// Cobra emits `compdef _vibium vibium`, which is only defined once compinit has
// run — sourcing the script from a shell where it has not fails with
// "compdef: command not found" (issue #201).
const compinitGuard = `(( $+functions[compdef] )) || { autoload -Uz compinit && compinit -u; }`

func newCompletionCmd(rootCmd *cobra.Command) *cobra.Command {
	return &cobra.Command{
		Use:   "completion [bash|zsh|fish|powershell]",
		Short: "Generate a shell completion script",
		Long: `Generate a shell completion script.

zsh — source it directly, or install it on your fpath:

  source <(vibium completion zsh)
  vibium completion zsh > "${fpath[1]}/_vibium"

bash:

  source <(vibium completion bash)`,
		Example: `  vibium completion zsh
  # → #compdef vibium
  #   (( $+functions[compdef] )) || { autoload -Uz compinit && compinit -u; }
  #   compdef _vibium vibium
  #   ...`,
		Args:      cobra.ExactArgs(1),
		ValidArgs: []string{"bash", "zsh", "fish", "powershell"},
		RunE: func(cmd *cobra.Command, args []string) error {
			switch args[0] {
			case "bash":
				return rootCmd.GenBashCompletion(os.Stdout)
			case "fish":
				return rootCmd.GenFishCompletion(os.Stdout, true)
			case "powershell":
				return rootCmd.GenPowerShellCompletionWithDesc(os.Stdout)
			case "zsh":
				var buf bytes.Buffer
				if err := rootCmd.GenZshCompletion(&buf); err != nil {
					return err
				}
				_, err := os.Stdout.WriteString(withCompinitGuard(buf.String()))
				return err
			}
			return fmt.Errorf("unsupported shell %q", args[0])
		},
	}
}

// withCompinitGuard inserts the guard after the leading #compdef directive, so
// an fpath install still sees #compdef on line 1.
func withCompinitGuard(script string) string {
	lines := strings.SplitN(script, "\n", 2)
	if len(lines) < 2 || !strings.HasPrefix(lines[0], "#compdef") {
		return compinitGuard + "\n" + script
	}
	return lines[0] + "\n" + compinitGuard + "\n" + lines[1]
}

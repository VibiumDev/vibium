package main

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/vibium/clicker/internal/paths"
)

//go:embed AI_ENV_TEMPLATE
var aiEnvTemplate string

//go:embed CLOUD_ENV_TEMPLATE
var cloudEnvTemplate string

// tildePath renders a path the way the docs and tutorials write it. Paths
// outside home (VIBIUM_CONFIG_DIR, XDG_CONFIG_HOME) print in full, because a
// ~/ prefix would name a file that isn't there.
func tildePath(path string) string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return path
	}
	if rel, err := filepath.Rel(home, path); err == nil && !strings.HasPrefix(rel, "..") && !filepath.IsAbs(rel) {
		return "~/" + filepath.ToSlash(rel)
	}
	return path
}

// configTemplate is one settings file vibium can lay down for the user.
type configTemplate struct {
	name    string
	file    string
	content *string
	what    string
}

var configTemplates = []configTemplate{
	{"ai", "ai.env", &aiEnvTemplate, "provider, model and API key for run and check"},
	{"cloud", "cloud-browser.env", &cloudEnvTemplate, "credentials for cloud browser vendors"},
}

func newConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage Vibium settings files",
	}
	cmd.AddCommand(newConfigInitCmd())
	return cmd
}

func newConfigInitCmd() *cobra.Command {
	var force, stdout bool

	cmd := &cobra.Command{
		Use:   "init [ai|cloud|all]",
		Short: "Write a starter settings file to ~/.config/vibium",
		Long: `Write a commented settings file to the Vibium config directory.

Every setting is present and documented in place, so the file is its own
reference. Vibium does not load these files automatically — source the one
you want in the shell that runs vibium.`,
		Example: `  vibium config init
  # Wrote ~/.config/vibium/ai.env (0600)

  vibium config init cloud
  # Wrote ~/.config/vibium/cloud-browser.env (0600)

  vibium config init all --force
  # Overwrites both, keeping a .bak of each

  vibium config init ai --stdout
  # Print the template instead of writing it`,
		Args:      cobra.MaximumNArgs(1),
		ValidArgs: []string{"ai", "cloud", "all"},
		RunE: func(cmd *cobra.Command, args []string) error {
			which := "ai"
			if len(args) == 1 {
				which = args[0]
			}
			var chosen []configTemplate
			for _, t := range configTemplates {
				if which == "all" || which == t.name {
					chosen = append(chosen, t)
				}
			}
			if len(chosen) == 0 {
				return fmt.Errorf("unknown settings file %q; choose ai, cloud or all", which)
			}
			if stdout {
				for _, t := range chosen {
					fmt.Fprint(cmd.OutOrStdout(), *t.content)
				}
				return nil
			}
			for _, t := range chosen {
				if err := writeConfigTemplate(cmd, t, force); err != nil {
					return err
				}
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "Overwrite an existing file, keeping a .bak copy")
	cmd.Flags().BoolVar(&stdout, "stdout", false, "Print the template instead of writing it")
	return cmd
}

// writeConfigTemplate lays down one template at 0600. These files hold API
// keys, so the mode is the point: a world-readable copy is how a key leaks to
// anything else running as another user on the machine.
func writeConfigTemplate(cmd *cobra.Command, t configTemplate, force bool) error {
	dir, err := paths.GetConfigDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("could not create %s: %w", dir, err)
	}
	path := filepath.Join(dir, t.file)

	if _, err := os.Stat(path); err == nil {
		if !force {
			// Never clobber real credentials on a bare `config init`.
			return fmt.Errorf("%s already exists; edit it, or pass --force to replace it (a .bak is kept)", path)
		}
		if err := os.Rename(path, path+".bak"); err != nil {
			return fmt.Errorf("could not back up %s: %w", path, err)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Kept the previous file as %s.bak\n", tildePath(path))
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("could not check %s: %w", path, err)
	}

	if err := os.WriteFile(path, []byte(*t.content), 0600); err != nil {
		return fmt.Errorf("could not write %s: %w", path, err)
	}
	// WriteFile respects umask, so an inherited one can widen the mode.
	if err := os.Chmod(path, 0600); err != nil {
		return fmt.Errorf("could not set permissions on %s: %w", path, err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Wrote %s (0600) — %s\n", tildePath(path), t.what)
	fmt.Fprintf(cmd.OutOrStdout(), "Edit it, then: source %s\n", tildePath(path))
	return nil
}

package main

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

//go:embed SKILL.md
var skillMD string

//go:embed VERIFY_SKILL.md
var verifySkillMD string

func newSkillCmd() *cobra.Command {
	var stdout bool

	cmd := &cobra.Command{
		Use:   "add-skill [vibe-check|verify]",
		Short: "Install a Vibium skill for Claude Code",
		Example: `  vibium add-skill
  # Installs skill to ~/.claude/skills/vibe-check/

  vibium add-skill verify
  # Installs skill to ~/.claude/skills/verify/

  vibium add-skill verify --stdout
  # Print skill content to stdout`,
		Args:      cobra.MaximumNArgs(1),
		ValidArgs: []string{"vibe-check", "verify"},
		RunE: func(cmd *cobra.Command, args []string) error {
			name := "vibe-check"
			if len(args) == 1 {
				name = args[0]
			}
			content := skillMD
			switch name {
			case "vibe-check":
			case "verify":
				content = verifySkillMD
			default:
				return fmt.Errorf("unknown skill %q; choose vibe-check or verify", name)
			}
			if stdout {
				fmt.Fprint(cmd.OutOrStdout(), content)
				return nil
			}
			return installSkill(name, content)
		},
	}
	cmd.Flags().BoolVar(&stdout, "stdout", false, "Print skill content to stdout instead of installing")
	return cmd
}

func installSkill(name, content string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("could not find home directory: %w", err)
	}

	skillDir := filepath.Join(home, ".claude", "skills", name)

	if err := os.MkdirAll(skillDir, 0755); err != nil {
		return fmt.Errorf("could not create skill directory: %w", err)
	}

	// Write SKILL.md
	skillPath := filepath.Join(skillDir, "SKILL.md")
	if err := os.WriteFile(skillPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("could not write SKILL.md: %w", err)
	}

	fmt.Printf("Installed Vibium skill to %s\n", skillDir)
	fmt.Println("Files:")
	fmt.Printf("  %s\n", skillPath)
	return nil
}

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

func newSkillCmd() *cobra.Command {
	var stdout bool

	cmd := &cobra.Command{
		Use:   "add-skill",
		Short: "Install Vibium browser skill for Claude Code",
		Example: `  vibium add-skill
  # Installs skill to ~/.claude/skills/vibe-check/

  vibium add-skill --stdout
  # Print skill content to stdout`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if stdout {
				fmt.Print(skillMD)
				return nil
			}
			return installSkill()
		},
	}
	cmd.Flags().BoolVar(&stdout, "stdout", false, "Print skill content to stdout instead of installing")
	return cmd
}

func installSkill() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("could not find home directory: %w", err)
	}

	skillDir := filepath.Join(home, ".claude", "skills", "vibe-check")

	if err := os.MkdirAll(skillDir, 0755); err != nil {
		return fmt.Errorf("could not create skill directory: %w", err)
	}

	// Write SKILL.md
	skillPath := filepath.Join(skillDir, "SKILL.md")
	if err := os.WriteFile(skillPath, []byte(skillMD), 0644); err != nil {
		return fmt.Errorf("could not write SKILL.md: %w", err)
	}

	fmt.Printf("Installed Vibium skill to %s\n", skillDir)
	fmt.Println("Files:")
	fmt.Printf("  %s\n", skillPath)
	return nil
}

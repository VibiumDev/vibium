package main

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

//go:embed SKILL.md
var skillMD string

//go:embed CHECK_SKILL.md
var checkSkillMD string

func newSkillCmd() *cobra.Command {
	var stdout bool
	var agent string

	cmd := &cobra.Command{
		Use:   "add-skill [browser|check]",
		Short: "Install a Vibium skill for Grok and Claude Code",
		Example: `  vibium add-skill
  # Installs for each agent already present (~/.claude, ~/.grok, or $GROK_HOME)

  vibium add-skill check --agent all
  # Installs the check skill for both agents, creating their directories

  vibium add-skill --agent grok
  # Grok only ($GROK_HOME/skills or ~/.grok/skills)

  vibium add-skill check --stdout
  # Print skill content to stdout`,
		Args:      cobra.MaximumNArgs(1),
		ValidArgs: []string{"browser", "check"},
		RunE: func(cmd *cobra.Command, args []string) error {
			name := "browser"
			if len(args) == 1 {
				name = args[0]
			}
			content := skillMD
			switch name {
			case "browser":
			case "check":
				content = checkSkillMD
			default:
				return fmt.Errorf("unknown skill %q; choose browser or check", name)
			}
			if stdout {
				fmt.Fprint(cmd.OutOrStdout(), content)
				return nil
			}
			return installSkill(name, content, agent)
		},
	}
	cmd.Flags().BoolVar(&stdout, "stdout", false, "Print skill content to stdout instead of installing")
	cmd.Flags().StringVar(&agent, "agent", "auto", "Install for claude, grok, all, or auto (agents already present)")
	return cmd
}

func installSkill(name, content, agent string) error {
	targets, err := skillInstallDirs(name, agent)
	if err != nil {
		return err
	}
	var dirs, files []string
	for _, skillDir := range targets {
		if err := os.MkdirAll(skillDir, 0755); err != nil {
			return fmt.Errorf("could not create skill directory: %w", err)
		}
		skillPath := filepath.Join(skillDir, "SKILL.md")
		if err := os.WriteFile(skillPath, []byte(content), 0644); err != nil {
			return fmt.Errorf("could not write SKILL.md: %w", err)
		}
		dirs = append(dirs, skillDir)
		files = append(files, skillPath)
	}

	if jsonOutput {
		printJSON(jsonEnvelope{OK: true, Result: map[string]interface{}{
			"skill": name,
			"dir":   dirs[0],
			"dirs":  dirs,
			"files": files,
		}})
		return nil
	}

	fmt.Println("Installed Vibium skill to:")
	for _, dir := range dirs {
		fmt.Printf("  %s\n", dir)
	}
	fmt.Println("Files:")
	for _, file := range files {
		fmt.Printf("  %s\n", file)
	}
	return nil
}

func skillInstallDirs(name, agent string) ([]string, error) {
	var agents []string
	switch agent {
	case "", "auto":
		agents = detectedSkillAgents()
	case "all":
		agents = []string{"claude", "grok"}
	case "claude", "grok":
		agents = []string{agent}
	default:
		return nil, fmt.Errorf("unknown agent %q; choose claude, grok, all, or auto", agent)
	}
	var dirs []string
	for _, a := range agents {
		dir, err := agentSkillDir(a, name)
		if err != nil {
			return nil, err
		}
		dirs = append(dirs, dir)
	}
	return dirs, nil
}

func agentSkillDir(agent, name string) (string, error) {
	switch agent {
	case "claude":
		return claudeSkillDir(name)
	case "grok":
		return grokSkillDir(name)
	default:
		return "", fmt.Errorf("unknown agent %q; choose claude or grok", agent)
	}
}

// presentSkillAgents lists agents already set up on this machine, in the
// order --agent all installs. A set GROK_HOME counts as Grok even before
// its directory exists; explicit configuration is intent.
func presentSkillAgents() []string {
	var agents []string
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	if isDir(filepath.Join(home, ".claude")) {
		agents = append(agents, "claude")
	}
	if strings.TrimSpace(os.Getenv("GROK_HOME")) != "" || isDir(filepath.Join(home, ".grok")) {
		agents = append(agents, "grok")
	}
	return agents
}

// detectedSkillAgents is presentSkillAgents with the fresh-machine
// fallback: no agent directories means Claude, the pre-flag behavior.
func detectedSkillAgents() []string {
	if agents := presentSkillAgents(); len(agents) > 0 {
		return agents
	}
	return []string{"claude"}
}

func claudeSkillDir(name string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("could not find home directory: %w", err)
	}
	return filepath.Join(home, ".claude", "skills", name), nil
}

func grokSkillDir(name string) (string, error) {
	if root := strings.TrimSpace(os.Getenv("GROK_HOME")); root != "" {
		return filepath.Join(root, "skills", name), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("could not find home directory: %w", err)
	}
	return filepath.Join(home, ".grok", "skills", name), nil
}

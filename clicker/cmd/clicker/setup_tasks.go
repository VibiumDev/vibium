package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/vibium/clicker/internal/paths"
)

func setupTasks(cmd *cobra.Command, ui *setupUI, quick bool) setupSection {
	path, err := linearEnvPath()
	if err != nil {
		return setupSection{Name: "tasks", Status: "failed", Message: err.Error()}
	}
	shown := tildePath(path)
	if quick && linearConfigPresent() {
		ui.skip("Linear settings are already present; skipping (--quick).")
		return setupSection{Name: "tasks", Status: "skipped", Message: "Linear settings are already present.", Path: path}
	}
	if !ui.interactive {
		if _, err := os.Stat(path); err == nil {
			ui.skip("Left existing %s unchanged (non-interactive).", shown)
			return setupSection{Name: "tasks", Status: "skipped", Message: "Existing linear.env was left unchanged.", Path: path}
		}
		ui.skip("Skipped Linear (optional). Re-run vibium setup tasks in a terminal to file Run results.")
		return setupSection{Name: "tasks", Status: "skipped", Message: "Linear is optional; skipped without a terminal."}
	}

	ui.println("%s", maybePaint(ui.useColor(), brandText, "Task management is optional. Linear files Run and Check results as signed issues."))
	ok, err := ui.confirm("Configure Linear?", false)
	if err != nil {
		return setupSection{Name: "tasks", Status: "failed", Message: err.Error()}
	}
	if !ok {
		ui.skip("Skipped Linear.")
		return setupSection{Name: "tasks", Status: "skipped", Message: "Linear declined."}
	}

	key := firstNonEmpty(os.Getenv("VIBIUM_LINEAR_API_KEY"), os.Getenv("LINEAR_API_KEY"))
	label := "LINEAR_API_KEY"
	if key != "" {
		label += " [saved]"
	}
	entered, err := ui.promptSecret(label)
	if err != nil {
		return setupSection{Name: "tasks", Status: "failed", Message: err.Error()}
	}
	if entered != "" {
		key = entered
	}
	if strings.TrimSpace(key) == "" {
		return setupSection{Name: "tasks", Status: "failed", Message: "A Linear API key is required to configure tasks."}
	}

	team := os.Getenv("VIBIUM_LINEAR_TEAM")
	team, err = ui.prompt("Linear team key (ENG, RAV, …)", team)
	if err != nil {
		return setupSection{Name: "tasks", Status: "failed", Message: err.Error()}
	}
	team = strings.TrimSpace(strings.ToUpper(team))
	if team == "" {
		return setupSection{Name: "tasks", Status: "failed", Message: "A Linear team key is required."}
	}

	kv := map[string]string{
		"VIBIUM_LINEAR_API_KEY": key,
		"VIBIUM_LINEAR_TEAM":    team,
	}
	if err := writeLinearEnv(path, kv); err != nil {
		return setupSection{Name: "tasks", Status: "failed", Message: err.Error()}
	}
	for k, v := range kv {
		_ = os.Setenv(k, v)
	}
	ui.ok("Wrote %s (0600). Issues will be signed by Vibium with artifact paths.", shown)
	return setupSection{Name: "tasks", Status: "done", Message: "Wrote Linear settings.", Path: path}
}

func linearEnvPath() (string, error) {
	dir, err := paths.GetConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "linear.env"), nil
}

func linearConfigPresent() bool {
	path, err := linearEnvPath()
	if err != nil {
		return false
	}
	_, err = os.Stat(path)
	return err == nil
}

func writeLinearEnv(path string, kv map[string]string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("could not create %s: %w", dir, err)
	}
	if _, err := os.Stat(path); err == nil {
		if err := os.Rename(path, path+".bak"); err != nil {
			return fmt.Errorf("could not back up %s: %w", path, err)
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("could not check %s: %w", path, err)
	}
	var b strings.Builder
	b.WriteString("# Written by vibium setup tasks. Mode 0600.\n")
	for _, k := range []string{"VIBIUM_LINEAR_API_KEY", "LINEAR_API_KEY", "VIBIUM_LINEAR_TEAM", "VIBIUM_LINEAR_TEAM_ID"} {
		if v, ok := kv[k]; ok {
			b.WriteString("export " + k + "=" + shellSingleQuote(v) + "\n")
		}
	}
	if err := os.WriteFile(path, []byte(b.String()), 0o600); err != nil {
		return fmt.Errorf("could not write %s: %w", path, err)
	}
	if err := os.Chmod(path, 0o600); err != nil {
		return fmt.Errorf("could not set permissions on %s: %w", path, err)
	}
	return nil
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

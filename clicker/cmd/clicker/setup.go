package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/vibium/clicker/internal/browser"
	"github.com/vibium/clicker/internal/paths"
	"github.com/vibium/clicker/internal/verifier"
)

var setupProviders = []string{"openai", "xai", "anthropic", "google", "openai-compatible", "local"}

type setupSection struct {
	Name    string   `json:"name"`
	Status  string   `json:"status"`
	Message string   `json:"message"`
	Path    string   `json:"path,omitempty"`
	Paths   []string `json:"paths,omitempty"`
	Agent   string   `json:"agent,omitempty"`
	Agents  []string `json:"agents,omitempty"`
}

type setupCommandResult struct {
	Sections []setupSection `json:"sections"`
	Ready    *setupResult   `json:"ready,omitempty"`
}

func newSetupCmd() *cobra.Command {
	var nonInteractive, quick bool
	cmd := &cobra.Command{
		Annotations: map[string]string{"standalone": "true"},
		Use:         "setup [browser|ai|skills]",
		Short:       "Configure browser, AI, and agent skills",
		Long: `Interactive setup for AI settings, agent skills, and the local browser.

Sections can be run on their own: vibium setup browser|ai|skills.
Prompts come first; the browser install runs after them.
--non-interactive never prompts (CI, agents).
--quick fills only what is missing.
After the selected sections, setup runs the matching vibium ready checks.
This process uses the AI values immediately; later commands load empty AI
variables from ai.env automatically.`,
		Example: `  vibium setup
  # AI, skills, browser, then readiness. Prompts on a terminal.

  vibium setup --non-interactive
  # Never prompts; skips AI questions and leaves existing files alone.

  vibium setup --quick
  # Only fill what is missing.

  vibium setup ai
  # Provider, model, and API key only.`,
		Args:      cobra.MaximumNArgs(1),
		ValidArgs: []string{"browser", "ai", "skills"},
		Run: func(cmd *cobra.Command, args []string) {
			section := ""
			if len(args) == 1 {
				section = args[0]
			}
			runSetup(cmd, section, nonInteractive, quick)
		},
	}
	cmd.Flags().BoolVar(&nonInteractive, "non-interactive", false, "Never prompt")
	cmd.Flags().BoolVar(&quick, "quick", false, "Only fill what is missing")
	return cmd
}

func runSetup(cmd *cobra.Command, section string, nonInteractive, quick bool) {
	// Prompts require a real terminal; --non-interactive and --json never
	// prompt even on one, so agent harnesses with a PTY cannot hang here.
	interactive := !jsonOutput && !nonInteractive && inputIsTTY(cmd)
	ui := newSetupUI(cmd, interactive)
	ui.banner()

	want := map[string]bool{"browser": true, "ai": true, "skills": true}
	if section != "" {
		if section != "browser" && section != "ai" && section != "skills" {
			err := fmt.Errorf("unknown setup section %q; choose browser, ai, or skills", section)
			if jsonOutput {
				printJSON(jsonEnvelope{OK: false, Error: err.Error()})
			} else {
				fmt.Fprintln(cmd.ErrOrStderr(), err)
			}
			os.Exit(1)
		}
		want = map[string]bool{section: true}
	}

	// Prompt-driven sections run first so all questions land before the
	// browser download; the user can leave once the prompts are done.
	result := setupCommandResult{}
	// A cancelled section means the user hit Ctrl-C in a prompt: stop the
	// wizard quietly instead of reporting a failure.
	addSection := func(s setupSection) {
		result.Sections = append(result.Sections, s)
		if s.Status == "cancelled" {
			fmt.Fprintln(cmd.ErrOrStderr(), "Setup cancelled.")
			os.Exit(1)
		}
	}
	if want["ai"] {
		ui.heading("AI")
		addSection(setupAI(cmd, ui, quick))
	}
	if want["skills"] {
		ui.heading("Skills")
		addSection(setupSkills(cmd, ui, quick))
	}
	if want["browser"] {
		ui.heading("Browser")
		addSection(setupBrowser(cmd, ui, quick))
	}

	readyScope := "all"
	switch section {
	case "browser":
		readyScope = "browser"
	case "ai":
		readyScope = "ai"
	case "skills":
		readyScope = ""
	}
	if readyScope != "" {
		ready := runSetupReadiness(cmd, readyScope)
		result.Ready = &ready
		if !jsonOutput {
			writeReadiness(cmd, ready)
			if hint := setupTryHint(&ready); hint != "" {
				ui.println("")
				ui.ok("%s", hint)
			}
		}
	}

	ok := true
	var failed []string
	for _, s := range result.Sections {
		if s.Status == "failed" {
			ok = false
			failed = append(failed, s.Name)
		}
	}
	if jsonOutput {
		env := jsonEnvelope{OK: ok, Result: result}
		if !ok {
			env.Error = "Setup failed: " + strings.Join(failed, ", ")
		}
		printJSON(env)
	} else if !ok {
		fmt.Fprintf(cmd.ErrOrStderr(), "Setup failed: %s\n", strings.Join(failed, ", "))
	}
	if !ok {
		os.Exit(1)
	}
}

func setupBrowser(cmd *cobra.Command, ui *setupUI, quick bool) setupSection {
	engine := engineName
	if engine == "" {
		engine = defaultEngine()
	}
	if engine != "chrome" && engine != "firefox" {
		return setupSection{Name: "browser", Status: "failed", Message: fmt.Sprintf("unsupported engine %q (supported: chrome, firefox)", engine)}
	}
	if quick && browser.EngineInstalled(engine) {
		ui.skip("%s is already installed; skipping (--quick).", engine)
		return setupSection{Name: "browser", Status: "skipped", Message: engine + " is already installed."}
	}
	if browser.EngineInstalled(engine) {
		ui.ok("%s is already installed.", engine)
		part := checkBrowserSetup(cmd, nil)
		if !part.Ready {
			return setupSection{Name: "browser", Status: "failed", Message: browserSectionMessage(part)}
		}
		path := selectedBrowserPath(part)
		return setupSection{Name: "browser", Status: "done", Message: engine + " is already installed.", Path: path}
	}
	ui.println("Installing %s...", engine)
	var installPath string
	var err error
	if engine == "firefox" {
		installPath, err = browser.InstallFirefox()
	} else {
		result, ierr := browser.Install()
		err = ierr
		if result != nil {
			installPath = result.ChromePath
		}
	}
	if err != nil {
		return setupSection{Name: "browser", Status: "failed", Message: err.Error()}
	}
	part := checkBrowserSetup(cmd, nil)
	if !part.Ready {
		return setupSection{Name: "browser", Status: "failed", Message: browserSectionMessage(part), Path: installPath}
	}
	if installPath == "" {
		installPath = selectedBrowserPath(part)
	}
	ui.ok("Installed %s.", engine)
	return setupSection{Name: "browser", Status: "done", Message: "Installed " + engine + ".", Path: installPath}
}

func selectedBrowserPath(part setupResult) string {
	for _, b := range part.Browsers {
		if b.Selected {
			return b.Path
		}
	}
	return ""
}

func browserSectionMessage(part setupResult) string {
	for _, c := range part.Checks {
		if c.Status == "failed" {
			if c.Fix != "" {
				return c.Message + " " + c.Fix
			}
			return c.Message
		}
	}
	return "browser setup failed"
}

func setupAI(cmd *cobra.Command, ui *setupUI, quick bool) setupSection {
	path, err := aiEnvPath()
	if err != nil {
		return setupSection{Name: "ai", Status: "failed", Message: err.Error()}
	}
	shown := tildePath(path)
	if quick && aiConfigValid() {
		ui.skip("AI settings are already valid; skipping (--quick).")
		return setupSection{Name: "ai", Status: "skipped", Message: "AI settings are already valid.", Path: path}
	}
	if !ui.interactive {
		if _, err := os.Stat(path); err == nil {
			ui.skip("Left existing %s unchanged (non-interactive).", shown)
			return setupSection{Name: "ai", Status: "skipped", Message: "Existing ai.env was left unchanged.", Path: path}
		}
		ui.skip("No terminal for prompts. Re-run vibium setup in a terminal to configure AI now.")
		return setupSection{Name: "ai", Status: "skipped", Message: "No ai.env yet; configure AI in a terminal with vibium setup."}
	}

	provider := os.Getenv("VIBIUM_AI_PROVIDER")
	if provider == "" {
		provider = "openai"
	}
	ui.println("%s", maybePaint(ui.useColor(), brandText, "Configure AI for run and check now."))
	defIdx := 0
	for i, p := range setupProviders {
		if p == provider {
			defIdx = i
		}
	}
	idx, err := ui.selectOne("Provider", providerOptions(), defIdx)
	if err != nil {
		return aiSectionError(err)
	}
	provider = setupProviders[idx]

	modelDefault := os.Getenv("VIBIUM_AI_MODEL")
	if modelDefault == "" && provider == "openai" {
		modelDefault = "gpt-5.6-sol"
	}
	if modelDefault == "" && provider == "xai" {
		modelDefault = "grok-4"
	}
	if modelDefault == "" && provider == "anthropic" {
		modelDefault = "claude-sonnet-4-6"
	}
	if modelDefault == "" && provider == "google" {
		modelDefault = "gemini-2.5-flash"
	}
	var model string
	for tries := 0; tries < 3; tries++ {
		model, err = ui.prompt("Model", modelDefault)
		if err != nil {
			return aiSectionError(err)
		}
		if strings.TrimSpace(model) != "" {
			break
		}
		ui.println("A model is required.")
	}
	if strings.TrimSpace(model) == "" {
		return setupSection{Name: "ai", Status: "failed", Message: "A model is required."}
	}

	credVar := verifier.Config{Provider: provider}.CredentialVariable()
	existingKey := os.Getenv(credVar)
	// Matches verifier.Config.Checks. When Grok subscription login lands,
	// xai becomes key-or-login here rather than key-required.
	needsKey := provider == "openai" || provider == "xai" || provider == "anthropic" || provider == "google"
	key := existingKey
	if needsKey || existingKey != "" {
		label := credVar
		if existingKey != "" {
			label += " [saved]"
		} else {
			label += " (Enter to skip for now)"
		}
		entered, err := ui.promptSecret(label)
		if err != nil {
			return setupSection{Name: "ai", Status: "failed", Message: err.Error()}
		}
		if entered != "" {
			key = entered
		}
	}

	baseURL := os.Getenv("VIBIUM_AI_BASE_URL")
	if provider == "openai-compatible" || provider == "local" {
		if baseURL == "" && provider == "local" {
			baseURL = "http://127.0.0.1:8080/v1"
		}
		baseURL, err = ui.prompt("API base URL", baseURL)
		if err != nil {
			return setupSection{Name: "ai", Status: "failed", Message: err.Error()}
		}
	} else {
		baseURL = ""
	}

	effort := ""
	if provider == "openai" || provider == "xai" || provider == "openai-compatible" {
		effort = os.Getenv("VIBIUM_AI_REASONING_EFFORT")
		if effort == "" && provider == "openai" {
			effort = "none"
		}
	}

	kv := map[string]string{
		"VIBIUM_AI_PROVIDER": provider,
		"VIBIUM_AI_MODEL":    model,
	}
	if key != "" {
		kv[credVar] = key
	}
	if baseURL != "" {
		kv["VIBIUM_AI_BASE_URL"] = baseURL
	}
	if effort != "" {
		kv["VIBIUM_AI_REASONING_EFFORT"] = effort
	}
	if err := writeAIEnv(path, kv); err != nil {
		return setupSection{Name: "ai", Status: "failed", Message: err.Error()}
	}
	applyWrittenAI(kv)
	ui.ok("Wrote %s (0600).", shown)
	return setupSection{Name: "ai", Status: "done", Message: "Wrote AI settings.", Path: path}
}

func setupSkills(cmd *cobra.Command, ui *setupUI, quick bool) setupSection {
	// Same detection as add-skill --agent auto, so the wizard and the
	// standalone command install to the same places.
	found := presentSkillAgents()
	if !ui.interactive && len(found) == 0 {
		ui.skip("Skipped skills: no ~/.grok or ~/.claude directory.")
		return setupSection{Name: "skills", Status: "skipped", Message: "No ~/.grok or ~/.claude directory; skills were not installed."}
	}
	targets := found
	if len(targets) == 0 {
		targets = []string{"claude"}
	}
	if quick {
		var missing []string
		for _, a := range targets {
			if !skillsPresent(a) {
				missing = append(missing, a)
			}
		}
		if len(missing) == 0 {
			agent := strings.Join(targets, ",")
			ui.skip("Skills already installed for %s; skipping (--quick).", agent)
			return setupSection{Name: "skills", Status: "skipped", Message: "Skills already installed.", Agent: agent, Agents: targets}
		}
		targets = missing
	} else if ui.interactive && len(found) > 1 {
		opts := make([]selectOption, 0, len(found)+1)
		for _, a := range found {
			opts = append(opts, selectOption{value: a})
		}
		opts = append(opts, selectOption{value: "both"})
		idx, err := ui.selectOne("Install skills for", opts, 0)
		if err != nil {
			return skillsSectionError(err, strings.Join(found, ","))
		}
		if opts[idx].value != "both" {
			targets = []string{opts[idx].value}
		}
	}
	agent := strings.Join(targets, ",")
	if ui.interactive {
		ok, err := ui.confirm(fmt.Sprintf("Install browser and check skills for %s?", strings.Join(targets, " and ")), true)
		if err != nil {
			return skillsSectionError(err, agent)
		}
		if !ok {
			ui.skip("Skipped skills.")
			return setupSection{Name: "skills", Status: "skipped", Message: "Skills install declined.", Agent: agent, Agents: targets}
		}
	}

	var paths []string
	for _, target := range targets {
		for _, name := range []string{"browser", "check"} {
			dir, skillPath, err := setupWriteSkill(name, target)
			if err != nil {
				return setupSection{Name: "skills", Status: "failed", Message: err.Error(), Agent: agent, Agents: targets, Paths: paths}
			}
			paths = append(paths, skillPath)
			ui.ok("Installed %s skill to %s.", name, dir)
		}
	}
	return setupSection{Name: "skills", Status: "done", Message: "Installed browser and check skills.", Agent: agent, Agents: targets, Paths: paths}
}

func skillsSectionError(err error, agent string) setupSection {
	if errors.Is(err, errSetupCancelled) {
		return setupSection{Name: "skills", Status: "cancelled", Message: "Setup cancelled.", Agent: agent}
	}
	return setupSection{Name: "skills", Status: "failed", Message: err.Error(), Agent: agent}
}

func runSetupReadiness(cmd *cobra.Command, scope string) setupResult {
	ready := newReadyCmd()
	target := ready
	if scope == "ai" || scope == "browser" {
		found, _, err := ready.Find([]string{scope})
		if err == nil {
			target = found
		}
	}
	target.SetContext(cmd.Context())
	probe := (&verifier.Model{}).Probe
	config, _ := verifier.ResolveConfig("check", verifier.Overrides{})
	if strings.TrimSpace(config.APIKey) == "" {
		probe = func(context.Context, verifier.Config) error { return nil }
	}
	result := runReadiness(target, nil, probe)
	if strings.TrimSpace(config.APIKey) == "" {
		// A keyless run is the normal first pass, not a failure: setup just
		// wrote ai.env without a key. Turn the credential complaint into the
		// one remaining step instead of ending the wizard on FAILED.
		shown := "~/.config/vibium/ai.env"
		if p, err := aiEnvPath(); err == nil {
			shown = tildePath(p)
		}
		cred := config.CredentialVariable()
		for i, c := range result.Checks {
			if c.Name == cred && c.Status == "failed" {
				result.Checks[i].Status = "skipped"
				result.Checks[i].Message = "No API key yet; add it to " + shown + "."
				result.Checks[i].Fix = ""
			}
		}
		failed := 0
		for _, c := range result.Checks {
			if c.Status == "failed" {
				failed++
			}
		}
		// With the key as the only gap, the probe was skipped because of
		// the key, not a broken configuration; say so. Real configuration
		// failures keep the stock message.
		if failed == 0 {
			for i, c := range result.Checks {
				if c.Name == "provider" {
					result.Checks[i].Status = "skipped"
					result.Checks[i].Message = "No API key; provider was not contacted."
				}
			}
			if !result.Ready {
				result.Summary = "Add your API key to " + shown + ", then run vibium ready ai."
			}
		}
	}
	return result
}

func aiConfigValid() bool {
	config, err := verifier.ConfigFromEnv()
	return err == nil && config.Validate() == nil
}

// setupTryHint suggests a first command after a fully ready run. Run needs
// working AI, so the hint only appears when the AI checks passed too; a
// keyless run already ends with its own next step.
func setupTryHint(ready *setupResult) string {
	if ready == nil || !ready.Ready || !aiConfigValid() {
		return ""
	}
	return `Try: vibium run "open example.com and describe the page"`
}

// providerOptions annotates the providers whose extra requirements are not
// obvious from the name; the key-required ones stay plain.
func providerOptions() []selectOption {
	opts := make([]selectOption, len(setupProviders))
	for i, p := range setupProviders {
		hint := ""
		switch p {
		case "openai-compatible":
			hint = "needs base URL"
		case "local":
			hint = "no API key, needs base URL"
		}
		opts[i] = selectOption{value: p, hint: hint}
	}
	return opts
}

// aiSectionError maps a prompt error onto the AI section, keeping user
// cancels distinct from real failures.
func aiSectionError(err error) setupSection {
	if errors.Is(err, errSetupCancelled) {
		return setupSection{Name: "ai", Status: "cancelled", Message: "Setup cancelled."}
	}
	return setupSection{Name: "ai", Status: "failed", Message: err.Error()}
}

func writeAIEnv(path string, kv map[string]string) error {
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
	b.WriteString("# Written by vibium setup. Mode 0600.\n")
	order := []string{"VIBIUM_AI_PROVIDER", "VIBIUM_AI_MODEL", "VIBIUM_AI_BASE_URL", "VIBIUM_AI_REASONING_EFFORT", "OPENAI_API_KEY", "XAI_API_KEY", "ANTHROPIC_API_KEY", "GOOGLE_API_KEY"}
	written := map[string]bool{}
	for _, k := range order {
		if v, ok := kv[k]; ok {
			b.WriteString("export " + k + "=" + shellSingleQuote(v) + "\n")
			written[k] = true
		}
	}
	for k, v := range kv {
		if written[k] {
			continue
		}
		b.WriteString("export " + k + "=" + shellSingleQuote(v) + "\n")
	}
	if err := os.WriteFile(path, []byte(b.String()), 0o600); err != nil {
		return fmt.Errorf("could not write %s: %w", path, err)
	}
	if err := os.Chmod(path, 0o600); err != nil {
		return fmt.Errorf("could not set permissions on %s: %w", path, err)
	}
	return nil
}

func applyWrittenAI(kv map[string]string) {
	for _, k := range []string{"VIBIUM_AI_PROVIDER", "VIBIUM_AI_MODEL", "VIBIUM_AI_BASE_URL", "VIBIUM_AI_REASONING_EFFORT"} {
		if _, ok := kv[k]; !ok {
			_ = os.Unsetenv(k)
		}
	}
	for k, v := range kv {
		_ = os.Setenv(k, v)
	}
}

func shellSingleQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

func aiEnvPath() (string, error) {
	dir, err := paths.GetConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "ai.env"), nil
}

func setupWriteSkill(name, agent string) (string, string, error) {
	content := skillMD
	switch name {
	case "browser":
	case "check":
		content = checkSkillMD
	default:
		return "", "", fmt.Errorf("unknown skill %q; choose browser or check", name)
	}
	skillDir, err := agentSkillDir(agent, name)
	if err != nil {
		return "", "", err
	}
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		return "", "", fmt.Errorf("could not create skill directory: %w", err)
	}
	skillPath := filepath.Join(skillDir, "SKILL.md")
	if err := os.WriteFile(skillPath, []byte(content), 0644); err != nil {
		return "", "", fmt.Errorf("could not write SKILL.md: %w", err)
	}
	return skillDir, skillPath, nil
}

func skillsPresent(agent string) bool {
	for _, name := range []string{"browser", "check"} {
		dir, err := agentSkillDir(agent, name)
		if err != nil {
			return false
		}
		if _, err := os.Stat(filepath.Join(dir, "SKILL.md")); err != nil {
			return false
		}
	}
	return true
}

func isDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

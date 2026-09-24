package main

import (
	"slices"
	"strings"
	"testing"

	"github.com/vibium/clicker/internal/verifier"
)

// #581: every command with AI flags lists their values under AI Flags and
// says where to find model IDs; commands without them keep plain Flags.
func TestAIFlagHelp(t *testing.T) {
	root, _ := newRootCmd("vibium")
	for _, path := range []string{"check", "run", "ready", "ready ai"} {
		c, _, err := root.Find(strings.Fields(path))
		if err != nil {
			t.Fatalf("%s: %v", path, err)
		}
		usage := c.UsageString()
		_, sections, _ := strings.Cut(usage, "\nFlags:\n")
		flags, rest, ok := strings.Cut(sections, "\nAI Flags:\n")
		if !ok {
			t.Fatalf("%s: no AI Flags section:\n%s", path, usage)
		}
		ai, models, ok := strings.Cut(rest, "\nModel IDs:\n")
		if !ok {
			t.Fatalf("%s: no Model IDs section:\n%s", path, usage)
		}
		for _, name := range []string{"--provider", "--model", "--ai-base-url", "--reasoning-effort"} {
			if strings.Contains(flags, name) {
				t.Errorf("%s: %s listed under Flags", path, name)
			}
			if !strings.Contains(ai, name) {
				t.Errorf("%s: %s missing from AI Flags", path, name)
			}
		}
		for _, want := range []string{verifier.OrList(verifier.ProviderNames()), verifier.OrList(verifier.ReasoningEfforts), "env: VIBIUM_AI_PROVIDER", "env: VIBIUM_AI_MODEL"} {
			if !strings.Contains(ai, want) {
				t.Errorf("%s: AI Flags missing %q", path, want)
			}
		}
		for _, p := range verifier.Providers {
			if !strings.Contains(models, p.Name) || (p.ModelsURL != "" && !strings.Contains(models, p.ModelsURL)) {
				t.Errorf("%s: Model IDs missing %s", path, p.Name)
			}
		}
	}

	browserCmd, _, _ := root.Find([]string{"ready", "browser"})
	if usage := browserCmd.UsageString(); strings.Contains(usage, "AI Flags:") || strings.Contains(usage, "Model IDs:") {
		t.Errorf("ready browser has no AI flags but shows their sections:\n%s", usage)
	}

	ai, _, _ := root.Find([]string{"ready", "ai"})
	if !slices.Equal(ai.ValidArgs, verifier.ProviderNames()) {
		t.Errorf("ready ai ValidArgs %v, want %v", ai.ValidArgs, verifier.ProviderNames())
	}
}

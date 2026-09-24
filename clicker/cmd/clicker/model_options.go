package main

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/vibium/clicker/internal/verifier"
)

// aiFlagGroup marks the flags help lists under "AI Flags:".
const aiFlagGroup = "vibium_ai_flag"

// The default template's local-flags line, split into Flags, AI Flags, and Model IDs
// for commands that have AI flags. Subcommands inherit the template.
const localFlagsSection = "{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}"
const aiFlagsSection = `{{if hasAIFlags .}}{{nonAIFlagUsages . | trimTrailingWhitespaces}}

AI Flags:
{{aiFlagUsages . | trimTrailingWhitespaces}}

Model IDs:
{{aiModelIDs}}{{else}}` + localFlagsSection + `{{end}}`

func init() {
	cobra.AddTemplateFunc("nonAIFlagUsages", func(c *cobra.Command) string { return groupFlagUsages(c, false) })
	cobra.AddTemplateFunc("aiFlagUsages", func(c *cobra.Command) string { return groupFlagUsages(c, true) })
	cobra.AddTemplateFunc("aiModelIDs", aiModelIDs)
	cobra.AddTemplateFunc("hasAIFlags", func(c *cobra.Command) bool { return groupFlagUsages(c, true) != "" })
}

func groupFlagUsages(c *cobra.Command, ai bool) string {
	set := pflag.NewFlagSet(c.Name(), pflag.ContinueOnError)
	c.LocalFlags().VisitAll(func(f *pflag.Flag) {
		if _, marked := f.Annotations[aiFlagGroup]; marked == ai {
			set.AddFlag(f)
		}
	})
	return set.FlagUsages()
}

func aiModelIDs() string {
	var lines []string
	for _, p := range verifier.Providers {
		where := p.ModelsURL
		if where == "" {
			where = "the model name your server serves"
		}
		lines = append(lines, fmt.Sprintf("  %-19s %s", p.Name, where))
	}
	return strings.Join(lines, "\n")
}

func addModelFlags(cmd *cobra.Command) {
	var noEffort []string
	for _, p := range verifier.Providers {
		if !p.ReasoningEffort {
			noEffort = append(noEffort, p.Name)
		}
	}
	cmd.Flags().String("provider", "", "AI provider: "+verifier.OrList(verifier.ProviderNames())+" (env: VIBIUM_AI_PROVIDER); changing it requires --model and resets endpoint/effort defaults")
	cmd.Flags().String("model", "", "Model ID for the provider; see Model IDs below (env: VIBIUM_AI_MODEL)")
	cmd.Flags().String("ai-base-url", "", "AI provider API base URL, not the site under test (env: VIBIUM_AI_BASE_URL); empty resets the provider default")
	cmd.Flags().String("reasoning-effort", "", verifier.OrList(verifier.ReasoningEfforts)+"; not for "+verifier.OrList(noEffort)+" (env: VIBIUM_AI_REASONING_EFFORT); empty uses the model default")
	for _, name := range []string{"provider", "model", "ai-base-url", "reasoning-effort"} {
		cmd.Flags().SetAnnotation(name, aiFlagGroup, []string{"true"})
	}
	if template := cmd.UsageTemplate(); !strings.Contains(template, aiFlagsSection) {
		cmd.SetUsageTemplate(strings.Replace(template, localFlagsSection, aiFlagsSection, 1))
	}
	example := "\n  vibium " + cmd.Name()
	if cmd.Name() == "ai" {
		example = "\n  vibium ready ai"
	}
	if input := map[string]string{"check": "the saved timezone is America/Chicago", "run": "change my timezone to America/Chicago and save it"}[cmd.Name()]; input != "" {
		example += ` "` + input + `"`
	}
	cmd.Example += example + ` --provider local --model my-model --ai-base-url http://127.0.0.1:8080/v1 --reasoning-effort ""` + "\n  # Uses these settings for one invocation; credentials still come from the environment."
}

func modelOverrides(cmd *cobra.Command) verifier.Overrides {
	var overrides verifier.Overrides
	for name, target := range map[string]**string{"provider": &overrides.Provider, "model": &overrides.Model, "ai-base-url": &overrides.BaseURL, "reasoning-effort": &overrides.ReasoningEffort} {
		if cmd.Flags().Changed(name) {
			value, _ := cmd.Flags().GetString(name)
			*target = &value
		}
	}
	return overrides
}

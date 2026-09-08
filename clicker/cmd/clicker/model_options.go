package main

import (
	"github.com/spf13/cobra"
	"github.com/vibium/clicker/internal/verifier"
)

func addModelFlags(cmd *cobra.Command) {
	cmd.Flags().String("provider", "", "Override the provider for this call; changing provider requires --model and resets endpoint/effort defaults")
	cmd.Flags().String("model", "", "Override the model for this call")
	cmd.Flags().String("base-url", "", "Override the API base URL for this call; an empty value resets the provider default")
	cmd.Flags().String("reasoning-effort", "", "Override OpenAI-compatible reasoning effort; an empty value uses the model default")
	example := "\n  vibium " + cmd.Name()
	if cmd.Name() == "ai" {
		example = "\n  vibium ready ai"
	}
	if input := map[string]string{"check": "the saved timezone is America/Chicago", "run": "change my timezone to America/Chicago and save it"}[cmd.Name()]; input != "" {
		example += ` "` + input + `"`
	}
	cmd.Example += example + ` --provider local --model my-model --base-url http://127.0.0.1:8080/v1 --reasoning-effort ""` + "\n  # Uses these settings for one invocation; credentials still come from the environment."
}

func modelOverrides(cmd *cobra.Command) verifier.Overrides {
	var overrides verifier.Overrides
	for name, target := range map[string]**string{"provider": &overrides.Provider, "model": &overrides.Model, "base-url": &overrides.BaseURL, "reasoning-effort": &overrides.ReasoningEffort} {
		if cmd.Flags().Changed(name) {
			value, _ := cmd.Flags().GetString(name)
			*target = &value
		}
	}
	return overrides
}

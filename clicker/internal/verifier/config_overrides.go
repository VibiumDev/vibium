package verifier

import (
	"fmt"
	"os"
)

// Overrides are public per-invocation options. Credentials are deliberately absent.
// Pointers distinguish omission from an explicit empty endpoint/effort reset.
type Overrides struct {
	Provider        *string `json:"provider,omitempty"`
	Model           *string `json:"model,omitempty"`
	BaseURL         *string `json:"baseURL,omitempty"`
	ReasoningEffort *string `json:"reasoningEffort,omitempty"`
}

func IsOverride(name string) bool {
	switch name {
	case "provider", "model", "baseURL", "reasoningEffort":
		return true
	}
	return false
}

func ConfigFromParams(role string, params map[string]interface{}) (Config, error) {
	var overrides Overrides
	for name, target := range map[string]**string{"provider": &overrides.Provider, "model": &overrides.Model, "baseURL": &overrides.BaseURL, "reasoningEffort": &overrides.ReasoningEffort} {
		if value, exists := params[name]; exists {
			text, ok := value.(string)
			if !ok {
				return Config{}, fmt.Errorf("%s must be a string", name)
			}
			*target = &text
		}
	}
	return ResolveConfig(role, overrides)
}

// ResolveConfig uses shared AI defaults without mutating the process environment.
// Switching provider clears provider-specific defaults before applying options.
func ResolveConfig(role string, overrides Overrides) (Config, error) {
	if role != "check" && role != "run" {
		return Config{}, fmt.Errorf("unknown model operation")
	}
	c := Config{Role: role}
	prefix := c.Prefix()
	c.Provider, c.Model = os.Getenv(prefix+"PROVIDER"), os.Getenv(prefix+"MODEL")
	c.BaseURL, c.ReasoningEffort = os.Getenv(prefix+"BASE_URL"), os.Getenv(prefix+"REASONING_EFFORT")
	providerChanged := overrides.Provider != nil && *overrides.Provider != c.Provider
	if providerChanged {
		c.Provider = *overrides.Provider
		c.Model, c.BaseURL, c.ReasoningEffort = "", "", ""
	}
	if overrides.Model != nil {
		c.Model = *overrides.Model
	}
	if overrides.BaseURL != nil {
		c.BaseURL = *overrides.BaseURL
	}
	if overrides.ReasoningEffort != nil {
		c.ReasoningEffort = *overrides.ReasoningEffort
	}
	c.APIKey = os.Getenv(c.CredentialVariable())
	err := c.Validate()
	if err != nil && providerChanged && overrides.Model == nil {
		return c, fmt.Errorf("%w; changing provider requires an explicit model override (--model in CLI)", err)
	}
	return c, err
}

// RecordingMetadata is an allowlist, never a serialization of Config. Endpoint
// paths can carry private routing information, so URLs and credentials stay out.
func (c Config) RecordingMetadata() map[string]interface{} {
	return map[string]interface{}{
		"provider": c.Provider, "model": c.Model, "reasoningEffort": c.ReasoningEffort,
		"maxOutputTokens": MaxOutputTokens, "maxActions": MaxActions, "timeoutMs": Timeout.Milliseconds(),
	}
}

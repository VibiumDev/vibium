package ai

import "strings"

// openAIModelEfforts maps OpenAI model-name prefixes to the reasoning effort
// the family needs for Chat Completions function tools. First match wins;
// unlisted families keep the provider default. chatToolsUnsupported marks
// families that reject function tools on Chat Completions with every
// reasoning effort, including none and unset (#621).
var openAIModelEfforts = []struct {
	prefix               string
	effort               string
	chatToolsUnsupported bool
}{
	{prefix: "gpt-6.1", chatToolsUnsupported: true},
	{prefix: "gpt-6", effort: "none"},
	{prefix: "gpt-5", effort: "none"},
}

// OpenAIEffortDefault returns the reasoning effort a native OpenAI model needs
// for Chat Completions function tools, or "" to keep the provider default.
func OpenAIEffortDefault(model string) string {
	for _, m := range openAIModelEfforts {
		if strings.HasPrefix(strings.TrimSpace(model), m.prefix) {
			return m.effort
		}
	}
	return ""
}

// OpenAIChatToolsProblem says why a native OpenAI model cannot serve as the
// verifier, or "" when it can.
func OpenAIChatToolsProblem(model string) string {
	model = strings.TrimSpace(model)
	for _, m := range openAIModelEfforts {
		if strings.HasPrefix(model, m.prefix) && m.chatToolsUnsupported {
			return model + " supports function tools only on OpenAI's Responses API, which Vibium does not use yet. Choose another model, such as gpt-5.6-sol."
		}
	}
	return ""
}

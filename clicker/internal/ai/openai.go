package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// Model selects a native provider adapter for the shared operation loop.
type Model struct{ Client *http.Client }

type toolCall struct {
	Signature  string `json:"-"` // opaque Gemini continuation metadata; memory only
	ProviderID string `json:"-"`
	ID         string `json:"id"`
	Type       string `json:"type"`
	Function   struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}
type message struct {
	Role       string      `json:"role"`
	Content    interface{} `json:"content"`
	ToolCalls  []toolCall  `json:"tool_calls,omitempty"`
	ToolCallID string      `json:"tool_call_id,omitempty"`
}

// complete requests one model turn. A non-empty force names a tool the model
// must call; it is applied on native openai and xai only, so the compatibility
// floor for openai-compatible and local servers stays at plain function tools.
func (v *Model) complete(ctx context.Context, config Config, messages []message, functions []interface{}, force string) (message, error) {
	switch config.Provider {
	case "anthropic":
		return v.completeAnthropic(ctx, config, messages, functions, force)
	case "google":
		return v.completeGoogle(ctx, config, messages, functions, force)
	default:
		return v.completeOpenAI(ctx, config, messages, functions, force)
	}
}

func (v *Model) completeOpenAI(ctx context.Context, config Config, messages []message, functions []interface{}, force string) (message, error) {
	base := config.Endpoint()
	// Parallel tool calls keep the provider default (enabled): the loop
	// executes a turn's calls one at a time in request order, so a batch of
	// observations saves round trips without reordering effects (#594). A
	// forced turn must deliver exactly one result call, so it disables them.
	payload := map[string]interface{}{"model": config.Model, "messages": messages, "tools": functions, "max_completion_tokens": MaxOutputTokens}
	if force != "" && (config.Provider == "openai" || config.Provider == "xai") {
		payload["tool_choice"] = map[string]interface{}{"type": "function", "function": map[string]string{"name": force}}
		payload["parallel_tool_calls"] = false
	}
	if config.ReasoningEffort != "" {
		payload["reasoning_effort"] = config.ReasoningEffort
	}
	headers := map[string]string{}
	if config.APIKey != "" {
		headers["Authorization"] = "Bearer " + config.APIKey
	}
	data, contentType, err := v.post(ctx, base+"/chat/completions", payload, headers)
	if err != nil {
		return message{}, err
	}
	var completion struct {
		Choices []struct {
			Message      message `json:"message"`
			FinishReason string  `json:"finish_reason"`
		} `json:"choices"`
	}
	if json.Unmarshal(data, &completion) != nil || len(completion.Choices) != 1 {
		return message{}, invalidProviderResponse(config, contentType, "a chat-completions message")
	}
	choice := completion.Choices[0]
	if choice.FinishReason != "stop" && choice.FinishReason != "tool_calls" {
		return message{}, fmt.Errorf("verifier response incomplete or refused")
	}
	return choice.Message, nil
}

// StripJSONFence extracts the body of a Markdown code fence when one is
// present, so a fenced verdict parses like a bare one.
//
// The instructions say to return ONLY a JSON object, and OpenAI models
// comply, but Anthropic models reliably wrap the object in a ```json fence
// and sometimes lead into it with a sentence of prose. The wrapper carries
// no information, so tolerating it keeps the verdict parse provider-neutral.
// Content without a complete fence is returned unchanged; whatever comes
// back still has to survive the strict JSON parse and verdict validation.
func StripJSONFence(content string) string {
	body := strings.TrimSpace(content)
	start := strings.Index(body, "```")
	if start < 0 {
		return content
	}
	body = body[start+3:]
	newline := strings.IndexByte(body, '\n')
	if newline < 0 {
		return content
	}
	body = body[newline+1:] // drop the info string line ("json", or empty)
	end := strings.Index(body, "```")
	if end < 0 {
		return content
	}
	return strings.TrimSpace(body[:end])
}

// parseResult validates the structured verdict without exposing model content.
func parseResult(msg message, claim string) (Result, error) {
	content, ok := msg.Content.(string)
	if !ok {
		return Result{}, fmt.Errorf("verifier returned no verdict")
	}
	return ParseVerdict(content, claim)
}

// ParseVerdict validates a structured verdict, for the operations that framed
// one: Check in internal/check parses return_verdict content through it.
func ParseVerdict(content, claim string) (Result, error) {
	content = StripJSONFence(content)
	var result Result
	if len(content) > MaxText || json.Unmarshal([]byte(content), &result) != nil {
		return Result{}, fmt.Errorf("verifier returned an invalid JSON verdict")
	}
	result.Claim = claim
	if result.Evidence == nil {
		result.Evidence = []Evidence{}
	}
	if err := result.Validate(); err != nil {
		return Result{}, err
	}
	return result, nil
}

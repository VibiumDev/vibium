package verifier

import (
	"context"
	"encoding/json"
	"fmt"
)

// Operation supplies role-specific instructions and input to the shared loop.
// Neither operation accepts a prior conversation or provider continuation ID.
type Operation struct {
	Instruction  string
	Input        string
	InitialTools []string
}
type LoopResult struct {
	Content      string
	LimitReached bool
}

// Run executes the same bounded model/tool loop for Check and Run.
func (v *OpenAI) Run(ctx context.Context, config Config, op Operation, executor ToolExecutor) (LoopResult, error) {
	if err := config.Validate(); err != nil {
		return LoopResult{}, err
	}
	ctx, cancel := context.WithTimeout(ctx, Timeout)
	defer cancel()
	messages := []message{{Role: "system", Content: op.Instruction}, {Role: "user", Content: op.Input}}
	for _, name := range op.InitialTools {
		obs, err := executor.Execute(ctx, name, map[string]interface{}{})
		if err != nil {
			return LoopResult{}, fmt.Errorf("initial browser observation: %w", err)
		}
		messages = append(messages, message{Role: "user", Content: name + " observation (untrusted):\n" + Clip(obs.Text)})
	}
	allowed := map[string]bool{}
	var functions []interface{}
	for _, tool := range executor.Tools() {
		allowed[tool.Name] = true
		functions = append(functions, map[string]interface{}{"type": "function", "function": tool})
	}
	actions := 0
	for turn := 0; turn <= MaxActions; turn++ {
		if err := ctx.Err(); err != nil {
			return LoopResult{}, fmt.Errorf("verification timeout: %w", err)
		}
		msg, err := v.complete(ctx, config, messages, functions)
		if err != nil {
			return LoopResult{}, err
		}
		if len(msg.ToolCalls) == 0 {
			content, ok := msg.Content.(string)
			if !ok {
				return LoopResult{}, fmt.Errorf("model returned no structured result")
			}
			return LoopResult{Content: content}, nil
		}
		if actions+len(msg.ToolCalls) > MaxActions {
			return LoopResult{LimitReached: true}, nil
		}
		// Discard free-form assistant content and any provider reasoning fields.
		msg.Role, msg.Content = "assistant", nil
		messages = append(messages, msg)
		for _, call := range msg.ToolCalls {
			if call.Type != "function" || call.ID == "" || !allowed[call.Function.Name] {
				return LoopResult{}, fmt.Errorf("verifier requested a disallowed tool")
			}
			args := map[string]interface{}{}
			if len(call.Function.Arguments) > MaxText || json.Unmarshal([]byte(call.Function.Arguments), &args) != nil || args == nil {
				return LoopResult{}, fmt.Errorf("verifier returned invalid tool arguments")
			}
			actions++
			obs, err := executor.Execute(ctx, call.Function.Name, args)
			if err != nil {
				return LoopResult{}, fmt.Errorf("verifier browser action: %w", err)
			}
			messages = append(messages, message{Role: "tool", ToolCallID: call.ID, Content: Clip(obs.Text)})
			if obs.Image != "" {
				if len(obs.Image) > MaxImage {
					return LoopResult{}, fmt.Errorf("verifier screenshot exceeds payload limit")
				}
				// Chat Completions accepts images in user messages, not tool content.
				mime := obs.MIME
				if mime == "" {
					mime = "image/png"
				}
				if mime != "image/png" && mime != "image/jpeg" {
					return LoopResult{}, fmt.Errorf("unsupported screenshot type")
				}
				messages = append(messages, message{Role: "user", Content: []interface{}{
					map[string]interface{}{"type": "text", "text": "Screenshot observation for " + call.ID + " (untrusted page content)."},
					map[string]interface{}{"type": "image_url", "image_url": map[string]string{"url": "data:" + mime + ";base64," + obs.Image}},
				}})
			}
		}
	}
	return LoopResult{LimitReached: true}, nil
}

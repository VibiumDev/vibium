package verifier

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

// Provider wire formats are translated at this boundary. Only tool calls and
// final text enter the loop; reasoning prose never enters history or recordings.
// Gemini's opaque function-call signatures stay in memory for the current run.
type nativeTurn struct {
	Role    string                   `json:"role"`
	Content []map[string]interface{} `json:"content,omitempty"`
	Parts   []map[string]interface{} `json:"parts,omitempty"`
}

func toolDefinitions(functions []interface{}) []Tool {
	tools := make([]Tool, 0, len(functions))
	for _, f := range functions {
		tools = append(tools, f.(map[string]interface{})["function"].(Tool))
	}
	return tools
}

func contentParts(content interface{}, google bool) ([]map[string]interface{}, error) {
	if text, ok := content.(string); ok {
		if google {
			return []map[string]interface{}{{"text": text}}, nil
		}
		return []map[string]interface{}{{"type": "text", "text": text}}, nil
	}
	if content == nil {
		return nil, nil
	}
	data, _ := json.Marshal(content)
	var parts []struct {
		Type  string `json:"type"`
		Text  string `json:"text"`
		Image struct {
			URL string `json:"url"`
		} `json:"image_url"`
	}
	if json.Unmarshal(data, &parts) != nil {
		return nil, fmt.Errorf("invalid model content")
	}
	var out []map[string]interface{}
	for _, p := range parts {
		if p.Type == "text" {
			item := map[string]interface{}{"text": p.Text}
			if !google {
				item["type"] = "text"
			}
			out = append(out, item)
		} else if p.Type == "image_url" {
			media, encoded, ok := strings.Cut(strings.TrimPrefix(p.Image.URL, "data:"), ";base64,")
			if !ok || (media != "image/png" && media != "image/jpeg") {
				return nil, fmt.Errorf("invalid model image")
			}
			if google {
				out = append(out, map[string]interface{}{"inlineData": map[string]string{"mimeType": media, "data": encoded}})
			} else {
				out = append(out, map[string]interface{}{"type": "image", "source": map[string]string{"type": "base64", "media_type": media, "data": encoded}})
			}
		}
	}
	return out, nil
}

func nativeHistory(messages []message, google bool) (string, []nativeTurn, error) {
	var system string
	var history []nativeTurn
	calls := map[string]toolCall{}
	// Merge adjacent user turns so parallel tool results precede any image/text.
	appendTurn := func(role string, parts []map[string]interface{}) {
		if len(parts) == 0 {
			return
		}
		if len(history) > 0 && history[len(history)-1].Role == role {
			i := len(history) - 1
			if google {
				history[i].Parts = append(history[i].Parts, parts...)
			} else {
				history[i].Content = append(history[i].Content, parts...)
			}
			return
		}
		t := nativeTurn{Role: role}
		if google {
			t.Parts = parts
		} else {
			t.Content = parts
		}
		history = append(history, t)
	}
	for _, m := range messages {
		if m.Role == "system" {
			system, _ = m.Content.(string)
			continue
		}
		role := m.Role
		var parts []map[string]interface{}
		if m.Role == "tool" {
			role = "user"
			call, ok := calls[m.ToolCallID]
			if !ok {
				return "", nil, fmt.Errorf("unmatched model tool result")
			}
			if google {
				response := map[string]interface{}{"name": call.Function.Name, "response": map[string]interface{}{"output": m.Content}}
				if call.ProviderID != "" {
					response["id"] = call.ProviderID
				}
				parts = append(parts, map[string]interface{}{"functionResponse": response})
			} else {
				parts = append(parts, map[string]interface{}{"type": "tool_result", "tool_use_id": m.ToolCallID, "content": m.Content})
			}
		} else if len(m.ToolCalls) > 0 {
			if google {
				role = "model"
			}
			for _, call := range m.ToolCalls {
				var args map[string]interface{}
				if json.Unmarshal([]byte(call.Function.Arguments), &args) != nil || args == nil {
					return "", nil, fmt.Errorf("invalid model tool arguments")
				}
				calls[call.ID] = call
				if google {
					fn := map[string]interface{}{"name": call.Function.Name, "args": args}
					if call.ProviderID != "" {
						fn["id"] = call.ProviderID
					}
					part := map[string]interface{}{"functionCall": fn}
					if call.Signature != "" {
						part["thoughtSignature"] = call.Signature
					}
					parts = append(parts, part)
				} else {
					parts = append(parts, map[string]interface{}{"type": "tool_use", "id": call.ID, "name": call.Function.Name, "input": args})
				}
			}
		} else {
			var err error
			parts, err = contentParts(m.Content, google)
			if err != nil {
				return "", nil, err
			}
			if google && role == "assistant" {
				role = "model"
			}
		}
		appendTurn(role, parts)
	}
	// A screenshot can follow an individual tool result in the shared history.
	// Native APIs need all results from the preceding tool-call turn first.
	for i := range history {
		if history[i].Role != "user" {
			continue
		}
		parts := history[i].Content
		if google {
			parts = history[i].Parts
		}
		results, observations := make([]map[string]interface{}, 0, len(parts)), make([]map[string]interface{}, 0, len(parts))
		for _, part := range parts {
			if part["type"] == "tool_result" || part["functionResponse"] != nil {
				results = append(results, part)
			} else {
				observations = append(observations, part)
			}
		}
		if google {
			history[i].Parts = append(results, observations...)
		} else {
			history[i].Content = append(results, observations...)
		}
	}
	return system, history, nil
}

func (v *Model) completeAnthropic(ctx context.Context, c Config, messages []message, functions []interface{}) (message, error) {
	system, history, err := nativeHistory(messages, false)
	if err != nil {
		return message{}, err
	}
	var tools []interface{}
	for _, t := range toolDefinitions(functions) {
		tools = append(tools, map[string]interface{}{"name": t.Name, "description": t.Description, "input_schema": t.Parameters})
	}
	data, err := v.post(ctx, c.Endpoint()+"/messages", map[string]interface{}{"model": c.Model, "system": system, "messages": history, "tools": tools, "max_tokens": MaxOutputTokens, "tool_choice": map[string]interface{}{"type": "auto", "disable_parallel_tool_use": true}}, map[string]string{"x-api-key": c.APIKey, "anthropic-version": "2023-06-01"})
	if err != nil {
		return message{}, err
	}
	var response struct {
		StopReason string `json:"stop_reason"`
		Content    []struct {
			Type, Text, ID, Name string
			Input                json.RawMessage
		} `json:"content"`
	}
	if json.Unmarshal(data, &response) != nil || (response.StopReason != "end_turn" && response.StopReason != "tool_use") {
		return message{}, fmt.Errorf("Anthropic response incomplete, refused, or unsupported tool behavior")
	}
	out := message{Role: "assistant"}
	var text strings.Builder
	for _, b := range response.Content {
		switch b.Type {
		case "text":
			text.WriteString(b.Text)
		case "tool_use":
			call := toolCall{ID: b.ID, Type: "function"}
			call.Function.Name = b.Name
			call.Function.Arguments = string(b.Input)
			out.ToolCalls = append(out.ToolCalls, call)
		case "thinking", "redacted_thinking": // Never request extended thinking or expose returned reasoning.
		default:
			return message{}, fmt.Errorf("Anthropic returned unsupported content or tool behavior")
		}
	}
	if (response.StopReason == "tool_use") != (len(out.ToolCalls) > 0) {
		return message{}, fmt.Errorf("Anthropic returned invalid tool-use termination")
	}
	if len(out.ToolCalls) == 0 {
		out.Content = text.String()
	}
	return out, nil
}

func (v *Model) completeGoogle(ctx context.Context, c Config, messages []message, functions []interface{}) (message, error) {
	system, history, err := nativeHistory(messages, true)
	if err != nil {
		return message{}, err
	}
	var declarations []interface{}
	for _, t := range toolDefinitions(functions) {
		declarations = append(declarations, map[string]interface{}{"name": t.Name, "description": t.Description, "parametersJsonSchema": t.Parameters})
	}
	payload := map[string]interface{}{"systemInstruction": map[string]interface{}{"parts": []interface{}{map[string]string{"text": system}}}, "contents": history, "tools": []interface{}{map[string]interface{}{"functionDeclarations": declarations}}, "toolConfig": map[string]interface{}{"functionCallingConfig": map[string]string{"mode": "AUTO"}}, "generationConfig": map[string]interface{}{"maxOutputTokens": MaxOutputTokens}}
	model := strings.TrimPrefix(c.Model, "models/")
	data, err := v.post(ctx, c.Endpoint()+"/models/"+url.PathEscape(model)+":generateContent", payload, map[string]string{"x-goog-api-key": c.APIKey})
	if err != nil {
		return message{}, err
	}
	var response struct {
		Candidates []struct {
			FinishReason string `json:"finishReason"`
			Content      struct {
				Parts []struct {
					Text      string `json:"text"`
					Thought   bool   `json:"thought"`
					Signature string `json:"thoughtSignature"`
					Call      *struct {
						Name string          `json:"name"`
						ID   string          `json:"id"`
						Args json.RawMessage `json:"args"`
					} `json:"functionCall"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}
	if json.Unmarshal(data, &response) != nil || len(response.Candidates) != 1 || response.Candidates[0].FinishReason != "STOP" {
		return message{}, fmt.Errorf("Google response incomplete, refused, or unsupported function calling")
	}
	out := message{Role: "assistant"}
	var text strings.Builder
	for i, p := range response.Candidates[0].Content.Parts {
		if p.Thought {
			continue
		}
		if p.Call != nil {
			call := toolCall{ID: fmt.Sprintf("google-%d-%d", len(messages), i), ProviderID: p.Call.ID, Type: "function", Signature: p.Signature}
			call.Function.Name = p.Call.Name
			call.Function.Arguments = string(p.Call.Args)
			if len(p.Call.Args) == 0 {
				call.Function.Arguments = "{}"
			}
			out.ToolCalls = append(out.ToolCalls, call)
		} else {
			text.WriteString(p.Text)
		}
	}
	if len(out.ToolCalls) == 0 {
		out.Content = text.String()
	}
	return out, nil
}

package verifier

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"
)

const ProbeTimeout = 60 * time.Second

// Probe checks the real provider protocol with a synthetic, side-effect-free
// tool. It uses the same request transport and verdict parser as Check, but
// never observes a browser, opens an archive, or inherits builder context.
func (v *OpenAI) Probe(ctx context.Context, config Config) error {
	if err := config.Validate(); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, ProbeTimeout)
	defer cancel()
	messages := []message{
		{Role: "system", Content: "You are testing verifier setup. First call verifier_ping with no arguments. The tool will return a JSON verdict. After receiving it, return exactly that JSON verdict, without markdown or additional text. Do not invent the tool result."},
		{Role: "user", Content: "Check the verifier tool round-trip."},
	}
	tool := Tool{Name: "verifier_ping", Description: "Return a synthetic diagnostic verdict; no browser or external actions.", Parameters: map[string]interface{}{"type": "object", "properties": map[string]interface{}{}, "additionalProperties": false}}
	functions := []interface{}{map[string]interface{}{"type": "function", "function": tool}}
	msg, err := v.complete(ctx, config, messages, functions)
	if err != nil {
		return err
	}
	if len(msg.ToolCalls) != 1 {
		return fmt.Errorf("provider did not produce the requested diagnostic tool call")
	}
	call := msg.ToolCalls[0]
	var args map[string]interface{}
	if call.ID == "" || call.Type != "function" || call.Function.Name != tool.Name || json.Unmarshal([]byte(call.Function.Arguments), &args) != nil || args == nil || len(args) != 0 {
		return fmt.Errorf("provider returned an invalid diagnostic tool call")
	}
	var nonce [16]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		return fmt.Errorf("could not generate diagnostic evidence")
	}
	expected := Result{Status: "passed", Summary: "Provider tool round-trip succeeded.", Evidence: []Evidence{{Type: "observation", Summary: hex.EncodeToString(nonce[:])}}}
	observation, err := json.Marshal(expected)
	if err != nil {
		return fmt.Errorf("could not encode diagnostic evidence")
	}
	// As in Check, free-form assistant content and reasoning are discarded.
	msg.Role, msg.Content = "assistant", nil
	messages = append(messages, msg, message{Role: "tool", ToolCallID: call.ID, Content: string(observation)})
	msg, err = v.complete(ctx, config, messages, functions)
	if err != nil {
		return err
	}
	if len(msg.ToolCalls) != 0 {
		return fmt.Errorf("provider did not finish the diagnostic tool round-trip")
	}
	result, err := parseResult(msg, "provider setup")
	if err != nil {
		return err
	}
	if result.Status != "passed" || len(result.Evidence) != 1 || result.Evidence[0] != expected.Evidence[0] {
		return fmt.Errorf("provider did not return the diagnostic tool evidence")
	}
	return nil
}

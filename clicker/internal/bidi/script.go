package bidi

import (
	"encoding/json"
	"fmt"
)

// RealmInfo represents information about a JavaScript realm.
type RealmInfo struct {
	Realm   string `json:"realm"`
	Origin  string `json:"origin"`
	Type    string `json:"type"`
	Context string `json:"context,omitempty"`
}

// GetRealmsResult represents the result of script.getRealms.
type GetRealmsResult struct {
	Realms []RealmInfo `json:"realms"`
}

// GetRealms returns the available JavaScript realms.
func (c *Client) GetRealms(context string) (*GetRealmsResult, error) {
	params := map[string]interface{}{}
	if context != "" {
		params["context"] = context
	}

	msg, err := c.SendCommand("script.getRealms", params)
	if err != nil {
		return nil, err
	}

	var result GetRealmsResult
	if err := json.Unmarshal(msg.Result, &result); err != nil {
		return nil, fmt.Errorf("failed to parse script.getRealms result: %w", err)
	}

	return &result, nil
}

// EvaluateResult represents the result of script.evaluate.
type EvaluateResult struct {
	Type   string          `json:"type"`
	Result json.RawMessage `json:"result"`
}

// RemoteValue represents a value returned from script evaluation.
type RemoteValue struct {
	Type  string      `json:"type"`
	Value interface{} `json:"value,omitempty"`
}

// Evaluate evaluates a JavaScript expression and returns the result.
// If context is empty, it uses the first available context.
func (c *Client) Evaluate(context, expression string) (interface{}, error) {
	// If no context provided, get the first one from the tree
	if context == "" {
		tree, err := c.GetTree()
		if err != nil {
			return nil, fmt.Errorf("failed to get browsing context: %w", err)
		}
		if len(tree.Contexts) == 0 {
			return nil, fmt.Errorf("no browsing contexts available")
		}
		context = tree.Contexts[0].Context
	}

	params := map[string]interface{}{
		"expression":      expression,
		"target":          map[string]interface{}{"context": context},
		"awaitPromise":    true,
		"resultOwnership": "none",
	}

	msg, err := c.SendCommand("script.evaluate", params)
	if err != nil {
		return nil, err
	}

	// Parse the result
	var evalResult struct {
		Type   string          `json:"type"`
		Result json.RawMessage `json:"result"`
	}
	if err := json.Unmarshal(msg.Result, &evalResult); err != nil {
		return nil, fmt.Errorf("failed to parse script.evaluate result: %w", err)
	}

	if evalResult.Type == "exception" {
		return nil, fmt.Errorf("script exception: %s", string(evalResult.Result))
	}

	// Parse the remote value
	var remoteValue RemoteValue
	if err := json.Unmarshal(evalResult.Result, &remoteValue); err != nil {
		return nil, fmt.Errorf("failed to parse remote value: %w", err)
	}

	return convertRemoteValue(remoteValue), nil
}

// CallFunction calls a JavaScript function with arguments.
// If context is empty, it uses the first available context.
func (c *Client) CallFunction(context, functionDeclaration string, args []interface{}) (interface{}, error) {
	// If no context provided, get the first one from the tree
	if context == "" {
		tree, err := c.GetTree()
		if err != nil {
			return nil, fmt.Errorf("failed to get browsing context: %w", err)
		}
		if len(tree.Contexts) == 0 {
			return nil, fmt.Errorf("no browsing contexts available")
		}
		context = tree.Contexts[0].Context
	}

	// Convert args to serialized values
	serializedArgs := make([]map[string]interface{}, len(args))
	for i, arg := range args {
		serializedArgs[i] = serializeValue(arg)
	}

	params := map[string]interface{}{
		"functionDeclaration": functionDeclaration,
		"target":              map[string]interface{}{"context": context},
		"arguments":           serializedArgs,
		"awaitPromise":        true,
		"resultOwnership":     "none",
	}

	msg, err := c.SendCommand("script.callFunction", params)
	if err != nil {
		return nil, err
	}

	// Parse the result
	var callResult struct {
		Type   string          `json:"type"`
		Result json.RawMessage `json:"result"`
	}
	if err := json.Unmarshal(msg.Result, &callResult); err != nil {
		return nil, fmt.Errorf("failed to parse script.callFunction result: %w", err)
	}

	if callResult.Type == "exception" {
		return nil, fmt.Errorf("script exception: %s", string(callResult.Result))
	}

	// Parse the remote value
	var remoteValue RemoteValue
	if err := json.Unmarshal(callResult.Result, &remoteValue); err != nil {
		return nil, fmt.Errorf("failed to parse remote value: %w", err)
	}

	return convertRemoteValue(remoteValue), nil
}

// serializeValue converts a Go value to a BiDi serialized value.
func serializeValue(v interface{}) map[string]interface{} {
	switch val := v.(type) {
	case nil:
		return map[string]interface{}{"type": "undefined"}
	case bool:
		return map[string]interface{}{"type": "boolean", "value": val}
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
		return map[string]interface{}{"type": "number", "value": val}
	case string:
		return map[string]interface{}{"type": "string", "value": val}
	default:
		// For complex types, try to serialize as string
		return map[string]interface{}{"type": "string", "value": fmt.Sprintf("%v", val)}
	}
}

// convertRemoteValue recursively converts a BiDi RemoteValue into a plain Go
// value suitable for JSON serialization. Primitives pass through unchanged;
// arrays and objects are reconstructed by converting each child element.
func convertRemoteValue(rv RemoteValue) interface{} {
	switch rv.Type {
	case "string", "number", "boolean":
		return rv.Value
	case "null", "undefined":
		return nil
	case "array":
		items, ok := rv.Value.([]interface{})
		if !ok {
			return rv.Value
		}
		result := make([]interface{}, len(items))
		for i, item := range items {
			result[i] = convertRemoteValue(remoteValueFrom(item))
		}
		return result
	case "object":
		pairs, ok := rv.Value.([]interface{})
		if !ok {
			return rv.Value
		}
		result := make(map[string]interface{}, len(pairs))
		for _, pair := range pairs {
			kv, ok := pair.([]interface{})
			if !ok || len(kv) != 2 {
				continue
			}
			// BiDi allows non-string keys (e.g. a Map keyed by objects), but a
			// plain JS object always uses string keys. A non-string key can't be
			// a Go map[string]interface{} key, so such entries are skipped.
			key, ok := kv[0].(string)
			if !ok {
				continue
			}
			result[key] = convertRemoteValue(remoteValueFrom(kv[1]))
		}
		return result
	default:
		return rv.Value
	}
}

// remoteValueFrom interprets a child element of an array/object RemoteValue
// (decoded by encoding/json into interface{}) as a RemoteValue. Children arrive
// as map[string]interface{} carrying "type"/"value" fields; anything else is
// wrapped so its raw value passes through convertRemoteValue's default case.
func remoteValueFrom(v interface{}) RemoteValue {
	m, ok := v.(map[string]interface{})
	if !ok {
		return RemoteValue{Value: v}
	}
	rv := RemoteValue{Value: m["value"]}
	if t, ok := m["type"].(string); ok {
		rv.Type = t
	}
	return rv
}

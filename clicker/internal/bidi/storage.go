package bidi

import (
	"encoding/json"
	"fmt"
)

// Cookie represents a browser cookie.
type Cookie struct {
	Name     string  `json:"name"`
	Value    string  `json:"value"`
	Domain   string  `json:"domain,omitempty"`
	Path     string  `json:"path,omitempty"`
	Secure   bool    `json:"secure,omitempty"`
	HTTPOnly bool    `json:"httpOnly,omitempty"`
	SameSite string  `json:"sameSite,omitempty"`
	Size     float64 `json:"size,omitempty"`
}

// UnmarshalJSON accepts the cookie value either as a plain string or as a BiDi
// typed value object ({"type":"string","value":"..."}). storage.getCookies
// returns the latter, which previously failed to unmarshal into the string
// field and crashed every storage_state call (issue #150).
func (c *Cookie) UnmarshalJSON(data []byte) error {
	type cookieAlias Cookie // avoid recursion into this method
	aux := struct {
		Value json.RawMessage `json:"value"`
		*cookieAlias
	}{cookieAlias: (*cookieAlias)(c)}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	if len(aux.Value) == 0 {
		return nil
	}
	var s string
	if err := json.Unmarshal(aux.Value, &s); err == nil {
		c.Value = s
		return nil
	}
	var typed struct {
		Value string `json:"value"`
	}
	if err := json.Unmarshal(aux.Value, &typed); err != nil {
		return fmt.Errorf("failed to parse cookie value: %w", err)
	}
	c.Value = typed.Value
	return nil
}

// PartitionKey represents a storage partition key for cookies.
type PartitionKey struct {
	UserContext string `json:"userContext,omitempty"`
}

// GetCookies returns all cookies for the given browsing context.
func (c *Client) GetCookies(context string) ([]Cookie, error) {
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
		"partition": map[string]interface{}{
			"type":    "context",
			"context": context,
		},
	}

	msg, err := c.SendCommand("storage.getCookies", params)
	if err != nil {
		return nil, err
	}

	var result struct {
		Cookies      []Cookie     `json:"cookies"`
		PartitionKey PartitionKey `json:"partitionKey"`
	}
	if err := json.Unmarshal(msg.Result, &result); err != nil {
		return nil, fmt.Errorf("failed to parse storage.getCookies result: %w", err)
	}

	return result.Cookies, nil
}

// SetCookie sets a cookie in the given browsing context.
func (c *Client) SetCookie(context string, cookie Cookie) error {
	if context == "" {
		tree, err := c.GetTree()
		if err != nil {
			return fmt.Errorf("failed to get browsing context: %w", err)
		}
		if len(tree.Contexts) == 0 {
			return fmt.Errorf("no browsing contexts available")
		}
		context = tree.Contexts[0].Context
	}

	cookieMap := map[string]interface{}{
		"name":  cookie.Name,
		"value": map[string]interface{}{"type": "string", "value": cookie.Value},
	}
	if cookie.Domain != "" {
		cookieMap["domain"] = cookie.Domain
	}
	if cookie.Path != "" {
		cookieMap["path"] = cookie.Path
	}

	params := map[string]interface{}{
		"cookie": cookieMap,
		"partition": map[string]interface{}{
			"type":    "context",
			"context": context,
		},
	}

	_, err := c.SendCommand("storage.setCookie", params)
	return err
}


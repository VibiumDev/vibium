package features

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/vibium/clicker/internal/bidi"
)

// ActionabilityResult contains all actionability check results.
type ActionabilityResult struct {
	Visible        bool `json:"visible"`
	Stable         bool `json:"stable"`
	ReceivesEvents bool `json:"receivesEvents"`
	Enabled        bool `json:"enabled"`
	Editable       bool `json:"editable"`
}

// CheckVisible verifies the element has a non-empty bounding box and is not hidden.
// An element is visible if:
// - It has width > 0 and height > 0
// - visibility is not "hidden"
// - display is not "none"
func CheckVisible(client *bidi.Client, context, selector string) (bool, error) {
	script := `
		(selector) => {
			const el = document.querySelector(selector);
			if (!el) return JSON.stringify({ error: 'not found' });

			const rect = el.getBoundingClientRect();
			if (rect.width === 0 || rect.height === 0) {
				return JSON.stringify({ visible: false, reason: 'zero size' });
			}

			const style = window.getComputedStyle(el);
			if (style.visibility === 'hidden') {
				return JSON.stringify({ visible: false, reason: 'visibility hidden' });
			}
			if (style.display === 'none') {
				return JSON.stringify({ visible: false, reason: 'display none' });
			}

			return JSON.stringify({ visible: true });
		}
	`

	result, err := callCheckFunction(client, context, selector, script)
	if err != nil {
		return false, err
	}

	var data struct {
		Visible bool   `json:"visible"`
		Reason  string `json:"reason,omitempty"`
		Error   string `json:"error,omitempty"`
	}
	if err := json.Unmarshal([]byte(result), &data); err != nil {
		return false, fmt.Errorf("failed to parse visibility result: %w", err)
	}

	if data.Error != "" {
		return false, fmt.Errorf("element %s", data.Error)
	}

	if !data.Visible && data.Reason != "" {
		return false, fmt.Errorf("element not visible: %s", data.Reason)
	}

	return data.Visible, nil
}

// CheckStable verifies the element's bounding box hasn't changed between two checks.
// Compares position at t and t+50ms - if same, element is stable (not animating).
func CheckStable(client *bidi.Client, context, selector string) (bool, error) {
	// Get bounding box at time t
	box1, err := getBoundingBox(client, context, selector)
	if err != nil {
		return false, err
	}

	// Wait 50ms
	time.Sleep(50 * time.Millisecond)

	// Get bounding box at time t+50ms
	box2, err := getBoundingBox(client, context, selector)
	if err != nil {
		return false, err
	}

	// Compare - stable if all values match
	stable := box1.X == box2.X &&
		box1.Y == box2.Y &&
		box1.Width == box2.Width &&
		box1.Height == box2.Height

	return stable, nil
}

// CheckReceivesEvents verifies the element is the hit target at its center point.
// Uses elementFromPoint() to check if the element (or a descendant) receives pointer events.
func CheckReceivesEvents(client *bidi.Client, context, selector string) (bool, error) {
	script := `
		(selector) => {
			const el = document.querySelector(selector);
			if (!el) return JSON.stringify({ error: 'not found' });

			const rect = el.getBoundingClientRect();
			const centerX = rect.x + rect.width / 2;
			const centerY = rect.y + rect.height / 2;

			// Get element at center point
			const hitTarget = document.elementFromPoint(centerX, centerY);
			if (!hitTarget) {
				return JSON.stringify({ receivesEvents: false, reason: 'no element at point' });
			}

			// Check if hit target is the element or a descendant
			if (el === hitTarget || el.contains(hitTarget)) {
				return JSON.stringify({ receivesEvents: true });
			}

			// Element is obscured by another element
			return JSON.stringify({
				receivesEvents: false,
				reason: 'obscured by ' + hitTarget.tagName.toLowerCase()
			});
		}
	`

	result, err := callCheckFunction(client, context, selector, script)
	if err != nil {
		return false, err
	}

	var data struct {
		ReceivesEvents bool   `json:"receivesEvents"`
		Reason         string `json:"reason,omitempty"`
		Error          string `json:"error,omitempty"`
	}
	if err := json.Unmarshal([]byte(result), &data); err != nil {
		return false, fmt.Errorf("failed to parse receivesEvents result: %w", err)
	}

	if data.Error != "" {
		return false, fmt.Errorf("element %s", data.Error)
	}

	if !data.ReceivesEvents && data.Reason != "" {
		return false, fmt.Errorf("element does not receive events: %s", data.Reason)
	}

	return data.ReceivesEvents, nil
}

// CheckEnabled verifies the element is not disabled.
// An element is disabled if:
// - It has the [disabled] attribute
// - It has aria-disabled="true"
// - It's inside a disabled <fieldset>
func CheckEnabled(client *bidi.Client, context, selector string) (bool, error) {
	script := `
		(selector) => {
			const el = document.querySelector(selector);
			if (!el) return JSON.stringify({ error: 'not found' });

			// Check disabled attribute
			if (el.disabled === true) {
				return JSON.stringify({ enabled: false, reason: 'disabled attribute' });
			}

			// Check aria-disabled
			if (el.getAttribute('aria-disabled') === 'true') {
				return JSON.stringify({ enabled: false, reason: 'aria-disabled' });
			}

			// Check if inside disabled fieldset
			const fieldset = el.closest('fieldset[disabled]');
			if (fieldset) {
				// Exception: elements in the first legend are not disabled
				const legend = fieldset.querySelector('legend');
				if (!legend || !legend.contains(el)) {
					return JSON.stringify({ enabled: false, reason: 'inside disabled fieldset' });
				}
			}

			return JSON.stringify({ enabled: true });
		}
	`

	result, err := callCheckFunction(client, context, selector, script)
	if err != nil {
		return false, err
	}

	var data struct {
		Enabled bool   `json:"enabled"`
		Reason  string `json:"reason,omitempty"`
		Error   string `json:"error,omitempty"`
	}
	if err := json.Unmarshal([]byte(result), &data); err != nil {
		return false, fmt.Errorf("failed to parse enabled result: %w", err)
	}

	if data.Error != "" {
		return false, fmt.Errorf("element %s", data.Error)
	}

	if !data.Enabled && data.Reason != "" {
		return false, fmt.Errorf("element not enabled: %s", data.Reason)
	}

	return data.Enabled, nil
}

// CheckEditable verifies the element can accept text input.
// An element is editable if:
// - It is enabled (not disabled)
// - It does not have [readonly] attribute
// - It does not have aria-readonly="true"
// - For contenteditable, it must be "true" or ""
func CheckEditable(client *bidi.Client, context, selector string) (bool, error) {
	// First check if enabled
	enabled, err := CheckEnabled(client, context, selector)
	if err != nil {
		return false, err
	}
	if !enabled {
		return false, nil
	}

	script := `
		(selector) => {
			const el = document.querySelector(selector);
			if (!el) return JSON.stringify({ error: 'not found' });

			// Check readonly attribute
			if (el.readOnly === true) {
				return JSON.stringify({ editable: false, reason: 'readonly attribute' });
			}

			// Check aria-readonly
			if (el.getAttribute('aria-readonly') === 'true') {
				return JSON.stringify({ editable: false, reason: 'aria-readonly' });
			}

			// For input/textarea, check if it's a type that accepts text
			const tag = el.tagName.toLowerCase();
			if (tag === 'input') {
				const type = (el.type || 'text').toLowerCase();
				const textTypes = ['text', 'password', 'email', 'number', 'search', 'tel', 'url'];
				if (!textTypes.includes(type)) {
					return JSON.stringify({ editable: false, reason: 'input type ' + type + ' not editable' });
				}
			}

			// Check contenteditable
			if (el.isContentEditable) {
				return JSON.stringify({ editable: true });
			}

			// For form elements, they're editable if we got here
			if (tag === 'input' || tag === 'textarea') {
				return JSON.stringify({ editable: true });
			}

			// Non-form elements without contenteditable are not editable
			return JSON.stringify({ editable: false, reason: 'not a form element or contenteditable' });
		}
	`

	result, err := callCheckFunction(client, context, selector, script)
	if err != nil {
		return false, err
	}

	var data struct {
		Editable bool   `json:"editable"`
		Reason   string `json:"reason,omitempty"`
		Error    string `json:"error,omitempty"`
	}
	if err := json.Unmarshal([]byte(result), &data); err != nil {
		return false, fmt.Errorf("failed to parse editable result: %w", err)
	}

	if data.Error != "" {
		return false, fmt.Errorf("element %s", data.Error)
	}

	if !data.Editable && data.Reason != "" {
		return false, fmt.Errorf("element not editable: %s", data.Reason)
	}

	return data.Editable, nil
}

// CheckAll runs all actionability checks and returns the results.
func CheckAll(client *bidi.Client, context, selector string) (*ActionabilityResult, error) {
	result := &ActionabilityResult{}

	var err error

	result.Visible, err = CheckVisible(client, context, selector)
	if err != nil {
		return nil, fmt.Errorf("visible check failed: %w", err)
	}

	result.Stable, err = CheckStable(client, context, selector)
	if err != nil {
		return nil, fmt.Errorf("stable check failed: %w", err)
	}

	result.ReceivesEvents, err = CheckReceivesEvents(client, context, selector)
	if err != nil {
		return nil, fmt.Errorf("receivesEvents check failed: %w", err)
	}

	result.Enabled, err = CheckEnabled(client, context, selector)
	if err != nil {
		return nil, fmt.Errorf("enabled check failed: %w", err)
	}

	result.Editable, err = CheckEditable(client, context, selector)
	if err != nil {
		return nil, fmt.Errorf("editable check failed: %w", err)
	}

	return result, nil
}

// callCheckFunction is a helper to execute a script and return the JSON string result.
func callCheckFunction(client *bidi.Client, context, selector, script string) (string, error) {
	if context == "" {
		tree, err := client.GetTree()
		if err != nil {
			return "", fmt.Errorf("failed to get browsing context: %w", err)
		}
		if len(tree.Contexts) == 0 {
			return "", fmt.Errorf("no browsing contexts available")
		}
		context = tree.Contexts[0].Context
	}

	params := map[string]interface{}{
		"functionDeclaration": script,
		"target":              map[string]interface{}{"context": context},
		"arguments": []map[string]interface{}{
			{"type": "string", "value": selector},
		},
		"awaitPromise":    false,
		"resultOwnership": "root",
	}

	msg, err := client.SendCommand("script.callFunction", params)
	if err != nil {
		return "", err
	}

	// Parse the result
	var callResult struct {
		Type   string          `json:"type"`
		Result json.RawMessage `json:"result"`
	}
	if err := json.Unmarshal(msg.Result, &callResult); err != nil {
		return "", fmt.Errorf("failed to parse script.callFunction result: %w", err)
	}

	if callResult.Type == "exception" {
		return "", fmt.Errorf("script exception: %s", string(callResult.Result))
	}

	// Parse the remote value (string containing JSON)
	var remoteValue struct {
		Type  string `json:"type"`
		Value string `json:"value,omitempty"`
	}

	if err := json.Unmarshal(callResult.Result, &remoteValue); err != nil {
		return "", fmt.Errorf("failed to parse remote value: %w", err)
	}

	return remoteValue.Value, nil
}

// getBoundingBox returns the element's bounding box coordinates.
func getBoundingBox(client *bidi.Client, context, selector string) (*bidi.BoxInfo, error) {
	script := `
		(selector) => {
			const el = document.querySelector(selector);
			if (!el) return JSON.stringify({ error: 'not found' });

			const rect = el.getBoundingClientRect();
			return JSON.stringify({
				x: rect.x,
				y: rect.y,
				width: rect.width,
				height: rect.height
			});
		}
	`

	result, err := callCheckFunction(client, context, selector, script)
	if err != nil {
		return nil, err
	}

	var data struct {
		X      float64 `json:"x"`
		Y      float64 `json:"y"`
		Width  float64 `json:"width"`
		Height float64 `json:"height"`
		Error  string  `json:"error,omitempty"`
	}
	if err := json.Unmarshal([]byte(result), &data); err != nil {
		return nil, fmt.Errorf("failed to parse bounding box: %w", err)
	}

	if data.Error != "" {
		return nil, fmt.Errorf("element %s", data.Error)
	}

	return &bidi.BoxInfo{
		X:      data.X,
		Y:      data.Y,
		Width:  data.Width,
		Height: data.Height,
	}, nil
}

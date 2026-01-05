package bidi

import (
	"encoding/json"
	"fmt"
)

// Event names for browsingContext events.
const (
	EventNavigationStarted = "browsingContext.navigationStarted"
	EventDOMContentLoaded  = "browsingContext.domContentLoaded"
	EventLoad              = "browsingContext.load"
)

// NavigationInfoEvent represents the params for navigation-related events:
// browsingContext.navigationStarted, browsingContext.domContentLoaded, and browsingContext.load.
type NavigationInfoEvent struct {
	Context    string `json:"context"`
	Navigation string `json:"navigation"`
	Timestamp  int64  `json:"timestamp"`
	URL        string `json:"url"`
}

// ParseNavigationInfoEvent parses an Event's Params into a NavigationInfoEvent.
// Returns nil if the event is not a navigation event.
func ParseNavigationInfoEvent(event *Event) (*NavigationInfoEvent, error) {
	switch event.Method {
	case EventNavigationStarted, EventDOMContentLoaded, EventLoad:
		var info NavigationInfoEvent
		if err := json.Unmarshal(event.Params, &info); err != nil {
			return nil, fmt.Errorf("failed to parse navigation info event: %w", err)
		}
		return &info, nil
	default:
		return nil, fmt.Errorf("not a navigation event: %s", event.Method)
	}
}

// BrowsingContextInfo represents a browsing context in the tree.
type BrowsingContextInfo struct {
	Context  string                `json:"context"`
	URL      string                `json:"url"`
	Children []BrowsingContextInfo `json:"children,omitempty"`
	Parent   string                `json:"parent,omitempty"`
}

// GetTreeResult represents the result of browsingContext.getTree.
type GetTreeResult struct {
	Contexts []BrowsingContextInfo `json:"contexts"`
}

// GetTree returns the tree of browsing contexts.
func (c *Client) GetTree() (*GetTreeResult, error) {
	msg, err := c.SendCommand("browsingContext.getTree", map[string]interface{}{})
	if err != nil {
		return nil, err
	}

	var result GetTreeResult
	if err := json.Unmarshal(msg.Result, &result); err != nil {
		return nil, fmt.Errorf("failed to parse browsingContext.getTree result: %w", err)
	}

	return &result, nil
}

// NavigationInfo represents the result of a navigation.
type NavigationInfo struct {
	Navigation string `json:"navigation"`
	URL        string `json:"url"`
}

// NavigateResult represents the result of browsingContext.navigate.
type NavigateResult struct {
	Navigation string `json:"navigation"`
	URL        string `json:"url"`
}

// Navigate navigates a browsing context to a URL.
// If context is empty, it uses the first available context.
func (c *Client) Navigate(context, url string) (*NavigateResult, error) {
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
		"context": context,
		"url":     url,
		"wait":    "complete", // Wait for page load to complete
	}

	msg, err := c.SendCommand("browsingContext.navigate", params)
	if err != nil {
		return nil, err
	}

	var result NavigateResult
	if err := json.Unmarshal(msg.Result, &result); err != nil {
		return nil, fmt.Errorf("failed to parse browsingContext.navigate result: %w", err)
	}

	return &result, nil
}

// GetCurrentURL returns the URL of the first browsing context.
func (c *Client) GetCurrentURL() (string, error) {
	tree, err := c.GetTree()
	if err != nil {
		return "", err
	}
	if len(tree.Contexts) == 0 {
		return "", fmt.Errorf("no browsing contexts available")
	}
	return tree.Contexts[0].URL, nil
}

// CaptureScreenshotResult represents the result of browsingContext.captureScreenshot.
type CaptureScreenshotResult struct {
	Data string `json:"data"` // Base64-encoded PNG
}

// CaptureScreenshot captures a screenshot of the viewport.
// If context is empty, it uses the first available context.
// Returns base64-encoded PNG data.
func (c *Client) CaptureScreenshot(context string) (string, error) {
	// If no context provided, get the first one from the tree
	if context == "" {
		tree, err := c.GetTree()
		if err != nil {
			return "", fmt.Errorf("failed to get browsing context: %w", err)
		}
		if len(tree.Contexts) == 0 {
			return "", fmt.Errorf("no browsing contexts available")
		}
		context = tree.Contexts[0].Context
	}

	params := map[string]interface{}{
		"context": context,
	}

	msg, err := c.SendCommand("browsingContext.captureScreenshot", params)
	if err != nil {
		return "", err
	}

	var result CaptureScreenshotResult
	if err := json.Unmarshal(msg.Result, &result); err != nil {
		return "", fmt.Errorf("failed to parse browsingContext.captureScreenshot result: %w", err)
	}

	return result.Data, nil
}

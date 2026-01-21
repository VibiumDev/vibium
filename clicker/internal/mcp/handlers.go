package mcp

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"

	"github.com/vibium/clicker/internal/bidi"
	"github.com/vibium/clicker/internal/browser"
	"github.com/vibium/clicker/internal/features"
	"github.com/vibium/clicker/internal/log"
)

// Handlers manages browser session state and executes tool calls.
type Handlers struct {
	launchResult  *browser.LaunchResult
	client        *bidi.Client
	conn          *bidi.Connection
	screenshotDir string
}

// NewHandlers creates a new Handlers instance.
// screenshotDir specifies where screenshots are saved. If empty, file saving is disabled.
func NewHandlers(screenshotDir string) *Handlers {
	return &Handlers{
		screenshotDir: screenshotDir,
	}
}

// Call executes a tool by name with the given arguments.
func (h *Handlers) Call(name string, args map[string]interface{}) (*ToolsCallResult, error) {
	log.Debug("tool call", "name", name, "args", args)

	switch name {
	case "browser_launch":
		return h.browserLaunch(args)
	case "browser_navigate":
		return h.browserNavigate(args)
	case "browser_click":
		return h.browserClick(args)
	case "browser_type":
		return h.browserType(args)
	case "browser_screenshot":
		return h.browserScreenshot(args)
	case "browser_find":
		return h.browserFind(args)
	case "browser_quit":
		return h.browserQuit(args)
	default:
		return nil, fmt.Errorf("unknown tool: %s", name)
	}
}

// Close cleans up any active browser sessions.
func (h *Handlers) Close() {
	if h.conn != nil {
		h.conn.Close()
		h.conn = nil
	}
	if h.launchResult != nil {
		h.launchResult.Close()
		h.launchResult = nil
	}
	h.client = nil
}

// browserLaunch launches a new browser session.
func (h *Handlers) browserLaunch(args map[string]interface{}) (*ToolsCallResult, error) {
	// Close any existing session
	h.Close()

	// Parse options
	headless := false // Default: show browser for better first-time UX
	if val, ok := args["headless"].(bool); ok {
		headless = val
	}

	width := 0
	if val, ok := args["width"].(float64); ok {
		width = int(val)
	}

	height := 0
	if val, ok := args["height"].(float64); ok {
		height = int(val)
	}

	// Launch browser
	launchResult, err := browser.Launch(browser.LaunchOptions{
		Headless: headless,
		Width:    width,
		Height:   height,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to launch browser: %w", err)
	}

	// Connect to BiDi
	conn, err := bidi.Connect(launchResult.WebSocketURL)
	if err != nil {
		launchResult.Close()
		return nil, fmt.Errorf("failed to connect to browser: %w", err)
	}

	h.launchResult = launchResult
	h.conn = conn
	h.client = bidi.NewClient(conn)

	// Build result message
	msg := fmt.Sprintf("Browser launched (headless: %v", headless)
	if width > 0 || height > 0 {
		w := width
		if w == 0 {
			w = 1280
		}
		ht := height
		if ht == 0 {
			ht = 720
		}
		msg += fmt.Sprintf(", size: %dx%d", w, ht)
	}
	msg += ")"

	return &ToolsCallResult{
		Content: []Content{{
			Type: "text",
			Text: msg,
		}},
	}, nil
}

// browserNavigate navigates to a URL.
func (h *Handlers) browserNavigate(args map[string]interface{}) (*ToolsCallResult, error) {
	if err := h.ensureBrowser(); err != nil {
		return nil, err
	}

	url, ok := args["url"].(string)
	if !ok || url == "" {
		return nil, fmt.Errorf("url is required")
	}

	result, err := h.client.Navigate("", url)
	if err != nil {
		return nil, fmt.Errorf("failed to navigate: %w", err)
	}

	return &ToolsCallResult{
		Content: []Content{{
			Type: "text",
			Text: fmt.Sprintf("Navigated to %s", result.URL),
		}},
	}, nil
}

// browserClick clicks an element.
func (h *Handlers) browserClick(args map[string]interface{}) (*ToolsCallResult, error) {
	if err := h.ensureBrowser(); err != nil {
		return nil, err
	}

	selector, ok := args["selector"].(string)
	if !ok || selector == "" {
		return nil, fmt.Errorf("selector is required")
	}

	// Wait for element to be actionable
	opts := features.DefaultWaitOptions()
	if err := features.WaitForClick(h.client, "", selector, opts); err != nil {
		return nil, err
	}

	// Click the element
	if err := h.client.ClickElement("", selector); err != nil {
		return nil, fmt.Errorf("failed to click: %w", err)
	}

	return &ToolsCallResult{
		Content: []Content{{
			Type: "text",
			Text: fmt.Sprintf("Clicked element: %s", selector),
		}},
	}, nil
}

// browserType types text into an element.
func (h *Handlers) browserType(args map[string]interface{}) (*ToolsCallResult, error) {
	if err := h.ensureBrowser(); err != nil {
		return nil, err
	}

	selector, ok := args["selector"].(string)
	if !ok || selector == "" {
		return nil, fmt.Errorf("selector is required")
	}

	text, ok := args["text"].(string)
	if !ok {
		return nil, fmt.Errorf("text is required")
	}

	// Wait for element to be actionable
	opts := features.DefaultWaitOptions()
	if err := features.WaitForType(h.client, "", selector, opts); err != nil {
		return nil, err
	}

	// Type into the element
	if err := h.client.TypeIntoElement("", selector, text); err != nil {
		return nil, fmt.Errorf("failed to type: %w", err)
	}

	return &ToolsCallResult{
		Content: []Content{{
			Type: "text",
			Text: fmt.Sprintf("Typed into element: %s", selector),
		}},
	}, nil
}

// browserScreenshot captures a screenshot.
func (h *Handlers) browserScreenshot(args map[string]interface{}) (*ToolsCallResult, error) {
	if err := h.ensureBrowser(); err != nil {
		return nil, err
	}

	base64Data, err := h.client.CaptureScreenshot("")
	if err != nil {
		return nil, fmt.Errorf("failed to capture screenshot: %w", err)
	}

	// If filename provided, save to file (only if screenshotDir is configured)
	if filename, ok := args["filename"].(string); ok && filename != "" {
		if h.screenshotDir == "" {
			return nil, fmt.Errorf("screenshot file saving is disabled (use --screenshot-dir to enable)")
		}

		// Create directory if it doesn't exist
		if err := os.MkdirAll(h.screenshotDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create screenshot directory: %w", err)
		}

		// Use only the basename to prevent path traversal
		safeName := filepath.Base(filename)
		fullPath := filepath.Join(h.screenshotDir, safeName)

		pngData, err := base64.StdEncoding.DecodeString(base64Data)
		if err != nil {
			return nil, fmt.Errorf("failed to decode screenshot: %w", err)
		}
		if err := os.WriteFile(fullPath, pngData, 0644); err != nil {
			return nil, fmt.Errorf("failed to save screenshot: %w", err)
		}
		return &ToolsCallResult{
			Content: []Content{{
				Type: "text",
				Text: fmt.Sprintf("Screenshot saved to %s", fullPath),
			}},
		}, nil
	}

	return &ToolsCallResult{
		Content: []Content{{
			Type:     "image",
			Data:     base64Data,
			MimeType: "image/png",
		}},
	}, nil
}

// browserFind finds an element and returns its info.
func (h *Handlers) browserFind(args map[string]interface{}) (*ToolsCallResult, error) {
	if err := h.ensureBrowser(); err != nil {
		return nil, err
	}

	selector, ok := args["selector"].(string)
	if !ok || selector == "" {
		return nil, fmt.Errorf("selector is required")
	}

	info, err := h.client.FindElement("", selector)
	if err != nil {
		return nil, err
	}

	return &ToolsCallResult{
		Content: []Content{{
			Type: "text",
			Text: fmt.Sprintf("tag=%s, text=\"%s\", box={x:%.0f, y:%.0f, w:%.0f, h:%.0f}",
				info.Tag, info.Text, info.Box.X, info.Box.Y, info.Box.Width, info.Box.Height),
		}},
	}, nil
}

// browserQuit closes the browser session.
func (h *Handlers) browserQuit(args map[string]interface{}) (*ToolsCallResult, error) {
	if h.launchResult == nil {
		return &ToolsCallResult{
			Content: []Content{{
				Type: "text",
				Text: "No browser session to close",
			}},
		}, nil
	}

	h.Close()

	return &ToolsCallResult{
		Content: []Content{{
			Type: "text",
			Text: "Browser session closed",
		}},
	}, nil
}

// ensureBrowser checks that a browser session is active.
func (h *Handlers) ensureBrowser() error {
	if h.client == nil {
		return fmt.Errorf("no browser session. Call browser_launch first")
	}
	return nil
}

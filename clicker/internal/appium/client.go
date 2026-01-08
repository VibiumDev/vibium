package appium

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Client is a lightweight HTTP client for Appium/WebDriver.
type Client struct {
	BaseURL   string
	SessionID string
	HTTP      *http.Client
}

// NewClient creates a new Appium client.
func NewClient(url string) *Client {
	// Ensure URL has no trailing slash
	url = strings.TrimRight(url, "/")
	return &Client{
		BaseURL: url,
		HTTP: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// StartSession starts a new session with the given capabilities.
func (c *Client) StartSession(caps map[string]interface{}) (string, error) {
	reqBody := NewSessionRequest{
		Capabilities: Capabilities{
			AlwaysMatch: caps,
		},
	}

	var sessResp struct {
		Value struct {
			SessionID    string                 `json:"sessionId"`
			Capabilities map[string]interface{} `json:"capabilities"`
		} `json:"value"`
	}

	if err := c.post("/session", reqBody, &sessResp); err != nil {
		return "", err
	}

	c.SessionID = sessResp.Value.SessionID
	return c.SessionID, nil
}

// GetPageSource returns the current page source (XML).
func (c *Client) GetPageSource() (string, error) {
	var resp Response
	if err := c.get(fmt.Sprintf("/session/%s/source", c.SessionID), &resp); err != nil {
		return "", err
	}
	return resp.Value.(string), nil
}

// FindElement finds an element by strategy (e.g., "id", "xpath", "accessibility id").
func (c *Client) FindElement(strategy, selector string) (string, error) {
	reqBody := map[string]string{
		"using": strategy,
		"value": selector,
	}

	var resp struct {
		Value map[string]string `json:"value"`
	}

	if err := c.post(fmt.Sprintf("/session/%s/element", c.SessionID), reqBody, &resp); err != nil {
		return "", err
	}

	// Element ID key can vary (element-6066-11e4-a52e-4f735466cecf), but usually standard in JSON wire
	// We iterate to find the value
	for _, v := range resp.Value {
		return v, nil
	}
	return "", fmt.Errorf("element not found in response")
}

// ClickElement clicks the element with the given ID.
func (c *Client) ClickElement(elementID string) error {
	return c.post(fmt.Sprintf("/session/%s/element/%s/click", c.SessionID, elementID), nil, nil)
}

// TypeElement types text into the element.
func (c *Client) TypeElement(elementID, text string) error {
	reqBody := map[string]interface{}{
		"text": text,
		"value": strings.Split(text, ""), // WebDriver spec often expects an array of characters
	}
	return c.post(fmt.Sprintf("/session/%s/element/%s/value", c.SessionID, elementID), reqBody, nil)
}

// Quit closes the session.
func (c *Client) Quit() error {
	if c.SessionID == "" {
		return nil
	}
	if err := c.delete(fmt.Sprintf("/session/%s", c.SessionID)); err != nil {
		return err
	}
	c.SessionID = ""
	return nil
}

// Helpers

func (c *Client) post(path string, body interface{}, result interface{}) error {
	return c.do("POST", path, body, result)
}

func (c *Client) get(path string, result interface{}) error {
	return c.do("GET", path, nil, result)
}

func (c *Client) delete(path string) error {
	return c.do("DELETE", path, nil, nil)
}

func (c *Client) do(method, path string, body interface{}, result interface{}) error {
	url := c.BaseURL + path
	
	var bodyReader io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal body: %w", err)
		}
		bodyReader = bytes.NewBuffer(jsonBody)
	}

	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode >= 400 {
		return fmt.Errorf("status %d: %s", resp.StatusCode, string(respBody))
	}

	if result != nil {
		if err := json.Unmarshal(respBody, result); err != nil {
			return fmt.Errorf("failed to unmarshal response: %w", err)
		}
	}

	return nil
}

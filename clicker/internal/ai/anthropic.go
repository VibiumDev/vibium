package ai

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

const (
	anthropicAPIURL    = "https://api.anthropic.com/v1/messages"
	anthropicVersion   = "2023-06-01"
	anthropicBetaCache = "prompt-caching-2024-07-31"

	ModelHaiku  = "claude-haiku-4-5-20251001"
	ModelSonnet = "claude-sonnet-4-6"
)

// AnthropicResolver calls the Anthropic Messages API.
// Prompt caching is enabled: the image (if present) or the text block is marked
// ephemeral so repeated calls on the same page context get a cache hit.
type AnthropicResolver struct {
	apiKey  string
	model   string
	client  *http.Client
	baseURL string // overridable for tests; defaults to anthropicAPIURL
}

// NewAnthropicResolver creates a resolver for the given model.
// apiKey may be empty — if so, VIBIUM_ANTHROPIC_API_KEY is read at call time.
func NewAnthropicResolver(model, apiKey string) *AnthropicResolver {
	return &AnthropicResolver{
		apiKey: apiKey,
		model:  model,
		client: &http.Client{},
	}
}

// NewAnthropicResolverFromEnv creates a resolver reading the API key from
// VIBIUM_ANTHROPIC_API_KEY. Returns an error if the key is not set.
func NewAnthropicResolverFromEnv(model string) (*AnthropicResolver, error) {
	key := os.Getenv("VIBIUM_ANTHROPIC_API_KEY")
	if key == "" {
		return nil, fmt.Errorf("VIBIUM_ANTHROPIC_API_KEY is not set")
	}
	return NewAnthropicResolver(model, key), nil
}

// NewAnthropicResolverForTest creates a resolver pointed at a custom base URL,
// for use in unit tests with httptest.Server.
func NewAnthropicResolverForTest(model, apiKey, baseURL string) *AnthropicResolver {
	r := NewAnthropicResolver(model, apiKey)
	r.baseURL = baseURL
	return r
}

// Resolve sends the prompt (and optional screenshot) to the Anthropic API and
// returns the model's text response.
func (r *AnthropicResolver) Resolve(ctx context.Context, req Request) (Response, error) {
	body, err := r.buildRequest(req)
	if err != nil {
		return Response{}, fmt.Errorf("build request: %w", err)
	}

	url := r.baseURL
	if url == "" {
		url = anthropicAPIURL
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return Response{}, fmt.Errorf("create request: %w", err)
	}

	apiKey := r.apiKey
	if apiKey == "" {
		apiKey = os.Getenv("VIBIUM_ANTHROPIC_API_KEY")
	}
	if apiKey == "" {
		return Response{}, fmt.Errorf("VIBIUM_ANTHROPIC_API_KEY is not set")
	}

	httpReq.Header.Set("x-api-key", apiKey)
	httpReq.Header.Set("anthropic-version", anthropicVersion)
	httpReq.Header.Set("anthropic-beta", anthropicBetaCache)
	httpReq.Header.Set("content-type", "application/json")

	resp, err := r.client.Do(httpReq)
	if err != nil {
		return Response{}, fmt.Errorf("call API: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return Response{}, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return Response{}, fmt.Errorf("API error %d: %s", resp.StatusCode, respBody)
	}

	return parseResponse(respBody)
}

// buildRequest constructs the Anthropic Messages API request body.
// When an image is present it is sent first with cache_control so the vision
// context is cached across multiple calls on the same screenshot.
// When text-only, the text block itself is cached.
func (r *AnthropicResolver) buildRequest(req Request) ([]byte, error) {
	type cacheControl struct {
		Type string `json:"type"`
	}
	type textBlock struct {
		Type         string        `json:"type"`
		Text         string        `json:"text"`
		CacheControl *cacheControl `json:"cache_control,omitempty"`
	}
	type imageSource struct {
		Type      string `json:"type"`
		MediaType string `json:"media_type"`
		Data      string `json:"data"`
	}
	type imageBlock struct {
		Type         string        `json:"type"`
		Source       imageSource   `json:"source"`
		CacheControl *cacheControl `json:"cache_control,omitempty"`
	}

	var content []any

	if len(req.Image) > 0 {
		// Image cached; prompt text is not (it's the variable part)
		content = []any{
			imageBlock{
				Type: "image",
				Source: imageSource{
					Type:      "base64",
					MediaType: "image/png",
					Data:      base64.StdEncoding.EncodeToString(req.Image),
				},
				CacheControl: &cacheControl{Type: "ephemeral"},
			},
			textBlock{Type: "text", Text: req.Prompt},
		}
	} else {
		// Text-only: cache the prompt (a11y tree / map data is stable per page)
		content = []any{
			textBlock{
				Type:         "text",
				Text:         req.Prompt,
				CacheControl: &cacheControl{Type: "ephemeral"},
			},
		}
	}

	payload := map[string]any{
		"model":      r.model,
		"max_tokens": 256,
		"messages": []map[string]any{
			{"role": "user", "content": content},
		},
	}

	return json.Marshal(payload)
}

type anthropicResponse struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

func parseResponse(body []byte) (Response, error) {
	var r anthropicResponse
	if err := json.Unmarshal(body, &r); err != nil {
		return Response{}, fmt.Errorf("parse response: %w", err)
	}
	if r.Error != nil {
		return Response{}, fmt.Errorf("API error: %s", r.Error.Message)
	}
	for _, block := range r.Content {
		if block.Type == "text" {
			return Response{Text: block.Text}, nil
		}
	}
	return Response{}, fmt.Errorf("no text content in response")
}

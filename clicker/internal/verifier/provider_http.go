package verifier

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// post returns the response body and its Content-Type media type, so parse
// failures can say what the endpoint answered with. The media type is not
// checked here: providers that omit or mislabel it but answer valid JSON
// keep working.
func (v *Model) post(ctx context.Context, endpoint string, payload interface{}, headers map[string]string) ([]byte, string, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, "", fmt.Errorf("encode verifier request")
	}
	if len(body) > 8*1024*1024 {
		return nil, "", fmt.Errorf("verifier context payload limit reached")
	}
	request, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, "", fmt.Errorf("invalid verifier endpoint")
	}
	request.Header.Set("Content-Type", "application/json")
	for key, value := range headers {
		request.Header.Set(key, value)
	}
	client := v.Client
	if client == nil {
		client = &http.Client{Timeout: Timeout, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	}
	response, err := client.Do(request)
	if err != nil {
		if ctx.Err() != nil {
			return nil, "", fmt.Errorf("verification timeout: %w", ctx.Err())
		}
		return nil, "", fmt.Errorf("verifier provider request failed (check endpoint and connectivity)")
	}
	defer response.Body.Close()
	// Never echo provider bodies: they can contain credentials or reasoning.
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		var detail struct {
			Error struct {
				Code  string `json:"code"`
				Param string `json:"param"`
			} `json:"error"`
		}
		json.NewDecoder(io.LimitReader(response.Body, 8192)).Decode(&detail)
		// Report only recognized protocol error codes and request field names,
		// never provider-supplied prose (which may echo secrets or model content).
		suffix := ""
		switch detail.Error.Code {
		case "invalid_function_parameters", "unsupported_parameter", "unsupported_value", "model_not_found", "insufficient_quota", "rate_limit_exceeded":
			suffix = " (" + detail.Error.Code + ")"
		}
		field := strings.Split(strings.Split(detail.Error.Param, ".")[0], "[")[0]
		switch field {
		case "model", "messages", "tools", "max_completion_tokens", "parallel_tool_calls", "reasoning_effort":
			suffix += " in " + field
		}
		return nil, "", fmt.Errorf("verifier provider returned HTTP %d%s", response.StatusCode, suffix)
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, 1024*1024+1))
	if err != nil || len(data) > 1024*1024 {
		return nil, "", fmt.Errorf("invalid or oversized verifier response")
	}
	contentType := strings.ToLower(strings.TrimSpace(strings.SplitN(response.Header.Get("Content-Type"), ";", 2)[0]))
	return data, contentType, nil
}

// invalidProviderResponse says what the endpoint answered with when a reply
// cannot be parsed, without echoing the body. Media types come from a small
// allowlist and the hint names the setting that chose the endpoint, never
// its value, matching the value-free discipline of Config.Checks.
func invalidProviderResponse(config Config, contentType, want string) error {
	got := "JSON that is not " + want
	if contentType != "" && contentType != "application/json" && !strings.HasSuffix(contentType, "+json") {
		switch contentType {
		case "text/html", "text/plain", "text/xml", "application/xml", "text/event-stream", "application/octet-stream":
			got = contentType + ", not JSON"
		default:
			got = "a non-JSON content type"
		}
	}
	hint := ""
	if config.BaseURL != "" {
		hint = "; check --base-url / VIBIUM_AI_BASE_URL"
	}
	return fmt.Errorf("AI provider returned %s%s", got, hint)
}

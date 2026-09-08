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

func (v *Model) post(ctx context.Context, endpoint string, payload interface{}, headers map[string]string) ([]byte, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("encode verifier request")
	}
	if len(body) > 8*1024*1024 {
		return nil, fmt.Errorf("verifier context payload limit reached")
	}
	request, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("invalid verifier endpoint")
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
			return nil, fmt.Errorf("verification timeout: %w", ctx.Err())
		}
		return nil, fmt.Errorf("verifier provider request failed (check endpoint and connectivity)")
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
		return nil, fmt.Errorf("verifier provider returned HTTP %d%s", response.StatusCode, suffix)
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, 1024*1024+1))
	if err != nil || len(data) > 1024*1024 {
		return nil, fmt.Errorf("invalid or oversized verifier response")
	}
	return data, nil
}

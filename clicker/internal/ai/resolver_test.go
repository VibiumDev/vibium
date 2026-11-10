package ai_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/vibium/clicker/internal/ai"
)

// mockResolver is a test double that returns a fixed response.
type mockResolver struct {
	text string
	err  error
}

func (m *mockResolver) Resolve(_ context.Context, _ ai.Request) (ai.Response, error) {
	return ai.Response{Text: m.text}, m.err
}

func TestMockResolver(t *testing.T) {
	r := &mockResolver{text: "@e3"}
	resp, err := r.Resolve(context.Background(), ai.Request{Prompt: "click login"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Text != "@e3" {
		t.Fatalf("got %q, want %q", resp.Text, "@e3")
	}
}

func TestAnthropicResolver_TextOnly(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("x-api-key") == "" {
			http.Error(w, "missing api key", http.StatusUnauthorized)
			return
		}
		w.Header().Set("content-type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"content": []map[string]any{
				{"type": "text", "text": "@e2"},
			},
		})
	}))
	defer srv.Close()

	// Point resolver at test server by temporarily overriding the URL.
	// We test the full request/response cycle without hitting the real API.
	r := ai.NewAnthropicResolverForTest(ai.ModelHaiku, "test-key", srv.URL)
	resp, err := r.Resolve(context.Background(), ai.Request{Prompt: "click login"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Text != "@e2" {
		t.Fatalf("got %q, want %q", resp.Text, "@e2")
	}
}

func TestAnthropicResolver_WithImage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)

		// Verify image block is present and cached
		msgs := body["messages"].([]any)
		content := msgs[0].(map[string]any)["content"].([]any)
		imgBlock := content[0].(map[string]any)
		if imgBlock["type"] != "image" {
			http.Error(w, "expected image block first", http.StatusBadRequest)
			return
		}
		cc := imgBlock["cache_control"].(map[string]any)
		if cc["type"] != "ephemeral" {
			http.Error(w, "expected cache_control ephemeral on image", http.StatusBadRequest)
			return
		}

		w.Header().Set("content-type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"content": []map[string]any{
				{"type": "text", "text": "@e5"},
			},
		})
	}))
	defer srv.Close()

	r := ai.NewAnthropicResolverForTest(ai.ModelSonnet, "test-key", srv.URL)
	resp, err := r.Resolve(context.Background(), ai.Request{
		Prompt: "which element is the submit button?",
		Image:  []byte{0x89, 0x50, 0x4e, 0x47}, // minimal PNG header
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Text != "@e5" {
		t.Fatalf("got %q, want %q", resp.Text, "@e5")
	}
}

func TestAnthropicResolver_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":{"message":"invalid api key"}}`))
	}))
	defer srv.Close()

	r := ai.NewAnthropicResolverForTest(ai.ModelHaiku, "bad-key", srv.URL)
	_, err := r.Resolve(context.Background(), ai.Request{Prompt: "click login"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestAnthropicResolver_MissingAPIKey(t *testing.T) {
	os.Unsetenv("VIBIUM_ANTHROPIC_API_KEY")
	_, err := ai.NewAnthropicResolverFromEnv(ai.ModelHaiku)
	if err == nil {
		t.Fatal("expected error for missing API key, got nil")
	}
}

// TestAnthropicResolver_Integration calls the real Anthropic API.
// Skipped unless VIBIUM_ANTHROPIC_API_KEY is set.
func TestAnthropicResolver_Integration(t *testing.T) {
	key := os.Getenv("VIBIUM_ANTHROPIC_API_KEY")
	if key == "" {
		t.Skip("VIBIUM_ANTHROPIC_API_KEY not set")
	}

	r := ai.NewAnthropicResolver(ai.ModelHaiku, key)
	resp, err := r.Resolve(context.Background(), ai.Request{
		Prompt: "Reply with exactly: @e1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Text == "" {
		t.Fatal("expected non-empty response")
	}
	t.Logf("response: %s", resp.Text)
}

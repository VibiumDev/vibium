package verifier

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestPerCallConfigResolution(t *testing.T) {
	for _, role := range []string{"run", "check"} {
		t.Run(role, func(t *testing.T) {
			prefix := (Config{Role: role}).Prefix()
			t.Setenv(prefix+"PROVIDER", "openai")
			t.Setenv(prefix+"MODEL", "original-model")
			t.Setenv(prefix+"BASE_URL", "https://original.example/v1")
			t.Setenv(prefix+"REASONING_EFFORT", "high")
			t.Setenv("OPENAI_API_KEY", "original-key")
			t.Setenv("ANTHROPIC_API_KEY", "anthropic-key")
			cases := []struct {
				name                                   string
				params                                 map[string]interface{}
				provider, model, endpoint, effort, key string
				fails                                  bool
			}{
				{"defaults", nil, "openai", "original-model", "https://original.example/v1", "high", "original-key", false},
				{"same provider", map[string]interface{}{"provider": "openai"}, "openai", "original-model", "https://original.example/v1", "high", "original-key", false},
				{"model only", map[string]interface{}{"model": "eval-model"}, "openai", "eval-model", "https://original.example/v1", "high", "original-key", false},
				{"native switch", map[string]interface{}{"provider": "anthropic", "model": "claude"}, "anthropic", "claude", "https://api.anthropic.com/v1", "", "anthropic-key", false},
				{"local switch", map[string]interface{}{"provider": "local", "model": "local-model"}, "local", "local-model", "http://127.0.0.1:8080/v1", "", "original-key", false},
				{"empty resets", map[string]interface{}{"baseURL": "", "reasoningEffort": ""}, "openai", "original-model", "https://api.openai.com/v1", "", "original-key", false},
				{name: "switch needs model", params: map[string]interface{}{"provider": "anthropic"}, fails: true},
				{name: "compatible needs endpoint", params: map[string]interface{}{"provider": "openai-compatible", "model": "other"}, fails: true},
				{name: "empty model", params: map[string]interface{}{"model": ""}, fails: true},
				{name: "invalid provider", params: map[string]interface{}{"provider": "PRIVATE-SETTING"}, fails: true},
				{name: "null model", params: map[string]interface{}{"model": nil}, fails: true},
				{name: "numeric endpoint", params: map[string]interface{}{"baseURL": 123}, fails: true},
				{name: "credentials in endpoint", params: map[string]interface{}{"baseURL": "https://user:PRIVATE-SETTING@example.com"}, fails: true},
			}
			for _, row := range cases {
				t.Run(row.name, func(t *testing.T) {
					c, err := ConfigFromParams(role, row.params)
					if row.fails {
						if err == nil || strings.Contains(err.Error(), "PRIVATE-SETTING") {
							t.Fatalf("unsafe or missing error: %v", err)
						}
						return
					}
					if err != nil || c.Provider != row.provider || c.Model != row.model || c.Endpoint() != row.endpoint || c.ReasoningEffort != row.effort || c.APIKey != row.key {
						t.Fatalf("incorrect resolution: %v", err)
					}
				})
			}
			original, err := ConfigForRole(role)
			if err != nil || original.Model != "original-model" || original.Provider != "openai" || original.ReasoningEffort != "high" {
				t.Fatal("override changed later invocations")
			}
			data, _ := json.Marshal(original.RecordingMetadata())
			if strings.Contains(string(data), "original-key") || strings.Contains(string(data), "original.example") {
				t.Fatal("private config entered metadata")
			}
		})
	}
}

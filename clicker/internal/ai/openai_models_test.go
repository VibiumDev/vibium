package ai

import (
	"strings"
	"testing"
)

func TestOpenAIEffortDefaultPerModelFamily(t *testing.T) {
	for _, tc := range []struct {
		model, effort string
		unsupported   bool
	}{
		{"gpt-5.6-sol", "none", false},
		{"gpt-5-mini", "none", false},
		{"gpt-6-sol", "none", false},
		{"gpt-6.1-sol", "", true},
		{" gpt-6.1-sol ", "", true},
		{"gpt-4.1-mini", "", false},
		{"o4-mini", "", false},
		{"", "", false},
	} {
		if got := OpenAIEffortDefault(tc.model); got != tc.effort {
			t.Errorf("OpenAIEffortDefault(%q) = %q, want %q", tc.model, got, tc.effort)
		}
		problem := OpenAIChatToolsProblem(tc.model)
		if (problem != "") != tc.unsupported {
			t.Errorf("OpenAIChatToolsProblem(%q) = %q", tc.model, problem)
		}
		if tc.unsupported && !strings.Contains(problem, strings.TrimSpace(tc.model)) {
			t.Errorf("problem does not name the model: %q", problem)
		}
	}
}

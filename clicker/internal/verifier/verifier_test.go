package verifier

import "testing"

// Validation and help read the same provider and effort lists (#581).
func TestProviderAndEffortChecks(t *testing.T) {
	problem := func(c Config, variable string) string {
		for _, check := range c.Checks() {
			if check.Variable == variable {
				return check.Error
			}
		}
		t.Fatalf("no %s check", variable)
		return ""
	}
	if got, want := problem(Config{Provider: "nope"}, "VIBIUM_AI_PROVIDER"), "set VIBIUM_AI_PROVIDER to openai, xai, anthropic, google, openai-compatible, or local"; got != want {
		t.Errorf("provider problem %q, want %q", got, want)
	}
	for _, p := range Providers {
		if got := problem(Config{Provider: p.Name}, "VIBIUM_AI_PROVIDER"); got != "" {
			t.Errorf("%s rejected: %s", p.Name, got)
		}
		for _, effort := range ReasoningEfforts {
			rejected := problem(Config{Provider: p.Name, ReasoningEffort: effort}, "VIBIUM_AI_REASONING_EFFORT") != ""
			if rejected == p.ReasoningEffort {
				t.Errorf("%s with effort %s: rejected=%v", p.Name, effort, rejected)
			}
		}
	}
	if problem(Config{Provider: "openai", ReasoningEffort: "extreme"}, "VIBIUM_AI_REASONING_EFFORT") == "" {
		t.Error("unknown effort accepted")
	}
	if problem(Config{Provider: "anthropic"}, "VIBIUM_AI_REASONING_EFFORT") != "" {
		t.Error("empty effort rejected for anthropic")
	}
}

func TestOrList(t *testing.T) {
	for _, tc := range []struct {
		in   []string
		want string
	}{{nil, ""}, {[]string{"a"}, "a"}, {[]string{"a", "b"}, "a or b"}, {[]string{"a", "b", "c"}, "a, b, or c"}} {
		if got := OrList(tc.in); got != tc.want {
			t.Errorf("OrList(%v) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

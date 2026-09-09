package verifier

import "testing"

// Live Anthropic runs return the verdict wrapped in a Markdown fence, often
// after a sentence of prose, which the strict JSON parse rejected — so every
// check against that provider died with "invalid JSON verdict" while OpenAI
// passed.
func TestStripJSONFence(t *testing.T) {
	verdict := `{"status": "failed", "summary": "total shows NaN", "evidence": []}`

	cases := []struct {
		name, in, want string
	}{
		{"bare JSON unchanged", verdict, verdict},
		{"json fence", "```json\n" + verdict + "\n```", verdict},
		{"bare fence", "```\n" + verdict + "\n```", verdict},
		{"fence with trailing newline", "```json\n" + verdict + "\n```\n", verdict},
		// The shape a live Anthropic check actually returned.
		{"prose before the fence", "The evidence is clear. Here is the result:\n\n```json\n" + verdict + "\n```", verdict},
		{"prose after the fence", "```json\n" + verdict + "\n```\nLet me know if you need more.", verdict},
		{"unterminated fence unchanged", "```json\n" + verdict, "```json\n" + verdict},
		{"one-line fence unchanged", "```json " + verdict + "```", "```json " + verdict + "```"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := StripJSONFence(c.in); got != c.want {
				t.Errorf("StripJSONFence(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}

// The end-to-end path: a fenced verdict must parse, matching what a live
// Anthropic check returns.
func TestParseResultAcceptsFencedVerdict(t *testing.T) {
	fenced := "Here is the result:\n\n```json\n{\"status\": \"failed\", \"summary\": \"cart total shows NaN\", \"evidence\": [{\"type\": \"observation\", \"summary\": \"total reads $NaN\"}]}\n```"
	result, err := parseResult(message{Content: fenced}, "does the cart work?")
	if err != nil {
		t.Fatalf("parseResult: %v", err)
	}
	if result.Status != "failed" || len(result.Evidence) != 1 {
		t.Errorf("got status %q with %d evidence items, want failed with 1", result.Status, len(result.Evidence))
	}
}

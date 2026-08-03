package api

import (
	"errors"
	"testing"
)

func TestPromptTrackerObserve(t *testing.T) {
	tr := NewPromptTracker()

	if _, open := tr.OpenPrompt("ctx-1"); open {
		t.Fatal("a fresh tracker should report no open prompt")
	}

	tr.Observe([]byte(`{"method":"browsingContext.userPromptOpened","params":{"context":"ctx-1","type":"alert","message":"hi"}}`))
	promptType, open := tr.OpenPrompt("ctx-1")
	if !open {
		t.Fatal("userPromptOpened should mark the context blocked")
	}
	if promptType != "alert" {
		t.Errorf("prompt type = %q, want %q", promptType, "alert")
	}

	// A different context is unaffected.
	if _, open := tr.OpenPrompt("ctx-2"); open {
		t.Error("a prompt in ctx-1 should not block ctx-2")
	}

	tr.Observe([]byte(`{"method":"browsingContext.userPromptClosed","params":{"context":"ctx-1","accepted":true,"type":"alert"}}`))
	if _, open := tr.OpenPrompt("ctx-1"); open {
		t.Error("userPromptClosed should clear the block")
	}
}

func TestPromptTrackerIgnoresOtherEvents(t *testing.T) {
	tr := NewPromptTracker()
	for _, raw := range []string{
		`{"method":"browsingContext.load","params":{"context":"ctx-1"}}`,
		`{"method":"network.beforeRequestSent","params":{"context":"ctx-1"}}`,
		`{"id":1,"type":"success","result":{}}`,
		`not json at all`,
		`{"method":"browsingContext.userPromptOpened","params":{}}`,
	} {
		tr.Observe([]byte(raw))
	}
	if _, open := tr.OpenPrompt("ctx-1"); open {
		t.Error("only userPromptOpened with a context should block")
	}
}

func TestPromptTrackerClear(t *testing.T) {
	tr := NewPromptTracker()
	tr.Observe([]byte(`{"method":"browsingContext.userPromptOpened","params":{"context":"ctx-1","type":"confirm"}}`))
	tr.Clear("ctx-1")
	if _, open := tr.OpenPrompt("ctx-1"); open {
		t.Error("Clear should drop the block, so a destroyed context cannot block forever")
	}
}

func TestCheckPromptBlocked(t *testing.T) {
	tr := NewPromptTracker()
	tr.Observe([]byte(`{"method":"browsingContext.userPromptOpened","params":{"context":"ctx-1","type":"alert"}}`))

	tests := []struct {
		name    string
		method  string
		params  map[string]interface{}
		blocked bool
	}{
		{
			name:    "script.callFunction on a blocked context",
			method:  "script.callFunction",
			params:  map[string]interface{}{"target": map[string]interface{}{"context": "ctx-1"}},
			blocked: true,
		},
		{
			name:    "input.performActions on a blocked context",
			method:  "input.performActions",
			params:  map[string]interface{}{"context": "ctx-1"},
			blocked: true,
		},
		{
			name:    "navigate on a blocked context",
			method:  "browsingContext.navigate",
			params:  map[string]interface{}{"context": "ctx-1"},
			blocked: true,
		},
		{
			// This is the command that clears the block; blocking it would be a
			// deadlock.
			name:    "handleUserPrompt is never blocked",
			method:  "browsingContext.handleUserPrompt",
			params:  map[string]interface{}{"context": "ctx-1"},
			blocked: false,
		},
		{
			name:    "getTree is not prompt-sensitive",
			method:  "browsingContext.getTree",
			params:  map[string]interface{}{"context": "ctx-1"},
			blocked: false,
		},
		{
			name:    "an unrelated context is not blocked",
			method:  "script.callFunction",
			params:  map[string]interface{}{"target": map[string]interface{}{"context": "ctx-2"}},
			blocked: false,
		},
		{
			name:    "no context means no decision",
			method:  "script.callFunction",
			params:  map[string]interface{}{},
			blocked: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := checkPromptBlocked(tr, tt.method, tt.params)
			if tt.blocked {
				var blockErr *BlockedByPromptError
				if !errors.As(err, &blockErr) {
					t.Fatalf("error = %v, want *BlockedByPromptError", err)
				}
				if blockErr.PromptType != "alert" {
					t.Errorf("PromptType = %q, want %q", blockErr.PromptType, "alert")
				}
				return
			}
			if err != nil {
				t.Errorf("error = %v, want nil", err)
			}
		})
	}
}

func TestCheckPromptBlockedNilTracker(t *testing.T) {
	if err := checkPromptBlocked(nil, "script.callFunction", map[string]interface{}{"context": "ctx-1"}); err != nil {
		t.Errorf("a nil tracker should never block: %v", err)
	}
}

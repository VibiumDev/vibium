package agent

import (
	"context"
	"github.com/vibium/clicker/internal/api"
	"os"
	"path/filepath"
	"testing"
)

func TestVerifierToolBoundary(t *testing.T) {
	v := &verifyTools{h: &Handlers{}}
	for _, tc := range []struct {
		name string
		args map[string]interface{}
	}{
		{"browser_evaluate", map[string]interface{}{"expression": "1"}},
		{"browser_start", map[string]interface{}{}},
		{"browser_screenshot", map[string]interface{}{"filename": "/tmp/forbidden.png"}},
		{"browser_screenshot", map[string]interface{}{"annotate": true}},
		{"browser_map", map[string]interface{}{"page": "other-page"}},
		{"browser_navigate", map[string]interface{}{"url": "file:///tmp/secret"}},
		{"browser_navigate", map[string]interface{}{"url": "javascript:alert(1)"}},
		{"browser_fill", map[string]interface{}{"selector": "input", "text": false}},
		{"browser_click", map[string]interface{}{"selector": "button", "timeout": float64(-1)}},
		{"browser_click", map[string]interface{}{}},
		{"browser_scroll", map[string]interface{}{"amount": float64(100)}},
	} {
		if _, err := v.Execute(context.Background(), tc.name, tc.args); err == nil {
			t.Errorf("accepted %s %v", tc.name, tc.args)
		}
	}
	// Filtering must not mutate the schemas used by existing clients.
	for _, tool := range GetToolSchemas() {
		if tool.Name == "browser_screenshot" {
			if _, ok := tool.InputSchema["properties"].(map[string]interface{})["filename"]; !ok {
				t.Fatal("mutated shared schema")
			}
		}
	}
}

func TestVerifyRecordingPreservesActiveRecorder(t *testing.T) {
	dir := t.TempDir()
	original := filepath.Join(dir, "workflow.zip")
	recorder := api.NewRecorder()
	recorder.Start(api.RecordingStartOptions{Name: "builder", Path: original}, nil)
	group := recorder.StartGroup("Builder workflow")
	h := &Handlers{recorder: recorder}
	if _, err := h.startVerifyRecording(original); err == nil {
		t.Fatal("allowed active recording destination")
	}
	output := filepath.Join(dir, "verification.zip")
	finish, err := h.startVerifyRecording(output)
	if err != nil {
		t.Fatal(err)
	}
	recorder.StartGroup("Verify: claim")
	recorder.StopGroup()
	if err := finish(); err != nil {
		t.Fatal(err)
	}
	if h.recorder != recorder || !recorder.IsRecording() {
		t.Fatal("interrupted caller recording")
	}
	if recorder.Options().Path != original {
		t.Fatal("redirected original recording")
	}
	if _, err := os.Stat(original); !os.IsNotExist(err) {
		t.Fatal("wrote caller destination prematurely")
	}
	data, err := os.ReadFile(output)
	if err != nil || len(data) == 0 {
		t.Fatal("missing output archive", err)
	}
	if _, err := h.startVerifyRecording(output); err == nil {
		t.Fatal("overwrote existing artifact")
	}
	// The group stack still belongs to the caller, and can be closed normally.
	recorder.SetGroupTitle(group, "Builder continued")
	recorder.StopGroup()
	if _, err := recorder.Stop(); err != nil {
		t.Fatal(err)
	}
}

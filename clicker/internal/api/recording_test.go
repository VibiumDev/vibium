package api

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"io"
	"strings"
	"testing"
	"time"
)

// traceLines unzips a recording and returns the decoded trace.trace events.
func traceLines(t *testing.T, zipData []byte) []map[string]interface{} {
	t.Helper()

	zr, err := zip.NewReader(bytes.NewReader(zipData), int64(len(zipData)))
	if err != nil {
		t.Fatalf("open recording zip: %v", err)
	}
	for _, f := range zr.File {
		if f.Name != "trace.trace" {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			t.Fatalf("open trace entry: %v", err)
		}
		defer rc.Close()
		data, err := io.ReadAll(rc)
		if err != nil {
			t.Fatalf("read trace entry: %v", err)
		}

		var events []map[string]interface{}
		for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
			if line == "" {
				continue
			}
			var evt map[string]interface{}
			if err := json.Unmarshal([]byte(line), &evt); err != nil {
				t.Fatalf("parse trace line %q: %v", line, err)
			}
			events = append(events, evt)
		}
		return events
	}
	t.Fatal("recording zip has no trace.trace entry")
	return nil
}

// findDropEvent returns the vibium.eventsDropped trace event, or nil.
func findDropEvent(events []map[string]interface{}) map[string]interface{} {
	for _, evt := range events {
		if evt["method"] == "vibium.eventsDropped" {
			return evt
		}
	}
	return nil
}

// TestNoteDroppedEventsStampsTrace checks that a recording that lost BiDi
// events says so in the trace, with the count.
func TestNoteDroppedEventsStampsTrace(t *testing.T) {
	rec := NewRecorder()
	rec.Start(RecordingStartOptions{Name: "drops"}, nil)
	rec.NoteDroppedEvents(7)

	zipData, err := rec.Stop()
	if err != nil {
		t.Fatalf("stop recording: %v", err)
	}

	drop := findDropEvent(traceLines(t, zipData))
	if drop == nil {
		t.Fatal("trace has no vibium.eventsDropped event, want one with count 7")
	}
	params, _ := drop["params"].(map[string]interface{})
	if count, _ := params["count"].(float64); count != 7 {
		t.Fatalf("eventsDropped count = %v, want 7", params["count"])
	}
}

// TestNoDropEventWhenNothingDropped checks that a clean recording does not
// claim drops.
func TestNoDropEventWhenNothingDropped(t *testing.T) {
	rec := NewRecorder()
	rec.Start(RecordingStartOptions{Name: "clean"}, nil)
	rec.NoteDroppedEvents(0)

	zipData, err := rec.Stop()
	if err != nil {
		t.Fatalf("stop recording: %v", err)
	}

	if drop := findDropEvent(traceLines(t, zipData)); drop != nil {
		t.Fatalf("trace claims dropped events on a clean recording: %v", drop)
	}
}

func TestVerificationGroupAndResults(t *testing.T) {
	r := NewRecorder()
	r.Start(RecordingStartOptions{}, nil)
	outer := r.StartGroup("outer")
	parent := r.StartGroup("Verify: claim")
	r.SetGroupParams(parent, map[string]interface{}{"method": "vibium:verify.run", "claim": "claim"})
	child := r.NextCallId()
	r.RecordAction(child, "vibium:page.map", map[string]interface{}{}, "", "page")
	r.RecordActionEnd(child, "", time.Now(), nil)
	r.RecordCallOutcome(child, map[string]string{"observed": "name"}, nil)
	r.StopGroup()
	r.RecordCallOutcome(parent, map[string]string{"status": "passed", "summary": "persisted"}, nil)
	r.SetGroupTitle(parent, "Verify: claim — PASS: persisted")
	r.StopGroup()
	data, err := r.Stop()
	if err != nil {
		t.Fatal(err)
	}
	events := traceLines(t, data)
	for _, e := range events {
		if e["type"] == "before" && e["callId"] == parent {
			if e["parentId"] != outer || e["method"] != "tracingGroup" {
				t.Fatal(e)
			}
		}
		if e["type"] == "before" && e["callId"] == child {
			if e["parentId"] != parent {
				t.Fatal(e)
			}
		}
		if e["type"] == "after" && (e["callId"] == child || e["callId"] == parent) {
			if e["result"] == nil {
				t.Fatal("missing result", e)
			}
		}
	}
}

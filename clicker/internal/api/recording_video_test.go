package api

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
)

type recordingTestClient struct {
	messages []string
}

func (c *recordingTestClient) ID() uint64   { return 1 }
func (c *recordingTestClient) Close() error { return nil }
func (c *recordingTestClient) Send(msg string) error {
	c.messages = append(c.messages, msg)
	return nil
}

func TestVideoSupportErrorExplainsFirefoxOutputFailure(t *testing.T) {
	err := videoSupportError(errors.New(
		`unknown error: NS_ERROR_FAILURE [nsIProperties.get]`,
	))
	if !strings.Contains(err.Error(), "could not resolve its screencast output directory") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestVideoSupportErrorNamesTheInstallCommand(t *testing.T) {
	err := videoSupportError(errors.New(`unknown command: browsingContext.startScreencast`))
	if !strings.Contains(err.Error(), "vibium install --engine firefox") {
		t.Fatalf("error should name the install command, got: %v", err)
	}
}

func TestRequiredVideoOnRemoteConnectionFailsClearly(t *testing.T) {
	client := &recordingTestClient{}
	router := NewRouter("firefox", true, "ws://remote.example/session", nil)
	router.handleRecordingStart(&BrowserSession{Client: client}, bidiCommand{
		ID:     7,
		Params: map[string]interface{}{"video": true},
	})

	if len(client.messages) != 1 || !strings.Contains(client.messages[0], RemoteVideoMessage) {
		t.Fatalf("response = %#v, want remote video error", client.messages)
	}
}

func TestParseRecordingOptionsVideoShapes(t *testing.T) {
	tests := []struct {
		name   string
		params map[string]interface{}
		want   VideoOptions
	}{
		{name: "omitted", params: map[string]interface{}{}, want: VideoOptions{Mode: VideoAuto}},
		{name: "true", params: map[string]interface{}{"video": true}, want: VideoOptions{Mode: VideoRequired}},
		{name: "false", params: map[string]interface{}{"video": false}, want: VideoOptions{Mode: VideoOff}},
		{
			name:   "dimensions",
			params: map[string]interface{}{"video": map[string]interface{}{"width": 1280.0, "height": 720.0, "frameRate": 30.0}},
			want:   VideoOptions{Mode: VideoRequired, Width: 1280, Height: 720, FrameRate: 30},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseRecordingOptions(tt.params).Video
			if got != tt.want {
				t.Fatalf("Video = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestRecordingZipEmbedsVideoTrack(t *testing.T) {
	engineFile := t.TempDir() + "/screencast.webm"
	webm := []byte{0x1a, 0x45, 0xdf, 0xa3, 1, 2, 3, 4}
	if err := writeTestFile(engineFile, webm); err != nil {
		t.Fatal(err)
	}

	rec := NewRecorder()
	rec.Start(RecordingStartOptions{Screenshots: true}, nil)
	rec.SetVideoTrack(&VideoTrack{
		Context:    "CTX-1",
		ID:         "sc-1",
		EnginePath: engineFile,
		StartedAt:  time.Now().UnixMilli(),
		Width:      1280,
		Height:     720,
	})
	rec.FinishVideo(engineFile, "")

	data, err := rec.Stop()
	if err != nil {
		t.Fatal(err)
	}

	entries := readZipEntries(t, data)
	if !bytes.Equal(entries["video/ctx-1.webm"], webm) {
		t.Fatalf("video/ctx-1.webm missing or wrong, entries: %v", entryNames(entries))
	}

	var index struct {
		Version int `json:"version"`
		Videos  []struct {
			File     string  `json:"file"`
			Context  string  `json:"context"`
			OffsetMs float64 `json:"offsetMs"`
			Width    int     `json:"width"`
			MimeType string  `json:"mimeType"`
		} `json:"videos"`
	}
	if err := json.Unmarshal(entries["video/index.json"], &index); err != nil {
		t.Fatalf("video/index.json unreadable: %v", err)
	}
	if index.Version != 1 || len(index.Videos) != 1 {
		t.Fatalf("unexpected index: %+v", index)
	}
	v := index.Videos[0]
	if v.File != "video/ctx-1.webm" || v.Context != "ctx-1" || v.Width != 1280 || v.MimeType != "video/webm" {
		t.Fatalf("unexpected video entry: %+v", v)
	}

	// Stop() moves the file into the zip: the engine temp must be gone.
	if fileExists(engineFile) {
		t.Fatal("engine temp file should be deleted after Stop")
	}
}

func TestRecordingZipRecordsVideoErrorWithoutFile(t *testing.T) {
	rec := NewRecorder()
	rec.Start(RecordingStartOptions{Screenshots: true}, nil)
	rec.SetVideoTrack(&VideoTrack{Context: "ctx-2", ID: "sc-2", StartedAt: time.Now().UnixMilli()})
	rec.FinishVideo("", "screencast write failed: disk full")

	data, err := rec.Stop()
	if err != nil {
		t.Fatal(err)
	}
	entries := readZipEntries(t, data)
	if !strings.Contains(string(entries["video/index.json"]), "disk full") {
		t.Fatalf("index should record the error, got: %s", entries["video/index.json"])
	}
	for name := range entries {
		if strings.HasSuffix(name, ".webm") {
			t.Fatalf("no video file should be present, found %s", name)
		}
	}
}

func TestChunkZipCarriesVideoRangeButNoFile(t *testing.T) {
	engineFile := t.TempDir() + "/screencast.webm"
	if err := writeTestFile(engineFile, []byte{1, 2, 3}); err != nil {
		t.Fatal(err)
	}

	rec := NewRecorder()
	rec.Start(RecordingStartOptions{Screenshots: true}, nil)
	rec.SetVideoTrack(&VideoTrack{
		Context:    "ctx-3",
		ID:         "sc-3",
		EnginePath: engineFile,
		StartedAt:  time.Now().UnixMilli(),
	})
	rec.StartChunk("part2", "", nil)

	data, err := rec.StopChunk()
	if err != nil {
		t.Fatal(err)
	}
	entries := readZipEntries(t, data)
	for name := range entries {
		if strings.HasSuffix(name, ".webm") {
			t.Fatalf("chunk artifacts carry no video file, found %s", name)
		}
	}
	if !strings.Contains(string(entries["video/index.json"]), "videoRange") {
		t.Fatalf("chunk manifest should record videoRange, got: %s", entries["video/index.json"])
	}
}

func TestSummaryReportsVideoUnavailable(t *testing.T) {
	rec := NewRecorder()
	rec.Start(RecordingStartOptions{Screenshots: true}, nil)
	rec.SetVideoUnavailable("engine says no")

	fields := rec.Summary().ResultFields()
	if fields["videoUnavailable"] != "engine says no" {
		t.Fatalf("fields = %v", fields)
	}
	if _, ok := fields["videos"]; ok {
		t.Fatal("videos must be absent when videoUnavailable is set")
	}
}

func TestSavedSentenceShapes(t *testing.T) {
	withVideo := RecordingSavedSentence("record.zip", RecordingSummary{
		Steps:  23,
		Videos: []VideoSummary{{DurationMs: 14200}},
	})
	if withVideo != "Saved record.zip (23 steps, 14s video)" {
		t.Fatalf("sentence = %q", withVideo)
	}

	unavailable := RecordingSavedSentence("record.zip", RecordingSummary{
		Steps:            3,
		VideoUnavailable: "no engine",
	})
	if unavailable != "Saved record.zip (3 steps) — video unavailable: no engine" {
		t.Fatalf("sentence = %q", unavailable)
	}
}

func TestRecordingOperationsAreSerialized(t *testing.T) {
	session := &BrowserSession{}
	firstEntered := make(chan struct{})
	releaseFirst := make(chan struct{})
	secondEntered := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		session.recordingMu.Lock()
		defer session.recordingMu.Unlock()
		close(firstEntered)
		<-releaseFirst
	}()
	<-firstEntered

	go func() {
		defer wg.Done()
		session.recordingMu.Lock()
		defer session.recordingMu.Unlock()
		close(secondEntered)
	}()

	select {
	case <-secondEntered:
		t.Fatal("concurrent recording operation entered before the active operation completed")
	case <-time.After(25 * time.Millisecond):
	}
	close(releaseFirst)
	select {
	case <-secondEntered:
	case <-time.After(time.Second):
		t.Fatal("waiting recording operation did not proceed")
	}
	wg.Wait()
}

func TestClosingSessionRejectsQueuedRecordingOperation(t *testing.T) {
	session := &BrowserSession{}
	if !session.beginRecordingOperation() {
		t.Fatal("first operation was unexpectedly rejected")
	}

	session.mu.Lock()
	session.closed = true
	session.mu.Unlock()
	session.endRecordingOperation()

	if session.beginRecordingOperation() {
		session.endRecordingOperation()
		t.Fatal("operation was accepted after session shutdown began")
	}
}

func readZipEntries(t *testing.T, data []byte) map[string][]byte {
	t.Helper()
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatalf("invalid zip: %v", err)
	}
	entries := map[string][]byte{}
	for _, f := range zr.File {
		rc, err := f.Open()
		if err != nil {
			t.Fatalf("open %s: %v", f.Name, err)
		}
		content, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			t.Fatalf("read %s: %v", f.Name, err)
		}
		entries[f.Name] = content
	}
	return entries
}

func writeTestFile(path string, data []byte) error {
	return os.WriteFile(path, data, 0o644)
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func entryNames(entries map[string][]byte) []string {
	names := make([]string, 0, len(entries))
	for name := range entries {
		names = append(names, name)
	}
	return names
}

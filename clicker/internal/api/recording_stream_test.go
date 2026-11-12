package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/vibium/clicker/internal/speech"
	"github.com/vibium/clicker/internal/watch/watchtest"
)

// sessionURL is the BiDi URL a session would have connected to on srv.
func sessionURL(srv *httptest.Server) string {
	return "ws" + strings.TrimPrefix(srv.URL, "http") + "/session"
}

type streamIndex struct {
	Videos []struct {
		Context    string  `json:"context"`
		File       string  `json:"file"`
		MimeType   string  `json:"mimeType"`
		Width      int     `json:"width"`
		Height     int     `json:"height"`
		OffsetMs   float64 `json:"offsetMs"`
		DurationMs int64   `json:"durationMs"`
		Error      string  `json:"error"`
		Audio      bool    `json:"audio"`
	} `json:"videos"`
}

func TestStreamVideoGoesIntoTheZip(t *testing.T) {
	srv, _ := watchtest.Roadie(t)
	opts := ParseRecordingOptions(map[string]interface{}{})
	stream, err := OpenStreamVideo(opts, sessionURL(srv), http.Header{"Authorization": {"Bearer local"}})
	if err != nil || stream == nil {
		t.Fatalf("stream %v, %v", stream, err)
	}
	rec := NewRecorder()
	rec.Start(opts, nil)
	sess := &fakeVideoSession{}
	if err := StartRecordingVideo(sess, rec, opts, true, nil, stream); err != nil {
		t.Fatal(err)
	}
	if sess.started {
		t.Error("the engine was asked for a screencast as well")
	}
	if rec.VideoUnavailable() != "" || rec.ActiveVideo() == nil {
		t.Fatalf("video unavailable: %q", rec.VideoUnavailable())
	}
	time.Sleep(500 * time.Millisecond)

	StopRecordingVideo(sess, rec)
	temp := rec.ActiveVideo().EnginePath
	data, err := rec.Stop()
	if err != nil {
		t.Fatal(err)
	}
	if temp == "" || fileExists(temp) {
		t.Errorf("the stream's temporary file %q should be gone after stop", temp)
	}

	entries := readZipEntries(t, data)
	var index streamIndex
	if err := json.Unmarshal(entries["video/index.json"], &index); err != nil || len(index.Videos) != 1 {
		t.Fatalf("index %s: %v", entries["video/index.json"], err)
	}
	v := index.Videos[0]
	if v.Context != "ctx-remote" || v.File != "video/ctx-remote.mp4" || v.MimeType != "video/mp4" {
		t.Errorf("entry %+v", v)
	}
	if v.Width != watchtest.Width || v.Height != watchtest.Height || v.Error != "" {
		t.Errorf("entry %+v", v)
	}
	// The stream was joined just before the recording started.
	if v.OffsetMs > 0 || v.OffsetMs < -2000 {
		t.Errorf("offset %v ms, want just before the start", v.OffsetMs)
	}
	if v.DurationMs < 400 || v.DurationMs > 3000 {
		t.Errorf("duration %d ms for half a second", v.DurationMs)
	}
	if file := entries[v.File]; len(file) < 1000 || !bytes.Equal(file[4:8], []byte("ftyp")) {
		t.Errorf("%s is %d bytes and not an MP4", v.File, len(file))
	}
	// Nothing was said or heard, so the file has no sound.
	if v.Audio || bytes.Contains(entries[v.File], []byte("mp4a")) {
		t.Errorf("a silent recording's video has an audio track: %+v", v)
	}

	summary := rec.Summary()
	if len(summary.Videos) != 1 || summary.Videos[0].DurationMs != v.DurationMs || summary.Videos[0].Width != watchtest.Width {
		t.Errorf("summary %+v", summary.Videos)
	}
}

func TestStreamVideoIsLeftAloneWhenNotWanted(t *testing.T) {
	srv, rig := watchtest.Roadie(t)
	for name, args := range map[string]map[string]interface{}{
		"video off":   {"video": false},
		"remote keep": {"video": map[string]interface{}{"remote": "keep"}},
	} {
		if stream, err := OpenStreamVideo(ParseRecordingOptions(args), sessionURL(srv), nil); stream != nil || err != nil {
			t.Errorf("%s: %v, %v", name, stream, err)
		}
	}
	if stream, err := OpenStreamVideo(ParseRecordingOptions(map[string]interface{}{"video": true}), "", nil); stream != nil || err != nil {
		t.Errorf("local browser: %v, %v", stream, err)
	}
	if _, auth := rig.Viewers(); len(auth) != 0 {
		t.Errorf("%d viewers joined", len(auth))
	}
}

func TestRemoteBrowserWithoutAStreamKeepsItsMessage(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	defer srv.Close()

	required := ParseRecordingOptions(map[string]interface{}{"video": true})
	if _, err := OpenStreamVideo(required, sessionURL(srv), nil); err == nil || err.Error() != RemoteVideoMessage {
		t.Errorf("required: %v", err)
	}

	auto := ParseRecordingOptions(map[string]interface{}{})
	stream, err := OpenStreamVideo(auto, sessionURL(srv), nil)
	if stream != nil || err != nil {
		t.Fatalf("auto: %v, %v", stream, err)
	}
	rec := NewRecorder()
	rec.Start(auto, nil)
	if err := StartRecordingVideo(&fakeVideoSession{}, rec, auto, true, nil, stream); err != nil {
		t.Fatal(err)
	}
	if rec.VideoUnavailable() != RemoteVideoMessage {
		t.Errorf("unavailable %q", rec.VideoUnavailable())
	}
}

func TestStreamThatCannotBeJoinedSaysWhy(t *testing.T) {
	srv, rig := watchtest.Roadie(t)
	rig.Refuse = "the encoder is busy"

	required := ParseRecordingOptions(map[string]interface{}{"video": true})
	if _, err := OpenStreamVideo(required, sessionURL(srv), nil); err == nil || !strings.Contains(err.Error(), "the encoder is busy") {
		t.Errorf("required: %v", err)
	}

	auto := ParseRecordingOptions(map[string]interface{}{})
	stream, err := OpenStreamVideo(auto, sessionURL(srv), nil)
	if stream == nil || err != nil {
		t.Fatalf("auto: %v, %v", stream, err)
	}
	rec := NewRecorder()
	rec.Start(auto, nil)
	if err := StartRecordingVideo(&fakeVideoSession{}, rec, auto, true, nil, stream); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(rec.VideoUnavailable(), "the encoder is busy") || rec.ActiveVideo() != nil {
		t.Errorf("unavailable %q", rec.VideoUnavailable())
	}
}

func TestStreamVideoSurvivesALostSession(t *testing.T) {
	srv, _ := watchtest.Roadie(t)
	out := t.TempDir() + "/lost.zip"
	opts := RecordingStartOptions{Path: out}
	stream, err := OpenStreamVideo(opts, sessionURL(srv), nil)
	if err != nil || stream == nil {
		t.Fatalf("stream %v, %v", stream, err)
	}
	rec := NewRecorder()
	rec.Start(opts, nil)
	if err := StartRecordingVideo(&fakeVideoSession{}, rec, opts, true, nil, stream); err != nil {
		t.Fatal(err)
	}
	time.Sleep(300 * time.Millisecond)

	FinalizeRecordingOffline(rec)
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	entries := readZipEntries(t, data)
	var index streamIndex
	json.Unmarshal(entries["video/index.json"], &index)
	if len(index.Videos) != 1 || index.Videos[0].Error != "" || len(entries["video/ctx-remote.mp4"]) == 0 {
		t.Errorf("index %s, entries %v", entries["video/index.json"], entryNames(entries))
	}
}

func TestAbandonedStreamVideoLeavesNoFile(t *testing.T) {
	srv, _ := watchtest.Roadie(t)
	dir := t.TempDir()
	t.Setenv("TMPDIR", dir)
	opts := ParseRecordingOptions(map[string]interface{}{})
	stream, err := OpenStreamVideo(opts, sessionURL(srv), nil)
	if err != nil || stream == nil {
		t.Fatalf("stream %v, %v", stream, err)
	}
	rec := NewRecorder()
	rec.Start(opts, nil)
	if err := StartRecordingVideo(&fakeVideoSession{}, rec, opts, true, nil, stream); err != nil {
		t.Fatal(err)
	}
	if left, _ := os.ReadDir(dir); len(left) != 1 {
		t.Fatalf("%d files while recording, want the one being written", len(left))
	}
	rec.RemoveEngineFile()
	if left, _ := os.ReadDir(dir); len(left) != 0 {
		t.Errorf("%d files left behind", len(left))
	}
}

func steady(level int16, ms int) []int16 {
	out := make([]int16, ms*speech.Rate/1000)
	for i := range out {
		out[i] = level
	}
	return out
}

func TestStreamVideoCarriesWhatWasSaidAndHeard(t *testing.T) {
	srv, _ := watchtest.Roadie(t)
	opts := ParseRecordingOptions(map[string]interface{}{})
	stream, err := OpenStreamVideo(opts, sessionURL(srv), nil)
	if err != nil || stream == nil {
		t.Fatalf("stream %v, %v", stream, err)
	}
	rec := NewRecorder()
	rec.Start(opts, nil)
	sess := &fakeVideoSession{}
	if err := StartRecordingVideo(sess, rec, opts, true, nil, stream); err != nil {
		t.Fatal(err)
	}
	now := rec.TimelineMs(time.Now())
	clip := rec.AddAudioClip(speech.Audio{SampleRate: speech.Rate, Channels: 1, Samples: steady(9000, 200)}, map[string]interface{}{"text": "hello"})
	rec.NoteAudioPlay(clip, "call@1", now, now+200)
	rec.AppendAudio("session", steady(-4000, 300), now+250)
	time.Sleep(700 * time.Millisecond)

	StopRecordingVideo(sess, rec)
	data, err := rec.Stop()
	if err != nil {
		t.Fatal(err)
	}
	entries := readZipEntries(t, data)
	var index streamIndex
	if err := json.Unmarshal(entries["video/index.json"], &index); err != nil || len(index.Videos) != 1 {
		t.Fatalf("index %s: %v", entries["video/index.json"], err)
	}
	v := index.Videos[0]
	if !v.Audio || v.Error != "" {
		t.Errorf("entry %+v", v)
	}
	file := entries[v.File]
	if !bytes.Contains(file, []byte("soun")) || !bytes.Contains(file, []byte("mp4a")) {
		t.Errorf("%s has no audio track", v.File)
	}
	// The audio stays in the zip as it was, for the Record Player.
	if len(entries["audio/out-0001.wav"]) == 0 || len(entries["audio/in-session.wav"]) == 0 {
		t.Errorf("entries %v", entryNames(entries))
	}

	summary := rec.Summary()
	if len(summary.Videos) != 1 || !summary.Videos[0].Audio {
		t.Fatalf("summary %+v", summary.Videos)
	}
	videos := summary.ResultFields()["videos"].([]interface{})
	if videos[0].(map[string]interface{})["audio"] != true {
		t.Errorf("result %v", videos[0])
	}
	if sentence := RecordingSavedSentence("r.zip", summary); !strings.Contains(sentence, "video with sound") {
		t.Errorf("sentence %q", sentence)
	}
}

func TestAudioMixPutsEverySoundAtItsTime(t *testing.T) {
	rec := NewRecorder()
	rec.Start(ParseRecordingOptions(map[string]interface{}{}), nil)
	clip := rec.AddAudioClip(speech.Audio{SampleRate: speech.Rate, Channels: 1, Samples: steady(1000, 10)}, nil)
	rec.NoteAudioPlay(clip, "call@1", 100, 110)
	rec.NoteAudioPlay(clip, "call@2", 400, 410)
	rec.AppendAudio("session", steady(2000, 20), 105)
	loud := rec.AddAudioClip(speech.Audio{SampleRate: speech.Rate, Channels: 1, Samples: steady(30000, 10)}, nil)
	rec.NoteAudioPlay(loud, "call@3", 200, 210)
	rec.AppendAudio("speakers", steady(30000, 10), 200)

	mix := func(fromMs float64, length time.Duration) []int16 {
		rec.mu.Lock()
		defer rec.mu.Unlock()
		return rec.mixAudioLocked(fromMs, length)
	}
	// From 50 ms on, a sample is 1/16 ms.
	got := mix(50, 200*time.Millisecond)
	if len(got) != 200*speech.Rate/1000 {
		t.Fatalf("%d samples for 200 ms", len(got))
	}
	for at, want := range map[int]int16{
		799:  0,     // just before the clip
		800:  1000,  // the clip alone, from 100 ms
		880:  3000,  // the clip and what was heard, from 105 ms
		960:  2000,  // what was heard alone
		1200: 0,     // after both
		2400: 32767, // two loud sounds at once stay in range
	} {
		if got[at] != want {
			t.Errorf("sample %d is %d, want %d", at, got[at], want)
		}
	}
	// The second playing is past this stretch; a later stretch has it at its start.
	if late := mix(400, 5*time.Millisecond); len(late) != 80 || late[0] != 1000 {
		t.Errorf("the second playing: %v", late)
	}
	if none := mix(1000, time.Second); none != nil {
		t.Errorf("%d samples where nothing sounded", len(none))
	}
	if none := NewRecorder().mixAudioLocked(0, time.Second); none != nil {
		t.Error("a recording without audio gave a mix")
	}
}

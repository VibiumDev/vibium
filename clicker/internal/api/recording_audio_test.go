package api

import (
	"archive/zip"
	"bytes"
	"encoding/binary"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// fakeRoadie serves /audio and /audio-in/monitor the way a roadie host does:
// a params text message, then binary PCM chunks.
func fakeRoadie(t *testing.T, outputChunks, inputChunks [][]byte) *httptest.Server {
	t.Helper()
	upgrader := websocket.Upgrader{}
	serve := func(params string, chunks [][]byte) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			conn, err := upgrader.Upgrade(w, r, nil)
			if err != nil {
				return
			}
			defer conn.Close()
			if err := conn.WriteMessage(websocket.TextMessage, []byte(params)); err != nil {
				return
			}
			for _, chunk := range chunks {
				if err := conn.WriteMessage(websocket.BinaryMessage, chunk); err != nil {
					return
				}
			}
			// Hold the stream open until the recorder closes it.
			for {
				if _, _, err := conn.ReadMessage(); err != nil {
					return
				}
			}
		}
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/audio", serve(`{"sampleRate":48000,"channels":2,"format":"f32-planar"}`, outputChunks))
	mux.HandleFunc("/audio-in/monitor", serve(`{"sampleRate":16000,"channels":1,"format":"s16le"}`, inputChunks))
	return httptest.NewServer(mux)
}

func startAudioRecording(t *testing.T, source string) *Recorder {
	t.Helper()
	recorder := NewRecorder()
	opts := RecordingStartOptions{Screenshots: true}
	opts.Audio.Source = source
	recorder.Start(opts, nil)
	if err := StartRecordingAudio(recorder, opts); err != nil {
		recorder.RemoveEngineFile()
		t.Fatalf("StartRecordingAudio: %v", err)
	}
	return recorder
}

func TestAudioRecordingEmbedsWavTracks(t *testing.T) {
	output := [][]byte{make([]byte, 4*48), make([]byte, 4*48)} // f32 stereo chunks
	// 100 ms of s16 mono at 16 kHz, patterned so the zip round-trip is checkable.
	inputChunk := make([]byte, 3200)
	for i := range inputChunk {
		inputChunk[i] = byte(i)
	}
	input := [][]byte{inputChunk}
	ts := fakeRoadie(t, output, input)
	defer ts.Close()

	recorder := startAudioRecording(t, ts.URL)

	// Capture is asynchronous; wait for both spools to fill.
	waitForAudioBytes(t, recorder, int64(len(output[0])*2), int64(len(input[0])))

	StopRecordingAudio(recorder)
	zipData, err := recorder.Stop()
	if err != nil {
		t.Fatal(err)
	}

	zr, err := zip.NewReader(bytes.NewReader(zipData), int64(len(zipData)))
	if err != nil {
		t.Fatal(err)
	}
	files := map[string][]byte{}
	for _, f := range zr.File {
		r, err := f.Open()
		if err != nil {
			t.Fatal(err)
		}
		data, _ := io.ReadAll(r)
		r.Close()
		files[f.Name] = data
	}

	index, ok := files["audio/index.json"]
	if !ok {
		t.Fatalf("audio/index.json missing; zip holds %v", zipNames(zr))
	}
	var manifest struct {
		Version int `json:"version"`
		Audio   []struct {
			File       string  `json:"file"`
			Direction  string  `json:"direction"`
			OffsetMs   float64 `json:"offsetMs"`
			SampleRate int     `json:"sampleRate"`
			Channels   int     `json:"channels"`
			DurationMs int64   `json:"durationMs"`
			Error      string  `json:"error"`
		} `json:"audio"`
	}
	if err := json.Unmarshal(index, &manifest); err != nil {
		t.Fatal(err)
	}
	if len(manifest.Audio) != 2 {
		t.Fatalf("expected 2 audio entries, got %d", len(manifest.Audio))
	}

	byDirection := map[string]int{}
	for i, entry := range manifest.Audio {
		byDirection[entry.Direction] = i
		if entry.Error != "" {
			t.Errorf("audio %s error: %s", entry.Direction, entry.Error)
		}
	}

	out := manifest.Audio[byDirection["output"]]
	if out.File != "audio/output.wav" || out.SampleRate != 48000 || out.Channels != 2 {
		t.Errorf("unexpected output entry: %+v", out)
	}
	in := manifest.Audio[byDirection["input"]]
	if in.File != "audio/input.wav" || in.SampleRate != 16000 || in.Channels != 1 {
		t.Errorf("unexpected input entry: %+v", in)
	}

	wav := files["audio/input.wav"]
	if len(wav) != 44+len(inputChunk) {
		t.Fatalf("input.wav is %d bytes, want 44-byte header + %d data bytes", len(wav), len(inputChunk))
	}
	if string(wav[0:4]) != "RIFF" || string(wav[8:12]) != "WAVE" {
		t.Errorf("input.wav has no RIFF/WAVE magic")
	}
	if format := binary.LittleEndian.Uint16(wav[20:22]); format != 1 {
		t.Errorf("input.wav format = %d, want 1 (integer PCM)", format)
	}
	if !bytes.Equal(wav[44:], input[0]) {
		t.Errorf("input.wav data does not match the streamed PCM")
	}

	outWav := files["audio/output.wav"]
	if format := binary.LittleEndian.Uint16(outWav[20:22]); format != 3 {
		t.Errorf("output.wav format = %d, want 3 (IEEE float)", format)
	}
	if dataLen := binary.LittleEndian.Uint32(outWav[40:44]); int(dataLen) != len(outWav)-44 {
		t.Errorf("output.wav data length %d does not match header %d", len(outWav)-44, dataLen)
	}

	// Spools are deleted once the zip is built.
	summary := recorder.Summary()
	if len(summary.Audio) != 2 {
		t.Fatalf("expected 2 audio summaries, got %d", len(summary.Audio))
	}
	for _, a := range summary.Audio {
		if a.DurationMs <= 0 {
			t.Errorf("audio %s durationMs = %d, want > 0", a.Direction, a.DurationMs)
		}
	}
}

func TestAudioRecordingStartFailsOnDeadSource(t *testing.T) {
	recorder := NewRecorder()
	opts := RecordingStartOptions{}
	opts.Audio.Source = "http://127.0.0.1:1" // nothing listens here
	recorder.Start(opts, nil)
	if err := StartRecordingAudio(recorder, opts); err == nil {
		t.Fatal("expected an error for an unreachable source")
	} else if !strings.Contains(err.Error(), "unreachable") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestAudioRecordingStartFailsOn503(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/audio", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "no audio available", http.StatusServiceUnavailable)
	})
	ts := httptest.NewServer(mux)
	defer ts.Close()

	recorder := NewRecorder()
	opts := RecordingStartOptions{}
	opts.Audio.Source = ts.URL
	recorder.Start(opts, nil)
	err := StartRecordingAudio(recorder, opts)
	if err == nil {
		t.Fatal("expected an error for a 503 source")
	}
	if !strings.Contains(err.Error(), "503") {
		t.Errorf("error does not name the 503: %v", err)
	}
}

func TestAudioRecordingLostStreamStillDelivers(t *testing.T) {
	upgrader := websocket.Upgrader{}
	drop := func(params string, chunk []byte) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			conn, err := upgrader.Upgrade(w, r, nil)
			if err != nil {
				return
			}
			conn.WriteMessage(websocket.TextMessage, []byte(params))
			conn.WriteMessage(websocket.BinaryMessage, chunk)
			conn.Close() // abrupt: no close handshake
		}
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/audio", drop(`{"sampleRate":48000,"channels":2,"format":"f32-planar"}`, make([]byte, 96)))
	mux.HandleFunc("/audio-in/monitor", drop(`{"sampleRate":16000,"channels":1,"format":"s16le"}`, make([]byte, 32)))
	ts := httptest.NewServer(mux)
	defer ts.Close()

	recorder := startAudioRecording(t, ts.URL)
	waitForAudioBytes(t, recorder, 96, 32)

	StopRecordingAudio(recorder)
	zipData, err := recorder.Stop()
	if err != nil {
		t.Fatalf("stop must deliver despite the lost stream: %v", err)
	}

	zr, _ := zip.NewReader(bytes.NewReader(zipData), int64(len(zipData)))
	var index []byte
	for _, f := range zr.File {
		if f.Name == "audio/index.json" {
			r, _ := f.Open()
			index, _ = io.ReadAll(r)
			r.Close()
		}
	}
	if !strings.Contains(string(index), "audio stream lost") {
		t.Errorf("index does not record the lost stream: %s", index)
	}
	// The received frames still embed.
	if !strings.Contains(string(index), "audio/output.wav") {
		t.Errorf("partial output track missing from index: %s", index)
	}
}

// waitForAudioBytes polls until both capture spools report the expected byte
// counts, bounded by the test deadline semantics of the callers.
func waitForAudioBytes(t *testing.T, recorder *Recorder, outputBytes, inputBytes int64) {
	t.Helper()
	ac := recorder.AudioCapture()
	if ac == nil {
		t.Fatal("no audio capture attached")
	}
	for i := 0; i < 500; i++ {
		ac.mu.Lock()
		var out, in int64
		for _, track := range ac.tracks {
			if track.Direction == "output" {
				out = track.Bytes
			} else {
				in = track.Bytes
			}
		}
		ac.mu.Unlock()
		if out >= outputBytes && in >= inputBytes {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("audio spools did not fill in time")
}

func zipNames(zr *zip.Reader) []string {
	names := make([]string, 0, len(zr.File))
	for _, f := range zr.File {
		names = append(names, f.Name)
	}
	return names
}

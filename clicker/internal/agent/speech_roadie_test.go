package agent

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"github.com/vibium/clicker/internal/api"
	"github.com/vibium/clicker/internal/bidi"
	"github.com/vibium/clicker/internal/speech"
)

func toneSamples(hz float64, d time.Duration) []int16 {
	out := make([]int16, int(d.Seconds()*speech.Rate))
	for i := range out {
		out[i] = int16(6000 * math.Sin(2*math.Pi*hz*float64(i)/speech.Rate))
	}
	return out
}

// fakeRoadie speaks roadie:audio. Its clock starts at 5000 ms. A clip
// "plays" at once: play answers with the times it would have taken. The
// device stays quiet while it is spoken to, waits thinkMs, then plays a
// tone for a second.
type fakeRoadie struct {
	mu      sync.Mutex
	audio   bool
	thinkMs float64
	dropAt  int // chunk index dropped from the capture, 0 for none
	log     []string
	played  []byte
	ws      *websocket.Conn
	wmu     sync.Mutex
}

func (f *fakeRoadie) send(v interface{}) {
	f.wmu.Lock()
	defer f.wmu.Unlock()
	f.ws.WriteJSON(v)
}

func (f *fakeRoadie) serve(t *testing.T) string {
	up := websocket.Upgrader{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ws, err := up.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer ws.Close()
		f.ws = ws
		for {
			var cmd struct {
				ID     int    `json:"id"`
				Method string `json:"method"`
				Params struct {
					Data       string `json:"data"`
					SampleRate int    `json:"sampleRate"`
				} `json:"params"`
			}
			if ws.ReadJSON(&cmd) != nil {
				return
			}
			f.mu.Lock()
			f.log = append(f.log, cmd.Method)
			f.mu.Unlock()
			ok := func(result interface{}) {
				f.send(map[string]interface{}{"type": "success", "id": cmd.ID, "result": result})
			}
			if !f.audio || !strings.HasPrefix(cmd.Method, "roadie:audio.") {
				f.send(map[string]interface{}{"type": "error", "id": cmd.ID, "error": "unknown command", "message": "unsupported method: " + cmd.Method})
				continue
			}
			switch cmd.Method {
			case "roadie:audio.clock":
				ok(map[string]float64{"now": 5000})
			case "roadie:audio.capture.start":
				ok(map[string]interface{}{"capture": "c1", "sampleRate": 16000, "startTime": 5000})
			case "roadie:audio.capture.stop":
				ok(map[string]interface{}{})
			case "roadie:audio.play":
				pcm, _ := base64.StdEncoding.DecodeString(cmd.Params.Data)
				f.mu.Lock()
				f.played = pcm
				f.mu.Unlock()
				ms := float64(len(pcm)/2) * 1000 / 16000
				start, end := 5040.0, 5040.0+ms
				ok(map[string]float64{"startTime": start, "endTime": end})
				// What the device plays from the start of the capture on.
				heard := make([]int16, int((end-5000+f.thinkMs)*16))
				heard = append(heard, toneSamples(400, time.Second)...)
				heard = append(heard, make([]int16, 3*16000)...)
				for i := 0; i*1600 < len(heard); i++ {
					if f.dropAt != 0 && i == f.dropAt {
						continue
					}
					piece := speech.Audio{SampleRate: 16000, Channels: 1, Samples: heard[i*1600 : min((i+1)*1600, len(heard))]}
					f.send(map[string]interface{}{"type": "event", "method": "roadie:audio.chunk", "params": map[string]interface{}{
						"capture": "c1", "sample": i * 1600, "time": 5000 + float64(i)*100, "data": base64.StdEncoding.EncodeToString(piece.Bytes()),
					}})
				}
			}
		}
	}))
	t.Cleanup(srv.Close)
	return "ws" + strings.TrimPrefix(srv.URL, "http")
}

func roadieHandlers(t *testing.T, f *fakeRoadie) *Handlers {
	t.Helper()
	conn, err := bidi.Connect(f.serve(t))
	if err != nil {
		t.Fatal(err)
	}
	h := &Handlers{}
	h.client = bidi.NewClient(conn)
	h.client.SetEventHandler(h.handleBidiEvent)
	t.Cleanup(func() { h.client.Close() })
	return h
}

func speechCall(t *testing.T, h *Handlers, tool string, args map[string]interface{}, out interface{}) error {
	t.Helper()
	res, err := h.SpeechCLI(tool, args, map[string]string{})
	if err != nil {
		return err
	}
	if err := json.Unmarshal([]byte(res.Content[0].Text), out); err != nil {
		t.Fatal(err)
	}
	return nil
}

func TestSayAndListenThroughRoadie(t *testing.T) {
	f := &fakeRoadie{audio: true, thinkMs: 640}
	h := roadieHandlers(t, f)
	dir := t.TempDir()
	// A question recorded at 24 kHz: Roadie takes 16 kHz mono.
	question := speech.Audio{SampleRate: 24000, Channels: 1, Samples: make([]int16, 12000)}
	for i := range question.Samples {
		question.Samples[i] = int16(5000 * math.Sin(2*math.Pi*300*float64(i)/24000))
	}
	os.WriteFile(filepath.Join(dir, "q.wav"), question.WAV(), 0o600)

	var said speech.SayResult
	if err := speechCall(t, h, "vibium_say", map[string]interface{}{"file": filepath.Join(dir, "q.wav")}, &said); err != nil {
		t.Fatal(err)
	}
	if said.Target != "session" || said.DurationMs != 500 {
		t.Fatalf("with a session open the default target is the session: %+v", said)
	}
	f.mu.Lock()
	played, log := len(f.played), strings.Join(f.log, " ")
	f.mu.Unlock()
	if played != 16000 {
		t.Fatalf("Roadie got %d bytes; a 500 ms clip is 16000 bytes at 16 kHz mono", played)
	}
	if !strings.Contains(log, "roadie:audio.capture.start roadie:audio.play") {
		t.Fatalf("capture must be running before the clip plays, so an instant reply is heard: %s", log)
	}

	var heard speech.ListenResult
	saved := filepath.Join(dir, "reply.wav")
	if err := speechCall(t, h, "vibium_listen", map[string]interface{}{"transcribe": false, "path": saved}, &heard); err != nil {
		t.Fatal(err)
	}
	if heard.Target != "session" || !heard.Speech || heard.StoppedBy != "silence" {
		t.Fatalf("%+v", heard)
	}
	if heard.LatencyMs == nil || math.Abs(*heard.LatencyMs-640) > 20 {
		t.Fatalf("latency %v, want the 640 ms between the end of the clip and the reply, on Roadie's clock", heard.LatencyMs)
	}
	if info, err := os.Stat(saved); err != nil || info.Size() < 50000 {
		t.Fatalf("the reply was not saved: %v", err)
	}
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		f.mu.Lock()
		stopped := strings.Contains(strings.Join(f.log, " "), "roadie:audio.capture.stop")
		f.mu.Unlock()
		if stopped {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("the capture was left running on the device after listen returned")
}

func TestRoadieDroppedAudioKeepsItsPlaceInTime(t *testing.T) {
	// Roadie drops the second chunk. The reply must still be timed from
	// the chunk times, not from a count of samples.
	f := &fakeRoadie{audio: true, thinkMs: 640, dropAt: 2}
	h := roadieHandlers(t, f)
	clip := filepath.Join(t.TempDir(), "q.wav")
	os.WriteFile(clip, speech.Audio{SampleRate: 16000, Channels: 1, Samples: toneSamples(300, 500*time.Millisecond)}.WAV(), 0o600)
	var said speech.SayResult
	if err := speechCall(t, h, "vibium_say", map[string]interface{}{"file": clip}, &said); err != nil {
		t.Fatal(err)
	}
	var heard speech.ListenResult
	if err := speechCall(t, h, "vibium_listen", map[string]interface{}{"transcribe": false}, &heard); err != nil {
		t.Fatal(err)
	}
	if heard.LatencyMs == nil || math.Abs(*heard.LatencyMs-640) > 20 {
		t.Fatalf("latency %v after a dropped chunk, want 640 ms", heard.LatencyMs)
	}
}

func TestASessionWithoutAudioSaysSo(t *testing.T) {
	f := &fakeRoadie{audio: false}
	h := roadieHandlers(t, f)
	clip := filepath.Join(t.TempDir(), "q.wav")
	os.WriteFile(clip, speech.Audio{SampleRate: 16000, Channels: 1, Samples: toneSamples(300, 200*time.Millisecond)}.WAV(), 0o600)
	var said speech.SayResult
	err := speechCall(t, h, "vibium_say", map[string]interface{}{"file": clip}, &said)
	if speech.Code(err) != "no_audio" || !strings.Contains(err.Error(), "--target local") {
		t.Fatalf("got %v; the default must not fall back to the speakers", err)
	}
	// Asking for the speakers or a file still works with the session open.
	out := filepath.Join(t.TempDir(), "copy.wav")
	if err := speechCall(t, h, "vibium_say", map[string]interface{}{"file": clip, "target": out}, &said); err != nil || said.Target != out {
		t.Fatalf("%+v %v", said, err)
	}
}

func TestRoadieCaptureFillsGapsWithSilence(t *testing.T) {
	c := &roadieCapture{h: &Handlers{}, chunks: make(chan speech.Chunk, 8), done: make(chan struct{})}
	c.deliver(speech.Chunk{Samples: make([]int16, 1600), Time: 1000})
	c.deliver(speech.Chunk{Samples: make([]int16, 1600), Time: 1300}) // 200 ms missing
	ctx := context.Background()
	first, _ := c.Next(ctx)
	gap, _ := c.Next(ctx)
	second, _ := c.Next(ctx)
	if first.Time != 1000 || gap.Time != 1100 || len(gap.Samples) != 3200 || second.Time != 1300 {
		t.Fatalf("times %.0f %.0f %.0f, gap of %d samples", first.Time, gap.Time, second.Time, len(gap.Samples))
	}
}

func TestSayAndListenAreRecordedWithTheirAudio(t *testing.T) {
	f := &fakeRoadie{audio: true, thinkMs: 640}
	h := roadieHandlers(t, f)
	h.recorder = api.NewRecorder()
	h.recorder.Start(api.RecordingStartOptions{}, nil)
	dir := t.TempDir()
	clip := filepath.Join(dir, "q.wav")
	os.WriteFile(clip, speech.Audio{SampleRate: 16000, Channels: 1, Samples: toneSamples(300, 500*time.Millisecond)}.WAV(), 0o600)

	if _, err := h.Call("vibium_say", map[string]interface{}{"file": clip}); err != nil {
		t.Fatal(err)
	}
	if _, err := h.Call("vibium_listen", map[string]interface{}{"transcribe": false}); err != nil {
		t.Fatal(err)
	}
	// The same clip again is stored once and played twice.
	if _, err := h.Call("vibium_say", map[string]interface{}{"file": clip}); err != nil {
		t.Fatal(err)
	}
	h.stopAudioRecording()
	raw, err := h.recorder.Stop()
	if err != nil {
		t.Fatal(err)
	}
	zr, err := zip.NewReader(bytes.NewReader(raw), int64(len(raw)))
	if err != nil {
		t.Fatal(err)
	}
	files := map[string][]byte{}
	for _, zf := range zr.File {
		rc, _ := zf.Open()
		files[zf.Name], _ = io.ReadAll(rc)
		rc.Close()
	}
	for _, name := range []string{"trace.trace", "audio/index.json", "audio/in-session.wav", "audio/out-0001.wav"} {
		if len(files[name]) == 0 {
			t.Fatalf("the zip has no %s; it has %d entries", name, len(files))
		}
	}
	if _, twice := files["audio/out-0002.wav"]; twice {
		t.Fatal("the same clip was stored twice")
	}

	// The steps are ordinary before/after pairs, and the after carries the result.
	var sayCall, listenCall string
	results := map[string]map[string]interface{}{}
	for _, line := range strings.Split(strings.TrimSpace(string(files["trace.trace"])), "\n") {
		var ev map[string]interface{}
		json.Unmarshal([]byte(line), &ev)
		id, _ := ev["callId"].(string)
		switch {
		case ev["type"] == "before" && ev["method"] == "vibium:speech.say" && sayCall == "":
			sayCall = id
			if ev["title"] != "Speech.say" || ev["class"] != "Speech" {
				t.Errorf("the say step is titled %v in class %v", ev["title"], ev["class"])
			}
		case ev["type"] == "before" && ev["method"] == "vibium:speech.listen":
			listenCall = id
		case ev["type"] == "after":
			if r, ok := ev["result"].(map[string]interface{}); ok {
				results[id] = r
			}
		}
	}
	if sayCall == "" || listenCall == "" {
		t.Fatalf("say and listen are not steps in the trace:\n%s", files["trace.trace"])
	}
	if results[sayCall]["clip"] != "out-0001" || results[listenCall]["track"] != "in-session" || results[listenCall]["latencyFrom"] != sayCall {
		t.Fatalf("step results: say %v, listen %v", results[sayCall], results[listenCall])
	}

	var manifest struct {
		Version int
		Tracks  []struct {
			ID, File            string
			StartMs, DurationMs float64
			Speech              []struct{ StartMs, EndMs float64 }
			Clock               struct {
				Source string
				Sync   []map[string]float64
			}
		}
		Clips []struct {
			ID, File, Source string
			DurationMs       float64
			Plays            []struct {
				CallID         string
				StartMs, EndMs float64
			}
		}
		Transcripts []struct {
			CallID, Track, LatencyFrom string
			OnsetMs, LatencyMs         float64
		}
	}
	if err := json.Unmarshal(files["audio/index.json"], &manifest); err != nil {
		t.Fatal(err)
	}
	if manifest.Version != 1 || len(manifest.Tracks) != 1 || len(manifest.Clips) != 1 || len(manifest.Transcripts) != 1 {
		t.Fatalf("manifest: %s", files["audio/index.json"])
	}
	track, sent, said := manifest.Tracks[0], manifest.Clips[0], manifest.Transcripts[0]
	if track.ID != "in-session" || track.Clock.Source != "roadie" || len(track.Clock.Sync) != 1 || track.DurationMs < 4000 {
		t.Fatalf("track %+v", track)
	}
	if sent.Source != "file" || sent.DurationMs != 500 || len(sent.Plays) != 2 || sent.Plays[0].CallID != sayCall {
		t.Fatalf("clip %+v", sent)
	}
	if said.CallID != listenCall || said.LatencyFrom != sayCall || math.Abs(said.LatencyMs-640) > 20 {
		t.Fatalf("transcript %+v", said)
	}
	// Everything is on one timeline: the reply's onset is the end of the
	// clip's playing plus the latency, and the track marks speech there.
	play := sent.Plays[0]
	if math.Abs(said.OnsetMs-(play.EndMs+said.LatencyMs)) > 1 {
		t.Fatalf("onset %.1f is not the play's end %.1f plus the latency %.1f", said.OnsetMs, play.EndMs, said.LatencyMs)
	}
	if len(track.Speech) == 0 || math.Abs(track.Speech[0].StartMs-said.OnsetMs) > 30 {
		t.Fatalf("the track's speech markers %v do not show the reply at %.1f", track.Speech, said.OnsetMs)
	}
	heard, err := speech.ParseWAV(files["audio/in-session.wav"])
	if err != nil || heard.SampleRate != 16000 || heard.Channels != 1 {
		t.Fatalf("track audio: %v %+v", err, heard.SampleRate)
	}
}

package speech

import (
	"context"
	"encoding/binary"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// fakeSherpa stands in for a sherpa-onnx streaming WebSocket server: it
// takes samples until Done, then sends the answers and its own Done!.
// What it was sent is kept for the test.
type fakeSherpa struct {
	*httptest.Server
	mu      sync.Mutex
	samples []float32
	origin  string
	texts   []string
}

func newFakeSherpa(t *testing.T, answers ...string) *fakeSherpa {
	t.Helper()
	f := &fakeSherpa{}
	up := websocket.Upgrader{}
	f.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		f.origin = r.Header.Get("Origin")
		f.mu.Unlock()
		conn, err := up.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		for {
			kind, data, err := conn.ReadMessage()
			if err != nil {
				return
			}
			f.mu.Lock()
			if kind == websocket.TextMessage {
				f.texts = append(f.texts, string(data))
			} else {
				for i := 0; i+4 <= len(data); i += 4 {
					f.samples = append(f.samples, math.Float32frombits(binary.LittleEndian.Uint32(data[i:])))
				}
			}
			f.mu.Unlock()
			if kind == websocket.TextMessage && string(data) == "Done" {
				break
			}
		}
		for _, a := range answers {
			conn.WriteMessage(websocket.TextMessage, []byte(a))
		}
	}))
	t.Cleanup(f.Close)
	return f
}

func clip(seconds float64) Audio {
	a := Audio{SampleRate: Rate, Channels: 1, Samples: make([]int16, int(seconds*Rate))}
	for i := range a.Samples {
		a.Samples[i] = int16(i%2000 - 1000)
	}
	return a
}

func TestSherpaServerTranscribes(t *testing.T) {
	f := newFakeSherpa(t,
		`{"text": " COME", "segment": 0, "start_time": 0.0, "is_final": false}`,
		`{"text": " COME HERE WATSON", "segment": 0, "start_time": 0.0, "is_final": true, "tokens": [" COME", " HE", "RE", " WA", "T", "SON"], "timestamps": [0.32, 0.6, 0.68, 0.96, 1.08, 1.16]}`,
		`{"text": " I", "segment": 1, "start_time": 3.2, "is_final": false}`,
		`{"text": " I WANT YOU", "segment": 1, "start_time": 3.2, "is_final": true, "is_eof": true, "tokens": [" I", " WANT", " YOU"], "timestamps": [0.4, 0.56, 0.8]}`,
		`Done!`)
	s := NewSherpaServer("http" + strings.TrimPrefix(f.URL, "http") + "/")
	if !strings.HasPrefix(s.URL, "ws://127.0.0.1:") || strings.HasSuffix(s.URL, "/") {
		t.Fatalf("an http address became %s", s.URL)
	}
	audio := clip(1.3)
	got, err := s.Transcribe(context.Background(), TranscriptionRequest{Audio: audio})
	if err != nil {
		t.Fatal(err)
	}
	// Capitals alone carry no case, so the text comes in lower case.
	if got.Text != "come here watson i want you" {
		t.Errorf("text %q", got.Text)
	}
	if len(got.Segments) != 2 || got.Segments[0].Text != "come here watson" || got.Segments[1].Text != "i want you" ||
		got.Segments[0].Start != 320*time.Millisecond || got.Segments[1].Start != 3600*time.Millisecond {
		t.Errorf("segments %+v", got.Segments)
	}
	var words []string
	for _, w := range got.Words {
		words = append(words, w.Word)
		if w.End <= w.Start {
			t.Errorf("%s ends at %s, before it starts at %s", w.Word, w.End, w.Start)
		}
	}
	if strings.Join(words, " ") != "come here watson i want you" {
		t.Errorf("words %v", words)
	}
	// A word runs to the next one's start; times count from the audio's start.
	if w := got.Words[1]; w.Start != 600*time.Millisecond || w.End != 960*time.Millisecond {
		t.Errorf("here: %s to %s", w.Start, w.End)
	}
	if w := got.Words[5]; w.Start != 4*time.Second {
		t.Errorf("you starts at %s", w.Start)
	}

	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.samples) != len(audio.Samples) || len(f.texts) != 1 || f.texts[0] != "Done" {
		t.Fatalf("the server got %d samples of %d and %v", len(f.samples), len(audio.Samples), f.texts)
	}
	for _, i := range []int{0, 1, 999, len(audio.Samples) - 1} {
		if want := float32(audio.Samples[i]) / 32768; f.samples[i] != want {
			t.Errorf("sample %d went as %v, want %v", i, f.samples[i], want)
		}
	}
	// A server may turn a browser's page away by its Origin; Vibium sends none.
	if f.origin != "" {
		t.Errorf("sent the origin %q", f.origin)
	}
	if _, err := s.Synthesize(context.Background(), SynthesisRequest{Text: "hi"}); err != ErrUnsupported {
		t.Errorf("synthesize: %v", err)
	}
}

// A server that sends only text and segment numbers, and text with case
// of its own, is read as it is.
func TestSherpaServerPlainAnswers(t *testing.T) {
	f := newFakeSherpa(t, `{"text": "Come here,", "segment": 0}`, `{"text": "Come here, Watson.", "segment": 0}`, `{"text": "", "segment": 1}`, `not json`, `Done!`)
	got, err := NewSherpaServer(f.URL).Transcribe(context.Background(), TranscriptionRequest{Audio: clip(0.2)})
	if err != nil || got.Text != "Come here, Watson." || len(got.Segments) != 0 || len(got.Words) != 0 {
		t.Errorf("%+v %v", got, err)
	}
	// Stereo is mixed down before it goes.
	stereo := Audio{SampleRate: Rate, Channels: 2, Samples: []int16{100, 300, -200, -400}}
	if _, err := NewSherpaServer(f.URL).Transcribe(context.Background(), TranscriptionRequest{Audio: stereo}); err != nil {
		t.Fatal(err)
	}
	f.mu.Lock()
	n := len(f.samples)
	last := f.samples[n-2:]
	f.mu.Unlock()
	if last[0] != 200.0/32768 || last[1] != -300.0/32768 {
		t.Errorf("stereo went as %v", last)
	}
}

func TestSherpaServerFailuresSayWhy(t *testing.T) {
	_, err := NewSherpaServer("ws://127.0.0.1:1").Transcribe(context.Background(), TranscriptionRequest{Audio: clip(0.1)})
	if Code(err) != "engine_failed" || !strings.Contains(err.Error(), "no sherpa-onnx server answered at ws://127.0.0.1:1") || !strings.Contains(err.Error(), "VIBIUM_SHERPA_URL") {
		t.Errorf("no server: %v", err)
	}
	// A server that hangs up without its Done! did not finish.
	f := newFakeSherpa(t, `{"text": "HALF", "segment": 0}`)
	_, err = NewSherpaServer(f.URL).Transcribe(context.Background(), TranscriptionRequest{Audio: clip(0.1)})
	if Code(err) != "engine_failed" || !strings.Contains(err.Error(), "stopped before it finished") {
		t.Errorf("a server that hangs up: %v", err)
	}
	_, err = NewSherpaServer(f.URL).Transcribe(context.Background(), TranscriptionRequest{Audio: Audio{SampleRate: 8000, Channels: 1, Samples: make([]int16, 800)}})
	if Code(err) != "engine_failed" || !strings.Contains(err.Error(), "16000 Hz") {
		t.Errorf("audio at another rate: %v", err)
	}
	// A call that is cancelled does not wait for a silent server.
	silent := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := (&websocket.Upgrader{}).Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}))
	defer silent.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()
	start := time.Now()
	_, err = NewSherpaServer(silent.URL).Transcribe(ctx, TranscriptionRequest{Audio: clip(0.1)})
	if err != context.DeadlineExceeded || time.Since(start) > 5*time.Second {
		t.Errorf("a cancelled call: %v after %s", err, time.Since(start))
	}
}

func TestSherpaServerIsChosenByItsURL(t *testing.T) {
	// The URL alone picks it, ahead of an installed engine and a hosted key.
	e := envOf(map[string]string{"VIBIUM_SHERPA_URL": "ws://127.0.0.1:6010", "VIBIUM_LISTEN_MODEL": "zipformer-en", "OPENAI_API_KEY": "sk-test"}, "whisper-cli")
	hear, err := e.Listen("", "")
	if err != nil || hear.Provider.Name() != "sherpa-server" || hear.Provider.(*SherpaServer).URL != "ws://127.0.0.1:6010" || hear.Model != "zipformer-en" {
		t.Fatalf("listen %+v %v", hear, err)
	}
	// It listens only: say goes on to the next choice.
	if say, err := e.Say("", "", ""); err != nil || say.Provider.Name() != "openai" {
		t.Errorf("say %+v %v", say, err)
	}
	// Named with no URL, it is sherpa-onnx's own default address.
	hear, err = envOf(nil).Listen("sherpa-server", "")
	if err != nil || hear.Provider.(*SherpaServer).URL != SherpaServerURL {
		t.Errorf("named %+v %v", hear, err)
	}
	if _, err := envOf(nil).Say("sherpa-server", "", ""); Code(err) != "not_configured" {
		t.Errorf("say through it: %v", err)
	}
	// A command the user set, or a provider they named, still wins.
	both := envOf(map[string]string{"VIBIUM_SHERPA_URL": "ws://x", "VIBIUM_LISTEN_COMMAND": "mystt {input}"})
	if hear, _ := both.Listen("", ""); hear.Provider.Name() != "command" {
		t.Errorf("with a command set: %s", hear.Provider.Name())
	}
	if hear, _ := e.Listen("openai", ""); hear.Provider.Name() != "openai" {
		t.Errorf("with a provider named: %s", hear.Provider.Name())
	}
}

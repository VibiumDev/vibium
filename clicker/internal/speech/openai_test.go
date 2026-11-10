package speech

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestOpenAITranscribe(t *testing.T) {
	var gotAuth, gotModel, gotLanguage, gotFormat string
	var gotBytes int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/audio/transcriptions" {
			http.NotFound(w, r)
			return
		}
		gotAuth = r.Header.Get("Authorization")
		r.ParseMultipartForm(1 << 20)
		gotModel, gotLanguage, gotFormat = r.FormValue("model"), r.FormValue("language"), r.FormValue("response_format")
		f, _, _ := r.FormFile("file")
		raw, _ := io.ReadAll(f)
		gotBytes = len(raw)
		w.Write([]byte(`{"text": " Come here, Watson. "}`))
	}))
	defer srv.Close()
	o := NewOpenAI("sk-test")
	o.BaseURL = srv.URL
	clip := Audio{SampleRate: Rate, Channels: 1, Samples: make([]int16, Rate)}
	tr, err := o.Transcribe(context.Background(), TranscriptionRequest{Audio: clip, Model: "gpt-transcribe", Language: "en-US"})
	if err != nil || tr.Text != "Come here, Watson." {
		t.Fatalf("%+v %v", tr, err)
	}
	if gotAuth != "Bearer sk-test" || gotModel != "gpt-transcribe" || gotLanguage != "en" || gotFormat != "json" || gotBytes != 32044 {
		t.Fatalf("request: auth %q model %q language %q format %q, %d bytes", gotAuth, gotModel, gotLanguage, gotFormat, gotBytes)
	}
}

func TestOpenAITranscribeErrorNamesTheCause(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(401)
		w.Write([]byte(`{"error": {"message": "Incorrect API key provided"}}`))
	}))
	defer srv.Close()
	o := NewOpenAI("sk-secret-value")
	o.BaseURL = srv.URL
	_, err := o.Transcribe(context.Background(), TranscriptionRequest{Audio: Audio{SampleRate: Rate, Channels: 1}, Model: "m"})
	if Code(err) != "provider_error" || !strings.Contains(err.Error(), "401") || !strings.Contains(err.Error(), "Incorrect API key") {
		t.Fatalf("%v", err)
	}
	if strings.Contains(err.Error(), "sk-secret-value") {
		t.Fatal("the error carries the key")
	}
}

// realtimeServer plays the part of the Realtime API. reads lists, per
// connection, what the model claims to have said.
func realtimeServer(t *testing.T, reads []string, seen *[]map[string]interface{}) (*httptest.Server, *int32) {
	t.Helper()
	var conns int32
	up := websocket.Upgrader{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer sk-test" || r.URL.Query().Get("model") == "" {
			http.Error(w, `{"error": {"message": "bad request"}}`, 401)
			return
		}
		c, err := up.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer c.Close()
		n := int(atomic.AddInt32(&conns, 1)) - 1
		for {
			var ev map[string]interface{}
			if err := c.ReadJSON(&ev); err != nil {
				return
			}
			if n == 0 && seen != nil {
				*seen = append(*seen, ev)
			}
			if ev["type"] != "response.create" {
				continue
			}
			pcm := Audio{SampleRate: 24000, Channels: 1, Samples: tone(24000, 300, -20, 500*time.Millisecond)}.Bytes()
			half := len(pcm) / 2
			c.WriteJSON(map[string]string{"type": "response.created"})
			c.WriteJSON(map[string]string{"type": "response.output_audio.delta", "delta": base64.StdEncoding.EncodeToString(pcm[:half])})
			c.WriteJSON(map[string]string{"type": "response.output_audio.delta", "delta": base64.StdEncoding.EncodeToString(pcm[half:])})
			c.WriteJSON(map[string]string{"type": "response.output_audio_transcript.done", "transcript": reads[min(n, len(reads)-1)]})
			c.WriteJSON(map[string]interface{}{"type": "response.done", "response": map[string]string{"status": "completed"}})
		}
	}))
	return srv, &conns
}

func realtimeClient(srv *httptest.Server) *OpenAI {
	o := NewOpenAI("sk-test")
	o.RealtimeURL = "ws" + strings.TrimPrefix(srv.URL, "http")
	return o
}

func TestOpenAISpeechEndpoint(t *testing.T) {
	var got map[string]interface{}
	var auth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/audio/speech" {
			http.NotFound(w, r)
			return
		}
		auth = r.Header.Get("Authorization")
		json.NewDecoder(r.Body).Decode(&got)
		raw := Audio{SampleRate: 24000, Channels: 1, Samples: tone(24000, 300, -20, 500*time.Millisecond)}.WAV()
		for i := 40; i < 44; i++ { // the endpoint streams: no data length
			raw[i] = 0xFF
		}
		w.Write(raw)
	}))
	defer srv.Close()
	o := NewOpenAI("sk-test")
	o.BaseURL = srv.URL
	a, err := o.Synthesize(context.Background(), SynthesisRequest{Text: "Come here, Watson", Model: "gpt-4o-mini-tts", Voice: "alloy", Speed: 1.25})
	if err != nil || a.SampleRate != 24000 || len(a.Samples) != 12000 {
		t.Fatalf("%d samples at %d Hz: %v", len(a.Samples), a.SampleRate, err)
	}
	if auth != "Bearer sk-test" || got["model"] != "gpt-4o-mini-tts" || got["voice"] != "alloy" || got["input"] != "Come here, Watson" || got["response_format"] != "wav" || got["speed"] != 1.25 {
		t.Fatalf("request %v", got)
	}

	refused := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(400)
		w.Write([]byte(`{"error": {"message": "string too long"}}`))
	}))
	defer refused.Close()
	o.BaseURL = refused.URL
	if _, err := o.Synthesize(context.Background(), SynthesisRequest{Text: "x", Model: "gpt-4o-mini-tts", Voice: "alloy"}); Code(err) != "provider_error" || !strings.Contains(err.Error(), "string too long") {
		t.Fatalf("%v", err)
	}
}

func TestOpenAIRealtimeModelsSpeakThroughTheRealtimeAPI(t *testing.T) {
	var seen []map[string]interface{}
	srv, conns := realtimeServer(t, []string{"Come here, Watson."}, &seen)
	defer srv.Close()
	a, err := realtimeClient(srv).Synthesize(context.Background(), SynthesisRequest{Text: "Come here, Watson", Model: "gpt-realtime-2.1-mini", Voice: "alloy"})
	if err != nil {
		t.Fatal(err)
	}
	if a.SampleRate != 24000 || a.Channels != 1 || len(a.Samples) != 12000 || *conns != 1 {
		t.Fatalf("%d samples at %d Hz over %d connections", len(a.Samples), a.SampleRate, *conns)
	}
	raw, _ := json.Marshal(seen)
	// The text gains the final period it lacked: without one the model
	// drops the last word half the time.
	for _, want := range []string{`"output_modalities":["audio"]`, `"voice":"alloy"`, `"rate":24000`, `"type":"input_text"`, `"text":"Come here, Watson."`, "Read the user's message aloud exactly"} {
		if !strings.Contains(string(raw), want) {
			t.Errorf("the session is missing %s", want)
		}
	}
}

func TestOpenAIRealtimeRefusal(t *testing.T) {
	srv, _ := realtimeServer(t, []string{"x"}, nil)
	defer srv.Close()
	o := realtimeClient(srv)
	o.Key = "sk-wrong"
	_, err := o.Synthesize(context.Background(), SynthesisRequest{Text: "hi", Model: "gpt-realtime-2.1-mini", Voice: "alloy"})
	if Code(err) != "provider_error" || !strings.Contains(err.Error(), "401") {
		t.Fatalf("%v", err)
	}
}

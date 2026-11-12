package speech

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// fakePiper stands in for python3 -m piper.http_server: /info names the
// loaded voice and /synthesize answers a WAV.
func fakePiper(t *testing.T, voice string, got *map[string]interface{}) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/info":
			json.NewEncoder(w).Encode(map[string]interface{}{"voice": map[string]string{"name": voice}})
		case r.Method == http.MethodPost && r.URL.Path == "/synthesize":
			raw, _ := io.ReadAll(r.Body)
			if got != nil {
				json.Unmarshal(raw, got)
			}
			w.Header().Set("Content-Type", "audio/wav")
			w.Write(Audio{SampleRate: 16000, Channels: 1, Samples: make([]int16, 1600)}.WAV())
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestPiperServerSpeaks(t *testing.T) {
	var got map[string]interface{}
	srv := fakePiper(t, "en_US-danny-low", &got)
	p := NewPiperServer(srv.URL + "/")
	a, err := p.Synthesize(context.Background(), SynthesisRequest{Text: `She said "hello".`})
	if err != nil || a.SampleRate != 16000 || len(a.Samples) != 1600 {
		t.Fatalf("audio %d Hz, %d samples, %v", a.SampleRate, len(a.Samples), err)
	}
	if len(got) != 1 || got["text"] != `She said "hello".` {
		t.Errorf("a plain request sent %v", got)
	}
	if _, err := p.Synthesize(context.Background(), SynthesisRequest{Text: "hi", Voice: "en_US-amy-low", Speed: 2}); err != nil {
		t.Fatal(err)
	}
	if got["voice"] != "en_US-amy-low" || got["length_scale"] != 0.5 {
		t.Errorf("voice and speed went as %v", got)
	}
	if _, err := p.Transcribe(context.Background(), TranscriptionRequest{}); err != ErrUnsupported {
		t.Errorf("transcribe: %v", err)
	}
}

func TestPiperServerCacheFollowsTheLoadedVoice(t *testing.T) {
	danny := NewPiperServer(fakePiper(t, "en_US-danny-low", nil).URL)
	if s := danny.CacheSettings(SynthesisRequest{}); s["server_voice"] != "en_US-danny-low" || s["url"] != danny.URL {
		t.Errorf("settings %v", s)
	}
	// A request that names its voice does not depend on the server's own.
	if s := danny.CacheSettings(SynthesisRequest{Voice: "en_US-amy-low"}); len(s) != 1 || s["url"] != danny.URL {
		t.Errorf("settings with a voice %v", s)
	}
	gone := NewPiperServer("http://127.0.0.1:1")
	if s := gone.CacheSettings(SynthesisRequest{}); s["server_voice"] != "" {
		t.Errorf("settings with no server %v", s)
	}
}

func TestPiperServerFailuresSayWhy(t *testing.T) {
	_, err := NewPiperServer("http://127.0.0.1:1").Synthesize(context.Background(), SynthesisRequest{Text: "hi"})
	if Code(err) != "engine_failed" || !strings.Contains(err.Error(), "piper.http_server") || !strings.Contains(err.Error(), "VIBIUM_PIPER_URL") {
		t.Errorf("no server: %v", err)
	}
	broken := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("how") == "text" {
			w.Write([]byte("not a wav"))
			return
		}
		http.Error(w, "ValueError: No text provided", http.StatusInternalServerError)
	}))
	defer broken.Close()
	_, err = NewPiperServer(broken.URL).Synthesize(context.Background(), SynthesisRequest{Text: "hi"})
	if Code(err) != "engine_failed" || !strings.Contains(err.Error(), "HTTP 500") || !strings.Contains(err.Error(), "No text provided") {
		t.Errorf("a server error: %v", err)
	}
	p := NewPiperServer(broken.URL)
	p.URL += "/x?how=text&"
	if _, err := p.Synthesize(context.Background(), SynthesisRequest{Text: "hi"}); Code(err) != "engine_failed" || !strings.Contains(err.Error(), "PCM WAV") {
		t.Errorf("an answer that is no WAV: %v", err)
	}
}

func TestPiperServerIsChosenByItsURL(t *testing.T) {
	// The URL alone picks it, ahead of an installed engine and a hosted key.
	e := envOf(map[string]string{"VIBIUM_PIPER_URL": "http://127.0.0.1:5005", "OPENAI_API_KEY": "sk-test"}, "piper")
	say, err := e.Say("", "", "")
	if err != nil || say.Provider.Name() != "piper-server" || say.Provider.(*PiperServer).URL != "http://127.0.0.1:5005" || say.Voice != "" {
		t.Fatalf("say %+v %v", say, err)
	}
	// It speaks only: listen goes on to the next choice.
	if hear, err := e.Listen("", ""); err != nil || hear.Provider.Name() != "openai" {
		t.Errorf("listen %+v %v", hear, err)
	}
	// Named with no URL, it is Piper's own default address.
	say, err = envOf(nil).Say("piper-server", "", "en_US-amy-low")
	if err != nil || say.Provider.(*PiperServer).URL != PiperServerURL || say.Voice != "en_US-amy-low" {
		t.Errorf("named %+v %v", say, err)
	}
	if _, err := envOf(nil).Listen("piper-server", ""); Code(err) != "not_configured" {
		t.Errorf("listen through it: %v", err)
	}
	// A command the user set, or a provider they named, still wins.
	both := envOf(map[string]string{"VIBIUM_PIPER_URL": "http://x", "VIBIUM_SAY_COMMAND": "mytts {output}"})
	if say, _ := both.Say("", "", ""); say.Provider.Name() != "command" {
		t.Errorf("with a command set: %s", say.Provider.Name())
	}
	if say, _ := e.Say("openai", "", ""); say.Provider.Name() != "openai" {
		t.Errorf("with a provider named: %s", say.Provider.Name())
	}
}

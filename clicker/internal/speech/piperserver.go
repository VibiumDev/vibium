package speech

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"
)

// PiperServerURL is where Piper's own HTTP server listens by default.
const PiperServerURL = "http://127.0.0.1:5000"

// PiperServer speaks through a Piper HTTP server the user runs
// (python3 -m piper.http_server). The server keeps its voice loaded, and
// loading is most of what the piper command takes per call. Piper stays
// the user's own program: Vibium sends text and gets a WAV back.
type PiperServer struct {
	URL  string
	HTTP *http.Client
}

func NewPiperServer(url string) *PiperServer {
	if url == "" {
		url = PiperServerURL
	}
	return &PiperServer{URL: strings.TrimRight(url, "/"), HTTP: &http.Client{Timeout: 2 * time.Minute}}
}

func (p *PiperServer) Name() string { return "piper-server" }

func (p *PiperServer) Transcribe(context.Context, TranscriptionRequest) (Transcript, error) {
	return Transcript{}, ErrUnsupported
}

// CacheSettings names the server and, when the request names no voice,
// the voice the server has loaded. A server started again with another
// voice must not replay the old one's audio from the cache.
func (p *PiperServer) CacheSettings(req SynthesisRequest) map[string]string {
	settings := map[string]string{"url": p.URL}
	if req.Voice == "" {
		settings["server_voice"] = p.loadedVoice()
	}
	return settings
}

// loadedVoice is "" for a server that does not answer. The synthesis that
// follows says why.
func (p *PiperServer) loadedVoice() string {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.URL+"/info", nil)
	if err != nil {
		return ""
	}
	resp, err := p.HTTP.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	var info struct {
		Voice struct {
			Name string `json:"name"`
		} `json:"voice"`
	}
	if resp.StatusCode != http.StatusOK || json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&info) != nil {
		return ""
	}
	return info.Voice.Name
}

func (p *PiperServer) Synthesize(ctx context.Context, req SynthesisRequest) (Audio, error) {
	body := map[string]interface{}{"text": req.Text}
	if req.Voice != "" {
		body["voice"] = req.Voice
	}
	// Piper's length_scale is the inverse of a speaking rate.
	if req.Speed > 0 && req.Speed != 1 {
		body["length_scale"] = 1 / req.Speed
	}
	raw, _ := json.Marshal(body)
	hreq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.URL+"/synthesize", bytes.NewReader(raw))
	if err != nil {
		return Audio{}, err
	}
	hreq.Header.Set("Content-Type", "application/json")
	resp, err := p.HTTP.Do(hreq)
	if err != nil {
		return Audio{}, errf("engine_failed", "no Piper server answered at %s: %v. Start one with `python3 -m piper.http_server -m <voice.onnx> --host 127.0.0.1`, or set VIBIUM_PIPER_URL", p.URL, err)
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(io.LimitReader(resp.Body, 64<<20))
	if resp.StatusCode != http.StatusOK {
		msg := strings.TrimSpace(string(data))
		if len(msg) > 300 {
			msg = msg[len(msg)-300:]
		}
		return Audio{}, errf("engine_failed", "the Piper server at %s answered HTTP %d: %s", p.URL, resp.StatusCode, msg)
	}
	audio, err := ParseWAV(data)
	if err != nil || len(audio.Samples) == 0 {
		return Audio{}, errf("engine_failed", "the Piper server at %s did not send a PCM WAV", p.URL)
	}
	return audio, nil
}

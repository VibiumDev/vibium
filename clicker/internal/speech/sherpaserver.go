package speech

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"math"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/gorilla/websocket"
)

// SherpaServerURL is where sherpa-onnx's streaming WebSocket servers
// listen by default.
const SherpaServerURL = "ws://127.0.0.1:6006"

// SherpaServer transcribes through a sherpa-onnx streaming WebSocket
// server the user runs. The server keeps its model loaded, and loading
// is most of what a local engine takes per call. Vibium sends the audio
// and gets the words back.
type SherpaServer struct {
	URL string
}

func NewSherpaServer(url string) *SherpaServer {
	if url == "" {
		url = SherpaServerURL
	}
	// An http address is the same server's WebSocket one.
	if strings.HasPrefix(url, "http") {
		url = "ws" + strings.TrimPrefix(url, "http")
	}
	return &SherpaServer{URL: strings.TrimRight(url, "/")}
}

func (s *SherpaServer) Name() string { return "sherpa-server" }

func (s *SherpaServer) Synthesize(context.Context, SynthesisRequest) (Audio, error) {
	return Audio{}, ErrUnsupported
}

func (s *SherpaServer) CacheSettings(SynthesisRequest) map[string]string { return nil }

// sherpaResult is one message from the server: the text so far of one
// segment. A server starts a new segment after a pause in the speech.
// Token times count from the segment's start.
type sherpaResult struct {
	Text       string    `json:"text"`
	Segment    int       `json:"segment"`
	StartTime  float64   `json:"start_time"`
	Tokens     []string  `json:"tokens"`
	Timestamps []float64 `json:"timestamps"`
}

// sherpaChunk is how much audio one message carries, in samples.
const sherpaChunk = Rate / 2

// sherpaQuiet is how long the server may send nothing before the call
// gives up on it.
const sherpaQuiet = 2 * time.Minute

// Transcribe sends the audio as 32-bit float samples, then the text
// Done, and reads results until the server's own Done!.
func (s *SherpaServer) Transcribe(ctx context.Context, req TranscriptionRequest) (Transcript, error) {
	audio := req.Audio.Mono()
	if audio.SampleRate != Rate {
		return Transcript{}, errf("engine_failed", "the sherpa-onnx server takes %d Hz audio, not %d Hz", Rate, audio.SampleRate)
	}
	dialer := websocket.Dialer{HandshakeTimeout: 10 * time.Second}
	conn, _, err := dialer.DialContext(ctx, s.URL, nil)
	if err != nil {
		return Transcript{}, errf("engine_failed", "no sherpa-onnx server answered at %s: %v. Start a streaming WebSocket server with a model loaded, or set VIBIUM_SHERPA_URL", s.URL, err)
	}
	defer conn.Close()
	// The connection has no deadline of its own: a call that is cancelled
	// or runs out of time closes it, which ends the read below.
	done := make(chan struct{})
	defer close(done)
	go func() {
		select {
		case <-ctx.Done():
			conn.Close()
		case <-done:
		}
	}()

	sendErr := make(chan error, 1)
	go func() {
		buf := make([]byte, 4*sherpaChunk)
		for i := 0; i < len(audio.Samples); i += sherpaChunk {
			chunk := audio.Samples[i:min(i+sherpaChunk, len(audio.Samples))]
			for j, v := range chunk {
				binary.LittleEndian.PutUint32(buf[4*j:], math.Float32bits(float32(v)/32768))
			}
			if err := conn.WriteMessage(websocket.BinaryMessage, buf[:4*len(chunk)]); err != nil {
				sendErr <- err
				return
			}
		}
		sendErr <- conn.WriteMessage(websocket.TextMessage, []byte("Done"))
	}()

	segments := map[int]sherpaResult{}
	for {
		conn.SetReadDeadline(time.Now().Add(sherpaQuiet))
		kind, data, err := conn.ReadMessage()
		if err != nil {
			if ctx.Err() != nil {
				return Transcript{}, ctx.Err()
			}
			select {
			case werr := <-sendErr:
				if werr != nil {
					err = werr
				}
			default:
			}
			return Transcript{}, errf("engine_failed", "the sherpa-onnx server at %s stopped before it finished: %v", s.URL, err)
		}
		if kind != websocket.TextMessage {
			continue
		}
		if strings.TrimSpace(string(data)) == "Done!" {
			break
		}
		var r sherpaResult
		if json.Unmarshal(data, &r) != nil {
			continue
		}
		segments[r.Segment] = r
	}
	return sherpaTranscript(segments), nil
}

// sherpaTranscript joins the segments' last results in order. A model
// trained on upper-case text answers in capitals, which say nothing
// about case, so such text is given in lower case.
func sherpaTranscript(segments map[int]sherpaResult) Transcript {
	order := make([]int, 0, len(segments))
	for n := range segments {
		order = append(order, n)
	}
	sort.Ints(order)
	lower := true
	for _, r := range segments {
		lower = lower && !strings.ContainsFunc(r.Text, unicode.IsLower)
	}
	fix := func(text string) string {
		if lower {
			text = strings.ToLower(text)
		}
		return strings.Join(strings.Fields(text), " ")
	}
	at := func(seconds float64) time.Duration { return time.Duration(seconds * float64(time.Second)) }

	var t Transcript
	var texts []string
	for _, n := range order {
		r := segments[n]
		text := fix(r.Text)
		if text == "" {
			continue
		}
		texts = append(texts, text)
		if len(r.Tokens) == 0 || len(r.Tokens) != len(r.Timestamps) {
			continue
		}
		// A token that starts with a space, or with the mark some models
		// use for one, starts a word.
		first := len(t.Words)
		for i, token := range r.Tokens {
			start := at(r.StartTime + r.Timestamps[i])
			piece := strings.TrimLeft(token, " ▁")
			if len(t.Words) == first || piece != token {
				t.Words = append(t.Words, Word{Start: start, End: start})
			}
			w := &t.Words[len(t.Words)-1]
			w.Word += fix(piece)
			w.End = start
		}
		// A token's time is when it was heard, so a word ends where the
		// next begins, and the last one a little after its last token.
		for i := first; i < len(t.Words); i++ {
			if i+1 < len(t.Words) {
				t.Words[i].End = t.Words[i+1].Start
			} else {
				t.Words[i].End += 200 * time.Millisecond
			}
		}
		if len(t.Words) > first {
			t.Segments = append(t.Segments, Segment{Start: t.Words[first].Start, End: t.Words[len(t.Words)-1].End, Text: text})
		}
	}
	t.Text = strings.Join(texts, " ")
	return t
}

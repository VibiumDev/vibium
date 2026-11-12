package agent

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/vibium/clicker/internal/bidi"
	"github.com/vibium/clicker/internal/speech"
)

// roadieAudio is the session target for a device on a Roadie: say plays
// into the device's microphone and listen captures what the device plays,
// over the roadie:audio extension on the session's own socket. Every time
// is on Roadie's clock, so reply latency never crosses a network.
type roadieAudio struct {
	h      *Handlers
	client *bidi.Client
	// One capture runs on the device however many readers there are: a
	// recording's track and a listen read the same stream.
	tap *speech.Tap
}

// roadieChunkGap is how far a chunk may start after the one before it
// ended before the difference counts as dropped audio.
const roadieChunkGap = 30.0 // ms

const roadieMaxClip = 60 * time.Second

func speechErr(code, message string) error { return &speech.Error{Code: code, Message: message} }

// roadieAudioFor returns the target when the session speaks roadie:audio.
func (h *Handlers) roadieAudioFor(client *bidi.Client) (*roadieAudio, bool) {
	if _, err := client.SendCommandWithTimeout("roadie:audio.clock", map[string]interface{}{}, 10*time.Second); err != nil {
		return nil, false
	}
	r := &roadieAudio{h: h, client: client}
	r.tap = speech.NewTap(r.open)
	return r, true
}

// Now is the present on Roadie's clock.
func (r *roadieAudio) Now() float64 {
	now, _ := r.Sync()
	return now
}

// Sync reads Roadie's clock a few times and keeps the reading with the
// shortest round trip, whose error is at most half that trip. It returns
// Roadie's time now and that round trip in milliseconds.
func (r *roadieAudio) Sync() (float64, float64) {
	best, bestRTT := 0.0, -1.0
	var bestAt time.Time
	for i := 0; i < 5; i++ {
		sent := time.Now()
		resp, err := r.client.SendCommandWithTimeout("roadie:audio.clock", map[string]interface{}{}, 5*time.Second)
		got := time.Now()
		var res struct {
			Now float64 `json:"now"`
		}
		if err != nil || json.Unmarshal(resp.Result, &res) != nil {
			continue
		}
		if rtt := float64(got.Sub(sent)) / float64(time.Millisecond); bestRTT < 0 || rtt < bestRTT {
			best, bestRTT, bestAt = res.Now, rtt, sent.Add(got.Sub(sent)/2)
		}
	}
	if bestRTT < 0 {
		return 0, 0
	}
	return best + float64(time.Since(bestAt))/float64(time.Millisecond), bestRTT
}

func (r *roadieAudio) Name() string             { return "session" }
func (r *roadieAudio) Key() string              { return "session" }
func (r *roadieAudio) Live() bool               { return true }
func (r *roadieAudio) EchoGuard() time.Duration { return 0 }

func (r *roadieAudio) Play(_ context.Context, a speech.Audio) (float64, float64, error) {
	clip := a.Resample(speech.Rate)
	if clip.Duration() > roadieMaxClip {
		return 0, 0, speechErr("no_audio", "a clip for the device is at most 60 seconds")
	}
	resp, err := r.client.SendCommandWithTimeout("roadie:audio.play", map[string]interface{}{
		"format": "s16le", "sampleRate": speech.Rate, "channels": 1,
		"data": base64.StdEncoding.EncodeToString(clip.Bytes()),
	}, clip.Duration()+30*time.Second)
	if err != nil {
		return 0, 0, speechErr("no_audio", "the device did not take the audio: "+err.Error())
	}
	var res struct {
		StartTime float64 `json:"startTime"`
		EndTime   float64 `json:"endTime"`
	}
	if err := json.Unmarshal(resp.Result, &res); err != nil || res.EndTime < res.StartTime {
		return 0, 0, speechErr("no_audio", "the device gave no playback times")
	}
	return res.StartTime, res.EndTime, nil
}

func (r *roadieAudio) Capture(ctx context.Context) (speech.Capture, error) { return r.tap.Capture(ctx) }

func (r *roadieAudio) open(context.Context) (speech.Capture, error) {
	c := &roadieCapture{h: r.h, client: r.client, chunks: make(chan speech.Chunk, 256), done: make(chan struct{})}
	// The sink is in place before the capture starts, so no chunk is lost.
	r.h.setAudioSink(c)
	resp, err := r.client.SendCommandWithTimeout("roadie:audio.capture.start", map[string]interface{}{
		"format": "s16le", "sampleRate": speech.Rate, "channels": 1, "chunkMs": 100,
	}, 15*time.Second)
	if err != nil {
		r.h.clearAudioSink(c)
		return nil, speechErr("no_audio", "the device's audio cannot be captured: "+err.Error())
	}
	var res struct {
		Capture string `json:"capture"`
	}
	json.Unmarshal(resp.Result, &res)
	c.id = res.Capture
	return c, nil
}

// roadieCapture turns roadie:audio.chunk events into a capture.
type roadieCapture struct {
	h      *Handlers
	client *bidi.Client
	id     string
	chunks chan speech.Chunk
	done   chan struct{}
	once   sync.Once

	next    float64 // when the next chunk should start; 0 before the first
	pending *speech.Chunk
}

// deliver takes one chunk event. It never blocks the event dispatcher: a
// reader that has fallen 25 seconds behind loses audio.
func (c *roadieCapture) deliver(chunk speech.Chunk) {
	select {
	case c.chunks <- chunk:
	case <-c.done:
	default:
	}
}

func (c *roadieCapture) Next(ctx context.Context) (speech.Chunk, error) {
	if c.pending != nil {
		chunk := *c.pending
		c.pending = nil
		c.next = chunk.Time + float64(len(chunk.Samples))*1000/speech.Rate
		return chunk, nil
	}
	select {
	case chunk := <-c.chunks:
		// Roadie drops audio when its source falls behind. The times show
		// it; silence keeps the audio on its clock.
		if gap := chunk.Time - c.next; c.next > 0 && gap > roadieChunkGap {
			c.pending = &chunk
			silence := speech.Chunk{Samples: make([]int16, int(gap*speech.Rate/1000)), Time: c.next, Lost: true}
			c.next = chunk.Time
			return silence, nil
		}
		c.next = chunk.Time + float64(len(chunk.Samples))*1000/speech.Rate
		return chunk, nil
	case <-c.done:
		return speech.Chunk{}, io.EOF
	case <-ctx.Done():
		return speech.Chunk{}, ctx.Err()
	}
}

func (c *roadieCapture) Close() {
	c.once.Do(func() {
		close(c.done)
		c.h.clearAudioSink(c)
		// Best effort: the session may already be gone.
		go c.client.SendCommandWithTimeout("roadie:audio.capture.stop", map[string]interface{}{"capture": c.id}, 5*time.Second)
	})
}

func (h *Handlers) setAudioSink(c *roadieCapture) {
	h.audioMu.Lock()
	h.audioSink = c
	h.audioMu.Unlock()
}

func (h *Handlers) clearAudioSink(c *roadieCapture) {
	h.audioMu.Lock()
	if h.audioSink == c {
		h.audioSink = nil
	}
	h.audioMu.Unlock()
}

// audioEvent routes a roadie:audio.chunk event to the running capture. It
// reports whether the event was one, so audio stays out of the BiDi log.
func (h *Handlers) audioEvent(msg string) bool {
	if !strings.Contains(msg, `"roadie:audio.chunk"`) {
		return false
	}
	var ev struct {
		Method string `json:"method"`
		Params struct {
			Capture string  `json:"capture"`
			Time    float64 `json:"time"`
			Data    string  `json:"data"`
		} `json:"params"`
	}
	if json.Unmarshal([]byte(msg), &ev) != nil || ev.Method != "roadie:audio.chunk" {
		return false
	}
	h.audioMu.Lock()
	sink := h.audioSink
	h.audioMu.Unlock()
	if sink == nil || (sink.id != "" && sink.id != ev.Params.Capture) {
		return true
	}
	pcm, err := base64.StdEncoding.DecodeString(ev.Params.Data)
	if err != nil {
		return true
	}
	sink.deliver(speech.Chunk{Samples: speech.FromBytes(pcm, speech.Rate, 1).Samples, Time: ev.Params.Time})
	return true
}

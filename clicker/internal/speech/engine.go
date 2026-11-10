package speech

import (
	"context"
	"errors"
	"io"
	"math"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// SayOptions are one say call's options. Empty values take the defaults.
type SayOptions struct {
	Text string
	// File is a WAV to play as it is, in place of text. Nothing is
	// rendered, checked, trimmed or cached.
	File     string
	Provider string
	Model    string
	Voice    string
	Language string
	Speed    float64
	NoCache  bool
	CacheDir string
}

// SayResult is what say returns.
type SayResult struct {
	Text       string  `json:"text"`
	Target     string  `json:"target"`
	Provider   string  `json:"provider"`
	Model      string  `json:"model"`
	Voice      string  `json:"voice"`
	DurationMs float64 `json:"durationMs"`
	Cached     bool    `json:"cached"`
	SHA256     string  `json:"sha256"`
	Path       string  `json:"path,omitempty"`
	File       string  `json:"file,omitempty"`

	// For a recording: the clip as played, and when its first and last
	// sample played on the target's clock.
	Clip      Audio   `json:"-"`
	PlayStart float64 `json:"-"`
	PlayEnd   float64 `json:"-"`
}

// ListenOptions are one listen call's options. Negative durations take
// the defaults; a zero Silence turns endpointing off.
type ListenOptions struct {
	Provider     string
	Model        string
	Language     string
	Silence      time.Duration
	Timeout      time.Duration
	Duration     time.Duration
	AllowSilence bool
	Path         string
	// NoTranscribe captures and detects speech without asking a provider
	// for the words. Onset and latency are still measured.
	NoTranscribe bool
}

// Listen defaults.
const (
	DefaultSilence  = time.Second
	DefaultTimeout  = 10 * time.Second
	DefaultDuration = 30 * time.Second
)

type SegmentResult struct {
	StartMs float64 `json:"startMs"`
	EndMs   float64 `json:"endMs"`
	Text    string  `json:"text"`
}

type WordResult struct {
	StartMs float64 `json:"startMs"`
	EndMs   float64 `json:"endMs"`
	Word    string  `json:"word"`
}

// ListenResult is what listen returns. Times count from the start of the
// capture, which is the end of the preceding say when there was one.
type ListenResult struct {
	Text   string `json:"text"`
	Speech bool   `json:"speech"`
	// Transcribed is false when the capture was not sent to a provider.
	Transcribed bool            `json:"transcribed"`
	Segments    []SegmentResult `json:"segments"`
	Words       []WordResult    `json:"words,omitempty"`
	OnsetMs     *float64        `json:"onsetMs,omitempty"`
	SpeechEndMs *float64        `json:"speechEndMs,omitempty"`
	LatencyMs   *float64        `json:"latencyMs,omitempty"`
	StoppedBy   string          `json:"stoppedBy"`
	DurationMs  float64         `json:"durationMs"`
	Target      string          `json:"target"`
	Provider    string          `json:"provider"`
	Model       string          `json:"model"`
	Path        string          `json:"path,omitempty"`

	// LostMs is how much of the capture is silence standing in for audio
	// the target dropped on its way here. A transcript may miss words
	// when it is not zero.
	LostMs float64 `json:"lostMs,omitempty"`

	// CaptureStart is when the capture began on the target's clock, for a
	// recording. Every time above counts from it.
	CaptureStart float64 `json:"-"`
}

const (
	windowLength  = 30 * time.Second
	clipPad       = 50 * time.Millisecond
	transcribePad = 300 * time.Millisecond
)

// noAudioAfter is how long a live target may stay silent on the wire
// before the pipe counts as broken.
var noAudioAfter = 3 * time.Second

// window is audio captured since a say ended, kept for the next listen.
type window struct {
	pump  *pump
	start float64 // target-clock ms: the end of the say
}

// Engine runs say and listen. It keeps one capture window per target, so
// a reply that starts before listen is called is still heard.
type Engine struct {
	Env Env

	mu      sync.Mutex
	windows map[string]*window
	busy    map[string]bool
}

func NewEngine(env Env) *Engine {
	return &Engine{Env: env, windows: map[string]*window{}, busy: map[string]bool{}}
}

// Close drops every capture window.
func (e *Engine) Close() {
	e.mu.Lock()
	defer e.mu.Unlock()
	for key, w := range e.windows {
		w.pump.Close()
		delete(e.windows, key)
	}
}

func (e *Engine) acquire(t Target) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.busy[t.Key()] {
		return errf("busy", "a say or listen is already running on %s", t.Name())
	}
	e.busy[t.Key()] = true
	return nil
}

func (e *Engine) release(t Target) {
	e.mu.Lock()
	delete(e.busy, t.Key())
	e.mu.Unlock()
}

func (e *Engine) takeWindow(key string) *window {
	e.mu.Lock()
	defer e.mu.Unlock()
	w := e.windows[key]
	delete(e.windows, key)
	return w
}

// Say renders the text, or loads it from the cache, and plays it into the
// target. It returns when playback has ended.
func (e *Engine) Say(ctx context.Context, t Target, o SayOptions) (*SayResult, error) {
	if o.File != "" {
		if strings.TrimSpace(o.Text) != "" {
			return nil, errors.New("say takes text or a file, not both")
		}
		raw, err := os.ReadFile(o.File)
		if err != nil {
			return nil, errf("no_audio", "cannot read %s: %v", o.File, err)
		}
		audio, err := ParseWAV(raw)
		if err != nil || len(audio.Samples) == 0 {
			return nil, errf("no_audio", "%s is not a PCM WAV file", o.File)
		}
		res := &SayResult{Target: t.Name(), Provider: "file", File: o.File, DurationMs: audio.DurationMs(), SHA256: audio.SHA256()}
		return res, e.play(ctx, t, audio, res)
	}
	if strings.TrimSpace(o.Text) == "" {
		return nil, errors.New("say needs text or a file")
	}
	choice, err := e.Env.Say(o.Provider, o.Model, o.Voice)
	if err != nil {
		return nil, err
	}
	dir := o.CacheDir
	if dir == "" {
		if dir, err = DefaultCacheDir(); err != nil {
			return nil, err
		}
	}
	cache := Cache{Dir: dir}
	req := SynthesisRequest{Text: o.Text, Model: choice.Model, Voice: choice.Voice, Language: o.Language, Speed: o.Speed}
	key := CacheKey{Version: cacheVersion, Text: o.Text, Provider: choice.Provider.Name(), Model: choice.Model, Voice: choice.Voice,
		Language: o.Language, Speed: o.Speed, Settings: choice.Provider.CacheSettings(req)}

	var audio Audio
	cached := false
	if !o.NoCache {
		audio, cached = cache.Get(key)
	}
	if !cached {
		if audio, err = e.render(ctx, choice, req); err != nil {
			return nil, err
		}
		if err := cache.Put(key, audio); err != nil {
			return nil, err
		}
	}

	res := &SayResult{Text: o.Text, Target: t.Name(), Provider: choice.Provider.Name(), Model: choice.Model, Voice: choice.Voice,
		DurationMs: audio.DurationMs(), Cached: cached, SHA256: audio.SHA256()}
	if err := e.play(ctx, t, audio, res); err != nil {
		return nil, err
	}
	return res, nil
}

// play sends a clip into the target and opens the capture window behind it.
func (e *Engine) play(ctx context.Context, t Target, audio Audio, res *SayResult) error {
	if err := e.acquire(t); err != nil {
		return err
	}
	defer e.release(t)
	if old := e.takeWindow(t.Key()); old != nil {
		old.pump.Close()
	}
	// Capture runs during playback, so a reply that starts the moment the
	// clip ends is not missed. A target that cannot be heard still speaks.
	var p *pump
	if t.Live() {
		if c, err := t.Capture(context.Background()); err == nil {
			p = startPump(c)
		}
	}
	begin, end, err := t.Play(ctx, audio)
	if err != nil {
		if p != nil {
			p.Close()
		}
		return err
	}
	res.Clip, res.PlayStart, res.PlayEnd = audio, begin, end
	if p != nil {
		p.Limit(end + float64(windowLength/time.Millisecond))
		e.mu.Lock()
		e.windows[t.Key()] = &window{pump: p, start: end}
		e.mu.Unlock()
	}
	if f, ok := t.(FileTarget); ok {
		res.Path = f.Path
	}
	return nil
}

// renderAttempts is how many times say asks a provider for a clip that
// says the text.
const renderAttempts = 3

// render synthesizes the text and checks the audio by transcribing it,
// when the provider can. A speech model's output is not trusted: a clip
// that does not say the text is rendered again, and never cached.
func (e *Engine) render(ctx context.Context, choice Choice, req SynthesisRequest) (Audio, error) {
	language := req.Language
	if language == "" {
		language = "en"
	}
	heard := ""
	for attempt := 1; attempt <= renderAttempts; attempt++ {
		rendered, err := choice.Provider.Synthesize(ctx, req)
		if err != nil {
			return Audio{}, err
		}
		audio := rendered.Mono().Trim(clipPad)
		if choice.HearModel == "" {
			return audio, nil
		}
		t, err := choice.Provider.Transcribe(ctx, TranscriptionRequest{Audio: audio.Resample(Rate), Model: choice.HearModel, Language: language})
		if errors.Is(err, ErrUnsupported) {
			return audio, nil
		}
		if err != nil {
			return Audio{}, err
		}
		if SaysTheText(req.Text, t.Text) {
			return audio, nil
		}
		heard = t.Text
	}
	return Audio{}, errf("provider_error", "%s did not say the text in %d renders; the last one said %q",
		choice.Provider.Name(), renderAttempts, heard)
}

// Listen captures from the target until the reply ends, and transcribes it.
func (e *Engine) Listen(ctx context.Context, t Target, o ListenOptions) (*ListenResult, error) {
	if o.Silence < 0 {
		o.Silence = DefaultSilence
	}
	if o.Timeout <= 0 {
		o.Timeout = DefaultTimeout
	}
	if o.Duration <= 0 {
		o.Duration = DefaultDuration
	}
	if o.Language == "" {
		o.Language = "en"
	}
	var choice Choice
	if !o.NoTranscribe {
		var err error
		if choice, err = e.Env.Listen(o.Provider, o.Model); err != nil {
			return nil, err
		}
	}
	if err := e.acquire(t); err != nil {
		return nil, err
	}
	defer e.release(t)

	var src *pump
	start, fromSay := math.NaN(), false
	if w := e.takeWindow(t.Key()); w != nil {
		src, start, fromSay = w.pump, w.start, true
		src.Unlimit()
	} else {
		c, err := t.Capture(ctx)
		if err != nil {
			return nil, err
		}
		src = startPump(c)
	}
	defer src.Close()

	det := NewDetector(o.Silence)
	if fromSay {
		det.Skip(t.EchoGuard())
	}
	var heard []int16
	lost := 0
	stoppedBy := ""
	for stoppedBy == "" {
		wait := ctx
		var cancel context.CancelFunc = func() {}
		if t.Live() {
			wait, cancel = context.WithTimeout(ctx, noAudioAfter)
		}
		chunk, err := src.Next(wait)
		cancel()
		switch {
		case err == io.EOF:
			det.Flush()
			stoppedBy = "eof"
			continue
		case errors.Is(err, context.DeadlineExceeded) && ctx.Err() == nil:
			return nil, errf("no_audio", "no audio arrived from %s for %s", t.Name(), noAudioAfter)
		case err != nil:
			return nil, err
		}
		samples := chunk.Samples
		if math.IsNaN(start) {
			start = chunk.Time
		}
		if before := int(math.Round((start - chunk.Time) * Rate / 1000)); before > 0 {
			if before >= len(samples) {
				continue
			}
			samples = samples[before:]
		}
		if room := int(o.Duration.Seconds()*Rate) - len(heard); len(samples) > room {
			samples = samples[:room]
		}
		heard = append(heard, samples...)
		if chunk.Lost {
			lost += len(samples)
		}
		det.Write(samples)
		_, started := det.Onset()
		if _, ended := det.End(); ended {
			stoppedBy = "silence"
		} else if len(heard) >= int(o.Duration.Seconds()*Rate) {
			stoppedBy = "duration"
		} else if !started && det.Elapsed() >= o.Timeout {
			stoppedBy = "timeout"
		}
	}

	captured := Audio{SampleRate: Rate, Channels: 1, Samples: heard}
	res := &ListenResult{Segments: []SegmentResult{}, StoppedBy: stoppedBy, DurationMs: captured.DurationMs(), Target: t.Name(), CaptureStart: start,
		LostMs: float64(lost) * 1000 / Rate}
	if !o.NoTranscribe {
		res.Provider, res.Model = choice.Provider.Name(), choice.Model
	}
	if o.Path != "" {
		if err := os.MkdirAll(filepath.Dir(o.Path), 0o755); err != nil {
			return nil, err
		}
		if err := writeAtomic(o.Path, captured.WAV()); err != nil {
			return nil, err
		}
		res.Path = o.Path
	}
	onset, started := det.Onset()
	if !started {
		if !o.AllowSilence {
			return nil, errf("no_speech", "listened to %s for %.1f s and heard no speech (peak level %.0f dBFS)",
				t.Name(), captured.Duration().Seconds(), captured.PeakDB())
		}
		return res, nil
	}
	end := det.LastSpeech()
	if end < onset {
		end = captured.Duration()
	}

	onsetMs, endMs := ms(onset), ms(end)
	res.Speech = true
	res.OnsetMs, res.SpeechEndMs = &onsetMs, &endMs
	if fromSay {
		latency := onsetMs
		res.LatencyMs = &latency
	}
	if o.NoTranscribe {
		return res, nil
	}

	from := max(onset-transcribePad, 0)
	to := min(end+transcribePad, captured.Duration())
	clip := Audio{SampleRate: Rate, Channels: 1, Samples: heard[samplesAt(from):min(samplesAt(to), len(heard))]}
	transcript, err := choice.Provider.Transcribe(ctx, TranscriptionRequest{Audio: clip, Model: choice.Model, Language: o.Language})
	if err != nil {
		return nil, err
	}
	res.Text, res.Transcribed = transcript.Text, true
	for _, s := range transcript.Segments {
		res.Segments = append(res.Segments, SegmentResult{StartMs: ms(from + s.Start), EndMs: ms(from + s.End), Text: s.Text})
	}
	if len(res.Segments) == 0 {
		res.Segments = append(res.Segments, SegmentResult{StartMs: onsetMs, EndMs: endMs, Text: transcript.Text})
	}
	for _, w := range transcript.Words {
		res.Words = append(res.Words, WordResult{StartMs: ms(from + w.Start), EndMs: ms(from + w.End), Word: w.Word})
	}
	return res, nil
}

func samplesAt(d time.Duration) int { return int(d.Seconds() * Rate) }

func ms(d time.Duration) float64 { return float64(d) / float64(time.Millisecond) }

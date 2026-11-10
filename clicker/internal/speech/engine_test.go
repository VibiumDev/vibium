package speech

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

// fakeProvider speaks a tone and hears a fixed sentence.
type fakeProvider struct {
	mu          sync.Mutex
	synthesized int
	heard       []Audio
	reply       Transcript
	speech      Audio
}

func (f *fakeProvider) Name() string                                     { return "fake" }
func (f *fakeProvider) CacheSettings(SynthesisRequest) map[string]string { return nil }
func (f *fakeProvider) Synthesize(context.Context, SynthesisRequest) (Audio, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.synthesized++
	return f.speech, nil
}
func (f *fakeProvider) Transcribe(_ context.Context, req TranscriptionRequest) (Transcript, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.heard = append(f.heard, req.Audio)
	return f.reply, nil
}

// fakeEngine returns an engine whose environment resolves to the fake
// provider through a real code path: a command name looked up by presets
// is not needed, so the hosted table is swapped for the test.
func fakeEngine(t *testing.T, p *fakeProvider) *Engine {
	t.Helper()
	saved := hostedProviders
	hostedProviders = []hosted{{name: "fake", key: "FAKE_KEY", make: func(string) Provider { return p },
		sayModel: "m1", sayVoice: "v1", hearModel: "h1"}}
	t.Cleanup(func() { hostedProviders = saved })
	env := map[string]string{"FAKE_KEY": "k"}
	return NewEngine(Env{Getenv: func(k string) string { return env[k] }, LookPath: func(string) (string, error) { return "", os.ErrNotExist }})
}

func paddedSpeech() Audio {
	mono := join(quiet(24000, 400*time.Millisecond), tone(24000, 300, -20, time.Second), quiet(24000, 2500*time.Millisecond))
	stereo := make([]int16, 0, len(mono)*2)
	for _, s := range mono {
		stereo = append(stereo, s, s)
	}
	return Audio{SampleRate: 24000, Channels: 2, Samples: stereo}
}

func writeWAV(t *testing.T, path string, samples []int16) {
	t.Helper()
	if err := os.WriteFile(path, Audio{SampleRate: Rate, Channels: 1, Samples: samples}.WAV(), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestSayRendersOnceThenReplaysTheSameBytes(t *testing.T) {
	p := &fakeProvider{speech: paddedSpeech()}
	e := fakeEngine(t, p)
	dir := t.TempDir()
	opts := SayOptions{Text: "Come here, Watson.", CacheDir: filepath.Join(dir, "cache")}
	first, err := e.Say(context.Background(), FileTarget{Path: filepath.Join(dir, "a.wav")}, opts)
	if err != nil {
		t.Fatal(err)
	}
	second, err := e.Say(context.Background(), FileTarget{Path: filepath.Join(dir, "b.wav")}, opts)
	if err != nil {
		t.Fatal(err)
	}
	if first.Cached || !second.Cached || p.synthesized != 1 {
		t.Fatalf("cached %v then %v after %d renders", first.Cached, second.Cached, p.synthesized)
	}
	if first.SHA256 != second.SHA256 {
		t.Fatal("a cached phrase did not replay the same audio")
	}
	if first.Provider != "fake" || first.Model != "m1" || first.Voice != "v1" || first.Path == "" {
		t.Fatalf("result %+v", first)
	}
	if first.DurationMs < 1090 || first.DurationMs > 1120 {
		t.Fatalf("clip is %.0f ms: padding was not trimmed to the second of speech", first.DurationMs)
	}
	a, _ := os.ReadFile(filepath.Join(dir, "a.wav"))
	b, _ := os.ReadFile(filepath.Join(dir, "b.wav"))
	if len(a) == 0 || string(a) != string(b) {
		t.Fatal("the two files differ")
	}

	opts.NoCache = true
	if again, err := e.Say(context.Background(), FileTarget{Path: filepath.Join(dir, "c.wav")}, opts); err != nil || again.Cached || p.synthesized != 2 {
		t.Fatalf("no-cache did not render again: %+v %v", again, err)
	}
	other := opts
	other.NoCache, other.Voice = false, "v2"
	if res, _ := e.Say(context.Background(), FileTarget{Path: filepath.Join(dir, "d.wav")}, other); res.Cached {
		t.Fatal("another voice replayed the first voice's audio")
	}
}

func TestListenToFile(t *testing.T) {
	p := &fakeProvider{reply: Transcript{Text: "Okay, your timer is set."}}
	e := fakeEngine(t, p)
	path := filepath.Join(t.TempDir(), "reply.wav")
	writeWAV(t, path, mix(join(quiet(Rate, 1500*time.Millisecond), tone(Rate, 300, -20, 2*time.Second), quiet(Rate, 3*time.Second)), hiss(-60, 6500*time.Millisecond, 4)))

	res, err := e.Listen(context.Background(), FileTarget{Path: path}, ListenOptions{Silence: -1, Path: filepath.Join(t.TempDir(), "heard.wav")})
	if err != nil {
		t.Fatal(err)
	}
	if res.Text != "Okay, your timer is set." || !res.Speech || res.StoppedBy != "silence" {
		t.Fatalf("%+v", res)
	}
	if res.OnsetMs == nil || *res.OnsetMs < 1480 || *res.OnsetMs > 1520 || *res.SpeechEndMs < 3470 || *res.SpeechEndMs > 3530 {
		t.Fatalf("onset %v end %v, want 1500 and 3500", *res.OnsetMs, *res.SpeechEndMs)
	}
	if res.LatencyMs != nil {
		t.Fatal("latency was reported without a preceding say")
	}
	if got := p.heard[0].Duration(); got < 2550*time.Millisecond || got > 2650*time.Millisecond {
		t.Fatalf("the provider got %v of audio, want the 2 s of speech plus 300 ms at each end", got)
	}
	if len(res.Segments) != 1 || res.Segments[0].StartMs != *res.OnsetMs || res.Segments[0].Text != res.Text {
		t.Fatalf("segments %+v", res.Segments)
	}
	if res.DurationMs < 4400 || res.DurationMs > 4700 {
		t.Fatalf("captured %.0f ms: the capture should stop one second after the speech", res.DurationMs)
	}
	if saved, err := os.ReadFile(res.Path); err != nil || len(saved) < 100000 {
		t.Fatalf("the captured audio was not saved: %v", err)
	}
}

func TestListenMapsProviderTimesToCaptureTime(t *testing.T) {
	p := &fakeProvider{reply: Transcript{Text: "hello there",
		Segments: []Segment{{Start: 300 * time.Millisecond, End: 900 * time.Millisecond, Text: "hello there"}},
		Words:    []Word{{Start: 300 * time.Millisecond, End: 500 * time.Millisecond, Word: "hello"}}}}
	e := fakeEngine(t, p)
	path := filepath.Join(t.TempDir(), "r.wav")
	writeWAV(t, path, join(quiet(Rate, 2*time.Second), tone(Rate, 300, -20, time.Second), quiet(Rate, 2*time.Second)))
	res, err := e.Listen(context.Background(), FileTarget{Path: path}, ListenOptions{Silence: -1})
	if err != nil {
		t.Fatal(err)
	}
	// The clip sent starts 300 ms before the onset at 2000 ms.
	if s := res.Segments[0]; s.StartMs < 1990 || s.StartMs > 2010 || s.EndMs < 2590 || s.EndMs > 2610 {
		t.Fatalf("segment %+v", s)
	}
	if w := res.Words[0]; w.StartMs < 1990 || w.StartMs > 2010 || w.Word != "hello" {
		t.Fatalf("word %+v", w)
	}
}

func TestListenStopRules(t *testing.T) {
	p := &fakeProvider{reply: Transcript{Text: "x"}}
	e := fakeEngine(t, p)
	dir := t.TempDir()
	speech := filepath.Join(dir, "speech.wav")
	writeWAV(t, speech, join(quiet(Rate, time.Second), tone(Rate, 300, -20, time.Second), quiet(Rate, 4*time.Second)))
	silent := filepath.Join(dir, "silent.wav")
	writeWAV(t, silent, hiss(-65, 12*time.Second, 5))
	ctx := context.Background()

	if res, err := e.Listen(ctx, FileTarget{Path: speech}, ListenOptions{Silence: 0}); err != nil || res.StoppedBy != "eof" || res.DurationMs != 6000 {
		t.Fatalf("with endpointing off the whole file should be read: %+v %v", res, err)
	}
	if res, err := e.Listen(ctx, FileTarget{Path: speech}, ListenOptions{Silence: 0, Duration: 2500 * time.Millisecond}); err != nil || res.StoppedBy != "duration" || res.DurationMs != 2500 {
		t.Fatalf("duration cap: %+v %v", res, err)
	}
	_, err := e.Listen(ctx, FileTarget{Path: silent}, ListenOptions{Silence: -1})
	if Code(err) != "no_speech" {
		t.Fatalf("silence should fail with no_speech, got %v", err)
	}
	res, err := e.Listen(ctx, FileTarget{Path: silent}, ListenOptions{Silence: -1, AllowSilence: true})
	if err != nil || res.Speech || res.Text != "" || res.StoppedBy != "timeout" || res.DurationMs < 9900 || res.DurationMs > 10200 {
		t.Fatalf("allow-silence: %+v %v", res, err)
	}
	if len(p.heard) != 2 {
		t.Fatalf("the provider was called %d times; silence must not be transcribed", len(p.heard))
	}
	if _, err := e.Listen(ctx, FileTarget{Path: filepath.Join(dir, "missing.wav")}, ListenOptions{}); Code(err) != "no_audio" {
		t.Fatalf("a missing file should be no_audio, got %v", err)
	}
}

// liveTarget is a device with its own clock: playing takes the clip's
// length, and capture delivers scripted audio stamped on that clock.
type liveTarget struct {
	clock   float64 // ms; advanced by Play
	script  []int16 // what the device plays, from clock 0
	played  []Audio
	never   bool // capture delivers nothing
	guard   time.Duration
	started chan struct{}
}

func (l *liveTarget) Name() string             { return "session" }
func (l *liveTarget) Key() string              { return "session:test" }
func (l *liveTarget) Live() bool               { return true }
func (l *liveTarget) EchoGuard() time.Duration { return l.guard }
func (l *liveTarget) Play(_ context.Context, a Audio) (float64, float64, error) {
	l.played = append(l.played, a)
	start := l.clock
	l.clock += a.DurationMs()
	return start, l.clock, nil
}
func (l *liveTarget) Capture(context.Context) (Capture, error) {
	if l.started != nil {
		close(l.started)
	}
	if l.never {
		return &blockedCapture{done: make(chan struct{})}, nil
	}
	return &sliceCapture{samples: l.script}, nil
}

type blockedCapture struct {
	done chan struct{}
	once sync.Once
}

func (b *blockedCapture) Next(ctx context.Context) (Chunk, error) {
	select {
	case <-b.done:
		return Chunk{}, io.EOF
	case <-ctx.Done():
		return Chunk{}, ctx.Err()
	}
}
func (b *blockedCapture) Close() { b.once.Do(func() { close(b.done) }) }

func TestListenAfterSayMeasuresLatencyOnTheTargetClock(t *testing.T) {
	p := &fakeProvider{speech: Audio{SampleRate: Rate, Channels: 1, Samples: tone(Rate, 300, -20, time.Second)}, reply: Transcript{Text: "It's sunny."}}
	e := fakeEngine(t, p)
	// The device stays quiet while it is spoken to (the 1.0 s clip with its
	// 50 ms at each end), thinks for 640 ms, then answers for 1.5 s.
	dev := &liveTarget{script: join(quiet(Rate, 1740*time.Millisecond), tone(Rate, 400, -18, 1500*time.Millisecond), quiet(Rate, 3*time.Second))}
	ctx := context.Background()
	said, err := e.Say(ctx, dev, SayOptions{Text: "What's the weather?", CacheDir: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	if said.DurationMs != 1100 || len(dev.played) != 1 {
		t.Fatalf("say %+v", said)
	}
	res, err := e.Listen(ctx, dev, ListenOptions{Silence: -1})
	if err != nil {
		t.Fatal(err)
	}
	if res.LatencyMs == nil || *res.LatencyMs < 620 || *res.LatencyMs > 660 {
		t.Fatalf("latency %v, want 640 ms from the end of the say to the first sound of the answer", res.LatencyMs)
	}
	if *res.OnsetMs != *res.LatencyMs || res.Text != "It's sunny." || res.StoppedBy != "silence" {
		t.Fatalf("%+v", res)
	}
}

func TestEchoGuardKeepsTheQuestionOutOfTheAnswer(t *testing.T) {
	p := &fakeProvider{speech: Audio{SampleRate: Rate, Channels: 1, Samples: tone(Rate, 300, -20, time.Second)}, reply: Transcript{Text: "ok"}}
	e := fakeEngine(t, p)
	// A room: the question's tail rings for 120 ms after playback ends.
	dev := &liveTarget{guard: 200 * time.Millisecond,
		script: join(tone(Rate, 300, -20, 1220*time.Millisecond), quiet(Rate, 880*time.Millisecond), tone(Rate, 400, -18, time.Second), quiet(Rate, 2*time.Second))}
	if _, err := e.Say(context.Background(), dev, SayOptions{Text: "hello", CacheDir: t.TempDir()}); err != nil {
		t.Fatal(err)
	}
	res, err := e.Listen(context.Background(), dev, ListenOptions{Silence: -1})
	if err != nil {
		t.Fatal(err)
	}
	if *res.LatencyMs < 980 || *res.LatencyMs > 1020 {
		t.Fatalf("latency %.0f ms: the echo of the question was taken for the reply", *res.LatencyMs)
	}
}

func TestListenReportsABrokenPipeAsNoAudio(t *testing.T) {
	saved := noAudioAfter
	noAudioAfter = 150 * time.Millisecond
	defer func() { noAudioAfter = saved }()
	e := fakeEngine(t, &fakeProvider{})
	_, err := e.Listen(context.Background(), &liveTarget{never: true}, ListenOptions{})
	if Code(err) != "no_audio" {
		t.Fatalf("got %v, want no_audio", err)
	}
}

func TestOneCallPerTargetAtATime(t *testing.T) {
	saved := noAudioAfter
	noAudioAfter = 2 * time.Second
	defer func() { noAudioAfter = saved }()
	e := fakeEngine(t, &fakeProvider{})
	dev := &liveTarget{never: true, started: make(chan struct{})}
	done := make(chan error, 1)
	go func() {
		_, err := e.Listen(context.Background(), dev, ListenOptions{})
		done <- err
	}()
	<-dev.started
	_, err := e.Listen(context.Background(), &liveTarget{never: true}, ListenOptions{})
	if Code(err) != "busy" {
		t.Fatalf("a second listen on a busy target got %v, want busy", err)
	}
	if first := <-done; Code(first) != "no_audio" {
		t.Fatalf("first listen: %v", first)
	}
}

func TestSayNeedsTextAndAProvider(t *testing.T) {
	e := fakeEngine(t, &fakeProvider{})
	if _, err := e.Say(context.Background(), FileTarget{Path: filepath.Join(t.TempDir(), "x.wav")}, SayOptions{Text: "  "}); err == nil {
		t.Fatal("empty text was accepted")
	}
	bare := NewEngine(Env{Getenv: func(string) string { return "" }, LookPath: func(string) (string, error) { return "", errors.New("none") }})
	if _, err := bare.Say(context.Background(), FileTarget{Path: "x.wav"}, SayOptions{Text: "hi"}); Code(err) != "not_configured" {
		t.Fatalf("got %v, want not_configured", err)
	}
}

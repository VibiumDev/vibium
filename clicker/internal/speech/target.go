package speech

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

// Chunk is captured 16 kHz mono audio. Time is when its first sample was
// captured, in milliseconds on the target's clock.
type Chunk struct {
	Samples []int16
	Time    float64
	// Lost marks silence that stands in for audio the target dropped on
	// its way here.
	Lost bool
}

// Capture is a running capture. Next returns io.EOF when a source that
// ends, such as a file, has run out.
type Capture interface {
	Next(ctx context.Context) (Chunk, error)
	Close()
}

// Target is where audio goes or comes from: a session's device, this
// machine's speakers and microphone, or a WAV file.
type Target interface {
	// Name is the target as results show it.
	Name() string
	// Key identifies the target for the capture window and the busy rule.
	Key() string
	// Play plays a clip and waits for it to end. It returns when the first
	// and last sample played, in milliseconds on the target's clock.
	Play(ctx context.Context, a Audio) (start, end float64, err error)
	Capture(ctx context.Context) (Capture, error)
	// Live is false for a target read as fast as possible, in audio time.
	Live() bool
	// EchoGuard is how long after playback the target still hears the
	// clip itself, as a room does.
	EchoGuard() time.Duration
	// Now is the present on the target's clock, in milliseconds: the clock
	// Play's result and every captured chunk are stamped with.
	Now() float64
}

// chunkSamples is 100 ms of audio.
const chunkSamples = Rate / 10

// FileTarget writes say's audio to a WAV file and reads listen's from one.
type FileTarget struct{ Path string }

// IsWAVPath reports whether a target names a WAV file.
func IsWAVPath(target string) bool { return strings.EqualFold(filepath.Ext(target), ".wav") }

func (f FileTarget) Name() string             { return f.Path }
func (f FileTarget) Key() string              { return "file:" + f.Path }
func (f FileTarget) Live() bool               { return false }
func (f FileTarget) EchoGuard() time.Duration { return 0 }
func (f FileTarget) Now() float64             { return 0 }

func (f FileTarget) Play(_ context.Context, a Audio) (float64, float64, error) {
	if err := os.MkdirAll(filepath.Dir(f.Path), 0o755); err != nil {
		return 0, 0, errf("no_audio", "cannot write %s: %v", f.Path, err)
	}
	if err := writeAtomic(f.Path, a.WAV()); err != nil {
		return 0, 0, errf("no_audio", "cannot write %s: %v", f.Path, err)
	}
	return 0, a.DurationMs(), nil
}

func (f FileTarget) Capture(context.Context) (Capture, error) {
	raw, err := os.ReadFile(f.Path)
	if err != nil {
		return nil, errf("no_audio", "cannot read %s: %v", f.Path, err)
	}
	audio, err := ParseWAV(raw)
	if err != nil {
		return nil, errf("no_audio", "%s is not a PCM WAV file", f.Path)
	}
	return &sliceCapture{samples: audio.Resample(Rate).Samples}, nil
}

// sliceCapture hands out audio it already holds.
type sliceCapture struct {
	samples []int16
	pos     int
}

func (s *sliceCapture) Next(context.Context) (Chunk, error) {
	if s.pos >= len(s.samples) {
		return Chunk{}, io.EOF
	}
	end := min(s.pos+chunkSamples, len(s.samples))
	c := Chunk{Samples: s.samples[s.pos:end], Time: float64(s.pos) * 1000 / Rate}
	s.pos = end
	return c, nil
}

func (s *sliceCapture) Close() {}

// LocalTarget plays on this machine's default speakers and captures from
// its default microphone by running a system program. Vibium links no
// audio library.
type LocalTarget struct {
	play    []string // receives a WAV on stdin, or at {input}
	capture []string // writes raw 16 kHz mono 16-bit PCM to stdout
	missing string   // what to install when a program is absent
}

// NewLocalTarget picks the system programs for this operating system.
func NewLocalTarget() *LocalTarget {
	switch runtime.GOOS {
	case "darwin":
		return &LocalTarget{play: []string{"afplay", "{input}"},
			capture: []string{"sox", "-q", "-d", "-t", "raw", "-r", "16000", "-e", "signed", "-b", "16", "-c", "1", "-"},
			missing: "afplay ships with macOS; capture needs sox (brew install sox)"}
	case "windows":
		return &LocalTarget{play: []string{"powershell", "-NoProfile", "-Command", "(New-Object Media.SoundPlayer '{input}').PlaySync()"},
			missing: "capture from the microphone is not available on Windows yet"}
	default:
		return &LocalTarget{play: []string{"aplay", "-q", "-"},
			capture: []string{"arecord", "-q", "-t", "raw", "-f", "S16_LE", "-r", "16000", "-c", "1", "-"},
			missing: "install alsa-utils for aplay and arecord"}
	}
}

func (l *LocalTarget) Name() string             { return "local" }
func (l *LocalTarget) Key() string              { return "local" }
func (l *LocalTarget) Live() bool               { return true }
func (l *LocalTarget) EchoGuard() time.Duration { return 200 * time.Millisecond }
func (l *LocalTarget) Now() float64             { return localClock() }

var clockStart = time.Now()

// localClock is this process's monotonic clock in milliseconds.
func localClock() float64 { return float64(time.Since(clockStart)) / float64(time.Millisecond) }

func (l *LocalTarget) Play(ctx context.Context, a Audio) (float64, float64, error) {
	if len(l.play) == 0 {
		return 0, 0, errf("no_audio", "this machine cannot play audio: %s", l.missing)
	}
	argv := append([]string(nil), l.play...)
	var stdin io.Reader = bytes.NewReader(a.WAV())
	for i, arg := range argv {
		if strings.Contains(arg, "{input}") {
			f, err := os.CreateTemp("", "vibium-say-*.wav")
			if err != nil {
				return 0, 0, err
			}
			defer os.Remove(f.Name())
			f.Write(a.WAV())
			f.Close()
			argv[i] = strings.ReplaceAll(arg, "{input}", f.Name())
			stdin = nil
		}
	}
	cmd := exec.CommandContext(ctx, argv[0], argv[1:]...)
	cmd.Stdin = stdin
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	start := localClock()
	if err := cmd.Run(); err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return 0, 0, errf("no_audio", "%s is not installed: %s", argv[0], l.missing)
		}
		return 0, 0, errf("no_audio", "%s could not play: %s", argv[0], lastLine(stderr.String(), err))
	}
	return start, localClock(), nil
}

func (l *LocalTarget) Capture(ctx context.Context) (Capture, error) {
	if len(l.capture) == 0 {
		return nil, errf("no_audio", "this machine cannot capture audio: %s", l.missing)
	}
	ctx, cancel := context.WithCancel(ctx)
	cmd := exec.CommandContext(ctx, l.capture[0], l.capture[1:]...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		cancel()
		return nil, err
	}
	stderr := &bytes.Buffer{}
	cmd.Stderr = stderr
	// A capture program's children can outlive it and hold its pipes open.
	cmd.WaitDelay = 500 * time.Millisecond
	if err := cmd.Start(); err != nil {
		cancel()
		if errors.Is(err, exec.ErrNotFound) {
			return nil, errf("no_audio", "%s is not installed: %s", l.capture[0], l.missing)
		}
		return nil, errf("no_audio", "%s could not start: %v", l.capture[0], err)
	}
	return &processCapture{cmd: cmd, out: stdout, stderr: stderr, cancel: cancel, name: l.capture[0]}, nil
}

// processCapture reads raw PCM from a capture program's stdout.
type processCapture struct {
	cmd    *exec.Cmd
	out    io.Reader
	stderr *bytes.Buffer
	cancel context.CancelFunc
	name   string
	once   sync.Once
}

func (p *processCapture) Next(context.Context) (Chunk, error) {
	buf := make([]byte, chunkSamples*2)
	n, err := io.ReadFull(p.out, buf)
	if n < 2 {
		if err == nil || err == io.EOF || err == io.ErrUnexpectedEOF {
			return Chunk{}, errf("no_audio", "%s stopped: %s", p.name, lastLine(p.stderr.String(), err))
		}
		return Chunk{}, errf("no_audio", "%s: %v", p.name, err)
	}
	a := FromBytes(buf[:n-n%2], Rate, 1)
	return Chunk{Samples: a.Samples, Time: localClock() - float64(len(a.Samples))*1000/Rate}, nil
}

func (p *processCapture) Close() {
	p.once.Do(func() {
		p.cancel()
		p.cmd.Wait()
	})
}

func lastLine(stderr string, err error) string {
	lines := strings.Split(strings.TrimSpace(stderr), "\n")
	if last := strings.TrimSpace(lines[len(lines)-1]); last != "" {
		return last
	}
	if err != nil {
		return err.Error()
	}
	return "no output"
}

// pump keeps reading a capture so a program's pipe never fills while
// nobody is listening. It holds at most keep of audio past its mark.
type pump struct {
	source Capture
	cancel context.CancelFunc

	mu     sync.Mutex
	wake   chan struct{}
	chunks []Chunk
	err    error
	until  float64 // stop reading once a chunk starts after this; 0 is never
}

func startPump(source Capture) *pump {
	ctx, cancel := context.WithCancel(context.Background())
	p := &pump{source: source, cancel: cancel, wake: make(chan struct{}, 1)}
	go func() {
		for {
			c, err := p.source.Next(ctx)
			p.mu.Lock()
			if err == nil && p.until > 0 && c.Time > p.until {
				err = io.EOF
			}
			if err != nil {
				p.err = err
			} else {
				p.chunks = append(p.chunks, c)
			}
			p.mu.Unlock()
			select {
			case p.wake <- struct{}{}:
			default:
			}
			if err != nil {
				p.source.Close()
				return
			}
		}
	}()
	return p
}

// Limit stops the pump once audio passes the given target time. Reading
// resumes nothing: a window that nobody listened to simply ends.
func (p *pump) Limit(until float64) {
	p.mu.Lock()
	p.until = until
	p.mu.Unlock()
}

// Unlimit lets a listener read past the window's limit.
func (p *pump) Unlimit() { p.Limit(0) }

func (p *pump) Next(ctx context.Context) (Chunk, error) {
	for {
		p.mu.Lock()
		if len(p.chunks) > 0 {
			c := p.chunks[0]
			p.chunks = p.chunks[1:]
			p.mu.Unlock()
			return c, nil
		}
		err := p.err
		p.mu.Unlock()
		if err != nil {
			return Chunk{}, err
		}
		select {
		case <-p.wake:
		case <-ctx.Done():
			return Chunk{}, ctx.Err()
		}
	}
}

func (p *pump) Close() {
	p.cancel()
	p.source.Close()
}

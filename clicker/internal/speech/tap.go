package speech

import (
	"context"
	"io"
	"sync"
	"time"
)

// Tap shares one capture among several readers. A device gives one audio
// stream, and during a recording two things want it: the recording's
// continuous track and whichever listen is running. The capture opens with
// the first reader and closes with the last.
type Tap struct {
	open func(context.Context) (Capture, error)

	mu      sync.Mutex
	source  Capture
	cancel  context.CancelFunc
	readers map[*tapReader]bool
}

// NewTap wraps a function that opens the underlying capture.
func NewTap(open func(context.Context) (Capture, error)) *Tap {
	return &Tap{open: open, readers: map[*tapReader]bool{}}
}

// tapBuffer is how many chunks a reader may fall behind: 30 s of 100 ms
// chunks. A reader further behind loses the oldest audio.
const tapBuffer = 300

// Capture returns a reader that sees audio from now on.
func (t *Tap) Capture(context.Context) (Capture, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.source == nil {
		ctx, cancel := context.WithCancel(context.Background())
		source, err := t.open(ctx)
		if err != nil {
			cancel()
			return nil, err
		}
		t.source, t.cancel = source, cancel
		go t.pump(ctx, source)
	}
	r := &tapReader{tap: t, chunks: make(chan Chunk, tapBuffer), done: make(chan struct{})}
	t.readers[r] = true
	return r, nil
}

func (t *Tap) pump(ctx context.Context, source Capture) {
	for {
		chunk, err := source.Next(ctx)
		t.mu.Lock()
		if t.source != source {
			t.mu.Unlock()
			return
		}
		if err != nil {
			// The source ended or failed: every reader sees it, and the
			// next Capture opens a fresh one.
			for r := range t.readers {
				r.fail(err)
			}
			t.readers = map[*tapReader]bool{}
			t.source, t.cancel = nil, nil
			t.mu.Unlock()
			source.Close()
			return
		}
		for r := range t.readers {
			select {
			case r.chunks <- chunk:
			default:
				select {
				case <-r.chunks:
				default:
				}
				select {
				case r.chunks <- chunk:
				default:
				}
			}
		}
		t.mu.Unlock()
	}
}

func (t *Tap) drop(r *tapReader) {
	t.mu.Lock()
	delete(t.readers, r)
	var source Capture
	if len(t.readers) == 0 && t.source != nil {
		source = t.source
		t.cancel()
		t.source, t.cancel = nil, nil
	}
	t.mu.Unlock()
	if source != nil {
		source.Close()
	}
}

type tapReader struct {
	tap    *Tap
	chunks chan Chunk
	done   chan struct{}
	once   sync.Once
	mu     sync.Mutex
	err    error
}

func (r *tapReader) fail(err error) {
	r.mu.Lock()
	r.err = err
	r.mu.Unlock()
	r.once.Do(func() { close(r.done) })
}

func (r *tapReader) Next(ctx context.Context) (Chunk, error) {
	select {
	case chunk := <-r.chunks:
		return chunk, nil
	default:
	}
	select {
	case chunk := <-r.chunks:
		return chunk, nil
	case <-r.done:
		// Audio that arrived before the end is still read.
		select {
		case chunk := <-r.chunks:
			return chunk, nil
		default:
		}
		r.mu.Lock()
		err := r.err
		r.mu.Unlock()
		if err == nil {
			err = io.EOF
		}
		return Chunk{}, err
	case <-ctx.Done():
		return Chunk{}, ctx.Err()
	}
}

func (r *tapReader) Close() {
	r.once.Do(func() { close(r.done) })
	r.tap.drop(r)
}

// Stretch is a stretch of speech in a track, from the track's first sample.
type Stretch struct{ Start, End time.Duration }

// SpeechStretches finds every stretch of speech in 16 kHz mono audio. A
// pause shorter than silence stays inside its stretch.
func SpeechStretches(samples []int16, silence time.Duration) []Stretch {
	var out []Stretch
	offset := 0
	for offset < len(samples) {
		d := NewDetector(silence)
		d.Write(samples[offset:])
		d.Flush()
		onset, ok := d.Onset()
		if !ok {
			break
		}
		end := d.LastSpeech()
		if end <= onset {
			end = d.Elapsed()
		}
		base := time.Duration(offset) * time.Second / Rate
		out = append(out, Stretch{Start: base + onset, End: base + end})
		next := offset + samplesAt(end) + vadHop
		if _, ended := d.End(); !ended || next <= offset {
			break
		}
		offset = next
	}
	return out
}

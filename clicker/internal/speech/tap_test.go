package speech

import (
	"context"
	"errors"
	"io"
	"sync"
	"testing"
	"time"
)

// feedCapture is a capture a test feeds by hand.
type feedCapture struct {
	chunks chan Chunk
	closed chan struct{}
	once   sync.Once
}

func newFeed() *feedCapture {
	return &feedCapture{chunks: make(chan Chunk, 64), closed: make(chan struct{})}
}
func (f *feedCapture) Next(ctx context.Context) (Chunk, error) {
	select {
	case c, ok := <-f.chunks:
		if !ok {
			return Chunk{}, io.EOF
		}
		return c, nil
	case <-f.closed:
		return Chunk{}, io.EOF
	case <-ctx.Done():
		return Chunk{}, ctx.Err()
	}
}
func (f *feedCapture) Close() { f.once.Do(func() { close(f.closed) }) }

func TestTapSharesOneCapture(t *testing.T) {
	var feeds []*feedCapture
	tap := NewTap(func(context.Context) (Capture, error) {
		f := newFeed()
		feeds = append(feeds, f)
		return f, nil
	})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	recorder, _ := tap.Capture(ctx)
	feeds[0].chunks <- Chunk{Time: 100, Samples: []int16{1}}
	if c, err := recorder.Next(ctx); err != nil || c.Time != 100 {
		t.Fatalf("%v %v", c, err)
	}
	// A listener that joins later hears audio from then on, and the
	// recorder keeps hearing everything.
	listener, _ := tap.Capture(ctx)
	feeds[0].chunks <- Chunk{Time: 200, Samples: []int16{2}}
	a, _ := recorder.Next(ctx)
	b, _ := listener.Next(ctx)
	if a.Time != 200 || b.Time != 200 || len(feeds) != 1 {
		t.Fatalf("recorder %v, listener %v, %d captures opened", a.Time, b.Time, len(feeds))
	}
	// The listener leaving does not end the capture for the recorder.
	listener.Close()
	feeds[0].chunks <- Chunk{Time: 300}
	if c, err := recorder.Next(ctx); err != nil || c.Time != 300 {
		t.Fatalf("after the listener left: %v %v", c, err)
	}
	select {
	case <-feeds[0].closed:
		t.Fatal("the capture closed while a reader remained")
	default:
	}
	// The last reader leaving closes it, and the next reader opens a new one.
	recorder.Close()
	select {
	case <-feeds[0].closed:
	case <-time.After(2 * time.Second):
		t.Fatal("the capture stayed open with no readers")
	}
	again, _ := tap.Capture(ctx)
	defer again.Close()
	if len(feeds) != 2 {
		t.Fatalf("%d captures opened, want a second one", len(feeds))
	}
}

func TestTapPassesOnTheEndAndFailures(t *testing.T) {
	feed := newFeed()
	tap := NewTap(func(context.Context) (Capture, error) { return feed, nil })
	ctx := context.Background()
	r, _ := tap.Capture(ctx)
	feed.chunks <- Chunk{Time: 1}
	close(feed.chunks)
	if c, err := r.Next(ctx); err != nil || c.Time != 1 {
		t.Fatalf("audio before the end must still be read: %v %v", c, err)
	}
	if _, err := r.Next(ctx); err != io.EOF {
		t.Fatalf("want EOF, got %v", err)
	}
	failing := NewTap(func(context.Context) (Capture, error) { return nil, errors.New("no audio device") })
	if _, err := failing.Capture(ctx); err == nil {
		t.Fatal("an open failure was swallowed")
	}
}

func TestSpeechStretchesFindsEachUtterance(t *testing.T) {
	word := tone(Rate, 300, -20, 800*time.Millisecond)
	track := mix(join(quiet(Rate, time.Second), word, quiet(Rate, 300*time.Millisecond), word, quiet(Rate, 3*time.Second), word, quiet(Rate, 2*time.Second)), hiss(-62, 10*time.Second, 7))
	got := SpeechStretches(track, 700*time.Millisecond)
	if len(got) != 2 {
		t.Fatalf("%d stretches %v: a 300 ms pause joins two words, a 3 s pause separates them", len(got), got)
	}
	if !near(got[0].Start, time.Second, 30*time.Millisecond) || !near(got[0].End, 2900*time.Millisecond, 40*time.Millisecond) {
		t.Fatalf("first stretch %v", got[0])
	}
	if !near(got[1].Start, 5900*time.Millisecond, 40*time.Millisecond) || !near(got[1].End, 6700*time.Millisecond, 40*time.Millisecond) {
		t.Fatalf("second stretch %v", got[1])
	}
	if len(SpeechStretches(hiss(-62, 3*time.Second, 8), time.Second)) != 0 {
		t.Fatal("speech found in noise")
	}
}

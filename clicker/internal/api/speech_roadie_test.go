package api

import (
	"context"
	"testing"

	"github.com/vibium/clicker/internal/speech"
)

func TestRoadieCaptureFillsGapsWithSilence(t *testing.T) {
	c := &roadieCapture{s: NewSpeech(), chunks: make(chan speech.Chunk, 8), done: make(chan struct{})}
	c.deliver(speech.Chunk{Samples: make([]int16, 1600), Time: 1000})
	c.deliver(speech.Chunk{Samples: make([]int16, 1600), Time: 1300}) // 200 ms missing
	ctx := context.Background()
	first, _ := c.Next(ctx)
	gap, _ := c.Next(ctx)
	second, _ := c.Next(ctx)
	if first.Time != 1000 || gap.Time != 1100 || len(gap.Samples) != 3200 || second.Time != 1300 {
		t.Fatalf("times %.0f %.0f %.0f, gap of %d samples", first.Time, gap.Time, second.Time, len(gap.Samples))
	}
}

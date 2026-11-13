package speech

import (
	"math"
	"sort"
	"time"
)

// Detector finds where speech starts and ends in 16 kHz mono audio. It is
// an adaptive energy detector with no model, so every provider gets the
// same onset.
type Detector struct {
	silence time.Duration

	// High-pass filter state.
	prevIn, prevOut float64
	coeff           float64

	pending []float64 // filtered samples not yet framed
	levels  []float64 // one level per hop, from the start

	floor     float64
	haveFloor bool
	next      int // first frame not yet judged

	speaking bool
	quiet    int
	onset    int // frame index, valid when started
	end      int // frame index, valid when ended
	started  bool
	ended    bool
	skip     int // frames to ignore at the start
}

const (
	vadFrame    = Rate / 50  // 20 ms
	vadHop      = Rate / 100 // 10 ms
	vadWarm     = 30         // frames that seed the floor
	vadOnDB     = 12.0
	vadOffDB    = 8.0
	vadMinOnDB  = -50.0
	vadFloorMin = -75.0
	vadFloorMax = -45.0
	vadRun      = 3
	vadWindow   = 5
)

// NewDetector returns a detector that ends speech after silence without it.
// A zero silence never ends speech.
func NewDetector(silence time.Duration) *Detector {
	return &Detector{silence: silence, coeff: math.Exp(-2 * math.Pi * 100 / Rate)}
}

// Skip makes the detector ignore the first d of audio. Time still counts
// from the first sample.
func (d *Detector) Skip(t time.Duration) { d.skip = int(t / (10 * time.Millisecond)) }

// Write feeds samples in order.
func (d *Detector) Write(samples []int16) {
	for _, s := range samples {
		x := float64(s) / 32768
		y := d.coeff * (d.prevOut + x - d.prevIn)
		d.prevIn, d.prevOut = x, y
		d.pending = append(d.pending, y)
	}
	for len(d.pending) >= vadFrame {
		var sum float64
		for _, v := range d.pending[:vadFrame] {
			sum += v * v
		}
		d.levels = append(d.levels, 10*math.Log10(sum/vadFrame+1e-12))
		d.pending = d.pending[vadHop:]
	}
	d.judge(false)
}

// Flush judges audio shorter than the warm-up, at the end of a source.
func (d *Detector) Flush() { d.judge(true) }

func (d *Detector) judge(final bool) {
	if !d.haveFloor {
		if len(d.levels) < vadWarm && !final {
			return
		}
		d.floor = -60
		if len(d.levels) >= vadWarm {
			warm := append([]float64(nil), d.levels[:vadWarm]...)
			sort.Float64s(warm)
			d.floor = clamp((warm[vadWarm/2-1]+warm[vadWarm/2])/2, vadFloorMin, vadFloorMax)
		}
		d.haveFloor = true
	}
	for ; d.next < len(d.levels) && !d.ended; d.next++ {
		i := d.next
		if i < d.skip {
			continue
		}
		on := math.Max(d.floor+vadOnDB, vadMinOnDB)
		off := math.Max(d.floor+vadOffDB, vadMinOnDB-(vadOnDB-vadOffDB))
		level := d.levels[i]
		if !d.speaking {
			lo := i - vadWindow + 1
			if lo < d.skip {
				lo = d.skip
			}
			if lo < 0 {
				lo = 0
			}
			above, first := 0, -1
			for k := lo; k <= i; k++ {
				if d.levels[k] > on {
					above++
					if first < 0 {
						first = k
					}
				}
			}
			if above >= vadRun {
				d.speaking, d.started, d.onset, d.quiet = true, true, first, 0
			} else if level < on {
				d.floor = 0.98*d.floor + 0.02*clamp(level, vadFloorMin, vadFloorMax)
			}
			continue
		}
		if level < off {
			d.quiet++
			if d.silence > 0 && time.Duration(d.quiet)*10*time.Millisecond >= d.silence {
				d.end, d.ended = i-d.quiet+1, true
			}
		} else {
			d.quiet = 0
		}
	}
}

func clamp(v, lo, hi float64) float64 { return math.Max(lo, math.Min(hi, v)) }

func frameTime(frame int) time.Duration { return time.Duration(frame) * 10 * time.Millisecond }

// Onset is when speech started, counted from the first sample.
func (d *Detector) Onset() (time.Duration, bool) { return frameTime(d.onset), d.started }

// End is when speech ended, once silence has followed it.
func (d *Detector) End() (time.Duration, bool) { return frameTime(d.end), d.ended }

// LastSpeech is where speech last stopped, for audio that ran out before
// silence was long enough to end it.
func (d *Detector) LastSpeech() time.Duration {
	if d.ended {
		return frameTime(d.end)
	}
	return frameTime(len(d.levels) - d.quiet)
}

// Elapsed is how much audio has been judged.
func (d *Detector) Elapsed() time.Duration { return frameTime(len(d.levels)) }

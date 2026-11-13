package speech

import (
	"math/rand"
	"testing"
	"time"
)

// hiss is a noise floor at a level in dBFS, like an idle line.
func hiss(db float64, d time.Duration, seed int64) []int16 {
	r := rand.New(rand.NewSource(seed))
	n := int(d.Seconds() * Rate)
	amp := 32767.0
	for x := db; x < 0; x += 20 {
		amp /= 10
	}
	out := make([]int16, n)
	for i := range out {
		out[i] = clip16(r.NormFloat64() * amp)
	}
	return out
}

func mix(a, b []int16) []int16 {
	out := make([]int16, len(a))
	for i := range a {
		v := int(a[i])
		if i < len(b) {
			v += int(b[i])
		}
		out[i] = clip16(float64(v))
	}
	return out
}

func near(got, want, tolerance time.Duration) bool {
	d := got - want
	return d >= -tolerance && d <= tolerance
}

func TestDetectorFindsOnsetAndEnd(t *testing.T) {
	sig := join(quiet(Rate, 1500*time.Millisecond), tone(Rate, 300, -20, 2*time.Second), quiet(Rate, 1500*time.Millisecond))
	sig = mix(sig, hiss(-60, 5*time.Second, 1))
	d := NewDetector(time.Second)
	for i := 0; i < len(sig); i += 1600 { // 100 ms chunks, as a live source sends
		d.Write(sig[i:min(i+1600, len(sig))])
	}
	onset, ok := d.Onset()
	if !ok || !near(onset, 1500*time.Millisecond, 20*time.Millisecond) {
		t.Fatalf("onset %v %v, want 1.5 s within one hop", onset, ok)
	}
	end, ok := d.End()
	if !ok || !near(end, 3500*time.Millisecond, 30*time.Millisecond) {
		t.Fatalf("end %v %v, want 3.5 s", end, ok)
	}
}

func TestDetectorIgnoresClicksAndHiss(t *testing.T) {
	sig := hiss(-60, 3*time.Second, 2)
	for i := 16000; i < 16032; i++ { // a 2 ms click
		sig[i] = 20000
	}
	d := NewDetector(time.Second)
	d.Write(sig)
	d.Flush()
	if at, ok := d.Onset(); ok {
		t.Fatalf("a click or the noise floor counted as speech at %v", at)
	}
}

func TestDetectorBridgesShortPauses(t *testing.T) {
	word := tone(Rate, 300, -20, 600*time.Millisecond)
	sig := join(quiet(Rate, 500*time.Millisecond), word, quiet(Rate, 400*time.Millisecond), word, quiet(Rate, 1500*time.Millisecond))
	d := NewDetector(time.Second)
	d.Write(mix(sig, hiss(-62, 4*time.Second, 3)))
	end, ok := d.End()
	if !ok || !near(end, 2100*time.Millisecond, 30*time.Millisecond) {
		t.Fatalf("end %v %v: a 400 ms pause should not end a reply", end, ok)
	}
}

func TestDetectorNeverEndsWithoutSilenceRule(t *testing.T) {
	sig := join(quiet(Rate, 500*time.Millisecond), tone(Rate, 300, -20, time.Second), quiet(Rate, 3*time.Second))
	d := NewDetector(0)
	d.Write(sig)
	if _, ok := d.End(); ok {
		t.Fatal("endpointing was off but speech ended")
	}
	if at := d.LastSpeech(); !near(at, 1500*time.Millisecond, 30*time.Millisecond) {
		t.Fatalf("last speech at %v", at)
	}
}

func TestDetectorSkipIgnoresTheStart(t *testing.T) {
	sig := join(tone(Rate, 300, -20, 150*time.Millisecond), quiet(Rate, 850*time.Millisecond), tone(Rate, 300, -20, time.Second), quiet(Rate, 1200*time.Millisecond))
	d := NewDetector(time.Second)
	d.Skip(200 * time.Millisecond)
	d.Write(sig)
	onset, ok := d.Onset()
	if !ok || !near(onset, time.Second, 20*time.Millisecond) {
		t.Fatalf("onset %v %v: the echo in the first 200 ms should not count", onset, ok)
	}
}

func TestDetectorShortAudio(t *testing.T) {
	d := NewDetector(time.Second)
	d.Write(tone(Rate, 300, -20, 200*time.Millisecond))
	d.Flush()
	if _, ok := d.Onset(); !ok {
		t.Fatal("200 ms of speech with no lead-in went unheard")
	}
}

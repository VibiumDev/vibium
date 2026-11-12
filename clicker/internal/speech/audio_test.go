package speech

import (
	"math"
	"testing"
	"time"
)

// tone is a sine burst at a level in dBFS, for audio that stands in for speech.
func tone(rate int, hz float64, db float64, d time.Duration) []int16 {
	n := int(d.Seconds() * float64(rate))
	amp := math.Pow(10, db/20) * 32767 * math.Sqrt2
	out := make([]int16, n)
	for i := range out {
		out[i] = clip16(amp * math.Sin(2*math.Pi*hz*float64(i)/float64(rate)))
	}
	return out
}

func quiet(rate int, d time.Duration) []int16 { return make([]int16, int(d.Seconds()*float64(rate))) }

func join(parts ...[]int16) []int16 {
	var out []int16
	for _, p := range parts {
		out = append(out, p...)
	}
	return out
}

func TestWAVRoundTrip(t *testing.T) {
	a := Audio{SampleRate: 22050, Channels: 2, Samples: []int16{0, 1, -1, 32767, -32768, 12345}}
	got, err := ParseWAV(a.WAV())
	if err != nil {
		t.Fatal(err)
	}
	if got.SampleRate != 22050 || got.Channels != 2 || len(got.Samples) != 6 || got.Samples[3] != 32767 || got.Samples[4] != -32768 {
		t.Fatalf("round trip changed the audio: %+v", got)
	}
	if got.SHA256() != a.SHA256() {
		t.Fatal("the hash changed across a round trip")
	}
}

func TestParseWAVStreamingHeader(t *testing.T) {
	raw := Audio{SampleRate: 24000, Channels: 1, Samples: []int16{5, 6, 7}}.WAV()
	for i := 40; i < 44; i++ { // a streamed file declares no data length
		raw[i] = 0xFF
	}
	got, err := ParseWAV(raw)
	if err != nil || len(got.Samples) != 3 || got.Samples[2] != 7 {
		t.Fatalf("streamed WAV: %v %+v", err, got)
	}
}

func TestParseWAVRejectsOtherBytes(t *testing.T) {
	if _, err := ParseWAV([]byte("ID3 not audio at all")); err == nil {
		t.Fatal("non-WAV bytes were accepted")
	}
}

func TestResampleKeepsLengthLevelAndPitch(t *testing.T) {
	in := Audio{SampleRate: 24000, Channels: 1, Samples: tone(24000, 440, -12, time.Second)}
	out := in.Resample(16000)
	if out.SampleRate != 16000 || len(out.Samples) != 16000 {
		t.Fatalf("got %d samples at %d Hz", len(out.Samples), out.SampleRate)
	}
	if db := levelDB(out.Samples[1600:14400]); math.Abs(db-(-12)) > 0.3 {
		t.Fatalf("level moved to %.2f dBFS", db)
	}
	crossings := 0
	for i := 1601; i < 14400; i++ {
		if (out.Samples[i-1] < 0) != (out.Samples[i] < 0) {
			crossings++
		}
	}
	if hz := float64(crossings) / 2 / 0.8; math.Abs(hz-440) > 3 {
		t.Fatalf("pitch moved to %.1f Hz", hz)
	}
}

func TestResampleRemovesWhatTheNewRateCannotCarry(t *testing.T) {
	in := Audio{SampleRate: 48000, Channels: 1, Samples: tone(48000, 11000, -6, 500*time.Millisecond)}
	out := in.Resample(16000)
	if db := levelDB(out.Samples[800:7200]); db > -50 {
		t.Fatalf("an 11 kHz tone survived at %.1f dBFS and would alias", db)
	}
}

func TestMonoAveragesChannels(t *testing.T) {
	a := Audio{SampleRate: 8000, Channels: 2, Samples: []int16{100, 300, -50, 50}}
	m := a.Mono()
	if m.Channels != 1 || len(m.Samples) != 2 || m.Samples[0] != 200 || m.Samples[1] != 0 {
		t.Fatalf("%+v", m)
	}
}

func TestTrimCutsPaddingAndKeepsSpeech(t *testing.T) {
	a := Audio{SampleRate: Rate, Channels: 1, Samples: join(quiet(Rate, 400*time.Millisecond), tone(Rate, 300, -20, time.Second), quiet(Rate, 2500*time.Millisecond))}
	got := a.Trim(50 * time.Millisecond)
	if d := got.Duration(); d < 1090*time.Millisecond || d > 1120*time.Millisecond {
		t.Fatalf("trimmed to %v, want the 1 s of speech plus 50 ms at each end", d)
	}
	abrupt := Audio{SampleRate: Rate, Channels: 1, Samples: tone(Rate, 300, -20, time.Second)}
	padded := abrupt.Trim(50 * time.Millisecond)
	if d := padded.Duration(); d != 1100*time.Millisecond || padded.Samples[0] != 0 || padded.Samples[799] != 0 {
		t.Fatalf("a clip with no lead-in should gain 50 ms of silence at each end, got %v", d)
	}
	silent := Audio{SampleRate: Rate, Channels: 1, Samples: quiet(Rate, time.Second)}
	if len(silent.Trim(50*time.Millisecond).Samples) != len(silent.Samples) {
		t.Fatal("silence was trimmed to nothing")
	}
}

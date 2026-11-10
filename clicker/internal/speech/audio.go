// Package speech turns text into audio and audio into text for the say and
// listen commands. It holds no speech engine: providers call a hosted API
// or run a program the user installed.
package speech

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"math"
	"time"
)

// Rate is the sample rate speech is detected and transcribed at.
const Rate = 16000

// Audio is 16-bit PCM. Samples of several channels are interleaved.
type Audio struct {
	SampleRate int
	Channels   int
	Samples    []int16
}

// Frames is the number of samples per channel.
func (a Audio) Frames() int {
	if a.Channels < 1 {
		return 0
	}
	return len(a.Samples) / a.Channels
}

func (a Audio) Duration() time.Duration {
	if a.SampleRate < 1 {
		return 0
	}
	return time.Duration(float64(a.Frames()) / float64(a.SampleRate) * float64(time.Second))
}

// DurationMs is the length in milliseconds, for results.
func (a Audio) DurationMs() float64 {
	if a.SampleRate < 1 {
		return 0
	}
	return math.Round(float64(a.Frames())/float64(a.SampleRate)*1e6) / 1e3
}

// Mono averages the channels into one.
func (a Audio) Mono() Audio {
	if a.Channels <= 1 {
		return Audio{SampleRate: a.SampleRate, Channels: 1, Samples: a.Samples}
	}
	out := make([]int16, a.Frames())
	for i := range out {
		sum := 0
		for c := 0; c < a.Channels; c++ {
			sum += int(a.Samples[i*a.Channels+c])
		}
		out[i] = int16(sum / a.Channels)
	}
	return Audio{SampleRate: a.SampleRate, Channels: 1, Samples: out}
}

// Resample converts to mono at the given rate with a windowed-sinc filter.
// The result is the same on every machine: each product is rounded before
// it is summed, so no platform fuses the multiply into the add.
func (a Audio) Resample(rate int) Audio {
	m := a.Mono()
	if m.SampleRate == rate || m.SampleRate < 1 || rate < 1 {
		return Audio{SampleRate: rate, Channels: 1, Samples: m.Samples}
	}
	in := m.Samples
	step := float64(m.SampleRate) / float64(rate)
	cutoff := 1.0
	if rate < m.SampleRate {
		cutoff = float64(rate) / float64(m.SampleRate)
	}
	const zeroCrossings = 16
	half := zeroCrossings / cutoff
	n := int((int64(len(in))*int64(rate) + int64(m.SampleRate)/2) / int64(m.SampleRate))
	out := make([]int16, n)
	for i := range out {
		center := float64(i) * step
		lo := int(math.Ceil(center - half))
		hi := int(math.Floor(center + half))
		if lo < 0 {
			lo = 0
		}
		if hi > len(in)-1 {
			hi = len(in) - 1
		}
		var acc, norm float64
		for k := lo; k <= hi; k++ {
			x := float64(k) - center
			w := float64(sinc(x*cutoff) * (0.5 + 0.5*math.Cos(math.Pi*x/half)))
			acc += float64(float64(in[k]) * w)
			norm += w
		}
		if norm != 0 {
			out[i] = clip16(acc / norm)
		}
	}
	return Audio{SampleRate: rate, Channels: 1, Samples: out}
}

func sinc(x float64) float64 {
	if x == 0 {
		return 1
	}
	return math.Sin(math.Pi*x) / (math.Pi * x)
}

func clip16(v float64) int16 {
	v = math.Round(v)
	if v > math.MaxInt16 {
		return math.MaxInt16
	}
	if v < math.MinInt16 {
		return math.MinInt16
	}
	return int16(v)
}

// trimFloor is the level below which the ends of a rendered clip count as
// silence.
const trimFloor = -50.0

// Trim cuts mono audio to its speech with exactly pad of silence at each
// end. Providers pad their output by varying amounts; playback must end
// where the speech ends. A short lead-in keeps a device's audio path from
// clipping the first sound.
func (a Audio) Trim(pad time.Duration) Audio {
	if a.Channels != 1 || a.SampleRate < 1 {
		return a
	}
	frame := a.SampleRate / 100
	if frame < 1 || len(a.Samples) < frame {
		return a
	}
	first, last := -1, -1
	for i := 0; i+frame <= len(a.Samples); i += frame {
		if levelDB(a.Samples[i:i+frame]) > trimFloor {
			if first < 0 {
				first = i
			}
			last = i + frame
		}
	}
	if first < 0 {
		return a
	}
	keep := int(pad.Seconds() * float64(a.SampleRate))
	out := make([]int16, 0, last-first+2*keep)
	out = append(out, make([]int16, keep)...)
	out = append(out, a.Samples[first:last]...)
	out = append(out, make([]int16, keep)...)
	return Audio{SampleRate: a.SampleRate, Channels: 1, Samples: out}
}

// levelDB is the RMS level of samples in dB relative to full scale.
func levelDB(samples []int16) float64 {
	if len(samples) == 0 {
		return -120
	}
	var sum float64
	for _, s := range samples {
		v := float64(s) / 32768
		sum += v * v
	}
	return 10 * math.Log10(sum/float64(len(samples))+1e-12)
}

// PeakDB is the highest 20 ms level in the audio, for error messages.
func (a Audio) PeakDB() float64 {
	m := a.Mono()
	frame := m.SampleRate / 50
	peak := -120.0
	if frame < 1 {
		return peak
	}
	for i := 0; i+frame <= len(m.Samples); i += frame {
		if db := levelDB(m.Samples[i : i+frame]); db > peak {
			peak = db
		}
	}
	return peak
}

// Bytes is the samples as little-endian PCM.
func (a Audio) Bytes() []byte {
	out := make([]byte, len(a.Samples)*2)
	for i, s := range a.Samples {
		binary.LittleEndian.PutUint16(out[i*2:], uint16(s))
	}
	return out
}

// FromBytes reads little-endian 16-bit PCM.
func FromBytes(pcm []byte, rate, channels int) Audio {
	samples := make([]int16, len(pcm)/2)
	for i := range samples {
		samples[i] = int16(binary.LittleEndian.Uint16(pcm[i*2:]))
	}
	return Audio{SampleRate: rate, Channels: channels, Samples: samples}
}

// SHA256 identifies the exact audio: its rate, channels and samples.
func (a Audio) SHA256() string {
	h := sha256.New()
	fmt.Fprintf(h, "%d/%d/", a.SampleRate, a.Channels)
	h.Write(a.Bytes())
	return hex.EncodeToString(h.Sum(nil))
}

// WAV encodes the audio as a 16-bit PCM WAV file.
func (a Audio) WAV() []byte {
	data := a.Bytes()
	var b bytes.Buffer
	b.WriteString("RIFF")
	binary.Write(&b, binary.LittleEndian, uint32(36+len(data)))
	b.WriteString("WAVEfmt ")
	binary.Write(&b, binary.LittleEndian, uint32(16))
	binary.Write(&b, binary.LittleEndian, uint16(1))
	binary.Write(&b, binary.LittleEndian, uint16(a.Channels))
	binary.Write(&b, binary.LittleEndian, uint32(a.SampleRate))
	binary.Write(&b, binary.LittleEndian, uint32(a.SampleRate*a.Channels*2))
	binary.Write(&b, binary.LittleEndian, uint16(a.Channels*2))
	binary.Write(&b, binary.LittleEndian, uint16(16))
	b.WriteString("data")
	binary.Write(&b, binary.LittleEndian, uint32(len(data)))
	b.Write(data)
	return b.Bytes()
}

// ErrNotWAV says the bytes are not a WAV file this package reads.
var ErrNotWAV = errors.New("not a PCM WAV file")

// ParseWAV decodes integer or float PCM of any rate and channel count.
// A data chunk with no declared length, as streaming encoders write, is
// read to the end.
func ParseWAV(raw []byte) (Audio, error) {
	if len(raw) < 12 || string(raw[0:4]) != "RIFF" || string(raw[8:12]) != "WAVE" {
		return Audio{}, ErrNotWAV
	}
	var format, channels, bits uint16
	var rate uint32
	var data []byte
	haveFmt, haveData := false, false
	for pos := 12; pos+8 <= len(raw) && !haveData; {
		id := string(raw[pos : pos+4])
		size := int(binary.LittleEndian.Uint32(raw[pos+4 : pos+8]))
		body := pos + 8
		switch id {
		case "fmt ":
			if size < 16 || body+16 > len(raw) {
				return Audio{}, ErrNotWAV
			}
			format = binary.LittleEndian.Uint16(raw[body:])
			channels = binary.LittleEndian.Uint16(raw[body+2:])
			rate = binary.LittleEndian.Uint32(raw[body+4:])
			bits = binary.LittleEndian.Uint16(raw[body+14:])
			if format == 0xFFFE && size >= 26 && body+26 <= len(raw) {
				format = binary.LittleEndian.Uint16(raw[body+24:])
			}
			haveFmt = true
		case "data":
			end := body + size
			if size == 0 || size == 0xFFFFFFFF || end > len(raw) {
				end = len(raw)
			}
			data = raw[body:end]
			haveData = true
		}
		pos = body + size + size%2
	}
	if !haveFmt || !haveData || channels < 1 || rate < 1 {
		return Audio{}, ErrNotWAV
	}
	width := int(bits) / 8
	if width < 1 {
		return Audio{}, ErrNotWAV
	}
	n := len(data) / width
	n -= n % int(channels)
	samples := make([]int16, n)
	for i := 0; i < n; i++ {
		p := data[i*width:]
		switch {
		case format == 1 && bits == 8:
			samples[i] = int16(int(p[0])-128) << 8
		case format == 1 && bits == 16:
			samples[i] = int16(binary.LittleEndian.Uint16(p))
		case format == 1 && bits == 24:
			samples[i] = int16(binary.LittleEndian.Uint16(p[1:]))
		case format == 1 && bits == 32:
			samples[i] = int16(binary.LittleEndian.Uint16(p[2:]))
		case format == 3 && bits == 32:
			samples[i] = clip16(float64(math.Float32frombits(binary.LittleEndian.Uint32(p))) * 32767)
		case format == 3 && bits == 64:
			samples[i] = clip16(math.Float64frombits(binary.LittleEndian.Uint64(p)) * 32767)
		default:
			return Audio{}, fmt.Errorf("%w: format %d with %d bits", ErrNotWAV, format, bits)
		}
	}
	return Audio{SampleRate: int(rate), Channels: int(channels), Samples: samples}, nil
}

// ReadWAV reads a whole WAV stream.
func ReadWAV(r io.Reader) (Audio, error) {
	raw, err := io.ReadAll(r)
	if err != nil {
		return Audio{}, err
	}
	return ParseWAV(raw)
}

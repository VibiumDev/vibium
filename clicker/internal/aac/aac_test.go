package aac

import (
	"bytes"
	"encoding/binary"
	"math"
	"math/big"
	"math/rand"
	"os/exec"
	"testing"
)

// The decoder below reads what this package writes: one long sine window a
// frame, one channel, no tools. It follows the standard's text and not the
// encoder's code, so the two check each other.

type bitReader struct {
	data []byte
	pos  int
}

func (r *bitReader) get(n int) uint32 {
	var v uint32
	for ; n > 0; n-- {
		var bit byte
		if r.pos/8 < len(r.data) {
			bit = r.data[r.pos/8] >> (7 - r.pos%8) & 1
		}
		v = v<<1 | uint32(bit)
		r.pos++
	}
	return v
}

type codeword struct {
	bits uint8
	code uint32
}

var (
	sfWords   = map[codeword]int{}
	bookWords [11]map[codeword]int
)

func init() {
	for i, c := range sfCodes {
		sfWords[codeword{sfBits[i], c}] = i
	}
	for b := range bookCodes {
		bookWords[b] = map[codeword]int{}
		for i, c := range bookCodes[b] {
			bookWords[b][codeword{bookBits[b][i], uint32(c)}] = i
		}
	}
}

func (r *bitReader) word(t *testing.T, words map[codeword]int) int {
	var w codeword
	for w.bits < 20 {
		w.code = w.code<<1 | r.get(1)
		w.bits++
		if i, ok := words[w]; ok {
			return i
		}
	}
	t.Fatalf("no codeword at bit %d", r.pos)
	return 0
}

// spectrum reads one frame's coefficients.
func spectrum(t *testing.T, frame []byte) [FrameSize]float64 {
	t.Helper()
	var spec [FrameSize]float64
	r := &bitReader{data: frame}
	if id := r.get(3); id != 0 {
		t.Fatalf("element %d, want a single channel", id)
	}
	r.get(4)
	gain := int(r.get(8))
	if head := r.get(4); head != 0 {
		t.Fatalf("window bits %04b, want one long sine window", head)
	}
	last := int(r.get(6))
	if r.get(1) != 0 {
		t.Fatal("prediction is on")
	}
	var cb [numBands]int
	for b := 0; b < last; {
		c, n := int(r.get(4)), 0
		for {
			incr := int(r.get(5))
			n += incr
			if incr != 31 {
				break
			}
		}
		if n == 0 || b+n > last {
			t.Fatalf("a section of %d bands at band %d of %d", n, b, last)
		}
		for ; n > 0; n-- {
			cb[b] = c
			b++
		}
	}
	var sf [numBands]int
	step := gain
	for b := 0; b < last; b++ {
		if cb[b] != 0 {
			step += r.word(t, sfWords) - 60
			sf[b] = step
		}
	}
	if r.get(3) != 0 {
		t.Fatal("pulses, noise shaping or gain control are on")
	}
	for b := 0; b < last; b++ {
		if cb[b] == 0 {
			continue
		}
		k := books[cb[b]]
		scale := math.Pow(2, 0.25*float64(sf[b]-sfOffset))
		for at := bands[b]; at < bands[b+1]; at += k.dim {
			i := r.word(t, bookWords[cb[b]-1])
			v := make([]int, k.dim)
			base := k.lav + 1
			if k.signed {
				base = 2*k.lav + 1
			}
			for d := k.dim - 1; d >= 0; d-- {
				v[d] = i % base
				i /= base
				if k.signed {
					v[d] -= k.lav
				}
			}
			neg := make([]bool, k.dim)
			if !k.signed {
				for d := range v {
					neg[d] = v[d] != 0 && r.get(1) == 1
				}
			}
			for d := range v {
				if cb[b] == escBook && v[d] == 16 {
					ones := 0
					for r.get(1) == 1 {
						ones++
					}
					v[d] = 1<<(ones+4) + int(r.get(ones+4))
				}
				x := math.Pow(math.Abs(float64(v[d])), 4.0/3) * scale
				if neg[d] || v[d] < 0 {
					x = -x
				}
				spec[at+d] = x
			}
		}
	}
	if r.get(3) != 7 {
		t.Fatalf("no end of frame at bit %d", r.pos-3)
	}
	if left := len(frame)*8 - r.pos; left < 0 || left > 7 {
		t.Fatalf("%d bits after the end of the frame", left)
	}
	return spec
}

var cosines = func() (c [8192]float64) {
	for i := range c {
		c[i] = math.Cos(2 * math.Pi * float64(i) / 8192)
	}
	return c
}()

// phase is cos(2π/N (n + n0)(k + 1/2)) for N = 2048 and n0 = (N/2 + 1)/2.
func phase(n, k int) float64 { return cosines[(2*n+1025)*(2*k+1)%8192] }

func sine(n int) float64 { return math.Sin(math.Pi * (float64(n) + 0.5) / (2 * FrameSize)) }

// decode gives the samples a decoder plays, the Delay included.
func decode(t *testing.T, frames [][]byte) []float64 {
	t.Helper()
	out := make([]float64, 0, len(frames)*FrameSize)
	var tail [FrameSize]float64
	for _, f := range frames {
		spec := spectrum(t, f)
		for n := 0; n < 2*FrameSize; n++ {
			var y float64
			for k, s := range spec {
				if s != 0 {
					y += s * phase(n, k)
				}
			}
			y *= sine(n) / FrameSize
			if n < FrameSize {
				out = append(out, tail[n]+y)
			} else {
				tail[n-FrameSize] = y
			}
		}
	}
	return out
}

func snr(want []int16, got []float64) float64 {
	var signal, noise float64
	for i, w := range want {
		d := float64(w) - got[i]
		signal += float64(w) * float64(w)
		noise += d * d
	}
	if noise == 0 {
		return math.Inf(1)
	}
	return 10 * math.Log10(signal/noise)
}

// voice is a second of something shaped like speech: a pitch with its
// harmonics under two resonances, rising and falling a few times a second.
func voice(seconds, peak float64) []int16 {
	out := make([]int16, int(seconds*Rate))
	rng := rand.New(rand.NewSource(1))
	for i := range out {
		at := float64(i) / Rate
		var v float64
		for h := 1.0; h*140 < 7000; h++ {
			f := h * 140
			shape := math.Exp(-math.Pow((f-600)/400, 2)) + 0.5*math.Exp(-math.Pow((f-1800)/600, 2)) + 0.02
			v += shape * math.Sin(2*math.Pi*f*at+h)
		}
		v *= 0.25 * (0.55 + 0.45*math.Sin(2*math.Pi*3*at))
		v += 0.002 * rng.NormFloat64()
		out[i] = int16(peak * math.Max(-1, math.Min(1, v)))
	}
	return out
}

func tone(seconds, hz, peak float64) []int16 {
	out := make([]int16, int(seconds*Rate))
	for i := range out {
		out[i] = int16(peak * math.Sin(2*math.Pi*hz*float64(i)/Rate))
	}
	return out
}

func hiss(seconds, peak float64) []int16 {
	out := make([]int16, int(seconds*Rate))
	rng := rand.New(rand.NewSource(2))
	for i := range out {
		out[i] = int16(peak * (2*rng.Float64() - 1))
	}
	return out
}

func square(seconds, hz float64) []int16 {
	out := make([]int16, int(seconds*Rate))
	for i := range out {
		out[i] = 32767
		if math.Mod(float64(i)*hz/Rate, 1) >= 0.5 {
			out[i] = -32768
		}
	}
	return out
}

func checkFrames(t *testing.T, samples []int16, frames [][]byte) {
	t.Helper()
	if want := (len(samples)+FrameSize-1)/FrameSize + 1; len(frames) != want {
		t.Fatalf("%d frames for %d samples, want %d", len(frames), len(samples), want)
	}
	for i, f := range frames {
		if len(f)*8 > 6144 {
			t.Fatalf("frame %d is %d bits, over the format's 6144", i, len(f)*8)
		}
	}
}

func TestTransformMatchesItsDefinition(t *testing.T) {
	e := newEncoder()
	rng := rand.New(rand.NewSource(3))
	var x [2 * FrameSize]float64
	for i := range x {
		x[i] = 2*rng.Float64() - 1
	}
	e.mdct(&x)
	for k := 0; k < FrameSize; k += 37 {
		var want float64
		for n, v := range x {
			want += 2 * v * sine(n) * phase(n, k)
		}
		if math.Abs(e.spec[k]-want) > 1e-8 {
			t.Fatalf("coefficient %d is %v, want %v", k, e.spec[k], want)
		}
	}
}

func TestCodebooksArePrefixCodes(t *testing.T) {
	check := func(name string, codes []uint32, lengths []uint8) {
		// A complete prefix code fills the code space exactly.
		sum := new(big.Int)
		seen := map[codeword]bool{}
		for i, n := range lengths {
			if codes[i]>>n != 0 {
				t.Errorf("%s: code %d does not fit its %d bits", name, i, n)
			}
			sum.Add(sum, new(big.Int).Lsh(big.NewInt(1), uint(32-n)))
			for p := uint8(1); p <= n; p++ {
				if seen[codeword{p, codes[i] >> (n - p)}] && p < n {
					t.Errorf("%s: code %d starts with another code", name, i)
				}
			}
			if seen[codeword{n, codes[i]}] {
				t.Errorf("%s: code %d is used twice", name, i)
			}
			seen[codeword{n, codes[i]}] = true
		}
		if sum.Cmp(new(big.Int).Lsh(big.NewInt(1), 32)) != 0 {
			t.Errorf("%s: the codes do not fill the code space", name)
		}
	}
	check("scalefactors", sfCodes[:], sfBits[:])
	for b := range bookCodes {
		k := books[b+1]
		size := 1
		for d := 0; d < k.dim; d++ {
			if k.signed {
				size *= 2*k.lav + 1
			} else {
				size *= k.lav + 1
			}
		}
		if len(bookCodes[b]) != size || len(bookBits[b]) != size {
			t.Errorf("codebook %d has %d codes, want %d", b+1, len(bookCodes[b]), size)
		}
		wide := make([]uint32, len(bookCodes[b]))
		for i, c := range bookCodes[b] {
			wide[i] = uint32(c)
		}
		check("codebook", wide, bookBits[b])
	}
}

func TestSpeechComesBackClean(t *testing.T) {
	for _, peak := range []float64{30000, 3000, 300} {
		in := voice(2, peak)
		frames := Encode(in)
		checkFrames(t, in, frames)
		out := decode(t, frames)
		got := snr(in, out[Delay:])
		size := 0
		for _, f := range frames {
			size += len(f)
		}
		t.Logf("peak %5.0f: %.1f dB, %d kbit/s", peak, got, size*8*Rate/(len(frames)*FrameSize)/1000)
		if got < 25 {
			t.Errorf("speech peaking at %.0f comes back %.1f dB above the noise, want 25", peak, got)
		}
	}
}

func TestSilenceIsSmallAndStaysSilent(t *testing.T) {
	in := make([]int16, 3*FrameSize)
	frames := Encode(in)
	checkFrames(t, in, frames)
	for i, f := range frames {
		if len(f) != 4 {
			t.Errorf("silent frame %d is %d bytes, want 4", i, len(f))
		}
	}
	for i, v := range decode(t, frames) {
		if v != 0 {
			t.Fatalf("sample %d of silence is %v", i, v)
		}
	}
	if Encode(nil) != nil {
		t.Error("no samples made frames")
	}
}

func TestLoudSoundFitsItsFrames(t *testing.T) {
	for name, in := range map[string][]int16{
		"noise":  hiss(1, 32767),
		"square": square(1, 440),
		"tone":   tone(1, 1000, 32767),
		"click":  append(make([]int16, 1500), append([]int16{32767, -32768, 32767}, make([]int16, 1500)...)...),
	} {
		frames := Encode(in)
		checkFrames(t, in, frames)
		got := snr(in, decode(t, frames)[Delay:])
		t.Logf("%s: %.1f dB", name, got)
		if got < 6 {
			t.Errorf("%s comes back %.1f dB above the noise", name, got)
		}
	}
}

func TestQuietSoundIsKept(t *testing.T) {
	in := tone(1, 440, 40)
	if got := snr(in, decode(t, Encode(in))[Delay:]); got < 15 {
		t.Errorf("a tone at -58 dB comes back %.1f dB above the noise, want 15", got)
	}
}

// adts wraps frames the way a .aac file holds them.
func adts(frames [][]byte) []byte {
	var out bytes.Buffer
	for _, f := range frames {
		n := 7 + len(f)
		out.Write([]byte{0xff, 0xf1, 1<<6 | 8<<2, 1<<6 | byte(n>>11), byte(n >> 3), byte(n&7)<<5 | 0x1f, 0xfc})
		out.Write(f)
	}
	return out.Bytes()
}

// FFmpeg's decoder is the second opinion on the format: it must give what
// the decoder above gives.
func TestFFmpegHearsTheSame(t *testing.T) {
	ffmpeg, err := exec.LookPath("ffmpeg")
	if err != nil {
		t.Skip("ffmpeg is not installed")
	}
	for name, in := range map[string][]int16{
		"speech": voice(2, 20000),
		"quiet":  voice(1, 300),
		"noise":  hiss(1, 32767),
		"square": square(1, 440),
		"mixed":  append(voice(1, 30000), append(make([]int16, 3000), hiss(1, 2000)...)...),
	} {
		frames := Encode(in)
		cmd := exec.Command(ffmpeg, "-v", "error", "-f", "aac", "-i", "pipe:0", "-f", "s16le", "-ac", "1", "-ar", "16000", "pipe:1")
		cmd.Stdin = bytes.NewReader(adts(frames))
		raw, err := cmd.Output()
		if err != nil {
			t.Fatalf("%s: ffmpeg: %v", name, err)
		}
		theirs := make([]int16, len(raw)/2)
		binary.Read(bytes.NewReader(raw), binary.LittleEndian, theirs)
		ours := decode(t, frames)
		// Some builds drop the decoder's first frame, the Delay.
		if len(theirs) == len(ours)-Delay {
			ours = ours[Delay:]
		}
		if len(theirs) != len(ours) {
			t.Fatalf("%s: ffmpeg gave %d samples, want %d", name, len(theirs), len(ours))
		}
		worst := 0.0
		for i, v := range theirs {
			want := math.Max(-32768, math.Min(32767, ours[i]))
			worst = math.Max(worst, math.Abs(float64(v)-want))
		}
		if worst > 2 {
			t.Errorf("%s: ffmpeg's samples differ by up to %.1f", name, worst)
		}
	}
}

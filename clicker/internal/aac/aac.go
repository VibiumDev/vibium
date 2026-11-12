// Package aac encodes 16 kHz mono sound as AAC-LC, the audio every MP4
// player takes. It has no model of hearing: each frame gets the finest
// step that fits a frame, down to one fine enough for speech.
package aac

import (
	"math"
	"math/bits"
	"math/cmplx"
)

const (
	// Rate is the sample rate, in samples a second.
	Rate = 16000
	// FrameSize is how many samples one frame adds.
	FrameSize = 1024
	// Delay is how many samples a decoder gives before the first one that
	// was encoded.
	Delay = FrameSize

	// maxFrameBits is under the 6144 bits the format allows a channel.
	maxFrameBits = 6000
	// finestStep is the scalefactor below which a finer step is not heard.
	finestStep = 128
	sfOffset   = 100
	maxValue   = 8191
)

// Config is the stream's AudioSpecificConfig: AAC-LC, 16 kHz, one channel.
var Config = []byte{0x14, 0x08}

// bands are the scalefactor bands of a long window at 16 kHz.
var bands = [...]int{
	0, 8, 16, 24, 32, 40, 48, 56, 64, 72, 80, 88, 100, 112, 124, 136,
	148, 160, 172, 184, 196, 212, 228, 244, 260, 280, 300, 320, 344, 368, 396, 424,
	456, 492, 532, 572, 616, 664, 716, 772, 832, 896, 960, 1024,
}

const numBands = len(bands) - 1

// book is how a spectral codebook groups values: dim at a time, each up
// to lav, with the sign in the code or after it.
type book struct {
	dim, lav int
	signed   bool
}

var books = [12]book{
	{}, {4, 1, true}, {4, 1, true}, {4, 2, false}, {4, 2, false},
	{2, 4, true}, {2, 4, true}, {2, 7, false}, {2, 7, false},
	{2, 12, false}, {2, 12, false}, {2, 16, false},
}

const escBook = 11

// index is where a group of values sits in codebook b.
func (k book) index(v []int32) int {
	i := 0
	for _, x := range v[:k.dim] {
		if k.signed {
			i = i*(2*k.lav+1) + int(x) + k.lav
			continue
		}
		if x < 0 {
			x = -x
		}
		i = i*(k.lav+1) + min(int(x), k.lav)
	}
	return i
}

// escape splits a value of 16 or more into its count of leading ones and
// the bits after them.
func escape(v int32) (ones int, rest uint32) {
	ones = bits.Len32(uint32(v)) - 5
	return ones, uint32(v) - 1<<(ones+4)
}

// cost is the bits codebook b takes for the values of one band.
func cost(b int, v []int32) int {
	k := books[b]
	n := 0
	for ; len(v) >= k.dim; v = v[k.dim:] {
		n += int(bookBits[b-1][k.index(v)])
		if k.signed {
			continue
		}
		for _, x := range v[:k.dim] {
			if x < 0 {
				x = -x
			}
			if x != 0 {
				n++
			}
			if b == escBook && x >= 16 {
				ones, _ := escape(x)
				n += 2*ones + 5
			}
		}
	}
	return n
}

type bitWriter struct {
	buf  []byte
	acc  uint64
	fill int
}

func (w *bitWriter) put(v uint32, n int) {
	w.acc = w.acc<<n | uint64(v)
	w.fill += n
	for w.fill >= 8 {
		w.fill -= 8
		w.buf = append(w.buf, byte(w.acc>>w.fill))
	}
}

func (w *bitWriter) finish() []byte {
	if w.fill > 0 {
		w.put(0, 8-w.fill)
	}
	return w.buf
}

type encoder struct {
	window [2 * FrameSize]float64
	twist  [FrameSize / 2]complex128
	roots  [FrameSize / 4]complex128

	spec  [FrameSize]float64
	power [FrameSize]float64 // |spec|^(3/4)
	q     [FrameSize]int32
	cb    [numBands]int
}

func newEncoder() *encoder {
	e := &encoder{}
	for n := range e.window {
		e.window[n] = math.Sin(math.Pi * (float64(n) + 0.5) / (2 * FrameSize))
	}
	for n := range e.twist {
		e.twist[n] = cmplx.Rect(1, -math.Pi*float64(8*n+1)/(8*FrameSize))
	}
	for n := range e.roots {
		e.roots[n] = cmplx.Rect(1, -2*math.Pi*float64(n)/(FrameSize/2))
	}
	return e
}

// fft is an in-place transform of FrameSize/2 points.
func (e *encoder) fft(a *[FrameSize / 2]complex128) {
	const n = FrameSize / 2
	shift := 32 - bits.Len32(n-1)
	for i := range a {
		if j := int(bits.Reverse32(uint32(i)) >> shift); j > i {
			a[i], a[j] = a[j], a[i]
		}
	}
	for size := 2; size <= n; size <<= 1 {
		half, step := size/2, n/size
		for start := 0; start < n; start += size {
			for k := 0; k < half; k++ {
				t := a[start+k+half] * e.roots[k*step]
				a[start+k+half] = a[start+k] - t
				a[start+k] += t
			}
		}
	}
}

// mdct fills e.spec from two frames of samples: the transform of ISO/IEC
// 14496-3 4.6.11 under a sine window, by way of an FFT a quarter its size.
func (e *encoder) mdct(x *[2 * FrameSize]float64) {
	const n = FrameSize
	// Folded about the middle of each half, the two frames are one frame's
	// worth of values with the same transform.
	var u [n]float64
	for i := 0; i < n/2; i++ {
		u[i] = -x[3*n/2-1-i]*e.window[3*n/2-1-i] - x[3*n/2+i]*e.window[3*n/2+i]
		u[n/2+i] = x[i]*e.window[i] - x[n-1-i]*e.window[n-1-i]
	}
	var t [n / 2]complex128
	for i := range t {
		t[i] = complex(u[2*i], u[n-1-2*i]) * e.twist[i]
	}
	e.fft(&t)
	for k := range t {
		y := t[k] * e.twist[k]
		e.spec[2*k] = 2 * real(y)
		e.spec[n-1-2*k] = -2 * imag(y)
	}
}

// plan quantizes the spectrum with one step and picks each band's
// codebook. It returns the frame's size in bits, or false when a value
// is too large for the step.
func (e *encoder) plan(sf int) (int, bool) {
	gain := math.Pow(2, -3.0/16*float64(sf-sfOffset))
	for i, p := range e.power {
		v := p*gain + 0.4054
		if v > maxValue+1 {
			return 0, false
		}
		q := int32(v)
		if q > maxValue {
			return 0, false
		}
		if e.spec[i] < 0 {
			q = -q
		}
		e.q[i] = q
	}
	total := 26 + 3 + 3
	last := 0
	for b := 0; b < numBands; b++ {
		v := e.q[bands[b]:bands[b+1]]
		var peak int32
		for _, x := range v {
			peak = max(peak, x, -x)
		}
		e.cb[b] = 0
		if peak == 0 {
			continue
		}
		best := -1
		for c := 1; c <= escBook; c++ {
			if c < escBook && int(peak) > books[c].lav {
				continue
			}
			if n := cost(c, v); best < 0 || n < best {
				best, e.cb[b] = n, c
			}
		}
		total += best + 1
		last = b + 1
	}
	for b := 0; b < last; {
		run := b + 1
		for run < last && e.cb[run] == e.cb[b] {
			run++
		}
		total += 4 + 5*((run-b)/31+1)
		b = run
	}
	return total, true
}

// frame encodes the spectrum in e.spec as one raw data block.
func (e *encoder) frame() []byte {
	for i, v := range e.spec {
		e.power[i] = math.Pow(math.Abs(v), 0.75)
	}
	// The finest step that fits: frames get smaller as the step grows.
	lo, hi := finestStep, 255
	for lo < hi {
		mid := (lo + hi) / 2
		if n, ok := e.plan(mid); ok && n <= maxFrameBits {
			hi = mid
		} else {
			lo = mid + 1
		}
	}
	sf := lo
	for n, ok := e.plan(sf); (!ok || n > maxFrameBits) && sf < 255; n, ok = e.plan(sf) {
		sf++
	}

	last := 0
	for b, c := range e.cb {
		if c != 0 {
			last = b + 1
		}
	}
	w := &bitWriter{buf: make([]byte, 0, maxFrameBits/8+8)}
	w.put(0, 3) // single channel element
	w.put(0, 4) // its tag
	w.put(uint32(sf), 8)
	w.put(0, 1) // reserved
	w.put(0, 2) // one long window
	w.put(0, 1) // sine
	w.put(uint32(last), 6)
	w.put(0, 1) // no prediction
	for b := 0; b < last; {
		run := b + 1
		for run < last && e.cb[run] == e.cb[b] {
			run++
		}
		w.put(uint32(e.cb[b]), 4)
		n := run - b
		for ; n >= 31; n -= 31 {
			w.put(31, 5)
		}
		w.put(uint32(n), 5)
		b = run
	}
	// Every band has the global step, so each difference is zero.
	for b := 0; b < last; b++ {
		if e.cb[b] != 0 {
			w.put(sfCodes[60], int(sfBits[60]))
		}
	}
	w.put(0, 3) // no pulses, noise shaping or gain control
	for b := 0; b < last; b++ {
		c := e.cb[b]
		if c == 0 {
			continue
		}
		k := books[c]
		for v := e.q[bands[b]:bands[b+1]]; len(v) >= k.dim; v = v[k.dim:] {
			i := k.index(v)
			w.put(uint32(bookCodes[c-1][i]), int(bookBits[c-1][i]))
			if k.signed {
				continue
			}
			for _, x := range v[:k.dim] {
				if x != 0 {
					w.put(uint32(x>>31)&1, 1)
				}
			}
			if c != escBook {
				continue
			}
			for _, x := range v[:k.dim] {
				if x = max(x, -x); x >= 16 {
					ones, rest := escape(x)
					w.put(1<<ones-1, ones)
					w.put(0, 1)
					w.put(rest, ones+4)
				}
			}
		}
	}
	w.put(7, 3) // end
	return w.finish()
}

// Encode turns samples at Rate into frames of FrameSize samples each. A
// decoder gives Delay samples first, then the samples.
func Encode(samples []int16) [][]byte {
	if len(samples) == 0 {
		return nil
	}
	e := newEncoder()
	frames := (len(samples)+FrameSize-1)/FrameSize + 1
	out := make([][]byte, 0, frames)
	var x [2 * FrameSize]float64
	for f := 0; f < frames; f++ {
		copy(x[:FrameSize], x[FrameSize:])
		for i := 0; i < FrameSize; i++ {
			x[FrameSize+i] = 0
			if at := f*FrameSize + i; at < len(samples) {
				x[FrameSize+i] = float64(samples[at])
			}
		}
		e.mdct(&x)
		out = append(out, e.frame())
	}
	return out
}

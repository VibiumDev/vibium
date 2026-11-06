package sift

import "math"

const (
	sigma0       = 1.6
	cameraBlur   = 0.5
	intervals    = 3
	minOctaveDim = 16
	maxOctaves   = 8
)

type octave struct {
	w, h  int
	gauss []*grayImage // intervals+3 layers
	dog   []*grayImage // intervals+2 layers
}

func numOctaves(w, h int) int {
	m := min(w, h)
	n := 0
	for m >= minOctaveDim && n < maxOctaves {
		m /= 2
		n++
	}
	return n
}

func gaussKernel(sigma float64) []float32 {
	r := int(math.Ceil(4 * sigma))
	k := make([]float32, 2*r+1)
	var sum float64
	for i := -r; i <= r; i++ {
		v := math.Exp(-float64(i*i) / (2 * sigma * sigma))
		k[i+r] = float32(v)
		sum += v
	}
	for i := range k {
		k[i] = float32(float64(k[i]) / sum)
	}
	return k
}

// clampIdx replicates the edge pixel. Mirroring would invent structure
// that the page around a cropped reference does not have.
func clampIdx(i, n int) int {
	if i < 0 {
		return 0
	}
	if i >= n {
		return n - 1
	}
	return i
}

// padReplicate grows the image by p pixels on every side with the edge
// pixel, so features near a small reference's border keep their support.
func padReplicate(src *grayImage, p int) *grayImage {
	w, h := src.w+2*p, src.h+2*p
	dst := newGray(w, h)
	for y := 0; y < h; y++ {
		sy := clampIdx(y-p, src.h)
		in := src.pix[sy*src.w : (sy+1)*src.w]
		out := dst.pix[y*w : (y+1)*w]
		for x := 0; x < p; x++ {
			out[x] = in[0]
		}
		copy(out[p:p+src.w], in)
		for x := p + src.w; x < w; x++ {
			out[x] = in[src.w-1]
		}
	}
	return dst
}

// bufPool recycles layer buffers between bands of one octave.
type bufPool struct{ free [][]float32 }

func (p *bufPool) get(n int) []float32 {
	for i, b := range p.free {
		if cap(b) >= n {
			p.free = append(p.free[:i], p.free[i+1:]...)
			return b[:n]
		}
	}
	return make([]float32, n)
}

func (p *bufPool) put(b []float32) { p.free = append(p.free, b) }

func gaussianBlur(src *grayImage, sigma float64) *grayImage {
	dst := newGray(src.w, src.h)
	gaussianBlurInto(src, sigma, dst, newGray(src.w, src.h))
	return dst
}

const blurBlock = 64

// convolve keeps four independent accumulators so the FMAs pipeline
// instead of serializing on one register.
func convolve(win, k []float32) float32 {
	win = win[:len(k)]
	var s0, s1, s2, s3 float32
	i := 0
	for ; i+4 <= len(k); i += 4 {
		s0 += k[i] * win[i]
		s1 += k[i+1] * win[i+1]
		s2 += k[i+2] * win[i+2]
		s3 += k[i+3] * win[i+3]
	}
	for ; i < len(k); i++ {
		s0 += k[i] * win[i]
	}
	return (s0 + s1) + (s2 + s3)
}

// gaussianBlurInto writes the separable blur of src into dst using tmp as
// the horizontal intermediate. Both must match src in size.
func gaussianBlurInto(src *grayImage, sigma float64, dst, tmp *grayImage) {
	k := gaussKernel(sigma)
	r := len(k) / 2
	w, h := src.w, src.h
	parallelFor(h, func(lo, hi, _ int) {
		pad := make([]float32, w+2*r)
		for y := lo; y < hi; y++ {
			row := src.pix[y*w : (y+1)*w]
			for i := range pad {
				pad[i] = row[clampIdx(i-r, w)]
			}
			out := tmp.pix[y*w : (y+1)*w]
			for x := range out {
				out[x] = convolve(pad[x:x+len(k)], k)
			}
		}
	})
	parallelFor(h, func(lo, hi, _ int) {
		var acc [blurBlock]float32
		for y := lo; y < hi; y++ {
			out := dst.pix[y*w : (y+1)*w]
			for x0 := 0; x0 < w; x0 += blurBlock {
				a := acc[:min(blurBlock, w-x0)]
				clear(a)
				for i, kv := range k {
					sy := clampIdx(y+i-r, h)
					in := tmp.pix[sy*w+x0 : sy*w+x0+len(a)]
					for j := range a {
						a[j] += kv * in[j]
					}
				}
				copy(out[x0:], a)
			}
		}
	})
}

// upsampleRows returns rows [y0, y1) of the 2x bilinear upsample of src,
// aligned so upsampled pixel 2i sits on source pixel i.
func upsampleRows(src *grayImage, y0, y1 int, pool *bufPool) *grayImage {
	w := src.w * 2
	dst := &grayImage{w: w, h: y1 - y0, pix: pool.get(w * (y1 - y0))}
	parallelFor(y1-y0, func(lo, hi, _ int) {
		for y := lo; y < hi; y++ {
			sy := (y0 + y) / 2
			sy1 := min(sy+1, src.h-1)
			fy := float32((y0+y)%2) * 0.5
			out := dst.pix[y*w : (y+1)*w]
			for x := range out {
				sx := x / 2
				sx1 := min(sx+1, src.w-1)
				fx := float32(x%2) * 0.5
				top := src.at(sx, sy)*(1-fx) + src.at(sx1, sy)*fx
				bot := src.at(sx, sy1)*(1-fx) + src.at(sx1, sy1)*fx
				out[x] = top*(1-fy) + bot*fy
			}
		}
	})
	return dst
}

// rowSource yields rows [y0, y1) of an octave base plus the pooled buffer
// backing them, or nil when the rows are a view the caller must not free.
// The first octave is produced on demand so a doubled screen is never
// resident at full size.
type rowSource func(y0, y1 int) (*grayImage, []float32)

// initialSource upsamples (optionally) and blurs g to sigma0 per request,
// with enough context rows that the result equals a whole-image blur.
func initialSource(g *grayImage, upsample bool, pool *bufPool) (int, int, rowSource) {
	w, h := g.w, g.h
	initBlur := cameraBlur
	if upsample {
		w, h = 2*w, 2*h
		initBlur = 2 * cameraBlur
	}
	sigma := math.Sqrt(sigma0*sigma0 - initBlur*initBlur)
	ctx := int(math.Ceil(4*sigma)) + 1
	return w, h, func(y0, y1 int) (*grayImage, []float32) {
		wy0, wy1 := max(0, y0-ctx), min(h, y1+ctx)
		var win *grayImage
		if upsample {
			win = upsampleRows(g, wy0, wy1, pool)
		} else {
			win = &grayImage{w: w, h: wy1 - wy0, pix: g.pix[wy0*w : wy1*w]}
		}
		out := &grayImage{w: w, h: win.h, pix: pool.get(w * win.h)}
		tmp := &grayImage{w: w, h: win.h, pix: pool.get(w * win.h)}
		gaussianBlurInto(win, sigma, out, tmp)
		pool.put(tmp.pix)
		if upsample {
			pool.put(win.pix)
		}
		return &grayImage{w: w, h: y1 - y0, pix: out.pix[(y0-wy0)*w : (y1-wy0)*w]}, out.pix
	}
}

func sliceSource(img *grayImage) rowSource {
	return func(y0, y1 int) (*grayImage, []float32) {
		return &grayImage{w: img.w, h: y1 - y0, pix: img.pix[y0*img.w : y1*img.w]}, nil
	}
}

const (
	// bandPixels bounds the area of one band so a full octave stack of a
	// large image never has to be resident at once.
	bandPixels = 1 << 21
	// bandOverlap covers the summed blur radii (43 rows), the widest
	// descriptor window (38) and the refinement walk (16), so a band's
	// owned rows are computed exactly as they would be unbanded.
	bandOverlap = 100
	refineWalk  = 16
)

// processOctave detects and describes one octave in horizontal bands and
// returns the next octave's base when wanted. Band boundaries are a
// function of the image size only, so results stay deterministic.
func processOctave(w, h int, src rowSource, coordScale float32, wantNext bool, pool *bufPool) ([]keypoint, []float32, *grayImage) {
	bandRows := h
	if w*h > bandPixels {
		bandRows = max(bandPixels/w, 2*bandOverlap) &^ 1
	}
	var kps []keypoint
	var desc []float32
	var next *grayImage
	if wantNext {
		next = &grayImage{w: w / 2, h: h / 2, pix: pool.get(w / 2 * (h / 2))}
	}
	for y0 := 0; y0 < h; y0 += bandRows {
		y1 := min(y0+bandRows, h)
		subY0 := max(0, y0-bandOverlap)
		subY1 := min(h, y1+bandOverlap)
		sub, owned := src(subY0, subY1)
		oct := buildOctave(sub, pool)
		cands := detectCandidates(oct, y0-subY0, y1-subY0)
		k, d := describeOctave(oct, cands, coordScale)
		for i := range k {
			k[i].y += float32(subY0) * coordScale
		}
		kps = append(kps, k...)
		desc = append(desc, d...)
		if next != nil {
			g := oct.gauss[intervals]
			for y := (y0 + 1) / 2; y < (y1+1)/2 && y < next.h; y++ {
				in := g.pix[(2*y-subY0)*w:]
				out := next.pix[y*next.w : (y+1)*next.w]
				for x := range out {
					out[x] = in[2*x]
				}
			}
		}
		oct.release(pool)
		if owned != nil {
			pool.put(owned)
		}
	}
	return kps, desc, next
}

// buildOctave blurs base (already at sigma0) up through intervals+2 more
// layers and differences neighbours into the DoG stack. Layers come from
// pool; release returns them.
func buildOctave(base *grayImage, pool *bufPool) *octave {
	w, h := base.w, base.h
	n := w * h
	oct := &octave{w: w, h: h}
	oct.gauss = make([]*grayImage, intervals+3)
	oct.gauss[0] = base
	tmp := &grayImage{w: w, h: h, pix: pool.get(n)}
	for i := 1; i < len(oct.gauss); i++ {
		prev := sigma0 * math.Pow(2, float64(i-1)/intervals)
		cur := sigma0 * math.Pow(2, float64(i)/intervals)
		oct.gauss[i] = &grayImage{w: w, h: h, pix: pool.get(n)}
		gaussianBlurInto(oct.gauss[i-1], math.Sqrt(cur*cur-prev*prev), oct.gauss[i], tmp)
	}
	pool.put(tmp.pix)
	oct.dog = make([]*grayImage, intervals+2)
	for i := range oct.dog {
		d := &grayImage{w: w, h: h, pix: pool.get(n)}
		a, b := oct.gauss[i+1].pix, oct.gauss[i].pix
		parallelFor(n, func(lo, hi, _ int) {
			for j := lo; j < hi; j++ {
				d.pix[j] = a[j] - b[j]
			}
		})
		oct.dog[i] = d
	}
	return oct
}

func (oct *octave) release(pool *bufPool) {
	for _, g := range oct.gauss[1:] {
		pool.put(g.pix)
	}
	for _, d := range oct.dog {
		pool.put(d.pix)
	}
}

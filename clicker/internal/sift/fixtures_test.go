package sift

import (
	"image"
	"image/color"
	"image/draw"
	"math"
	"math/rand/v2"
)

// shape is a primitive in unit coordinates so one pattern can be rendered
// crisply at any pixel size.
type shape struct {
	kind       int // 0 rect, 1 ellipse, 2 bar
	x, y, w, h float64
	c          color.RGBA
}

type pattern []shape

func newRNG(seed uint64) *rand.Rand {
	return rand.New(rand.NewPCG(seed, seed^0x9e3779b97f4a7c15))
}

var grays = []uint8{0, 40, 90, 150, 210, 255}

// newPattern composes three tiers so the result has structure at every
// octave: a few large high-contrast regions, medium shapes, then fine detail.
func newPattern(seed uint64, n int) pattern {
	rng := newRNG(seed)
	p := make(pattern, 0, n)
	tiers := []struct {
		count    int
		lo, hi   float64
		contrast []uint8
	}{
		{n / 5, 0.16, 0.36, []uint8{0, 60, 190, 255}},
		{n * 3 / 10, 0.07, 0.15, grays},
		{n - n/5 - n*3/10, 0.03, 0.06, grays},
	}
	for _, tier := range tiers {
		for i := 0; i < tier.count; i++ {
			size := tier.lo + rng.Float64()*(tier.hi-tier.lo)
			s := shape{
				kind: rng.IntN(3),
				x:    rng.Float64()*0.95 - 0.05,
				y:    rng.Float64()*0.95 - 0.05,
				w:    size * (0.7 + 0.6*rng.Float64()),
				h:    size * (0.7 + 0.6*rng.Float64()),
			}
			if s.kind == 2 {
				s.h = size * 0.3
				s.w = size * 2
			}
			if rng.IntN(4) == 0 {
				s.c = color.RGBA{uint8(rng.IntN(256)), uint8(rng.IntN(256)), uint8(rng.IntN(256)), 255}
			} else {
				g := tier.contrast[rng.IntN(len(tier.contrast))]
				s.c = color.RGBA{g, g, g, 255}
			}
			p = append(p, s)
		}
	}
	return p
}

// renderPattern rasterizes with 3x3 supersampling, as a browser would
// anti-alias vector art. Later shapes paint over earlier ones.
func renderPattern(p pattern, size int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	fs := float64(size)
	const ss = 3
	bg := [3]float64{128, 128, 128}
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			var acc [3]float64
			for sy := 0; sy < ss; sy++ {
				for sx := 0; sx < ss; sx++ {
					fx := (float64(x) + (float64(sx)+0.5)/ss) / fs
					fy := (float64(y) + (float64(sy)+0.5)/ss) / fs
					c := bg
					for i := len(p) - 1; i >= 0; i-- {
						if p[i].covers(fx, fy) {
							c = [3]float64{float64(p[i].c.R), float64(p[i].c.G), float64(p[i].c.B)}
							break
						}
					}
					for k := range acc {
						acc[k] += c[k]
					}
				}
			}
			img.SetRGBA(x, y, color.RGBA{uint8(acc[0]/(ss*ss) + 0.5), uint8(acc[1]/(ss*ss) + 0.5), uint8(acc[2]/(ss*ss) + 0.5), 255})
		}
	}
	return img
}

func (s shape) covers(fx, fy float64) bool {
	if fx < s.x || fx >= s.x+s.w || fy < s.y || fy >= s.y+s.h {
		return false
	}
	if s.kind != 1 {
		return true
	}
	dx := (fx - s.x - s.w/2) / (s.w / 2)
	dy := (fy - s.y - s.h/2) / (s.h / 2)
	return dx*dx+dy*dy <= 1
}

// newScreen fills a canvas with a two-axis gradient plus light noise.
func newScreen(w, h int, seed uint64) *image.RGBA {
	rng := newRNG(seed)
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			base := 150 + 60*float64(x)/float64(w) - 40*float64(y)/float64(h)
			n := float64(rng.IntN(5) - 2)
			v := uint8(math.Max(0, math.Min(255, base+n)))
			img.SetRGBA(x, y, color.RGBA{v, uint8(math.Max(0, float64(v)-8)), uint8(math.Min(255, float64(v)+6)), 255})
		}
	}
	return img
}

func paste(dst draw.Image, src image.Image, at image.Point) image.Rectangle {
	r := src.Bounds().Sub(src.Bounds().Min).Add(at)
	draw.Draw(dst, r, src, src.Bounds().Min, draw.Src)
	return r
}

func fillRect(dst draw.Image, r image.Rectangle, c color.RGBA) {
	draw.Draw(dst, r, image.NewUniform(c), image.Point{}, draw.Src)
}

// textLines draws rows of short dashes that mimic rendered text.
func textLines(dst draw.Image, r image.Rectangle, seed uint64) {
	rng := newRNG(seed)
	for y := r.Min.Y; y+10 <= r.Max.Y; y += 14 {
		x := r.Min.X
		for x < r.Max.X-4 {
			w := 3 + rng.IntN(12)
			if x+w > r.Max.X {
				break
			}
			g := uint8(20 + rng.IntN(60))
			fillRect(dst, image.Rect(x, y+rng.IntN(3), x+w, y+8+rng.IntN(3)), color.RGBA{g, g, g, 255})
			x += w + 2 + rng.IntN(5)
		}
	}
}

// resample scales by area-averaged supersampling, as a thumbnail tool would.
func resample(src *image.RGBA, w, h int) *image.RGBA {
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	sb := src.Bounds()
	sw, sh := float64(sb.Dx()), float64(sb.Dy())
	ss := max(1, int(math.Ceil(math.Max(sw/float64(w), sh/float64(h)))))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			var acc [4]float64
			for sy := 0; sy < ss; sy++ {
				for sx := 0; sx < ss; sx++ {
					fx := (float64(x) + (float64(sx)+0.5)/float64(ss)) * sw / float64(w)
					fy := (float64(y) + (float64(sy)+0.5)/float64(ss)) * sh / float64(h)
					c := bilinear(src, fx-0.5, fy-0.5)
					for i := range acc {
						acc[i] += float64(c[i])
					}
				}
			}
			n := float64(ss * ss)
			dst.SetRGBA(x, y, color.RGBA{uint8(acc[0]/n + 0.5), uint8(acc[1]/n + 0.5), uint8(acc[2]/n + 0.5), 255})
		}
	}
	return dst
}

func bilinear(src *image.RGBA, fx, fy float64) [4]float64 {
	b := src.Bounds()
	x0 := int(math.Floor(fx))
	y0 := int(math.Floor(fy))
	tx, ty := fx-float64(x0), fy-float64(y0)
	get := func(x, y int) [4]float64 {
		x = min(max(x, 0), b.Dx()-1)
		y = min(max(y, 0), b.Dy()-1)
		i := src.PixOffset(x, y)
		return [4]float64{float64(src.Pix[i]), float64(src.Pix[i+1]), float64(src.Pix[i+2]), float64(src.Pix[i+3])}
	}
	a, bb, c, d := get(x0, y0), get(x0+1, y0), get(x0, y0+1), get(x0+1, y0+1)
	var out [4]float64
	for i := range out {
		out[i] = (a[i]*(1-tx)+bb[i]*tx)*(1-ty) + (c[i]*(1-tx)+d[i]*tx)*ty
	}
	return out
}

// adjust applies v' = alpha*(v-128) + 128 + beta*255 per channel.
func adjust(src *image.RGBA, alpha, beta float64) *image.RGBA {
	dst := image.NewRGBA(src.Bounds())
	for i := 0; i+3 < len(src.Pix); i += 4 {
		for c := 0; c < 3; c++ {
			v := alpha*(float64(src.Pix[i+c])-128) + 128 + beta*255
			dst.Pix[i+c] = uint8(math.Max(0, math.Min(255, v)))
		}
		dst.Pix[i+3] = 255
	}
	return dst
}

func solid(size int, g uint8) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	fillRect(img, img.Bounds(), color.RGBA{g, g, g, 255})
	return img
}

const (
	refSeed   = 11
	refShapes = 80
	screenW   = 1280
	screenH   = 800
)

// decoySlots are screen positions that no test target overlaps.
var decoySlots = []image.Point{{40, 40}, {1130, 40}, {40, 660}, {1130, 660}, {600, 20}}

// scene builds the standard screen: gradient, noise, three decoy icons from
// other seeds, text blocks and a few boxes. scale renders the whole scene at
// a device pixel ratio for the high-DPI case.
func scene(scale int) *image.RGBA {
	w, h := screenW*scale, screenH*scale
	scr := newScreen(w, h, 99)
	for i, pt := range decoySlots {
		d := renderPattern(newPattern(uint64(100+i), refShapes), 96*scale)
		paste(scr, d, pt.Mul(scale))
	}
	fillRect(scr, image.Rect(200*scale, 40*scale, 560*scale, 70*scale), color.RGBA{230, 230, 235, 255})
	fillRect(scr, image.Rect(640*scale, 40*scale, 1060*scale, 44*scale), color.RGBA{40, 40, 60, 255})
	textLines(scr, image.Rect(200*scale, 700*scale, 560*scale, 790*scale), 7)
	textLines(scr, image.Rect(640*scale, 700*scale, 1060*scale, 790*scale), 8)
	return scr
}

func refImage() *image.RGBA { return renderPattern(newPattern(refSeed, refShapes), 96) }

func iou(a, b image.Rectangle) float64 { return rectIoU(a, b) }

func centerDist(m Match, truth image.Rectangle) float64 {
	cx := float64(truth.Min.X+truth.Max.X) / 2
	cy := float64(truth.Min.Y+truth.Max.Y) / 2
	return math.Hypot(float64(m.Center.X)-cx, float64(m.Center.Y)-cy)
}

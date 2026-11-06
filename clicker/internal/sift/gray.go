package sift

import (
	"image"
	"math"
	"runtime"
	"sync"
)

type grayImage struct {
	w, h int
	pix  []float32
}

func newGray(w, h int) *grayImage {
	return &grayImage{w: w, h: h, pix: make([]float32, w*h)}
}

func (g *grayImage) at(x, y int) float32 { return g.pix[y*g.w+x] }

const (
	lumR = 0.299 / 255
	lumG = 0.587 / 255
	lumB = 0.114 / 255
)

// toGray returns luminance in [0,1]. Transparent pixels are composited
// over mid-gray, the least biased guess for an unknown page background.
func toGray(img image.Image) *grayImage {
	b := img.Bounds()
	g := newGray(b.Dx(), b.Dy())
	switch src := img.(type) {
	case *image.RGBA:
		for y := 0; y < g.h; y++ {
			i := src.PixOffset(b.Min.X, b.Min.Y+y)
			row := g.pix[y*g.w : (y+1)*g.w]
			for x := range row {
				p := src.Pix[i : i+4 : i+4]
				row[x] = float32(p[0])*lumR + float32(p[1])*lumG + float32(p[2])*lumB + float32(255-p[3])*(0.5/255)
				i += 4
			}
		}
	case *image.NRGBA:
		for y := 0; y < g.h; y++ {
			i := src.PixOffset(b.Min.X, b.Min.Y+y)
			row := g.pix[y*g.w : (y+1)*g.w]
			for x := range row {
				p := src.Pix[i : i+4 : i+4]
				a := float32(p[3]) / 255
				l := float32(p[0])*lumR + float32(p[1])*lumG + float32(p[2])*lumB
				row[x] = l*a + (1-a)*0.5
				i += 4
			}
		}
	case *image.Gray:
		for y := 0; y < g.h; y++ {
			i := src.PixOffset(b.Min.X, b.Min.Y+y)
			row := g.pix[y*g.w : (y+1)*g.w]
			for x := range row {
				row[x] = float32(src.Pix[i]) / 255
				i++
			}
		}
	default:
		for y := 0; y < g.h; y++ {
			row := g.pix[y*g.w : (y+1)*g.w]
			for x := range row {
				r, gg, bb, a := img.At(b.Min.X+x, b.Min.Y+y).RGBA()
				row[x] = (float32(r)*0.299+float32(gg)*0.587+float32(bb)*0.114)/65535 + float32(65535-a)*(0.5/65535)
			}
		}
	}
	return g
}

// normalizeFn is the contrast normalization applied before the pyramid.
// Tests swap it to compare candidates; see README.
var normalizeFn = stretchGlobal

// stretchGlobal maps the 0.5th..99.5th percentiles onto [0,1].
func stretchGlobal(g *grayImage) {
	const bins = 1024
	var hist [bins]int
	for _, v := range g.pix {
		i := int(v*(bins-1) + 0.5)
		if i < 0 {
			i = 0
		} else if i >= bins {
			i = bins - 1
		}
		hist[i]++
	}
	n := len(g.pix)
	lo := percentileBin(hist[:], n*5/1000)
	hi := percentileBin(hist[:], n*995/1000)
	if hi-lo < 0.02 {
		return
	}
	scale := 1 / (hi - lo)
	for i, v := range g.pix {
		v = (v - lo) * scale
		if v < 0 {
			v = 0
		} else if v > 1 {
			v = 1
		}
		g.pix[i] = v
	}
}

func percentileBin(hist []int, rank int) float32 {
	acc := 0
	for i, c := range hist {
		acc += c
		if acc > rank {
			return float32(i) / float32(len(hist)-1)
		}
	}
	return 1
}

// normalizeLocal divides each pixel's deviation from its neighbourhood mean
// by the neighbourhood standard deviation, floored so flat areas stay flat.
func normalizeLocal(g *grayImage) {
	const (
		sigma = 24.0
		floor = 0.10
		gain  = 0.25
	)
	mean := gaussianBlur(g, sigma)
	sq := newGray(g.w, g.h)
	for i, v := range g.pix {
		sq.pix[i] = v * v
	}
	sq = gaussianBlur(sq, sigma)
	for i, v := range g.pix {
		m := mean.pix[i]
		variance := sq.pix[i] - m*m
		if variance < 0 {
			variance = 0
		}
		o := (v-m)/(float32(math.Sqrt(float64(variance)))+floor)*gain + 0.5
		if o < 0 {
			o = 0
		} else if o > 1 {
			o = 1
		}
		g.pix[i] = o
	}
}

// parallelFor splits [0,n) into contiguous chunks, one per worker, and runs
// fn on each. The chunk index lets callers collect results in a fixed order.
func parallelFor(n int, fn func(lo, hi, chunk int)) {
	if n <= 0 {
		return
	}
	workers := runtime.GOMAXPROCS(0)
	if workers > n {
		workers = n
	}
	if workers <= 1 {
		fn(0, n, 0)
		return
	}
	size := (n + workers - 1) / workers
	var wg sync.WaitGroup
	for c := 0; c*size < n; c++ {
		lo := c * size
		hi := min(lo+size, n)
		wg.Add(1)
		go func(lo, hi, c int) {
			defer wg.Done()
			fn(lo, hi, c)
		}(lo, hi, c)
	}
	wg.Wait()
}

func numChunks(n int) int {
	if n <= 0 {
		return 0
	}
	workers := min(runtime.GOMAXPROCS(0), n)
	if workers <= 1 {
		return 1
	}
	size := (n + workers - 1) / workers
	return (n + size - 1) / size
}

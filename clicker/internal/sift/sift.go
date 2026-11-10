// Package sift locates every occurrence of a reference image inside a
// screenshot with SIFT features (Lowe 2004) verified by a multi-instance
// similarity RANSAC. It is pure Go and depends only on the standard library.
package sift

import (
	"errors"
	"image"
	"math"
	"sort"
	"time"
)

// Options tunes FindAll. Zero fields take the defaults listed per field.
type Options struct {
	MaxFeatures  int     // keypoint cap per image, strongest first; 0 = 10000
	Ratio        float64 // Lowe ratio threshold; 0 = 0.75
	MinInliers   int     // RANSAC inliers needed to accept an instance; 0 = 8
	MaxInstances int     // cap on returned instances; 0 = 10
	MaxPixels    int     // decoded pixel budget per image; 0 = 16_000_000; above it → ErrTooLarge
	Upsample     bool    // double a small reference, and the screen, before building their pyramids (Lowe's -1 octave)
}

// DefaultOptions returns the defaults FindAll uses for zero fields, with
// Upsample on.
func DefaultOptions() Options {
	return Options{
		MaxFeatures:  10000,
		Ratio:        0.75,
		MinInliers:   8,
		MaxInstances: 10,
		MaxPixels:    16_000_000,
		Upsample:     true,
	}
}

func (o Options) withDefaults() Options {
	d := DefaultOptions()
	if o.MaxFeatures <= 0 {
		o.MaxFeatures = d.MaxFeatures
	}
	if o.Ratio <= 0 || o.Ratio >= 1 {
		o.Ratio = d.Ratio
	}
	if o.MinInliers < 4 {
		if o.MinInliers <= 0 {
			o.MinInliers = d.MinInliers
		} else {
			o.MinInliers = 4
		}
	}
	if o.MaxInstances <= 0 {
		o.MaxInstances = d.MaxInstances
	}
	if o.MaxPixels <= 0 {
		o.MaxPixels = d.MaxPixels
	}
	return o
}

// Match is one verified instance of the reference on the screen.
type Match struct {
	Box      image.Rectangle // instance bounds in screen pixel coordinates, clipped to the screen
	Center   image.Point
	Inliers  int     // RANSAC inliers supporting this instance
	Matched  int     // ratio-test survivors assigned to this instance before RANSAC
	Scale    float64 // estimated reference→screen scale
	Rotation float64 // radians
	Score    float64 // Inliers / reference keypoints. Raw support, NOT a probability.
}

// Stats reports work done by one FindAll call.
type Stats struct {
	RefKeypoints, ScreenKeypoints, RatioMatches int
	Elapsed                                     time.Duration
}

var (
	ErrTooLarge   = errors.New("sift: image exceeds the pixel budget")
	ErrTooSmall   = errors.New("sift: reference image is smaller than 8x8")
	ErrNoFeatures = errors.New("sift: reference image is too small or flat to match")
)

const (
	minRefDim      = 8
	upsampleMaxDim = 256
	refPad         = 24
	// Screens above this are left at native resolution: they are either
	// high-DPI captures, which already hold the fine scales, or so large
	// that the doubled first octave would dominate latency.
	upsampleMaxScreenPixels = 8_500_000
)

// FindAll returns every verified instance of ref in screen, strongest first.
// No instance is an empty slice with a nil error.
func FindAll(ref, screen image.Image, opts Options) ([]Match, Stats, error) {
	start := time.Now()
	o := opts.withDefaults()
	var st Stats
	if ref == nil || screen == nil {
		return nil, st, errors.New("sift: nil image")
	}
	rb, sb := ref.Bounds(), screen.Bounds()
	if rb.Dx() < minRefDim || rb.Dy() < minRefDim {
		return nil, st, ErrTooSmall
	}
	if rb.Dx()*rb.Dy() > o.MaxPixels || sb.Dx()*sb.Dy() > o.MaxPixels {
		return nil, st, ErrTooLarge
	}

	up := o.Upsample && min(rb.Dx(), rb.Dy()) <= upsampleMaxDim
	refF := extract(toGray(ref), o.MaxFeatures, up, refPad)
	st.RefKeypoints = len(refF.kps)
	if len(refF.kps) < o.MinInliers {
		st.Elapsed = time.Since(start)
		return nil, st, ErrNoFeatures
	}

	out := []Match{}
	if sb.Dx() < minRefDim || sb.Dy() < minRefDim {
		st.Elapsed = time.Since(start)
		return out, st, nil
	}
	scrF := extract(toGray(screen), o.MaxFeatures, o.Upsample && sb.Dx()*sb.Dy() <= upsampleMaxScreenPixels, 0)
	st.ScreenKeypoints = len(scrF.kps)
	if len(scrF.kps) == 0 {
		st.Elapsed = time.Since(start)
		return out, st, nil
	}

	ms := matchDescriptors(refF, scrF, o.Ratio, o.MaxInstances+1)
	st.RatioMatches = len(ms)
	for _, inst := range findInstances(ms, refF, scrF, rb.Dx(), rb.Dy(), sb, o) {
		box := inst.box.Add(sb.Min).Intersect(sb)
		out = append(out, Match{
			Box:      box,
			Center:   image.Pt((box.Min.X+box.Max.X)/2, (box.Min.Y+box.Max.Y)/2),
			Inliers:  inst.nInliers,
			Matched:  inst.matched,
			Scale:    inst.s.scale(),
			Rotation: inst.s.rotation(),
			Score:    float64(inst.nInliers) / float64(len(refF.kps)),
		})
	}
	sort.Slice(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if a.Inliers != b.Inliers {
			return a.Inliers > b.Inliers
		}
		if a.Box.Min.Y != b.Box.Min.Y {
			return a.Box.Min.Y < b.Box.Min.Y
		}
		return a.Box.Min.X < b.Box.Min.X
	})
	st.Elapsed = time.Since(start)
	return out, st, nil
}

// Find returns the best instance, or nil with a nil error when there is none.
func Find(ref, screen image.Image, opts Options) (*Match, Stats, error) {
	all, st, err := FindAll(ref, screen, opts)
	if err != nil || len(all) == 0 {
		return nil, st, err
	}
	m := all[0]
	return &m, st, nil
}

// extract builds the pyramid and returns oriented, described keypoints in
// the image's own pixel coordinates. pad is edge-replicated around the
// image first; keypoints that land in the padding are dropped.
func extract(g *grayImage, maxFeatures int, upsample bool, pad int) *features {
	normalizeFn(g)
	w, h := g.w, g.h
	if pad > 0 {
		g = padReplicate(g, pad)
	}
	coordScale := float32(1)
	if upsample {
		coordScale = 0.5
	}
	var pool bufPool
	bw, bh, src := initialSource(g, upsample, &pool)

	f := &features{}
	nOct := numOctaves(bw, bh)
	var base *grayImage
	for o := 0; o < nOct; o++ {
		kps, desc, next := processOctave(bw, bh, src, coordScale, o+1 < nOct, &pool)
		if base != nil {
			pool.put(base.pix)
		}
		base = next
		for i := range kps {
			k := &kps[i]
			k.x -= float32(pad)
			k.y -= float32(pad)
			if k.x < 0 || k.y < 0 || k.x >= float32(w) || k.y >= float32(h) {
				continue
			}
			f.kps = append(f.kps, *k)
			f.desc = append(f.desc, desc[i*descLen:(i+1)*descLen]...)
		}
		if next != nil {
			bw, bh, src = next.w, next.h, sliceSource(next)
		}
		coordScale *= 2
	}
	sortByResponse(f)
	if len(f.kps) > maxFeatures {
		f.kps = f.kps[:maxFeatures]
		f.desc = f.desc[:maxFeatures*descLen]
	}
	f.minX, f.minY = float32(math.Inf(1)), float32(math.Inf(1))
	f.maxX, f.maxY = float32(math.Inf(-1)), float32(math.Inf(-1))
	for _, k := range f.kps {
		f.minX, f.maxX = min(f.minX, k.x), max(f.maxX, k.x)
		f.minY, f.maxY = min(f.minY, k.y), max(f.maxY, k.y)
	}
	return f
}

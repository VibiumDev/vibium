package sift

import (
	"math"
	"sort"
)

const (
	imgBorder   = 5
	contrastThr = 0.04
	edgeThr     = 10.0
	maxRefine   = 5

	oriBins      = 36
	oriSigFactor = 1.5
	oriRadiusF   = 3.0
	oriPeakRatio = 0.8
)

type keypoint struct {
	x, y     float32 // original image pixels
	scale    float32 // sigma in original image pixels
	angle    float32 // radians, [0, 2pi)
	response float32
}

// candidate is a refined DoG extremum in octave coordinates.
type candidate struct {
	layer    int
	ox, oy   float32
	oscale   float32 // sigma within the octave
	response float32
}

type features struct {
	kps  []keypoint
	desc []float32 // len(kps)*descLen
	// Keypoint extent in reference pixels, used by the spread test.
	minX, minY, maxX, maxY float32
}

// detectCandidates scans rows near [ownLo, ownHi) and keeps the extrema
// whose refined row lands inside that range.
func detectCandidates(oct *octave, ownLo, ownHi int) []candidate {
	w, h := oct.w, oct.h
	scanLo := max(imgBorder, ownLo-refineWalk)
	scanHi := min(h-imgBorder, ownHi+refineWalk)
	rows := scanHi - scanLo
	if rows <= 0 || w-2*imgBorder <= 0 {
		return nil
	}
	preThr := float32(0.5 * contrastThr / intervals)
	out := make([][]candidate, numChunks(rows))
	parallelFor(rows, func(lo, hi, chunk int) {
		var found []candidate
		for layer := 1; layer <= intervals; layer++ {
			cur, prev, next := oct.dog[layer].pix, oct.dog[layer-1].pix, oct.dog[layer+1].pix
			for r := lo + scanLo; r < hi+scanLo; r++ {
				for c := imgBorder; c < w-imgBorder; c++ {
					i := r*w + c
					v := cur[i]
					if v > -preThr && v < preThr {
						continue
					}
					if !isExtremum(v, cur, prev, next, i, w) {
						continue
					}
					cand, ok := refine(oct, layer, r, c)
					if !ok {
						continue
					}
					if row := int(math.Floor(float64(cand.oy))); row < ownLo || row >= ownHi {
						continue
					}
					found = append(found, cand)
				}
			}
		}
		out[chunk] = found
	})
	var all []candidate
	for _, f := range out {
		all = append(all, f...)
	}
	return all
}

func isExtremum(v float32, cur, prev, next []float32, i, w int) bool {
	if v > 0 {
		for _, s := range [3][]float32{prev, cur, next} {
			for _, o := range [3]int{-w, 0, w} {
				j := i + o
				if v < s[j-1] || v < s[j] || v < s[j+1] {
					return false
				}
			}
		}
		return true
	}
	for _, s := range [3][]float32{prev, cur, next} {
		for _, o := range [3]int{-w, 0, w} {
			j := i + o
			if v > s[j-1] || v > s[j] || v > s[j+1] {
				return false
			}
		}
	}
	return true
}

// refine interpolates the extremum to sub-pixel position and scale
// (Lowe sec. 4) and rejects low-contrast and edge responses.
func refine(oct *octave, layer, r, c int) (candidate, bool) {
	w, h := oct.w, oct.h
	var xc, xr, xi float32
	var dx, dy, ds, dxx, dyy, dxy float32
	var converged bool
	for it := 0; it < maxRefine; it++ {
		cur, prev, next := oct.dog[layer].pix, oct.dog[layer-1].pix, oct.dog[layer+1].pix
		i := r*w + c
		v := cur[i]
		dx = (cur[i+1] - cur[i-1]) * 0.5
		dy = (cur[i+w] - cur[i-w]) * 0.5
		ds = (next[i] - prev[i]) * 0.5
		dxx = cur[i+1] - 2*v + cur[i-1]
		dyy = cur[i+w] - 2*v + cur[i-w]
		dss := next[i] - 2*v + prev[i]
		dxy = (cur[i+w+1] - cur[i+w-1] - cur[i-w+1] + cur[i-w-1]) * 0.25
		dxs := (next[i+1] - next[i-1] - prev[i+1] + prev[i-1]) * 0.25
		dys := (next[i+w] - next[i-w] - prev[i+w] + prev[i-w]) * 0.25

		var ok bool
		xc, xr, xi, ok = solve3(dxx, dxy, dxs, dyy, dys, dss, -dx, -dy, -ds)
		if !ok {
			return candidate{}, false
		}
		if abs32(xc) < 0.5 && abs32(xr) < 0.5 && abs32(xi) < 0.5 {
			converged = true
			break
		}
		if abs32(xc) > 1e6 || abs32(xr) > 1e6 || abs32(xi) > 1e6 {
			return candidate{}, false
		}
		c += int(math.Round(float64(xc)))
		r += int(math.Round(float64(xr)))
		layer += int(math.Round(float64(xi)))
		if layer < 1 || layer > intervals || c < imgBorder || c >= w-imgBorder || r < imgBorder || r >= h-imgBorder {
			return candidate{}, false
		}
	}
	if !converged {
		return candidate{}, false
	}
	v := oct.dog[layer].pix[r*w+c]
	contr := v + 0.5*(dx*xc+dy*xr+ds*xi)
	if abs32(contr)*intervals < contrastThr {
		return candidate{}, false
	}
	tr := dxx + dyy
	det := dxx*dyy - dxy*dxy
	if det <= 0 || tr*tr*edgeThr >= (edgeThr+1)*(edgeThr+1)*det {
		return candidate{}, false
	}
	return candidate{
		layer:    layer,
		ox:       float32(c) + xc,
		oy:       float32(r) + xr,
		oscale:   float32(sigma0 * math.Pow(2, float64(float32(layer)+xi)/intervals)),
		response: abs32(contr),
	}, true
}

// solve3 solves the symmetric system [a b c; b d e; c e f] x = rhs by Cramer's rule.
func solve3(a, b, c, d, e, f, r0, r1, r2 float32) (x, y, z float32, ok bool) {
	det := a*(d*f-e*e) - b*(b*f-c*e) + c*(b*e-c*d)
	if abs32(det) < 1e-12 {
		return 0, 0, 0, false
	}
	inv := 1 / det
	x = (r0*(d*f-e*e) - b*(r1*f-r2*e) + c*(r1*e-r2*d)) * inv
	y = (a*(r1*f-r2*e) - r0*(b*f-c*e) + c*(b*r2-c*r1)) * inv
	z = (a*(d*r2-e*r1) - b*(b*r2-c*r1) + r0*(b*e-c*d)) * inv
	return x, y, z, true
}

func abs32(v float32) float32 {
	if v < 0 {
		return -v
	}
	return v
}

// orientations returns the dominant gradient directions around a candidate
// from a 36-bin histogram; every peak within 80% of the maximum counts.
func orientations(img *grayImage, cand candidate, hist *[oriBins]float32) []float32 {
	for i := range hist {
		hist[i] = 0
	}
	cx := int(math.Round(float64(cand.ox)))
	cy := int(math.Round(float64(cand.oy)))
	sig := oriSigFactor * cand.oscale
	radius := int(math.Round(float64(oriRadiusF * sig)))
	expScale := -1 / (2 * sig * sig)
	w, h := img.w, img.h
	const binScale = oriBins / (2 * math.Pi)
	for i := -radius; i <= radius; i++ {
		y := cy + i
		if y <= 0 || y >= h-1 {
			continue
		}
		wy := float32(math.Exp(float64(float32(i*i) * expScale)))
		for j := -radius; j <= radius; j++ {
			x := cx + j
			if x <= 0 || x >= w-1 {
				continue
			}
			gx := img.pix[y*w+x+1] - img.pix[y*w+x-1]
			gy := img.pix[(y-1)*w+x] - img.pix[(y+1)*w+x]
			mag := float32(math.Sqrt(float64(gx*gx + gy*gy)))
			wgt := wy * float32(math.Exp(float64(float32(j*j)*expScale)))
			bin := int(math.Round(float64(fastAtan2(gy, gx) * binScale)))
			if bin >= oriBins {
				bin -= oriBins
			}
			hist[bin] += wgt * mag
		}
	}
	var smooth [oriBins]float32
	for i := range smooth {
		p2, p1 := hist[(i-2+oriBins)%oriBins], hist[(i-1+oriBins)%oriBins]
		n1, n2 := hist[(i+1)%oriBins], hist[(i+2)%oriBins]
		smooth[i] = (p2+n2)*(1.0/16) + (p1+n1)*(4.0/16) + hist[i]*(6.0/16)
	}
	var peak float32
	for _, v := range smooth {
		peak = max(peak, v)
	}
	if peak <= 0 {
		return nil
	}
	var angles []float32
	thr := peak * oriPeakRatio
	for i := range smooth {
		l, r := smooth[(i-1+oriBins)%oriBins], smooth[(i+1)%oriBins]
		v := smooth[i]
		if v <= l || v <= r || v < thr {
			continue
		}
		den := l - 2*v + r
		binf := float32(i)
		if den != 0 {
			binf += 0.5 * (l - r) / den
		}
		for binf < 0 {
			binf += oriBins
		}
		for binf >= oriBins {
			binf -= oriBins
		}
		angles = append(angles, binf*(2*math.Pi/oriBins))
	}
	return angles
}

// fastAtan2 returns atan2 in [0, 2pi) with error under 0.3 degrees, which is
// far below the orientation and descriptor bin widths.
func fastAtan2(y, x float32) float32 {
	const (
		p1 = 0.9997878412794807
		p3 = -0.3258083974640975
		p5 = 0.1555786518463281
		p7 = -0.04432655554792128
	)
	ax, ay := abs32(x), abs32(y)
	var a float32
	if ax >= ay {
		c := ay / (ax + 1e-20)
		c2 := c * c
		a = (((p7*c2+p5)*c2+p3)*c2 + p1) * c
	} else {
		c := ax / (ay + 1e-20)
		c2 := c * c
		a = math.Pi/2 - (((p7*c2+p5)*c2+p3)*c2+p1)*c
	}
	if x < 0 {
		a = math.Pi - a
	}
	if y < 0 {
		a = 2*math.Pi - a
	}
	if a >= 2*math.Pi {
		a -= 2 * math.Pi
	}
	return a
}

// sortByResponse orders keypoints strongest first with a total tie-break so
// the cap and the output order never depend on goroutine scheduling.
func sortByResponse(f *features) {
	n := len(f.kps)
	idx := make([]int, n)
	for i := range idx {
		idx[i] = i
	}
	kps := f.kps
	sort.Slice(idx, func(a, b int) bool {
		ka, kb := kps[idx[a]], kps[idx[b]]
		if ka.response != kb.response {
			return ka.response > kb.response
		}
		if ka.y != kb.y {
			return ka.y < kb.y
		}
		if ka.x != kb.x {
			return ka.x < kb.x
		}
		if ka.scale != kb.scale {
			return ka.scale < kb.scale
		}
		return ka.angle < kb.angle
	})
	nk := make([]keypoint, n)
	nd := make([]float32, n*descLen)
	for i, j := range idx {
		nk[i] = kps[j]
		copy(nd[i*descLen:(i+1)*descLen], f.desc[j*descLen:(j+1)*descLen])
	}
	f.kps, f.desc = nk, nd
}

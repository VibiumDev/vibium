package sift

import "math"

const (
	descWidth    = 4 // histograms per side
	descOriBins  = 8
	descLen      = descWidth * descWidth * descOriBins
	descBinScale = 3.0 // histogram bin width in units of keypoint sigma
	descClamp    = 0.2
)

// describe fills out with the 128-D descriptor of a keypoint at octave
// position (ox, oy), scale oscale and orientation angle (Lowe sec. 6).
func describe(img *grayImage, ox, oy, oscale, angle float32, out []float32, hist []float32) {
	const (
		d = descWidth
		n = descOriBins
	)
	for i := range hist {
		hist[i] = 0
	}
	cx := int(math.Round(float64(ox)))
	cy := int(math.Round(float64(oy)))
	histWidth := descBinScale * oscale
	cosT := float32(math.Cos(float64(angle))) / histWidth
	sinT := float32(math.Sin(float64(angle))) / histWidth
	radius := int(math.Round(float64(histWidth * math.Sqrt2 * (d + 1) * 0.5)))
	w, h := img.w, img.h
	radius = min(radius, int(math.Sqrt(float64(w*w+h*h))))
	expScale := -1 / (d * d * 0.5) / (histWidth * histWidth)
	const binScale = n / (2 * math.Pi)

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
			fj, fi := float32(j), float32(i)
			cRot := fj*cosT - fi*sinT
			rRot := fj*sinT + fi*cosT
			rbin := rRot + d/2 - 0.5
			cbin := cRot + d/2 - 0.5
			if rbin <= -1 || rbin >= d || cbin <= -1 || cbin >= d {
				continue
			}
			gx := img.pix[y*w+x+1] - img.pix[y*w+x-1]
			gy := img.pix[(y-1)*w+x] - img.pix[(y+1)*w+x]
			ori := fastAtan2(gy, gx) - angle
			for ori < 0 {
				ori += 2 * math.Pi
			}
			for ori >= 2*math.Pi {
				ori -= 2 * math.Pi
			}
			mag := float32(math.Sqrt(float64(gx*gx+gy*gy))) * wy * float32(math.Exp(float64(float32(j*j)*expScale)))
			obin := ori * binScale

			r0 := int(math.Floor(float64(rbin)))
			c0 := int(math.Floor(float64(cbin)))
			o0 := int(math.Floor(float64(obin)))
			rbin -= float32(r0)
			cbin -= float32(c0)
			obin -= float32(o0)
			if o0 < 0 {
				o0 += n
			} else if o0 >= n {
				o0 -= n
			}

			v1 := mag * rbin
			v0 := mag - v1
			v11 := v1 * cbin
			v10 := v1 - v11
			v01 := v0 * cbin
			v00 := v0 - v01
			v111 := v11 * obin
			v110 := v11 - v111
			v101 := v10 * obin
			v100 := v10 - v101
			v011 := v01 * obin
			v010 := v01 - v011
			v001 := v00 * obin
			v000 := v00 - v001

			// hist is (d+2)x(d+2)x(n+2); the extra ring absorbs out-of-range
			// neighbours and the two extra orientation slots wrap below.
			idx := ((r0+1)*(d+2)+(c0+1))*(n+2) + o0
			hist[idx] += v000
			hist[idx+1] += v001
			hist[idx+(n+2)] += v010
			hist[idx+(n+3)] += v011
			hist[idx+(d+2)*(n+2)] += v100
			hist[idx+(d+2)*(n+2)+1] += v101
			hist[idx+(d+3)*(n+2)] += v110
			hist[idx+(d+3)*(n+2)+1] += v111
		}
	}

	k := 0
	for i := 0; i < d; i++ {
		for j := 0; j < d; j++ {
			base := ((i+1)*(d+2) + (j + 1)) * (n + 2)
			hist[base] += hist[base+n]
			hist[base+1] += hist[base+n+1]
			for o := 0; o < n; o++ {
				out[k] = hist[base+o]
				k++
			}
		}
	}

	var nrm float32
	for _, v := range out {
		nrm += v * v
	}
	thr := float32(math.Sqrt(float64(nrm))) * descClamp
	nrm = 0
	for i, v := range out {
		if v > thr {
			v = thr
			out[i] = v
		}
		nrm += v * v
	}
	if nrm > 0 {
		inv := 1 / float32(math.Sqrt(float64(nrm)))
		for i := range out {
			out[i] *= inv
		}
	}
}

const descHistLen = (descWidth + 2) * (descWidth + 2) * (descOriBins + 2)

// describeOctave assigns orientations and descriptors to every candidate of
// an octave. coordScale maps octave pixels to reference pixels.
func describeOctave(oct *octave, cands []candidate, coordScale float32) ([]keypoint, []float32) {
	type part struct {
		kps  []keypoint
		desc []float32
	}
	parts := make([]part, numChunks(len(cands)))
	parallelFor(len(cands), func(lo, hi, chunk int) {
		var p part
		var ohist [oriBins]float32
		dhist := make([]float32, descHistLen)
		for _, c := range cands[lo:hi] {
			img := oct.gauss[c.layer]
			for _, ang := range orientations(img, c, &ohist) {
				p.kps = append(p.kps, keypoint{
					x:        c.ox * coordScale,
					y:        c.oy * coordScale,
					scale:    c.oscale * coordScale,
					angle:    ang,
					response: c.response,
				})
				start := len(p.desc)
				p.desc = append(p.desc, make([]float32, descLen)...)
				describe(img, c.ox, c.oy, c.oscale, ang, p.desc[start:start+descLen], dhist)
			}
		}
		parts[chunk] = p
	})
	var kps []keypoint
	var desc []float32
	for _, p := range parts {
		kps = append(kps, p.kps...)
		desc = append(desc, p.desc...)
	}
	return kps, desc
}

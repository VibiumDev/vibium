package sift

import "math"

type match struct {
	ref, scr int32
	dist2    float32
}

const maxKNN = 11

// matchDescriptors finds, for every reference descriptor, its k nearest
// screen descriptors and keeps the smallest prefix separated from the next
// neighbour by the Lowe ratio. With k=2 this is the classic ratio test.
// Larger k lets a descriptor match several identical copies, which the plain
// test rejects because the first two neighbours are then equally close.
func matchDescriptors(ref, scr *features, ratio float64, k int) []match {
	nr, ns := len(ref.kps), len(scr.kps)
	if nr == 0 || ns == 0 {
		return nil
	}
	k = max(2, min(k, maxKNN))
	ratio2 := float32(ratio * ratio)
	scrNorm := norms(scr.desc, ns)
	refNorm := norms(ref.desc, nr)
	parts := make([][]match, numChunks(nr))
	parallelFor(nr, func(lo, hi, chunk int) {
		var out []match
		var best [maxKNN]float32
		var bestIdx [maxKNN]int32
		for i := lo; i < hi; i++ {
			a := ref.desc[i*descLen : (i+1)*descLen]
			na := refNorm[i]
			for j := range k {
				best[j] = float32(math.Inf(1))
				bestIdx[j] = -1
			}
			for j := 0; j < ns; j++ {
				d := na + scrNorm[j] - 2*dot128(a, scr.desc[j*descLen:(j+1)*descLen])
				if d >= best[k-1] {
					continue
				}
				if d < 0 {
					d = 0
				}
				p := k - 1
				for p > 0 && best[p-1] > d {
					best[p] = best[p-1]
					bestIdx[p] = bestIdx[p-1]
					p--
				}
				best[p] = d
				bestIdx[p] = int32(j)
			}
			// Accept the prefix that ends at the first gap and extends
			// through every following neighbour that is itself
			// distinctive against its successor. An exact copy (d≈0)
			// must not hide a second, degraded copy behind it.
			m := 0
			for j := 1; j < k && bestIdx[j-1] >= 0; j++ {
				if best[j-1] < ratio2*best[j] {
					m = j
				} else if m > 0 {
					break
				}
			}
			for q := 0; q < m; q++ {
				out = append(out, match{ref: int32(i), scr: bestIdx[q], dist2: best[q]})
			}
		}
		parts[chunk] = out
	})
	var all []match
	for _, p := range parts {
		all = append(all, p...)
	}
	return all
}

func norms(desc []float32, n int) []float32 {
	out := make([]float32, n)
	for i := range out {
		d := desc[i*descLen : (i+1)*descLen]
		out[i] = dot128(d, d)
	}
	return out
}

func dot128(a, b []float32) float32 {
	a = a[:descLen]
	b = b[:descLen]
	var s0, s1, s2, s3 float32
	for i := 0; i < descLen; i += 4 {
		s0 += a[i] * b[i]
		s1 += a[i+1] * b[i+1]
		s2 += a[i+2] * b[i+2]
		s3 += a[i+3] * b[i+3]
	}
	return (s0 + s1) + (s2 + s3)
}

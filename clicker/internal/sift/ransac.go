package sift

import (
	"image"
	"math"
	"math/rand/v2"
)

const (
	ransacMaxIter   = 2000
	ransacMinIter   = 100
	ransacConf      = 0.99
	inlierTolPx     = 4.0
	minScale        = 0.2
	maxScale        = 5.0
	scaleTolFactor  = 2.0
	angleTol        = math.Pi / 4
	minSpread       = 0.25
	minVisible      = 0.5
	minBoxSide      = 4.0
	maxOverlapIoU   = 0.5
	minSampleDistPx = 1.5
)

// similarity maps reference pixels to screen pixels:
// x' = a*x - b*y + tx, y' = b*x + a*y + ty.
type similarity struct{ a, b, tx, ty float64 }

func (s similarity) scale() float64    { return math.Hypot(s.a, s.b) }
func (s similarity) rotation() float64 { return math.Atan2(s.b, s.a) }

func (s similarity) apply(x, y float64) (float64, float64) {
	return s.a*x - s.b*y + s.tx, s.b*x + s.a*y + s.ty
}

// corr is a ratio-test match flattened for geometry.
type corr struct {
	rx, ry, sx, sy float64
	logScale       float64 // ln(screen sigma / ref sigma)
	dAngle         float64 // screen angle - ref angle, wrapped
}

func similarityFrom2(p, q corr) (similarity, bool) {
	dpx, dpy := q.rx-p.rx, q.ry-p.ry
	dqx, dqy := q.sx-p.sx, q.sy-p.sy
	den := dpx*dpx + dpy*dpy
	if den < minSampleDistPx*minSampleDistPx {
		return similarity{}, false
	}
	a := (dqx*dpx + dqy*dpy) / den
	b := (dqy*dpx - dqx*dpy) / den
	return similarity{a: a, b: b, tx: p.sx - (a*p.rx - b*p.ry), ty: p.sy - (b*p.rx + a*p.ry)}, true
}

func wrapAngle(a float64) float64 {
	for a > math.Pi {
		a -= 2 * math.Pi
	}
	for a <= -math.Pi {
		a += 2 * math.Pi
	}
	return a
}

// inliers returns the indices of pool entries consistent with s in
// position, keypoint scale and keypoint orientation.
func inliers(pool []corr, s similarity, out []int) []int {
	out = out[:0]
	sc := s.scale()
	if sc < minScale || sc > maxScale {
		return out
	}
	tol := inlierTolPx * math.Max(1, sc)
	tol2 := tol * tol
	logSc := math.Log(sc)
	rot := s.rotation()
	logTol := math.Log(scaleTolFactor)
	for i, c := range pool {
		px, py := s.apply(c.rx, c.ry)
		dx, dy := px-c.sx, py-c.sy
		if dx*dx+dy*dy > tol2 {
			continue
		}
		if math.Abs(c.logScale-logSc) > logTol {
			continue
		}
		if math.Abs(wrapAngle(c.dAngle-rot)) > angleTol {
			continue
		}
		out = append(out, i)
	}
	return out
}

func leastSquares(pool []corr, idx []int) (similarity, bool) {
	if len(idx) < 2 {
		return similarity{}, false
	}
	var mrx, mry, msx, msy float64
	for _, i := range idx {
		c := pool[i]
		mrx += c.rx
		mry += c.ry
		msx += c.sx
		msy += c.sy
	}
	n := float64(len(idx))
	mrx, mry, msx, msy = mrx/n, mry/n, msx/n, msy/n
	var num, numB, den float64
	for _, i := range idx {
		c := pool[i]
		px, py := c.rx-mrx, c.ry-mry
		qx, qy := c.sx-msx, c.sy-msy
		num += px*qx + py*qy
		numB += px*qy - py*qx
		den += px*px + py*py
	}
	if den < 1e-9 {
		return similarity{}, false
	}
	a, b := num/den, numB/den
	return similarity{a: a, b: b, tx: msx - (a*mrx - b*mry), ty: msy - (b*mrx + a*mry)}, true
}

// ransac finds the similarity with the largest consensus in pool and
// refines it by least squares on its inliers.
func ransac(pool []corr, rng *rand.Rand) (similarity, []int) {
	n := len(pool)
	if n < 2 {
		return similarity{}, nil
	}
	var best similarity
	bestCount := 0
	iters := ransacMaxIter
	buf := make([]int, 0, n)
	for it := 0; it < iters; it++ {
		i := rng.IntN(n)
		j := rng.IntN(n - 1)
		if j >= i {
			j++
		}
		s, ok := similarityFrom2(pool[i], pool[j])
		if !ok {
			continue
		}
		buf = inliers(pool, s, buf)
		if len(buf) > bestCount {
			bestCount = len(buf)
			best = s
			w := float64(bestCount) / float64(n)
			if pFail := 1 - w*w; pFail > 0 && pFail < 1 {
				need := int(math.Ceil(math.Log(1-ransacConf) / math.Log(pFail)))
				iters = min(iters, max(need, ransacMinIter))
			}
		}
	}
	if bestCount < 2 {
		return similarity{}, nil
	}
	idx := inliers(pool, best, nil)
	for range 3 {
		s, ok := leastSquares(pool, idx)
		if !ok {
			break
		}
		next := inliers(pool, s, nil)
		if len(next) < len(idx) {
			break
		}
		best, idx = s, next
	}
	return best, idx
}

// instance is an accepted RANSAC result before conversion to a Match.
type instance struct {
	s        similarity
	box      image.Rectangle // unclipped, screen pixels relative to the screen origin
	nInliers int
	matched  int
}

// projectBox returns the axis-aligned bounds of the reference rectangle
// under s, in continuous screen coordinates.
func projectBox(s similarity, w, h float64) (minX, minY, maxX, maxY float64) {
	minX, minY = math.Inf(1), math.Inf(1)
	maxX, maxY = math.Inf(-1), math.Inf(-1)
	for _, c := range [4][2]float64{{0, 0}, {w, 0}, {0, h}, {w, h}} {
		x, y := s.apply(c[0], c[1])
		minX, maxX = math.Min(minX, x), math.Max(maxX, x)
		minY, maxY = math.Min(minY, y), math.Max(maxY, y)
	}
	return
}

func rectIoU(a, b image.Rectangle) float64 {
	inter := a.Intersect(b)
	if inter.Empty() {
		return 0
	}
	ia := float64(inter.Dx() * inter.Dy())
	ua := float64(a.Dx()*a.Dy()+b.Dx()*b.Dy()) - ia
	if ua <= 0 {
		return 0
	}
	return ia / ua
}

// verify applies the acceptance rules to a RANSAC consensus.
func verify(pool []corr, idx []int, s similarity, ref *features, refW, refH int, screen image.Rectangle, minInliers int) (instance, bool) {
	if len(idx) < minInliers {
		return instance{}, false
	}
	sc := s.scale()
	if sc < minScale || sc > maxScale {
		return instance{}, false
	}
	minX, minY, maxX, maxY := projectBox(s, float64(refW), float64(refH))
	if maxX-minX < minBoxSide || maxY-minY < minBoxSide {
		return instance{}, false
	}
	box := image.Rect(int(math.Round(minX)), int(math.Round(minY)), int(math.Round(maxX)), int(math.Round(maxY)))
	full := image.Rect(0, 0, screen.Dx(), screen.Dy())
	vis := box.Intersect(full)
	if vis.Empty() || float64(vis.Dx()*vis.Dy()) < minVisible*float64(box.Dx()*box.Dy()) {
		return instance{}, false
	}

	// Inliers must cover a fair share of where the reference has features.
	ix0, iy0 := math.Inf(1), math.Inf(1)
	ix1, iy1 := math.Inf(-1), math.Inf(-1)
	for _, i := range idx {
		c := pool[i]
		ix0, ix1 = math.Min(ix0, c.rx), math.Max(ix1, c.rx)
		iy0, iy1 = math.Min(iy0, c.ry), math.Max(iy1, c.ry)
	}
	extentX := math.Max(1, float64(ref.maxX-ref.minX))
	extentY := math.Max(1, float64(ref.maxY-ref.minY))
	if (ix1-ix0)/extentX < minSpread || (iy1-iy0)/extentY < minSpread {
		return instance{}, false
	}
	return instance{s: s, box: box, nInliers: len(idx)}, true
}

// findInstances runs RANSAC repeatedly, removing each accepted instance's
// support, until the pool is exhausted or the instance cap is reached.
func findInstances(ms []match, ref, scr *features, refW, refH int, screen image.Rectangle, o Options) []instance {
	pool := make([]corr, len(ms))
	for i, m := range ms {
		rk, sk := ref.kps[m.ref], scr.kps[m.scr]
		pool[i] = corr{
			rx: float64(rk.x), ry: float64(rk.y),
			sx: float64(sk.x), sy: float64(sk.y),
			logScale: math.Log(float64(sk.scale) / float64(rk.scale)),
			dAngle:   wrapAngle(float64(sk.angle) - float64(rk.angle)),
		}
	}
	rng := rand.New(rand.NewPCG(0x5157_7f3a_1d2c_9b01, 0x0ddb_a11c_e5a7_0001))
	var found []instance
	maxRounds := 2*o.MaxInstances + 4
	for round := 0; round < maxRounds && len(pool) >= o.MinInliers && len(found) < o.MaxInstances; round++ {
		s, idx := ransac(pool, rng)
		if len(idx) < o.MinInliers {
			break
		}
		inst, ok := verify(pool, idx, s, ref, refW, refH, screen, o.MinInliers)
		if ok {
			for _, f := range found {
				if rectIoU(f.box, inst.box) > maxOverlapIoU {
					ok = false
					break
				}
			}
		}
		drop := make([]bool, len(pool))
		for _, i := range idx {
			drop[i] = true
		}
		if ok {
			for i, c := range pool {
				if !drop[i] && image.Pt(int(c.sx), int(c.sy)).In(inst.box) {
					drop[i] = true
				}
			}
		}
		removed := 0
		kept := pool[:0]
		for i, c := range pool {
			if drop[i] {
				removed++
				continue
			}
			kept = append(kept, c)
		}
		if ok {
			inst.matched = removed
			found = append(found, inst)
		}
		pool = kept
	}
	return found
}

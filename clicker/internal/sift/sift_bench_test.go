package sift

import (
	"fmt"
	"image"
	"os"
	"runtime"
	"sort"
	"testing"
	"time"
)

func benchScene(scale int) (*image.RGBA, *image.RGBA) {
	ref := refImage()
	scr := scene(scale)
	paste(scr, renderPattern(newPattern(refSeed, refShapes), 96*scale), image.Pt(417*scale, 233*scale))
	return ref, scr
}

func benchmarkFindAll(b *testing.B, scale int) {
	ref, scr := benchScene(scale)
	b.ResetTimer()
	for b.Loop() {
		if _, _, err := FindAll(ref, scr, DefaultOptions()); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkFindAll1280x800(b *testing.B)  { benchmarkFindAll(b, 1) }
func BenchmarkFindAll2560x1600(b *testing.B) { benchmarkFindAll(b, 2) }

// TestReport prints latency percentiles and allocation for both screen
// sizes. It runs only with SIFT_REPORT=1.
func TestReport(t *testing.T) {
	if os.Getenv("SIFT_REPORT") == "" {
		t.Skip("set SIFT_REPORT=1")
	}
	const iters = 20
	for _, scale := range []int{1, 2} {
		ref, scr := benchScene(scale)
		var st Stats
		lat := make([]time.Duration, 0, iters)
		var before, after runtime.MemStats
		runtime.GC()
		runtime.ReadMemStats(&before)
		for range iters {
			var err error
			_, st, err = FindAll(ref, scr, DefaultOptions())
			if err != nil {
				t.Fatal(err)
			}
			lat = append(lat, st.Elapsed)
		}
		runtime.ReadMemStats(&after)
		perRun := (after.TotalAlloc - before.TotalAlloc) / iters

		runtime.GC()
		runtime.ReadMemStats(&before)
		peak := before.HeapInuse
		stop := make(chan struct{})
		done := make(chan struct{})
		go func() {
			defer close(done)
			var m runtime.MemStats
			for {
				select {
				case <-stop:
					return
				default:
					runtime.ReadMemStats(&m)
					peak = max(peak, m.HeapInuse)
					time.Sleep(2 * time.Millisecond)
				}
			}
		}()
		if _, _, err := FindAll(ref, scr, DefaultOptions()); err != nil {
			t.Fatal(err)
		}
		close(stop)
		<-done

		sort.Slice(lat, func(i, j int) bool { return lat[i] < lat[j] })
		p50 := lat[len(lat)/2]
		p95 := lat[(len(lat)*95+99)/100-1]
		fmt.Printf("REPORT %dx%d: p50=%v p95=%v alloc/run=%.1fMB peakHeapInuse=%.1fMB refKP=%d screenKP=%d ratioMatches=%d\n",
			screenW*scale, screenH*scale, p50.Round(time.Millisecond), p95.Round(time.Millisecond),
			float64(perRun)/1e6, float64(peak-before.HeapInuse)/1e6, st.RefKeypoints, st.ScreenKeypoints, st.RatioMatches)
	}
}

// TestReportNormalization compares contrast normalizations on
// brightness/contrast-shifted copies and on the lookalike guard. It runs only
// with SIFT_REPORT=1.
func TestReportNormalization(t *testing.T) {
	if os.Getenv("SIFT_REPORT") == "" {
		t.Skip("set SIFT_REPORT=1")
	}
	saved := normalizeFn
	defer func() { normalizeFn = saved }()
	modes := []struct {
		name string
		fn   func(*grayImage)
	}{
		{"none", func(*grayImage) {}},
		{"global", stretchGlobal},
		{"local", normalizeLocal},
	}
	ref := refImage()
	type cfg struct{ alpha, beta float64 }
	cfgs := []cfg{{1, 0}, {0.6, 0.15}, {0.4, 0.2}, {0.3, 0.3}, {0.2, 0.3}, {0.15, 0.35}}
	for _, mode := range modes {
		normalizeFn = mode.fn
		row := fmt.Sprintf("NORMALIZE %-6s", mode.name)
		for _, c := range cfgs {
			scr := scene(1)
			truth := paste(scr, adjust(ref, c.alpha, c.beta), image.Pt(640, 300))
			ms, st, err := FindAll(ref, scr, DefaultOptions())
			if err != nil {
				t.Fatal(err)
			}
			inl := 0
			if m, ok := matchFor(ms, truth, 0.7); ok {
				inl = m.Inliers
			}
			row += fmt.Sprintf(" | a=%.2f b=%.2f: n=%d inl=%d kp=%d", c.alpha, c.beta, len(ms), inl, st.ScreenKeypoints)
		}
		scr := scene(1)
		paste(scr, renderPattern(newPattern(refSeed+1, refShapes), 96), image.Pt(417, 233))
		ms, st, err := FindAll(ref, scr, DefaultOptions())
		if err != nil {
			t.Fatal(err)
		}
		row += fmt.Sprintf(" | lookalike: n=%d elapsed=%v", len(ms), st.Elapsed.Round(time.Millisecond))
		fmt.Println(row)
	}
}

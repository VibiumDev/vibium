package sift

import (
	"encoding/json"
	"errors"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func mustFindAll(t *testing.T, ref, screen image.Image) ([]Match, Stats) {
	t.Helper()
	ms, st, err := FindAll(ref, screen, DefaultOptions())
	if err != nil {
		t.Fatalf("FindAll: %v", err)
	}
	t.Logf("ref=%d screen=%d ratio=%d elapsed=%v matches=%d", st.RefKeypoints, st.ScreenKeypoints, st.RatioMatches, st.Elapsed, len(ms))
	return ms, st
}

// expectOne asserts a single match that overlaps truth with at least minIoU.
func expectOne(t *testing.T, ms []Match, truth image.Rectangle, minIoU float64) Match {
	t.Helper()
	if len(ms) != 1 {
		t.Fatalf("got %d matches, want 1: %+v", len(ms), ms)
	}
	m := ms[0]
	if v := iou(m.Box, truth); v < minIoU {
		t.Fatalf("IoU %.3f < %.2f; box=%v truth=%v", v, minIoU, m.Box, truth)
	}
	t.Logf("box=%v truth=%v iou=%.3f inliers=%d scale=%.3f rot=%.3f score=%.3f", m.Box, truth, iou(m.Box, truth), m.Inliers, m.Scale, m.Rotation, m.Score)
	return m
}

func TestExactCopy(t *testing.T) {
	ref := refImage()
	scr := scene(1)
	truth := paste(scr, ref, image.Pt(417, 233))
	ms, _ := mustFindAll(t, ref, scr)
	m := expectOne(t, ms, truth, 0.8)
	if d := centerDist(m, truth); d > 3 {
		t.Fatalf("center off by %.2f px", d)
	}
	if m.Inliers < DefaultOptions().MinInliers {
		t.Fatalf("inliers %d below MinInliers", m.Inliers)
	}
}

func TestScaledCopies(t *testing.T) {
	ref := refImage()
	scr := scene(1)
	small := paste(scr, resample(ref, 48, 48), image.Pt(300, 300))
	large := paste(scr, resample(ref, 144, 144), image.Pt(760, 380))
	ms, _ := mustFindAll(t, ref, scr)
	if len(ms) != 2 {
		t.Fatalf("got %d matches, want 2: %+v", len(ms), ms)
	}
	for _, want := range []struct {
		box   image.Rectangle
		scale float64
	}{{small, 0.5}, {large, 1.5}} {
		m, ok := matchFor(ms, want.box, 0.7)
		if !ok {
			t.Fatalf("no match with IoU >= 0.7 for %v: %+v", want.box, ms)
		}
		if math.Abs(m.Scale-want.scale)/want.scale > 0.15 {
			t.Fatalf("scale %.3f not within 15%% of %.2f", m.Scale, want.scale)
		}
		t.Logf("truth=%v box=%v iou=%.3f scale=%.3f inliers=%d", want.box, m.Box, iou(m.Box, want.box), m.Scale, m.Inliers)
	}
}

func matchFor(ms []Match, truth image.Rectangle, minIoU float64) (Match, bool) {
	best, bestIoU := Match{}, 0.0
	for _, m := range ms {
		if v := iou(m.Box, truth); v > bestIoU {
			best, bestIoU = m, v
		}
	}
	return best, bestIoU >= minIoU
}

func TestBrightnessContrastShift(t *testing.T) {
	ref := refImage()
	scr := scene(1)
	truth := paste(scr, adjust(ref, 0.6, 0.15), image.Pt(640, 300))
	ms, _ := mustFindAll(t, ref, scr)
	expectOne(t, ms, truth, 0.8)
}

func TestOccludedCopy(t *testing.T) {
	ref := refImage()
	scr := scene(1)
	truth := paste(scr, ref, image.Pt(500, 350))
	// Cover the right quarter: 24 of 96 columns.
	fillRect(scr, image.Rect(truth.Max.X-24, truth.Min.Y, truth.Max.X, truth.Max.Y), color.RGBA{250, 250, 250, 255})
	ms, _ := mustFindAll(t, ref, scr)
	expectOne(t, ms, truth, 0.7)
}

func TestThreeCopies(t *testing.T) {
	ref := refImage()
	scr := scene(1)
	truths := []image.Rectangle{
		paste(scr, ref, image.Pt(250, 200)),
		paste(scr, ref, image.Pt(700, 250)),
		paste(scr, ref, image.Pt(480, 520)),
	}
	ms, _ := mustFindAll(t, ref, scr)
	if len(ms) != 3 {
		t.Fatalf("got %d matches, want 3: %+v", len(ms), ms)
	}
	for i := range ms {
		for j := i + 1; j < len(ms); j++ {
			if !ms[i].Box.Intersect(ms[j].Box).Empty() {
				t.Fatalf("boxes overlap: %v %v", ms[i].Box, ms[j].Box)
			}
		}
	}
	used := map[int]bool{}
	for _, truth := range truths {
		found := false
		for i, m := range ms {
			if !used[i] && iou(m.Box, truth) >= 0.8 {
				used[i] = true
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("no distinct match for truth %v: %+v", truth, ms)
		}
	}
}

func TestSimilarNonMatch(t *testing.T) {
	ref := refImage()
	scr := scene(1)
	lookalike := renderPattern(newPattern(refSeed+1, refShapes), 96)
	paste(scr, lookalike, image.Pt(417, 233))
	ms, _ := mustFindAll(t, ref, scr)
	if len(ms) != 0 {
		t.Fatalf("false positive: %+v", ms)
	}
}

func TestTinyAndFlatReferences(t *testing.T) {
	scr := scene(1)
	paste(scr, refImage(), image.Pt(417, 233))

	tiny := resample(refImage(), 8, 8)
	ms, _, err := FindAll(tiny, scr, DefaultOptions())
	if !errors.Is(err, ErrNoFeatures) && !errors.Is(err, ErrTooSmall) {
		t.Fatalf("tiny: err=%v matches=%v", err, ms)
	}
	if len(ms) != 0 {
		t.Fatalf("tiny produced matches: %+v", ms)
	}

	ms, _, err = FindAll(solid(96, 180), scr, DefaultOptions())
	if !errors.Is(err, ErrNoFeatures) {
		t.Fatalf("flat: err=%v matches=%v", err, ms)
	}
	if len(ms) != 0 {
		t.Fatalf("flat produced matches: %+v", ms)
	}

	if _, _, err := FindAll(solid(7, 0), scr, DefaultOptions()); !errors.Is(err, ErrTooSmall) {
		t.Fatalf("7x7: err=%v", err)
	}
}

func TestAbsentReference(t *testing.T) {
	ms, _ := mustFindAll(t, refImage(), scene(1))
	if len(ms) != 0 {
		t.Fatalf("false positive: %+v", ms)
	}
	if ms == nil {
		t.Fatal("no-match result must be an empty slice, not nil")
	}
}

func TestHighDPI(t *testing.T) {
	ref := refImage()
	scr := scene(2)
	truth := paste(scr, renderPattern(newPattern(refSeed, refShapes), 192), image.Pt(834, 466))
	ms, _ := mustFindAll(t, ref, scr)
	m := expectOne(t, ms, truth, 0.8)
	if math.Abs(m.Scale-2)/2 > 0.15 {
		t.Fatalf("scale %.3f not within 15%% of 2", m.Scale)
	}
}

func TestDeterminism(t *testing.T) {
	ref := refImage()
	scr := scene(1)
	paste(scr, ref, image.Pt(250, 200))
	paste(scr, resample(ref, 144, 144), image.Pt(700, 300))
	a, _ := mustFindAll(t, ref, scr)
	b, _ := mustFindAll(t, ref, scr)
	if !reflect.DeepEqual(a, b) {
		t.Fatalf("runs differ:\n%+v\n%+v", a, b)
	}
}

func TestFlatScreenIsNotAnError(t *testing.T) {
	ms, st, err := FindAll(refImage(), solid(300, 200), DefaultOptions())
	if err != nil || len(ms) != 0 {
		t.Fatalf("err=%v matches=%v", err, ms)
	}
	if st.RefKeypoints < DefaultOptions().MinInliers {
		t.Fatalf("reference keypoints %d", st.RefKeypoints)
	}
}

func TestTooLarge(t *testing.T) {
	o := DefaultOptions()
	o.MaxPixels = 1000
	if _, _, err := FindAll(refImage(), scene(1), o); !errors.Is(err, ErrTooLarge) {
		t.Fatalf("err=%v", err)
	}
}

func TestFind(t *testing.T) {
	ref := refImage()
	scr := scene(1)
	truth := paste(scr, ref, image.Pt(100, 300))
	m, _, err := Find(ref, scr, DefaultOptions())
	if err != nil || m == nil {
		t.Fatalf("err=%v match=%v", err, m)
	}
	if iou(m.Box, truth) < 0.8 {
		t.Fatalf("box=%v truth=%v", m.Box, truth)
	}
	none, _, err := Find(ref, scene(1), DefaultOptions())
	if err != nil || none != nil {
		t.Fatalf("absent: err=%v match=%v", err, none)
	}
}

// --- PNG fixtures -----------------------------------------------------------

type truthFile struct {
	Ref    string          `json:"ref"`
	Screen string          `json:"screen"`
	Boxes  [][4]int        `json:"boxes"`
	Scales []float64       `json:"scales"`
	Notes  string          `json:"notes"`
	Absent map[string]bool `json:"absent"`
}

// buildUIFixture draws a mock application window with the reference icon at
// 1x and 1.25x, two sibling icons from other seeds, and text blocks.
func buildUIFixture() (ref, screen *image.RGBA, truth truthFile) {
	icon := newPattern(21, 80)
	ref = renderPattern(icon, 80)
	screen = image.NewRGBA(image.Rect(0, 0, 640, 400))
	for y := 0; y < 400; y++ {
		g := uint8(236 - y/20)
		fillRect(screen, image.Rect(0, y, 640, y+1), color.RGBA{g, g, g + 4, 255})
	}
	fillRect(screen, image.Rect(0, 0, 640, 36), color.RGBA{52, 58, 72, 255})
	fillRect(screen, image.Rect(0, 36, 160, 400), color.RGBA{220, 222, 228, 255})
	for i := 0; i < 6; i++ {
		y := 56 + i*44
		fillRect(screen, image.Rect(16, y, 144, y+30), color.RGBA{200, 203, 210, 255})
		textLines(screen, image.Rect(24, y+8, 136, y+22), uint64(300+i))
	}
	fillRect(screen, image.Rect(190, 60, 610, 62), color.RGBA{120, 120, 130, 255})
	textLines(screen, image.Rect(190, 80, 610, 150), 41)
	textLines(screen, image.Rect(190, 260, 400, 380), 42)
	fillRect(screen, image.Rect(430, 320, 600, 360), color.RGBA{66, 133, 244, 255})
	fillRect(screen, image.Rect(434, 324, 596, 356), color.RGBA{80, 150, 255, 255})
	for i, pt := range []image.Point{{190, 166}, {460, 170}} {
		paste(screen, renderPattern(newPattern(uint64(60+i), 80), 80), pt)
	}
	b1 := paste(screen, ref, image.Pt(320, 170))
	b2 := paste(screen, resample(ref, 100, 100), image.Pt(500, 262))
	truth = truthFile{
		Ref:    "ui-ref.png",
		Screen: "ui-screen.png",
		Boxes:  [][4]int{{b1.Min.X, b1.Min.Y, b1.Dx(), b1.Dy()}, {b2.Min.X, b2.Min.Y, b2.Dx(), b2.Dy()}},
		Scales: []float64{1, 1.25},
		Notes:  "mock window; icons at 190,166 and 460,170 are decoys from other seeds",
	}
	return ref, screen, truth
}

func writePNG(t *testing.T, path string, img image.Image) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	enc := png.Encoder{CompressionLevel: png.BestCompression}
	if err := enc.Encode(f, img); err != nil {
		t.Fatal(err)
	}
}

// TestGenerateFixtures rewrites testdata/ when SIFT_GEN_FIXTURES=1.
func TestGenerateFixtures(t *testing.T) {
	if os.Getenv("SIFT_GEN_FIXTURES") == "" {
		t.Skip("set SIFT_GEN_FIXTURES=1 to regenerate testdata/")
	}
	ref, screen, truth := buildUIFixture()
	if err := os.MkdirAll("testdata", 0o755); err != nil {
		t.Fatal(err)
	}
	writePNG(t, filepath.Join("testdata", truth.Ref), ref)
	writePNG(t, filepath.Join("testdata", truth.Screen), screen)
	absent := renderPattern(newPattern(77, 80), 80)
	writePNG(t, filepath.Join("testdata", "ui-absent.png"), absent)
	truth.Absent = map[string]bool{"ui-absent.png": true}
	raw, err := json.MarshalIndent(truth, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join("testdata", "ui-truth.json"), append(raw, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
}

func loadPNG(t *testing.T, path string) image.Image {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	img, err := png.Decode(f)
	if err != nil {
		t.Fatal(err)
	}
	return img
}

func TestPNGFixtures(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("testdata", "ui-truth.json"))
	if err != nil {
		t.Fatal(err)
	}
	var truth truthFile
	if err := json.Unmarshal(raw, &truth); err != nil {
		t.Fatal(err)
	}
	ref := loadPNG(t, filepath.Join("testdata", truth.Ref))
	screen := loadPNG(t, filepath.Join("testdata", truth.Screen))
	ms, _ := mustFindAll(t, ref, screen)
	if len(ms) != len(truth.Boxes) {
		t.Fatalf("got %d matches, want %d: %+v", len(ms), len(truth.Boxes), ms)
	}
	for i, b := range truth.Boxes {
		want := image.Rect(b[0], b[1], b[0]+b[2], b[1]+b[3])
		m, ok := matchFor(ms, want, 0.8)
		if !ok {
			t.Fatalf("no match with IoU >= 0.8 for %v: %+v", want, ms)
		}
		if s := truth.Scales[i]; math.Abs(m.Scale-s)/s > 0.15 {
			t.Fatalf("scale %.3f not within 15%% of %.2f", m.Scale, s)
		}
		t.Logf("truth=%v box=%v iou=%.3f scale=%.3f inliers=%d", want, m.Box, iou(m.Box, want), m.Scale, m.Inliers)
	}
	for name := range truth.Absent {
		absent := loadPNG(t, filepath.Join("testdata", name))
		ms, _ := mustFindAll(t, absent, screen)
		if len(ms) != 0 {
			t.Fatalf("%s: false positive %+v", name, ms)
		}
	}
}

// TestFixturesInSync fails when testdata/ no longer matches the generator.
// The generator anti-aliases with float arithmetic, and a toolchain or CPU
// can round a channel one level differently, so a pixel may differ by a few
// levels without the fixture being stale.
func TestFixturesInSync(t *testing.T) {
	ref, screen, truth := buildUIFixture()
	for _, c := range []struct {
		name string
		img  image.Image
	}{{truth.Ref, ref}, {truth.Screen, screen}} {
		got := loadPNG(t, filepath.Join("testdata", c.name))
		if !nearlyImage(got, c.img, 4) {
			t.Fatalf("%s differs from the generator; rerun with SIFT_GEN_FIXTURES=1", c.name)
		}
	}
}

// nearlyImage reports whether every channel of every pixel is within tol
// 8-bit levels, with the same bounds.
func nearlyImage(a, b image.Image, tol int) bool {
	if a.Bounds() != b.Bounds() {
		return false
	}
	within := func(x, y uint32) bool {
		d := int(x>>8) - int(y>>8)
		return d <= tol && d >= -tol
	}
	bd := a.Bounds()
	for y := bd.Min.Y; y < bd.Max.Y; y++ {
		for x := bd.Min.X; x < bd.Max.X; x++ {
			r1, g1, b1, a1 := a.At(x, y).RGBA()
			r2, g2, b2, a2 := b.At(x, y).RGBA()
			if !within(r1, r2) || !within(g1, g2) || !within(b1, b2) || !within(a1, a2) {
				return false
			}
		}
	}
	return true
}

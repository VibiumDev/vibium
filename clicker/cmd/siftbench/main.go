// Command siftbench runs the sift matcher on two PNG files and prints JSON,
// so results can be compared with other matchers such as
// scripts/siftbench/opencv_reference.py.
//
//	siftbench --ref icon.png --screen shot.png [--truth x,y,w,h] [--all]
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"image"
	"image/png"
	"os"
	"strconv"
	"strings"

	"github.com/vibium/clicker/internal/sift"
)

type box struct {
	X, Y, W, H int
}

type outMatch struct {
	Box      box     `json:"box"`
	Center   [2]int  `json:"center"`
	Inliers  int     `json:"inliers"`
	Matched  int     `json:"matched"`
	Scale    float64 `json:"scale"`
	Rotation float64 `json:"rotation"`
	Score    float64 `json:"score"`
}

type outStats struct {
	RefKeypoints    int     `json:"ref_keypoints"`
	ScreenKeypoints int     `json:"screen_keypoints"`
	RatioMatches    int     `json:"ratio_matches"`
	ElapsedMs       float64 `json:"elapsed_ms"`
}

type output struct {
	Matcher    string     `json:"matcher"`
	Matches    []outMatch `json:"matches"`
	Stats      outStats   `json:"stats"`
	IoUvsTruth *float64   `json:"iou_vs_truth,omitempty"`
	Error      string     `json:"error,omitempty"`
}

func main() {
	refPath := flag.String("ref", "", "reference PNG")
	screenPath := flag.String("screen", "", "screenshot PNG")
	truth := flag.String("truth", "", "ground-truth box as x,y,w,h")
	all := flag.Bool("all", false, "report every instance instead of the best one")
	minInliers := flag.Int("min-inliers", 0, "RANSAC inliers needed per instance (0 = default)")
	ratio := flag.Float64("ratio", 0, "Lowe ratio (0 = default)")
	noUpsample := flag.Bool("no-upsample", false, "do not double a small reference")
	flag.Parse()
	if *refPath == "" || *screenPath == "" {
		fmt.Fprintln(os.Stderr, "usage: siftbench --ref a.png --screen b.png [--truth x,y,w,h] [--all]")
		os.Exit(2)
	}

	ref, err := loadPNG(*refPath)
	if err != nil {
		fail(err)
	}
	screen, err := loadPNG(*screenPath)
	if err != nil {
		fail(err)
	}

	opts := sift.DefaultOptions()
	opts.MinInliers = *minInliers
	opts.Ratio = *ratio
	opts.Upsample = !*noUpsample

	matches, stats, err := sift.FindAll(ref, screen, opts)
	out := output{
		Matcher: "go-sift",
		Matches: []outMatch{},
		Stats: outStats{
			RefKeypoints:    stats.RefKeypoints,
			ScreenKeypoints: stats.ScreenKeypoints,
			RatioMatches:    stats.RatioMatches,
			ElapsedMs:       float64(stats.Elapsed.Microseconds()) / 1000,
		},
	}
	if err != nil {
		out.Error = err.Error()
	}
	if !*all && len(matches) > 1 {
		matches = matches[:1]
	}
	for _, m := range matches {
		out.Matches = append(out.Matches, outMatch{
			Box:      box{m.Box.Min.X, m.Box.Min.Y, m.Box.Dx(), m.Box.Dy()},
			Center:   [2]int{m.Center.X, m.Center.Y},
			Inliers:  m.Inliers,
			Matched:  m.Matched,
			Scale:    m.Scale,
			Rotation: m.Rotation,
			Score:    m.Score,
		})
	}
	if *truth != "" {
		tb, err := parseBox(*truth)
		if err != nil {
			fail(err)
		}
		best := 0.0
		for _, m := range matches {
			best = max(best, iou(m.Box, tb))
		}
		out.IoUvsTruth = &best
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(out); err != nil {
		fail(err)
	}
	if err != nil {
		os.Exit(1)
	}
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "siftbench:", err)
	os.Exit(1)
}

func loadPNG(path string) (image.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return png.Decode(f)
}

func parseBox(s string) (image.Rectangle, error) {
	parts := strings.Split(s, ",")
	if len(parts) != 4 {
		return image.Rectangle{}, fmt.Errorf("truth must be x,y,w,h, got %q", s)
	}
	var v [4]int
	for i, p := range parts {
		n, err := strconv.Atoi(strings.TrimSpace(p))
		if err != nil {
			return image.Rectangle{}, fmt.Errorf("truth %q: %w", s, err)
		}
		v[i] = n
	}
	return image.Rect(v[0], v[1], v[0]+v[2], v[1]+v[3]), nil
}

func iou(a, b image.Rectangle) float64 {
	inter := a.Intersect(b)
	if inter.Empty() {
		return 0
	}
	ia := float64(inter.Dx() * inter.Dy())
	return ia / (float64(a.Dx()*a.Dy()+b.Dx()*b.Dy()) - ia)
}

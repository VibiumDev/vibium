// findbench measures the two find backends against labelled cases, outside
// the hot path and without touching a browser.
//
//	findbench image --manifest cases.jsonl
//	findbench ground --manifest cases.jsonl [--provider P --model M --ai-base-url U --reasoning-effort E]
//
// Each manifest line is one case. For image: {"screenshot": "shot.png",
// "reference": "ref.png", "truth": [{"x":..,"y":..,"width":..,"height":..}]}
// where truth lists every occurrence (empty for a no-match case). For ground:
// {"screenshot": "shot.png", "description": "the Add to Cart button",
// "truth": {"x":..,"y":..,"width":..,"height":..}} with truth null when the
// element is absent. Coordinates are screenshot pixels.
//
// The report is JSON: per case the boxes, the best IoU, whether the best
// box's center lies in the target, and latency; then a summary with hit rate
// (IoU >= 0.5), center-in-target rate, false positives, misses, and p50/p95
// latency. Token cost is not surfaced by the provider adapters; read it from
// the provider's dashboard for the run.
package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"math"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/vibium/clicker/internal/ai"
	"github.com/vibium/clicker/internal/envfile"
	"github.com/vibium/clicker/internal/sift"
)

type box struct {
	X, Y, Width, Height float64
}

func (b box) iou(o box) float64 {
	ix := math.Max(0, math.Min(b.X+b.Width, o.X+o.Width)-math.Max(b.X, o.X))
	iy := math.Max(0, math.Min(b.Y+b.Height, o.Y+o.Height)-math.Max(b.Y, o.Y))
	inter := ix * iy
	union := b.Width*b.Height + o.Width*o.Height - inter
	if union <= 0 {
		return 0
	}
	return inter / union
}

func (b box) contains(x, y float64) bool {
	return x >= b.X && x <= b.X+b.Width && y >= b.Y && y <= b.Y+b.Height
}

type caseResult struct {
	Case      int     `json:"case"`
	Boxes     []box   `json:"boxes"`
	BestIoU   float64 `json:"best_iou"`
	CenterHit bool    `json:"center_in_target"`
	Expected  int     `json:"expected"`
	Found     int     `json:"found"`
	LatencyMs float64 `json:"latency_ms"`
	Error     string  `json:"error,omitempty"`
	Reason    string  `json:"reason,omitempty"`
}

type summary struct {
	Cases          int     `json:"cases"`
	Hits           int     `json:"hits"`
	CenterHits     int     `json:"center_hits"`
	FalsePositives int     `json:"false_positives"`
	Misses         int     `json:"misses"`
	Errors         int     `json:"errors"`
	P50Ms          float64 `json:"p50_ms"`
	P95Ms          float64 `json:"p95_ms"`
	Provider       string  `json:"provider,omitempty"`
	Model          string  `json:"model,omitempty"`
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: findbench image|ground --manifest cases.jsonl")
		os.Exit(2)
	}
	fs := flag.NewFlagSet(os.Args[1], flag.ExitOnError)
	manifest := fs.String("manifest", "", "JSONL file of cases")
	provider := fs.String("provider", "", "AI provider override")
	model := fs.String("model", "", "model override")
	baseURL := fs.String("ai-base-url", "", "AI base URL override")
	effort := fs.String("reasoning-effort", "", "reasoning effort override")
	fs.Parse(os.Args[2:])
	if *manifest == "" {
		fmt.Fprintln(os.Stderr, "--manifest is required")
		os.Exit(2)
	}
	lines, err := readLines(*manifest)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	dir := filepath.Dir(*manifest)
	var results []caseResult
	var sum summary
	switch os.Args[1] {
	case "image":
		results = runImage(dir, lines)
	case "ground":
		// The same ~/.config/vibium/ai.env the CLI reads, unless VIBIUM_LOAD_AI_ENV=0.
		if err := envfile.LoadAIEnv(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		var o ai.Overrides
		for v, target := range map[*string]**string{provider: &o.Provider, model: &o.Model, baseURL: &o.BaseURL, effort: &o.ReasoningEffort} {
			if *v != "" {
				*target = v
			}
		}
		cfg, err := ai.ResolveConfig("find", o)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		sum.Provider, sum.Model = cfg.Provider, cfg.Model
		results = runGround(dir, lines, cfg)
	default:
		fmt.Fprintln(os.Stderr, "unknown mode", os.Args[1])
		os.Exit(2)
	}
	summarize(&sum, results)
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.Encode(map[string]interface{}{"cases": results, "summary": sum})
}

func readLines(path string) ([]json.RawMessage, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var out []json.RawMessage
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 1<<24)
	for sc.Scan() {
		if len(sc.Bytes()) > 0 {
			out = append(out, append(json.RawMessage{}, sc.Bytes()...))
		}
	}
	return out, sc.Err()
}

func loadImage(dir, name string) (image.Image, []byte, error) {
	data, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		return nil, nil, err
	}
	img, _, err := image.Decode(bytes.NewReader(data))
	return img, data, err
}

func runImage(dir string, lines []json.RawMessage) []caseResult {
	var out []caseResult
	for i, line := range lines {
		var c struct {
			Screenshot, Reference string
			Truth                 []box
		}
		r := caseResult{Case: i + 1}
		if err := json.Unmarshal(line, &c); err != nil {
			r.Error = err.Error()
			out = append(out, r)
			continue
		}
		r.Expected = len(c.Truth)
		screen, _, err := loadImage(dir, c.Screenshot)
		if err == nil {
			var ref image.Image
			if ref, _, err = loadImage(dir, c.Reference); err == nil {
				start := time.Now()
				var matches []sift.Match
				matches, _, err = sift.FindAll(ref, screen, sift.DefaultOptions())
				r.LatencyMs = float64(time.Since(start).Microseconds()) / 1000
				for _, m := range matches {
					r.Boxes = append(r.Boxes, box{float64(m.Box.Min.X), float64(m.Box.Min.Y), float64(m.Box.Dx()), float64(m.Box.Dy())})
				}
			}
		}
		if err != nil {
			r.Error = err.Error()
		}
		r.Found = len(r.Boxes)
		score(&r, c.Truth)
		out = append(out, r)
	}
	return out
}

func runGround(dir string, lines []json.RawMessage, cfg ai.Config) []caseResult {
	var out []caseResult
	for i, line := range lines {
		var c struct {
			Screenshot, Description string
			Truth                   *box
			All                     bool
		}
		r := caseResult{Case: i + 1}
		if err := json.Unmarshal(line, &c); err != nil {
			r.Error = err.Error()
			out = append(out, r)
			continue
		}
		var truth []box
		if c.Truth != nil {
			truth = []box{*c.Truth}
		}
		r.Expected = len(truth)
		img, data, err := loadImage(dir, c.Screenshot)
		if err == nil {
			start := time.Now()
			var res ai.GroundResult
			res, err = (&ai.Model{}).Ground(context.Background(), cfg, ai.GroundRequest{
				Description: c.Description, Screenshot: data, Width: img.Bounds().Dx(), Height: img.Bounds().Dy(), All: c.All, Limit: 10,
			})
			r.LatencyMs = float64(time.Since(start).Microseconds()) / 1000
			r.Reason = res.Reason
			for _, b := range res.Boxes {
				r.Boxes = append(r.Boxes, box{b.X, b.Y, b.Width, b.Height})
			}
		}
		if err != nil {
			r.Error = err.Error()
		}
		r.Found = len(r.Boxes)
		score(&r, truth)
		out = append(out, r)
	}
	return out
}

// score compares reported boxes with the truth. A single-target case is a
// hit when its best box overlaps the target; a multi-target case when every
// target is covered.
func score(r *caseResult, truth []box) {
	for _, t := range truth {
		for _, b := range r.Boxes {
			if iou := t.iou(b); iou > r.BestIoU {
				r.BestIoU = iou
				r.CenterHit = t.contains(b.X+b.Width/2, b.Y+b.Height/2)
			}
		}
	}
}

func summarize(s *summary, results []caseResult) {
	var lat []float64
	for _, r := range results {
		s.Cases++
		if r.Error != "" {
			s.Errors++
			continue
		}
		lat = append(lat, r.LatencyMs)
		switch {
		case r.Expected == 0 && r.Found > 0:
			s.FalsePositives++
		case r.Expected > 0 && r.Found == 0:
			s.Misses++
		case r.Expected > 0 && r.BestIoU >= 0.5:
			s.Hits++
		}
		if r.CenterHit {
			s.CenterHits++
		}
	}
	sort.Float64s(lat)
	if n := len(lat); n > 0 {
		s.P50Ms = lat[n/2]
		s.P95Ms = lat[int(math.Min(float64(n-1), math.Ceil(0.95*float64(n))-1))]
	}
}

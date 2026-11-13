package api

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	"math"
	"strings"
	"time"

	"github.com/vibium/clicker/internal/ai"
	"github.com/vibium/clicker/internal/sift"
)

// cvWindow bounds how long local matching keeps retrying before an auto
// search consults a model.
const cvWindow = 3 * time.Second

// vlmScreenshotLimit is the PNG size above which the screenshot goes to the
// model as JPEG, so the request stays inside the provider payload limit.
const vlmScreenshotLimit = 2 * 1024 * 1024

// screenCapture is one screenshot with the scale from viewport CSS pixels
// to image pixels, which differ by the device pixel ratio.
type screenCapture struct {
	img            image.Image
	data           []byte // PNG or JPEG bytes for the model
	width, height  int    // image pixels
	cssW, cssH     int    // viewport CSS pixels
	scaleX, scaleY float64
	state          *PageState
}

// captureTries is how many times a screenshot is asked for when the
// browser has no frame to give. Chrome says so now and then under load, and
// has a frame a moment later.
const captureTries = 3

// screenshotOfPage takes a screenshot, asking again when the browser had
// no frame.
func screenshotOfPage(s Session, context string) (string, error) {
	b64, err := Screenshot(s, context, false)
	for try := 1; err != nil && try < captureTries && strings.Contains(err.Error(), "Unable to capture screenshot"); try++ {
		time.Sleep(150 * time.Millisecond)
		b64, err = Screenshot(s, context, false)
	}
	return b64, err
}

func captureScreen(s Session, context string) (*screenCapture, error) {
	b64, err := screenshotOfPage(s, context)
	if err != nil {
		return nil, fmt.Errorf("screenshot failed: %w", err)
	}
	data, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return nil, fmt.Errorf("screenshot failed: invalid image data")
	}
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("screenshot failed: %w", err)
	}
	c := &screenCapture{img: img, data: data, width: img.Bounds().Dx(), height: img.Bounds().Dy()}
	if c.width == 0 || c.height == 0 {
		return nil, fmt.Errorf("screenshot failed: empty image")
	}
	if err := c.measure(s, context); err != nil {
		return nil, err
	}
	if len(data) > vlmScreenshotLimit {
		var buf bytes.Buffer
		if jpeg.Encode(&buf, img, &jpeg.Options{Quality: 80}) == nil {
			c.data = buf.Bytes()
		}
	}
	c.state, _ = readPageState(s, context)
	return c, nil
}

// measure sets the CSS size the screenshot covers and the scale to it.
// A classic scrollbar is inside innerWidth but outside the visual viewport,
// and browsers differ on which one a screenshot covers (Chrome the visual
// viewport, Firefox the window). The capture is at the device pixel ratio,
// so the width whose ratio lands on the DPR is the one covered.
// documentElement.clientWidth is no help: in quirks mode it is the document.
func (c *screenCapture) measure(s Session, context string) error {
	out, err := EvalSimpleScript(s, context, "() => [innerWidth, innerHeight, visualViewport ? visualViewport.width : innerWidth, visualViewport ? visualViewport.height : innerHeight, devicePixelRatio].join(',')")
	var innerW, innerH, visualW, visualH, dpr float64
	if err == nil {
		if n, _ := fmt.Sscanf(out, "%g,%g,%g,%g,%g", &innerW, &innerH, &visualW, &visualH, &dpr); n != 5 {
			err = fmt.Errorf("unexpected viewport metrics %q", out)
		}
	}
	if err != nil || innerW <= 0 || innerH <= 0 || dpr <= 0 {
		w, h, err := ScreenSize(s, context)
		if err != nil {
			return err
		}
		c.cssW, c.cssH = w, h
		c.scaleX, c.scaleY = float64(c.width)/float64(w), float64(c.height)/float64(h)
		return nil
	}
	pick := func(px int, inner, visual float64) float64 {
		if visual <= 0 || math.Abs(float64(px)/inner-dpr) <= math.Abs(float64(px)/visual-dpr) {
			return inner
		}
		return visual
	}
	w, h := pick(c.width, innerW, visualW), pick(c.height, innerH, visualH)
	c.cssW, c.cssH = int(math.Round(w)), int(math.Round(h))
	c.scaleX, c.scaleY = float64(c.width)/w, float64(c.height)/h
	return nil
}

func readPageState(s Session, context string) (*PageState, bool) {
	out, err := EvalSimpleScript(s, context, pageStateScript())
	if err != nil {
		return nil, false
	}
	var st PageState
	if json.Unmarshal([]byte(out), &st) != nil {
		return nil, false
	}
	return &st, true
}

func decodeReference(data []byte) (image.Image, error) {
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("reference image: %w", err)
	}
	return img, nil
}

// matchImage runs one local pass and converts matches to viewport boxes.
func matchImage(s Session, context string, ref image.Image, shot *screenCapture, loc Locator, limit int) ([]Found, sift.Stats, error) {
	// The matcher's ratio test looks past MaxInstances neighbours, so a
	// repeated thumbnail still matches when only one hit is wanted. Match
	// with the full default and cut the list afterwards.
	opts := sift.DefaultOptions()
	if limit > opts.MaxInstances {
		opts.MaxInstances = limit
	}
	matches, stats, err := sift.FindAll(ref, shot.img, opts)
	if limit > 0 && len(matches) > limit {
		matches = matches[:limit]
	}
	if err != nil {
		if errors.Is(err, sift.ErrNoFeatures) || errors.Is(err, sift.ErrTooSmall) {
			return nil, stats, fmt.Errorf("reference image is too small or flat to match (%d keypoints); use a larger or more detailed crop", stats.RefKeypoints)
		}
		return nil, stats, fmt.Errorf("image matching failed: %w", err)
	}
	var found []Found
	for i, m := range matches {
		box := BoxInfo{
			X: float64(m.Box.Min.X) / shot.scaleX, Y: float64(m.Box.Min.Y) / shot.scaleY,
			Width: float64(m.Box.Dx()) / shot.scaleX, Height: float64(m.Box.Dy()) / shot.scaleY,
		}
		f := Found{Label: visualLabel("", "image", box), Role: "image", Box: box, Index: i, Source: SourceImage, Method: "sift"}
		if loc.Details {
			// scale is per CSS pixel: a 2x screenshot of an unscaled image reports 1.
			f.Details = map[string]interface{}{
				"inliers": m.Inliers, "matched": m.Matched, "scale": round3(m.Scale / shot.scaleX), "rotation": round3(m.Rotation),
				"score": round3(m.Score), "refKeypoints": stats.RefKeypoints, "screenKeypoints": stats.ScreenKeypoints,
			}
		}
		found = append(found, snapToDOM(s, context, f))
	}
	return found, stats, nil
}

func round3(f float64) float64 { return math.Round(f*1000) / 1000 }

func findByImage(s Session, context string, loc Locator, all bool) (FindResult, error) {
	ref, err := decodeReference(loc.Image)
	if err != nil {
		return FindResult{}, err
	}
	cfg, cfgErr := ai.ResolveConfig("find", loc.AI)
	if loc.Strategy == StrategyVLM {
		if cfgErr != nil {
			return FindResult{}, fmt.Errorf("strategy vlm needs a configured vision model: %w", cfgErr)
		}
		shot, err := captureScreen(s, context)
		if err != nil {
			return FindResult{}, err
		}
		return groundImage(s, context, loc, cfg, shot, all)
	}
	limit := loc.Limit
	if !all {
		limit = 1
	}
	deadline := time.Now().Add(loc.Timeout)
	window := deadline
	if loc.Strategy == StrategyAuto && cfgErr == nil && time.Until(deadline) > cvWindow {
		window = time.Now().Add(cvWindow)
	}
	var shot *screenCapture
	var stats sift.Stats
	for {
		if shot, err = captureScreen(s, context); err != nil {
			return FindResult{}, err
		}
		var found []Found
		found, stats, err = matchImage(s, context, ref, shot, loc, limit)
		if err != nil {
			return FindResult{}, err
		}
		if len(found) > 0 {
			return FindResult{Matches: found, Method: "sift", State: shot.state, Image: loc.Image}, nil
		}
		if time.Now().After(window) {
			break
		}
		time.Sleep(500 * time.Millisecond)
	}
	if loc.Strategy == StrategyAuto && cfgErr == nil {
		return groundImage(s, context, loc, cfg, shot, all)
	}
	res := FindResult{Method: "sift", State: shot.state, Image: loc.Image}
	if all {
		return res, nil
	}
	hint := fmt.Sprintf("no verified match for the reference image (%d keypoints in the reference, %d on screen)", stats.RefKeypoints, stats.ScreenKeypoints)
	if loc.Strategy == StrategyAuto && cfgErr != nil {
		hint += "; a vision model would be consulted next, but none is configured"
	}
	return res, &NotFoundError{Locator: loc, Timeout: loc.Timeout, Hint: hint}
}

// groundImage asks the model for regions that correspond to the reference.
func groundImage(s Session, context string, loc Locator, cfg ai.Config, shot *screenCapture, all bool) (FindResult, error) {
	req := ai.GroundRequest{Reference: loc.Image, Screenshot: shot.data, Width: shot.width, Height: shot.height, All: all, Limit: loc.Limit}
	res, err := groundMatches(s, context, loc, cfg, req, shot, all)
	res.Image = loc.Image
	for i := range res.Matches {
		if res.Matches[i].Role == "" {
			res.Matches[i].Role = "image"
		}
	}
	return res, err
}

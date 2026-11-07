package ai

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"
)

// GroundTimeout bounds one grounding call: a single model turn, no loop.
const GroundTimeout = 90 * time.Second

// GroundMaxBoxes caps how many boxes one call may report.
const GroundMaxBoxes = 20

// GroundMaxImage bounds each image sent for grounding.
const GroundMaxImage = 6 * 1024 * 1024

// GroundRequest asks a vision model where something is on one screenshot.
type GroundRequest struct {
	Description string
	Screenshot  []byte // PNG or JPEG
	Reference   []byte // optional PNG or JPEG the target should correspond to
	Width       int    // screenshot pixel size
	Height      int
	All         bool
	Limit       int
	Scope       *GroundBox // optional region in screenshot pixels
}

// GroundBox is a box in screenshot pixels.
type GroundBox struct {
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
	Role   string  `json:"role,omitempty"`
	Label  string  `json:"label,omitempty"`
}

// GroundResult is what the model reported, after validation.
type GroundResult struct {
	Boxes    []GroundBox
	Reason   string
	Provider string
	Model    string
	Elapsed  time.Duration
}

const groundInstruction = `You locate user-interface elements on one screenshot. You are given the screenshot's pixel size and a request. Answer only by calling report_matches with pixel bounding boxes on that screenshot: x and y are the top-left corner, width and height the size, all in pixels of the given image.
Report only elements that are actually visible and satisfy the whole request, including any color, position, icon, or appearance words. If a reference image is supplied, report only regions that visually correspond to it, not merely similar things. If nothing qualifies, report an empty list and say why in reason. Never guess, never pad the list, and never describe or suggest actions: you cannot act.`

// groundTool is the one function the model may call.
func groundTool() Tool {
	return Tool{
		Name:        "report_matches",
		Description: "Report the bounding boxes of the requested element(s) on the screenshot, or an empty list.",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"matches": map[string]interface{}{
					"type": "array",
					"items": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"x":      map[string]interface{}{"type": "number"},
							"y":      map[string]interface{}{"type": "number"},
							"width":  map[string]interface{}{"type": "number"},
							"height": map[string]interface{}{"type": "number"},
							"role":   map[string]interface{}{"type": "string", "description": "button, link, textbox, image, heading, or similar"},
							"label":  map[string]interface{}{"type": "string", "description": "the visible text or name, if any"},
						},
						"required": []string{"x", "y", "width", "height"},
					},
				},
				"reason": map[string]interface{}{"type": "string", "description": "why the list is empty or ambiguous"},
			},
			"required": []string{"matches"},
		},
	}
}

func imageMIME(data []byte) (string, error) {
	switch {
	case bytes.HasPrefix(data, []byte("\x89PNG\r\n\x1a\n")):
		return "image/png", nil
	case bytes.HasPrefix(data, []byte{0xff, 0xd8, 0xff}):
		return "image/jpeg", nil
	}
	return "", fmt.Errorf("image must be a PNG or JPEG")
}

func imagePart(data []byte) (map[string]interface{}, error) {
	if len(data) > GroundMaxImage {
		return nil, fmt.Errorf("image exceeds the %d MB grounding limit", GroundMaxImage/1024/1024)
	}
	mime, err := imageMIME(data)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{"type": "image_url", "image_url": map[string]string{"url": "data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(data)}}, nil
}

// Ground runs one bounded model turn and returns validated boxes. It performs
// no actions and keeps no history.
func (v *Model) Ground(ctx context.Context, c Config, req GroundRequest) (GroundResult, error) {
	if err := c.Validate(); err != nil {
		return GroundResult{}, err
	}
	if req.Width <= 0 || req.Height <= 0 {
		return GroundResult{}, fmt.Errorf("screenshot size is unknown")
	}
	limit := req.Limit
	if limit <= 0 || limit > GroundMaxBoxes {
		limit = GroundMaxBoxes
	}
	if !req.All {
		limit = 1
	}
	shot, err := imagePart(req.Screenshot)
	if err != nil {
		return GroundResult{}, fmt.Errorf("screenshot: %w", err)
	}
	var ask strings.Builder
	fmt.Fprintf(&ask, "The screenshot is %d by %d pixels.\n", req.Width, req.Height)
	if req.Scope != nil {
		fmt.Fprintf(&ask, "Only consider the region x=%d y=%d width=%d height=%d.\n", int(req.Scope.X), int(req.Scope.Y), int(req.Scope.Width), int(req.Scope.Height))
	}
	switch {
	case req.Description != "" && req.All:
		fmt.Fprintf(&ask, "Find every distinct element matching this request, at most %d: %s", limit, req.Description)
	case req.Description != "":
		fmt.Fprintf(&ask, "Find the single element matching this request: %s", req.Description)
	case req.All:
		fmt.Fprintf(&ask, "Find every region that visually corresponds to the reference image, at most %d.", limit)
	default:
		ask.WriteString("Find the single region that visually corresponds to the reference image.")
	}
	parts := []interface{}{map[string]interface{}{"type": "text", "text": ask.String()}, shot}
	if len(req.Reference) > 0 {
		ref, err := imagePart(req.Reference)
		if err != nil {
			return GroundResult{}, fmt.Errorf("reference image: %w", err)
		}
		parts = append(parts, map[string]interface{}{"type": "text", "text": "Reference image the target should correspond to:"}, ref)
	}
	messages := []message{
		{Role: "system", Content: groundInstruction},
		{Role: "user", Content: parts},
	}
	functions := []interface{}{map[string]interface{}{"type": "function", "function": groundTool()}}

	ctx, cancel := context.WithTimeout(ctx, GroundTimeout)
	defer cancel()
	start := time.Now()
	msg, err := v.complete(ctx, c, messages, functions, "report_matches")
	result := GroundResult{Provider: c.Provider, Model: c.Model, Elapsed: time.Since(start)}
	if err != nil {
		return result, groundError(c, err)
	}
	var call *toolCall
	for i := range msg.ToolCalls {
		if msg.ToolCalls[i].Function.Name == "report_matches" {
			call = &msg.ToolCalls[i]
			break
		}
	}
	if call == nil {
		return result, fmt.Errorf("model answered without reporting matches; try a vision-capable model that supports tool calls")
	}
	var report struct {
		Matches []GroundBox `json:"matches"`
		Reason  string      `json:"reason"`
	}
	if err := json.Unmarshal([]byte(call.Function.Arguments), &report); err != nil {
		return result, fmt.Errorf("model returned malformed matches")
	}
	result.Reason = strings.TrimSpace(report.Reason)
	result.Boxes = validBoxes(report.Matches, req, limit)
	return result, nil
}

// groundError names the vision requirement when the provider rejects the
// message content, which is what a text-only model does with an image.
func groundError(c Config, err error) error {
	text := err.Error()
	if strings.Contains(text, "HTTP 400") && strings.Contains(text, "in messages") {
		return fmt.Errorf("model %s rejected the screenshot; find needs a vision-capable model (%sMODEL or --model)", c.Model, c.Prefix())
	}
	return err
}

// validBoxes drops boxes that are not finite, lie outside the screenshot or
// the scope, are degenerate, or duplicate an earlier box.
func validBoxes(in []GroundBox, req GroundRequest, limit int) []GroundBox {
	w, h := float64(req.Width), float64(req.Height)
	var out []GroundBox
	for _, b := range in {
		if !finite(b.X) || !finite(b.Y) || !finite(b.Width) || !finite(b.Height) {
			continue
		}
		if b.Width < 2 || b.Height < 2 {
			continue
		}
		// Clip a box that spills a little; drop one that is mostly outside.
		x0, y0 := math.Max(0, b.X), math.Max(0, b.Y)
		x1, y1 := math.Min(w, b.X+b.Width), math.Min(h, b.Y+b.Height)
		if x1-x0 < 2 || y1-y0 < 2 || (x1-x0)*(y1-y0) < 0.5*b.Width*b.Height {
			continue
		}
		b.X, b.Y, b.Width, b.Height = x0, y0, x1-x0, y1-y0
		if b.Width*b.Height > 0.9*w*h {
			continue
		}
		if req.Scope != nil && !inside(b, *req.Scope) {
			continue
		}
		dup := false
		for _, o := range out {
			if iou(b, o) > 0.7 {
				dup = true
				break
			}
		}
		if dup {
			continue
		}
		b.Role = strings.ToLower(strings.TrimSpace(b.Role))
		b.Label = strings.TrimSpace(b.Label)
		out = append(out, b)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Y != out[j].Y {
			return out[i].Y < out[j].Y
		}
		return out[i].X < out[j].X
	})
	if len(out) > limit {
		out = out[:limit]
	}
	return out
}

func finite(f float64) bool { return !math.IsNaN(f) && !math.IsInf(f, 0) }

func inside(b, scope GroundBox) bool {
	cx, cy := b.X+b.Width/2, b.Y+b.Height/2
	return cx >= scope.X && cx <= scope.X+scope.Width && cy >= scope.Y && cy <= scope.Y+scope.Height
}

func iou(a, b GroundBox) float64 {
	ix := math.Max(0, math.Min(a.X+a.Width, b.X+b.Width)-math.Max(a.X, b.X))
	iy := math.Max(0, math.Min(a.Y+a.Height, b.Y+b.Height)-math.Max(a.Y, b.Y))
	inter := ix * iy
	union := a.Width*a.Height + b.Width*b.Height - inter
	if union <= 0 {
		return 0
	}
	return inter / union
}

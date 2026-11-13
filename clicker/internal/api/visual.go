package api

import (
	"errors"
	"fmt"
	"math"
	"strings"
	"time"
)

// ErrVisualUnsupported is returned when a command that needs a DOM element
// is given a coordinate-backed ref.
var ErrVisualUnsupported = errors.New("this ref is a visual match with no DOM element; it supports click, dblclick, hover, tap, highlight, and bounds")

// StaleRefError says a visual ref's coordinates no longer describe the page.
type StaleRefError struct {
	Ref    string
	Reason string
}

func (e *StaleRefError) Error() string {
	return fmt.Sprintf("%s is stale: %s since it was found; run find again", e.Ref, e.Reason)
}

// visualMethods are the commands that work from a box alone. The key is the
// wire method or the MCP tool name.
var visualMethods = map[string]bool{
	"vibium:element.click": true, "vibium:element.dblclick": true, "vibium:element.hover": true,
	"vibium:element.tap": true, "vibium:element.highlight": true, "vibium:element.bounds": true,
	"vibium:element.scrollIntoView": true,
	"browser_click":                 true, "browser_dblclick": true, "browser_hover": true, "browser_tap": true,
	"browser_highlight": true,
}

// ResolveRefParams rewrites @eN selectors in params before a command runs.
// A DOM-backed ref becomes its selector. A visual ref stays as the ref, with
// its entry attached under visualRef, and only commands that can act on a box
// may proceed.
func ResolveRefParams(t *RefTable, method string, params map[string]interface{}) error {
	if t == nil || params == nil {
		return nil
	}
	for _, key := range []string{"selector", "source", "target"} {
		sel, ok := params[key].(string)
		if !ok || !IsRef(sel) {
			continue
		}
		e, found := t.Lookup(sel)
		if !found {
			return fmt.Errorf("%s is not a mapped element; run find or map first", sel)
		}
		if !e.Visual() {
			params[key] = e.Selector
			continue
		}
		if key != "selector" || !visualMethods[method] {
			return fmt.Errorf("%s: %w", sel, ErrVisualUnsupported)
		}
		params["visualRef"] = e
	}
	return nil
}

// VisualElementInfo checks that a coordinate-backed ref still describes the
// page, re-detecting an image match after a scroll or resize, and returns an
// ElementInfo pointing at its center.
func VisualElementInfo(s Session, context string, e *RefEntry) (*ElementInfo, error) {
	if e.Method == nativeTreeMethod {
		if err := verifyNativeRef(s, e); err != nil {
			return nil, err
		}
	} else if state, ok := readPageState(s, context); ok && e.State != nil {
		switch {
		case state.URL != e.State.URL:
			return nil, &StaleRefError{Ref: e.Ref, Reason: "the page navigated"}
		case state.ScrollX != e.State.ScrollX || state.ScrollY != e.State.ScrollY ||
			state.Width != e.State.Width || state.Height != e.State.Height:
			if len(e.Image) == 0 {
				return nil, &StaleRefError{Ref: e.Ref, Reason: "the page scrolled or resized"}
			}
			if err := redetect(s, context, e, state); err != nil {
				return nil, err
			}
		}
	}
	cx, cy := e.Center()
	return &ElementInfo{Tag: e.Tag, Text: e.Label, Box: e.Box, Point: PointInfo{X: float64(cx), Y: float64(cy)}}, nil
}

// redetect runs the local image match again and keeps the instance nearest
// to where the old one should be after the scroll.
func redetect(s Session, context string, e *RefEntry, state *PageState) error {
	loc := Locator{Image: e.Image, Strategy: StrategyCV, Timeout: 2 * time.Second, Limit: 10}
	res, err := findByImage(s, context, loc, true)
	if err != nil {
		return err
	}
	wantX := e.Box.X + e.Box.Width/2 - (state.ScrollX - e.State.ScrollX)
	wantY := e.Box.Y + e.Box.Height/2 - (state.ScrollY - e.State.ScrollY)
	best, bestDist := -1, math.Inf(1)
	for i, m := range res.Matches {
		d := math.Hypot(m.Box.X+m.Box.Width/2-wantX, m.Box.Y+m.Box.Height/2-wantY)
		if d < bestDist {
			best, bestDist = i, d
		}
	}
	if best < 0 {
		return &StaleRefError{Ref: e.Ref, Reason: "the page scrolled and the image is no longer on screen"}
	}
	m := res.Matches[best]
	e.Box, e.Selector, e.Label, e.Tag, e.Text, e.Role, e.Name, e.State = m.Box, m.Selector, m.Label, m.Tag, m.Text, m.Role, m.Name, state
	return nil
}

// HighlightBox outlines a viewport box for two seconds.
func HighlightBox(s Session, context string, box BoxInfo) error {
	_, err := CallScript(s, context, highlightBoxScript(), []map[string]interface{}{
		{"type": "number", "value": box.X}, {"type": "number", "value": box.Y},
		{"type": "number", "value": box.Width}, {"type": "number", "value": box.Height},
		{"type": "number", "value": 2000},
	})
	return err
}

// FormatRef renders one mapped element the way find and map print it.
func FormatRef(e *RefEntry) string {
	return e.Ref + " " + e.Label
}

// FormatRefs renders a list, one per line.
func FormatRefs(entries []*RefEntry) string {
	lines := make([]string, len(entries))
	for i, e := range entries {
		lines[i] = FormatRef(e)
	}
	return strings.Join(lines, "\n")
}

// RefJSON is the structured form of one mapped element for --json output.
func RefJSON(e *RefEntry, details bool) map[string]interface{} {
	m := map[string]interface{}{
		"ref":    e.Ref,
		"role":   e.Role,
		"tag":    e.Tag,
		"label":  e.Label,
		"text":   e.Text,
		"name":   e.Name,
		"box":    e.Box,
		"source": e.Source,
	}
	if e.Selector != "" {
		m["selector"] = e.Selector
	}
	if details {
		d := map[string]interface{}{"method": e.Method}
		for k, v := range e.Details {
			d[k] = v
		}
		m["details"] = d
	}
	return m
}

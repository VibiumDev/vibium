package api

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
)

// SwipeDirections are the directions a swipe moves the finger.
var SwipeDirections = []string{"up", "down", "left", "right"}

// SwipeEdges are the named start points a swipe takes in "from".
var SwipeEdges = []string{"top", "bottom", "left", "right", "top-left", "top-right", "bottom-left", "bottom-right", "center"}

// SwipeSpeeds maps the named speeds to pixels per second.
var SwipeSpeeds = map[string]float64{"slow": 500, "normal": 1000, "fast": 2500}

const defaultSwipeDuration = 300

// SwipeOptions is a swipe as a caller asks for it: four coordinates, or a
// direction resolved against the viewport.
type SwipeOptions struct {
	Coords         bool
	X1, Y1, X2, Y2 float64

	Direction    string
	From         string
	FromX, FromY float64
	FromPoint    bool
	Distance     float64

	Speed    float64 // px/s; 0 means use Duration
	Duration int     // ms; 0 means the default
	Hold     int     // ms the finger rests at the end before lifting
}

// SwipePath is a swipe resolved to viewport coordinates and timing.
type SwipePath struct {
	X1, Y1, X2, Y2 int
	Duration       int
	Hold           int
}

func paramFloat(params map[string]interface{}, key string) (float64, bool, error) {
	switch v := params[key].(type) {
	case nil:
		return 0, false, nil
	case float64:
		return v, true, nil
	case string:
		f, err := strconv.ParseFloat(strings.TrimSpace(v), 64)
		if err != nil {
			return 0, false, fmt.Errorf("%s must be a number, got %q", key, v)
		}
		return f, true, nil
	default:
		return 0, false, fmt.Errorf("%s must be a number", key)
	}
}

func paramString(params map[string]interface{}, key string) string {
	s, _ := params[key].(string)
	return strings.ToLower(strings.TrimSpace(s))
}

func oneOf(v string, list []string) bool {
	for _, s := range list {
		if v == s {
			return true
		}
	}
	return false
}

// ParseSwipeSpeed reads a speed as a name in SwipeSpeeds or pixels per second.
func ParseSwipeSpeed(v interface{}) (float64, error) {
	switch s := v.(type) {
	case nil:
		return 0, nil
	case float64:
		if s <= 0 {
			return 0, fmt.Errorf("speed must be positive, got %v", s)
		}
		return s, nil
	case string:
		name := strings.ToLower(strings.TrimSpace(s))
		if px, ok := SwipeSpeeds[name]; ok {
			return px, nil
		}
		f, err := strconv.ParseFloat(name, 64)
		if err != nil || f <= 0 {
			return 0, fmt.Errorf("speed must be slow, normal, fast, or pixels per second, got %q", s)
		}
		return f, nil
	default:
		return 0, fmt.Errorf("speed must be slow, normal, fast, or pixels per second")
	}
}

// ParseSwipeOptions reads a swipe from wire or MCP params. It is the one
// parser for vibium:touch.swipe and browser_swipe.
func ParseSwipeOptions(params map[string]interface{}) (SwipeOptions, error) {
	var o SwipeOptions
	var coords [4]float64
	n := 0
	for i, k := range []string{"x1", "y1", "x2", "y2"} {
		v, ok, err := paramFloat(params, k)
		if err != nil {
			return o, err
		}
		if ok {
			coords[i] = v
			n++
		}
	}
	o.Direction = paramString(params, "direction")
	switch {
	case n == 4 && o.Direction != "":
		return o, fmt.Errorf("give either x1, y1, x2, y2 or a direction, not both")
	case n == 4:
		o.Coords = true
		o.X1, o.Y1, o.X2, o.Y2 = coords[0], coords[1], coords[2], coords[3]
	case n > 0:
		return o, fmt.Errorf("x1, y1, x2 and y2 are all required for a coordinate swipe")
	case o.Direction == "":
		return o, fmt.Errorf("swipe needs a direction (%s) or x1, y1, x2, y2", strings.Join(SwipeDirections, ", "))
	case !oneOf(o.Direction, SwipeDirections):
		return o, fmt.Errorf("direction must be one of %s, got %q", strings.Join(SwipeDirections, ", "), o.Direction)
	}

	o.From = paramString(params, "from")
	fx, okX, err := paramFloat(params, "fromX")
	if err != nil {
		return o, err
	}
	fy, okY, err := paramFloat(params, "fromY")
	if err != nil {
		return o, err
	}
	if okX != okY {
		return o, fmt.Errorf("fromX and fromY go together")
	}
	if okX {
		if o.From != "" {
			return o, fmt.Errorf("give either from or fromX/fromY, not both")
		}
		o.FromPoint, o.FromX, o.FromY = true, fx, fy
	}
	if o.From != "" && !oneOf(o.From, SwipeEdges) {
		return o, fmt.Errorf("from must be one of %s, got %q", strings.Join(SwipeEdges, ", "), o.From)
	}
	if o.Distance, _, err = paramFloat(params, "distance"); err != nil {
		return o, err
	}
	if o.Distance < 0 {
		return o, fmt.Errorf("distance must be positive, got %v", o.Distance)
	}
	if o.Coords && (o.From != "" || o.FromPoint || o.Distance != 0) {
		return o, fmt.Errorf("from and distance apply to a direction swipe, not to x1, y1, x2, y2")
	}

	if o.Speed, err = ParseSwipeSpeed(params["speed"]); err != nil {
		return o, err
	}
	d, hasDuration, err := paramFloat(params, "duration")
	if err != nil {
		return o, err
	}
	if hasDuration && d > 0 {
		if o.Speed > 0 {
			return o, fmt.Errorf("give either speed or duration, not both")
		}
		o.Duration = int(d)
	}
	h, _, err := paramFloat(params, "hold")
	if err != nil {
		return o, err
	}
	if h < 0 {
		return o, fmt.Errorf("hold must not be negative, got %v", h)
	}
	o.Hold = int(h)
	return o, nil
}

// ScreenSize returns the size of the coordinate space pointer actions use: the
// page viewport in a browser, the phone's screen through Roadie.
func ScreenSize(s Session, context string) (int, int, error) {
	if w, h, ok := QueryViewport(s, context); ok {
		return w, h, nil
	}
	resp, err := s.SendBidiCommand("roadie:screen.getViewport", map[string]interface{}{})
	if err != nil {
		return 0, 0, fmt.Errorf("cannot read the viewport size: %w", err)
	}
	var r struct {
		Result struct {
			Width  int `json:"width"`
			Height int `json:"height"`
		} `json:"result"`
	}
	if err := json.Unmarshal(resp, &r); err != nil || r.Result.Width <= 0 || r.Result.Height <= 0 {
		return 0, 0, fmt.Errorf("cannot read the viewport size")
	}
	return r.Result.Width, r.Result.Height, nil
}

// ResolveSwipe turns SwipeOptions into a path. A direction swipe starts at the
// viewport center, a named edge or corner, or a point, and moves half the viewport along
// its axis unless distance says otherwise. The end is clamped to the viewport.
func ResolveSwipe(o SwipeOptions, width, height int) (SwipePath, error) {
	var x1, y1, x2, y2 float64
	if o.Coords {
		x1, y1, x2, y2 = o.X1, o.Y1, o.X2, o.Y2
	} else {
		maxX, maxY := float64(width-1), float64(height-1)
		x1, y1 = math.Floor(float64(width)/2), math.Floor(float64(height)/2)
		if strings.HasPrefix(o.From, "top") {
			y1 = 0
		} else if strings.HasPrefix(o.From, "bottom") {
			y1 = maxY
		}
		if strings.HasSuffix(o.From, "left") {
			x1 = 0
		} else if strings.HasSuffix(o.From, "right") {
			x1 = maxX
		}
		if o.FromPoint {
			x1, y1 = o.FromX, o.FromY
		}
		dx, dy := 0.0, 0.0
		axis := float64(height)
		switch o.Direction {
		case "up":
			dy = -1
		case "down":
			dy = 1
		case "left":
			dx, axis = -1, float64(width)
		case "right":
			dx, axis = 1, float64(width)
		}
		dist := o.Distance
		if dist == 0 {
			dist = math.Round(axis / 2)
		}
		x2 = math.Max(0, math.Min(maxX, x1+dx*dist))
		y2 = math.Max(0, math.Min(maxY, y1+dy*dist))
		if x2 == x1 && y2 == y1 {
			return SwipePath{}, fmt.Errorf("a swipe %s from (%d, %d) has no room to move", o.Direction, int(x1), int(y1))
		}
	}

	duration := defaultSwipeDuration
	if o.Duration > 0 {
		duration = o.Duration
	}
	if o.Speed > 0 {
		duration = int(math.Max(1, math.Round(math.Hypot(x2-x1, y2-y1)/o.Speed*1000)))
	}
	return SwipePath{
		X1: int(x1), Y1: int(y1), X2: int(x2), Y2: int(y2),
		Duration: duration,
		Hold:     o.Hold,
	}, nil
}

// Swipe resolves and performs a swipe, returning the path it took.
func Swipe(s Session, context string, o SwipeOptions) (SwipePath, error) {
	var w, h int
	if !o.Coords {
		var err error
		if w, h, err = ScreenSize(s, context); err != nil {
			return SwipePath{}, err
		}
	}
	p, err := ResolveSwipe(o, w, h)
	if err != nil {
		return SwipePath{}, err
	}
	return p, TouchSwipe(s, context, p)
}

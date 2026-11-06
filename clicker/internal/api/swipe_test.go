package api

import (
	"strings"
	"testing"
)

func TestResolveSwipe(t *testing.T) {
	const w, h = 400, 800
	tests := []struct {
		name   string
		params map[string]interface{}
		want   SwipePath
	}{
		{"coordinates", map[string]interface{}{"x1": 10.0, "y1": 20.0, "x2": 30.0, "y2": 40.0},
			SwipePath{10, 20, 30, 40, 300, 0}},
		{"up from center", map[string]interface{}{"direction": "up"},
			SwipePath{200, 400, 200, 0, 300, 0}},
		{"down from center", map[string]interface{}{"direction": "down"},
			SwipePath{200, 400, 200, 799, 300, 0}},
		{"left by distance", map[string]interface{}{"direction": "left", "distance": 150.0},
			SwipePath{200, 400, 50, 400, 300, 0}},
		{"right from left edge", map[string]interface{}{"direction": "right", "from": "left"},
			SwipePath{0, 400, 200, 400, 300, 0}},
		{"up from bottom", map[string]interface{}{"direction": "up", "from": "bottom"},
			SwipePath{200, 799, 200, 399, 300, 0}},
		{"down from top", map[string]interface{}{"direction": "down", "from": "top"},
			SwipePath{200, 0, 200, 400, 300, 0}},
		{"down from top-right", map[string]interface{}{"direction": "down", "from": "top-right"},
			SwipePath{399, 0, 399, 400, 300, 0}},
		{"up from bottom-left", map[string]interface{}{"direction": "up", "from": "bottom-left"},
			SwipePath{0, 799, 0, 399, 300, 0}},
		{"from a point", map[string]interface{}{"direction": "up", "fromX": 100.0, "fromY": 600.0, "distance": 100.0},
			SwipePath{100, 600, 100, 500, 300, 0}},
		{"named speed", map[string]interface{}{"direction": "up", "from": "bottom", "speed": "fast"},
			SwipePath{200, 799, 200, 399, 160, 0}},
		{"numeric speed", map[string]interface{}{"direction": "up", "distance": 200.0, "speed": 400.0},
			SwipePath{200, 400, 200, 200, 500, 0}},
		{"speed on coordinates", map[string]interface{}{"x1": 0.0, "y1": 0.0, "x2": 300.0, "y2": 400.0, "speed": "slow"},
			SwipePath{0, 0, 300, 400, 1000, 0}},
		{"duration and hold", map[string]interface{}{"direction": "up", "duration": 600.0, "hold": 500.0},
			SwipePath{200, 400, 200, 0, 600, 500}},
		{"clamped to the viewport", map[string]interface{}{"direction": "up", "distance": 5000.0},
			SwipePath{200, 400, 200, 0, 300, 0}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o, err := ParseSwipeOptions(tt.params)
			if err != nil {
				t.Fatalf("ParseSwipeOptions: %v", err)
			}
			got, err := ResolveSwipe(o, w, h)
			if err != nil {
				t.Fatalf("ResolveSwipe: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestSwipeErrors(t *testing.T) {
	tests := []struct {
		name   string
		params map[string]interface{}
		want   string
	}{
		{"nothing", map[string]interface{}{}, "needs a direction"},
		{"bad direction", map[string]interface{}{"direction": "sideways"}, "direction must be one of"},
		{"partial coordinates", map[string]interface{}{"x1": 1.0, "y1": 2.0}, "all required"},
		{"both forms", map[string]interface{}{"direction": "up", "x1": 1.0, "y1": 2.0, "x2": 3.0, "y2": 4.0}, "not both"},
		{"bad edge", map[string]interface{}{"direction": "up", "from": "middle"}, "from must be one of"},
		{"edge and point", map[string]interface{}{"direction": "up", "from": "top", "fromX": 1.0, "fromY": 2.0}, "not both"},
		{"half a point", map[string]interface{}{"direction": "up", "fromX": 1.0}, "go together"},
		{"from on coordinates", map[string]interface{}{"x1": 1.0, "y1": 2.0, "x2": 3.0, "y2": 4.0, "from": "top"}, "direction swipe"},
		{"speed and duration", map[string]interface{}{"direction": "up", "speed": "fast", "duration": 200.0}, "not both"},
		{"bad speed", map[string]interface{}{"direction": "up", "speed": "warp"}, "speed must be"},
		{"no room", map[string]interface{}{"direction": "up", "from": "top"}, "no room"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o, err := ParseSwipeOptions(tt.params)
			if err == nil {
				_, err = ResolveSwipe(o, 400, 800)
			}
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Errorf("got error %v, want one containing %q", err, tt.want)
			}
		})
	}
}

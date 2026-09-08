package api

import (
	"testing"
	"time"
)

// An explicit timeout of 0 must mean "check once, do not wait", not fall back
// to the 30s default the way the old `> 0` guard made it (#411).
func TestFindTimeout(t *testing.T) {
	cases := []struct {
		name   string
		params map[string]interface{}
		want   time.Duration
	}{
		{"absent uses default", map[string]interface{}{}, DefaultTimeout},
		{"null uses default", map[string]interface{}{"timeout": nil}, DefaultTimeout},
		{"zero means no wait", map[string]interface{}{"timeout": float64(0)}, 0},
		{"explicit value honored", map[string]interface{}{"timeout": float64(1500)}, 1500 * time.Millisecond},
	}
	for _, tc := range cases {
		if got := findTimeout(tc.params); got != tc.want {
			t.Errorf("%s: findTimeout = %v, want %v", tc.name, got, tc.want)
		}
	}
}

package main

import (
	"testing"
	"time"
)

func TestTimeoutValueSet(t *testing.T) {
	cases := []struct {
		in      string
		want    time.Duration
		wantErr bool
	}{
		{"5s", 5 * time.Second, false},
		{"5000", 5000 * time.Millisecond, false},
		{"500ms", 500 * time.Millisecond, false},
		{"2m", 2 * time.Minute, false},
		{"1.5s", 1500 * time.Millisecond, false},
		{"1500", 1500 * time.Millisecond, false},
		{" 5s ", 5 * time.Second, false},
		{"0", 0, false},
		{"-1", 0, true},
		{"-5s", 0, true},
		{"abc", 0, true},
		{"", 0, true},
	}
	for _, c := range cases {
		var d time.Duration
		v := &timeoutValue{d: &d}
		err := v.Set(c.in)
		if c.wantErr {
			if err == nil {
				t.Errorf("Set(%q): expected error, got %v", c.in, d)
			}
			continue
		}
		if err != nil {
			t.Errorf("Set(%q): unexpected error: %v", c.in, err)
			continue
		}
		if d != c.want {
			t.Errorf("Set(%q): got %v, want %v", c.in, d, c.want)
		}
	}
}

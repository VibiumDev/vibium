package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/vibium/clicker/internal/api"
)

// timeoutValue is a pflag.Value for --timeout that accepts either a Go duration
// string ("5s", "500ms", "2m") or a bare number interpreted as milliseconds
// ("5000"). The parsed value is stored as a time.Duration.
type timeoutValue struct{ d *time.Duration }

func (t *timeoutValue) String() string {
	if t.d == nil {
		return ""
	}
	return t.d.String()
}

func (t *timeoutValue) Type() string { return "timeout" }

func (t *timeoutValue) Set(s string) error {
	s = strings.TrimSpace(s)
	if s == "" {
		return fmt.Errorf("empty timeout")
	}
	// A bare number means milliseconds (e.g. "5000", "1500").
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		if f < 0 {
			return fmt.Errorf("negative timeout: %s", s)
		}
		*t.d = time.Duration(f * float64(time.Millisecond))
		return nil
	}
	// Otherwise a Go duration string (e.g. "5s", "500ms", "2m").
	d, err := time.ParseDuration(s)
	if err != nil {
		return fmt.Errorf("invalid timeout %q: use e.g. 5s or 5000 (bare number = milliseconds)", s)
	}
	if d < 0 {
		return fmt.Errorf("negative timeout: %s", s)
	}
	*t.d = d
	return nil
}

// addTimeoutFlag registers a unified --timeout flag on cmd, storing the parsed
// value into *target. The target is seeded with the default so --help shows it
// and commands can forward it unconditionally.
func addTimeoutFlag(cmd *cobra.Command, target *time.Duration) {
	*target = api.DefaultTimeout
	cmd.Flags().Var(&timeoutValue{d: target}, "timeout",
		"Max time to wait, e.g. 5s or 5000 (bare number = milliseconds)")
}

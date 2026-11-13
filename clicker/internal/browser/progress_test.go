package browser

import (
	"bytes"
	"os"
	"testing"
)

// Progress must stay late-bound: pipe and mcp reassign os.Stdout to os.Stderr
// inside their own Run, long after this package is initialized. A writer
// captured at init would pin download progress to the real stdout and corrupt
// the BiDi protocol stream pipe writes there.
func TestProgressFollowsStdoutReassignment(t *testing.T) {
	origProgress, origStdout := Progress, os.Stdout
	defer func() { Progress, os.Stdout = origProgress, origStdout }()

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()

	Progress = nil // the default: follow whatever os.Stdout is now
	os.Stdout = w  // what pipe.go does
	progressf("downloading %d%%\n", 40)
	w.Close()

	var got bytes.Buffer
	if _, err := got.ReadFrom(r); err != nil {
		t.Fatal(err)
	}
	if got.String() != "downloading 40%\n" {
		t.Errorf("progress did not follow the reassignment: got %q", got.String())
	}
}

// With Progress set explicitly, which is what the CLI does under --json, it
// wins over os.Stdout so the envelope has stdout to itself.
func TestProgressHonorsExplicitWriter(t *testing.T) {
	origProgress := Progress
	defer func() { Progress = origProgress }()

	var buf bytes.Buffer
	Progress = &buf
	progressf("installing %s\n", "chrome")
	if buf.String() != "installing chrome\n" {
		t.Errorf("explicit writer ignored: got %q", buf.String())
	}
}

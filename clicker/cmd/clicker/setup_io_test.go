package main

import (
	"bytes"
	"testing"
)

func TestPromptsSurviveMultiLinePaste(t *testing.T) {
	// A paste of several answers arrives in one read; every prompt after the
	// first must see the lines the previous read buffered.
	in := bytes.NewBufferString("first\nsecond\nthird\n")
	ui := &setupUI{in: in, out: ioDiscard(), err: ioDiscard(), interactive: true}
	for _, want := range []string{"first", "second", "third"} {
		got, err := ui.prompt("q", "unanswered")
		if err != nil {
			t.Fatal(err)
		}
		if got != want {
			t.Fatalf("got %q, want %q", got, want)
		}
	}
}

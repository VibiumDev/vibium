package api

import (
	"errors"
	"testing"
)

const shotOK = `{"result":{"data":"aGVsbG8="}}`

// Chrome now and then has no frame to give: "Unable to capture
// screenshot". A find by image asked a moment later gets one, and must not
// fail on the first answer.
func TestFindScreenshotAsksAgainWhenTheBrowserHasNoFrame(t *testing.T) {
	noFrame := errors.New("unknown error: Unable to capture screenshot")

	f := &scriptedSession{}
	f.step("", noFrame)
	f.step(shotOK, nil)
	if data, err := screenshotOfPage(f, "ctx"); err != nil || data != "aGVsbG8=" || f.calls != 2 {
		t.Fatalf("data %q, err %v, %d calls; want the screenshot on the second", data, err, f.calls)
	}

	// A browser that never has a frame is reported, after a few tries.
	f = &scriptedSession{}
	f.step("", noFrame)
	if _, err := screenshotOfPage(f, "ctx"); !errors.Is(err, noFrame) || f.calls != captureTries {
		t.Fatalf("err %v after %d calls; want the error after %d", err, f.calls, captureTries)
	}

	// Any other failure is not asked about twice.
	gone := errors.New("no such frame")
	f = &scriptedSession{}
	f.step("", gone)
	if _, err := screenshotOfPage(f, "ctx"); !errors.Is(err, gone) || f.calls != 1 {
		t.Fatalf("err %v after %d calls; want one call", err, f.calls)
	}
}

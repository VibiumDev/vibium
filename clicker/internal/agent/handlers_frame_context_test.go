package agent

import "testing"

// After browser_frame the ambient context is the frame and activeFrameParent
// records the page it lives in. Navigation must return to that page instead
// of replacing the iframe's document and trapping every later command inside
// it (#510).
func TestLeaveActiveFrameReturnsToParentPage(t *testing.T) {
	h := &Handlers{activeContext: "frame-ctx", activeFrameParent: "page-ctx"}

	h.leaveActiveFrame()

	if h.activeContext != "page-ctx" {
		t.Errorf("activeContext = %q, want the parent page-ctx", h.activeContext)
	}
	if h.activeFrameParent != "" {
		t.Errorf("activeFrameParent = %q, want cleared", h.activeFrameParent)
	}
}

// On a plain page — the common case — leaving is a no-op, so ordinary
// navigation is untouched.
func TestLeaveActiveFrameNoopOnPage(t *testing.T) {
	h := &Handlers{activeContext: "page-ctx"}

	h.leaveActiveFrame()

	if h.activeContext != "page-ctx" {
		t.Errorf("activeContext = %q, want unchanged page-ctx", h.activeContext)
	}
	if h.activeFrameParent != "" {
		t.Errorf("activeFrameParent = %q, want still empty", h.activeFrameParent)
	}
}

// Closing the page a frame belongs to kills the frame with it: both the
// active context and the parent record must clear, or later commands target
// a dead frame (#510). This is the "trap survives closing the page" case in
// the report.
func TestClearFrameIfContextClosedDropsDeadFrame(t *testing.T) {
	h := &Handlers{activeContext: "frame-ctx", activeFrameParent: "page-ctx"}

	h.clearFrameIfContextClosed("page-ctx")

	if h.activeContext != "" || h.activeFrameParent != "" {
		t.Errorf("after closing the frame's page: activeContext=%q activeFrameParent=%q, want both empty",
			h.activeContext, h.activeFrameParent)
	}
}

// Closing an unrelated page leaves the frame state alone.
func TestClearFrameIfContextClosedIgnoresOtherPages(t *testing.T) {
	h := &Handlers{activeContext: "frame-ctx", activeFrameParent: "page-ctx"}

	h.clearFrameIfContextClosed("some-other-page")

	if h.activeContext != "frame-ctx" || h.activeFrameParent != "page-ctx" {
		t.Errorf("unrelated close changed frame state: activeContext=%q activeFrameParent=%q",
			h.activeContext, h.activeFrameParent)
	}
}

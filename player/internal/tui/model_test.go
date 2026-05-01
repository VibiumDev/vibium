package tui

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/vibium/player/internal/kitty"
	"github.com/vibium/player/internal/recording"
)

func TestKeymapActionAndFrameStepping(t *testing.T) {
	m := New(testRecording(), nil, kitty.PixelSize{Width: 8, Height: 16})
	m.width = 20
	m.height = 10

	m.Update(tea.KeyMsg{Type: tea.KeyRight})
	if m.selIdx != 1 || m.playing || m.frameMode {
		t.Fatalf("right: sel=%d playing=%v frameMode=%v", m.selIdx, m.playing, m.frameMode)
	}
	if m.virtTime != 1000 {
		t.Fatalf("right virtTime = %d", m.virtTime)
	}

	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	if m.frameIdx != 3 || !m.frameMode {
		t.Fatalf("k: frameIdx=%d frameMode=%v", m.frameIdx, m.frameMode)
	}
	if m.virtTime != 5000 {
		t.Fatalf("k virtTime = %d", m.virtTime)
	}

	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if m.frameIdx != 2 || !m.frameMode {
		t.Fatalf("j: frameIdx=%d frameMode=%v", m.frameIdx, m.frameMode)
	}

	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	if !m.playing || m.virtTime != 0 {
		t.Fatalf("r: playing=%v virtTime=%d", m.playing, m.virtTime)
	}
}

func TestTickCompressesIdleGap(t *testing.T) {
	rec := &recording.Recording{
		StartMs: 0,
		EndMs:   6000,
		Frames: []recording.Frame{
			{Timestamp: 0, SHA1: "f0.png", Width: 100, Height: 100},
			{Timestamp: 5000, SHA1: "f1.png", Width: 100, Height: 100},
		},
		Actions: []recording.Action{
			{CallID: "call@1", Title: "First", StartTime: 0, EndTime: 100},
			{CallID: "call@2", Title: "Second", StartTime: 5000, EndTime: 5100},
		},
		Boxes:     map[string]recording.Rect{},
		Resources: map[string][]byte{},
	}
	m := New(rec, nil, kitty.PixelSize{Width: 8, Height: 16})
	m.width = 20
	m.height = 10
	m.playing = true
	m.lastTick = time.Unix(0, 0)

	m.Update(tickMsg(time.Unix(0, int64(100*time.Millisecond))))
	if m.virtTime != 4500 {
		t.Fatalf("idle-compressed virtTime = %d", m.virtTime)
	}
}

func TestSelectedImagePrefersClickBeforeSnapshot(t *testing.T) {
	m := New(testRecording(), nil, kitty.PixelSize{Width: 8, Height: 16})
	m.selIdx = 1
	m.frameMode = false

	selection, ok := m.selectedImage()
	if !ok {
		t.Fatal("selectedImage returned false")
	}
	if selection.sha != "before-click.png" {
		t.Fatalf("sha = %q", selection.sha)
	}

	m.frameMode = true
	m.frameIdx = 2
	selection, ok = m.selectedImage()
	if !ok {
		t.Fatal("selectedImage frame mode returned false")
	}
	if selection.sha != "f2.png" {
		t.Fatalf("frame-mode sha = %q", selection.sha)
	}
}

func TestRenderSkipsDuplicateImage(t *testing.T) {
	rec := testRecording()
	rec.Resources = map[string][]byte{
		"f0.png":           testPNG(t),
		"f1.png":           testPNG(t),
		"f2.png":           testPNG(t),
		"before-click.png": testPNG(t),
	}
	var out bytes.Buffer
	m := New(rec, &out, kitty.PixelSize{Width: 8, Height: 16})
	m.width = 80
	m.height = 24

	cmd := m.renderCmd()
	if cmd == nil {
		t.Fatal("first render returned nil")
	}
	cmd()
	firstLen := out.Len()
	if firstLen == 0 {
		t.Fatal("first render wrote no output")
	}

	if cmd := m.renderCmd(); cmd != nil {
		t.Fatal("second render should be skipped")
	}
	if out.Len() != firstLen {
		t.Fatalf("duplicate render wrote output: before=%d after=%d", firstLen, out.Len())
	}
}

func TestRenderSkipsDirtyDuplicateImageDuringPlayback(t *testing.T) {
	rec := testRecording()
	rec.Resources = map[string][]byte{
		"f0.png":           testPNG(t),
		"before-click.png": testPNG(t),
	}
	var out bytes.Buffer
	m := New(rec, &out, kitty.PixelSize{Width: 8, Height: 16})
	m.width = 80
	m.height = 24

	cmd := m.renderCmd()
	if cmd == nil {
		t.Fatal("first render returned nil")
	}
	cmd()
	firstLen := out.Len()

	m.playing = true
	m.lastTick = time.Unix(0, 0)
	cmd = m.handleTick(time.Unix(0, int64(100*time.Millisecond)))
	if cmd == nil {
		t.Fatal("tick returned nil")
	}
	cmd()

	if out.Len() != firstLen {
		t.Fatalf("same-frame playback tick repainted image: before=%d after=%d", firstLen, out.Len())
	}
}

func TestFitImagePreservesAspectRatio(t *testing.T) {
	m := New(testRecording(), nil, kitty.PixelSize{Width: 10, Height: 20})
	m.width = 80
	m.height = 24
	l := m.computeLayout()

	origin, size := m.fitImage(l, 1600, 900)
	if size.Cols != 54 || size.Rows != 15 {
		t.Fatalf("wide size = %+v", size)
	}
	if origin.Col != 2 || origin.Row != 2 {
		t.Fatalf("wide origin = %+v", origin)
	}

	origin, size = m.fitImage(l, 900, 1600)
	if size.Cols != 18 || size.Rows != 16 {
		t.Fatalf("tall size = %+v", size)
	}
	if origin.Col != 20 || origin.Row != 2 {
		t.Fatalf("tall origin = %+v", origin)
	}
}

func testRecording() *recording.Recording {
	return &recording.Recording{
		StartMs: 0,
		EndMs:   7000,
		Frames: []recording.Frame{
			{Timestamp: 0, SHA1: "f0.png", Width: 100, Height: 100},
			{Timestamp: 600, SHA1: "f1.png", Width: 100, Height: 100},
			{Timestamp: 1600, SHA1: "f2.png", Width: 100, Height: 100},
			{Timestamp: 5000, SHA1: "f3.png", Width: 100, Height: 100},
		},
		Actions: []recording.Action{
			{CallID: "call@1", Title: "Page.navigate", StartTime: 0, EndTime: 500},
			{CallID: "call@2", Title: "Element.click", BeforeImageSHA: "before-click.png", StartTime: 1000, EndTime: 1700},
			{CallID: "call@3", Title: "Element.text", StartTime: 5000, EndTime: 5100},
		},
		Boxes: map[string]recording.Rect{
			"call@2": {X: 10, Y: 10, W: 20, H: 20},
		},
		Resources: map[string][]byte{},
	}
}

func testPNG(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 10, 10))
	for y := 0; y < 10; y++ {
		for x := 0; x < 10; x++ {
			img.SetRGBA(x, y, color.RGBA{R: 1, G: 2, B: 3, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

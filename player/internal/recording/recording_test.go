package recording

import (
	"archive/zip"
	"bytes"
	"errors"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOpenSyntheticRecording(t *testing.T) {
	pngBytes := solidPNG(t)
	jpegBytes := solidJPEG(t)
	trace := strings.Join([]string{
		`{"type":"context-options","title":"Synthetic","monotonicTime":100}`,
		`{"type":"before","callId":"call@1","class":"Tracing","method":"tracingGroup","params":{"name":"Group"},"startTime":100,"title":"Group"}`,
		`{"type":"screencast-frame","sha1":"barepng","width":100,"height":100,"timestamp":105}`,
		`{"type":"before","callId":"call@2","class":"Element","method":"vibium:element.click","parentId":"call@1","beforeSnapshot":"before@call@2","params":{"selector":"#login"},"startTime":110,"title":"Element.click"}`,
		`{"type":"input","callId":"call@2","box":{"x":3,"y":4,"width":5,"height":6}}`,
		`{"type":"frame-snapshot","snapshot":{"snapshotName":"before@call@2","timestamp":111,"resourceOverrides":[{"sha1":"barepng"}]}}`,
		`{"type":"screencast-frame","sha1":"suffixed.jpeg","width":100,"height":100,"timestamp":120}`,
		`{"type":"after","callId":"call@2","endTime":130}`,
		`{"type":"after","callId":"call@1","endTime":140}`,
		`{"type":"before","callId":"call@3","class":"Page","method":"vibium:page.title","afterSnapshot":"after@call@3","params":{},"startTime":150,"title":"Page.title"}`,
		`{"type":"frame-snapshot","snapshot":{"snapshotName":"after@call@3","timestamp":151,"resourceOverrides":[{"sha1":"suffixed.jpeg"}]}}`,
		`{"type":"after","callId":"call@3","afterSnapshot":"after@call@3","endTime":160}`,
	}, "\n")

	rec := openZip(t, map[string][]byte{
		"2-trace.trace":           []byte(trace),
		"2-trace.network":         nil,
		"resources/barepng":       pngBytes,
		"resources/suffixed.jpeg": jpegBytes,
	})
	defer rec.Close()

	if rec.Title != "Synthetic" {
		t.Fatalf("Title = %q", rec.Title)
	}
	if rec.StartMs != 100 || rec.EndMs != 160 {
		t.Fatalf("time range = %d..%d", rec.StartMs, rec.EndMs)
	}
	if len(rec.Frames) != 2 {
		t.Fatalf("frames = %d", len(rec.Frames))
	}
	if len(rec.Actions) != 3 {
		t.Fatalf("actions = %d", len(rec.Actions))
	}
	if rec.Actions[1].Depth != 1 {
		t.Fatalf("child depth = %d", rec.Actions[1].Depth)
	}
	if rec.Actions[1].BeforeImageSHA != "barepng" {
		t.Fatalf("before image = %q", rec.Actions[1].BeforeImageSHA)
	}
	if rec.Actions[2].AfterImageSHA != "suffixed.jpeg" {
		t.Fatalf("after image = %q", rec.Actions[2].AfterImageSHA)
	}
	if got := rec.Boxes["call@2"]; got != (Rect{X: 3, Y: 4, W: 5, H: 6}) {
		t.Fatalf("box = %+v", got)
	}
	if frame, ok := rec.FrameAt(119); !ok || frame.SHA1 != "barepng" {
		t.Fatalf("FrameAt(119) = %+v %v", frame, ok)
	}
	if action, ok := rec.ActionAt(115); !ok || action.CallID != "call@2" {
		t.Fatalf("ActionAt(115) = %+v %v", action, ok)
	}
	if _, contentType, err := rec.Resource("barepng"); err != nil || contentType != "image/png" {
		t.Fatalf("bare Resource contentType = %q err = %v", contentType, err)
	}
	if _, contentType, err := rec.Resource("suffixed.jpeg"); err != nil || contentType != "image/jpeg" {
		t.Fatalf("jpeg Resource contentType = %q err = %v", contentType, err)
	}
}

func TestOpenAcceptsTraceDotTrace(t *testing.T) {
	rec := openZip(t, map[string][]byte{
		"trace.trace": []byte(`{"type":"context-options","title":"Demo","monotonicTime":0}` + "\n"),
	})
	defer rec.Close()
	if rec.Title != "Demo" {
		t.Fatalf("Title = %q", rec.Title)
	}
}

func TestOpenRejectsMultipleTraceFiles(t *testing.T) {
	p := writeZip(t, map[string][]byte{
		"0-trace.trace": []byte(`{"type":"context-options"}` + "\n"),
		"1-trace.trace": []byte(`{"type":"context-options"}` + "\n"),
	})
	_, err := Open(p)
	if !errors.Is(err, ErrMultiTrace) {
		t.Fatalf("err = %v", err)
	}
}

func TestOpenRealVarPartsSmoke(t *testing.T) {
	p := filepath.Join("..", "..", "..", "clients", "javascript", "var-parts-trace.zip")
	rec, err := Open(p)
	if err != nil {
		t.Fatalf("Open(%s): %v", p, err)
	}
	defer rec.Close()
	if len(rec.Frames) == 0 || len(rec.Actions) == 0 {
		t.Fatalf("frames/actions = %d/%d", len(rec.Frames), len(rec.Actions))
	}
}

func openZip(t *testing.T, entries map[string][]byte) *Recording {
	t.Helper()
	p := writeZip(t, entries)
	rec, err := Open(p)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	return rec
}

func writeZip(t *testing.T, entries map[string][]byte) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "record.zip")
	f, err := os.Create(p)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(f)
	for name, data := range entries {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write(data); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	return p
}

func solidPNG(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 10, 10))
	fill(img, color.RGBA{G: 255, A: 255})
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func solidJPEG(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 10, 10))
	fill(img, color.RGBA{B: 255, A: 255})
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, nil); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func fill(img *image.RGBA, c color.RGBA) {
	for y := img.Bounds().Min.Y; y < img.Bounds().Max.Y; y++ {
		for x := img.Bounds().Min.X; x < img.Bounds().Max.X; x++ {
			img.SetRGBA(x, y, c)
		}
	}
}

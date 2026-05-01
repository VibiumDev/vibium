package overlay

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"testing"

	"github.com/vibium/player/internal/recording"
)

func TestDrawBox(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 100, 100))
	for y := 0; y < 100; y++ {
		for x := 0; x < 100; x++ {
			src.SetRGBA(x, y, color.RGBA{R: 10, G: 20, B: 30, A: 255})
		}
	}
	var jpegBytes bytes.Buffer
	if err := jpeg.Encode(&jpegBytes, src, nil); err != nil {
		t.Fatal(err)
	}

	out, err := DrawBox(jpegBytes.Bytes(), "image/jpeg", recording.Rect{X: 10, Y: 20, W: 30, H: 40}, image.Pt(100, 100))
	if err != nil {
		t.Fatal(err)
	}
	img, err := png.Decode(bytes.NewReader(out))
	if err != nil {
		t.Fatal(err)
	}

	assertRed(t, img, 10, 20)
	assertRed(t, img, 39, 20)
	assertRed(t, img, 10, 59)
	assertRed(t, img, 39, 59)
	if got := color.RGBAModel.Convert(img.At(25, 40)).(color.RGBA); got.R == 255 && got.G == 0 && got.B == 0 {
		t.Fatalf("interior pixel unexpectedly red: %+v", got)
	}
}

func TestDrawBoxScalesCoordinates(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 50, 50))
	var pngBytes bytes.Buffer
	if err := png.Encode(&pngBytes, src); err != nil {
		t.Fatal(err)
	}

	out, err := DrawBox(pngBytes.Bytes(), "image/png", recording.Rect{X: 20, Y: 20, W: 20, H: 20}, image.Pt(100, 100))
	if err != nil {
		t.Fatal(err)
	}
	img, err := png.Decode(bytes.NewReader(out))
	if err != nil {
		t.Fatal(err)
	}
	assertRed(t, img, 10, 10)
	assertRed(t, img, 19, 19)
}

func assertRed(t *testing.T, img image.Image, x, y int) {
	t.Helper()
	got := color.RGBAModel.Convert(img.At(x, y)).(color.RGBA)
	if got.R != 255 || got.G != 0 || got.B != 0 || got.A != 255 {
		t.Fatalf("pixel %d,%d = %+v", x, y, got)
	}
}

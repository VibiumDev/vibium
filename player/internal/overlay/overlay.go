package overlay

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"image/png"
	"io"

	"github.com/vibium/player/internal/recording"
)

func DrawBox(frame []byte, contentType string, box recording.Rect, frameSize image.Point) ([]byte, error) {
	img, err := decode(frame, contentType)
	if err != nil {
		return nil, err
	}

	bounds := img.Bounds()
	rgba := image.NewRGBA(bounds)
	draw.Draw(rgba, bounds, img, bounds.Min, draw.Src)

	rect := scaledRect(box, frameSize, bounds)
	if rect.Empty() {
		var out bytes.Buffer
		if err := png.Encode(&out, rgba); err != nil {
			return nil, err
		}
		return out.Bytes(), nil
	}

	red := color.RGBA{R: 255, A: 255}
	for i := 0; i < 2; i++ {
		drawHorizontal(rgba, rect.Min.X, rect.Max.X-1, rect.Min.Y+i, red)
		drawHorizontal(rgba, rect.Min.X, rect.Max.X-1, rect.Max.Y-1-i, red)
		drawVertical(rgba, rect.Min.Y, rect.Max.Y-1, rect.Min.X+i, red)
		drawVertical(rgba, rect.Min.Y, rect.Max.Y-1, rect.Max.X-1-i, red)
	}

	var out bytes.Buffer
	if err := png.Encode(&out, rgba); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

func decode(frame []byte, contentType string) (image.Image, error) {
	r := bytes.NewReader(frame)
	switch contentType {
	case "image/jpeg":
		return jpeg.Decode(r)
	case "image/png":
		return png.Decode(r)
	case "":
		img, _, err := image.Decode(r)
		return img, err
	default:
		img, _, err := image.Decode(r)
		if err == nil {
			return img, nil
		}
		return nil, fmt.Errorf("unsupported image content type %q", contentType)
	}
}

func scaledRect(box recording.Rect, frameSize image.Point, bounds image.Rectangle) image.Rectangle {
	if frameSize.X <= 0 {
		frameSize.X = bounds.Dx()
	}
	if frameSize.Y <= 0 {
		frameSize.Y = bounds.Dy()
	}
	scaleX := float64(bounds.Dx()) / float64(frameSize.X)
	scaleY := float64(bounds.Dy()) / float64(frameSize.Y)

	minX := bounds.Min.X + int(float64(box.X)*scaleX)
	minY := bounds.Min.Y + int(float64(box.Y)*scaleY)
	maxX := bounds.Min.X + int(float64(box.X+box.W)*scaleX)
	maxY := bounds.Min.Y + int(float64(box.Y+box.H)*scaleY)

	rect := image.Rect(minX, minY, maxX, maxY).Intersect(bounds)
	if rect.Dx() < 1 || rect.Dy() < 1 {
		return image.Rectangle{}
	}
	return rect
}

func drawHorizontal(img *image.RGBA, x1, x2, y int, c color.RGBA) {
	if y < img.Bounds().Min.Y || y >= img.Bounds().Max.Y {
		return
	}
	for x := x1; x <= x2; x++ {
		if x >= img.Bounds().Min.X && x < img.Bounds().Max.X {
			img.SetRGBA(x, y, c)
		}
	}
}

func drawVertical(img *image.RGBA, y1, y2, x int, c color.RGBA) {
	if x < img.Bounds().Min.X || x >= img.Bounds().Max.X {
		return
	}
	for y := y1; y <= y2; y++ {
		if y >= img.Bounds().Min.Y && y < img.Bounds().Max.Y {
			img.SetRGBA(x, y, c)
		}
	}
}

func EncodePNG(frame []byte, contentType string) ([]byte, error) {
	img, err := decode(frame, contentType)
	if err != nil {
		if err == io.EOF {
			return nil, fmt.Errorf("empty image")
		}
		return nil, err
	}
	var out bytes.Buffer
	if err := png.Encode(&out, img); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

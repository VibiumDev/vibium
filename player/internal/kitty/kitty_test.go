package kitty

import (
	"bytes"
	"strings"
	"testing"
)

func TestDisplayChunksPNG(t *testing.T) {
	png := bytes.Repeat([]byte{0x42}, 4097)
	var out bytes.Buffer
	if err := Display(&out, png, Cell{Row: 3, Col: 4}, Size{Cols: 20, Rows: 10}); err != nil {
		t.Fatal(err)
	}
	got := out.String()
	if !strings.HasPrefix(got, "\x1b7\x1b[3;4H") {
		t.Fatalf("cursor prefix missing: %q", got[:min(len(got), 20)])
	}
	if !strings.Contains(got, "f=100,a=T,C=1,z=-1,m=1,c=20,r=10;") {
		t.Fatalf("first chunk metadata missing")
	}
	if !strings.Contains(got, "f=100,a=T,C=1,z=-1,m=0;") {
		t.Fatalf("final chunk metadata missing")
	}
	if !strings.HasSuffix(got, "\x1b8") {
		t.Fatalf("cursor restore missing")
	}

	for _, part := range strings.Split(got, "\x1b_G") {
		if !strings.Contains(part, ";") {
			continue
		}
		payload := strings.SplitN(part, ";", 2)[1]
		payload = strings.TrimSuffix(payload, "\x1b\\")
		if len(payload) > maxChunk {
			t.Fatalf("payload chunk too large: %d", len(payload))
		}
	}
}

func TestClear(t *testing.T) {
	var out bytes.Buffer
	if err := Clear(&out); err != nil {
		t.Fatal(err)
	}
	if out.String() != "\x1b_Ga=d,d=A\x1b\\" {
		t.Fatalf("Clear = %q", out.String())
	}
}

func TestParseCellPixels(t *testing.T) {
	size, ok := parseCellPixels("prefix\x1b[6;17;9tsuffix")
	if !ok {
		t.Fatal("parseCellPixels returned false")
	}
	if size.Width != 9 || size.Height != 17 {
		t.Fatalf("size = %+v", size)
	}
}

func TestParseTextAreaReports(t *testing.T) {
	pixels, ok := parseTextAreaPixels("prefix\x1b[4;1560;1800tsuffix")
	if !ok {
		t.Fatal("parseTextAreaPixels returned false")
	}
	if pixels.Width != 1800 || pixels.Height != 1560 {
		t.Fatalf("pixels = %+v", pixels)
	}

	cells, ok := parseTextAreaCells("prefix\x1b[8;78;180tsuffix")
	if !ok {
		t.Fatal("parseTextAreaCells returned false")
	}
	if cells.Cols != 180 || cells.Rows != 78 {
		t.Fatalf("cells = %+v", cells)
	}
}

func TestDeriveCellPixels(t *testing.T) {
	size, ok := deriveCellPixels(
		PixelSize{Width: 1800, Height: 1560},
		Size{Cols: 180, Rows: 78},
	)
	if !ok {
		t.Fatal("deriveCellPixels returned false")
	}
	if size.Width != 10 || size.Height != 20 {
		t.Fatalf("size = %+v", size)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

package kitty

import (
	"encoding/base64"
	"fmt"
	"io"
	"math"
	"os"
	"strconv"
	"strings"
	"time"
)

const maxChunk = 4096

type Cell struct {
	Row int
	Col int
}

type Size struct {
	Cols int
	Rows int
}

type PixelSize struct {
	Width  int
	Height int
}

func Supported(in, out *os.File) bool {
	if in != nil && out != nil && isTerminal(in) {
		if probe(in, out, "\x1b_Gi=31,s=1,v=1,a=q,t=d,f=24;AAAA\x1b\\", ";OK") {
			return true
		}
	}
	return envSupported()
}

func CellPixels(in, out *os.File) PixelSize {
	if in != nil && out != nil && isTerminal(in) {
		if pixelResp, ok := query(in, out, "\x1b[14t", 200*time.Millisecond); ok {
			if cellResp, ok := query(in, out, "\x1b[18t", 200*time.Millisecond); ok {
				pixels, pixelsOK := parseTextAreaPixels(pixelResp)
				cells, cellsOK := parseTextAreaCells(cellResp)
				if pixelsOK && cellsOK {
					if size, ok := deriveCellPixels(pixels, cells); ok {
						return size
					}
				}
			}
		}
		resp, ok := query(in, out, "\x1b[16t", 200*time.Millisecond)
		if ok {
			if size, ok := parseCellPixels(resp); ok {
				return size
			}
		}
	}
	return PixelSize{Width: 8, Height: 16}
}

func Display(w io.Writer, png []byte, cell Cell, size Size) error {
	if cell.Row < 1 {
		cell.Row = 1
	}
	if cell.Col < 1 {
		cell.Col = 1
	}
	if size.Cols < 1 {
		size.Cols = 1
	}
	if size.Rows < 1 {
		size.Rows = 1
	}

	if _, err := fmt.Fprintf(w, "\x1b7\x1b[%d;%dH", cell.Row, cell.Col); err != nil {
		return err
	}
	defer io.WriteString(w, "\x1b8")

	encoded := base64.StdEncoding.EncodeToString(png)
	for offset := 0; offset < len(encoded); offset += maxChunk {
		end := offset + maxChunk
		if end > len(encoded) {
			end = len(encoded)
		}
		more := 0
		if end < len(encoded) {
			more = 1
		}
		prefix := fmt.Sprintf("\x1b_Gf=100,a=T,C=1,z=-1,m=%d", more)
		if offset == 0 {
			prefix += fmt.Sprintf(",c=%d,r=%d", size.Cols, size.Rows)
		}
		if _, err := io.WriteString(w, prefix+";"+encoded[offset:end]+"\x1b\\"); err != nil {
			return err
		}
	}
	return nil
}

func Clear(w io.Writer) error {
	_, err := io.WriteString(w, "\x1b_Ga=d,d=A\x1b\\")
	return err
}

func probe(in, out *os.File, queryText, want string) bool {
	resp, ok := query(in, out, queryText, 200*time.Millisecond)
	return ok && strings.Contains(resp, want)
}

func envSupported() bool {
	if os.Getenv("KITTY_WINDOW_ID") != "" {
		return true
	}
	switch os.Getenv("TERM") {
	case "xterm-kitty", "xterm-ghostty":
		return true
	}
	switch os.Getenv("TERM_PROGRAM") {
	case "WezTerm", "ghostty":
		return true
	}
	return false
}

func parseCellPixels(resp string) (PixelSize, bool) {
	height, width, ok := parseWindowReport(resp, 6)
	if !ok {
		return PixelSize{}, false
	}
	return PixelSize{Width: width, Height: height}, true
}

func parseTextAreaPixels(resp string) (PixelSize, bool) {
	height, width, ok := parseWindowReport(resp, 4)
	if !ok {
		return PixelSize{}, false
	}
	return PixelSize{Width: width, Height: height}, true
}

func parseTextAreaCells(resp string) (Size, bool) {
	rows, cols, ok := parseWindowReport(resp, 8)
	if !ok {
		return Size{}, false
	}
	return Size{Cols: cols, Rows: rows}, true
}

func parseWindowReport(resp string, code int) (int, int, bool) {
	start := strings.LastIndex(resp, fmt.Sprintf("\x1b[%d;", code))
	if start == -1 {
		return 0, 0, false
	}
	rest := resp[start+len(fmt.Sprintf("\x1b[%d;", code)):]
	end := strings.IndexByte(rest, 't')
	if end == -1 {
		return 0, 0, false
	}
	parts := strings.Split(rest[:end], ";")
	if len(parts) != 2 {
		return 0, 0, false
	}
	height, err := strconv.Atoi(parts[0])
	if err != nil || height <= 0 {
		return 0, 0, false
	}
	width, err := strconv.Atoi(parts[1])
	if err != nil || width <= 0 {
		return 0, 0, false
	}
	return height, width, true
}

func deriveCellPixels(pixels PixelSize, cells Size) (PixelSize, bool) {
	if pixels.Width <= 0 || pixels.Height <= 0 || cells.Cols <= 0 || cells.Rows <= 0 {
		return PixelSize{}, false
	}
	width := int(math.Round(float64(pixels.Width) / float64(cells.Cols)))
	height := int(math.Round(float64(pixels.Height) / float64(cells.Rows)))
	if width <= 0 || height <= 0 {
		return PixelSize{}, false
	}
	return PixelSize{Width: width, Height: height}, true
}

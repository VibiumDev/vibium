package api

import (
	"bytes"
	"testing"
)

// ebml builds an element from raw ID bytes, an explicitly encoded size, and
// a payload, so fixtures stay byte-exact.
func ebml(id []byte, size []byte, payload []byte) []byte {
	var b bytes.Buffer
	b.Write(id)
	b.Write(size)
	b.Write(payload)
	return b.Bytes()
}

// sized builds an element with a 1-byte encoded size for payloads < 127.
func sized(id []byte, payload []byte) []byte {
	return ebml(id, []byte{0x80 | byte(len(payload))}, payload)
}

var (
	idEBMLHeader  = []byte{0x1A, 0x45, 0xDF, 0xA3}
	idSegment     = []byte{0x18, 0x53, 0x80, 0x67}
	idSeekHead    = []byte{0x11, 0x4D, 0x9B, 0x74}
	idInfo        = []byte{0x15, 0x49, 0xA9, 0x66}
	idTracks      = []byte{0x16, 0x54, 0xAE, 0x6B}
	idCluster     = []byte{0x1F, 0x43, 0xB6, 0x75}
	idTrackEntry  = []byte{0xAE}
	idVideo       = []byte{0xE0}
	idPixelWidth  = []byte{0xB0}
	idPixelHeight = []byte{0xBA}
	idDispWidth   = []byte{0x54, 0xB0}
	idDispHeight  = []byte{0x54, 0xBA}
	unknownSize   = []byte{0xFF}
)

func videoElement(dims ...[]byte) []byte {
	var payload []byte
	for _, d := range dims {
		payload = append(payload, d...)
	}
	return sized(idVideo, payload)
}

func tracksFor(video []byte) []byte {
	return sized(idTracks, sized(idTrackEntry, video))
}

func TestWebmDimensionsKnownSizeSegment(t *testing.T) {
	video := videoElement(
		sized(idPixelWidth, []byte{0x05, 0x00}),  // 1280
		sized(idPixelHeight, []byte{0x03, 0x72}), // 882
	)
	tracks := tracksFor(video)
	segment := ebml(idSegment, []byte{0x80 | byte(len(tracks))}, tracks)
	file := append(sized(idEBMLHeader, nil), segment...)

	w, h, ok := webmDimensions(file)
	if !ok || w != 1280 || h != 882 {
		t.Fatalf("webmDimensions() = %d, %d, %v, want 1280, 882, true", w, h, ok)
	}
}

// Firefox live-muxes: the Segment declares an unknown size, and SeekHead and
// Info precede Tracks. The walker must enter the unknown-size master and
// skip the unknown siblings by their declared sizes.
func TestWebmDimensionsLiveMuxedSegment(t *testing.T) {
	video := videoElement(
		sized(idPixelWidth, []byte{0x05, 0x00}),
		sized(idPixelHeight, []byte{0x02, 0xD0}), // 720
	)
	var segmentBody []byte
	segmentBody = append(segmentBody, sized(idSeekHead, bytes.Repeat([]byte{0x00}, 12))...)
	segmentBody = append(segmentBody, sized(idInfo, bytes.Repeat([]byte{0x00}, 8))...)
	segmentBody = append(segmentBody, tracksFor(video)...)
	segmentBody = append(segmentBody, ebml(idCluster, unknownSize, []byte{0xDE, 0xAD})...)

	file := append(sized(idEBMLHeader, nil), ebml(idSegment, unknownSize, segmentBody)...)

	w, h, ok := webmDimensions(file)
	if !ok || w != 1280 || h != 720 {
		t.Fatalf("webmDimensions() = %d, %d, %v, want 1280, 720, true", w, h, ok)
	}
}

func TestWebmDimensionsDisplayFallback(t *testing.T) {
	video := videoElement(
		sized(idDispWidth, []byte{0x02, 0x80}),  // 640
		sized(idDispHeight, []byte{0x01, 0xE0}), // 480
	)
	file := append(sized(idEBMLHeader, nil), ebml(idSegment, unknownSize, tracksFor(video))...)

	w, h, ok := webmDimensions(file)
	if !ok || w != 640 || h != 480 {
		t.Fatalf("webmDimensions() = %d, %d, %v, want 640, 480, true", w, h, ok)
	}
}

func TestWebmDimensionsPixelWinsOverDisplay(t *testing.T) {
	video := videoElement(
		sized(idPixelWidth, []byte{0x05, 0x00}),
		sized(idPixelHeight, []byte{0x03, 0x72}),
		sized(idDispWidth, []byte{0x02, 0x80}),
		sized(idDispHeight, []byte{0x01, 0xE0}),
	)
	file := append(sized(idEBMLHeader, nil), ebml(idSegment, unknownSize, tracksFor(video))...)

	w, h, ok := webmDimensions(file)
	if !ok || w != 1280 || h != 882 {
		t.Fatalf("webmDimensions() = %d, %d, %v, want 1280, 882, true", w, h, ok)
	}
}

func TestWebmDimensionsTruncatedBeforeTracks(t *testing.T) {
	var segmentBody []byte
	segmentBody = append(segmentBody, sized(idSeekHead, bytes.Repeat([]byte{0x00}, 12))...)
	file := append(sized(idEBMLHeader, nil), ebml(idSegment, unknownSize, segmentBody)...)

	if _, _, ok := webmDimensions(file); ok {
		t.Fatal("webmDimensions() ok = true for a file truncated before Tracks")
	}
}

func TestWebmDimensionsGarbage(t *testing.T) {
	if _, _, ok := webmDimensions([]byte("not a webm file at all")); ok {
		t.Fatal("webmDimensions() ok = true for garbage input")
	}
	if _, _, ok := webmDimensions(nil); ok {
		t.Fatal("webmDimensions() ok = true for empty input")
	}
}

package watch

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"

	"github.com/vibium/clicker/internal/watch/watchtest"
)

// fixture is watchtest's stream as the file holds it, access units of
// length-prefixed NAL units: 12 frames of 100x60, a key frame with its
// headers every 4.
func fixture(t *testing.T) [][]byte {
	t.Helper()
	var units [][]byte
	for _, f := range watchtest.Frames() {
		var unit []byte
		for _, nal := range bytes.Split(f, []byte{0, 0, 0, 1}) {
			if len(nal) > 0 {
				unit = binary.BigEndian.AppendUint32(unit, uint32(len(nal)))
				unit = append(unit, nal...)
			}
		}
		units = append(units, unit)
	}
	if len(units) != 12 {
		t.Fatalf("the fixture has %d frames, want 12", len(units))
	}
	return units
}

func TestSPSSize(t *testing.T) {
	units := fixture(t)
	_, sps, _ := inspect(units[0])
	if w, h, err := spsSize(sps); err != nil || w != 100 || h != 60 {
		t.Errorf("fixture: %dx%d, %v; want 100x60", w, h, err)
	}
	// What a Roadie sends for an iPhone: 448x976 coded, cropped to 444x972.
	roadie, _ := hex.DecodeString("2742c02895a0701efbb840000003004000000f382800162000005f5e6f7b80789135")
	if w, h, err := spsSize(roadie); err != nil || w != 444 || h != 972 {
		t.Errorf("roadie: %dx%d, %v; want 444x972", w, h, err)
	}
	for _, bad := range [][]byte{nil, {0x67}, {0x68, 1, 2, 3, 4, 5}, {0x67, 0x42, 0xc0, 0x28, 0xff}} {
		if _, _, err := spsSize(bad); err == nil {
			t.Errorf("%x: no error", bad)
		}
	}
}

func TestTimelineKeepsTheSendersSpacing(t *testing.T) {
	// 30 frames a second. Frame 3 arrives 50 ms late; the sender pauses for
	// a second before frame 6 and its clock only moves on by one frame.
	const step = clock / 30
	var frames []frame
	for i := int64(0); i < 10; i++ {
		f := frame{rtp: i * step, wall: i * step}
		if i == 3 {
			f.wall += clock / 20
		}
		if i >= 6 {
			f.wall += clock
		}
		frames = append(frames, f)
	}
	times, shift := timeline(frames, 10*clock)
	if shift != 0 {
		t.Errorf("shift %d, want 0", shift)
	}
	for i, got := range times {
		want := int64(i) * step
		if i >= 6 {
			want += clock
		}
		if got != want {
			t.Errorf("frame %d at %d, want %d", i, got, want)
		}
	}
}

func TestTimelineStartsAtZeroAndMovesForward(t *testing.T) {
	// The first frame arrives 30 ms late, so it belongs 30 ms earlier.
	const late = clock * 30 / 1000
	frames := []frame{{rtp: 0, wall: 0}, {rtp: 3000, wall: 3000 - late}, {rtp: 6000, wall: 6000 - late}, {rtp: 6000, wall: 6000 - late}}
	times, shift := timeline(frames, 10*clock)
	if shift != -late {
		t.Errorf("shift %d, want %d", shift, -late)
	}
	if want := []int64{0, 3000, 6000, 6001}; !equal(times, want) {
		t.Errorf("times %v, want %v", times, want)
	}
}

func equal(a, b []int64) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// boxes walks an MP4's boxes, depth first, into a map by path.
func boxes(t *testing.T, data []byte, path string, into map[string][]byte) {
	t.Helper()
	for len(data) >= 8 {
		size, kind, head := int(binary.BigEndian.Uint32(data)), string(data[4:8]), 8
		if size == 1 {
			size, head = int(binary.BigEndian.Uint64(data[8:])), 16
		}
		if size < head || size > len(data) {
			t.Fatalf("%s/%s: size %d of %d", path, kind, size, len(data))
		}
		body := data[head:size]
		into[path+kind] = body
		switch kind {
		case "moov", "trak", "mdia", "minf", "stbl":
			boxes(t, body, path+kind+"/", into)
		}
		data = data[size:]
	}
	if len(data) != 0 {
		t.Fatalf("%s: %d stray bytes", path, len(data))
	}
}

func TestMP4HoldsTheFramesAndTheirIndex(t *testing.T) {
	units := fixture(t)
	path := filepath.Join(t.TempDir(), "out.mp4")
	m, err := createMP4(path)
	if err != nil {
		t.Fatal(err)
	}
	var payload []byte
	for i, u := range units {
		key, sps, pps := inspect(u)
		if key && m.sps == nil {
			m.sps, m.pps = sps, pps
		}
		m.add(u, frame{key: key, rtp: int64(i) * 3000, wall: int64(i) * 3000})
		payload = append(payload, u...)
	}
	// Stopped half a second after the last frame arrived.
	info, err := m.finish(11*3000 + clock/2)
	if err != nil {
		t.Fatal(err)
	}
	if info.Width != 100 || info.Height != 60 || info.Frames != 12 {
		t.Errorf("info %+v", info)
	}
	if want := int64(11*3000 + clock/2); info.Duration != want {
		t.Errorf("duration %d, want %d", info.Duration, want)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if int64(len(data)) != info.Bytes {
		t.Errorf("file is %d bytes, info says %d", len(data), info.Bytes)
	}
	got := map[string][]byte{}
	boxes(t, data, "", got)
	if !bytes.Equal(got["mdat"], payload) {
		t.Error("mdat is not the frames as given")
	}
	stbl := "moov/trak/mdia/minf/stbl/"
	if n := binary.BigEndian.Uint32(got[stbl+"stsz"][8:]); n != 12 {
		t.Errorf("stsz counts %d samples", n)
	}
	if size := binary.BigEndian.Uint32(got[stbl+"stsz"][12:]); int(size) != len(units[0]) {
		t.Errorf("first sample size %d, want %d", size, len(units[0]))
	}
	// Key frames are samples 1, 5 and 9.
	stss := got[stbl+"stss"]
	if n := binary.BigEndian.Uint32(stss[4:]); n != 3 || binary.BigEndian.Uint32(stss[8:]) != 1 || binary.BigEndian.Uint32(stss[16:]) != 9 {
		t.Errorf("stss % x", stss)
	}
	// Eleven frames of 3000 ticks, then the last one held to the stop.
	stts := got[stbl+"stts"]
	if n := binary.BigEndian.Uint32(stts[4:]); n != 2 || binary.BigEndian.Uint32(stts[8:]) != 11 || binary.BigEndian.Uint32(stts[20:]) != clock/2 {
		t.Errorf("stts % x", stts)
	}
	if off := binary.BigEndian.Uint32(got[stbl+"stco"][8:]); !bytes.Equal(data[off:int(off)+len(units[0])], units[0]) {
		t.Errorf("stco points at %d, which is not the first frame", off)
	}
	if !bytes.Contains(got[stbl+"stsd"], m.sps) || !bytes.Contains(got[stbl+"stsd"], []byte("avcC")) {
		t.Error("stsd has no avcC with the parameter set")
	}
	if d := binary.BigEndian.Uint32(got["moov/trak/mdia/mdhd"][16:]); int64(d) != info.Duration {
		t.Errorf("mdhd duration %d", d)
	}
}

func TestMP4WithNoFramesIsAnError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "out.mp4")
	m, err := createMP4(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := m.finish(0); err == nil {
		t.Error("no error for a file without frames")
	}
}

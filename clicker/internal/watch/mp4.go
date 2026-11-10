package watch

import (
	"bufio"
	"encoding/binary"
	"errors"
	"os"
)

// clock is the MP4 track's timescale, and RTP's for video.
const clock = 90000

// mdatHeader is the 64-bit form of the box header, so the size written at
// the end never outgrows it.
const mdatHeader = 16

// frame is one H.264 access unit in the file.
type frame struct {
	size uint32
	key  bool
	rtp  int64 // ticks since the first frame, by the sender's clock
	wall int64 // ticks since the first frame, by its arrival here
}

// mp4File writes one H.264 video track: the frames as they arrive, the
// index once at the end. Nothing is re-encoded.
type mp4File struct {
	path     string
	f        *os.File
	w        *bufio.Writer
	frames   []frame
	sps, pps []byte
	bytes    int64
	head     int64
	err      error
}

func createMP4(path string) (*mp4File, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0o600)
	if err != nil {
		return nil, err
	}
	m := &mp4File{path: path, f: f, w: bufio.NewWriterSize(f, 64<<10)}
	ftyp := box("ftyp", []byte("isom"), u32(0x200), []byte("isomiso2avc1mp41"))
	m.head = int64(len(ftyp)) + mdatHeader
	m.w.Write(ftyp)
	m.w.Write(u32(1))
	m.w.WriteString("mdat")
	_, m.err = m.w.Write(make([]byte, 8))
	return m, m.err
}

// add appends an access unit of length-prefixed NAL units.
func (m *mp4File) add(data []byte, fr frame) {
	if m.err != nil || len(data) == 0 {
		return
	}
	fr.size = uint32(len(data))
	if _, m.err = m.w.Write(data); m.err != nil {
		return
	}
	m.bytes += int64(len(data))
	m.frames = append(m.frames, fr)
}

// mp4Info describes a finished file.
type mp4Info struct {
	Width, Height int
	Duration      int64 // ticks
	Shift         int64 // ticks the first frame moved from its arrival
	Frames        int
	Bytes         int64
}

// finish writes the index. end is when recording stopped, in ticks since
// the first frame arrived.
func (m *mp4File) finish(end int64) (mp4Info, error) {
	defer m.f.Close()
	if m.err != nil {
		return mp4Info{}, m.err
	}
	if len(m.frames) == 0 || m.sps == nil || m.pps == nil {
		return mp4Info{}, errors.New("no video arrived")
	}
	width, height, err := spsSize(m.sps)
	if err != nil {
		return mp4Info{}, err
	}
	times, shift := timeline(m.frames, 10*clock)
	durs := make([]uint32, len(times))
	var total int64
	for i := range times {
		d := int64(clock / 30)
		if i+1 < len(times) {
			d = times[i+1] - times[i]
		} else if rest := end - shift - times[i]; rest > 0 {
			// The last frame stays up until the stop, or a second if the
			// stream ended long before.
			d = min(rest, clock)
		}
		durs[i] = uint32(d)
		total += d
	}

	moov := m.moov(width, height, durs, total)
	if _, err := m.w.Write(moov); err != nil {
		return mp4Info{}, err
	}
	if err := m.w.Flush(); err != nil {
		return mp4Info{}, err
	}
	size := make([]byte, 8)
	binary.BigEndian.PutUint64(size, uint64(mdatHeader+m.bytes))
	if _, err := m.f.WriteAt(size, m.head-8); err != nil {
		return mp4Info{}, err
	}
	return mp4Info{
		Width: width, Height: height, Duration: total, Shift: shift,
		Frames: len(m.frames), Bytes: m.head + m.bytes + int64(len(moov)),
	}, nil
}

// discard drops the file.
func (m *mp4File) discard() {
	m.f.Close()
	os.Remove(m.path)
}

// timeline places each frame on the receiver's clock. The sender's
// timestamps give the spacing; arrival times give the position, taken as
// the least delay among the frames of the following window, so a frame
// held up on the way keeps its place and a sender that paused moves on.
// The first frame is at zero; shift is how far that moved it from its
// arrival.
func timeline(frames []frame, window int64) (times []int64, shift int64) {
	times = make([]int64, len(frames))
	for i, f := range frames {
		off := f.wall - f.rtp
		for j := i + 1; j < len(frames) && frames[j].wall-f.wall <= window; j++ {
			off = min(off, frames[j].wall-frames[j].rtp)
		}
		times[i] = f.rtp + off
		if i > 0 && times[i] <= times[i-1] {
			times[i] = times[i-1] + 1
		}
	}
	if len(times) > 0 {
		shift = times[0]
		for i := range times {
			times[i] -= shift
		}
	}
	return times, shift
}

func (m *mp4File) moov(width, height int, durs []uint32, total int64) []byte {
	matrix := cat(u32(0x10000), u32(0), u32(0), u32(0), u32(0x10000), u32(0), u32(0), u32(0), u32(0x40000000))
	ms := uint32(total * 1000 / clock)

	mvhd := full("mvhd", 0, 0, u32(0), u32(0), u32(1000), u32(ms),
		u32(0x10000), u16(0x100), make([]byte, 10), matrix, make([]byte, 24), u32(2))
	tkhd := full("tkhd", 0, 3, u32(0), u32(0), u32(1), u32(0), u32(ms),
		make([]byte, 8), u16(0), u16(0), u16(0), u16(0), matrix, u32(uint32(width)<<16), u32(uint32(height)<<16))
	mdhd := full("mdhd", 0, 0, u32(0), u32(0), u32(clock), u32(uint32(total)), u16(0x55c4), u16(0))
	hdlr := full("hdlr", 0, 0, u32(0), []byte("vide"), make([]byte, 12), []byte("VideoHandler\x00"))
	vmhd := full("vmhd", 0, 1, make([]byte, 8))
	dinf := box("dinf", full("dref", 0, 0, u32(1), full("url ", 0, 1)))

	avcC := box("avcC", []byte{1, m.sps[1], m.sps[2], m.sps[3], 0xff, 0xe1},
		u16(uint16(len(m.sps))), m.sps, []byte{1}, u16(uint16(len(m.pps))), m.pps)
	avc1 := box("avc1", make([]byte, 6), u16(1), make([]byte, 16),
		u16(uint16(width)), u16(uint16(height)), u32(0x480000), u32(0x480000), u32(0), u16(1),
		make([]byte, 32), u16(0x18), u16(0xffff), avcC)
	stsd := full("stsd", 0, 0, u32(1), avc1)

	var stts, stss, stsz []byte
	runs, keys := uint32(0), uint32(0)
	for i := 0; i < len(durs); {
		j := i
		for j < len(durs) && durs[j] == durs[i] {
			j++
		}
		stts = append(stts, cat(u32(uint32(j-i)), u32(durs[i]))...)
		runs++
		i = j
	}
	for i, f := range m.frames {
		if f.key {
			stss = append(stss, u32(uint32(i+1))...)
			keys++
		}
		stsz = append(stsz, u32(f.size)...)
	}
	n := uint32(len(m.frames))
	stbl := box("stbl", stsd,
		full("stts", 0, 0, u32(runs), stts),
		full("stss", 0, 0, u32(keys), stss),
		full("stsc", 0, 0, u32(1), u32(1), u32(n), u32(1)),
		full("stsz", 0, 0, u32(0), u32(n), stsz),
		full("stco", 0, 0, u32(1), u32(uint32(m.head))))

	trak := box("trak", tkhd, box("mdia", mdhd, hdlr, box("minf", vmhd, dinf, stbl)))
	return box("moov", mvhd, trak)
}

func u16(v uint16) []byte { return binary.BigEndian.AppendUint16(nil, v) }
func u32(v uint32) []byte { return binary.BigEndian.AppendUint32(nil, v) }

func cat(parts ...[]byte) []byte {
	var out []byte
	for _, p := range parts {
		out = append(out, p...)
	}
	return out
}

func box(kind string, parts ...[]byte) []byte {
	body := cat(parts...)
	return cat(u32(uint32(8+len(body))), []byte(kind), body)
}

func full(kind string, version byte, flags uint32, parts ...[]byte) []byte {
	head := u32(flags)
	head[0] = version
	return box(kind, append([][]byte{head}, parts...)...)
}

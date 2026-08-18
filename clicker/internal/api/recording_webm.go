package api

import "math/bits"

// webmDimensions extracts the video track dimensions from a WebM file's
// Tracks element, so the recording reports what the encoder actually
// produced rather than the viewport it was asked for (#358). ok is false
// when the header cannot be parsed.
//
// The walk descends only the masters on the path to the dimensions —
// Segment, Tracks, TrackEntry, Video — and skips unknown siblings
// (SeekHead, Void, Info) by their declared size. Firefox live-muxes, so
// the Segment usually declares an unknown size; such masters are entered
// and taken to run to the end of the scan window. The scan stops at the
// first Cluster: media data never precedes the track header.
func webmDimensions(data []byte) (width, height int, ok bool) {
	const scanLimit = 64 * 1024
	if len(data) > scanLimit {
		data = data[:scanLimit]
	}
	if len(data) < 4 || data[0] != 0x1A || data[1] != 0x45 || data[2] != 0xDF || data[3] != 0xA3 {
		return 0, 0, false
	}

	descend := map[uint64]bool{
		0x18538067: true, // Segment
		0x1654AE6B: true, // Tracks
		0xAE:       true, // TrackEntry
		0xE0:       true, // Video
	}

	// PixelWidth/PixelHeight describe the encoded stream; DisplayWidth/
	// DisplayHeight are the fallback when a muxer omits them. The first
	// video track wins.
	var pw, ph, dw, dh uint64

	var walk func(buf []byte)
	walk = func(buf []byte) {
		for len(buf) > 0 {
			id, idn := readEBMLID(buf)
			if idn == 0 {
				return
			}
			size, sn, unknown := readEBMLSize(buf[idn:])
			if sn == 0 {
				return
			}
			if id == 0x1F43B675 { // Cluster
				return
			}
			body := buf[idn+sn:]
			if descend[id] {
				if unknown || uint64(len(body)) <= size {
					// Unknown-size or truncated master: its children run to
					// the end of the window, and nothing follows it there.
					walk(body)
					return
				}
				walk(body[:size])
				buf = body[size:]
				continue
			}
			if unknown || uint64(len(body)) < size {
				return
			}
			switch id {
			case 0xB0: // PixelWidth
				if pw == 0 {
					pw = readEBMLUint(body, size)
				}
			case 0xBA: // PixelHeight
				if ph == 0 {
					ph = readEBMLUint(body, size)
				}
			case 0x54B0: // DisplayWidth
				if dw == 0 {
					dw = readEBMLUint(body, size)
				}
			case 0x54BA: // DisplayHeight
				if dh == 0 {
					dh = readEBMLUint(body, size)
				}
			}
			buf = body[size:]
		}
	}
	walk(data)

	if pw > 0 && ph > 0 {
		return int(pw), int(ph), true
	}
	if dw > 0 && dh > 0 {
		return int(dw), int(dh), true
	}
	return 0, 0, false
}

// readEBMLID reads an EBML element ID, marker bits included, returning the
// bytes consumed. n is 0 when the buffer does not hold a valid ID.
func readEBMLID(buf []byte) (id uint64, n int) {
	if len(buf) == 0 {
		return 0, 0
	}
	n = bits.LeadingZeros8(buf[0]) + 1
	if n > 4 || len(buf) < n {
		return 0, 0
	}
	for i := 0; i < n; i++ {
		id = id<<8 | uint64(buf[i])
	}
	return id, n
}

// readEBMLSize reads an EBML size vint, returning the bytes consumed and
// whether the size is the reserved "unknown" value (all value bits set).
// n is 0 when the buffer does not hold a valid size.
func readEBMLSize(buf []byte) (size uint64, n int, unknown bool) {
	if len(buf) == 0 {
		return 0, 0, false
	}
	n = bits.LeadingZeros8(buf[0]) + 1
	if n > 8 || len(buf) < n {
		return 0, 0, false
	}
	size = uint64(buf[0]) & (0xFF >> n)
	allOnes := size == uint64(0xFF)>>n
	for i := 1; i < n; i++ {
		size = size<<8 | uint64(buf[i])
		if buf[i] != 0xFF {
			allOnes = false
		}
	}
	return size, n, allOnes
}

// readEBMLUint reads a big-endian EBML unsigned integer payload.
func readEBMLUint(buf []byte, size uint64) uint64 {
	if size == 0 || size > 8 || uint64(len(buf)) < size {
		return 0
	}
	var v uint64
	for i := uint64(0); i < size; i++ {
		v = v<<8 | uint64(buf[i])
	}
	return v
}

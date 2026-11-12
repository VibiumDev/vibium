package watch

import "errors"

// bits reads an H.264 parameter set bit by bit, emulation prevention
// bytes already removed.
type bits struct {
	b   []byte
	pos int
	err bool
}

func (r *bits) bit() uint32 {
	if r.pos >= len(r.b)*8 {
		r.err = true
		return 0
	}
	v := r.b[r.pos/8] >> (7 - r.pos%8) & 1
	r.pos++
	return uint32(v)
}

func (r *bits) read(n int) uint32 {
	var v uint32
	for i := 0; i < n; i++ {
		v = v<<1 | r.bit()
	}
	return v
}

// ue reads an unsigned Exp-Golomb number.
func (r *bits) ue() uint32 {
	zeros := 0
	for r.bit() == 0 {
		if r.err || zeros > 31 {
			r.err = true
			return 0
		}
		zeros++
	}
	return 1<<zeros - 1 + r.read(zeros)
}

// se reads a signed Exp-Golomb number.
func (r *bits) se() int32 {
	v := r.ue()
	if v&1 == 1 {
		return int32(v/2 + 1)
	}
	return -int32(v / 2)
}

// unescape drops the 0x03 an encoder puts after two zero bytes.
func unescape(nal []byte) []byte {
	out := make([]byte, 0, len(nal))
	zeros := 0
	for _, c := range nal {
		if zeros >= 2 && c == 3 {
			zeros = 0
			continue
		}
		if c == 0 {
			zeros++
		} else {
			zeros = 0
		}
		out = append(out, c)
	}
	return out
}

// spsSize is the picture size an H.264 sequence parameter set declares,
// cropping applied. The NAL starts at its header byte.
func spsSize(nal []byte) (width, height int, err error) {
	if len(nal) < 5 || nal[0]&0x1f != 7 {
		return 0, 0, errors.New("not a sequence parameter set")
	}
	r := &bits{b: unescape(nal[1:])}
	profile := r.read(8)
	r.read(16) // constraint flags and level
	r.ue()     // seq_parameter_set_id
	chroma := uint32(1)
	switch profile {
	case 100, 110, 122, 244, 44, 83, 86, 118, 128, 138, 139, 134, 135:
		chroma = r.ue()
		if chroma == 3 {
			r.bit() // separate_colour_plane_flag
		}
		r.ue()  // bit_depth_luma_minus8
		r.ue()  // bit_depth_chroma_minus8
		r.bit() // qpprime_y_zero_transform_bypass_flag
		if r.bit() == 1 {
			lists := 8
			if chroma == 3 {
				lists = 12
			}
			for i := 0; i < lists; i++ {
				if r.bit() == 0 {
					continue
				}
				size := 16
				if i >= 6 {
					size = 64
				}
				last, next := int32(8), int32(8)
				for j := 0; j < size; j++ {
					if next != 0 {
						next = (last + r.se() + 256) % 256
					}
					if next != 0 {
						last = next
					}
				}
			}
		}
	}
	r.ue() // log2_max_frame_num_minus4
	switch r.ue() {
	case 0:
		r.ue()
	case 1:
		r.bit()
		r.se()
		r.se()
		for n := r.ue(); n > 0 && !r.err; n-- {
			r.se()
		}
	}
	r.ue()  // max_num_ref_frames
	r.bit() // gaps_in_frame_num_value_allowed_flag
	mbsWide := r.ue() + 1
	mapHigh := r.ue() + 1
	frameOnly := r.bit()
	if frameOnly == 0 {
		r.bit() // mb_adaptive_frame_field_flag
	}
	r.bit() // direct_8x8_inference_flag
	var left, right, top, bottom uint32
	if r.bit() == 1 {
		left, right, top, bottom = r.ue(), r.ue(), r.ue(), r.ue()
	}
	if r.err {
		return 0, 0, errors.New("the sequence parameter set is cut short")
	}
	unitX, unitY := uint32(1), 2-frameOnly
	if chroma == 1 || chroma == 2 {
		unitX = 2
	}
	if chroma == 1 {
		unitY *= 2
	}
	w := int64(mbsWide)*16 - int64(unitX)*int64(left+right)
	h := int64(2-frameOnly)*int64(mapHigh)*16 - int64(unitY)*int64(top+bottom)
	if w <= 0 || h <= 0 || w > 16384 || h > 16384 {
		return 0, 0, errors.New("the sequence parameter set declares no usable size")
	}
	return int(w), int(h), nil
}

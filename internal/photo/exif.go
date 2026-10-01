// Package photo loads images and turns them into fingerprints that stay
// close when a photo is resized, recompressed or lightly edited.
package photo

import (
	"bytes"
	"encoding/binary"
	"io"
)

// Orientation reads the EXIF orientation tag (1 to 8) from a JPEG. It
// returns 1, meaning "already upright", when there is no EXIF block or it
// can't be parsed. Phones usually store pixels sideways and set this tag
// instead of rotating them, so ignoring it would make a photo and its
// rotated export look like different pictures.
func Orientation(r io.Reader) int {
	var head [2]byte
	if _, err := io.ReadFull(r, head[:]); err != nil || head != [2]byte{0xFF, 0xD8} {
		return 1
	}
	for {
		var m [4]byte
		if _, err := io.ReadFull(r, m[:]); err != nil || m[0] != 0xFF {
			return 1
		}
		marker := m[1]
		size := int(binary.BigEndian.Uint16(m[2:])) - 2
		if marker == 0xDA || marker == 0xD9 || size < 0 {
			return 1 // start of image data: no EXIF before it
		}
		seg := make([]byte, size)
		if _, err := io.ReadFull(r, seg); err != nil {
			return 1
		}
		if marker == 0xE1 && bytes.HasPrefix(seg, []byte("Exif\x00\x00")) {
			return tiffOrientation(seg[6:])
		}
	}
}

func tiffOrientation(t []byte) int {
	if len(t) < 8 {
		return 1
	}
	var bo binary.ByteOrder
	switch string(t[:2]) {
	case "II":
		bo = binary.LittleEndian
	case "MM":
		bo = binary.BigEndian
	default:
		return 1
	}
	ifd := int(bo.Uint32(t[4:]))
	if ifd+2 > len(t) {
		return 1
	}
	n := int(bo.Uint16(t[ifd:]))
	for i := 0; i < n; i++ {
		e := ifd + 2 + i*12
		if e+12 > len(t) {
			return 1
		}
		if bo.Uint16(t[e:]) == 0x0112 {
			if v := int(bo.Uint16(t[e+8:])); v >= 1 && v <= 8 {
				return v
			}
			return 1
		}
	}
	return 1
}

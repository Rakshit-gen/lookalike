package photo

import (
	"bytes"
	"encoding/binary"
	"testing"
)

// jpegWithOrientation builds the start of a JPEG holding only an APP1
// EXIF segment with one IFD entry, then the start-of-scan marker.
func jpegWithOrientation(bo binary.ByteOrder, mark string, o uint16) []byte {
	var tiff bytes.Buffer
	tiff.WriteString(mark)
	binary.Write(&tiff, bo, uint16(42))
	binary.Write(&tiff, bo, uint32(8)) // IFD0 right after the header
	binary.Write(&tiff, bo, uint16(1)) // one entry
	binary.Write(&tiff, bo, uint16(0x0112))
	binary.Write(&tiff, bo, uint16(3)) // SHORT
	binary.Write(&tiff, bo, uint32(1))
	binary.Write(&tiff, bo, o)
	binary.Write(&tiff, bo, uint16(0))
	binary.Write(&tiff, bo, uint32(0)) // no next IFD

	seg := append([]byte("Exif\x00\x00"), tiff.Bytes()...)
	var b bytes.Buffer
	b.Write([]byte{0xFF, 0xD8})
	b.Write([]byte{0xFF, 0xE0, 0x00, 0x04, 0x00, 0x00}) // an APP0 to skip past
	b.Write([]byte{0xFF, 0xE1})
	binary.Write(&b, binary.BigEndian, uint16(len(seg)+2))
	b.Write(seg)
	b.Write([]byte{0xFF, 0xDA, 0x00, 0x02})
	return b.Bytes()
}

func TestOrientation(t *testing.T) {
	for _, c := range []struct {
		name string
		data []byte
		want int
	}{
		{"little endian", jpegWithOrientation(binary.LittleEndian, "II", 6), 6},
		{"big endian", jpegWithOrientation(binary.BigEndian, "MM", 8), 8},
		{"out of range", jpegWithOrientation(binary.BigEndian, "MM", 42), 1},
		{"not a jpeg", []byte("\x89PNG\r\n"), 1},
		{"truncated", jpegWithOrientation(binary.LittleEndian, "II", 6)[:20], 1},
		{"empty", nil, 1},
	} {
		if got := Orientation(bytes.NewReader(c.data)); got != c.want {
			t.Errorf("%s: got %d, want %d", c.name, got, c.want)
		}
	}
}

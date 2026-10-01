package photo

import (
	"bufio"
	"bytes"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"

	_ "golang.org/x/image/webp"
	"os"
)

// Info is what lookalike needs to know about one photo.
type Info struct {
	Width, Height int // as displayed, after EXIF orientation
	Vector        []float32
}

// Load decodes a JPEG, PNG, GIF or WebP file and fingerprints it.
func Load(path string) (Info, error) {
	f, err := os.Open(path)
	if err != nil {
		return Info{}, err
	}
	defer f.Close()

	br := bufio.NewReaderSize(f, 64<<10)
	head, _ := br.Peek(64 << 10)
	orient := 1
	if len(head) > 2 && head[0] == 0xFF && head[1] == 0xD8 {
		orient = Orientation(bytes.NewReader(head)) // EXIF sits in the first few KB
	}
	img, _, err := image.Decode(br)
	if err != nil {
		return Info{}, fmt.Errorf("%s: %w", path, err)
	}
	w, h := img.Bounds().Dx(), img.Bounds().Dy()
	if orient >= 5 { // 5 to 8 swap width and height
		w, h = h, w
	}
	return Info{Width: w, Height: h, Vector: Reduce(img).Orient(orient).Fingerprint()}, nil
}

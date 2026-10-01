package variant

import (
	"image"
	"image/color"
	"testing"
)

func checker() *image.RGBA {
	m := image.NewRGBA(image.Rect(0, 0, 40, 20))
	for y := 0; y < 20; y++ {
		for x := 0; x < 40; x++ {
			m.Set(x, y, color.RGBA{uint8(x * 6), uint8(y * 12), 0, 255})
		}
	}
	return m
}

func TestSizes(t *testing.T) {
	m := checker()
	for _, c := range []struct {
		name string
		img  image.Image
		w, h int
	}{
		{"half", Half(m), 20, 10},
		{"crop", Crop(m, 0.1), 32, 16},
		{"rotate", Rotate90(m), 20, 40},
		{"brighten", Brighten(m, 10), 40, 20},
	} {
		if b := c.img.Bounds(); b.Dx() != c.w || b.Dy() != c.h {
			t.Errorf("%s: %dx%d, want %dx%d", c.name, b.Dx(), b.Dy(), c.w, c.h)
		}
	}
}

func TestRotate90IsClockwise(t *testing.T) {
	m := checker()
	r := Rotate90(m)
	// The original's top-left pixel ends up top-right.
	if r.At(19, 0) != m.At(0, 0) {
		t.Fatal("not a clockwise quarter turn")
	}
}

func TestBrightenClips(t *testing.T) {
	m := image.NewRGBA(image.Rect(0, 0, 1, 1))
	m.Set(0, 0, color.RGBA{250, 10, 0, 255})
	r, g, _, _ := Brighten(m, 30).At(0, 0).RGBA()
	if r>>8 != 255 || g>>8 != 40 {
		t.Fatalf("got r=%d g=%d", r>>8, g>>8)
	}
}

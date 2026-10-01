package photo

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"testing"
)

// scene draws something photo-like: a sky gradient, a sun and a hill.
func scene(w, h int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			fx, fy := float64(x)/float64(w), float64(y)/float64(h)
			c := color.RGBA{uint8(80 + 100*fy), uint8(120 + 80*fy), 230, 255}
			if math.Hypot(fx-0.7, fy-0.25) < 0.1 {
				c = color.RGBA{250, 220, 90, 255}
			}
			if fy > 0.6+0.15*math.Sin(fx*6) {
				c = color.RGBA{40, uint8(110 + 60*fx), 50, 255}
			}
			img.Set(x, y, c)
		}
	}
	return img
}

func write(t *testing.T, path string, img image.Image, quality int) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if quality == 0 {
		err = png.Encode(f, img)
	} else {
		err = jpeg.Encode(f, img, &jpeg.Options{Quality: quality})
	}
	if err != nil {
		t.Fatal(err)
	}
}

func half(src image.Image) *image.RGBA {
	b := src.Bounds()
	out := image.NewRGBA(image.Rect(0, 0, b.Dx()/2, b.Dy()/2))
	for y := 0; y < b.Dy()/2; y++ {
		for x := 0; x < b.Dx()/2; x++ {
			out.Set(x, y, src.At(2*x, 2*y))
		}
	}
	return out
}

func load(t *testing.T, path string) Info {
	t.Helper()
	info, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	return info
}

func TestLoadMatchesCopiesOfTheSamePhoto(t *testing.T) {
	dir := t.TempDir()
	orig := scene(640, 480)
	write(t, filepath.Join(dir, "a.jpg"), orig, 92)
	write(t, filepath.Join(dir, "b.png"), orig, 0)
	write(t, filepath.Join(dir, "c.jpg"), half(orig), 40)

	a := load(t, filepath.Join(dir, "a.jpg"))
	if a.Width != 640 || a.Height != 480 {
		t.Fatalf("size %dx%d", a.Width, a.Height)
	}
	for _, name := range []string{"b.png", "c.jpg"} {
		if c := cosine(a.Vector, load(t, filepath.Join(dir, name)).Vector); c < 0.98 {
			t.Errorf("%s similarity %.4f", name, c)
		}
	}
}

func TestLoadAppliesOrientation(t *testing.T) {
	dir := t.TempDir()
	upright := scene(400, 300)
	write(t, filepath.Join(dir, "upright.jpg"), upright, 92)

	// Store the pixels turned 90 degrees counter clockwise and tag them
	// with orientation 6, the way a phone held upright does.
	b := upright.Bounds()
	stored := image.NewRGBA(image.Rect(0, 0, b.Dy(), b.Dx()))
	for y := 0; y < b.Dy(); y++ {
		for x := 0; x < b.Dx(); x++ {
			stored.Set(y, b.Dx()-1-x, upright.At(x, y))
		}
	}
	var buf bytes.Buffer
	jpeg.Encode(&buf, stored, &jpeg.Options{Quality: 92})
	raw := buf.Bytes()
	exif := jpegWithOrientation(binary.LittleEndian, "II", 6)
	app1 := exif[2+6 : len(exif)-4] // just the APP1 segment
	tagged := append(append([]byte{0xFF, 0xD8}, app1...), raw[2:]...)
	os.WriteFile(filepath.Join(dir, "rotated.jpg"), tagged, 0o644)

	r := load(t, filepath.Join(dir, "rotated.jpg"))
	if r.Width != 400 || r.Height != 300 {
		t.Fatalf("displayed size %dx%d", r.Width, r.Height)
	}
	if c := cosine(load(t, filepath.Join(dir, "upright.jpg")).Vector, r.Vector); c < 0.98 {
		t.Fatalf("rotated copy similarity %.4f", c)
	}
}

func TestLoadRejectsNonImages(t *testing.T) {
	p := filepath.Join(t.TempDir(), "notes.jpg")
	os.WriteFile(p, []byte("not an image"), 0o644)
	if _, err := Load(p); err == nil {
		t.Fatal("expected a decode error")
	}
}

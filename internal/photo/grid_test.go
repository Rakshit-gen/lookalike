package photo

import (
	"image"
	"image/color"
	"math"
	"testing"
)

func TestReduceAveragesBlocks(t *testing.T) {
	// Left half black, right half white, at a size that isn't a multiple of N.
	img := image.NewGray(image.Rect(0, 0, 100, 50))
	for y := 0; y < 50; y++ {
		for x := 50; x < 100; x++ {
			img.SetGray(x, y, color.Gray{255})
		}
	}
	g := Reduce(img)
	if g[0] != 0 || math.Abs(g[N-1]-255) > 1e-9 {
		t.Fatalf("corners %v %v", g[0], g[N-1])
	}
}

func TestReduceYCbCrMatchesGeneric(t *testing.T) {
	yc := image.NewYCbCr(image.Rect(0, 0, 64, 40), image.YCbCrSubsampleRatio420)
	for i := range yc.Y {
		yc.Y[i] = uint8(i * 7)
	}
	for i := range yc.Cb {
		yc.Cb[i], yc.Cr[i] = 128, 128 // gray, so luma equals the generic path
	}
	fast := Reduce(yc)
	slow := Reduce(struct{ image.Image }{yc}) // hides the concrete type
	for i := range fast {
		if math.Abs(fast[i]-slow[i]) > 1.5 {
			t.Fatalf("cell %d: fast %.1f slow %.1f", i, fast[i], slow[i])
		}
	}
}

func TestOrientRotationsRoundTrip(t *testing.T) {
	var g Grid
	for i := range g {
		g[i] = float64(i)
	}
	// 6 then 8 is a quarter turn one way and back.
	if back := g.Orient(6).Orient(8); *back != g {
		t.Fatal("6 then 8 is not the identity")
	}
	if back := g.Orient(3).Orient(3); *back != g {
		t.Fatal("3 twice is not the identity")
	}
	if same := g.Orient(1); *same != g {
		t.Fatal("1 changed the grid")
	}
	// Orientation 6: the camera's top-left pixel ends up top-right.
	if g.Orient(6)[N-1] != g[0] {
		t.Fatal("orientation 6 placed the corner wrong")
	}
}

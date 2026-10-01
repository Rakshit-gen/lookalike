package photo

import (
	"math"
	"testing"
)

func cosine(a, b []float32) float64 {
	var dot, na, nb float64
	for i := range a {
		dot += float64(a[i]) * float64(b[i])
		na += float64(a[i]) * float64(a[i])
		nb += float64(b[i]) * float64(b[i])
	}
	return dot / math.Sqrt(na*nb)
}

func gradient() *Grid {
	var g Grid
	for y := 0; y < N; y++ {
		for x := 0; x < N; x++ {
			g[y*N+x] = float64(x*4) + 40*math.Sin(float64(y)/5)
		}
	}
	return &g
}

func TestFingerprintIgnoresBrightnessAndContrast(t *testing.T) {
	g := gradient()
	var edited Grid
	for i, v := range g {
		edited[i] = v*0.7 + 30
	}
	if c := cosine(g.Fingerprint(), edited.Fingerprint()); c < 0.9999 {
		t.Fatalf("similarity after brightness and contrast change %.5f", c)
	}
}

func TestFingerprintSeparatesDifferentLayouts(t *testing.T) {
	g := gradient()
	flipped := g.Orient(3)
	if c := cosine(g.Fingerprint(), flipped.Fingerprint()); c > 0.5 {
		t.Fatalf("upside-down image scored %.3f", c)
	}
}

func TestFingerprintLength(t *testing.T) {
	if n := len(gradient().Fingerprint()); n != Dim {
		t.Fatalf("got %d values", n)
	}
}

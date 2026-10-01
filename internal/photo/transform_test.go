package photo

import (
	"math"
	"testing"
)

func TestTransformMatchesReorientedGrid(t *testing.T) {
	g := gradient()
	// Make it asymmetric in both axes so every orientation is different.
	for y := 0; y < N; y++ {
		for x := 0; x < N; x++ {
			g[y*N+x] += float64(x*y%7) * 3
		}
	}
	fp := g.Fingerprint()
	for o := 1; o <= 8; o++ {
		want := g.Orient(o).Fingerprint()
		got := Transform(fp, o)
		for i := range want {
			if math.Abs(float64(want[i]-got[i])) > 1e-2*(1+math.Abs(float64(want[i]))) {
				t.Fatalf("orientation %d, coefficient %d: got %v want %v", o, i, got[i], want[i])
			}
		}
	}
}

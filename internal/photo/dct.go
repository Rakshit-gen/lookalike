package photo

import "math"

// K is how many of the lowest DCT frequencies are kept along each axis.
const K = 8

// Dim is the fingerprint length: the K by K lowest frequencies minus the
// constant (DC) term.
const Dim = K*K - 1

// cosTable[u][x] is cos((2x+1)uπ / 2N), the DCT-II basis.
var cosTable = func() (t [K][N]float64) {
	for u := 0; u < K; u++ {
		for x := 0; x < N; x++ {
			t[u][x] = math.Cos(float64(2*x+1) * float64(u) * math.Pi / (2 * N))
		}
	}
	return t
}()

// Fingerprint is the low frequency part of the grid's 2D DCT. Low
// frequencies describe the layout of light and dark in the photo, which
// survives resizing and recompression. Fine detail, where those changes
// show up, lives in the high frequencies and is dropped.
//
// The DC term (overall brightness) is left out, so a brightened copy still
// matches. Comparing fingerprints by cosine similarity also ignores their
// length, which makes a contrast change not matter either.
func (g *Grid) Fingerprint() []float32 {
	// Rows first: tmp[y][u] is the 1D DCT of row y, only for u < K.
	var tmp [N][K]float64
	for y := 0; y < N; y++ {
		for u := 0; u < K; u++ {
			var s float64
			for x := 0; x < N; x++ {
				s += g[y*N+x] * cosTable[u][x]
			}
			tmp[y][u] = s
		}
	}
	out := make([]float32, 0, Dim)
	for v := 0; v < K; v++ {
		for u := 0; u < K; u++ {
			if u == 0 && v == 0 {
				continue
			}
			var s float64
			for y := 0; y < N; y++ {
				s += tmp[y][u] * cosTable[v][y]
			}
			out = append(out, float32(s))
		}
	}
	return out
}

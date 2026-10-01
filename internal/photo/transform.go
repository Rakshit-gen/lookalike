package photo

// Transform returns the fingerprint the photo would have after Orient(o),
// computed from the fingerprint alone. It works because flipping an image
// only changes the sign of some DCT coefficients, and swapping its axes
// swaps the coefficient indices. That lets a search for rotated copies use
// fingerprints already in the index, without decoding anything again.
func Transform(fp []float32, o int) []float32 {
	// For each orientation: whether the axes swap, and whether the source
	// x and y run backwards. These mirror the cases in Grid.Orient.
	var swap, fx, fy bool
	switch o {
	case 2:
		fx = true
	case 3:
		fx, fy = true, true
	case 4:
		fy = true
	case 5:
		swap = true
	case 6:
		swap, fy = true, true
	case 7:
		swap, fx, fy = true, true, true
	case 8:
		swap, fx = true, true
	}
	out := make([]float32, len(fp))
	for v := 0; v < K; v++ {
		for u := 0; u < K; u++ {
			if u == 0 && v == 0 {
				continue
			}
			su, sv := u, v
			if swap {
				su, sv = v, u
			}
			val := fp[index(su, sv)]
			// A reversed source axis flips the sign of the odd frequencies
			// along it.
			if fx && su%2 == 1 {
				val = -val
			}
			if fy && sv%2 == 1 {
				val = -val
			}
			out[index(u, v)] = val
		}
	}
	return out
}

// index is where coefficient (u, v) sits in a fingerprint: row by row of
// v, skipping the DC term at (0, 0).
func index(u, v int) int { return v*K + u - 1 }

package photo

import "image"

// N is the side of the grayscale grid a photo is reduced to before the
// fingerprint is taken.
const N = 32

// Grid is an N by N grayscale thumbnail, values 0 to 255, row major.
type Grid [N * N]float64

// Reduce averages img into an N by N grayscale grid. Every source pixel
// lands in exactly one cell, so this is a box filter: it smooths away the
// detail and noise that recompression changes. JPEGs decode to YCbCr, and
// their luma plane is read directly, which skips a color conversion per
// pixel.
func Reduce(img image.Image) *Grid {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	var sum Grid
	var count [N * N]int
	if w == 0 || h == 0 {
		return &sum
	}
	add := func(x, y int, v float64) {
		c := (y*N/h)*N + x*N/w
		sum[c] += v
		count[c]++
	}
	if yc, ok := img.(*image.YCbCr); ok {
		for y := 0; y < h; y++ {
			row := yc.Y[(y)*yc.YStride:]
			for x := 0; x < w; x++ {
				add(x, y, float64(row[x]))
			}
		}
	} else {
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				r, g, bl, _ := img.At(b.Min.X+x, b.Min.Y+y).RGBA()
				// Rec. 601 luma, same weights JPEG uses, scaled from 16 bit.
				add(x, y, (0.299*float64(r)+0.587*float64(g)+0.114*float64(bl))/257)
			}
		}
	}
	for i := range sum {
		if count[i] > 0 {
			sum[i] /= float64(count[i])
		}
	}
	return &sum
}

// Orient turns a grid stored the way the camera wrote it into the way the
// photo is meant to be viewed, for EXIF orientations 1 to 8. Because the
// grid is square, rotating it gives the same result as rotating the full
// image before reducing it, at a tiny fraction of the cost.
func (g *Grid) Orient(o int) *Grid {
	var out Grid
	for y := 0; y < N; y++ {
		for x := 0; x < N; x++ {
			sx, sy := x, y
			switch o {
			case 2: // mirrored
				sx = N - 1 - x
			case 3: // upside down
				sx, sy = N-1-x, N-1-y
			case 4: // mirrored vertically
				sy = N - 1 - y
			case 5: // mirrored along the main diagonal
				sx, sy = y, x
			case 6: // stored rotated 90 degrees counter clockwise
				sx, sy = y, N-1-x
			case 7:
				sx, sy = N-1-y, N-1-x
			case 8: // stored rotated 90 degrees clockwise
				sx, sy = N-1-y, x
			}
			out[y*N+x] = g[sy*N+sx]
		}
	}
	return &out
}

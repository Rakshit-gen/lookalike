// Package variant makes edited copies of a photo, the kind that pile up in
// a real library, so lookalike can be tested on known duplicates.
package variant

import (
	"image"
	"image/color"
)

// Kind is one way a copy can differ from the original.
type Kind struct {
	Name    string
	Quality int // JPEG quality the copy is saved at
	Make    func(image.Image) image.Image
}

// Kinds are the edits eval applies. Each one is something that happens to
// photos in practice: a chat app shrinking and recompressing, a phone
// export, an auto-enhance, a crop, a turn.
var Kinds = []Kind{
	{"half-size", 75, Half},
	{"heavy-jpeg", 20, same},
	{"brighter", 85, func(m image.Image) image.Image { return Brighten(m, 30) }},
	{"crop-5pct", 85, func(m image.Image) image.Image { return Crop(m, 0.05) }},
	{"crop-15pct", 85, func(m image.Image) image.Image { return Crop(m, 0.15) }},
	{"quarter-turn", 85, Rotate90},
}

func same(m image.Image) image.Image { return m }

// Half scales an image to half its width and height, averaging each 2x2
// block.
func Half(src image.Image) image.Image {
	b := src.Bounds()
	w, h := b.Dx()/2, b.Dy()/2
	out := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			var r, g, bl uint32
			for dy := 0; dy < 2; dy++ {
				for dx := 0; dx < 2; dx++ {
					cr, cg, cb, _ := src.At(b.Min.X+2*x+dx, b.Min.Y+2*y+dy).RGBA()
					r, g, bl = r+cr, g+cg, bl+cb
				}
			}
			out.Set(x, y, color.RGBA64{uint16(r / 4), uint16(g / 4), uint16(bl / 4), 0xffff})
		}
	}
	return out
}

// Brighten adds delta (0 to 255 scale) to every channel, clipping at white.
func Brighten(src image.Image, delta int) image.Image {
	b := src.Bounds()
	out := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	clip := func(v uint32) uint8 { return uint8(min(int(v>>8)+delta, 255)) }
	for y := 0; y < b.Dy(); y++ {
		for x := 0; x < b.Dx(); x++ {
			r, g, bl, _ := src.At(b.Min.X+x, b.Min.Y+y).RGBA()
			out.Set(x, y, color.RGBA{clip(r), clip(g), clip(bl), 255})
		}
	}
	return out
}

// Crop cuts frac of the width and height off each side.
func Crop(src image.Image, frac float64) image.Image {
	b := src.Bounds()
	dx, dy := int(float64(b.Dx())*frac), int(float64(b.Dy())*frac)
	r := image.Rect(b.Min.X+dx, b.Min.Y+dy, b.Max.X-dx, b.Max.Y-dy)
	out := image.NewRGBA(image.Rect(0, 0, r.Dx(), r.Dy()))
	for y := 0; y < r.Dy(); y++ {
		for x := 0; x < r.Dx(); x++ {
			out.Set(x, y, src.At(r.Min.X+x, r.Min.Y+y))
		}
	}
	return out
}

// Rotate90 turns an image a quarter turn clockwise.
func Rotate90(src image.Image) image.Image {
	b := src.Bounds()
	out := image.NewRGBA(image.Rect(0, 0, b.Dy(), b.Dx()))
	for y := 0; y < b.Dy(); y++ {
		for x := 0; x < b.Dx(); x++ {
			out.Set(b.Dy()-1-y, x, src.At(b.Min.X+x, b.Min.Y+y))
		}
	}
	return out
}

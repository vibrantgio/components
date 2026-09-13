package composite_test

import (
	"image"
	stdcolor "image/color"
	"testing"

	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"

	"github.com/vibrantgio/components/composite"
	"github.com/vibrantgio/components/golden"
	vgcolor "github.com/vibrantgio/theme/color"
)

var (
	white  = stdcolor.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}
	accent = stdcolor.NRGBA{R: 0x00, G: 0x7a, B: 0xff, A: 0xff}
	scrim  = stdcolor.NRGBA{A: 0x33}
)

func at(img *image.RGBA, x, y int) stdcolor.NRGBA {
	o := img.PixOffset(x, y)
	return stdcolor.NRGBA{R: img.Pix[o], G: img.Pix[o+1], B: img.Pix[o+2], A: img.Pix[o+3]}
}

// page paints the frame half white, half the platform's accent.
func page(gtx layout.Context, size image.Point) {
	paint.FillShape(gtx.Ops, white, clip.Rect(image.Rect(0, 0, size.X/2, size.Y)).Op())
	paint.FillShape(gtx.Ops, accent, clip.Rect(image.Rect(size.X/2, 0, size.X, size.Y)).Op())
}

// TestFlattenOverTheDeclaredPlaneLandsTheFlattenedByte pins the composite
// against the blend the platform does, per pixel and on both fills a page is
// made of. The transition between them is part of it: a single coverage
// cannot hold two fills at once, which is the whole reason the pixels are
// read.
func TestFlattenOverTheDeclaredPlaneLandsTheFlattenedByte(t *testing.T) {
	size := image.Pt(64, 32)
	img := golden.Capture(t, size, func(gtx layout.Context) layout.Dimensions {
		page(gtx, size)
		composite.Flatten(gtx, image.Rectangle{Max: size}, scrim)
		return layout.Dimensions{Size: size}
	})
	for _, tc := range []struct {
		name string
		x    int
		over stdcolor.NRGBA
	}{
		{"the white half", 8, white},
		{"the last white pixel", size.X/2 - 1, white},
		{"the first accent pixel", size.X / 2, accent},
		{"the accent half", size.X - 8, accent},
	} {
		if got, want := at(img, tc.x, size.Y/2), vgcolor.Flatten(scrim, tc.over); got != want {
			t.Errorf("%s: got %v, want %v", tc.name, got, want)
		}
	}
}

// TestFlattenFallsBackToTheFittedCoverage pins the other half of the
// contract: an overlay that is not the frame's own plane — a specimen tile
// with a scrim of its own — cannot be placed in what was rendered, so it is
// painted at the coverage fitted to Gio's blend instead, and lands within
// that fit's recorded three 255ths rather than the twenty-seven a raw
// coverage would.
func TestFlattenFallsBackToTheFittedCoverage(t *testing.T) {
	size := image.Pt(64, 32)
	tile := image.Rect(0, 0, size.X/2, size.Y)
	img := golden.Capture(t, size, func(gtx layout.Context) layout.Dimensions {
		page(gtx, size)
		composite.Flatten(gtx, tile, scrim)
		return layout.Dimensions{Size: size}
	})
	got := at(img, 8, size.Y/2)
	want := vgcolor.Flatten(scrim, white)
	for c, pair := range [][2]uint8{{got.R, want.R}, {got.G, want.G}, {got.B, want.B}} {
		if d := int(pair[0]) - int(pair[1]); d < -3 || d > 3 {
			t.Errorf("channel %d: got %d, want %d within three 255ths", c, pair[0], pair[1])
		}
	}
	if got == want {
		t.Errorf("the fallback landed the flattened byte exactly; this tile is not the declared plane and cannot")
	}
}

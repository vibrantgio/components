package list_test

import (
	"image"
	"image/color"
	"testing"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"

	"github.com/vibrantgio/components/golden"
	"github.com/vibrantgio/components/internal/focus"
	"github.com/vibrantgio/components/list"
)

// The list's own geometry for the halo reading. The viewport holds two and a
// half rows, so the third row — the one the focused control stands in — is
// laid out half outside it, which is the case the reading is about.
const (
	haloListX, haloListY = 10, 10 // where the list stands in the frame
	haloListW, haloListH = 60, 50 // the viewport
	haloRowH             = 20
	haloFocusedRow       = 2 // its top sits at 40, its foot 10 past the viewport
	haloFrame            = 80
)

// TestAFocusedRowsHaloStaysInsideTheViewport is the pixel proof that a
// deferred drawing made inside a list is cut by the list.
//
// A focused control's band straddles its own box, so it has to outlive the
// clip a gioui.org/widget.Clickable puts around whatever it wraps. It used to
// outlive that clip through op.Defer, which restores the transform and RESETS
// the clip: the band then outlived the list as well and painted straight over
// the window past the viewport's edge — the defect this reading pins shut.
// The band now goes to the sink [focus.Around] publishes just outside that one
// clip, so it escapes the clip it must and keeps every clip it stands in.
//
// The row is the full width of the viewport and fills it, so the band
// straddles the viewport's leading edge, its trailing edge and its foot at
// once, and every one of them has to cut.
func TestAFocusedRowsHaloStaysInsideTheViewport(t *testing.T) {
	standsOn := color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}
	fill := color.NRGBA{R: 0x20, G: 0x20, B: 0x20, A: 0xff}
	band := color.NRGBA{R: 0x00, G: 0x60, B: 0xff, A: 0xff}

	state := list.NewState()
	// Each item is its own index, so the row builder can say which row it is
	// drawing: a list hands its builder the item, never the position.
	rows := make([]int, 8)
	for i := range rows {
		rows[i] = i
	}

	img := golden.Capture(t, image.Pt(haloFrame, haloFrame), func(gtx layout.Context) layout.Dimensions {
		paint.FillShape(gtx.Ops, standsOn, clip.Rect{Max: gtx.Constraints.Max}.Op())
		defer op.Offset(image.Pt(haloListX, haloListY)).Push(gtx.Ops).Pop()
		gtx.Constraints = layout.Exact(image.Pt(haloListW, haloListH))
		list.Layout(gtx, state, rows, func(gtx layout.Context, row int) layout.Dimensions {
			size := image.Pt(gtx.Constraints.Max.X, haloRowH)
			box := image.Rectangle{Max: size}
			if row == haloFocusedRow {
				// A control as every control in this library is drawn: the
				// band recorded inside the clip a Clickable puts around what
				// it wraps, and painted outside it.
				focus.Around(gtx, func(gtx layout.Context) layout.Dimensions {
					defer clip.Rect(box).Push(gtx.Ops).Pop()
					paint.FillShape(gtx.Ops, fill, clip.Rect(box).Op())
					focus.Halo(gtx, box, 0, band, band)
					return layout.Dimensions{Size: size}
				})
			}
			return layout.Dimensions{Size: size}
		})
		return layout.Dimensions{Size: gtx.Constraints.Max}
	})
	if img == nil {
		return // headless unavailable; Capture called t.Skip
	}

	out := int(focus.Outside)
	// The rows the focused control's band runs down, in the frame's own
	// coordinates: its own, clear of the corners.
	y := haloListY + haloFocusedRow*haloRowH + haloRowH/4
	for _, c := range []struct {
		what string
		x, y int
		want color.NRGBA
	}{
		{"the column past the viewport's leading edge", haloListX - out, y, standsOn},
		{"the outer column of the band inside the leading edge", haloListX, y, band},
		{"the inner column of that band", haloListX + out - 1, y, band},
		{"the control's own fill", haloListX + out, y, fill},
		{"the inner column of the band at the trailing edge", haloListW + haloListX - out, y, band},
		{"the outer column of that band", haloListW + haloListX - 1, y, band},
		{"the column past the viewport's trailing edge", haloListW + haloListX, y, standsOn},
		{"the last row inside the viewport, where the band's foot never arrives", haloListX + out, haloListY + haloListH - 1, fill},
		{"the band's own column one row past the viewport's foot", haloListX, haloListY + haloListH, standsOn},
		{"the row the band's foot would have landed on", haloListX + haloListW/2, haloListY + (haloFocusedRow+1)*haloRowH - 1, standsOn},
	} {
		if at := img.RGBAAt(c.x, c.y); !nearlyEqual(at, c.want) {
			t.Errorf("%s (x=%d, y=%d) = %v, want %v", c.what, c.x, c.y, at, c.want)
		}
	}
}

// nearlyEqual allows the rasterizer's rounding on an opaque pixel: the claim
// is which colour a column carries, not which byte a blend landed on.
func nearlyEqual(got color.RGBA, want color.NRGBA) bool {
	const slack = 3
	off := func(a, b uint8) bool {
		if a > b {
			return a-b > slack
		}
		return b-a > slack
	}
	return !off(got.R, want.R) && !off(got.G, want.G) && !off(got.B, want.B) && got.A == want.A
}

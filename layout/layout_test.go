package layout

import (
	"image"
	"image/color"
	"reflect"
	"testing"

	gio "gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"

	golden "github.com/vibrantgio/components/golden"
)

// fill returns a layout.Widget that draws c over its entire constraint maximum.
func fill(c color.NRGBA) gio.Widget {
	return func(gtx gio.Context) gio.Dimensions {
		paint.FillShape(gtx.Ops, c, clip.Rect{Max: gtx.Constraints.Max}.Op())
		return gio.Dimensions{Size: gtx.Constraints.Max}
	}
}

// fixed returns a layout.Widget that draws c in a w×h rect regardless of constraints.
func fixed(w, h int, c color.NRGBA) gio.Widget {
	return func(gtx gio.Context) gio.Dimensions {
		sz := image.Pt(w, h)
		paint.FillShape(gtx.Ops, c, clip.Rect{Max: sz}.Op())
		return gio.Dimensions{Size: sz}
	}
}

var (
	red  = color.NRGBA{R: 0xff, A: 0xff}
	blue = color.NRGBA{B: 0xff, A: 0xff}
	wht  = color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}
)

// TestPillClampBoundary verifies that Pill clamps rad=9999 to min(w,h)/2=10 on
// a 40×20 rect, producing an identical op stream to Pill called with rad=10.
func TestPillClampBoundary(t *testing.T) {
	rect := image.Rect(0, 0, 40, 20) // min(40,20)/2 = 10
	var ops1, ops2 op.Ops
	Pill(&ops1, rect, 9999) // sentinel — must clamp to 10
	Pill(&ops2, rect, 10)   // already at the clamp limit
	if !reflect.DeepEqual(ops1, ops2) {
		t.Fatal("Pill(rect, 9999) must produce the same op stream as Pill(rect, 10) on a 40×20 rect")
	}
}

// TestInset renders a red fill inside Inset(16) within a white 100×100 box.
// Expected: red occupies the inner 68×68 region; outer 16dp is white.
func TestInset(t *testing.T) {
	size := image.Pt(100, 100)
	w := func(gtx gio.Context) gio.Dimensions {
		fill(wht)(gtx)
		return Inset(16).Layout(gtx, fill(red))
	}
	golden.Render(t, "inset-uniform", size, w)
}

// TestInsetXY renders a red fill inside InsetXY(8, 24) within a white 100×100 box.
// Expected: 8dp left/right, 24dp top/bottom in white; centre red.
func TestInsetXY(t *testing.T) {
	size := image.Pt(100, 100)
	w := func(gtx gio.Context) gio.Dimensions {
		fill(wht)(gtx)
		return InsetXY(8, 24).Layout(gtx, fill(red))
	}
	golden.Render(t, "inset-xy", size, w)
}

// TestHSpacer renders two colored boxes separated by an HSpacer(20) inside a Row.
// Expected: 20px blue | 20px white gap | 20px red, all 20px tall.
func TestHSpacer(t *testing.T) {
	size := image.Pt(60, 20)
	w := func(gtx gio.Context) gio.Dimensions {
		return Row(gtx, fixed(20, 20, blue), HSpacer(20), fixed(20, 20, red))
	}
	golden.Render(t, "hspacer", size, w)
}

// TestVSpacer renders two colored boxes separated by a VSpacer(20) inside a Col.
// Expected: 20px blue | 20px white gap | 20px red, all 20px wide.
func TestVSpacer(t *testing.T) {
	size := image.Pt(20, 60)
	w := func(gtx gio.Context) gio.Dimensions {
		return Col(gtx, fixed(20, 20, blue), VSpacer(20), fixed(20, 20, red))
	}
	golden.Render(t, "vspacer", size, w)
}

// TestRow renders two fixed-size boxes side by side.
// Expected: 20px blue on left, 20px red on right; total 40×20.
func TestRow(t *testing.T) {
	size := image.Pt(40, 20)
	w := func(gtx gio.Context) gio.Dimensions {
		return Row(gtx, fixed(20, 20, blue), fixed(20, 20, red))
	}
	golden.Render(t, "row", size, w)
}

// TestCol renders two fixed-size boxes stacked.
// Expected: 20px blue on top, 20px red below; total 20×40.
func TestCol(t *testing.T) {
	size := image.Pt(20, 40)
	w := func(gtx gio.Context) gio.Dimensions {
		return Col(gtx, fixed(20, 20, blue), fixed(20, 20, red))
	}
	golden.Render(t, "col", size, w)
}

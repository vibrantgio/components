package list_test

import (
	"image"
	"image/color"
	"testing"

	gioinput "gioui.org/io/input"
	"gioui.org/io/key"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"

	"github.com/vibrantgio/components/golden"
	"github.com/vibrantgio/components/internal/focus"
	"github.com/vibrantgio/components/list"
	vgcolor "github.com/vibrantgio/theme/color"
	"github.com/vibrantgio/theme/tokens"
)

// The list's own geometry for the ring reading. The viewport holds two and a
// half rows, so the third row — the one the focused control stands in — is
// laid out half outside it, which is the case the reading is about.
const (
	ringListX, ringListY = 10, 10 // where the list stands in the frame
	ringListW, ringListH = 60, 50 // the viewport
	ringRowH             = 20
	ringFocusedRow       = 2 // its top sits at 40, its foot 10 past the viewport
	ringFrame            = 80
)

// TestAFocusedRowsRingStaysInsideTheViewport is the pixel proof that a
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
func TestAFocusedRowsRingStaysInsideTheViewport(t *testing.T) {
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

	img := golden.Capture(t, image.Pt(ringFrame, ringFrame), func(gtx layout.Context) layout.Dimensions {
		paint.FillShape(gtx.Ops, standsOn, clip.Rect{Max: gtx.Constraints.Max}.Op())
		defer op.Offset(image.Pt(ringListX, ringListY)).Push(gtx.Ops).Pop()
		gtx.Constraints = layout.Exact(image.Pt(ringListW, ringListH))
		list.Layout(gtx, state, rows, func(gtx layout.Context, row int) layout.Dimensions {
			size := image.Pt(gtx.Constraints.Max.X, ringRowH)
			box := image.Rectangle{Max: size}
			if row == ringFocusedRow {
				// A control as every control in this library is drawn: the
				// band recorded inside the clip a Clickable puts around what
				// it wraps, and painted outside it.
				focus.Around(gtx, func(gtx layout.Context) layout.Dimensions {
					defer clip.Rect(box).Push(gtx.Ops).Pop()
					paint.FillShape(gtx.Ops, fill, clip.Rect(box).Op())
					focus.Ring(gtx, box, 0, band, band)
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
	y := ringListY + ringFocusedRow*ringRowH + ringRowH/4
	for _, c := range []struct {
		what string
		x, y int
		want color.NRGBA
	}{
		{"the column past the viewport's leading edge", ringListX - out, y, standsOn},
		{"the outer column of the band inside the leading edge", ringListX, y, band},
		{"the inner column of that band", ringListX + out - 1, y, band},
		{"the control's own fill", ringListX + out, y, fill},
		{"the inner column of the band at the trailing edge", ringListW + ringListX - out, y, band},
		{"the outer column of that band", ringListW + ringListX - 1, y, band},
		{"the column past the viewport's trailing edge", ringListW + ringListX, y, standsOn},
		{"the last row inside the viewport, where the band's foot never arrives", ringListX + out, ringListY + ringListH - 1, fill},
		{"the band's own column one row past the viewport's foot", ringListX, ringListY + ringListH, standsOn},
		{"the row the band's foot would have landed on", ringListX + ringListW/2, ringListY + (ringFocusedRow+1)*ringRowH - 1, standsOn},
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

// The geometry of the focusable-list reading below: a list standing in the
// frame with room on every side for the band to run past its box.
const (
	focusListX, focusListY = 12, 12
	focusListW, focusListH = 56, 48
	focusRowH              = 16
	focusFrame             = 80
)

// TestAFocusedListWearsTheRingOnItsOwnBox is the pixel proof of what a list
// at the front of a dialog shows: [list.FocusRing] draws the one ring around
// the LIST's box — not a row's — while the list holds the keyboard, and draws
// nothing there while it does not.
func TestAFocusedListWearsTheRingOnItsOwnBox(t *testing.T) {
	standsOn := color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}
	p := tokens.PlatformColors{
		KeyboardFocusIndicator:    color.NRGBA{R: 0x00, G: 0x60, B: 0xff, A: 0xff},
		SelectedContentBackground: color.NRGBA{R: 0x00, G: 0x64, B: 0xe1, A: 0xff},
	}
	band := vgcolor.Flatten(p.KeyboardFocusIndicator, standsOn)

	rows := make([]int, 6)
	for i := range rows {
		rows[i] = i
	}
	// A frame of the list, focused or not, with the keyboard handed to the
	// list through a real router so gtx.Focused answers as the window would.
	frame := func(t *testing.T, takeKeyboard bool) *image.RGBA {
		t.Helper()
		state := list.NewState()
		r := new(gioinput.Router)
		var focused bool
		w := func(gtx layout.Context) layout.Dimensions {
			paint.FillShape(gtx.Ops, standsOn, clip.Rect{Max: gtx.Constraints.Max}.Op())
			defer op.Offset(image.Pt(focusListX, focusListY)).Push(gtx.Ops).Pop()
			gtx.Constraints = layout.Exact(image.Pt(focusListW, focusListH))
			if takeKeyboard && !focused {
				gtx.Execute(key.FocusCmd{Tag: state.Focus()})
				focused = true
			}
			return list.FocusRing(gtx, state, p, standsOn, func(gtx layout.Context) layout.Dimensions {
				return list.LayoutSelectable(gtx, state, rows,
					func(gtx layout.Context, _ int, _ bool) layout.Dimensions {
						return layout.Dimensions{Size: image.Pt(gtx.Constraints.Max.X, focusRowH)}
					})
			})
		}
		for range 2 {
			var ops op.Ops
			w(layout.Context{
				Metric:      unit.Metric{PxPerDp: 1, PxPerSp: 1},
				Constraints: layout.Exact(image.Pt(focusFrame, focusFrame)),
				Ops:         &ops,
				Source:      r.Source(),
			})
			r.Frame(&ops)
		}
		return golden.Capture(t, image.Pt(focusFrame, focusFrame), func(gtx layout.Context) layout.Dimensions {
			gtx.Source = r.Source()
			return w(gtx)
		})
	}

	ring := int(list.RingWidth)
	y := focusListY + focusListH/2
	img := frame(t, true)
	if img == nil {
		return // headless unavailable; Capture called t.Skip
	}
	for _, c := range []struct {
		what string
		x, y int
		want color.NRGBA
	}{
		{"the column past the ring's outer edge", focusListX - ring - 1, y, standsOn},
		{"the ring's outer column", focusListX - ring, y, band},
		{"the ring's inner column", focusListX - 1, y, band},
		{"the list's own first column", focusListX, y, standsOn},
		{"the ring's inner row at the list's foot", focusListX + focusListW/2, focusListY + focusListH, band},
		{"the ring's outer row at the list's foot", focusListX + focusListW/2, focusListY + focusListH + ring - 1, band},
		{"the row past the ring's foot", focusListX + focusListW/2, focusListY + focusListH + ring, standsOn},
	} {
		if at := img.RGBAAt(c.x, c.y); !nearlyEqual(at, c.want) {
			t.Errorf("focused: %s (x=%d, y=%d) = %v, want %v", c.what, c.x, c.y, at, c.want)
		}
	}

	rest := frame(t, false)
	if rest == nil {
		return
	}
	for _, x := range []int{focusListX - ring, focusListX - 1, focusListX + focusListW, focusListX + focusListW + ring - 1} {
		if at := rest.RGBAAt(x, y); !nearlyEqual(at, standsOn) {
			t.Errorf("at rest: the column at x=%d carries %v: a list nothing focuses wears no band", x, at)
		}
	}
}

// TestTheRingIsDrawnOutsideTheListsBox is the pixel proof of where
// [list.FocusRing] draws the ring: around the list's outer edge, outside its
// content. The ring's inner edge is the box's outer edge, it is
// [list.RingWidth] wide past it, and no ring pixel lies over a row.
//
// Every row of the list paints the platform's selection fill, so the whole
// box is a fill the ring would change if it crossed it: every pixel of the
// box must read that fill exactly, and the ring's colour is the platform's
// keyboard focus indicator flattened onto the surface the list stands on on
// all four sides.
func TestTheRingIsDrawnOutsideTheListsBox(t *testing.T) {
	for _, sc := range []struct {
		name string
		p    tokens.PlatformColors
	}{
		{"light", tokens.PlatformLight},
		{"dark", tokens.PlatformDark},
	} {
		t.Run(sc.name, func(t *testing.T) {
			p := sc.p
			standsOn := p.WindowBackground
			filled := p.SelectedContentBackground
			band := vgcolor.Flatten(p.KeyboardFocusIndicator, standsOn)

			img := focusedFilledList(t, p)
			if img == nil {
				return // headless unavailable; Capture called t.Skip
			}
			box := image.Rect(focusListX, focusListY, focusListX+focusListW, focusListY+focusListH)
			for y := box.Min.Y; y < box.Max.Y; y++ {
				for x := box.Min.X; x < box.Max.X; x++ {
					if at := img.RGBAAt(x, y); at != (color.RGBA(filled)) {
						t.Fatalf("the list's box at (x=%d, y=%d) = %v, want the row's fill %v: a ring pixel lies over a row", x, y, at, filled)
					}
				}
			}
			ring := int(list.RingWidth)
			midX := (box.Min.X + box.Max.X) / 2
			midY := (box.Min.Y + box.Max.Y) / 2
			for _, side := range []struct {
				what  string
				inner image.Point // the ring's pixel touching the box
				step  image.Point // one pixel further out
			}{
				{"leading", image.Pt(box.Min.X-1, midY), image.Pt(-1, 0)},
				{"trailing", image.Pt(box.Max.X, midY), image.Pt(1, 0)},
				{"head", image.Pt(midX, box.Min.Y-1), image.Pt(0, -1)},
				{"foot", image.Pt(midX, box.Max.Y), image.Pt(0, 1)},
			} {
				for i := range ring {
					pt := side.inner.Add(side.step.Mul(i))
					if at := img.RGBAAt(pt.X, pt.Y); !nearlyEqual(at, band) {
						t.Errorf("%s side: the ring's pixel %d out from the box (x=%d, y=%d) = %v, want %v: the ring is not %d px wide past the box's edge",
							side.what, i+1, pt.X, pt.Y, at, band, ring)
					}
				}
				past := side.inner.Add(side.step.Mul(ring))
				if at := img.RGBAAt(past.X, past.Y); !nearlyEqual(at, standsOn) {
					t.Errorf("%s side: the pixel past the ring (x=%d, y=%d) = %v, want the surface %v", side.what, past.X, past.Y, at, standsOn)
				}
			}
		})
	}
}

// focusedFilledList renders a list holding the keyboard whose every row
// paints the platform's selection fill, standing on the platform's window
// background. Its first row is the selection.
func focusedFilledList(t *testing.T, p tokens.PlatformColors) *image.RGBA {
	t.Helper()
	standsOn := p.WindowBackground
	rows := make([]int, 6)
	for i := range rows {
		rows[i] = i
	}
	state := list.NewState()
	state.Select(0)
	r := new(gioinput.Router)
	var focused bool
	w := func(gtx layout.Context) layout.Dimensions {
		paint.FillShape(gtx.Ops, standsOn, clip.Rect{Max: gtx.Constraints.Max}.Op())
		defer op.Offset(image.Pt(focusListX, focusListY)).Push(gtx.Ops).Pop()
		gtx.Constraints = layout.Exact(image.Pt(focusListW, focusListH))
		if !focused {
			gtx.Execute(key.FocusCmd{Tag: state.Focus()})
			focused = true
		}
		return list.FocusRing(gtx, state, p, standsOn, func(gtx layout.Context) layout.Dimensions {
			return list.LayoutSelectable(gtx, state, rows,
				func(gtx layout.Context, _ int, _ bool) layout.Dimensions {
					size := image.Pt(gtx.Constraints.Max.X, focusRowH)
					paint.FillShape(gtx.Ops, p.SelectedContentBackground, clip.Rect{Max: size}.Op())
					return layout.Dimensions{Size: size}
				})
		})
	}
	for range 2 {
		var ops op.Ops
		w(layout.Context{
			Metric:      unit.Metric{PxPerDp: 1, PxPerSp: 1},
			Constraints: layout.Exact(image.Pt(focusFrame, focusFrame)),
			Ops:         &ops,
			Source:      r.Source(),
		})
		r.Frame(&ops)
	}
	return golden.Capture(t, image.Pt(focusFrame, focusFrame), func(gtx layout.Context) layout.Dimensions {
		gtx.Source = r.Source()
		return w(gtx)
	})
}

package focus_test

import (
	"image"
	"image/color"
	"testing"

	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"

	"github.com/vibrantgio/components/golden"
	"github.com/vibrantgio/components/internal/focus"
	"github.com/vibrantgio/theme/tokens"
)

// TestHaloStraddlesTheBoxAndOutlivesTheClip is the pixel proof of the two
// claims the halo is built on.
//
// The first is the measurement: the band straddles the control's own box,
// [focus.Outside] of it past that box and the rest over it, so a control
// keeps the box it reports and gains a band on both sides of that boundary.
//
// The second is the mechanism. A control reports the same box focused as at
// rest — the footprint is the pointer target and taking the keyboard may not
// grow it — so the half past the box would be cut away by the clip
// gioui.org/widget's Clickable puts around whatever it wraps. [focus.Halo]
// hands the band to the sink [focus.Around] publishes outside that clip, and
// the band lands whole. The test draws the halo inside exactly such a clip,
// under exactly such a sink, and reads the columns beyond it.
func TestHaloStraddlesTheBoxAndOutlivesTheClip(t *testing.T) {
	const frame, origin, side = 40, 10, 20
	out := int(focus.Outside)
	box := image.Rect(origin, origin, origin+side, origin+side)

	standsOn := color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}
	fill := color.NRGBA{R: 0x00, G: 0x00, B: 0x00, A: 0xff}
	past := focus.Ring(tokens.PlatformLight, standsOn)
	over := focus.Ring(tokens.PlatformLight, fill)

	img := golden.Capture(t, image.Pt(frame, frame), func(gtx layout.Context) layout.Dimensions {
		paint.FillShape(gtx.Ops, standsOn, clip.Rect{Max: gtx.Constraints.Max}.Op())
		return focus.Around(gtx, func(gtx layout.Context) layout.Dimensions {
			// The control's own box, and the clip a Clickable would put
			// around it: nothing drawn inside may reach a pixel beyond it.
			area := clip.Rect(box).Push(gtx.Ops)
			paint.FillShape(gtx.Ops, fill, clip.Rect(box).Op())
			focus.Halo(gtx, box, 0, past, over)
			area.Pop()
			return layout.Dimensions{Size: gtx.Constraints.Max}
		})
	})
	if img == nil {
		return // headless unavailable; Capture called t.Skip
	}

	y := origin + side/2 // a row clear of both corners
	for _, c := range []struct {
		what string
		x    int
		want color.NRGBA
	}{
		{"one column clear of the halo", origin - out - 1, standsOn},
		{"the outer column of the half past the box", origin - out, past},
		{"the inner column of the half past the box", origin - 1, past},
		{"the outer column of the half over the box", origin, over},
		{"the inner column of the half over the box", origin + out - 1, over},
		{"the control's own fill inside the halo", origin + out, fill},
	} {
		if at := img.RGBAAt(c.x, y); !nearlyEqual(at, c.want) {
			t.Errorf("%s (x=%d) = %v, want %v", c.what, c.x, at, c.want)
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

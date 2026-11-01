package inventory

import (
	"image"
	"testing"
	"time"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"

	"github.com/vibrantgio/components/golden"
	"github.com/vibrantgio/components/scrollbar"
	"github.com/vibrantgio/theme/tokens"
)

// bodyHeight lays body out once on a resting frame at width px, one pixel to
// the dp, on the section surface of c, and returns the height it took.
func bodyHeight(c tokens.PlatformColors, body layout.Widget, width int) int {
	return bodySize(c, body, width).Y
}

// bodySize lays body out once on a resting frame at width px, one pixel to
// the dp, on the section surface of c, and returns the size it took.
func bodySize(c tokens.PlatformColors, body layout.Widget, width int) image.Point {
	var ops op.Ops
	gtx := layout.Context{
		Ops:         &ops,
		Metric:      unit.Metric{PxPerDp: 1, PxPerSp: 1},
		Constraints: layout.Constraints{Max: image.Pt(width, 1<<14)},
		Now:         time.Unix(1700000000, 0),
	}
	return golden.On(SectionSurface(c), body)(gtx).Size
}

// TestTheFenceAndTheWideTableStandAtAlways lays the reading sample out at a
// width its table does not fit, with the window's bar fading and with the
// window's bar at Always.
//
// The fence overflows at this width, and so does the table. A fading bar
// takes no lane, so the body is as tall as with no window bar at all. A
// standing bar takes a lane of its own under what scrolls: the body grows by
// two lanes, one under its fence and one under its table. The standing frames
// also draw pixels the fading frames do not, which is the standing thumb.
func TestTheFenceAndTheWideTableStandAtAlways(t *testing.T) {
	const width = 200
	c := tokens.PlatformLight
	windowBar := scrollbar.FromTokens(c, c.ControlBackground)
	lane := int(windowBar.Always().Width())

	unset := testInventory(t)
	fading := testInventory(t)
	fading.SetScrollbar(windowBar)
	always := testInventory(t)
	always.SetScrollbar(windowBar.Always())

	for _, tc := range []struct {
		name  string
		body  func(*Inventory) layout.Widget
		lanes int
	}{
		{"the reading sample's fence and table", func(inv *Inventory) layout.Widget { return inv.readingBody(c) }, 2},
	} {
		base := bodyHeight(c, tc.body(unset), width)
		if got := bodyHeight(c, tc.body(fading), width); got != base {
			t.Errorf("%s with the window's bar fading is %d px tall; want %d, as with no window bar", tc.name, got, base)
		}
		if got, want := bodyHeight(c, tc.body(always), width), base+tc.lanes*lane; got != want {
			t.Errorf("%s with the window's bar at Always is %d px tall; want %d, %d lane(s) of %d px under what scrolls", tc.name, got, want, tc.lanes, lane)
		}
		size := image.Pt(width, base+tc.lanes*lane)
		if n := golden.PixelDiff(
			golden.Capture(t, size, resting(c, tc.body(fading))),
			golden.Capture(t, size, resting(c, tc.body(always))),
		); n == 0 {
			t.Errorf("%s at Always drew the same pixels as with the window's bar fading; want a standing bar", tc.name)
		}
	}
}

// resting holds body on a frame nothing has scrolled and nothing points at,
// so a fading bar shows no thumb. The text is told the section surface of c,
// the fill a section body stands on.
func resting(c tokens.PlatformColors, body layout.Widget) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		gtx.Now = time.Unix(1700000000, 0)
		return golden.On(SectionSurface(c), body)(gtx)
	}
}

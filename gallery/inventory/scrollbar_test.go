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
// the dp, and returns the height it took.
func bodyHeight(body layout.Widget, width int) int {
	var ops op.Ops
	gtx := layout.Context{
		Ops:         &ops,
		Metric:      unit.Metric{PxPerDp: 1, PxPerSp: 1},
		Constraints: layout.Constraints{Max: image.Pt(width, 1<<14)},
		Now:         time.Unix(1700000000, 0),
	}
	return body(gtx).Size.Y
}

// TestTheFenceAndTheWideTableStandAtAlways lays the two markdown sections
// out at a width the reading sample's table does not fit, with the window's
// bar fading and with the window's bar at Always.
//
// The code specimen's fence overflows at every width, and so does the
// reading sample's fence. At this width the reading sample's table overflows
// too. A fading bar takes no lane, so the sections are as tall as with no
// window bar at all. A standing bar takes a lane of its own under what
// scrolls: the code specimen grows by one lane, and the reading sample grows
// by two, one under its fence and one under its table. The standing frames
// also draw pixels the fading frames do not, which is the standing thumb.
func TestTheFenceAndTheWideTableStandAtAlways(t *testing.T) {
	const width = 200
	c := tokens.PlatformLight
	windowBar := scrollbar.FromTokens(c, c.ControlBackground)
	lane := int(windowBar.Width())

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
		{"the code specimen's fence", func(inv *Inventory) layout.Widget { return inv.codeBody(c) }, 1},
		{"the reading sample's fence and table", func(inv *Inventory) layout.Widget { return inv.readingBody(c) }, 2},
	} {
		base := bodyHeight(tc.body(unset), width)
		if got := bodyHeight(tc.body(fading), width); got != base {
			t.Errorf("%s with the window's bar fading is %d px tall; want %d, as with no window bar", tc.name, got, base)
		}
		if got, want := bodyHeight(tc.body(always), width), base+tc.lanes*lane; got != want {
			t.Errorf("%s with the window's bar at Always is %d px tall; want %d, %d lane(s) of %d px under what scrolls", tc.name, got, want, tc.lanes, lane)
		}
		size := image.Pt(width, base+tc.lanes*lane)
		if n := golden.PixelDiff(
			golden.Capture(t, size, resting(tc.body(fading))),
			golden.Capture(t, size, resting(tc.body(always))),
		); n == 0 {
			t.Errorf("%s at Always drew the same pixels as with the window's bar fading; want a standing bar", tc.name)
		}
	}
}

// resting holds body on a frame nothing has scrolled and nothing points at,
// so a fading bar shows no thumb.
func resting(body layout.Widget) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		gtx.Now = time.Unix(1700000000, 0)
		return body(gtx)
	}
}

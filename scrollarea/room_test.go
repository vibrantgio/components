package scrollarea_test

import (
	"image"
	"image/color"
	"testing"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"

	"github.com/vibrantgio/components/attachment"
	"github.com/vibrantgio/components/golden"
	"github.com/vibrantgio/components/scrollarea"
	"github.com/vibrantgio/components/tooltip"
	"github.com/vibrantgio/theme/tokens"
)

// scrolledTipWindow lays a scroll area across a 300 by 200 window that sets
// its available room, holding a 1000 px wide row with an anchored tooltip
// asked for below the 40 px cell at x 600, and scrolls the area 500 px so the
// cell is in view at x 100. The tooltip is shown or not.
func scrolledTipWindow(t *testing.T, shown bool) *image.RGBA {
	frame := image.Pt(300, 200)
	shaper := tokens.DefaultTypography.DeterministicShaper()
	state := scrollarea.NewState()
	content := func(gtx layout.Context) layout.Dimensions {
		cell := image.Rect(600, 0, 640, 40)
		tooltip.Render(shaper, tooltip.Props{
			Text:          "Where this goes",
			Placement:     tooltip.Bottom,
			AvailableRoom: attachment.AvailableRoom,
			Anchor:        func(layout.Context) image.Rectangle { return cell },
		}, shown, tokens.PlatformLight, tokens.Spacing, tokens.RadiusScale{}, tokens.DefaultTypography.SmallSystemFont)(gtx)
		return layout.Dimensions{Size: image.Pt(1000, 40)}
	}
	window := func(gtx layout.Context) layout.Dimensions {
		paint.FillShape(gtx.Ops, color.NRGBA{R: 245, G: 245, B: 245, A: 255}, clip.Rect{Max: frame}.Op())
		gtx = attachment.WithAvailableRoom(gtx, image.Rectangle{Max: frame})
		gtx.Constraints = layout.Constraints{Max: frame}
		scrollarea.Style{}.Layout(gtx, state, content)
		return layout.Dimensions{Size: frame}
	}
	// A first frame measures the content, so the offset can be set.
	window(layout.Context{Ops: new(op.Ops), Constraints: layout.Exact(frame)})
	state.SetOffset(500)
	return golden.Capture(t, frame, window)
}

// TestAScrolledTooltipIsPlacedAtItsCellInTheWindowsRoom holds the scroll area
// to the attachment: a tooltip in scrolled content is placed below its cell
// where the scroll put it, inside the window's room, and not against edges
// left unmoved by the scroll.
func TestAScrolledTooltipIsPlacedAtItsCellInTheWindowsRoom(t *testing.T) {
	shown, hidden := scrolledTipWindow(t, true), scrolledTipWindow(t, false)
	left, right := -1, -1
	b := shown.Bounds()
	for x := b.Min.X; x < b.Max.X; x++ {
		for y := b.Min.Y; y < b.Max.Y; y++ {
			if shown.RGBAAt(x, y) != hidden.RGBAAt(x, y) {
				if left < 0 {
					left = x
				}
				right = x
				break
			}
		}
	}
	if left < 0 {
		t.Fatal("the tooltip painted nothing inside the window")
	}
	// The cell is at 100..140 in the window; a surface centred on it and
	// kept inside the window spans its centre at 120.
	if left > 120 || right < 120 {
		t.Errorf("the tooltip spans columns %d..%d; want it across its cell's centre at 120", left, right)
	}
}

package list_test

import (
	"image"
	"image/color"
	"testing"

	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"

	"github.com/vibrantgio/components/golden"
	"github.com/vibrantgio/components/list"
	"github.com/vibrantgio/components/tooltip"
	"github.com/vibrantgio/theme/tokens"
)

// rowTipFrame, rowTipH, rowTipFirst and rowTipScroll are the window, the
// rows' height, the first row in view and the pixels the list is scrolled
// past that row's top: row 6 is at -15..25, row 7 at 25..65, and row 11 at
// 185..225, crossing the window's bottom edge.
var rowTipFrame = image.Pt(300, 200)

const (
	rowTipH      = 40
	rowTipFirst  = 6
	rowTipScroll = 15
)

// rowTipWindow lays a scrolled list out in the window the way a window's
// root lays out a layout that places its children after measuring them: under
// a tooltip.Surfaces, whose surfaces it then places in the window's room.
// Row tipRow holds an anchored tooltip asked for below the row, shown or
// not.
func rowTipWindow(t *testing.T, tipRow int, shown bool) *image.RGBA {
	shaper := tokens.DefaultTypography.DeterministicShaper()
	return golden.Capture(t, rowTipFrame, func(gtx layout.Context) layout.Dimensions {
		paint.FillShape(gtx.Ops, color.NRGBA{R: 245, G: 245, B: 245, A: 255}, clip.Rect{Max: rowTipFrame}.Op())
		window := gtx
		gtx, s := tooltip.WithSurfaces(gtx)
		state := list.NewStateAt(rowTipFirst)
		state.ScrollPixels(rowTipScroll)
		list.Layout(gtx, state, makeItems(20), func(gtx layout.Context, item int) layout.Dimensions {
			size := image.Pt(gtx.Constraints.Max.X, rowTipH)
			if item == tipRow {
				gtx.Constraints = layout.Exact(size)
				tooltip.Render(shaper, tooltip.Props{
					Text:      "Where this goes",
					Placement: tooltip.Bottom,
					Anchor:    func(layout.Context) image.Rectangle { return image.Rectangle{Max: size} },
				}, shown, tokens.PlatformLight, tokens.Spacing, tokens.RadiusScale{}, tokens.DefaultTypography.LabelSmall)(gtx)
			}
			return layout.Dimensions{Size: size}
		})
		tooltip.PlaceSurfaces(window, s, image.Rectangle{Max: rowTipFrame})
		return layout.Dimensions{Size: rowTipFrame}
	})
}

// paintedRows returns the first and last pixel rows where shown differs from
// hidden, -1 where none does.
func paintedRows(shown, hidden *image.RGBA) (top, bottom int) {
	top, bottom = -1, -1
	b := shown.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			if shown.RGBAAt(x, y) != hidden.RGBAAt(x, y) {
				if top < 0 {
					top = y
				}
				bottom = y
				break
			}
		}
	}
	return top, bottom
}

// TestAScrolledRowsTooltipIsPlacedInTheWindowsRoom holds the list's rows to
// the attachment: a row's tooltip is placed against the row where the scroll
// placed it, inside the window's room, below a row with room under it and
// above the row at the window's bottom edge.
func TestAScrolledRowsTooltipIsPlacedInTheWindowsRoom(t *testing.T) {
	// Row 7 is at 25..65, with room below it.
	foot := 2*rowTipH - rowTipScroll
	top, bottom := paintedRows(rowTipWindow(t, 7, true), rowTipWindow(t, 7, false))
	if top < 0 {
		t.Fatal("row 7's tooltip painted nothing")
	}
	if top < foot || bottom >= rowTipFrame.Y {
		t.Errorf("row 7's tooltip spans rows %d..%d; want it below the row's foot at %d and inside the window", top, bottom, foot)
	}
	// Row 11 is at 185..225, crossing the window's bottom edge: the tooltip
	// takes the side with room, above it.
	head := 5*rowTipH - rowTipScroll
	top, bottom = paintedRows(rowTipWindow(t, 11, true), rowTipWindow(t, 11, false))
	if top < 0 {
		t.Fatal("row 11's tooltip painted nothing")
	}
	if bottom >= head {
		t.Errorf("row 11's tooltip spans rows %d..%d; want it above the row's top at %d, inside the window", top, bottom, head)
	}
}

package list

import (
	"image"

	"gioui.org/io/event"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"

	"github.com/vibrantgio/components/attachment"
	"github.com/vibrantgio/components/scrollbar"
)

// Anchor defines how a scrollbar is attached to the list content.
type Anchor int

const (
	// Occupy reserves a lane of bar.Width() along the list's trailing
	// edge, narrowing the rows, while the rows overflow. Content is never
	// occluded. When the rows fit, the bar draws nothing and the rows take
	// the full width.
	Occupy Anchor = iota
	// Overlay floats the bar over the content's trailing edge. Rows keep
	// their full width, at the cost of the bar occluding whatever content
	// sits beneath it.
	Overlay
)

// AnchorFor returns the anchor a region's rows are laid out with under bar:
// [Occupy] when the bar stands, so the bar takes a lane of its own and the
// rows stop short of it, and [Overlay] when it fades, so the bar lies over
// the rows' trailing edge and takes no lane. A style with no fade delay is a
// bar that stands, which is what a style is once [scrollbar.Style.Always]
// has been applied to it for the platform's "Show scroll bars" setting
// Always.
func AnchorFor(bar scrollbar.Style) Anchor {
	if bar.FadeDelay == 0 {
		return Occupy
	}
	return Overlay
}

// LayoutScrollbar lays out items exactly like Layout and additionally draws
// bar along the list's trailing (east) edge, wired to the list's scroll
// position. Dragging the bar scrolls the list.
//
// anchor selects whether the bar reserves a gutter (Occupy) or floats over
// the content (Overlay). The list is vertical-only, so the bar is always
// vertical and anchored east.
//
// A pending [State.Reveal] is consumed here as it is in every other entry
// point, so a caller whose column scrolls but carries no selection of its
// own — a rail whose keyboard walks something other than the list's rows —
// asks for a row by index rather than measuring the column's pixels itself.
func LayoutScrollbar[T any](
	gtx layout.Context,
	state *State,
	bar scrollbar.Style,
	anchor Anchor,
	items []T,
	rowFn func(gtx layout.Context, item T) layout.Dimensions,
) layout.Dimensions {
	state.record(gtx)
	state.scrollIntoView(len(items))
	return state.scrolled(gtx, bar, anchor, len(items), func(gtx layout.Context, i int) layout.Dimensions {
		return rowFn(gtx, items[i])
	})
}

// LayoutSelectableScrollbar lays out items exactly like [LayoutSelectable] —
// one focus tag for the whole list, keyboard traversal over every row, the
// selection handed to rowFn — and additionally draws bar along the list's
// trailing edge exactly as [LayoutScrollbar] does.
//
// It exists because the two are not alternatives. A list long enough to need
// keyboard traversal is a list long enough to scroll, and a column that can
// scroll and does not say so reads as truncated; a caller that had to choose
// between the entry points would be choosing which half of that to lose.
//
// anchor selects whether the bar reserves a gutter (Occupy) or floats over the
// content (Overlay); the keyboard's own scrolling — every traversal move
// Reveals its new selection — moves the bar with it.
func LayoutSelectableScrollbar[T any](
	gtx layout.Context,
	state *State,
	bar scrollbar.Style,
	anchor Anchor,
	items []T,
	rowFn func(gtx layout.Context, item T, selected bool) layout.Dimensions,
) layout.Dimensions {
	n := len(items)
	state.record(gtx)
	state.dropStaleSelection(n)
	state.processKeys(gtx, n)
	state.scrollIntoView(n)

	// The focus target covers the whole viewport, the bar's gutter included:
	// the gutter is part of the list to a reader, and the bar's own pointer
	// areas are registered above this one either way.
	area := clip.Rect{Max: gtx.Constraints.Max}.Push(gtx.Ops)
	event.Op(gtx.Ops, &state.focus)
	area.Pop()

	sel := state.Selected()
	return state.scrolled(gtx, bar, anchor, n, func(gtx layout.Context, i int) layout.Dimensions {
		return rowFn(gtx, items[i], i == sel)
	})
}

// scrolled is the body both scrollbar entry points share: the rows under the
// bar, the bar on the trailing edge wired to the position the rows resolved,
// and the bar's own drag folded back in for the next frame. n is the item
// count and rowFn draws item i.
func (state *State) scrolled(
	gtx layout.Context,
	bar scrollbar.Style,
	anchor Anchor,
	n int,
	rowFn func(gtx layout.Context, i int) layout.Dimensions,
) layout.Dimensions {
	originalConstraints := gtx.Constraints
	barWidth := 0
	var listDims layout.Dimensions
	if anchor == Occupy {
		barWidth = gtx.Dp(bar.Width())
		listDims, barWidth = state.occupied(gtx, n, rowFn, barWidth)
	} else {
		listDims = state.layoutRows(gtx, n, rowFn)
	}
	gtx.Constraints = originalConstraints

	// Draw the scrollbar. layout.Direction respects the minimum, so pin the
	// minimum to the laid-out list size (re-widened by the lane for Occupy)
	// to ensure the bar lands on the trailing edge even when the incoming
	// minimum constraint was zero.
	start, end := scrollbar.FromListPosition(state.l.Position, n, listDims.Size.Y)
	gtx.Constraints.Min = listDims.Size
	gtx.Constraints.Min.X += barWidth
	layout.E.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return bar.Layout(gtx, &state.sb, layout.Vertical, start, end)
	})

	// Apply any scroll caused by interaction with the bar this frame.
	state.applyScrollDelta(state.sb.ScrollDistance(), n)

	// Report the lane as part of the occupied space.
	listDims.Size.X += barWidth
	return listDims
}

// occupied lays out the rows short of a lane lane pixels wide, and returns
// their dimensions and the lane they stop short of. A lane exists only for a
// bar that is drawn: when the rows short of the lane fit, the bar draws
// nothing, so the rows are laid out again across the full width and the lane
// returned is 0. Rows laid out wider are never taller, so rows that fit short
// of the lane fit across the full width too.
//
// The first layout is recorded, with its floating surfaces collected apart,
// so a layout that is discarded leaves no paint and places no surface.
func (state *State) occupied(gtx layout.Context, n int, rowFn func(gtx layout.Context, i int) layout.Dimensions, lane int) (layout.Dimensions, int) {
	full := gtx
	gtx.Constraints.Max.X = max(gtx.Constraints.Max.X-lane, 0)
	gtx.Constraints.Min.X = max(gtx.Constraints.Min.X-lane, 0)
	var surfaces *attachment.Surfaces
	if attachment.HasSurfaces(gtx) || attachment.HasAvailableRoom(gtx) {
		gtx, surfaces = attachment.WithSurfaces(gtx)
	}
	macro := op.Record(gtx.Ops)
	dims := state.layoutRows(gtx, n, rowFn)
	call := macro.Stop()
	if start, end := scrollbar.FromListPosition(state.l.Position, n, dims.Size.Y); start <= 0 && end >= 1 {
		return state.layoutRows(full, n, rowFn), 0
	}
	call.Add(full.Ops)
	attachment.PlaceSurfacesAt(full, surfaces, image.Point{})
	return dims, lane
}

// applyScrollDelta translates a scrollbar drag distance (a fraction of the
// total content in [-1, 1]) into a list scroll of delta × elements rows.
// The new position takes effect on the next layout.
func (s *State) applyScrollDelta(delta float32, elements int) {
	if delta != 0 {
		s.l.ScrollBy(delta * float32(elements))
	}
}

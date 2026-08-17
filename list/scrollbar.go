package list

import (
	"gioui.org/layout"

	"github.com/vibrantgio/components/scrollbar"
)

// Anchor defines how a scrollbar is attached to the list content.
type Anchor int

const (
	// Occupy reserves a gutter of bar.Width() along the list's trailing
	// edge, narrowing the rows. Content is never occluded, at the cost of
	// a slightly smaller content area even when the bar is idle.
	Occupy Anchor = iota
	// Overlay floats the bar over the content's trailing edge. Rows keep
	// their full width, at the cost of the bar occluding whatever content
	// sits beneath it.
	Overlay
)

// LayoutScrollbar lays out items exactly like Layout and additionally draws
// bar along the list's trailing (east) edge, wired to the list's scroll
// position. Dragging the bar scrolls the list.
//
// anchor selects whether the bar reserves a gutter (Occupy) or floats over
// the content (Overlay). The list is vertical-only, so the bar is always
// vertical and anchored east.
func LayoutScrollbar[T any](
	gtx layout.Context,
	state *State,
	bar scrollbar.Style,
	anchor Anchor,
	items []T,
	rowFn func(gtx layout.Context, item T) layout.Dimensions,
) layout.Dimensions {
	state.record(gtx)
	originalConstraints := gtx.Constraints
	barWidth := gtx.Dp(bar.Width())

	if anchor == Occupy {
		// Reserve the gutter so rows lay out narrower.
		gtx.Constraints.Max.X = max(gtx.Constraints.Max.X-barWidth, 0)
		gtx.Constraints.Min.X = max(gtx.Constraints.Min.X-barWidth, 0)
	}

	listDims := state.l.Layout(gtx, len(items), func(gtx layout.Context, i int) layout.Dimensions {
		return rowFn(gtx, items[i])
	})
	gtx.Constraints = originalConstraints

	// Draw the scrollbar. layout.Direction respects the minimum, so pin the
	// minimum to the laid-out list size (re-widened by the gutter for Occupy)
	// to ensure the bar lands on the trailing edge even when the incoming
	// minimum constraint was zero.
	start, end := scrollbar.FromListPosition(state.l.Position, len(items), listDims.Size.Y)
	gtx.Constraints.Min = listDims.Size
	if anchor == Occupy {
		gtx.Constraints.Min.X += barWidth
	}
	layout.E.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return bar.Layout(gtx, &state.sb, layout.Vertical, start, end)
	})

	// Apply any scroll caused by interaction with the bar this frame.
	state.applyScrollDelta(state.sb.ScrollDistance(), len(items))

	if anchor == Occupy {
		// Report the gutter as part of the occupied space.
		listDims.Size.X += barWidth
	}
	return listDims
}

// applyScrollDelta translates a scrollbar drag distance (a fraction of the
// total content in [-1, 1]) into a list scroll of delta × elements rows.
// The new position takes effect on the next layout.
func (s *State) applyScrollDelta(delta float32, elements int) {
	if delta != 0 {
		s.l.ScrollBy(delta * float32(elements))
	}
}

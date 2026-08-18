package list

import (
	"gioui.org/io/event"
	"gioui.org/layout"
	"gioui.org/op/clip"

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
	barWidth := gtx.Dp(bar.Width())

	if anchor == Occupy {
		// Reserve the gutter so rows lay out narrower.
		gtx.Constraints.Max.X = max(gtx.Constraints.Max.X-barWidth, 0)
		gtx.Constraints.Min.X = max(gtx.Constraints.Min.X-barWidth, 0)
	}

	listDims := state.l.Layout(gtx, n, rowFn)
	gtx.Constraints = originalConstraints

	// Draw the scrollbar. layout.Direction respects the minimum, so pin the
	// minimum to the laid-out list size (re-widened by the gutter for Occupy)
	// to ensure the bar lands on the trailing edge even when the incoming
	// minimum constraint was zero.
	start, end := scrollbar.FromListPosition(state.l.Position, n, listDims.Size.Y)
	gtx.Constraints.Min = listDims.Size
	if anchor == Occupy {
		gtx.Constraints.Min.X += barWidth
	}
	layout.E.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return bar.Layout(gtx, &state.sb, layout.Vertical, start, end)
	})

	// Apply any scroll caused by interaction with the bar this frame.
	state.applyScrollDelta(state.sb.ScrollDistance(), n)

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

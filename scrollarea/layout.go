package scrollarea

import (
	"image"

	"gioui.org/gesture"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"

	"github.com/vibrantgio/components/list"
	"github.com/vibrantgio/components/scrollbar"
)

// inf is the width the child is measured at: unbounded in practice, and small
// enough that a child doing arithmetic on its constraint does not overflow.
// It is gioui.org/layout's own choice for the same job in layout.List.
const inf = 1e6

// Layout shows w through a horizontally scrolling viewport.
//
// w is laid out once per frame at its natural width — the horizontal
// constraint it is given is unbounded, so it neither wraps nor stretches — and
// the result is clipped to the incoming maximum width. Pointer scrolling along
// the horizontal axis moves the viewport; the vertical axis is left alone (see
// the package documentation).
//
// The returned width is the child's when it fits and the viewport's when it
// does not, never more, so a caller can lay the area out among siblings
// without knowing whether this frame's content overflowed. The height is the
// child's own.
//
// Content past an edge is clipped at the edge and nothing is drawn over it.
// Content that fits is drawn exactly as it would be without a scroll area:
// the clip cuts nothing, and the area reserves no space.
func (s Style) Layout(gtx layout.Context, state *State, w layout.Widget) layout.Dimensions {
	return s.layout(gtx, state, w, nil)
}

// LayoutScrollbar shows w as [Style.Layout] does and draws bar at the area's
// own bottom edge, inside the area, wired to the offset the content resolved.
// Dragging the bar scrolls the area. The bar draws nothing while the content
// fits, and the area then takes no lane.
//
// The area is the region that scrolls, so its bar is at the area's edge.
// A bar that stands ([list.AnchorFor] answers [list.Occupy]) has a lane of
// its own below the content: the area is taller than its content by the bar's
// [scrollbar.Style.Width], and the content stops short of the lane. A bar
// that fades ([list.Overlay]) takes no lane: it is drawn over the content's
// bottom rows, and the area's height is its content's.
//
// The bar shows how much content there is and where the viewport is. The
// user can drag it.
func (s Style) LayoutScrollbar(gtx layout.Context, state *State, bar scrollbar.Style, w layout.Widget) layout.Dimensions {
	return s.layout(gtx, state, w, &bar)
}

// layout is the body both entry points share. bar is nil for the bare area.
func (s Style) layout(gtx layout.Context, state *State, w layout.Widget, bar *scrollbar.Style) layout.Dimensions {
	viewport := gtx.Constraints.Max.X

	// Measure the child at its natural width. Recording it also keeps the
	// child's draw commands out of the ops list until the clip is in place.
	cgtx := gtx
	cgtx.Constraints.Min.X = 0
	cgtx.Constraints.Max.X = inf
	macro := op.Record(gtx.Ops)
	child := w(cgtx)
	call := macro.Stop()

	// Clamp against this frame's geometry before reading the pointer, so a
	// viewport that grew or content that shrank since the last frame bounds
	// the gesture rather than the other way round.
	state.content, state.viewport = child.Size.X, viewport
	state.offset = clamp(state.offset, 0, state.MaxOffset())
	dist := s.scrolled(gtx, state)
	state.offset = clamp(state.offset+dist, 0, state.MaxOffset())

	size := image.Pt(clamp(child.Size.X, gtx.Constraints.Min.X, viewport), child.Size.Y)
	lane := 0
	if bar != nil && state.Overflows() && list.AnchorFor(*bar) == list.Occupy {
		lane = gtx.Dp(bar.Width())
	}
	area := clip.Rect{Max: size}.Push(gtx.Ops)
	state.scroll.Add(gtx.Ops)
	tr := op.Offset(image.Pt(-state.offset, 0)).Push(gtx.Ops)
	call.Add(gtx.Ops)
	tr.Pop()
	area.Pop()

	if bar != nil {
		s.scrollbar(gtx, state, *bar, size, lane)
	}
	return layout.Dimensions{Size: image.Pt(size.X, size.Y+lane), Baseline: child.Baseline}
}

// scrolled returns the pixels this frame's pointer events ask the viewport to
// move by.
//
// The vertical range is deliberately empty. A gesture.Scroll's ranges are what
// the router uses to decide whether an event is consumed here or offered to an
// ancestor, so an empty vertical range is how this area declines the vertical
// axis and lets a wheel over it scroll the column it sits in. The horizontal
// range is bounded by what is actually left to scroll in each direction, so an
// area already at its trailing edge declines a further rightward gesture too.
func (s Style) scrolled(gtx layout.Context, state *State) int {
	if !state.Overflows() {
		// Nothing to scroll: register no range at all, so the area is
		// transparent to the pointer on both axes.
		return 0
	}
	return state.scroll.Update(gtx.Metric, gtx.Source, gtx.Now, gesture.Horizontal,
		pointer.ScrollRange{Min: -state.offset, Max: state.MaxOffset() - state.offset},
		pointer.ScrollRange{})
}

// scrollbar draws the horizontal bar at the area's bottom edge and folds this
// frame's drag on it back into the offset. content is the size of the
// content's viewport. With a lane, the bar is drawn in the lane below the
// viewport; with none, it is drawn over the viewport's bottom rows.
func (s Style) scrollbar(gtx layout.Context, state *State, bar scrollbar.Style, content image.Point, lane int) {
	start, end := state.Fractions()
	h := gtx.Dp(bar.Width())
	defer op.Offset(image.Pt(0, content.Y+lane-h)).Push(gtx.Ops).Pop()
	gtx.Constraints = layout.Exact(image.Pt(content.X, h))
	bar.Layout(gtx, &state.sb, layout.Horizontal, start, end)
	if d := state.sb.ScrollDistance(); d != 0 {
		state.ScrollBy(int(d * float32(state.content)))
	}
}

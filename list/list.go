// Package list provides a virtual-scrolling list component for Gio.
//
// Only items in the current viewport are laid out — O(visible), not O(total).
//
// # Entry points
//
// There are two ways to lay out a list, both driven by the same State:
//
//   - [Layout] lays out the bare list: wheel/touch scrolling only, no
//     scrollbar is drawn.
//   - [LayoutScrollbar] additionally draws a scrollbar along the list's
//     trailing edge, wired to the scroll position; dragging the thumb or
//     clicking the track scrolls the list.
//
// LayoutScrollbar takes an [Anchor] that decides where the bar lives:
//
//   - [Occupy] reserves a gutter of the bar's width, narrowing the rows.
//     Pick it when rows must never be occluded — e.g. text that would
//     otherwise disappear under the thumb, or rows with interactive
//     controls at their trailing edge.
//   - [Overlay] floats the bar over the rows, keeping their full width.
//     Pick it when every pixel of row width matters and brief occlusion
//     along the trailing edge is acceptable.
//
// The bar's appearance is a scrollbar.Style; derive the default themed one
// with scrollbar.FromTokens (github.com/vibrantgio/prism/scrollbar).
//
// For lists with reorderable rows that contain interactive Gio widgets (editors,
// checkboxes, etc.), pair with keyed.Defer from prism/keyed to keep per-row
// widget state stable across reorders.
package list

import (
	"gioui.org/layout"
	"gioui.org/unit"

	"github.com/vibrantgio/prism/scrollbar"
	"github.com/vibrantgio/spectrum/tokens"
)

// RowHeight returns the standard list row height for a density: exactly
// ControlHeight — 36 dp at tokens.Comfortable, 28 dp at tokens.Compact (E1.3;
// the same rule sizes dropdown option rows). Rows are drawn by the caller's
// rowFn, so this is the height rowFn should give a standard single-line row;
// the list itself imposes no height. Adjacent full-width rows are their own
// pointer targets, so a row-builder wiring interaction should keep the row's
// hit area at the row bounds rather than extending it over its neighbours.
func RowHeight(d tokens.Density) unit.Dp {
	return unit.Dp(d.ControlHeight)
}

// State holds the scroll position across frames.
// Allocate once per list instance and reuse on every frame.
//
// The embedded scrollbar state (used only by LayoutScrollbar) is zero-value
// ready, so existing NewState/NewStateAt callers are unaffected.
type State struct {
	l layout.List
	// sb holds the scrollbar gesture state for LayoutScrollbar. It is a
	// scrollbar.State (which embeds gioui.org/widget.Scrollbar) rather than
	// widget.Scrollbar directly so &sb can be passed to scrollbar.Style.Layout
	// while ScrollDistance remains reachable via promotion.
	sb scrollbar.State
}

// NewState returns a State for a vertical list starting at the top.
func NewState() *State {
	return &State{l: layout.List{Axis: layout.Vertical}}
}

// NewStateAt returns a State whose initial first-visible item index is first.
// Intended for golden-image testing; production code uses NewState and lets
// pointer events drive scrolling.
func NewStateAt(first int) *State {
	return &State{l: layout.List{
		Axis:     layout.Vertical,
		Position: layout.Position{First: first},
	}}
}

// Layout lays out items in a virtual scrolling list. rowFn is called only for
// items in the current viewport: O(visible), not O(len(items)).
//
// rowFn must not retain gtx past its call; the closure is invoked once per
// visible item inside the layout.List callback.
func Layout[T any](
	gtx layout.Context,
	state *State,
	items []T,
	rowFn func(gtx layout.Context, item T) layout.Dimensions,
) layout.Dimensions {
	return state.l.Layout(gtx, len(items), func(gtx layout.Context, i int) layout.Dimensions {
		return rowFn(gtx, items[i])
	})
}

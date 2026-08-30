// Package list provides a virtual-scrolling list component for Gio.
//
// Only items in the current viewport are laid out — O(visible), not O(total).
//
// # Entry points
//
// There are four ways to lay out a list, all driven by the same State:
//
//   - [Layout] lays out the bare list: wheel/touch scrolling only, no
//     scrollbar is drawn.
//   - [LayoutScrollbar] additionally draws a scrollbar along the list's
//     trailing edge, wired to the scroll position; dragging the thumb or
//     clicking the track scrolls the list.
//   - [LayoutSelectable] adds a keyboard-movable selection over every row.
//   - [LayoutSelectableScrollbar] is both: the selection and the bar.
//
// The two scrollbar entry points take an [Anchor] that decides where the bar
// lives:
//
//   - [Occupy] reserves a gutter of the bar's width, narrowing the rows.
//     Pick it when rows must never be occluded — e.g. text that would
//     otherwise disappear under the thumb, or rows with interactive
//     controls at their trailing edge.
//   - [Overlay] floats the bar over the rows, keeping their full width.
//     Pick it when every pixel of row width matters and brief occlusion
//     along the trailing edge is acceptable.
//
// # Keyboard traversal
//
// [LayoutSelectable] adds a selection the keyboard can move over the whole
// list — every row, not only the laid-out ones. That distinction is the whole
// reason it lives here and not in each caller: focus tags cannot be the
// mechanism, because a row virtualisation never laid out has no tag to focus,
// so Tab reaches exactly what is on screen and nothing beyond it. The list
// instead takes a single focus tag of its own ([State.Focus]) that survives
// virtualisation, and moves an index. See [LayoutSelectable] for the key map
// and [State.Select]/[State.Reveal] for driving it from the caller's side.
//
// The bar's appearance is a scrollbar.Style; derive the default themed one
// with scrollbar.FromTokens (github.com/vibrantgio/components/scrollbar).
//
// For lists with reorderable rows that contain interactive Gio widgets (editors,
// checkboxes, etc.), pair with keyed.Defer from components/keyed to keep per-row
// widget state stable across reorders.
package list

import (
	"gioui.org/io/event"
	"gioui.org/io/key"
	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/unit"

	"github.com/vibrantgio/components/scrollbar"
	"github.com/vibrantgio/theme/tokens"
)

// RowHeight returns the standard list row height for a density: exactly
// ControlHeight — 36 dp at tokens.Comfortable, 28 dp at tokens.Compact; the
// same rule sizes dropdown option rows. Rows are drawn by the caller's
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

	// focus is the list's single keyboard focus target. One tag for
	// the whole list, never one per row: a row that virtualisation skipped
	// has no tag, so per-row tags can only ever reach what is on screen.
	focus focusTag
	// selPlusOne is the selected row index plus one, and scrollPlusOne the
	// pending Reveal target plus one. Both are stored offset by one so the
	// zero State means "nothing selected, nothing pending" — a State reached
	// without NewState still behaves, as it did before selection existed.
	selPlusOne    int
	scrollPlusOne int

	// viewport is the main-axis size in pixels the most recent layout ran
	// in, recorded so a caller can move the list by a viewport without
	// having to hold a layout.Context of its own. See move.go.
	viewport int
}

// focusTag is a non-zero-size type so its address is a unique event tag for
// the list's keyboard focus target. A zero-size struct{} field could share an
// address with a neighbouring field, which would break tag identity.
type focusTag struct{ _ byte }

// Focus returns the list's keyboard focus tag: the one event.Tag that
// [LayoutSelectable] registers, and the tag a caller passes to key.FocusCmd to
// hand the list the keyboard, or to key.Filter to read keys the list does not
// itself consume — Enter and Space, typically, since activating a selection is
// the caller's semantics and not the list's.
//
// It is stable for the lifetime of the State and independent of which rows are
// laid out.
func (s *State) Focus() event.Tag { return &s.focus }

// Selected returns the index of the selected row, or -1 when nothing is
// selected. It is an index into the caller's item slice, so it stays
// meaningful for rows the current frame never laid out.
//
// The answer is always either -1 or a row the last [LayoutSelectable] was
// given: a selection the item slice no longer carries is dropped rather than
// moved, so a caller driving a detail pane off it never shows a row the user
// did not choose.
func (s *State) Selected() int { return s.selPlusOne - 1 }

// Select sets the selection to row i, or clears it when i is negative. The
// viewport does not move: pair it with [State.Reveal] when the row may be
// offscreen. Selecting in response to a pointer click needs no Reveal — the
// row was visible enough to be clicked.
//
// i is not bounded here — the item count is not known until layout. The next
// [LayoutSelectable] drops a selection its slice does not carry; it does not
// clamp it to the last row.
func (s *State) Select(i int) {
	if i < 0 {
		s.selPlusOne = 0
		return
	}
	s.selPlusOne = i + 1
}

// Reveal schedules row i to be scrolled into view by the next
// [LayoutSelectable]. It moves the viewport the short way: a row above the
// window lands at the leading edge, a row below it at the trailing edge, and a
// row already in view does not move the list at all.
//
// It does not change the selection. Keyboard traversal calls it for every move
// it makes, so callers need it only when they select programmatically.
func (s *State) Reveal(i int) {
	if i < 0 {
		s.scrollPlusOne = 0
		return
	}
	s.scrollPlusOne = i + 1
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
	state.record(gtx)
	return state.l.Layout(gtx, len(items), func(gtx layout.Context, i int) layout.Dimensions {
		return rowFn(gtx, items[i])
	})
}

// LayoutSelectable lays out items exactly like [Layout] and additionally makes
// the whole list reachable from the keyboard. rowFn is told whether the row it
// is drawing is the selected one, so the caller renders selection its own way;
// the list draws nothing extra — a scrollbar included, for which see
// [LayoutSelectableScrollbar].
//
// The list is one focus target ([State.Focus]), which is what makes traversal
// possible at all: it is registered every frame regardless of which rows exist,
// so the keys below reach rows that were never laid out. While it holds focus:
//
//	Up / Down     move the selection one row, stopping at the ends (no wrap).
//	              With nothing selected, Down selects the first row and Up the
//	              last.
//	Home / End    select the first and last row.
//
// Every move Reveals the new selection, so the viewport follows it. Keys the
// list does not consume — Enter, Space, anything else — are left for the
// caller to filter on [State.Focus]; the list has no notion of activating a
// row.
//
// The focus target covers the list's viewport (gtx.Constraints.Max) and is
// registered beneath the rows, so per-row pointer handlers keep priority over
// it. Nothing here moves focus on its own: a caller that wants a pointer click
// to hand the list the keyboard executes key.FocusCmd{Tag: state.Focus()},
// which is also what makes subsequent arrow keys land on the list rather than
// on whatever the click focused.
func LayoutSelectable[T any](
	gtx layout.Context,
	state *State,
	items []T,
	rowFn func(gtx layout.Context, item T, selected bool) layout.Dimensions,
) layout.Dimensions {
	n := len(items)
	state.record(gtx)
	state.dropStaleSelection(n)
	state.processKeys(gtx, n)
	state.scrollIntoView(n)

	area := clip.Rect{Max: gtx.Constraints.Max}.Push(gtx.Ops)
	event.Op(gtx.Ops, &state.focus)
	area.Pop()

	sel := state.Selected()
	return state.l.Layout(gtx, n, func(gtx layout.Context, i int) layout.Dimensions {
		return rowFn(gtx, items[i], i == sel)
	})
}

// dropStaleSelection clears a selection that the current item slice no longer
// carries. Items shrink between frames — a filter narrowing a hundred rows to
// three is the ordinary case — and an index past the end names a row that does
// not exist.
//
// Clearing is what makes that legible: [State.Selected] answers -1, the value
// every caller already handles, and the stale index cannot reappear as an
// unrelated row when the slice grows back. Clamping to the last row instead
// would hand a caller driving a detail pane off Selected() a valid-looking
// index the user never chose, indistinguishable from a real selection.
//
// The pending [State.Reveal] target is deliberately not dropped here:
// scrollIntoView clamps it to the last row, because "move the viewport to that
// row" still has a nearest sensible answer once the named row is gone, while
// "the user chose that row" does not.
func (s *State) dropStaleSelection(n int) {
	if s.selPlusOne > n {
		s.selPlusOne = 0
	}
}

// processKeys drains this frame's traversal keys for the list's focus tag.
// The FocusFilter is what makes the tag focusable at all — without it the
// router refuses to hold focus on the list, and every key.Filter below is
// dead.
func (s *State) processKeys(gtx layout.Context, n int) {
	tag := &s.focus
	for {
		e, ok := gtx.Event(
			key.FocusFilter{Target: tag},
			key.Filter{Focus: tag, Name: key.NameUpArrow},
			key.Filter{Focus: tag, Name: key.NameDownArrow},
			key.Filter{Focus: tag, Name: key.NameHome},
			key.Filter{Focus: tag, Name: key.NameEnd},
		)
		if !ok {
			return
		}
		ke, ok := e.(key.Event)
		if !ok || ke.State != key.Press || n == 0 {
			continue
		}
		sel := s.Selected()
		switch ke.Name {
		case key.NameUpArrow:
			switch {
			case sel < 0:
				sel = n - 1
			case sel > 0:
				sel--
			}
		case key.NameDownArrow:
			switch {
			case sel < 0:
				sel = 0
			case sel < n-1:
				sel++
			}
		case key.NameHome:
			sel = 0
		case key.NameEnd:
			sel = n - 1
		default:
			continue
		}
		s.Select(sel)
		s.Reveal(sel)
	}
}

// scrollIntoView consumes a pending [State.Reveal] target, moving the viewport
// only as far as it must.
//
// The window it measures against is the previous frame's: Position.First and
// Position.Count are what the last layout resolved. That is enough because the
// only thing being decided is which edge to land the target on — the layout
// itself then re-derives everything from First.
func (s *State) scrollIntoView(n int) {
	if s.scrollPlusOne == 0 || n == 0 {
		return
	}
	target := s.scrollPlusOne - 1
	s.scrollPlusOne = 0
	if target >= n {
		target = n - 1
	}
	p := s.l.Position
	// visible is how many rows the window shows *whole*. Position.Count is one
	// more than that whenever the trailing child is only partly visible —
	// OffsetLast < 0 is exactly that case — and both decisions below need the
	// whole-row count, not the laid-out count:
	//
	//   - the window test, because a target that *is* the clipped trailing row
	//     is not "already visible" and must still scroll;
	//   - the landing, because a target placed Count-1 rows below the new first
	//     row would arrive clipped, exactly where it started.
	//
	// A partly visible *leading* child is not subtracted as well. It is still a
	// row this window covers, so it must not shift the trailing edge; the case
	// below catches it by index instead (target == First && Offset > 0).
	visible := p.Count
	if p.OffsetLast < 0 {
		visible--
	}
	if visible < 1 {
		// A single row taller than the viewport: no row shows whole, and the
		// window is one row wide regardless.
		visible = 1
	}
	switch {
	case p.Count == 0, target < p.First, target == p.First && p.Offset > 0:
		// Above the window, or scrolled part-way off its leading edge — and on
		// the first frame there is no window yet. Land it at the leading edge.
		s.l.ScrollTo(target)
	case target >= p.First+visible:
		// Below the window, or the clipped row at its trailing edge: land it at
		// the trailing edge with visible-1 whole rows above it.
		first := target - visible + 1
		if first < 0 {
			first = 0
		}
		s.l.ScrollTo(first)
	}
}

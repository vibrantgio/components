package list_test

import (
	"image"
	"testing"

	gioinput "gioui.org/io/input"
	"gioui.org/io/key"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"

	golden "github.com/vibrantgio/prism/golden"
	"github.com/vibrantgio/prism/list"
)

// viewportHeights is the set of viewport heights every scrolling test here runs
// at. Only the first is a whole number of rows; the others leave a partial row
// at the trailing edge, which is where F5.2's off-by-one lived — the window
// test counted that clipped row as visible, so a selection could land on it and
// the next arrow press then moved the viewport two rows. An exact multiple is
// the one shape in which no row is ever clipped, and so the one shape that
// cannot catch it.
var viewportHeights = []struct {
	name string
	h    int
}{
	{"exact-5-rows", viewH},             // 150 px: the shape the defect hid in
	{"4-rows-plus-17px", 137},           // 137 px
	{"4-rows-plus-23px", viewHPartial},  // 143 px: the permanent fixture height
	{"5-rows-plus-5px", 155},            // 155 px: barely a sixth row
	{"5-rows-plus-11px", 161},           // 161 px
	{"1-row-plus-5px", rowPx + rowPx/6}, // 35 px: the window is one whole row
}

// selectableList is a list driven through a real input.Router, recording which
// rows each frame actually laid out. It is the fixture for every test here:
// the question these tests ask is always "which rows exist, and can the
// keyboard reach the ones that do not".
type selectableList struct {
	state *list.State
	items []int
	r     *gioinput.Router
	ops   *op.Ops
	size  image.Point

	// laidOut holds the item indices rowFn was called with on the most recent
	// frame, in call order.
	laidOut []int
	// selectedRows holds the indices rowFn was told were selected. It is a
	// slice, not an int, because "exactly one row draws as selected" is itself
	// part of the contract.
	selectedRows []int
}

func newSelectableList(n int, size image.Point) *selectableList {
	return &selectableList{
		state: list.NewState(),
		items: makeItems(n),
		r:     new(gioinput.Router),
		ops:   new(op.Ops),
		size:  size,
	}
}

// frame lays out one frame and hands the ops to the router, the way a window
// would.
func (s *selectableList) frame() {
	s.ops.Reset()
	s.laidOut = s.laidOut[:0]
	s.selectedRows = s.selectedRows[:0]
	gtx := layout.Context{
		Metric:      unit.Metric{PxPerDp: 1, PxPerSp: 1},
		Constraints: layout.Exact(s.size),
		Ops:         s.ops,
		Source:      s.r.Source(),
	}
	list.LayoutSelectable(gtx, s.state, s.items, func(gtx layout.Context, item int, selected bool) layout.Dimensions {
		s.laidOut = append(s.laidOut, item)
		if selected {
			s.selectedRows = append(s.selectedRows, item)
		}
		return colorRowFn(gtx, item)
	})
	s.r.Frame(s.ops)
}

// focus hands the list the keyboard the way Tab does, through the router's own
// traversal rather than by reaching into the state.
func (s *selectableList) focus(t *testing.T) {
	t.Helper()
	s.frame()
	s.r.MoveFocus(key.FocusForward)
	s.frame()
}

func (s *selectableList) press(name key.Name) {
	s.r.Queue(key.Event{Name: name, State: key.Press})
	s.frame()
}

func (s *selectableList) sawRow(i int) bool {
	for _, v := range s.laidOut {
		if v == i {
			return true
		}
	}
	return false
}

// capture renders the list's present state and returns the pixels.
//
// Pixels are the only honest answer to "what does the viewport actually show".
// The laid-out set cannot say it: a row clipped by the viewport edge is laid out
// exactly like one that fits — that is precisely how F5.2's defect passed its
// own tests — and the *order* of that set is not the scroll position either,
// since Gio lays out its look-ahead children forwards or backwards depending on
// where the previous frame left Position.
//
// The extra frame is a pure re-render: the pending Reveal was already consumed
// by the frame under test, and laying out again from the same Position resolves
// to the same Position.
func (s *selectableList) capture(t *testing.T) *image.RGBA {
	t.Helper()
	return golden.Capture(t, s.size, func(gtx layout.Context) layout.Dimensions {
		return list.LayoutSelectable(gtx, s.state, s.items, func(gtx layout.Context, item int, _ bool) layout.Dimensions {
			return colorRowFn(gtx, item)
		})
	})
}

// fullyVisible reports whether row i is currently drawn whole, by counting its
// scanlines. Rows are flat colour and rowPx tall, so a whole row is rowPx
// scanlines of it and a clipped one fewer. rowColor repeats every 19 items and
// no viewport here is that tall, so within one frame the colour names the row
// unambiguously.
func (s *selectableList) fullyVisible(t *testing.T, i int) bool {
	t.Helper()
	img := s.capture(t)
	want := rowColor(i)
	b := img.Bounds()
	x := b.Min.X + s.size.X/2
	n := 0
	for y := b.Min.Y; y < b.Max.Y; y++ {
		if c := img.RGBAAt(x, y); c.R == want.R && c.G == want.G && c.B == want.B && c.A == want.A {
			n++
		}
	}
	return n == rowPx
}

// TestSelectionReachesRowNeverLaidOut is the test the whole task exists for.
//
// A 1000-item list in a 143 px viewport lays out five rows and a look-ahead;
// rows 6..999 have no focus tag, no clip area and no existence in the op tree,
// so no amount of Tab reaches them. End must select row 999 anyway, and the
// list must then scroll it into view so it is laid out — which is the proof
// that the selection is an index over the whole slice and not a walk over the
// tags that happen to exist.
func TestSelectionReachesRowNeverLaidOut(t *testing.T) {
	const n = 1000
	s := newSelectableList(n, image.Pt(viewW, viewHPartial))
	s.focus(t)

	firstFrame := append([]int(nil), s.laidOut...)
	if len(firstFrame) == 0 {
		t.Fatal("first frame laid out no rows at all")
	}
	if len(firstFrame) >= n {
		t.Fatalf("first frame laid out %d of %d rows; the list is not virtualising, so this test proves nothing", len(firstFrame), n)
	}
	last := n - 1
	for _, i := range firstFrame {
		if i == last {
			t.Fatalf("row %d was laid out on the first frame; pick a viewport that cannot show it", last)
		}
	}

	s.press(key.NameEnd)
	if got := s.state.Selected(); got != last {
		t.Fatalf("after End, Selected() = %d; want %d (a row the first frame never laid out)", got, last)
	}
	if !s.sawRow(last) {
		t.Fatalf("after End, row %d was still not laid out (laid out %v); the selection moved but the viewport did not follow", last, s.laidOut)
	}
	if len(s.selectedRows) != 1 || s.selectedRows[0] != last {
		t.Errorf("rowFn reported selected rows %v; want exactly [%d]", s.selectedRows, last)
	}

	// And back: Home reaches the other end from 999 rows away.
	s.press(key.NameHome)
	if got := s.state.Selected(); got != 0 {
		t.Fatalf("after Home, Selected() = %d; want 0", got)
	}
	if !s.sawRow(0) {
		t.Fatalf("after Home, row 0 was not laid out (laid out %v)", s.laidOut)
	}
}

// TestArrowTraversalMovesOneRow pins the arrow key map: Down from no selection
// picks the first row, each further Down advances one, Up retreats one, and
// both stop at the ends rather than wrapping.
func TestArrowTraversalMovesOneRow(t *testing.T) {
	const n = 20
	s := newSelectableList(n, image.Pt(viewW, viewHPartial))
	s.focus(t)

	if got := s.state.Selected(); got != -1 {
		t.Fatalf("Selected() before any key = %d; want -1 (nothing selected)", got)
	}
	for want := 0; want <= 2; want++ {
		s.press(key.NameDownArrow)
		if got := s.state.Selected(); got != want {
			t.Fatalf("after %d Down press(es), Selected() = %d; want %d", want+1, got, want)
		}
	}
	s.press(key.NameUpArrow)
	if got := s.state.Selected(); got != 1 {
		t.Fatalf("after Up, Selected() = %d; want 1", got)
	}

	// Up at the leading end stays put.
	s.press(key.NameUpArrow)
	s.press(key.NameUpArrow)
	if got := s.state.Selected(); got != 0 {
		t.Fatalf("after Up past the first row, Selected() = %d; want 0 (no wrap)", got)
	}

	// Down at the trailing end stays put.
	s.press(key.NameEnd)
	s.press(key.NameDownArrow)
	if got := s.state.Selected(); got != n-1 {
		t.Fatalf("after Down past the last row, Selected() = %d; want %d (no wrap)", got, n-1)
	}

	// Up from nothing selected picks the last row.
	fresh := newSelectableList(n, image.Pt(viewW, viewHPartial))
	fresh.focus(t)
	fresh.press(key.NameUpArrow)
	if got := fresh.state.Selected(); got != n-1 {
		t.Fatalf("Up with nothing selected chose %d; want %d (the last row)", got, n-1)
	}
}

// TestSelectionScrollsIntoViewOneRowAtATime checks the viewport follows the
// selection by the smallest move that works: stepping Down past the trailing
// edge scrolls one row, not one page, every intermediate selection is actually
// laid out — and it is laid out *whole*. A list that jumped the selection to
// the top of the viewport on every step would pass "the row is visible" and
// still be useless; a list that stops one row short leaves the selection on a
// clipped row and then moves two rows on the next press.
//
// It runs at every height in viewportHeights, because at an exact multiple of
// the row height no row is ever clipped and the two-row jump cannot occur.
func TestSelectionScrollsIntoViewOneRowAtATime(t *testing.T) {
	const n = 40
	for _, vh := range viewportHeights {
		t.Run(vh.name, func(t *testing.T) {
			s := newSelectableList(n, image.Pt(viewW, vh.h))
			s.focus(t)

			prevFirst := -1
			for i := 0; i < 12; i++ {
				s.press(key.NameDownArrow)
				if !s.sawRow(i) {
					t.Fatalf("after selecting row %d it was not laid out (laid out %v)", i, s.laidOut)
				}
				if !s.fullyVisible(t, i) {
					t.Fatalf("after selecting row %d it is drawn clipped in a %d px viewport of %d px rows; the selection must land whole in view",
						i, vh.h, rowPx)
				}
				first := s.laidOut[0]
				if first < prevFirst {
					t.Fatalf("selecting row %d scrolled the list backwards: first visible %d, was %d", i, first, prevFirst)
				}
				if first > prevFirst+1 && prevFirst >= 0 {
					t.Fatalf("selecting row %d scrolled the list by %d rows; one Down should move the viewport at most one row",
						i, first-prevFirst)
				}
				prevFirst = first
			}
			if prevFirst == 0 {
				t.Fatalf("twelve Down presses never scrolled the list; a %d px viewport cannot hold twelve %d px rows", vh.h, rowPx)
			}

			// Scrolling back up lands the selection at the leading edge, not the
			// trailing one.
			for i := 11; i >= 0; i-- {
				s.press(key.NameUpArrow)
			}
			if got := s.state.Selected(); got != 0 {
				t.Fatalf("Selected() = %d after walking back up; want 0", got)
			}
			if s.laidOut[0] != 0 {
				t.Fatalf("first visible row after walking back to the top = %d; want 0", s.laidOut[0])
			}
		})
	}
}

// TestRevealLandsRowFullyInView states the contract F5.2 repairs in one line:
// whatever the viewport height, the row Reveal names is drawn whole afterwards.
//
// The targets walk deliberately through the awkward ones — the clipped row at
// the trailing edge of the first window, the row just past it, a row far below,
// the last row (where the list end-aligns instead of honouring First), and back
// to the top.
func TestRevealLandsRowFullyInView(t *testing.T) {
	const n = 60
	targets := []int{4, 5, 12, 40, n - 1, 0}
	for _, vh := range viewportHeights {
		t.Run(vh.name, func(t *testing.T) {
			s := newSelectableList(n, image.Pt(viewW, vh.h))
			s.frame()
			for _, target := range targets {
				s.state.Reveal(target)
				s.frame()
				if !s.sawRow(target) {
					t.Fatalf("after Reveal(%d) in a %d px viewport the row was not laid out (laid out %v)", target, vh.h, s.laidOut)
				}
				if !s.fullyVisible(t, target) {
					t.Fatalf("after Reveal(%d) in a %d px viewport of %d px rows the row is drawn clipped; Reveal must land it whole",
						target, vh.h, rowPx)
				}
				// A second Reveal of a row already whole in view must not move
				// the list at all — the "already visible" half of the same test,
				// and the half the fix must not break: widening the window test
				// by one row would scroll rows that were fine where they were.
				before := s.capture(t)
				s.state.Reveal(target)
				s.frame()
				if n := golden.PixelDiff(before, s.capture(t)); n != 0 {
					t.Fatalf("re-revealing row %d, already whole in view, moved the list: %d pixel(s) changed", target, n)
				}
			}
		})
	}
}

// TestSelectionIgnoresKeysWithoutFocus guards the other half of the focus
// contract: the list moves nothing while the keyboard is somewhere else. The
// decoy is a widget.Clickable laid out beside the list and given focus, so the
// router has a real alternative to route to.
func TestSelectionIgnoresKeysWithoutFocus(t *testing.T) {
	state := list.NewState()
	items := makeItems(20)
	r := new(gioinput.Router)
	ops := new(op.Ops)
	var decoy widget.Clickable

	frame := func() {
		ops.Reset()
		gtx := layout.Context{
			Metric:      unit.Metric{PxPerDp: 1, PxPerSp: 1},
			Constraints: layout.Exact(image.Pt(viewW, viewHPartial)),
			Ops:         ops,
			Source:      r.Source(),
		}
		decoy.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			size := image.Pt(viewW, 10)
			paint.FillShape(gtx.Ops, rowColor(0), clip.Rect{Max: size}.Op())
			return layout.Dimensions{Size: size}
		})
		list.LayoutSelectable(gtx, state, items, func(gtx layout.Context, item int, _ bool) layout.Dimensions {
			return colorRowFn(gtx, item)
		})
		r.Frame(ops)
	}

	frame()
	// The decoy is registered first, so FocusForward lands on it.
	r.MoveFocus(key.FocusForward)
	frame()

	r.Queue(key.Event{Name: key.NameDownArrow, State: key.Press})
	frame()
	r.Queue(key.Event{Name: key.NameEnd, State: key.Press})
	frame()

	if got := state.Selected(); got != -1 {
		t.Fatalf("Selected() = %d after keys sent while another widget held focus; want -1", got)
	}
}

// TestSelectionDroppedWhenItemsShrink pins the documented behaviour: a
// selection the item slice no longer carries is dropped, not clamped to the
// last row.
//
// The case that decides it is a filtered list narrowing from 100 rows to 3.
// Clamping answers 2 — a valid-looking index the user never chose, which a
// caller driving a detail pane off Selected() cannot tell from a real
// selection. Dropping answers -1, which every caller already handles, and it
// keeps the stale index from reappearing as an unrelated row when the filter is
// cleared again.
func TestSelectionDroppedWhenItemsShrink(t *testing.T) {
	newGtx := func(ops *op.Ops) layout.Context {
		return layout.Context{
			Metric:      unit.Metric{PxPerDp: 1, PxPerSp: 1},
			Constraints: layout.Exact(image.Pt(viewW, viewHPartial)),
			Ops:         ops,
		}
	}
	layoutN := func(state *list.State, n int) {
		var ops op.Ops
		list.LayoutSelectable(newGtx(&ops), state, makeItems(n), func(gtx layout.Context, item int, _ bool) layout.Dimensions {
			return colorRowFn(gtx, item)
		})
	}

	state := list.NewState()
	state.Select(30)
	if got := state.Selected(); got != 30 {
		t.Fatalf("Selected() = %d immediately after Select(30); want 30 (the item count is not known until layout)", got)
	}

	layoutN(state, 10)
	if got := state.Selected(); got != -1 {
		t.Fatalf("Selected() = %d after laying out 10 items; want -1 (row 30 does not exist, so nothing is selected)", got)
	}

	// The filter case, end to end: a selection deep in a long list, then the
	// list narrows, then it widens again. The old index must not come back.
	state.Select(87)
	layoutN(state, 100)
	if got := state.Selected(); got != 87 {
		t.Fatalf("Selected() = %d with 100 items; want 87 (an index the slice carries)", got)
	}
	layoutN(state, 3)
	if got := state.Selected(); got != -1 {
		t.Fatalf("Selected() = %d after the list narrowed to 3 items; want -1, not a row the user never chose", got)
	}
	layoutN(state, 100)
	if got := state.Selected(); got != -1 {
		t.Fatalf("Selected() = %d after the list widened back to 100 items; want -1 (the dropped selection must not reappear)", got)
	}

	state.Select(-1)
	if got := state.Selected(); got != -1 {
		t.Fatalf("Selected() = %d after Select(-1); want -1", got)
	}
}

// TestRevealSurvivesShrinkingToTheLastRow is the other half of that decision:
// the pending Reveal target is clamped where the selection is dropped. "Move
// the viewport to that row" still has a nearest sensible answer once the named
// row is gone; "the user chose that row" does not.
func TestRevealSurvivesShrinkingToTheLastRow(t *testing.T) {
	s := newSelectableList(100, image.Pt(viewW, viewHPartial))
	s.frame()
	s.state.Reveal(87)
	s.items = makeItems(20)
	s.frame()
	if !s.sawRow(19) {
		t.Fatalf("Reveal(87) against a list that shrank to 20 items laid out %v; want the last row scrolled into view", s.laidOut)
	}
	if got := s.state.Selected(); got != -1 {
		t.Fatalf("Selected() = %d; want -1 (Reveal never selects)", got)
	}
}

// TestRevealScrollsWithoutSelecting keeps the two primitives orthogonal:
// Reveal moves the viewport and leaves the selection alone.
func TestRevealScrollsWithoutSelecting(t *testing.T) {
	s := newSelectableList(200, image.Pt(viewW, viewHPartial))
	s.frame()
	if s.sawRow(150) {
		t.Fatal("row 150 was visible before any Reveal")
	}
	s.state.Reveal(150)
	s.frame()
	if !s.sawRow(150) {
		t.Fatalf("after Reveal(150) the row was still not laid out (laid out %v)", s.laidOut)
	}
	if got := s.state.Selected(); got != -1 {
		t.Fatalf("Reveal changed the selection to %d; want -1 (unchanged)", got)
	}
}

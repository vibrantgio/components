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

	"github.com/vibrantgio/prism/list"
)

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

// TestSelectionReachesRowNeverLaidOut is the test the whole task exists for.
//
// A 1000-item list in a 150 px viewport lays out five rows and a look-ahead;
// rows 6..999 have no focus tag, no clip area and no existence in the op tree,
// so no amount of Tab reaches them. End must select row 999 anyway, and the
// list must then scroll it into view so it is laid out — which is the proof
// that the selection is an index over the whole slice and not a walk over the
// tags that happen to exist.
func TestSelectionReachesRowNeverLaidOut(t *testing.T) {
	const n = 1000
	s := newSelectableList(n, image.Pt(viewW, viewH))
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
	s := newSelectableList(n, image.Pt(viewW, viewH))
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
	fresh := newSelectableList(n, image.Pt(viewW, viewH))
	fresh.focus(t)
	fresh.press(key.NameUpArrow)
	if got := fresh.state.Selected(); got != n-1 {
		t.Fatalf("Up with nothing selected chose %d; want %d (the last row)", got, n-1)
	}
}

// TestSelectionScrollsIntoViewOneRowAtATime checks the viewport follows the
// selection by the smallest move that works: stepping Down past the trailing
// edge scrolls one row, not one page, and every intermediate selection is
// actually laid out. A list that jumped the selection to the top of the
// viewport on every step would pass "the row is visible" and still be useless.
func TestSelectionScrollsIntoViewOneRowAtATime(t *testing.T) {
	const n = 40
	s := newSelectableList(n, image.Pt(viewW, viewH))
	s.focus(t)

	prevFirst := -1
	for i := 0; i < 12; i++ {
		s.press(key.NameDownArrow)
		if !s.sawRow(i) {
			t.Fatalf("after selecting row %d it was not laid out (laid out %v)", i, s.laidOut)
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
		t.Fatal("twelve Down presses never scrolled the list; the viewport holds five rows")
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
			Constraints: layout.Exact(image.Pt(viewW, viewH)),
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

// TestSelectClampsToItemCount pins the documented clamp: a selection past the
// end of a shrunken slice is dropped rather than left to reappear when the
// slice grows back.
func TestSelectClampsToItemCount(t *testing.T) {
	state := list.NewState()
	state.Select(30)
	if got := state.Selected(); got != 30 {
		t.Fatalf("Selected() = %d immediately after Select(30); want 30 (the clamp is layout's job)", got)
	}

	var ops op.Ops
	gtx := layout.Context{
		Metric:      unit.Metric{PxPerDp: 1, PxPerSp: 1},
		Constraints: layout.Exact(image.Pt(viewW, viewH)),
		Ops:         &ops,
	}
	list.LayoutSelectable(gtx, state, makeItems(10), func(gtx layout.Context, item int, _ bool) layout.Dimensions {
		return colorRowFn(gtx, item)
	})
	if got := state.Selected(); got != 9 {
		t.Fatalf("Selected() = %d after laying out 10 items; want 9 (clamped)", got)
	}

	state.Select(-1)
	if got := state.Selected(); got != -1 {
		t.Fatalf("Selected() = %d after Select(-1); want -1", got)
	}
}

// TestRevealScrollsWithoutSelecting keeps the two primitives orthogonal:
// Reveal moves the viewport and leaves the selection alone.
func TestRevealScrollsWithoutSelecting(t *testing.T) {
	s := newSelectableList(200, image.Pt(viewW, viewH))
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

package list_test

import (
	"image"
	"testing"

	gioinput "gioui.org/io/input"
	"gioui.org/io/key"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"

	golden "github.com/vibrantgio/components/golden"
	"github.com/vibrantgio/components/list"
	"github.com/vibrantgio/components/scrollbar"
	"github.com/vibrantgio/theme/tokens"
)

// barredList is selectableList's counterpart for LayoutSelectableScrollbar:
// the same list driven through a real router, laid out with a bar on its
// trailing edge. The question these tests ask is whether adding the bar cost
// the keyboard anything — the two halves are laid out by one call, and a
// caller reaching for it is reaching for both.
type barredList struct {
	state  *list.State
	items  []int
	bar    scrollbar.Style
	anchor list.Anchor
	r      *gioinput.Router
	ops    *op.Ops
	size   image.Point

	laidOut      []int
	selectedRows []int
}

func newBarredList(n int, size image.Point, anchor list.Anchor) *barredList {
	return &barredList{
		state:  list.NewState(),
		items:  makeItems(n),
		bar:    scrollbar.FromTokens(tokens.DefaultLight),
		anchor: anchor,
		r:      new(gioinput.Router),
		ops:    new(op.Ops),
		size:   size,
	}
}

func (s *barredList) frame() {
	s.ops.Reset()
	s.laidOut = s.laidOut[:0]
	s.selectedRows = s.selectedRows[:0]
	gtx := layout.Context{
		Metric:      unit.Metric{PxPerDp: 1, PxPerSp: 1},
		Constraints: layout.Exact(s.size),
		Ops:         s.ops,
		Source:      s.r.Source(),
	}
	list.LayoutSelectableScrollbar(gtx, s.state, s.bar, s.anchor, s.items,
		func(gtx layout.Context, item int, selected bool) layout.Dimensions {
			s.laidOut = append(s.laidOut, item)
			if selected {
				s.selectedRows = append(s.selectedRows, item)
			}
			return colorRowFn(gtx, item)
		})
	s.r.Frame(s.ops)
}

func (s *barredList) focus(t *testing.T) {
	t.Helper()
	s.frame()
	s.r.MoveFocus(key.FocusForward)
	s.frame()
}

func (s *barredList) press(name key.Name) {
	s.r.Queue(key.Event{Name: name, State: key.Press})
	s.frame()
}

func (s *barredList) sawRow(i int) bool {
	for _, v := range s.laidOut {
		if v == i {
			return true
		}
	}
	return false
}

// TestSelectableScrollbarKeepsTheKeyboard is the reason the entry point
// exists: a list that shows its scroll position must still be traversable
// over every row, including the ones virtualisation never laid out.
func TestSelectableScrollbarKeepsTheKeyboard(t *testing.T) {
	for _, anchor := range []struct {
		name string
		a    list.Anchor
	}{{"occupy", list.Occupy}, {"overlay", list.Overlay}} {
		t.Run(anchor.name, func(t *testing.T) {
			const n = 200
			s := newBarredList(n, image.Pt(viewW, viewHPartial), anchor.a)
			s.focus(t)
			if got := s.state.Selected(); got != -1 {
				t.Fatalf("Selected() before any key = %d; want -1", got)
			}
			last := n - 1
			if s.sawRow(last) {
				t.Fatalf("row %d was laid out on the first frame; the list is not virtualising", last)
			}

			s.press(key.NameEnd)
			if got := s.state.Selected(); got != last {
				t.Fatalf("after End, Selected() = %d; want %d", got, last)
			}
			if !s.sawRow(last) {
				t.Fatalf("after End, row %d was not laid out (laid out %v)", last, s.laidOut)
			}
			if len(s.selectedRows) != 1 || s.selectedRows[0] != last {
				t.Errorf("rowFn reported selected rows %v; want exactly [%d]", s.selectedRows, last)
			}

			s.press(key.NameHome)
			if got := s.state.Selected(); got != 0 {
				t.Fatalf("after Home, Selected() = %d; want 0", got)
			}
			s.press(key.NameDownArrow)
			if got := s.state.Selected(); got != 1 {
				t.Fatalf("after Down from the first row, Selected() = %d; want 1", got)
			}
		})
	}
}

// TestSelectableScrollbarMovesTheBarWithTheSelection is the other half: the
// keyboard's own Reveal moves the viewport, and the bar reads that position
// rather than one of its own. The thumb's pixels are the only honest witness —
// a bar wired to a stale position draws in the same place every frame.
func TestSelectableScrollbarMovesTheBarWithTheSelection(t *testing.T) {
	const n = 200
	s := newBarredList(n, image.Pt(viewW, viewHPartial), list.Overlay)
	s.focus(t)

	shot := func() *image.RGBA {
		return golden.Capture(t, s.size, func(gtx layout.Context) layout.Dimensions {
			return list.LayoutSelectableScrollbar(gtx, s.state, s.bar, s.anchor, s.items,
				func(gtx layout.Context, item int, _ bool) layout.Dimensions {
					return colorRowFn(gtx, item)
				})
		})
	}
	top := shot()
	s.press(key.NameEnd)
	end := shot()
	if golden.PixelDiff(top, end) == 0 {
		t.Error("the bar drew identically at the top of the list and at its end")
	}
}

// TestSelectableScrollbarDrawsNothingExtraWhenEverythingFits pins the bar's
// own rule through the new entry point: a list that does not scroll shows no
// indicator at all, and an Overlay bar that draws nothing costs the rows
// nothing either.
func TestSelectableScrollbarDrawsNothingExtraWhenEverythingFits(t *testing.T) {
	size := image.Pt(viewW, 300) // three 30 px rows in a 300 px viewport
	bar := scrollbar.FromTokens(tokens.DefaultLight)
	rowFn := func(gtx layout.Context, item int, _ bool) layout.Dimensions {
		return colorRowFn(gtx, item)
	}

	var plainDims layout.Dimensions
	plain := golden.Capture(t, size, func(gtx layout.Context) layout.Dimensions {
		plainDims = list.LayoutSelectable(gtx, list.NewState(), makeItems(3), rowFn)
		return plainDims
	})
	var barredDims layout.Dimensions
	barred := golden.Capture(t, size, func(gtx layout.Context) layout.Dimensions {
		barredDims = list.LayoutSelectableScrollbar(gtx, list.NewState(), bar, list.Overlay, makeItems(3), rowFn)
		return barredDims
	})
	if barredDims != plainDims {
		t.Errorf("Overlay dims = %v; want %v (same as LayoutSelectable)", barredDims, plainDims)
	}
	if n := golden.PixelDiff(plain, barred); n != 0 {
		t.Errorf("a list that fits drew %d pixel(s) of bar; want none", n)
	}
}

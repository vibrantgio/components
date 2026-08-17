package list_test

import (
	"image"
	"testing"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"

	"github.com/vibrantgio/components/list"
)

// movingList is the fixture for the pixel-movement tests: a plain list.Layout
// over rows of known, deliberately unequal heights, laid out into a discarded
// op list. The question every test here asks is "where is the leading edge
// now", in pixels from the top of the content — which is what a reader paging
// a document experiences, and what a row index cannot express once rows differ
// in height.
type movingList struct {
	state   *list.State
	heights []int
	ops     *op.Ops
	size    image.Point
}

func newMovingList(heights []int, size image.Point) *movingList {
	return &movingList{
		state:   list.NewState(),
		heights: heights,
		ops:     new(op.Ops),
		size:    size,
	}
}

func (m *movingList) frame() {
	m.ops.Reset()
	gtx := layout.Context{
		Metric:      unit.Metric{PxPerDp: 1, PxPerSp: 1},
		Constraints: layout.Exact(m.size),
		Ops:         m.ops,
	}
	list.Layout(gtx, m.state, m.heights, func(gtx layout.Context, h int) layout.Dimensions {
		return layout.Dimensions{Size: image.Pt(gtx.Constraints.Max.X, h)}
	})
}

// top is the absolute scroll offset in pixels: the height of everything above
// the leading row plus how far into that row the viewport's leading edge sits.
func (m *movingList) top() int {
	p := m.state.Position()
	px := 0
	for i := 0; i < p.First && i < len(m.heights); i++ {
		px += m.heights[i]
	}
	return px + p.Offset
}

// content is the total height of every row.
func (m *movingList) content() int {
	px := 0
	for _, h := range m.heights {
		px += h
	}
	return px
}

// atEnd reports whether the viewport's trailing edge rests on the content's.
func (m *movingList) atEnd() bool {
	p := m.state.Position()
	return p.First+p.Count == len(m.heights) && p.OffsetLast >= 0
}

func evenHeights(n, h int) []int {
	s := make([]int, n)
	for i := range s {
		s[i] = h
	}
	return s
}

// TestViewportIsRecorded checks the pixel height a page is measured against is
// the one the list actually laid out in, and that it is zero before any layout
// — a caller asking for a page before the first frame must be able to tell.
func TestViewportIsRecorded(t *testing.T) {
	m := newMovingList(evenHeights(20, rowPx), image.Pt(viewW, viewH))
	if got := m.state.Viewport(); got != 0 {
		t.Fatalf("viewport before first layout = %d, want 0", got)
	}
	m.frame()
	if got := m.state.Viewport(); got != viewH {
		t.Fatalf("viewport = %d, want %d", got, viewH)
	}
}

// TestScrollPixelsMovesByPixels walks a page down and back up over rows of
// unequal height, where a row count would not name the same distance twice.
func TestScrollPixelsMovesByPixels(t *testing.T) {
	// Heights a row-average would get badly wrong: a run of short rows and
	// one very tall one, exactly the shape of a document with a long code
	// fence in it.
	heights := []int{20, 20, 20, 400, 20, 20, 20, 20, 20, 300, 20, 20}
	m := newMovingList(heights, image.Pt(viewW, viewH))
	m.frame()
	if got := m.top(); got != 0 {
		t.Fatalf("initial top = %d, want 0", got)
	}
	page := viewH - 20 // a viewport less a line of overlap
	for i, want := range []int{page, 2 * page, 3 * page} {
		m.state.ScrollPixels(page)
		m.frame()
		if got := m.top(); got != want {
			t.Fatalf("after %d page(s) down: top = %d, want %d", i+1, got, want)
		}
	}
	for i, want := range []int{2 * page, page, 0} {
		m.state.ScrollPixels(-page)
		m.frame()
		if got := m.top(); got != want {
			t.Fatalf("after %d page(s) back up: top = %d, want %d", i+1, got, want)
		}
	}
}

// TestScrollPixelsIsBoundedAtBothEnds is the pair of transitions a reader hits
// by holding a page key down: asking for more than there is must land on the
// end, never past it.
func TestScrollPixelsIsBoundedAtBothEnds(t *testing.T) {
	heights := []int{20, 20, 400, 20, 20, 20, 300, 20}
	m := newMovingList(heights, image.Pt(viewW, viewH))
	m.frame()

	for i := 0; i < 40; i++ {
		m.state.ScrollPixels(viewH - 20)
		m.frame()
	}
	if !m.atEnd() {
		t.Fatalf("after paging past the end: position %+v is not the end", m.state.Position())
	}
	if got, want := m.top(), m.content()-viewH; got != want {
		t.Fatalf("top at the end = %d, want %d", got, want)
	}

	for i := 0; i < 40; i++ {
		m.state.ScrollPixels(-(viewH - 20))
		m.frame()
	}
	if got := m.top(); got != 0 {
		t.Fatalf("top after paging past the start = %d, want 0", got)
	}
	if p := m.state.Position(); p.First != 0 || p.Offset != 0 {
		t.Fatalf("position at the start = %+v, want First 0 Offset 0", p)
	}
}

// TestScrollToEndAndStart drives the two ends directly, from the middle, in
// both orders.
func TestScrollToEndAndStart(t *testing.T) {
	heights := []int{20, 20, 400, 20, 20, 20, 300, 20, 40}
	m := newMovingList(heights, image.Pt(viewW, viewH))
	m.frame()
	m.state.ScrollPixels(200)
	m.frame()
	if m.top() != 200 {
		t.Fatalf("mid-document top = %d, want 200", m.top())
	}

	m.state.ScrollToEnd(len(heights))
	m.frame()
	if !m.atEnd() {
		t.Fatalf("ScrollToEnd left position %+v", m.state.Position())
	}
	if got, want := m.top(), m.content()-viewH; got != want {
		t.Fatalf("top after ScrollToEnd = %d, want %d", got, want)
	}

	m.state.ScrollToStart()
	m.frame()
	if got := m.top(); got != 0 {
		t.Fatalf("top after ScrollToStart = %d, want 0", got)
	}
}

// TestScrollToEndWithATallLastRow is the case a naive "put the last row at the
// leading edge" implementation gets wrong: the last row is taller than the
// viewport, so the document's end is inside it, not at its top.
func TestScrollToEndWithATallLastRow(t *testing.T) {
	heights := []int{40, 40, 40, 600}
	m := newMovingList(heights, image.Pt(viewW, viewH))
	m.frame()
	m.state.ScrollToEnd(len(heights))
	m.frame()
	if got, want := m.top(), m.content()-viewH; got != want {
		t.Fatalf("top = %d, want %d (the end of the tall last row)", got, want)
	}
}

// TestShorterThanTheViewportDoesNotMove is the whole-document-fits case: every
// movement is a no-op, because there is nowhere to go.
func TestShorterThanTheViewportDoesNotMove(t *testing.T) {
	heights := []int{20, 30, 20} // 70 px of content in a 150 px viewport
	for _, tc := range []struct {
		name string
		move func(m *movingList)
	}{
		{"page down", func(m *movingList) { m.state.ScrollPixels(viewH - 20) }},
		{"page up", func(m *movingList) { m.state.ScrollPixels(-(viewH - 20)) }},
		{"to the end", func(m *movingList) { m.state.ScrollToEnd(len(heights)) }},
		{"to the start", func(m *movingList) { m.state.ScrollToStart() }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := newMovingList(heights, image.Pt(viewW, viewH))
			m.frame()
			tc.move(m)
			m.frame()
			if p := m.state.Position(); p.First != 0 || p.Offset != 0 || m.top() != 0 {
				t.Fatalf("position = %+v, top = %d; want the document unmoved", p, m.top())
			}
		})
	}
}

// TestScrollPixelsFeedsTheScrollbarFractions checks the movement reaches the
// same numbers the scroll indicator reads, so a keyboard move wakes the bar
// exactly as a wheel scroll does.
func TestScrollPixelsFeedsTheScrollbarFractions(t *testing.T) {
	m := newMovingList(evenHeights(40, rowPx), image.Pt(viewW, viewH))
	m.frame()
	before := m.state.Position()
	m.state.ScrollPixels(viewH - 20)
	m.frame()
	after := m.state.Position()
	if before.First == after.First && before.Offset == after.Offset {
		t.Fatalf("position unchanged after a page down: %+v", after)
	}
	if after.Length <= 0 {
		t.Fatalf("Length = %d; the indicator cannot derive a proportion from it", after.Length)
	}
}

// TestScrollToPutsTheRowAtTheTop is the movement an outline entry makes: a row
// named by index goes to the leading edge, from either side of the viewport.
// Rows of unequal height are the point — the destination is a row, and the
// distance covered is whatever those rows happen to add up to.
func TestScrollToPutsTheRowAtTheTop(t *testing.T) {
	heights := []int{20, 40, 200, 30, 90, 250, 30, 60, 40, 30}
	m := newMovingList(heights, image.Pt(viewW, viewH))
	m.frame()

	top := func(i int) int {
		px := 0
		for j := 0; j < i; j++ {
			px += heights[j]
		}
		return px
	}
	// Down to a row below the fold, then back up to one above it: both
	// land the named row on the viewport's leading edge exactly.
	for _, i := range []int{5, 2, 4, 1} {
		m.state.ScrollTo(i)
		m.frame()
		if p := m.state.Position(); p.First != i || p.Offset != 0 {
			t.Fatalf("ScrollTo(%d): position = %+v, want First %d Offset 0", i, p, i)
		}
		if got, want := m.top(), top(i); got != want {
			t.Fatalf("ScrollTo(%d): top = %d, want %d", i, got, want)
		}
	}
}

// TestScrollToIsBoundedAndLeavesTheSelectionAlone covers the two things a
// caller may not have checked for itself: an index past the end lands on the
// content's end rather than past it, a negative index on the start, and neither
// disturbs which row is selected.
func TestScrollToIsBoundedAndLeavesTheSelectionAlone(t *testing.T) {
	heights := []int{20, 40, 200, 30, 90, 250, 30, 60}
	m := newMovingList(heights, image.Pt(viewW, viewH))
	m.state.Select(3)
	m.frame()

	m.state.ScrollTo(len(heights) + 50)
	m.frame()
	if !m.atEnd() {
		t.Fatalf("ScrollTo past the last row left position %+v, not the end", m.state.Position())
	}
	if got, want := m.top(), m.content()-viewH; got != want {
		t.Fatalf("top after ScrollTo past the end = %d, want %d", got, want)
	}

	m.state.ScrollTo(-4)
	m.frame()
	if got := m.top(); got != 0 {
		t.Fatalf("top after ScrollTo(-4) = %d, want 0", got)
	}
	if got := m.state.Selected(); got != 3 {
		t.Fatalf("selection = %d after scrolling, want 3 — scrolling is not selecting", got)
	}
}

// TestScrollToOnAListThatFits is the short-document case: every row is already
// on screen, so naming one moves nothing.
func TestScrollToOnAListThatFits(t *testing.T) {
	heights := []int{20, 30, 20}
	m := newMovingList(heights, image.Pt(viewW, viewH))
	m.frame()
	m.state.ScrollTo(2)
	m.frame()
	if p := m.state.Position(); p.First != 0 || p.Offset != 0 || m.top() != 0 {
		t.Fatalf("position = %+v, top = %d; want the list unmoved", p, m.top())
	}
}

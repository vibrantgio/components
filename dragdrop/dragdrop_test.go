package dragdrop

import (
	"image"
	"image/color"
	"testing"
	"time"

	"gioui.org/f32"
	"gioui.org/gesture"
	"gioui.org/io/event"
	"gioui.org/io/input"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
)

// The test window has two sequences side by side. Each item is 100 by
// 20 with a gap of 4, so item i of a sequence is at y 24*i. Sequence 0
// is at x 0 and holds three items; sequence 1 is at x 200 and is empty.
const (
	itemW, itemH, gap = 100, 20, 4
	colX1             = 200
)

type harness struct {
	t       *testing.T
	r       input.Router
	s       State
	n       [2]int
	clicks  [2][]gesture.Click
	clicked []Place
	moves   []Move
	scroll  int
	now     time.Time
}

func newHarness(t *testing.T) *harness {
	h := &harness{t: t, n: [2]int{3, 0}, now: time.Unix(0, 0)}
	h.clicks[0] = make([]gesture.Click, 3)
	h.frame()
	return h
}

// blocking draws the item's image and registers an event area over it that blocks
// the areas under it, as an app's own item does.
func blocking(gtx layout.Context, from Place) layout.Dimensions {
	paint.FillShape(gtx.Ops, color.NRGBA{A: 255}, clip.Rect{Max: gtx.Constraints.Min}.Op())
	a := clip.Rect{Max: gtx.Constraints.Min}.Push(gtx.Ops)
	event.Op(gtx.Ops, &blockTag)
	a.Pop()
	return layout.Dimensions{Size: gtx.Constraints.Min}
}

var blockTag int

func (h *harness) frame() {
	ops := new(op.Ops)
	h.now = h.now.Add(time.Second / 60)
	gtx := layout.Context{
		Ops:         ops,
		Constraints: layout.Exact(image.Pt(400, 300)),
		Metric:      unit.Metric{PxPerDp: 1, PxPerSp: 1},
		Source:      h.r.Source(),
		Now:         h.now,
	}
	for {
		if _, ok := gtx.Event(pointer.Filter{Target: &blockTag, Kinds: pointer.Press}); !ok {
			break
		}
	}
	h.s.Layout(gtx, Style{Line: color.NRGBA{B: 255, A: 255}}, blocking, func(gtx layout.Context) layout.Dimensions {
		for c := range 2 {
			st := op.Offset(image.Pt(c*colX1, 0)).Push(gtx.Ops)
			cg := gtx
			cg.Constraints = layout.Exact(image.Pt(itemW, 300))
			h.scroll += h.s.Scroll(cg, c)
			for i := range h.n[c] {
				it := op.Offset(image.Pt(0, i*(itemH+gap))).Push(gtx.Ops)
				ig := cg
				ig.Constraints = layout.Exact(image.Pt(itemW, itemH))
				h.s.Item(ig, Place{c, i}, gap, func(gtx layout.Context) layout.Dimensions {
					cl := &h.clicks[c][i]
					for {
						e, ok := cl.Update(gtx.Source)
						if !ok {
							break
						}
						if e.Kind == gesture.KindClick {
							h.clicked = append(h.clicked, Place{c, i})
						}
					}
					a := clip.Rect{Max: gtx.Constraints.Min}.Push(gtx.Ops)
					cl.Add(gtx.Ops)
					a.Pop()
					return layout.Dimensions{Size: gtx.Constraints.Min}
				})
				it.Pop()
			}
			top := h.n[c] * (itemH + gap)
			et := op.Offset(image.Pt(0, top-gap)).Push(gtx.Ops)
			if h.n[c] == 0 {
				et.Pop()
				et = op.Offset(image.Pt(0, 0)).Push(gtx.Ops)
			}
			eg := cg
			eg.Constraints = layout.Exact(image.Pt(itemW, 300-top))
			h.s.End(eg, c, h.n[c], gap)
			et.Pop()
			h.s.Edges(cg, c, image.Pt(itemW, 300))
			st.Pop()
		}
		return layout.Dimensions{Size: gtx.Constraints.Max}
	})
	h.r.Frame(ops)
	h.moves = append(h.moves, h.s.Update()...)
}

func (h *harness) pointer(kind pointer.Kind, x, y float32) {
	e := pointer.Event{Kind: kind, Source: pointer.Mouse, Position: f32.Pt(x, y)}
	if kind == pointer.Press || kind == pointer.Drag {
		e.Buttons = pointer.ButtonPrimary
	}
	if kind == pointer.Drag {
		e.Kind = pointer.Move
	}
	h.r.Queue(e)
	h.frame()
}

// drag presses at (x0, y0), moves in steps to (x1, y1) and releases
// there.
func (h *harness) drag(x0, y0, x1, y1 float32) {
	h.pointer(pointer.Move, x0, y0)
	h.pointer(pointer.Press, x0, y0)
	const steps = 4
	for i := 1; i <= steps; i++ {
		f := float32(i) / steps
		h.pointer(pointer.Drag, x0+(x1-x0)*f, y0+(y1-y0)*f)
	}
	h.pointer(pointer.Release, x1, y1)
}

func (h *harness) wantMoves(want ...Move) {
	h.t.Helper()
	if len(h.moves) != len(want) {
		h.t.Fatalf("moves = %v, want %v", h.moves, want)
	}
	for i := range want {
		if h.moves[i] != want[i] {
			h.t.Fatalf("moves = %v, want %v", h.moves, want)
		}
	}
}

func TestPressWithoutMoveIsAClick(t *testing.T) {
	h := newHarness(t)
	h.pointer(pointer.Move, 50, 30)
	h.pointer(pointer.Press, 50, 30)
	h.pointer(pointer.Drag, 51, 31)
	h.pointer(pointer.Release, 51, 31)
	h.wantMoves()
	if len(h.clicked) != 1 || h.clicked[0] != (Place{0, 1}) {
		t.Fatalf("clicked = %v, want the item at 0,1", h.clicked)
	}
}

func TestDragDownInItsOwnSequence(t *testing.T) {
	h := newHarness(t)
	// From item 0 to the trailing half of item 2: the gap after it.
	h.drag(50, 10, 50, 60)
	want := Move{From: Place{0, 0}, To: Place{0, 3}}
	h.wantMoves(want)
	if got := want.Index(); got != 2 {
		t.Fatalf("Index() = %d, want 2", got)
	}
	if len(h.clicked) != 0 {
		t.Fatalf("clicked = %v after a drag, want none", h.clicked)
	}
}

func TestDragUpInItsOwnSequence(t *testing.T) {
	h := newHarness(t)
	// From item 2 to the leading half of item 0.
	h.drag(50, 58, 50, 4)
	h.wantMoves(Move{From: Place{0, 2}, To: Place{0, 0}})
}

func TestDropNextToItselfChangesNothing(t *testing.T) {
	h := newHarness(t)
	// From item 1 to the trailing half of item 0: the gap before item 1.
	h.drag(50, 30, 50, 15)
	h.wantMoves()
}

func TestDropOnTheEndOfASequence(t *testing.T) {
	h := newHarness(t)
	// From item 0 to the space after item 2.
	h.drag(50, 10, 50, 150)
	h.wantMoves(Move{From: Place{0, 0}, To: Place{0, 3}})
}

func TestDropOnAnEmptySequence(t *testing.T) {
	h := newHarness(t)
	h.drag(50, 30, colX1+50, 100)
	want := Move{From: Place{0, 1}, To: Place{1, 0}}
	h.wantMoves(want)
	if got := want.Index(); got != 0 {
		t.Fatalf("Index() = %d, want 0", got)
	}
}

func TestDropOnAnItemOfAnotherSequence(t *testing.T) {
	h := newHarness(t)
	h.n[1] = 2
	h.clicks[1] = make([]gesture.Click, 2)
	h.frame()
	// From item 2 of sequence 0 to the leading half of item 1 of
	// sequence 1.
	h.drag(50, 58, colX1+50, 26)
	h.wantMoves(Move{From: Place{0, 2}, To: Place{1, 1}})
}

func TestReleaseOverNoPlaceChangesNothing(t *testing.T) {
	h := newHarness(t)
	h.drag(50, 10, 150, 100)
	h.wantMoves()
}

func TestEscapeCancelsTheDrag(t *testing.T) {
	h := newHarness(t)
	h.pointer(pointer.Move, 50, 10)
	h.pointer(pointer.Press, 50, 10)
	h.pointer(pointer.Drag, 50, 30)
	h.pointer(pointer.Drag, 50, 60)
	if p, ok := h.s.Target(); !ok || p != (Place{0, 3}) {
		t.Fatalf("Target() = %v, %v during the drag, want 0,3", p, ok)
	}
	h.r.Queue(key.Event{Name: key.NameEscape, State: key.Press})
	h.frame()
	if h.s.Dragging() {
		t.Fatal("Dragging() after Escape, want false")
	}
	h.pointer(pointer.Release, 50, 60)
	h.wantMoves()
}

func TestPointerNearTheBottomEdgeScrolls(t *testing.T) {
	h := newHarness(t)
	h.pointer(pointer.Move, 50, 10)
	h.pointer(pointer.Press, 50, 10)
	h.pointer(pointer.Drag, 50, 200)
	h.pointer(pointer.Drag, 50, 290)
	h.scroll = 0
	h.frame()
	if h.scroll <= 0 {
		t.Fatalf("scroll = %d near the bottom edge, want > 0", h.scroll)
	}
	h.pointer(pointer.Drag, 50, 150)
	h.scroll = 0
	h.frame()
	if h.scroll != 0 {
		t.Fatalf("scroll = %d away from the edges, want 0", h.scroll)
	}
	h.pointer(pointer.Release, 50, 150)
}

func TestReleaseOverTheItemAfterADragIsNoClick(t *testing.T) {
	h := newHarness(t)
	h.drag(50, 26, 50, 40)
	h.wantMoves()
	if len(h.clicked) != 0 {
		t.Fatalf("clicked = %v after a drag, want none", h.clicked)
	}
}

func TestContentAreasStillBlock(t *testing.T) {
	var r input.Router
	var s State
	var top, under int
	got := map[*int]int{}
	frame := func() {
		ops := new(op.Ops)
		gtx := layout.Context{Ops: ops, Constraints: layout.Exact(image.Pt(100, 100)), Source: r.Source()}
		for _, tg := range []*int{&top, &under} {
			for {
				e, ok := gtx.Event(pointer.Filter{Target: tg, Kinds: pointer.Press})
				if !ok {
					break
				}
				if e, ok := e.(pointer.Event); ok && e.Kind == pointer.Press {
					got[tg]++
				}
			}
		}
		s.Layout(gtx, Style{}, nil, func(gtx layout.Context) layout.Dimensions {
			for _, tg := range []*int{&under, &top} {
				a := clip.Rect{Max: image.Pt(50, 50)}.Push(gtx.Ops)
				event.Op(gtx.Ops, tg)
				a.Pop()
			}
			return layout.Dimensions{Size: gtx.Constraints.Max}
		})
		r.Frame(ops)
	}
	frame()
	r.Queue(pointer.Event{Kind: pointer.Press, Source: pointer.Mouse, Buttons: pointer.ButtonPrimary, Position: f32.Pt(10, 10)})
	frame()
	if got[&top] != 1 || got[&under] != 0 {
		t.Fatalf("presses: top %d, under %d; want 1 and 0", got[&top], got[&under])
	}
}

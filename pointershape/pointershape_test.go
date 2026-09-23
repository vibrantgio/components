package pointershape_test

import (
	"image"
	"testing"

	"gioui.org/f32"
	"gioui.org/io/event"
	gioinput "gioui.org/io/input"
	"gioui.org/io/pointer"
	"gioui.org/op"
	"gioui.org/op/clip"

	"github.com/vibrantgio/components/pointershape"
)

// probe is one window laid out through a router, so every reading is the
// shape the platform would be handed.
type probe struct {
	r     gioinput.Router
	ops   op.Ops
	frame func(ops *op.Ops)
}

func (p *probe) draw() {
	p.ops.Reset()
	p.frame(&p.ops)
	p.r.Frame(&p.ops)
}

// at moves the pointer to (x, y) and reports the shape the next frame
// declares there.
func (p *probe) at(x, y int) pointer.Cursor {
	p.r.Queue(pointer.Event{Kind: pointer.Move, Position: f32.Pt(float32(x), float32(y)), Source: pointer.Mouse})
	p.draw()
	return p.r.Cursor()
}

func newProbe(frame func(ops *op.Ops)) *probe {
	p := &probe{frame: frame}
	p.draw()
	p.draw()
	return p
}

// TestShapeStopsAtTheRegionsEdge is the defect this package closes. Gio's
// cursor op records its shape on the clip area enclosing it, and the router
// walks a hit area's ancestors for the first shape it meets, so a region that
// adds the op with no bound of its own hands its shape to everything inside
// the area around it. Declared through Over, the shape reaches the region's
// own edge and no further.
func TestShapeStopsAtTheRegionsEdge(t *testing.T) {
	var field, list int
	// A text field in the top half, a list under it, both inside one column.
	p := newProbe(func(ops *op.Ops) {
		column := clip.Rect(image.Rect(0, 0, 300, 300)).Push(ops)
		top := clip.Rect(image.Rect(0, 0, 300, 100)).Push(ops)
		event.Op(ops, &field)
		top.Pop()
		pointershape.OverSize(ops, image.Pt(300, 100), pointer.CursorText)
		bottom := op.Offset(image.Pt(0, 100)).Push(ops)
		rows := clip.Rect(image.Rect(0, 0, 300, 200)).Push(ops)
		event.Op(ops, &list)
		rows.Pop()
		bottom.Pop()
		column.Pop()
	})
	if got, want := p.at(150, 50), pointer.CursorText; got != want {
		t.Errorf("pointer over the field = %v; want %v", got, want)
	}
	if got, want := p.at(150, 200), pointer.CursorDefault; got != want {
		t.Errorf("pointer over the list beside the field = %v; want %v", got, want)
	}
}

// TestUnboundedShapeReachesTheRegionBeside records why the bound is needed:
// the same two regions with the shape added bare show the field's I-beam over
// the list. It is the reading the owner reported, and it must not come back.
func TestUnboundedShapeReachesTheRegionBeside(t *testing.T) {
	var field, list int
	p := newProbe(func(ops *op.Ops) {
		column := clip.Rect(image.Rect(0, 0, 300, 300)).Push(ops)
		top := clip.Rect(image.Rect(0, 0, 300, 100)).Push(ops)
		event.Op(ops, &field)
		top.Pop()
		pointer.CursorText.Add(ops) // no bound: the shape lands on the column
		bottom := op.Offset(image.Pt(0, 100)).Push(ops)
		rows := clip.Rect(image.Rect(0, 0, 300, 200)).Push(ops)
		event.Op(ops, &list)
		rows.Pop()
		bottom.Pop()
		column.Pop()
	})
	if got, want := p.at(150, 200), pointer.CursorText; got != want {
		t.Errorf("the defect no longer reproduces: unbounded shape over the list = %v; want %v", got, want)
	}
}

// TestTheDefaultNeedsNoDeclaring measures what the router does where no shape
// covers the pointer, and what declaring CursorDefault over a region is worth.
// The answer to both is the reason nothing in the library declares the
// default: the router reads CursorDefault wherever no bounded shape stands,
// and CursorDefault is the zero value of an area's shape, so declaring it
// shields a region from nothing. Bounding is the whole of the fix.
func TestTheDefaultNeedsNoDeclaring(t *testing.T) {
	var bare, shielded int
	p := newProbe(func(ops *op.Ops) {
		column := clip.Rect(image.Rect(0, 0, 300, 300)).Push(ops)
		pointer.CursorText.Add(ops) // unbounded, on the column
		top := clip.Rect(image.Rect(0, 0, 300, 100)).Push(ops)
		event.Op(ops, &bare)
		top.Pop()
		bottom := op.Offset(image.Pt(0, 100)).Push(ops)
		rows := clip.Rect(image.Rect(0, 0, 300, 200)).Push(ops)
		event.Op(ops, &shielded)
		pointer.CursorDefault.Add(ops)
		rows.Pop()
		bottom.Pop()
		column.Pop()
	})
	if got, want := p.at(150, 200), pointer.CursorText; got != want {
		t.Errorf("region declaring CursorDefault under an unbounded I-beam = %v; want %v (the default shields nothing)", got, want)
	}
	// Past every area in the frame there is nothing to declare a shape, and
	// the router reads the default there without being told.
	if got, want := p.at(400, 400), pointer.CursorDefault; got != want {
		t.Errorf("pointer past every area = %v; want %v", got, want)
	}
}

// TestShapeDeclaredAfterTheRegionsInput pins the ordering the package doc
// states: the router scans a frame's areas newest first and carries on from
// the parent of the input area it meets, so a shape declared before the
// region's own event.Op is stepped over.
func TestShapeDeclaredAfterTheRegionsInput(t *testing.T) {
	var link int
	after := newProbe(func(ops *op.Ops) {
		area := clip.Rect(image.Rect(0, 0, 100, 40)).Push(ops)
		event.Op(ops, &link)
		pointershape.OverSize(ops, image.Pt(100, 40), pointer.CursorPointer)
		area.Pop()
	})
	if got, want := after.at(50, 20), pointer.CursorPointer; got != want {
		t.Errorf("shape declared after the region's input = %v; want %v", got, want)
	}
	before := newProbe(func(ops *op.Ops) {
		area := clip.Rect(image.Rect(0, 0, 100, 40)).Push(ops)
		pointershape.OverSize(ops, image.Pt(100, 40), pointer.CursorPointer)
		event.Op(ops, &link)
		area.Pop()
	})
	if got, want := before.at(50, 20), pointer.CursorDefault; got != want {
		t.Errorf("shape declared before the region's input = %v; want %v", got, want)
	}
}

// TestShapeLastsOneFrame: a shape stands for the frame that declares it and no
// longer. The router recomputes what the pointer stands on every frame from
// its last position, so a region that stops declaring stops showing, with the
// pointer never moving.
func TestShapeLastsOneFrame(t *testing.T) {
	var tag int
	declare := true
	p := newProbe(func(ops *op.Ops) {
		area := clip.Rect(image.Rect(0, 0, 100, 40)).Push(ops)
		event.Op(ops, &tag)
		if declare {
			pointershape.OverSize(ops, image.Pt(100, 40), pointer.CursorText)
		}
		area.Pop()
	})
	if got, want := p.at(50, 20), pointer.CursorText; got != want {
		t.Fatalf("declared shape = %v; want %v", got, want)
	}
	declare = false
	p.draw()
	if got, want := p.r.Cursor(), pointer.CursorDefault; got != want {
		t.Errorf("shape after a frame that declares none, pointer unmoved = %v; want %v", got, want)
	}
}

// TestEmptyAreaDeclaresNothing: a region with no room to stand in declares no
// shape, so a collapsed control does not take the pointer's shape from what it
// covers nothing of.
func TestEmptyAreaDeclaresNothing(t *testing.T) {
	var tag int
	p := newProbe(func(ops *op.Ops) {
		area := clip.Rect(image.Rect(0, 0, 100, 40)).Push(ops)
		event.Op(ops, &tag)
		pointershape.OverSize(ops, image.Point{}, pointer.CursorText)
		area.Pop()
	})
	if got, want := p.at(50, 20), pointer.CursorDefault; got != want {
		t.Errorf("shape over an empty area = %v; want %v", got, want)
	}
}

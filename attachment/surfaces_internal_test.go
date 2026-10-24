package attachment

import (
	"image"
	"testing"

	"gioui.org/layout"
	"gioui.org/op"
)

// recordRoom adds a callback to the Surfaces set on gtx that appends the
// area it is called with, and name, to got.
func recordRoom(gtx layout.Context, name string, got *[]string, rooms *[]image.Rectangle) {
	s := surfacesFrom(gtx)
	s.draws = append(s.draws, func(_ layout.Context, room image.Rectangle) {
		*got = append(*got, name)
		*rooms = append(*rooms, room)
	})
}

// TestAvailableRoomAtMovesTheSurfaces sets a Surfaces, moves it to a child at an
// offset with AvailableRoomAt, records a callback in the child, and places the
// outer Surfaces: the callback is called with the area moved by the offset.
func TestAvailableRoomAtMovesTheSurfaces(t *testing.T) {
	gtx := layout.Context{Ops: new(op.Ops)}
	gtx, outer := WithSurfaces(gtx)
	var names []string
	var rooms []image.Rectangle
	child := AvailableRoomAt(gtx, image.Pt(30, 7))
	recordRoom(child, "child", &names, &rooms)
	PlaceSurfaces(layout.Context{Ops: gtx.Ops}, outer, image.Rect(0, 0, 100, 50))
	if len(rooms) != 1 || rooms[0] != image.Rect(-30, -7, 70, 43) {
		t.Fatalf("the callback was called with %v, want one call with (-30,-7)-(70,43)", rooms)
	}
	if AvailableRoomAt(layout.Context{}, image.Pt(1, 1)).Values != nil {
		t.Error("with nothing set, AvailableRoomAt set a value")
	}
}

// TestPlaceSurfacesUnderASetSurfacesAddsToIt places a Surfaces with a context
// that has another Surfaces set: nothing is called, the callbacks are added
// to the other in order, and they are called with the area it is placed in.
func TestPlaceSurfacesUnderASetSurfacesAddsToIt(t *testing.T) {
	gtx := layout.Context{Ops: new(op.Ops)}
	gtx, outer := WithSurfaces(gtx)
	igtx, inner := WithSurfaces(gtx)
	var names []string
	var rooms []image.Rectangle
	recordRoom(igtx, "first", &names, &rooms)
	recordRoom(igtx, "second", &names, &rooms)
	PlaceSurfaces(gtx, inner, image.Rect(0, 0, 1, 1))
	if len(names) != 0 {
		t.Fatalf("placed under a set Surfaces, %d callbacks were called, want none", len(names))
	}
	room := image.Rect(-5, -5, 200, 100)
	PlaceSurfaces(layout.Context{Ops: gtx.Ops}, outer, room)
	if len(names) != 2 || names[0] != "first" || names[1] != "second" {
		t.Fatalf("the callbacks were called as %v, want [first second]", names)
	}
	for _, r := range rooms {
		if r != room {
			t.Errorf("a callback was called with %v, want the outer area %v", r, room)
		}
	}
}

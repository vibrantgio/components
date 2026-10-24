package attachment

import (
	"image"
	"maps"

	"gioui.org/layout"
	"gioui.org/op"
)

// surfacesKey is the key a [Surfaces] is set under in layout.Context.Values.
const surfacesKey = "github.com/vibrantgio/components/attachment.Surfaces"

// Surfaces holds the floating surfaces laid out in one child of a layout that
// places the child only after measuring it: a row that centres what it holds,
// a centred box, a list row. Such a layout cannot pass the available room into
// the child, because the child's offset is decided after the child is laid
// out, and Gio offers no readback of the transform it then pushes. A floating
// surface laid out under a context with a Surfaces set therefore is not drawn
// then: its component records, with [Record], a callback that draws the
// surface for a given available room, and the layout calls the callbacks with
// [PlaceSurfaces] once it has placed the child.
//
// The type is named for what it holds: the Surface part of each floating
// surface laid out in the child, waiting to be placed.
//
// A callback draws in the coordinates the surface's anchor was laid out in.
// The layout therefore calls PlaceSurfaces inside the transform it placed the
// child at, and a child that moves a floating surface by an offset of its own
// sets a Surfaces of its own for it: with [AvailableRoomAt] where the offset
// is known before the surface is laid out, and with WithSurfaces and
// [PlaceSurfacesAt] where it is not.
type Surfaces struct {
	draws []func(gtx layout.Context, room image.Rectangle)
}

// WithSurfaces returns gtx with a new, empty Surfaces set and the available
// room removed, and that Surfaces. A layout calls it for each child it places
// only after measuring it, lays the child out with the returned context, and
// then calls [PlaceSurfacesAt] with the offset it placed the child at. The
// room is removed because it cannot be moved by an offset not yet decided: a
// surface in the child is placed by the Surfaces, in the room moved once the
// offset is known. gtx.Values is cloned rather than written, so the caller's
// context keeps what it had.
//
// It is named as the context setters of the layouts above it are: it returns
// the context with a value set.
func WithSurfaces(gtx layout.Context) (layout.Context, *Surfaces) {
	s := new(Surfaces)
	values := maps.Clone(gtx.Values)
	if values == nil {
		values = make(map[string]any, 1)
	}
	values[surfacesKey] = s
	delete(values, availableRoomKey)
	gtx.Values = values
	return gtx, s
}

// PlaceSurfaces calls every callback s recorded, in the order the surfaces
// were laid out, each drawing its surface in room, the available room in the
// child's coordinates: the layout's own area moved by the offset it placed
// the child at. The callbacks are called once: s is empty afterwards. A nil s
// draws nothing.
//
// Where gtx itself has a Surfaces set, the layout is laid out inside a child
// of another layout that places it only after measuring it, and room is not
// the window's: the callbacks are then added to the Surfaces gtx has set,
// in order, and called with the area that layout passes. A layout that
// placed the child at an offset calls [PlaceSurfacesAt] instead.
//
// It is named for what the call does: each surface is placed in room, by the
// same rules a surface given that available room is placed by.
func PlaceSurfaces(gtx layout.Context, s *Surfaces, room image.Rectangle) {
	if s == nil {
		return
	}
	draws := s.draws
	s.draws = nil
	if outer := surfacesFrom(gtx); outer != nil && outer != s {
		outer.draws = append(outer.draws, draws...)
		return
	}
	for _, draw := range draws {
		draw(gtx, room)
	}
}

// surfacesAt returns gtx with the Surfaces it has set moved into the
// coordinates of a child placed at off: a new Surfaces is set, and the
// callbacks recorded in it are added to the one gtx had as one callback that
// draws them inside a transform pushed at off, in the area moved by off. It
// is the Surfaces half of [AvailableRoomAt]. Where no Surfaces is set, or off
// is zero, gtx is returned as it is.
func surfacesAt(gtx layout.Context, off image.Point) layout.Context {
	outer := surfacesFrom(gtx)
	if outer == nil || off == (image.Point{}) {
		return gtx
	}
	values := maps.Clone(gtx.Values)
	inner := new(Surfaces)
	values[surfacesKey] = inner
	gtx.Values = values
	outer.draws = append(outer.draws, func(gtx layout.Context, room image.Rectangle) {
		st := op.Offset(off).Push(gtx.Ops)
		PlaceSurfaces(gtx, inner, room.Sub(off))
		st.Pop()
	})
	return gtx
}

// HasSurfaces reports whether gtx has a Surfaces set: whether a floating
// surface laid out under gtx is recorded to be placed by a layout above it
// rather than drawn at once. A layout that places its children only after
// measuring them and cannot read the available room itself asks it, and
// collects its children's surfaces only where a layout above it places them.
func HasSurfaces(gtx layout.Context) bool { return surfacesFrom(gtx) != nil }

// surfacesFrom reads the Surfaces set on gtx, nil where none is.
func surfacesFrom(gtx layout.Context) *Surfaces {
	s, _ := gtx.Values[surfacesKey].(*Surfaces)
	return s
}

// Record adds draw to the Surfaces set on gtx and reports whether one was
// set. A component that floats a surface calls it as it is laid out: where it
// reports true, the surface is drawn later by the layout that set the
// Surfaces, with the available room in the coordinates gtx has, and the
// component draws nothing now; where it reports false, the component draws
// the surface at once.
func Record(gtx layout.Context, draw func(gtx layout.Context, room image.Rectangle)) bool {
	s := surfacesFrom(gtx)
	if s == nil {
		return false
	}
	s.draws = append(s.draws, draw)
	return true
}

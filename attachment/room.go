package attachment

import (
	"image"
	"maps"

	"gioui.org/layout"
)

// availableRoomKey is the key the available room is set under in
// layout.Context.Values.
const availableRoomKey = "github.com/vibrantgio/components/attachment.AvailableRoom"

// unboundedRoom is the extent [AvailableRoom] reports where no window set the
// available room: wide and tall enough that a surface is never shortened by it
// and never kept off a side, so a surface given it is placed as one given no
// available room is.
const unboundedRoom = 1 << 20

// WithAvailableRoom returns gtx with r set as the available room, in gtx's
// own coordinates: the rectangle a floating surface laid out under gtx may be
// placed in. A window sets it to its content area before it lays out what it
// holds.
//
// The available room and the [Surfaces] are one mechanism: the room is set at
// the window, and every layout between the window and a floating surface
// moves both by the offset it places its child at, with [AvailableRoomAt]
// where the offset is known before the child is laid out and with
// [WithSurfaces] and [PlaceSurfacesAt] where it is not.
//
// The room is passed on the context because Gio offers no readback of the
// transform an ancestor pushed: the window is the one place where its edges
// are known. gtx.Values is cloned rather than written, so the caller's
// context keeps the room it had.
func WithAvailableRoom(gtx layout.Context, r image.Rectangle) layout.Context {
	values := maps.Clone(gtx.Values)
	if values == nil {
		values = make(map[string]any, 1)
	}
	values[availableRoomKey] = r
	gtx.Values = values
	return gtx
}

// AvailableRoom reads the available room in gtx's own coordinates. Its
// signature is the one a floating surface's AvailableRoom takes, so a caller
// passes the function itself.
//
// Where no window set the room it reports an unbounded rectangle, which places
// a surface as no available room at all does.
func AvailableRoom(gtx layout.Context) image.Rectangle {
	if r, ok := gtx.Values[availableRoomKey].(image.Rectangle); ok {
		return r
	}
	return image.Rect(-unboundedRoom, -unboundedRoom, unboundedRoom, unboundedRoom)
}

// HasAvailableRoom reports whether a window set the available room on gtx. A
// layout that places its children only after measuring them asks it, with
// [HasSurfaces], to know whether a floating surface in a child has anything to
// be placed in beyond the child's own box.
func HasAvailableRoom(gtx layout.Context) bool {
	_, ok := gtx.Values[availableRoomKey].(image.Rectangle)
	return ok
}

// AvailableRoomAt returns gtx moved into the coordinates of a child placed at
// off: the available room less off, and the [Surfaces] gtx has set moved by
// off too, so a floating surface in the child is placed at its anchor
// whichever of the two it is placed by. A layout calls it with the offset it
// pushes before it lays the child out. What gtx does not have set is not set
// on the result either.
func AvailableRoomAt(gtx layout.Context, off image.Point) layout.Context {
	if off == (image.Point{}) {
		return gtx
	}
	gtx = surfacesAt(gtx, off)
	if r, ok := gtx.Values[availableRoomKey].(image.Rectangle); ok {
		gtx = WithAvailableRoom(gtx, r.Sub(off))
	}
	return gtx
}

// PlaceSurfacesAt places the surfaces s recorded in a child of gtx's layout
// placed at off, in the available room gtx has moved by off. The caller has
// pushed the transform the child was placed at, or calls it from inside the
// child, so each surface is drawn in the child's coordinates. Where gtx has a
// Surfaces set, the layout is itself laid out in a child that is placed only
// after it is measured: the surfaces are then added to that Surfaces, moved
// by off, and placed once the outer layout places its child. A nil s places
// nothing.
func PlaceSurfacesAt(gtx layout.Context, s *Surfaces, off image.Point) {
	cgtx := AvailableRoomAt(gtx, off)
	PlaceSurfaces(cgtx, s, AvailableRoom(cgtx))
}

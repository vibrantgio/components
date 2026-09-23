// Package pointershape declares the shape the platform's pointer takes over a
// region: the arrow over controls, lists and chrome, the I-beam over text that
// can be selected or edited, the hand over a link. The shape belongs to the
// region under the pointer, so a region must not leave its shape behind.
//
// Gio's pointer.Cursor op carries no bounds. Adding it records the shape on
// whatever clip area encloses the op (io/input's pointerCollector.cursor
// writes it to the current area), and the router, hit-testing a point, walks
// that area's ancestors and takes the first shape it meets
// (pointerQueue.hit). A region that adds the op with no clip of its own
// therefore writes its shape onto the area around it, and every region inside
// that area wears it — a list beside a text field shows the I-beam.
//
// Declaring pointer.CursorDefault over the region beneath does not fix that:
// CursorDefault is the zero value of an area's cursor field, indistinguishable
// from declaring nothing, so the walk passes straight through it. Bounding the
// shape to the region that owns it is the whole of the fix; the router already
// reads CursorDefault wherever no bounded shape covers the pointer.
package pointershape

import (
	"image"

	"gioui.org/io/pointer"
	"gioui.org/op"
	"gioui.org/op/clip"
)

// Over declares shape for the pointer while it stands in area, given in the
// caller's own coordinates, for this frame alone. It paints nothing: the clip
// it pushes and pops bounds the shape and leaves the drawing state as it found
// it. An empty area declares nothing.
//
// Declare the shape after the region has registered its own input. The router
// scans a frame's areas newest first and, on meeting a region's input area,
// carries on from that area's parent — so a shape declared before the region's
// own event.Op is stepped over and the region shows the shape of whatever
// stands around it. A region whose content is replayed inside its input area,
// as widget.Clickable replays it, already declares after.
func Over(ops *op.Ops, area image.Rectangle, shape pointer.Cursor) {
	if area.Empty() {
		return
	}
	defer clip.Rect(area).Push(ops).Pop()
	shape.Add(ops)
}

// OverSize declares shape over the box a region of size reports, with its
// origin at the caller's own. It is Over for the common case, where a region
// declares its shape over exactly what it laid out.
func OverSize(ops *op.Ops, size image.Point, shape pointer.Cursor) {
	Over(ops, image.Rectangle{Max: size}, shape)
}

package list

import (
	"image"

	"gioui.org/layout"
	"gioui.org/op"

	"github.com/vibrantgio/components/tooltip"
)

// placedRow is one row the list laid out this frame: its size and the
// surfaces of the tooltips laid out in it.
type placedRow struct {
	size     image.Point
	surfaces *tooltip.Surfaces
}

// layoutRows lays n rows out with the list's layout.List, as every entry point
// does, and places the floating surfaces of the tooltips laid out in them.
//
// The list decides a row's offset only after it has measured the row, and it
// moves every row by the scroll, so no row can be given the available room
// as it is: a row's tooltip that read it would be placed against edges moved
// by the row's offset and the scroll. Where gtx has the available room or a
// [tooltip.Surfaces] set, each row is therefore laid out with a Surfaces of
// its own, and once the list has placed its rows the surfaces of every row in
// view are placed with [tooltip.PlaceSurfacesAt] at the row's offset, in the
// room moved by it. The surfaces of a row scrolled out of view are not
// placed: its anchor is out of view, so its surfaces are dismissed.
//
// Where gtx has neither, the rows are laid out with gtx as it is, and a
// tooltip in a row is placed as one laid out anywhere else under gtx is.
func (s *State) layoutRows(gtx layout.Context, n int, rowFn layout.ListElement) layout.Dimensions {
	if !tooltip.HasSurfaces(gtx) && !tooltip.HasAvailableRoom(gtx) {
		return s.l.Layout(gtx, n, rowFn)
	}
	rows := map[int]placedRow{}
	dims := s.l.Layout(gtx, n, func(gtx layout.Context, i int) layout.Dimensions {
		gtx, rs := tooltip.WithSurfaces(gtx)
		d := rowFn(gtx, i)
		rows[i] = placedRow{size: d.Size, surfaces: rs}
		return d
	})
	// The rows in view are placed one after the other from the first one in
	// view, which the scroll has moved up by the position's offset; the
	// list's rows are aligned to its leading edge.
	y := -s.l.Position.Offset
	for i := s.l.Position.First; i < s.l.Position.First+s.l.Position.Count; i++ {
		r, ok := rows[i]
		if !ok {
			break
		}
		off := image.Pt(0, y)
		st := op.Offset(off).Push(gtx.Ops)
		tooltip.PlaceSurfacesAt(gtx, r.surfaces, off)
		st.Pop()
		y += r.size.Y + s.l.Gap
	}
	return dims
}

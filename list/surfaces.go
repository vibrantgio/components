package list

import (
	"image"

	"gioui.org/layout"

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
// moves every row by the scroll, so no row can be given the available room:
// a row's tooltip that read the room passed into the list would be placed
// against edges moved by the row's offset and the scroll. The list cannot
// read that room itself either, since the layouts that set it are above this
// module. Where gtx has a [tooltip.Surfaces] set, each row is therefore laid
// out with a Surfaces of its own, and once the list has placed its rows the
// surfaces of every row in view are added to the Surfaces gtx has set, moved
// by the row's offset, so the layout that set it places them in the available
// room. The surfaces of a row scrolled out of view are not placed: its
// anchor is out of view, so its surfaces are dismissed.
//
// Where gtx has none, the rows are laid out with gtx as it is, and a tooltip
// in a row is placed as one laid out anywhere else under gtx is.
func (s *State) layoutRows(gtx layout.Context, n int, rowFn layout.ListElement) layout.Dimensions {
	if !tooltip.HasSurfaces(gtx) {
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
		tooltip.PlaceSurfaces(tooltip.SurfacesAt(gtx, off), r.surfaces, image.Rectangle{})
		y += r.size.Y + s.l.Gap
	}
	return dims
}

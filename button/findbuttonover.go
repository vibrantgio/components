package button

import (
	"image"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"
)

// findButtonFootDp is the clear space between the find button's bottom and
// the foot of the content's scrolling region: the top of the fixed band where
// the content column keeps one at its foot, and the column's foot where it
// keeps none.
//
// MEASURED, voicememos-collection-search-light.png, a 1x capture: the find
// button's bottom is y=1277 and the waveform's track starts at y=1312, 35
// clear rows below it.
const findButtonFootDp unit.Dp = 35

// FindButtonOver lays out the content column with the find button over it.
// content is the column's scrolling region. band, if non-nil, is the fixed
// band the column keeps at its foot: it takes its own height at the bottom,
// and content takes the rest. find, if non-nil, is the find button. The
// caller passes it while the search field on chrome holds a query that
// filters the collection, and nil when the field is empty.
//
// The find button stands centred on the column, findButtonFootDp above the
// band's top, or above the column's foot where band is nil. It is laid out
// after the content, so it stands over the content and takes the pointer
// there. Where the centring leaves an odd column, that column leads the find
// button.
//
// MEASURED, voicememos-collection-search-light.png: the find button's box
// runs x 1013-1097 in a content column that runs x 377-1732. 636 columns
// lead it and 635 trail it, so its centre line stands 0.5 pt trailing of the
// column's.
//
// Give it the content column's own share of the window, so that a sidebar
// beside the column keeps the find button off it.
func FindButtonOver(content, band, find layout.Widget) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		size := gtx.Constraints.Max
		foot := size.Y
		var bandCall op.CallOp
		if band != nil {
			bgtx := gtx
			bgtx.Constraints.Min = image.Pt(size.X, 0)
			m := op.Record(gtx.Ops)
			d := band(bgtx)
			bandCall = m.Stop()
			foot = max(size.Y-d.Size.Y, 0)
		}

		cgtx := gtx
		cgtx.Constraints = layout.Exact(image.Pt(size.X, foot))
		content(cgtx)

		if band != nil {
			off := op.Offset(image.Pt(0, foot)).Push(gtx.Ops)
			bandCall.Add(gtx.Ops)
			off.Pop()
		}

		if find != nil {
			fgtx := gtx
			fgtx.Constraints.Min = image.Point{}
			m := op.Record(gtx.Ops)
			d := find(fgtx)
			call := m.Stop()
			x := (size.X - d.Size.X + 1) / 2
			y := max(foot-gtx.Dp(findButtonFootDp)-d.Size.Y, 0)
			off := op.Offset(image.Pt(x, y)).Push(gtx.Ops)
			call.Add(gtx.Ops)
			off.Pop()
		}
		return layout.Dimensions{Size: size}
	}
}

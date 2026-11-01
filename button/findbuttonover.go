package button

import (
	"image"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"
)

// findButtonFootDp is the clear space between the find button's bottom and
// what stands below it: the track of the fixed band where the content column
// keeps one at its foot, and the column's foot where it keeps none.
//
// MEASURED, voicememos-collection-search-light.png, a 1x capture: the find
// button's bottom is y=1277 and the waveform's track starts at y=1312, 35
// clear rows below it.
const findButtonFootDp unit.Dp = 35

// findButtonBandTopDp is the clear space between the foot of the content's
// scrolling region and the find button's top, where the find button stands
// in a fixed band.
//
// MEASURED, voicememos-collection-search-light.png: the find button's top is
// y=1253. The shadow above it stops in a flat line across its whole width:
// y=1243 is 1-2/255 darker than #ffffff, and y=1242 is #ffffff at every
// column. The region's foot is y=1243, 10 rows above the find button.
const findButtonBandTopDp unit.Dp = 10

// FindButtonRoomDp is the room a fixed band keeps above its track for the
// find button: the clear space under the region's foot, the find button's
// height and the clear space down to the track.
const FindButtonRoomDp = findButtonBandTopDp + findButtonHeightDp + findButtonFootDp

// FindButtonOver lays out the content column with the find button at its
// foot. content is the column's scrolling region. band, if non-nil, is the
// fixed band the column keeps at its foot: it takes its own height at the
// bottom, and content takes the rest. find, if non-nil, is the find button.
// The caller passes it while the search field on chrome holds a query that
// filters the collection, and nil when the field is empty.
//
// Where the column keeps a band, the find button stands in the band, so no
// content scrolls under it. Its top stands findButtonBandTopDp below the
// band's top, the foot of the scrolling region, and the band keeps
// [FindButtonRoomDp] above its track. The find button is drawn over the
// band. Its shadow is the toolbar control's, painted in the band's deferred
// pass, and it fades out inside the band above the find button, so it never
// falls on the scrolling region.
//
// Where the column keeps no band, the find button stands findButtonFootDp
// above the column's foot, over the content.
//
// In both cases the find button is centred on the column and takes the
// pointer where it stands. Where the centring leaves an odd column, that
// column leads the find button.
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

		if find == nil {
			return layout.Dimensions{Size: size}
		}
		fgtx := gtx
		fgtx.Constraints.Min = image.Point{}
		m := op.Record(gtx.Ops)
		d := find(fgtx)
		call := m.Stop()
		x := (size.X - d.Size.X + 1) / 2
		y := max(foot-gtx.Dp(findButtonFootDp)-d.Size.Y, 0)
		if band != nil {
			y = foot + gtx.Dp(findButtonBandTopDp)
		}
		off := op.Offset(image.Pt(x, y)).Push(gtx.Ops)
		call.Add(gtx.Ops)
		off.Pop()
		return layout.Dimensions{Size: size}
	}
}

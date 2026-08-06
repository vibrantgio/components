// Package hit extends a control's pointer target beyond its visual bounds.
//
// Density (E1.3) shrinks the drawn control — tokens.Density.ControlHeight is
// 36 dp Comfortable, 28 dp Compact — but the WCAG 2.5.5 pointer target never
// shrinks below tokens.Density.MinHitTarget (44 dp). Before density the two
// rectangles coincided: a control's event area was exactly its visual area.
// Extend splits them: the event wrapper (a widget.Clickable's or widget.Bool's
// Layout) covers the hit rectangle — max(visual, floor) per axis, centred on
// the visual — while the dimensions reported to the parent stay the visual
// ones, so layout remains dense. The hit slop therefore extends outside the
// widget's bounds; where the slop of neighbouring controls overlaps, the
// control laid out later wins the overlap (Gio delivers to the topmost input
// area).
//
// That last sentence is why this package is for standalone controls only —
// button, checkbox, radio, text field, the dropdown's closed trigger. Stacked
// rows (list, table, dropdown options) tile edge to edge, so every row's slop
// would land on its neighbours and the boundary would stop meaning what it
// draws. Those rows are their own target at their own height, which clears
// WCAG 2.5.8 (AA, 24 dp) and not 2.5.5 (AAA, 44 dp); see tokens.MinHitTarget.
package hit

import (
	"image"

	"gioui.org/layout"
	"gioui.org/op"
)

// Extend lays out w inside the event wrapper lay (widget.Clickable.Layout or
// widget.Bool.Layout, bound to its receiver) so that the wrapper's pointer
// area is at least minPx on each axis, centred on w's visual bounds. The
// returned dimensions are w's own visual dimensions; w draws at the widget
// origin as usual.
func Extend(gtx layout.Context, minPx int, lay func(layout.Context, layout.Widget) layout.Dimensions, w layout.Widget) layout.Dimensions {
	var visual layout.Dimensions
	macro := op.Record(gtx.Ops)
	hitDims := lay(gtx, func(gtx layout.Context) layout.Dimensions {
		inner := op.Record(gtx.Ops)
		visual = w(gtx)
		call := inner.Stop()
		hit := visual.Size
		if hit.X < minPx {
			hit.X = minPx
		}
		if hit.Y < minPx {
			hit.Y = minPx
		}
		// Centre the visual inside the hit rectangle.
		off := op.Offset(image.Pt((hit.X-visual.Size.X)/2, (hit.Y-visual.Size.Y)/2)).Push(gtx.Ops)
		call.Add(gtx.Ops)
		off.Pop()
		return layout.Dimensions{Size: hit}
	})
	call := macro.Stop()
	// Shift back so the visual sits at the widget origin; the hit slop
	// extends outside the visual bounds.
	off := op.Offset(image.Pt(-(hitDims.Size.X-visual.Size.X)/2, -(hitDims.Size.Y-visual.Size.Y)/2)).Push(gtx.Ops)
	call.Add(gtx.Ops)
	off.Pop()
	return visual
}

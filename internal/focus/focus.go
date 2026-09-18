// Package focus holds the one focus idiom every control in this library
// wears, so that "what a focused control looks like" is written once.
//
// # The idiom
//
// A focused control wears a halo: a [Width] band in the platform's keyboard
// focus indicator lying on its own outline, [Outside] of it past the control
// and [Outside] of it over the control's own outermost band. The control
// keeps its edge and its fill, and measures the same box focused as at rest.
// A keyboard user learns one width, one colour and one place, and learns it
// once.
//
// MEASURED, save-dialog-light.png and save-dialog-dark.png, the focused
// "Save As:" field, whose box runs x 264-495 and whose ring is the only one
// in the reference: the band covers x 262-265 at the leading end and x
// 494-497 at the trailing one, and y 205-208 and y 231-234 down the column
// at x=350, with the sheet's own fill unblended at x=261, x=266, x=493,
// x=498, y=204, y=209, y=230 and y=235. Four px on every side, hard-edged,
// straddling the box: two px outside it and two px over it. The two px over
// it do not hide what is there — the field's own #f3f3f3 edge column reads
// through the band at x=264 — because the platform's name carries a
// coverage, which is why [Ring] flattens it onto what it lands on rather
// than replacing it.
//
// # Where the halo goes
//
// On the control's own outline, whatever the control is: a text field, a
// pop-up trigger, a chip and a button ring at the box they fill; a checkbox
// and a radio ring at their glyph's box, in the slack their footprint holds
// around it; a link rings at its glyphs padded [Outside] clear, a link having
// no box of its own to take. Nothing reports a larger box to make room — the
// footprint is the pointer target and taking the keyboard may not grow it —
// so [Halo] hands its band to op.Defer and paints past the clip a
// gioui.org/widget.Clickable puts around whatever it wraps, the way a
// toolbar control's shadow reaches past its band.
package focus

import (
	"image"
	"image/color"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"

	vgcolor "github.com/vibrantgio/theme/color"
	"github.com/vibrantgio/theme/tokens"
)

// Width is the halo's thickness: 4 dp, at every density and on every control.
// A halo is a keyboard affordance rather than an ornament, so it does not
// thin out when the controls around it tighten.
//
// MEASURED, save-dialog-{light,dark}.png: see the package doc.
const Width = unit.Dp(4)

// Outside is how much of [Width] lies past the control's own box: 2 dp, half
// the band. The other half lies over the control's outermost 2 dp.
const Outside = Width / 2

// Ring is the colour a focused control draws its halo in: the platform's
// keyboard focus indicator, flattened over beneath — the opaque fill that
// part of the band lies on, which is the surface the control stands on
// outside its box and the control's own fill inside it.
//
// One name for every control on every surface. The platform's value carries
// its coverage, so what the band lands as is the coverage and the fill under
// it, and nothing else: no control needs a second answer.
func Ring(p tokens.PlatformColors, beneath color.NRGBA) color.NRGBA {
	return vgcolor.Flatten(p.KeyboardFocusIndicator, beneath)
}

// Halo paints the focus halo on box's outline with corner radius radius:
// [Width] wide, [Outside] of it past box and the rest over box's own edge.
//
// outside is the colour the half past the box takes and over the colour the
// half on the box takes; they are [Ring] flattened onto the surface the
// control stands on and onto the control's own fill. Two colours rather than
// one because the platform's indicator carries a coverage and Gio's
// rasterizer would composite it in linear light where the platform
// composites in encoded sRGB — so each half is handed the colour it actually
// lands as, and a halo half on an accent fill and half on the window does
// not dissolve into either.
//
// The band goes to op.Defer, which restores the transform and resets the
// clip: a control reports the same box focused as at rest, so the half past
// that box would otherwise be cut away by the clip a
// gioui.org/widget.Clickable puts around whatever it wraps. Deferred ops run
// in the order they were deferred, so a halo drawn while the window's
// content laid out stands under every floating level deferred after it — a
// popover, a menu, a tooltip, a modal.
func Halo(gtx layout.Context, box image.Rectangle, radius int, outside, over color.NRGBA) {
	halo(gtx, clip.RRect{Rect: box, SE: radius, SW: radius, NE: radius, NW: radius}, box, outside, over)
}

// HaloEllipse is [Halo] around an elliptical control — the radio's disc,
// which has no corner radius to take.
func HaloEllipse(gtx layout.Context, box image.Rectangle, outside, over color.NRGBA) {
	halo(gtx, clip.Ellipse(box), box, outside, over)
}

// outline is the shape a control's box is drawn as: a rounded rectangle for
// every control but the radio, an ellipse for that one.
type outline interface {
	Op(*op.Ops) clip.Op
	Path(*op.Ops) clip.PathSpec
}

// halo paints the band on shape's outline, in two passes: the whole band in
// outside, then the half of it lying on the control in over, clipped to the
// control's own shape.
//
// Two passes rather than two strokes. A stroke is centred on the path it
// follows, so the only way to lay a band of half the width exactly on the
// inner half of the outline is to stroke that outline at the full width
// again under a clip of the shape — the same idiom the picker's trigger and
// the bordered toolbar control draw their edge with, and the one thing that
// keeps both sides of the inner half following the corner.
func halo(gtx layout.Context, shape outline, box image.Rectangle, outside, over color.NRGBA) {
	if box.Empty() || (outside.A == 0 && over.A == 0) {
		return
	}
	w := gtx.Dp(Width)
	if w < 1 {
		w = 1
	}
	macro := op.Record(gtx.Ops)
	if outside.A != 0 {
		paint.FillShape(gtx.Ops, outside, clip.Stroke{Path: shape.Path(gtx.Ops), Width: float32(w)}.Op())
	}
	if over.A != 0 && over != outside {
		inner := shape.Op(gtx.Ops).Push(gtx.Ops)
		paint.FillShape(gtx.Ops, over, clip.Stroke{Path: shape.Path(gtx.Ops), Width: float32(w)}.Op())
		inner.Pop()
	}
	op.Defer(gtx.Ops, macro.Stop())
}

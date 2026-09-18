// Package toolbarface holds the geometry components/picker's chrome-variant
// trigger is drawn from: the fill it stands off its band with and tints under
// the pointer, the hairline around it, the focus ring that replaces that
// hairline, the density's control height, the pointer target's placement,
// and the pop-up mark that says the control holds one of several values.
//
// It is internal because it is a seam and not a component: a caller reaches
// for picker.Toolbar or picker.RenderToolbar, and those document the control
// this draws.
package toolbarface

import (
	"image"
	"image/color"

	"gioui.org/font"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"

	vgcolor "github.com/vibrantgio/theme/color"
	"github.com/vibrantgio/theme/tokens"
	"github.com/vibrantgio/theme/typeset"

	"github.com/vibrantgio/components/internal/control"
	"github.com/vibrantgio/components/internal/focus"
)

// edgeDp is the rim's width — one hair at every density, the width every
// other derived edge in this library is drawn at (components/paragraph's
// chipEdge, components/input's field bezel). It is a width rather than a
// token because no scale in the system carries line weights.
const edgeDp = unit.Dp(1)

// The trigger draws the platform's pop-up mark — the stacked chevron pair —
// at the one size the platform draws it: see [control.DrawMark].
//
// MEASURED, finder-window-light.png: the Finder toolbar's view pop-up draws
// the pair 8 px wide and 11 px tall (x 726-733, upper y 21-25, lower y 27-31)
// in a control 36 px tall, which is the same eight by eleven the Save dialog's
// pop-up draws in one 24 px tall. The mark does not scale with the control it
// stands in, so this trigger draws no ratio of its own. The single chevron
// beside it in that capture (x 792-799, y 24-28) is the group control's, a
// PULL-DOWN — a menu of actions rather than a choice — and this component has
// no pull-down: a picker is single-choice by contract, so both of its
// triggers wear the pop-up's pair.
//
// The clearance between the mark's last column and the control's trailing
// edge is the pop-up's own [control.PopupMarkTrailDp], and the label's origin is
// [control.PopupLeadDp]: the two triggers are one control drawn in two places
// and spend one pair of insets.

// State is the explicit visual state a static render draws in. The zero value
// is a resting control.
type State struct {
	Hovered bool
	Pressed bool
	Focused bool
}

// state is the token vocabulary's name for the interaction the control is in.
// Press wins over hover, because a pressed control is under the pointer by
// definition and the deeper walk is the one that has something to say.
func (s State) state() tokens.State {
	switch {
	case s.Pressed:
		return tokens.StatePressed
	case s.Hovered:
		return tokens.StateHover
	}
	return tokens.StateNormal
}

// Fill is the control's own fill: the platform's measured toolbar control
// fill at rest, that fill under the platform's hover overlay while the
// pointer is on it, and under its press overlay while it is held.
//
// The control is a figure on its band and not part of it. MEASURED,
// finder-window-untinted-dark.png, a frontmost window: every bordered control
// in its toolbar reads #262626 against a band of #1e1e1e, eight levels
// lighter than what it stands on, and finder-window.png's view pop-up agrees
// in direction with #242d32 on a #232a2e band. The light value is that same
// control's #ffffff in finder-window-light.png, where the band beneath it is
// the content's own white and the control is told from it by its shadow; on
// the chrome material this library paints it stands eight levels lighter,
// which is the dark appearance's step to the level.
//
// The fill does not depend on what the control stands on, which is why this
// takes no surface: the platform gives its toolbar control a fill of its own,
// as it gives its push button one.
//
// A toolbar button is the one control on this platform that tints under the
// pointer — a push button, a list row and a sidebar row do not, which is the
// reading control-hover-{light,dark}.png records — so the overlay is applied
// here and nowhere else in this library. The light hover lands on the
// capture to the byte: the overlay over #ffffff is the #f2f2f2 that capture
// holds.
func Fill(p tokens.PlatformColors, state tokens.State) color.NRGBA {
	switch state {
	case tokens.StatePressed:
		return vgcolor.Flatten(p.PressOverlay, p.ToolbarControlFill)
	case tokens.StateHover:
		return vgcolor.Flatten(p.HoverOverlay, p.ToolbarControlFill)
	}
	return p.ToolbarControlFill
}

// Rim is the hairline around the control: the platform's separator flattened
// over the fill it is drawn on — but only where that hairline LIFTS the fill.
// Where it would darken it instead, the platform draws no hairline at all and
// this answers the zero value, which is no colour.
//
// MEASURED. Dark, finder-window-untinted-dark.png: the toolbar control wears a
// 1 px rim reading #404040 over its #262626 fill, lighter than both the fill
// and the #1e1e1e band — a highlight that lifts the control's edge. The seam
// over that fill gives #3b3b3b, five of 255 short of the pixel, which is the
// miss this name carries. Light, finder-window-light.png: the band steps from
// 249, 250, 251 straight up to the control's #ffffff with no darker row on any
// side — what stands outside the control there is its drop shadow, which falls
// AWAY from the control, and the seam's own black would be an edge the
// platform does not draw.
func Rim(p tokens.PlatformColors, beneath color.NRGBA) color.NRGBA {
	rim := vgcolor.Flatten(p.Separator, beneath)
	if rim.R <= beneath.R && rim.G <= beneath.G && rim.B <= beneath.B {
		return color.NRGBA{}
	}
	return rim
}

// Label is the colour the control's own wording reads in: the platform's
// control text, flattened over the fill the wording stands on.
func Label(p tokens.PlatformColors, beneath color.NRGBA) color.NRGBA {
	return vgcolor.Flatten(p.ControlText, beneath)
}

// Mark is the colour the pop-up mark reads in: the platform's control text,
// the name it draws a control's own marks in, flattened over the fill the
// mark stands on. It is the colour the control's wording reads in too — one
// control, one foreground.
//
// MEASURED, save-dialog-{light,dark}.png: the pop-up's pair reads 36 light and
// (224,225,226) dark on fills of #ececec and #333a3f, which is ControlText's
// 216 of 255 flattened onto each to the byte. The secondary label's coverage
// would land at 118 and 163. The toolbar's own pop-up agrees within what a
// thin diagonal can cover: in finder-window-light.png its pair peaks at 77 on
// a #ffffff fill, which is ControlText at 82% coverage, where SecondaryLabel
// would need 140% of a pixel to reach it.
func Mark(p tokens.PlatformColors, beneath color.NRGBA) color.NRGBA {
	return vgcolor.Flatten(p.ControlText, beneath)
}

// Pin is the edge of the offered box that a drawn shape is pinned to.
//
// It is a placement, not a stretch: the shape stays sized to its content and
// what changes is where in the offered box it is drawn and how much of that
// box is reported as used. Only the horizontal axis is pinned, because the
// vertical one is already settled by whatever row the shape stands in.
//
// The seam exists because a shape alone can be placed by its container and
// one handed on to a container that centres whatever it is given cannot: the
// reserved cap and the drawn shape then part company by half the slack, and
// the only place both widths are known is inside the layout.Widget. A pin
// says it there, once.
//
// It costs the container the drawn rect, which is the whole box as far as it
// can tell, so say it only where nothing upstream needs that rect. A
// container that aligns what it is given needs no pin at all, and a pinned
// shape would leave it aiming at a box nothing was drawn in.
type Pin uint8

const (
	// PinNone is the zero value: only the shape drawn is reported, so a row
	// of them is laid out at their own scale and the box around them is the
	// container's business.
	PinNone Pin = iota

	// PinLeading draws the shape at the leading edge of the offered box.
	PinLeading

	// PinTrailing draws the shape at the trailing edge of the offered box.
	PinTrailing
)

// Layout draws w at p's edge of the offered box — the horizontal half of
// gtx.Constraints.Max — and reports that box rather than w's own size, which
// is what lets a caller upstream find the pinned edge where it asked for it.
// PinNone lays w out untouched, so pinning nothing costs nothing.
//
// The whole of w is offset, so the pointer area registered over it travels
// with the shape it belongs to.
func (p Pin) Layout(gtx layout.Context, w layout.Widget) layout.Dimensions {
	if p == PinNone {
		return w(gtx)
	}
	macro := op.Record(gtx.Ops)
	dims := w(gtx)
	call := macro.Stop()

	box := dims.Size
	box.X = max(box.X, gtx.Constraints.Max.X)
	off := 0
	if p == PinTrailing {
		off = box.X - dims.Size.X
	}
	o := op.Offset(image.Pt(off, 0)).Push(gtx.Ops)
	call.Add(gtx.Ops)
	o.Pop()
	return layout.Dimensions{Size: box, Baseline: dims.Baseline}
}

// Draw paints the chrome variant's trigger: the fill it stands off its band
// with and tints under the pointer, the hairline or the focus ring that
// replaces it, the label, and the pop-up mark.
func Draw(
	gtx layout.Context,
	shaper *text.Shaper,
	label string,
	p tokens.PlatformColors,
	sp tokens.SpacingScale,
	labelStyle tokens.TextStyle,
	d tokens.Density,
	s State,
) layout.Dimensions {
	// The control carries an opaque fill of its own in every state, so every
	// coverage it draws over that fill is flattened onto it and nothing it
	// draws depends on the band beneath.
	fill := Fill(p, s.state())
	labelForeground := Label(p, fill)
	markForeground := Mark(p, fill)

	// The two triggers spend one pair of insets, both MEASURED off the Save
	// dialog's "File Format:" pop-up: the label's origin eleven columns in
	// from the fill's edge, and nine clear columns between the mark's last
	// column and the trailing edge. The same nine stands in the Finder
	// toolbar's 36 px pop-up (finder-window-light.png, the pair ending at
	// x=733 against a fill ending at x=742), so the clearance is fixed and not
	// a ratio of the control's height.
	lead := gtx.Dp(control.PopupLeadDp)
	trail := gtx.Dp(control.PopupMarkTrailDp)
	minH := gtx.Dp(unit.Dp(d.ControlHeight))
	gap := gtx.Dp(unit.Dp(sp.S3))
	mark := gtx.Dp(control.MarkWDp)

	// Record the label's material and its layout to learn its size before
	// anything is painted. typeset.Layout rather than widget.Label.Layout
	// because the role's line height has to be the height of the label box
	// and Gio alone reports the drawn glyph extent instead — see theme/typeset.
	mColor := op.Record(gtx.Ops)
	paint.ColorOp{Color: labelForeground}.Add(gtx.Ops)
	material := mColor.Stop()

	// The line box is capped to the control's height, as the form trigger's
	// is: a pop-up is not sized by the text it carries, and a Compact control
	// is shorter than the line box its role declares.
	labelGtx := gtx
	labelGtx.Constraints.Min = image.Point{}
	labelGtx.Constraints.Max.Y = minH
	if maxLabelW := gtx.Constraints.Max.X - lead - gap - mark - trail; maxLabelW > 0 {
		labelGtx.Constraints.Max.X = maxLabelW
	}
	mLabel := op.Record(gtx.Ops)
	labelDims := typeset.Layout(labelGtx, shaper,
		typeset.Label(labelStyle, 1), typeset.Font(labelStyle, font.Normal),
		unit.Sp(labelStyle.Size), label, material)
	labelCall := mLabel.Stop()

	// Sized to content across, not to the width it was given: the control
	// names a choice, and one that stretched would be a banner. Down, it is
	// the density's control height and nothing else, which is what the form
	// trigger draws and what keeps the two variants one control.
	w := lead + labelDims.Size.X + gap + mark + trail
	w = min(w, gtx.Constraints.Max.X)
	h := min(minH, gtx.Constraints.Max.Y)
	size := image.Pt(w, h)
	box := image.Rectangle{Max: size}

	// The corner is fully rounded — half the control's height. MEASURED,
	// finder-window-untinted-light.png: the toolbar's group pull-down spans
	// y 34-69 and its sub-pixel left edge reaches its extreme over rows 50-53,
	// the control's own middle, which is a capsule; circular fits to that
	// edge run 17.4 to 19.1 about the half-height's 18, the spread the
	// platform's continuous corner puts on a circular fit everywhere else in
	// this reference. finder-window.png's view pop-up agrees: 36 px tall,
	// its edge 12 columns in on its first row against a capsule's 13.8.
	//
	// This is the one place the two variants differ in shape. The form
	// trigger is the dialog's pop-up, which draws the button's rounded
	// rectangle; the toolbar's is a capsule, and the platform draws every
	// bordered control in a toolbar band that way.
	radius := box.Dy() / 2

	// A focused control's edge IS the focus ring: the ring replaces the rim
	// rather than being drawn inside it, and in the light appearance there is
	// no rim to replace — the platform draws none there, so a resting light
	// control is its fill and nothing else. Drawn inside, the two make a
	// three-line sandwich — hairline, a pixel of fill, then the ring — which
	// reads as a dirty halo around the outline, the same "a band beside a
	// boundary reads as part of that boundary" that holds components/button's
	// ring clear of its edge. Nothing else moves: the shape measures the same
	// box focused as at rest, and the label does not shift.
	band, edgeColor := max(gtx.Dp(edgeDp), 1), Rim(p, fill)
	if s.Focused {
		band, edgeColor = gtx.Dp(focus.Width), focus.Ring(p, fill)
	}
	if edgeColor.A == 0 {
		band = 0
	}

	// The fill as the control's whole shape, and the edge as a band laid ON
	// that shape rather than as a shape beneath it: the band is a stroke of
	// twice the edge's width centred on the shape's outline and clipped to
	// that shape, which puts every pixel of it inside the box this control
	// reports where a stroke of the edge's own width would fall half outside.
	// Drawn that way both of the band's sides follow the corner, which four
	// rectangles could not.
	outer := clip.RRect{Rect: box, SE: radius, SW: radius, NE: radius, NW: radius}
	paint.FillShape(gtx.Ops, fill, outer.Op(gtx.Ops))
	if band > 0 {
		edgePath := outer.Path(gtx.Ops)
		edgeArea := outer.Push(gtx.Ops)
		paint.FillShape(gtx.Ops, edgeColor, clip.Stroke{Path: edgePath, Width: float32(2 * band)}.Op())
		edgeArea.Pop()
	}

	// The label at its own origin, clipped to the control's shape so a line
	// box taller than the control is cut by the control rather than drawn
	// past it.
	area := outer.Push(gtx.Ops)
	lo := op.Offset(image.Pt(lead, (h-labelDims.Size.Y)/2)).Push(gtx.Ops)
	labelCall.Add(gtx.Ops)
	lo.Pop()
	area.Pop()

	// The mark is handed its own column at the shape's full height and
	// centres itself in it, at the one size the platform draws it.
	control.DrawMark(gtx, image.Rect(w-trail-mark, 0, w-trail, h), markForeground)

	pointer.CursorPointer.Add(gtx.Ops)
	return layout.Dimensions{Size: size}
}

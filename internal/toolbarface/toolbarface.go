// Package toolbarface holds the geometry components/picker's chrome-variant
// trigger is drawn from: the fill it stands off its band with and tints under
// the pointer, the hairline around it, the focus halo it wears over that
// hairline, the density's toolbar control height, the pointer target's
// placement, and the pop-up mark that says the control holds one of several
// values.
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
	"github.com/vibrantgio/components/internal/surface"
	"github.com/vibrantgio/components/pointershape"
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

	// StandsOn is the opaque fill the control stands on — its band. A
	// focused control's halo lies half past its own box, so the half out
	// there has to be flattened onto what is actually there. Alpha zero is
	// no answer, and the halo takes the control's own fill on both halves.
	StandsOn color.NRGBA

	// Checked is the persistent state of a control that records a yes — a
	// toolbar toggle that says whether the thing it governs stands. It is
	// not an interaction: it outlives the pointer, and the control draws it
	// as the platform draws the chosen segment of a segmented control. See
	// [CheckedPatch].
	Checked bool
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

// Rim is the hairline around the control, and it is
// [control.ToolbarControlRim]: the platform's measured #404040 in the dark
// appearance and no colour in the light one, where it draws none. The
// measurement lives with that name.
func Rim(p tokens.PlatformColors) color.NRGBA {
	return control.ToolbarControlRim(p)
}

// Label is the colour the control's own wording reads in: what the TOOLBAR
// draws its own words in. It is opaque, so what it stands on does not change
// it; beneath is taken for the signature [Mark] and every other face colour
// here shares and for the day a coverage is measured.
//
// MEASURED, finder-window-light.png: the band's own title "Applications"
// standing bare over a #ffffff band plateaus at #4d4d4d over 131 pixels, which
// is no name in the catalogue — ControlText's 216 of 255 gives #272727 there.
// The measurement lives with [tokens.PlatformColors.ToolbarLabel].
func Label(p tokens.PlatformColors, beneath color.NRGBA) color.NRGBA {
	return p.ToolbarLabel
}

// Mark is the colour a chrome control's mark reads in, and it is the colour
// its wording reads in: one control, one foreground, and one name for
// everything a toolbar says.
//
// MEASURED, finder-window-light.png: the group pull-down's grid glyph
// (x 769-786, y 17-34) plateaus at #4d4d4d over 24 pixels and the search
// capsule's magnifier (x 965-980, y 18-34) over 15 — the same plateau the
// band's title holds, so it is the drawn colour and not the partial coverage
// a thin stroke reaches. finder-window-untinted-dark.png reads #e9e9e9 for the
// same glyphs. A FORM control is a different reading: the Save dialog's pop-up
// draws its mark at ControlText exactly (36 light and (224,225,226) dark on
// fills of #ececec and #333a3f), which is why components/picker's form trigger
// keeps that name and this one answers for the toolbar.
func Mark(p tokens.PlatformColors, beneath color.NRGBA) color.NRGBA {
	return p.ToolbarLabel
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
// with and tints under the pointer, the hairline and the focus halo that
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
	minH := gtx.Dp(unit.Dp(d.ToolbarControlHeight))
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
	// the density's TOOLBAR control height and nothing else. That is a
	// different number from the form trigger's control height —
	// tokens.Density.ToolbarControlHeight's 36 against ControlHeight's 24 —
	// because the platform draws it as a different control: every bordered
	// control in the stored Finder toolbars measures 36 where the same
	// window's dialog pop-up measures 24. The two variants are one component
	// drawn in two places, and the place settles the height.
	w := lead + labelDims.Size.X + gap + mark + trail
	w = min(w, gtx.Constraints.Max.X)
	h := min(minH, gtx.Constraints.Max.Y)
	size := image.Pt(w, h)
	box := image.Rectangle{Max: size}

	// The corner is fully rounded — half the control's height, so it follows
	// the toolbar height the box was just measured to. MEASURED,
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

	outer := Capsule(gtx, box, radius, p, fill, s)

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

	pointershape.OverSize(gtx.Ops, size, pointer.CursorPointer)
	return layout.Dimensions{Size: size}
}

// Capsule paints the bordered toolbar control's own box into box: its fill,
// the rim around it, and the focus halo on that outline while it holds the
// keyboard. It reports the rounded shape it drew, which a caller pushes to
// clip whatever stands inside that box.
//
// Every bordered toolbar control in this library is drawn through it — the
// picker's chrome trigger and components/button's chrome variant — so a
// control labelled with a symbol and a pop-up are one box drawn in two places.
//
// The shadow the control casts on its band is NOT drawn here: it falls
// outside the box and so outside the clip a widget.Clickable puts around
// whatever it wraps. [Cast] draws it, in the band's own pass.
func Capsule(gtx layout.Context, box image.Rectangle, radius int, p tokens.PlatformColors, fill color.NRGBA, s State) clip.RRect {
	// The control keeps its own rim whatever the keyboard is doing: focus is
	// the halo laid on the outline and not a second answer to what the edge
	// is. In the light appearance there is no rim at all — the platform
	// draws none there, so a resting light control is its fill and nothing
	// else — and a focused one in that appearance is its fill under the
	// halo.
	band, edgeColor := max(gtx.Dp(edgeDp), 1), Rim(p)
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
	if s.Checked {
		CheckedPatch(gtx, box, p, fill)
	}
	if band > 0 {
		edgePath := outer.Path(gtx.Ops)
		edgeArea := outer.Push(gtx.Ops)
		paint.FillShape(gtx.Ops, edgeColor, clip.Stroke{Path: edgePath, Width: float32(2 * band)}.Op())
		edgeArea.Pop()
	}
	if s.Focused {
		standsOn := surface.Or(s.StandsOn, fill)
		focus.Halo(gtx, box, radius, focus.Ring(p, standsOn), focus.Ring(p, fill))
	}
	return outer
}

// CheckedPatch paints what a bordered toolbar control lays over its own fill
// while it records a yes: the patch the platform fills the chosen segment of
// a segmented control with, inset inside the control's box and cornered at
// half its own height.
//
// MEASURED, finder-window-untinted-dark.png, the four-segment view control at
// x 796-943, y 46-81 with its list segment chosen: the patch spans x 836-867,
// y 51-76 — 32 by 26 in a segment 37 wide and a control 36 tall, so five rows
// clear above and below and two and a half columns clear at either end — and
// reads #494949 against the control's own #262626.
// finder-window-untinted-light.png draws the same 32 by 26 (x 814-845,
// y 39-64) against its #f7f7f7. The coverage between patch and fill is
// tokens.PlatformColors.ToolbarCheckedOverlay; the insets are here, being
// lengths.
//
// The corner fits the patch's own half-height: the per-row coverage of its top
// runs 8, 6, 5, 4, 3, 2, 1 columns of inset against a capsule's 9.4, 7.7, 6.5,
// 5.5, 4.7, 4.0, 3.3 at r = 13 — the spread the platform's continuous corner
// puts on a circular fit everywhere in the reference, and the same shape the
// control around it is drawn with.
//
// beneath is the fill the patch is laid over, so the coverage lands on
// whatever the control carries in its interaction state: a checked control
// still tints under the pointer.
func CheckedPatch(gtx layout.Context, box image.Rectangle, p tokens.PlatformColors, beneath color.NRGBA) {
	insetY, insetX := gtx.Dp(control.ToolbarCheckedInsetYDp), gtx.Dp(control.ToolbarCheckedInsetXDp)
	patch := image.Rect(box.Min.X+insetX, box.Min.Y+insetY, box.Max.X-insetX, box.Max.Y-insetY)
	if patch.Dx() <= 0 || patch.Dy() <= 0 {
		return
	}
	radius := patch.Dy() / 2
	paint.FillShape(gtx.Ops, vgcolor.Flatten(p.ToolbarCheckedOverlay, beneath), clip.RRect{
		Rect: patch, SE: radius, SW: radius, NE: radius, NW: radius,
	}.Op(gtx.Ops))
}

// Shadow is the drop shadow a bordered toolbar control casts on its band
// under p's appearance: the peak coverage with the reach and the offset
// measured beside it. [Cast] spreads it.
func Shadow(p tokens.PlatformColors) tokens.DropShadow {
	return control.ToolbarShadowOf(p)
}

// Cast lays w out and paints the drop shadow the bordered toolbar control w
// drew casts on the band it stands on, sized to what w reported.
//
// The shadow is handed to [op.Defer], which is what makes a band's shadows
// one pass: every one of them is painted after every column of the window has
// laid out, so a control's shadow is not covered over one column and left
// standing over another — the defect a chrome row's control shows against a
// note column that paints its own fill, where the same control's neighbour
// over a chrome column keeps its whole ramp. Defer resets the clip stack and
// restores the transform, so the reach past the band is cut by the window and
// never by the column the control happens to stand in.
//
// The control's own ops stay where they are, which is what keeps the band
// ahead of the columns in the reading order. Only the shadow moves, and it is
// drawn with the control's box cut out of it, so painting it after the control
// lands what painting it under the control landed.
//
// WHILE THE CONTROL IS FOCUSED the cut is the focus halo's footprint instead —
// the control's box grown by [focus.OutsidePx], which is the half of the band
// that lies past the box. The halo is drawn inside this call and the shadow
// after it, so without the larger cut the ramp runs over the band: the find
// recess measured up to 11 of 255 of shadow lying on its own halo. The state
// the control is drawn in is what says which cut answers, and it is the state
// this call already receives.
//
// The radius is half the reported height: every bordered control in a stored
// toolbar band is a capsule. The reach and the offset are the appearance's
// own, which is why the whole reading is passed rather than its peak alone.
//
// shadow is [Shadow]'s reading, or a copy of it whose coverage the caller has
// faded with the control it belongs to.
func Cast(gtx layout.Context, shadow tokens.DropShadow, focused bool, w layout.Widget) layout.Dimensions {
	// This is also the call outside the control's Clickable, so it is where
	// the focus band w drew is painted: the band straddles the control's own
	// box and a Clickable clips what it wraps to that box.
	dims := focus.Around(gtx, w)
	box := image.Rectangle{Max: dims.Size}
	radius := dims.Size.Y / 2
	cut, cutRadius := box, radius
	if focused {
		cut = focus.Footprint(gtx, box)
		cutRadius = radius + focus.OutsidePx(gtx)
	}
	macro := op.Record(gtx.Ops)
	control.DrawToolbarShadowAround(gtx, box, radius, cut, cutRadius, shadow)
	op.Defer(gtx.Ops, macro.Stop())
	return dims
}

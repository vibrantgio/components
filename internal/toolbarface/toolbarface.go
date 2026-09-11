// Package toolbarface holds the geometry components/picker's chrome-variant
// trigger is drawn from: the fill it tints under the pointer, the hairline
// around it, the focus ring that replaces that hairline, the density's height
// and padding, the pointer target's placement, and the chevron that says a
// menu opens below.
//
// It is internal because it is a seam and not a component: a caller reaches
// for picker.Toolbar or picker.RenderToolbar, and those document the control
// this draws.
package toolbarface

import (
	"image"
	"image/color"

	"gioui.org/f32"
	"gioui.org/font"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"

	vglayout "github.com/vibrantgio/components/layout"
	vgcolor "github.com/vibrantgio/theme/color"
	"github.com/vibrantgio/theme/tokens"
	"github.com/vibrantgio/theme/typeset"

	"github.com/vibrantgio/components/internal/focus"
	"github.com/vibrantgio/components/internal/surface"
)

// edgeDp is the rim's width — one hair at every density, the width every
// other derived edge in this library is drawn at (components/paragraph's
// chipEdge, components/input's field bezel). It is a width rather than a
// token because no scale in the system carries line weights.
const edgeDp = unit.Dp(1)

// The pull-down chevron's proportions, measured off the stored macOS reference
// (reference/macos/mail-window.png in the org's .github repository;
// window-bounded capture, macOS 26.5.2, dark appearance, one pixel per dp on
// that display). Both pull-down controls in Mail's toolbar — the folder one and
// the flag one — draw a chevron measuring 9 × 5 px inside a control
// 29 px tall, identical to the pixel, and the folder control's chevron ends
// 9 px inside the control's own trailing edge.
//
// So the chevron is a RATIO of the control's height, not a fixed size:
//
//	chevronWidthRatio  the mark's width, 9 of the control's 29
//	chevronAspect      the mark's height, 5 of its own 9
//
// which at this system's 24 dp comfortable control comes out at 7.4 × 4.1 dp.
const (
	chevronWidthRatio = 9.0 / 29.0
	chevronAspect     = 5.0 / 9.0
)

// chevronStroke is the mark's line weight. The platform reference measured its
// chevron band at ≈1.44 px at 16 pt from an offscreen render — the platform
// draws diagonals heavier than its axis-aligned strokes — so 1.5 dp is that
// measurement at the nearest weight this system draws.
const chevronStroke = unit.Dp(1.5)

// chevronWidth is the mark's column at density d, in pixels:
// the platform's ratio of the CONTROL's height, so the mark keeps the
// platform's proportion at every density rather than taking a line box the way
// an inline glyph does.
func chevronWidth(gtx layout.Context, d tokens.Density) int {
	return gtx.Dp(unit.Dp(d.ControlHeight * chevronWidthRatio))
}

// chevron paints the pull-down mark — one chevron pointing down — spanning box
// horizontally and centred in it vertically.
//
// It is STATIC. On the platform a pull-down button's chevron says "a menu opens
// below this" and never "this is open": the glyph does not flip when the menu
// stands, and a trigger that flipped one would be describing its own menu in a
// vocabulary the platform reserves for a disclosure triangle.
func chevron(gtx layout.Context, box image.Rectangle, col color.NRGBA) {
	w := float32(box.Dx())
	h := w * chevronAspect
	stroke := float32(gtx.Dp(chevronStroke))
	if stroke < 1 {
		stroke = 1
	}
	x0 := float32(box.Min.X)
	top := float32(box.Min.Y) + (float32(box.Dy())-h)/2

	var p clip.Path
	p.Begin(gtx.Ops)
	p.MoveTo(f32.Pt(x0, top))
	p.LineTo(f32.Pt(x0+w/2, top+h))
	p.LineTo(f32.Pt(x0+w, top))
	paint.FillShape(gtx.Ops, col, clip.Stroke{Path: p.End(), Width: stroke}.Op())
}

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

// Fill is what the control lays over the chrome it stands on: nothing at
// rest, the platform's hover overlay under the pointer, its press overlay
// while it is held, each flattened over chrome and opaque.
//
// A toolbar button is the one control on this platform that tints under the
// pointer — a push button, a list row and a sidebar row do not, which is the
// reading control-hover-{light,dark}.png records — so the overlay is applied
// here and nowhere else in this library.
//
// At rest the return is the zero value, which is no colour at all: the
// chrome shows through untouched rather than being repainted as itself.
func Fill(p tokens.PlatformColors, state tokens.State, chrome color.NRGBA) color.NRGBA {
	switch state {
	case tokens.StatePressed:
		return vgcolor.Flatten(p.PressOverlay, chrome)
	case tokens.StateHover:
		return vgcolor.Flatten(p.HoverOverlay, chrome)
	}
	return color.NRGBA{}
}

// Rim is the hairline around the control: the platform's separator flattened
// over beneath — the control's own tint where it carries one and the chrome
// otherwise — which is how the platform draws every hairline it draws.
func Rim(p tokens.PlatformColors, beneath color.NRGBA) color.NRGBA {
	return vgcolor.Flatten(p.Separator, beneath)
}

// Label is the colour the control's own wording reads in: the platform's
// control text, flattened over the fill the wording stands on.
func Label(p tokens.PlatformColors, beneath color.NRGBA) color.NRGBA {
	return vgcolor.Flatten(p.ControlText, beneath)
}

// Mark is the colour the chevron reads in: the platform's secondary label,
// which is what it draws a control's own marks in beside that control's
// wording, flattened over the fill the mark stands on.
func Mark(p tokens.PlatformColors, beneath color.NRGBA) color.NRGBA {
	return vgcolor.Flatten(p.SecondaryLabel, beneath)
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
// The whole of w is offset, slop and all, so the pointer target stays
// centred on the shape it was extended around.
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

// Draw paints the pull-down trigger: the hairline or the focus ring that
// replaces it, the fill it tints under the pointer, the label, and the
// chevron that says a menu opens below.
func Draw(
	gtx layout.Context,
	shaper *text.Shaper,
	label string,
	p tokens.PlatformColors,
	chrome color.NRGBA,
	sp tokens.SpacingScale,
	rad tokens.RadiusScale,
	labelStyle tokens.TextStyle,
	d tokens.Density,
	s State,
) layout.Dimensions {
	// Every one of the platform's names this control draws carries a
	// coverage, so each is flattened over the fill it actually lands on: the
	// chrome at rest, and the control's own tint once it has one.
	chrome = surface.Or(chrome, p.SidebarMaterial)
	fill := Fill(p, s.state(), chrome)
	beneath := chrome
	if fill.A != 0 {
		beneath = fill
	}
	labelForeground := Label(p, beneath)
	glyphForeground := Mark(p, beneath)

	padH := gtx.Dp(unit.Dp(d.PaddingX))
	padV := gtx.Dp(unit.Dp(d.PaddingY))
	minH := gtx.Dp(unit.Dp(d.ControlHeight))
	gap := gtx.Dp(unit.Dp(sp.S2))
	// The chevron is not an inline glyph and does not take the label's line
	// box: it is the platform's own ratio of the CONTROL's height, so the mark
	// keeps the platform's proportion at every density.
	mark := chevronWidth(gtx, d)

	// Record the label's material and its layout to learn its size before
	// anything is painted. typeset.Layout rather than widget.Label.Layout
	// because the role's line height has to be the height of the label box
	// and Gio alone reports the drawn glyph extent instead — see theme/typeset.
	mColor := op.Record(gtx.Ops)
	paint.ColorOp{Color: labelForeground}.Add(gtx.Ops)
	material := mColor.Stop()

	labelGtx := gtx
	labelGtx.Constraints.Min = image.Point{}
	if maxLabelW := gtx.Constraints.Max.X - 2*padH - gap - mark; maxLabelW > 0 {
		labelGtx.Constraints.Max.X = maxLabelW
	}
	mLabel := op.Record(gtx.Ops)
	labelDims := typeset.Layout(labelGtx, shaper,
		typeset.Label(labelStyle, 1), typeset.Font(labelStyle, font.Normal),
		unit.Sp(labelStyle.Size), label, material)
	labelCall := mLabel.Stop()

	// Sized to content, not to the width it was given: the control names a
	// choice, and one that stretched would be a banner.
	w := labelDims.Size.X + gap + mark + 2*padH
	h := max(labelDims.Size.Y+2*padV, minH)
	w = min(w, gtx.Constraints.Max.X)
	h = min(h, gtx.Constraints.Max.Y)
	size := image.Pt(w, h)
	box := image.Rectangle{Max: size}

	// The edge, as nested fills — the shape in the edge's colour, the fill
	// inset by one hair inside it — and not as a stroke on the shape's path. A
	// stroke is centred on its path, so half a hair of it would fall outside
	// the box this control reports and every pixel of it would be a blend of
	// the two colours rather than either.
	//
	// A focused control's edge IS the focus ring: the ring replaces the rim
	// rather than being drawn inside it. Drawn inside, the two make a
	// three-line sandwich — hairline, a pixel of fill, then the ring — which
	// reads as a dirty halo around the outline, the same "a band beside a
	// boundary reads as part of that boundary" that holds components/button's
	// ring clear of its edge. A button has no rim to collide with; this
	// control does, so it trades its one hair for the ring's two while the
	// ring is up. Nothing else moves: the shape measures the same box focused
	// as at rest, and the label does not shift.
	//
	// The corner is the scale's Md stop, the SAME one components/button reads
	// for every variant it draws: the platform draws its pop-up control as a
	// rounded rectangle, and the rounded rectangle this system already owns is
	// the button's. Reading the stop rather than naming a number is what keeps
	// the two in step if the scale ever moves.
	radius := gtx.Dp(unit.Dp(rad.Md))
	band, edgeColor := max(gtx.Dp(edgeDp), 1), Rim(p, beneath)
	if s.Focused {
		band, edgeColor = gtx.Dp(focus.Width), focus.Ring(p, beneath)
	}
	if maxRad := min(box.Dx(), box.Dy()) / 2; radius > maxRad {
		radius = maxRad
	}
	inner, innerRad := box, radius
	if in := box.Inset(band); in.Dx() > 0 && in.Dy() > 0 {
		inner, innerRad = in, max(radius-band, 0)
	}
	// The fill inside the edge's shape, and the edge as a band laid ON that
	// shape rather than as a shape beneath it: at rest there is no fill to
	// lay over the edge's shape at all, so a shape painted in the edge's
	// colour would carry that colour across the whole interior instead of
	// leaving it a hairline.
	//
	// The band is a stroke of twice the edge's width centred on the shape's
	// outline and clipped to that shape, which puts every pixel of it inside
	// the box this control reports: a stroke of the edge's own width would
	// fall half outside it. Drawn that way both of the band's sides follow
	// the corner, which four rectangles could not.
	if fill.A != 0 {
		paint.FillShape(gtx.Ops, fill, vglayout.Pill(gtx.Ops, inner, innerRad))
	}
	outer := clip.RRect{Rect: box, SE: radius, SW: radius, NE: radius, NW: radius}
	edgePath := outer.Path(gtx.Ops)
	edgeArea := outer.Push(gtx.Ops)
	paint.FillShape(gtx.Ops, edgeColor, clip.Stroke{Path: edgePath, Width: float32(2 * band)}.Op())
	edgeArea.Pop()

	// Label and mark on one centred row: the label leads, the mark follows it
	// across the S2 gap, and the pair is centred in what the padding leaves.
	content := labelDims.Size.X + gap + mark
	offX := max((w-content)/2, padH)
	lo := op.Offset(image.Pt(offX, (h-labelDims.Size.Y)/2)).Push(gtx.Ops)
	labelCall.Add(gtx.Ops)
	lo.Pop()

	// The chevron is handed the mark's column at the shape's full height and
	// centres itself in it, so its own drawn height stays the platform's ratio
	// rather than being stretched to a box.
	mo := op.Offset(image.Pt(offX+labelDims.Size.X+gap, 0)).Push(gtx.Ops)
	chevron(gtx, image.Rect(0, 0, mark, h), glyphForeground)
	mo.Pop()

	pointer.CursorPointer.Add(gtx.Ops)
	return layout.Dimensions{Size: size}
}

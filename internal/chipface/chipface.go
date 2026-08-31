// Package chipface holds the one geometry the chip family draws, so that its
// faces can live in the packages they belong to without either of them
// redrawing it: components/chip's pill and components/picker's pull-down
// anchor.
//
// One geometry means one answer to every question that is not the face's own —
// the measured fill, the state walk, the two-sided rim, the walked inks, the
// focus ring that replaces that rim, the density's height and padding, the
// pointer target's placement. [Face] names the two things that vary on top of
// it: which corner the widget takes and which mark it carries.
//
// It is internal because it is a seam between two published packages, not a
// component: a caller reaches for chip.Render or picker.RenderAnchor, and each
// of those documents the face it draws.
package chipface

import (
	"image"
	"image/color"
	"math"

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
)

// edgeDp is the rim's width — one hair at every density, the width every
// other derived edge in this library is drawn at (components/richtext's
// chipEdge, components/input's field bezel). It is a width rather than a
// token because no scale in the system carries line weights.
const edgeDp = unit.Dp(1)

// Face is which member of the family a widget draws: the corner it takes and
// the mark it carries. Every other answer is the geometry's and is shared.
type Face uint8

const (
	// FaceChip is the pill: the scale's Full radius and the caller's own
	// glyph. It is the zero value, so a widget that names no face is a chip.
	FaceChip Face = iota

	// FaceAnchor is the pull-down anchor: the same geometry at the button's
	// own rounded-rect radius, with the single down chevron drawn here
	// instead of a caller's glyph.
	//
	// The mark is a claim about placement — a menu opens BELOW this control —
	// so it holds only while the caller places the menu there. A trigger the
	// menu stands OVER wears a different mark and is therefore a different
	// face, not this one with a flag; picker's package doc carries what that
	// face would need.
	FaceAnchor
)

// radius is the face's corner, off the radius scale rather than a number.
//
// The chip takes the scale's Full stop — a pill, which is the chip's ruled
// identity. The anchor takes Md, the SAME stop components/button
// reads for every one of its registers: the anchor is the platform's pop-up
// control, the platform draws that control as a rounded rectangle rather than
// a capsule, and the rounded rectangle this system already owns is the
// button's. Deriving it here rather than picking a number is what keeps the
// two in step if the scale ever moves.
func (f Face) radius(rad tokens.RadiusScale) float32 {
	if f == FaceAnchor {
		return rad.Md
	}
	return rad.Full
}

// The pull-down chevron's proportions, measured off the stored macOS reference
// (reference/macos/mail-window.png in the org's .github repository;
// window-bounded capture, macOS 26.5.2, dark appearance, one pixel per dp on
// that display). Both pull-down controls in Mail's toolbar — the folder one and
// the flag one — draw a chevron whose ink measures 9 × 5 px inside a control
// 29 px tall, identical to the pixel, and the folder control's chevron ends
// 9 px inside the control's own trailing edge.
//
// So the chevron is a RATIO of the control's height, not a fixed size:
//
//	chevronWidthRatio  the ink's width, 9 of the control's 29
//	chevronAspect      the ink's height, 5 of its own 9
//
// which at this system's 36 dp comfortable control comes out at 11.2 × 6.2 dp.
const (
	chevronWidthRatio = 9.0 / 29.0
	chevronAspect     = 5.0 / 9.0
)

// chevronStroke is the mark's line weight. The platform reference measured its
// chevron band at ≈1.44 px at 16 pt from an offscreen render — the platform
// draws diagonals heavier than its axis-aligned strokes — so 1.5 dp is that
// measurement at the nearest weight this system draws.
const chevronStroke = unit.Dp(1.5)

// chevronWidth is the mark's column for [FaceAnchor] at density d, in pixels:
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
// stands, and an anchor that flipped one would be describing its own menu in a
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

// Glyph is the painter a face draws its mark with: it fills a sizePx×sizePx
// box at the current origin in colour col. It is the same signature
// components/button gives an icon-only button and the same one
// components/icon's registry hands out, so a named glyph, a clip.Path drawn
// by hand and a chevron built for one screen are interchangeable here.
//
// A nil Glyph draws no mark; the geometry loses the mark and the gap before it
// and nothing else. [FaceAnchor] ignores it — its mark is the pull-down
// chevron.
type Glyph func(gtx layout.Context, sizePx int, col color.NRGBA)

// State is the explicit visual state a static render draws in. The zero value
// is a resting widget on the window ground.
type State struct {
	// Ground is the elevation storey of the surface hosting the widget, in
	// the same vocabulary the host names its own fill (tokens.SurfaceAt). It
	// is the input to every colour resolved here: the fill is the measured
	// step over this storey, and the rim is the neutral rung that clears the
	// graphic floor against both sides of the edge. A dialog at
	// tokens.Level2 passes Level2. The zero value is tokens.Level0, the
	// window ground.
	Ground tokens.ElevationLevel

	Hovered bool
	Pressed bool
	Focused bool
}

// state is the token vocabulary's name for the interaction the widget is in.
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

// darkFillStep is how far over its ground a resting face stands where the
// Background pin is the darkest surface the neutral ramp carries, in CIELAB
// L\*. It is a MEASUREMENT of the platform, not a derivation: 1.28 L\*, the
// step macOS takes between a unified toolbar's band and the pop-up capsules
// drawn on it — this family's exact role. components/chip's package doc quotes
// the capture, the two fills and the method.
const darkFillStep = 1.28

// lightFillStep is the same step where the pin is the lightest surface the
// ramp carries, in CIELAB L\*. It is a DERIVATION and components/chip's doc
// says so: the stored macOS reference holds no light-appearance capture to
// measure, so this half takes the ladder's own first storey over the paper —
// the 0.70 L\* the light scheme already spends on Level1 over Level0, spent
// identically over every ground rather than growing with the ground's position
// on the ladder.
const lightFillStep = 0.70

// fillStep is how far above its ground a resting fill stands, in CIELAB L\*.
// One number per scheme, and the scheme is never named: which half applies is
// read off the neutral surface band's direction, exactly as theme/tokens reads
// it for the floor's own two measurements.
//
// A scheme whose band climbs away from its 100 stop has its pin as the
// darkest surface the ramp carries — the dark scheme — and takes the
// platform's measured capsule step. One whose band descends has the pin as
// its lightest surface and almost no room above it, and takes the derived
// whisper.
func fillStep(c tokens.ColorTokens) float64 {
	pin, _, _ := vgcolor.LabFromNRGBA(c.Background)
	top := math.Inf(-1)
	for i := 1; i <= 4; i++ {
		l, _, _ := vgcolor.LabFromNRGBA(c.Ramps.Neutral.Step(i * 100))
		if l > top {
			top = l
		}
	}
	if top > pin {
		return darkFillStep
	}
	return lightFillStep
}

// restFill is the fill at rest: the surface the widget stands on, lifted by
// the measured step for its scheme, realized at the ground's own hue and
// chroma so the shape carries whatever tint the ladder carries and none of
// its own. Nothing is mixed and no colour is named — the step is a depth in
// L\* and the palette renders it, the way theme/tokens realizes a storey.
//
// It is a step over the LOCAL GROUND rather than a walk to the next storey.
// The storey above is correct as depth and wrong as loudness: in the dark
// scheme it stands 10.0 luminance over the window's paper where the platform's
// own toolbar capsules stand 2.65 over their band — a filled block at four
// times the platform's step, in the one role the platform draws as a
// near-hairline outline. The rim carries the edge; the fill does not shout
// under it.
func restFill(c tokens.ColorTokens, ground tokens.ElevationLevel) color.NRGBA {
	base := c.SurfaceAt(ground)
	l, _, _ := vgcolor.LabFromNRGBA(base)
	target := min(l+fillStep(c), 100)
	_, chroma, hue := vgcolor.OKLChFromNRGBA(base)
	return vgcolor.NRGBAFromToneChromaHue(target, chroma, hue)
}

// Fill is the family's ground: the measured step over the surface it stands
// on, walked by the interaction state.
//
// The walk is [tokens.ColorTokens.PinnedStateColor] — the same walk
// [tokens.ColorTokens.StateAt] takes from a storey, taken from the resting
// fill instead, because that fill is not a storey. Hover and press therefore
// follow the rest automatically and their stride is untouched.
func Fill(c tokens.ColorTokens, ground tokens.ElevationLevel, state tokens.State) color.NRGBA {
	return c.PinnedStateColor(restFill(c, ground), state)
}

// Rim is the edge, and whether there is one: the rung of the neutral ramp
// that reaches the graphic floor against BOTH of the edge's neighbours — the
// ground outside it and the fill inside it — or no rim at all when no rung can
// reach both.
//
// An edge has two sides and one colour, so a walk aimed at one side is a
// promise about the other. components/input's control border aims at the
// ground and gets away with it because a field's interior never moves; this
// family's does, one and two rungs under the pointer, and in the dark scheme
// those rungs are long. Aimed at the ground alone, the rim lands ON the
// pressed fill at level 1 — 1.00:1, the same colour twice — and aimed at the
// fill alone it vanishes into the ground at rest in the light scheme, where
// the storey step is 1.02:1 and the rim is the only thing there is. So both
// candidates are derived and the one that clears both sides is kept.
//
// When neither clears both, the two neighbours are further apart than twice
// the floor and no colour on any ramp could sit between them — which is
// exactly the case where no rim is needed, because a fill that far off its
// ground is carrying its own edge. That is this library's outline ruling in
// the elevation ladder's vocabulary: a fill that separates on its own needs no
// outline, and a fill that cannot never will. So the second return is false
// there and the caller draws the shape without one.
func Rim(c tokens.ColorTokens, ground tokens.ElevationLevel, state tokens.State) (color.NRGBA, bool) {
	below := c.SurfaceAt(ground)
	above := Fill(c, ground, state)
	for _, cand := range [...]color.NRGBA{
		c.MarkOn(tokens.RoleNeutral, below, tokens.GraphicFloor),
		c.MarkOn(tokens.RoleNeutral, above, tokens.GraphicFloor),
	} {
		if vgcolor.ContrastRatio(cand, below) >= tokens.GraphicFloor &&
			vgcolor.ContrastRatio(cand, above) >= tokens.GraphicFloor {
			return cand, true
		}
	}
	return color.NRGBA{}, false
}

// Ink is the colour something reads in when it is drawn on one of these
// fills: the Text pin while that pin clears floor against the fill, and
// otherwise the rung of the neutral ramp nearest its mid-value step that does.
//
// That is tokens.ColorTokens.InkOn's own rule, applied to the one role InkOn
// refuses. InkOn asks a role for its pinned base and RoleNeutral has none —
// the neutral ink's pin is the Text pin, which is derived against the
// Background pin already — so the rule is spelled out here rather than
// reinvented: pin first, walk only when the pin stops reading.
//
// Pass tokens.TextFloor for a label and tokens.GraphicFloor for a mark.
func Ink(c tokens.ColorTokens, fill color.NRGBA, floor float64) color.NRGBA {
	if vgcolor.ContrastRatio(c.Text, fill) >= floor {
		return c.Text
	}
	return c.MarkOn(tokens.RoleNeutral, fill, floor)
}

// Pin is the edge of the box a widget is offered that its shape is pinned to.
//
// It is a placement, not a stretch: the shape stays sized to its content and
// what changes is where in the offered box it is drawn and how much of that
// box the widget reports having used. Only the horizontal axis is pinned,
// because the vertical one is already settled by whatever row the widget
// stands in.
//
// The seam exists because a widget alone can be placed by its container and
// one handed on to a container that centres whatever it is given cannot: the
// reserved cap and the drawn shape then part company by half the slack, and
// the only place both widths are known is inside the widget. A pin says it
// there, once.
//
// It costs the container the drawn rect, which is the whole box as far as it
// can tell, so say it only where nothing upstream needs that rect. A
// container that aligns what it is given needs no pin at all, and a pinned
// widget would leave it aiming at a box nothing was drawn in.
type Pin uint8

const (
	// PinNone is the zero value: the widget reports the shape it drew and no
	// more, so a row of them is laid out at their own scale and the box
	// around them is the container's business.
	PinNone Pin = iota

	// PinLeading draws the shape at the leading edge of the offered box.
	PinLeading

	// PinTrailing draws the shape at the trailing edge of the offered box.
	PinTrailing
)

// Layout draws w at p's edge of the box the widget was offered — the
// horizontal half of gtx.Constraints.Max — and reports that box rather than
// w's own size, which is what lets a caller upstream find the pinned edge
// where it asked for it. PinNone lays w out untouched, so a widget that pins
// nothing pays nothing.
//
// The whole widget is offset, slop and all, so the pointer target stays
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

// Draw paints one member of the family. face selects which: the chip takes the
// pill's corner and the caller's own glyph, the anchor the button's corner and
// the pull-down chevron it draws itself. Everything else, geometry and colour
// alike, is shared — both walk their fill, wear the ring and ask for the
// cursor.
func Draw(
	gtx layout.Context,
	shaper *text.Shaper,
	label string,
	glyph Glyph,
	c tokens.ColorTokens,
	sp tokens.SpacingScale,
	rad tokens.RadiusScale,
	labelStyle tokens.TextStyle,
	d tokens.Density,
	s State,
	face Face,
) layout.Dimensions {
	st := s.state()
	fill := Fill(c, s.Ground, st)
	rim, rimmed := Rim(c, s.Ground, st)
	labelInk := Ink(c, fill, tokens.TextFloor)
	glyphInk := Ink(c, fill, tokens.GraphicFloor)

	padH := gtx.Dp(unit.Dp(d.PaddingX))
	padV := gtx.Dp(unit.Dp(d.PaddingY))
	minH := gtx.Dp(unit.Dp(d.ControlHeight))
	gap := gtx.Dp(unit.Dp(sp.S2))
	// The glyph is the label's own line box — see components/chip's package
	// doc for why that is the same number components/icon answers at each
	// density's own role. The anchor's chevron is not a glyph and does not
	// take the line box: it is the platform's own ratio of the CONTROL's
	// height, so the mark keeps the platform's proportion at every density.
	mark := 0
	switch {
	case face == FaceAnchor:
		mark = chevronWidth(gtx, d)
	case glyph != nil:
		mark = gtx.Dp(unit.Dp(labelStyle.LineHeight))
	}
	if mark == 0 {
		gap = 0
	}

	// Record the label's material and its layout to learn its size before
	// anything is painted. typeset.Layout rather than widget.Label.Layout
	// because the role's line height has to be the height of the label box
	// and Gio alone reports the glyph ink instead — see theme/typeset.
	mColor := op.Record(gtx.Ops)
	paint.ColorOp{Color: labelInk}.Add(gtx.Ops)
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

	// Sized to content, not to the width it was given: a chip is a summary of
	// something, and a summary that stretches is a banner.
	w := labelDims.Size.X + gap + mark + 2*padH
	h := max(labelDims.Size.Y+2*padV, minH)
	w = min(w, gtx.Constraints.Max.X)
	h = min(h, gtx.Constraints.Max.Y)
	size := image.Pt(w, h)
	box := image.Rectangle{Max: size}

	// The edge, as nested fills — the shape in the edge's colour, the fill
	// inset by one hair inside it — and not as a stroke on the shape's path. A
	// stroke is centred on its path, so half a hair of it would fall outside
	// the box the widget reports and every pixel of it would be a blend of the
	// two colours rather than either.
	//
	// A focused widget's edge IS the focus ring: the ring replaces the rim
	// rather than being drawn inside it. Drawn inside, the two make a
	// three-line sandwich — hairline, a pixel of fill, then the ring — which
	// reads as a dirty halo around the outline, the same "a band beside a
	// boundary reads as part of that boundary" that holds components/button's
	// ring clear of its edge. A button has no rim to collide with; this family
	// does, so it trades its one hair for the ring's two while the ring is up.
	// Nothing else moves: the shape measures the same box focused as at rest,
	// and the label does not shift.
	radius := gtx.Dp(unit.Dp(face.radius(rad)))
	band, edgeInk, edged := max(gtx.Dp(edgeDp), 1), rim, rimmed
	if s.Focused {
		band, edgeInk, edged = gtx.Dp(focus.Width), focus.Ring(c, fill), true
	}
	inner, innerRad := box, radius
	if edged {
		paint.FillShape(gtx.Ops, edgeInk, vglayout.Pill(gtx.Ops, box, radius))
		if in := box.Inset(band); in.Dx() > 0 && in.Dy() > 0 {
			inner, innerRad = in, max(radius-band, 0)
		}
	}
	paint.FillShape(gtx.Ops, fill, vglayout.Pill(gtx.Ops, inner, innerRad))

	// Label and mark on one centred row: the label leads, the mark follows it
	// across the S2 gap, and the pair is centred in what the padding leaves.
	content := labelDims.Size.X + gap + mark
	offX := max((w-content)/2, padH)
	lo := op.Offset(image.Pt(offX, (h-labelDims.Size.Y)/2)).Push(gtx.Ops)
	labelCall.Add(gtx.Ops)
	lo.Pop()

	switch {
	case mark == 0:
	case face == FaceAnchor:
		// The chevron is handed the mark's column at the shape's full height
		// and centres itself in it, so its own ink height stays the platform's
		// ratio rather than being stretched to a box.
		mo := op.Offset(image.Pt(offX+labelDims.Size.X+gap, 0)).Push(gtx.Ops)
		chevron(gtx, image.Rect(0, 0, mark, h), glyphInk)
		mo.Pop()
	case glyph != nil:
		mo := op.Offset(image.Pt(offX+labelDims.Size.X+gap, (h-mark)/2)).Push(gtx.Ops)
		glyph(gtx, mark, glyphInk)
		mo.Pop()
	}

	pointer.CursorPointer.Add(gtx.Ops)
	return layout.Dimensions{Size: size}
}

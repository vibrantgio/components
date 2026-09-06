// Package toolbarface holds the geometry components/picker's chrome-variant
// trigger is drawn from: the measured fill, the state walk, the two-sided rim,
// the walked foregrounds, the focus ring that replaces that rim, the density's
// height and padding, the pointer target's placement, and the chevron that says a
// menu opens below.
//
// It is internal because it is a seam and not a component: a caller reaches
// for picker.Toolbar or picker.RenderToolbar, and those document the control
// this draws.
package toolbarface

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
// is a resting control on the window's own surface.
type State struct {
	// Level is the level of the surface the control stands on — the control
	// has no level of its own — in the same vocabulary the host names its own fill
	// (tokens.SurfaceAt). It is the input to every colour resolved here: the
	// fill is the measured step over that surface, and the rim is the neutral
	// step that clears the graphic floor against both sides of the edge. A
	// dialog at tokens.Level2 passes Level2. The zero value is tokens.Level0,
	// the window's own surface.
	Level tokens.ElevationLevel

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

// darkFillStep is how far over the surface it stands on a resting trigger
// sits where the Background pin is the darkest surface the neutral ramp
// carries, in CIELAB
// L\*. It is a MEASUREMENT of the platform, not a derivation: the step macOS
// takes between a unified toolbar's band and the pop-up capsules drawn on it,
// which is this control's exact role. From the stored macOS reference
// (reference/macos/mail-window.png in the org's .github repository;
// window-bounded capture, macOS 26.5.2, dark appearance):
//
//	Mail's unified toolbar band          #232A2E   L* 16.555   luminance 40.80
//	its pop-up capsules on that band     #242D32   L* 17.837   luminance 43.45
//	                                               step 1.28   step +2.65
const darkFillStep = 1.28

// lightFillStep is the same step where the pin is the lightest surface the
// ramp carries, in CIELAB L\*. It is a DERIVATION and not a measurement: the
// stored macOS reference holds no light-appearance capture, so this half takes
// the first level over the content — the 0.70 L\* the light scheme
// already spends on Level1 over Level0 — spent identically over every surface
// rather than growing with that surface's own level. The light
// scheme has 3.12 L\* in total between its content and the tonal axis and spends
// all three levels inside it, so the platform's 1.28 would put one control
// above where a dialog sits.
const lightFillStep = 0.70

// fillStep is how far above the surface beneath a resting fill stands, in
// CIELAB L\*.
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

// restFill is the fill at rest: the surface the control stands on, lifted by
// the measured step for its scheme, realized at that surface's own hue and
// chroma so the shape carries whatever tint the levels carry and none of
// its own. Nothing is mixed and no colour is named — the step is a depth in
// L\* and the palette renders it, the way theme/tokens realizes a level.
//
// It is a step over the surface it stands on rather than a walk to the next
// level. The level above is correct as depth and too pronounced: in the dark
// scheme it stands 10.0 luminance over the window's content where the platform's
// own toolbar capsules stand 2.65 over their band — a filled block at four
// times the platform's step, in the one role the platform draws as a
// near-hairline outline. The rim carries the edge; the fill does not claim
// attention under it.
func restFill(c tokens.ColorTokens, level tokens.ElevationLevel) color.NRGBA {
	base := c.SurfaceAt(level)
	l, _, _ := vgcolor.LabFromNRGBA(base)
	target := min(l+fillStep(c), 100)
	_, chroma, hue := vgcolor.OKLChFromNRGBA(base)
	return vgcolor.NRGBAFromToneChromaHue(target, chroma, hue)
}

// Fill is the control's own fill: the measured step over the surface it stands
// on, walked by the interaction state and stopped short of any depth its own
// label could not be read on.
//
// The walk is [tokens.ColorTokens.PinnedStateColor] — the same walk
// [tokens.ColorTokens.StateAt] takes from a level, taken from the resting
// fill instead, because that fill is not a level. Hover and press therefore
// follow the rest automatically and their stride is untouched.
//
// The stopping is the label's, and it binds where the ramp's own two ends are
// too close together to write on its middle. A ramp writes with its ends, so
// between them lies a band of depths no step of it reaches tokens.TextFloor
// against — for the dark ramp, L\* 46.0 to 53.8 — and a control standing high
// among the levels walks into it: pressed on a level-2 plane the walk
// lands at 48.1 and hovered on a level-3 one at 47.6, where the best
// foreground the palette carries measures 4.09:1 and 4.21:1. A fill nothing
// can be written on is not a state to walk to, so the walk stops at the last depth on its way
// that the palette can still write on. Both fills come to rest at 45.7 with
// their label at 4.51:1, and nothing else on either scheme's levels moves: the
// light ramp's ends are a near-black and a near-white, its band lies at L\* 47
// to 52, and the deepest fill this family walks to in that scheme is 75.5.
func Fill(c tokens.ColorTokens, level tokens.ElevationLevel, state tokens.State) color.NRGBA {
	rest := restFill(c, level)
	return legible(c, rest, c.PinnedStateColor(rest, state))
}

// legible is the walk's stop: walked itself while the palette can write a
// label on it, and otherwise the last depth between rest and walked that it
// can, at walked's own hue and chroma.
//
// The depth is found by measuring the realized tone rather than by solving for
// the band's edge, because a tone is realized in 8-bit sRGB and a depth solved
// exactly on the edge rounds to either side of it; halving the interval keeps
// the answer on the side that measured legible. A rest fill that cannot be
// written on has no such side and is left alone — that is a defect in the
// resting appearance and belongs to the gates, not to a state walk.
func legible(c tokens.ColorTokens, rest, walked color.NRGBA) color.NRGBA {
	if writable(c, walked) || !writable(c, rest) {
		return walked
	}
	restL, _, _ := vgcolor.LabFromNRGBA(rest)
	walkedL, _, _ := vgcolor.LabFromNRGBA(walked)
	_, chroma, hue := vgcolor.OKLChFromNRGBA(walked)
	// lo is always a depth that measured legible, hi one that did not.
	lo, hi := restL, walkedL
	for range 24 {
		mid := (lo + hi) / 2
		if writable(c, vgcolor.NRGBAFromToneChromaHue(mid, chroma, hue)) {
			lo = mid
		} else {
			hi = mid
		}
	}
	return vgcolor.NRGBAFromToneChromaHue(lo, chroma, hue)
}

// writable reports whether the foreground this family would write a label in
// reaches its floor on fill — `Foreground`'s own answer, measured, since
// `Foreground` hands back the best-reading step when no step reaches the
// floor at all.
func writable(c tokens.ColorTokens, fill color.NRGBA) bool {
	return vgcolor.ContrastRatio(Foreground(c, fill, tokens.TextFloor), fill) >= tokens.TextFloor
}

// Rim is the edge, and whether there is one: the step of the neutral ramp
// that reaches the graphic floor against BOTH of the edge's neighbours — the
// surface outside it and the fill inside it — or no rim at all when no step
// can reach both.
//
// An edge has two sides and one colour, so a walk aimed at one side is a
// promise about the other, and this family's inner side moves besides — one
// and two steps under the pointer, and in the dark scheme those steps are
// long. Aimed at the surface alone, the rim lands ON the pressed fill at level
// 1 — 1.00:1, the same colour twice — and aimed at the fill alone it comes
// too close to that surface at rest in the light scheme, where the raise off
// the content measures 1.13:1 and the rim is the only thing there is. So both candidates are derived and
// the one that clears both sides is kept, which is the rule every two-sided
// edge in this library takes (components/internal/control's border, the focus
// ring that replaces this rim).
//
// When neither clears both, the two neighbours are further apart than twice
// the floor and no colour on any ramp could sit between them — which is
// exactly the case where no rim is needed, because a fill that far off its
// surface is carrying its own edge. That is this library's outline ruling in
// the elevation levels' vocabulary: a fill that separates on its own needs no
// outline, and a fill that cannot never will. So the second return is false
// there and the caller draws the shape without one.
func Rim(c tokens.ColorTokens, level tokens.ElevationLevel, state tokens.State) (color.NRGBA, bool) {
	below := c.SurfaceAt(level)
	above := Fill(c, level, state)
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

// `Foreground` is the colour something reads in when it is drawn on one of these
// fills: the Text pin while that pin clears floor against the fill, and
// otherwise the step of the neutral ramp nearest its mid-value that does.
//
// That is tokens.ColorTokens.ForegroundOnAtFloor's own rule, applied to the
// one role ForegroundOnAtFloor refuses. ForegroundOnAtFloor asks a role for
// its pinned base and RoleNeutral has none — the neutral foreground's pin
// is the Text pin, which is derived against the Background pin already — so
// the rule is spelled out here rather than reinvented: pin first, walk only
// when the pin stops reading.
//
// Pass tokens.TextFloor for a label and tokens.GraphicFloor for a mark.
func Foreground(c tokens.ColorTokens, fill color.NRGBA, floor float64) color.NRGBA {
	if vgcolor.ContrastRatio(c.Text, fill) >= floor {
		return c.Text
	}
	return c.MarkOn(tokens.RoleNeutral, fill, floor)
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

// Draw paints the pull-down trigger: the walked fill, the two-sided rim or the
// focus ring that replaces it, the label, and the chevron that says a menu
// opens below.
func Draw(
	gtx layout.Context,
	shaper *text.Shaper,
	label string,
	c tokens.ColorTokens,
	sp tokens.SpacingScale,
	rad tokens.RadiusScale,
	labelStyle tokens.TextStyle,
	d tokens.Density,
	s State,
) layout.Dimensions {
	st := s.state()
	fill := Fill(c, s.Level, st)
	rim, rimmed := Rim(c, s.Level, st)
	labelForeground := Foreground(c, fill, tokens.TextFloor)
	glyphForeground := Foreground(c, fill, tokens.GraphicFloor)

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
	band, edgeColor, edged := max(gtx.Dp(edgeDp), 1), rim, rimmed
	if s.Focused {
		band, edgeColor, edged = gtx.Dp(focus.Width), focus.Ring(c), true
	}
	inner, innerRad := box, radius
	if edged {
		paint.FillShape(gtx.Ops, edgeColor, vglayout.Pill(gtx.Ops, box, radius))
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

	// The chevron is handed the mark's column at the shape's full height and
	// centres itself in it, so its own drawn height stays the platform's ratio
	// rather than being stretched to a box.
	mo := op.Offset(image.Pt(offX+labelDims.Size.X+gap, 0)).Push(gtx.Ops)
	chevron(gtx, image.Rect(0, 0, mark, h), glyphForeground)
	mo.Pop()

	pointer.CursorPointer.Add(gtx.Ops)
	return layout.Dimensions{Size: size}
}

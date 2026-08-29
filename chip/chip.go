package chip

import (
	"image"
	"image/color"

	"gioui.org/font"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
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

// Glyph is the painter a chip draws its mark with: it fills a sizePx×sizePx
// box at the current origin in colour col. It is the same signature
// components/button gives an icon-only button and the same one
// components/icon's registry hands out, so a named glyph, a clip.Path drawn
// by hand and a chevron built for one screen are interchangeable here.
//
// A nil Glyph draws no mark and the chip is label-only; its geometry loses
// the mark and the gap before it and nothing else.
type Glyph func(gtx layout.Context, sizePx int, col color.NRGBA)

// RenderState holds the explicit visual state a static chip render draws in.
// The zero value is a resting chip on the window ground, so RenderState{} is
// the default chip.
//
// Intended for golden-image testing and static rendering; production code
// obtains the interaction half from the Gio event system.
type RenderState struct {
	// Ground is the elevation storey of the surface hosting the chip, in the
	// same vocabulary the host names its own fill (tokens.SurfaceAt). It is
	// the input to every colour the chip resolves: the fill is the storey one
	// rung nearer the viewer than this one, and the rim is the neutral rung
	// that clears the graphic floor against both. A dialog at tokens.Level2 passes Level2 and
	// its chips fill at Level3. The zero value is tokens.Level0, the window
	// ground.
	Ground tokens.ElevationLevel

	Hovered bool
	Pressed bool
	Focused bool
}

// state is the token vocabulary's name for the interaction the chip is in.
// Press wins over hover, because a pressed chip is under the pointer by
// definition and the deeper walk is the one that has something to say.
func (s RenderState) state() tokens.State {
	switch {
	case s.Pressed:
		return tokens.StatePressed
	case s.Hovered:
		return tokens.StateHover
	}
	return tokens.StateNormal
}

// Fill is the chip's ground: the storey one rung nearer the viewer than the
// one it stands on, walked by the interaction state.
//
// It is exported because a container that draws behind or beside a chip — a
// header band deciding what its own seam should clear, a test measuring the
// pill — needs the same answer the chip drew with, and re-deriving it at the
// call site is how two answers appear.
func Fill(c tokens.ColorTokens, ground tokens.ElevationLevel, state tokens.State) color.NRGBA {
	return c.StateAt(ground.Raised(), state)
}

// Rim is the chip's edge, and whether it has one: the rung of the neutral ramp
// that reaches the graphic floor against BOTH of the edge's neighbours — the
// ground outside it and the chip's own fill inside it — or no rim at all when
// no rung can reach both.
//
// An edge has two sides and one colour, so a walk aimed at one side is a
// promise about the other. components/input's control border aims at the
// ground and gets away with it because a field's interior never moves; a
// chip's does, one and two rungs under the pointer, and in the dark scheme
// those rungs are long. Aimed at the ground alone, the rim landed ON the
// pressed fill at level 1 — 1.00:1, the same colour twice — and aimed at the
// fill alone it would vanish into the ground at rest in the light scheme,
// where the storey step is 1.02:1 and the rim is the only thing there is. So
// both candidates are derived and the one that clears both sides is kept.
//
// When neither clears both, the two neighbours are further apart than twice
// the floor and no colour on any ramp could sit between them — which is
// exactly the case where the chip needs no rim, because a fill that far off
// its ground is carrying its own edge. That is patterns/tag's ruling in the
// elevation ladder's vocabulary: a fill that separates on its own needs no
// outline, and a fill that cannot never will. So the second return is false
// there and the caller draws the pill without one, which is what a pressed
// chip on a dark dialog looks like: a solid block under the finger.
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

// Ink is the colour something reads in when it is drawn on a chip's fill: the
// Text pin while that pin clears floor against fill, and otherwise the rung of
// the neutral ramp nearest its mid-value step that does.
//
// That is tokens.ColorTokens.InkOn's own rule, applied to the one role InkOn
// refuses. InkOn asks a role for its pinned base and RoleNeutral has none —
// the neutral ink's pin is the Text pin, which is derived against the
// Background pin already — so the rule is spelled out here rather than
// reinvented: pin first, walk only when the pin stops reading.
//
// Pass tokens.TextFloor for the label and tokens.GraphicFloor for the glyph.
func Ink(c tokens.ColorTokens, fill color.NRGBA, floor float64) color.NRGBA {
	if vgcolor.ContrastRatio(c.Text, fill) >= floor {
		return c.Text
	}
	return c.MarkOn(tokens.RoleNeutral, fill, floor)
}

// Render produces a layout.Widget drawing the interactive chip face in an
// explicit visual state, without event processing: the pill filled one storey
// over s.Ground and walked by the pointer, its one-dp rim, the label in the
// ink that clears the text floor on that fill, and the glyph in the ink that
// clears the graphic floor. When s.Focused, the focus ring — measured against
// that fill — takes the rim's place at the chip's edge, two dp instead of one.
//
// glyph may be nil, in which case the chip is label-only. labelStyle is the
// whole text style the label is set in; pass tokens.DefaultTypography.LabelLarge
// with tokens.Comfortable for the default desktop chip, or LabelMedium with
// tokens.Compact for the dense one (the package doc's table has the four
// combinations and the heights they draw at).
//
// The chip is sized to its content, clamped to the constraints it is handed,
// and asks for the pointer cursor. Extending its pointer area to
// tokens.MinHitTarget is the live path's job — see the package doc.
func Render(
	shaper *text.Shaper,
	label string,
	glyph Glyph,
	colors tokens.ColorTokens,
	sp tokens.SpacingScale,
	rad tokens.RadiusScale,
	labelStyle tokens.TextStyle,
	d tokens.Density,
	s RenderState,
) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		return draw(gtx, shaper, label, glyph, colors, sp, rad, labelStyle, d, s, true)
	}
}

// RenderBadge produces a layout.Widget drawing the non-interactive chip face:
// the same pill, the same rim, the same inks and the same geometry, held at
// rest. It takes a ground rather than a RenderState because there is no state
// to take — a badge does not hover, does not press, does not focus and does
// not ask for the pointer cursor.
//
// It is for a mark that keeps a fill: a count beside a heading, a build's
// status in a toolbar, a label that says what a pane is showing. Something a
// reader can click is a chip and takes [Render]; something a reader can only
// read is a badge and takes this, so the pointer never changes over a thing
// that would not have answered.
func RenderBadge(
	shaper *text.Shaper,
	label string,
	glyph Glyph,
	colors tokens.ColorTokens,
	sp tokens.SpacingScale,
	rad tokens.RadiusScale,
	labelStyle tokens.TextStyle,
	d tokens.Density,
	ground tokens.ElevationLevel,
) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		return draw(gtx, shaper, label, glyph, colors, sp, rad, labelStyle, d,
			RenderState{Ground: ground}, false)
	}
}

// draw paints one chip. interactive selects the face: true draws the state
// walk, the focus ring and the pointer cursor, false draws the badge — the
// identical pill with none of the three. Everything else, geometry and colour
// alike, is shared, which is what "one geometry, two faces" means in code.
func draw(
	gtx layout.Context,
	shaper *text.Shaper,
	label string,
	glyph Glyph,
	c tokens.ColorTokens,
	sp tokens.SpacingScale,
	rad tokens.RadiusScale,
	labelStyle tokens.TextStyle,
	d tokens.Density,
	s RenderState,
	interactive bool,
) layout.Dimensions {
	st := s.state()
	if !interactive {
		st = tokens.StateNormal
	}
	fill := Fill(c, s.Ground, st)
	rim, rimmed := Rim(c, s.Ground, st)
	labelInk := Ink(c, fill, tokens.TextFloor)
	glyphInk := Ink(c, fill, tokens.GraphicFloor)

	padH := gtx.Dp(unit.Dp(d.PaddingX))
	padV := gtx.Dp(unit.Dp(d.PaddingY))
	minH := gtx.Dp(unit.Dp(d.ControlHeight))
	gap := gtx.Dp(unit.Dp(sp.S2))
	// The glyph is the label's own line box — see the package doc for why
	// that is the same number components/icon answers at each density's own
	// role.
	mark := 0
	if glyph != nil {
		mark = gtx.Dp(unit.Dp(labelStyle.LineHeight))
	} else {
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
	h := labelDims.Size.Y + 2*padV
	if h < minH {
		h = minH
	}
	if w > gtx.Constraints.Max.X {
		w = gtx.Constraints.Max.X
	}
	if h > gtx.Constraints.Max.Y {
		h = gtx.Constraints.Max.Y
	}
	size := image.Pt(w, h)
	box := image.Rectangle{Max: size}

	// The rim as nested fills — the pill in the rim's colour, the fill inset
	// by one hair inside it — and not as a stroke on the pill's path. A
	// stroke is centred on its path, so half a hair of it would fall outside
	// the box the chip reports and every pixel of it would be a blend of the
	// two colours rather than either.
	// The edge, as nested fills — the pill in the edge's colour, the fill
	// inset inside it — and not as a stroke on the pill's path. A stroke is
	// centred on its path, so half its width would fall outside the box the
	// chip reports and every pixel of it would be a blend of the two colours
	// rather than either.
	//
	// A focused chip's edge IS the focus ring: the ring replaces the rim
	// rather than being drawn inside it. Drawn inside, the two made a
	// three-line sandwich — hairline, a pixel of fill, then the ring — which
	// a reviewer handed the rendering called a dirty halo around a purple
	// outline before naming anything else, and which is the same "a band
	// beside a boundary reads as part of that boundary" that put
	// components/button's ring clear of its edge. A button has no rim to
	// collide with; a chip does, so the chip trades its one hair for the
	// ring's two while the ring is up. Nothing else moves: the pill measures
	// the same box focused as at rest, and the label does not shift.
	radius := gtx.Dp(unit.Dp(rad.Full))
	focused := interactive && s.Focused
	band, edgeInk, edged := max(gtx.Dp(edgeDp), 1), rim, rimmed
	if focused {
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
	offX := (w - content) / 2
	if offX < padH {
		offX = padH
	}
	lo := op.Offset(image.Pt(offX, (h-labelDims.Size.Y)/2)).Push(gtx.Ops)
	labelCall.Add(gtx.Ops)
	lo.Pop()

	if glyph != nil && mark > 0 {
		mo := op.Offset(image.Pt(offX+labelDims.Size.X+gap, (h-mark)/2)).Push(gtx.Ops)
		glyph(gtx, mark, glyphInk)
		mo.Pop()
	}

	if interactive {
		pointer.CursorPointer.Add(gtx.Ops)
	}
	return layout.Dimensions{Size: size}
}

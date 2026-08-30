package chip

import (
	"image"
	"image/color"
	"math"

	"gioui.org/font"
	"gioui.org/io/pointer"
	"gioui.org/io/semantic"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"

	"github.com/reactivego/rx"

	vglayout "github.com/vibrantgio/components/layout"
	"github.com/vibrantgio/mvu"
	vgcolor "github.com/vibrantgio/theme/color"
	"github.com/vibrantgio/theme/theme"
	"github.com/vibrantgio/theme/tokens"
	"github.com/vibrantgio/theme/typeset"

	"github.com/vibrantgio/components/internal/focus"
	"github.com/vibrantgio/components/internal/hit"
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

// darkFillStep is how far over its ground a resting chip stands where the
// Background pin is the darkest surface the neutral ramp carries, in CIELAB
// L\*. It is a MEASUREMENT of the platform, not a derivation: 1.28 L\*, the
// step macOS takes between a unified toolbar's band and the pop-up capsules
// drawn on it — the chip's exact role. The package doc quotes the capture,
// the two fills and the method.
const darkFillStep = 1.28

// lightFillStep is the same step where the pin is the lightest surface the
// ramp carries, in CIELAB L\*. It is a DERIVATION and the package doc says
// so: the stored macOS reference holds no light-appearance capture to
// measure, so this half takes the ladder's own first storey over the paper —
// the 0.70 L\* the light scheme already spends on Level1 over Level0, now
// spent identically over every ground rather than growing with the ground's
// position on the ladder.
const lightFillStep = 0.70

// fillStep is how far above its ground a resting chip's fill stands, in
// CIELAB L\*. One number per scheme, and the scheme is never named: which
// half applies is read off the neutral surface band's direction, exactly as
// theme/tokens reads it for the floor's own two measurements.
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

// restFill is the chip's fill at rest: the surface it stands on, lifted by
// the measured step for its scheme, realized at the ground's own hue and
// chroma so the pill carries whatever tint the ladder carries and none of
// its own. Nothing is mixed and no colour is named — the step is a depth in
// L\* and the palette renders it, the way theme/tokens realizes a storey.
//
// It is a step over the LOCAL GROUND rather than a walk to the next storey,
// which is the change BB1.1 makes. The storey above was correct as depth and
// wrong as loudness: in the dark scheme it stood 10.0 luminance over the
// window's paper where the platform's own toolbar capsules stand 2.65 over
// their band — a filled block at four times the platform's step, in the one
// role the platform draws as a near-hairline outline. The rim still carries
// the edge; the fill no longer shouts under it.
func restFill(c tokens.ColorTokens, ground tokens.ElevationLevel) color.NRGBA {
	base := c.SurfaceAt(ground)
	l, _, _ := vgcolor.LabFromNRGBA(base)
	target := min(l+fillStep(c), 100)
	_, chroma, hue := vgcolor.OKLChFromNRGBA(base)
	return vgcolor.NRGBAFromToneChromaHue(target, chroma, hue)
}

// Fill is the chip's ground: the measured step over the surface it stands
// on, walked by the interaction state.
//
// The walk is [tokens.ColorTokens.PinnedStateColor] — the same walk
// [tokens.ColorTokens.StateAt] takes from a storey, taken from the chip's own
// resting fill instead, because that fill is no longer a storey. Hover and
// press therefore follow the new rest automatically and their stride is
// untouched.
//
// It is exported because a container that draws behind or beside a chip — a
// header band deciding what its own seam should clear, a test measuring the
// pill — needs the same answer the chip drew with, and re-deriving it at the
// call site is how two answers appear.
func Fill(c tokens.ColorTokens, ground tokens.ElevationLevel, state tokens.State) color.NRGBA {
	return c.PinnedStateColor(restFill(c, ground), state)
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

// Pin is the edge of the box a chip is offered that its pill is pinned to.
//
// It is a placement, not a stretch: the pill stays sized to its content —
// see the package doc — and what changes is where in the offered box it is
// drawn and how much of that box the widget reports having used. Only the
// horizontal axis is pinned, because the vertical one is already settled by
// whatever row the chip stands in.
//
// The seam exists because a chip alone can be placed by its container and a
// chip handed to a pattern cannot. patterns/popover measures its anchor and
// centres it in the canvas it was given, so a container that reserves a cap
// for the anchor and wants the pill on the cap's trailing edge — a picker
// standing over the content column it belongs to — has nothing to say. With
// a pin it says it once, to the chip, where the drawn width is known.
type Pin uint8

const (
	// PinNone is the zero value and the chip's own habit: the widget reports
	// the pill it drew and no more, so a row of chips is laid out at the
	// pills' own scale and the box around them is the container's business.
	PinNone Pin = iota

	// PinLeading draws the pill at the leading edge of the offered box.
	PinLeading

	// PinTrailing draws the pill at the trailing edge of the offered box.
	PinTrailing
)

// layout draws w at p's edge of the box the chip was offered — the horizontal
// half of gtx.Constraints.Max — and reports that box rather than w's own size,
// which is what lets a caller upstream of the chip find the pinned edge where
// it asked for it. PinNone lays w out untouched, so a chip that pins nothing
// pays nothing.
//
// The whole widget is offset, slop and all, so the pointer target stays
// centred on the pill it was extended around.
func (p Pin) layout(gtx layout.Context, w layout.Widget) layout.Dimensions {
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

// Props configures a [Chip] instance: what the pill says, what it stands on,
// and how an activation is delivered.
//
// There is no emphasis field and there will not be one. A chip has one weight
// by construction — see the package doc — and selection rides
// components/button's register instead. There is no Disabled field either: a
// chip that cannot be clicked is a badge, which is a different face and takes
// [RenderBadge].
type Props struct {
	// Label is the text the pill carries.
	Label string

	// Icon is the mark drawn after the label, in the label's own line box. A
	// nil Icon draws no mark and the chip is label-only. It is named Icon
	// rather than Glyph to match components/button's Props, so a caller
	// moving between the two components writes the same field name.
	Icon Glyph

	// Description is the screen-reader label. Falls back to Label when empty.
	Description string

	// Ground is the elevation storey of the surface hosting the chip, copied
	// straight into [RenderState.Ground] on every frame: the chip fills one
	// rung above it and derives its rim, its inks and its focus ring from the
	// pair. A dialog at tokens.Level2 passes Level2. The zero value is
	// tokens.Level0, the window ground. See [RenderState.Ground].
	Ground tokens.ElevationLevel

	// Pin is the edge of the offered box the pill is drawn at. The zero value
	// is [PinNone] and the chip reports the pill alone, which is every chip
	// laid out by its own container. Set it where the box is a cap the caller
	// sized and something between the caller and the chip does the placing —
	// see [Pin].
	Pin Pin

	// Clickable, if non-nil, is used instead of an internally-allocated one.
	// The caller then owns &Clickable as the chip's focus tag — usable with
	// key.FocusCmd, key.Filter{Focus: …} and an external Tab cycle — and may
	// detect activation via Clickable.Clicked(gtx). This is what lets a
	// container that drives focus itself — a popover anchored on the chip —
	// avoid a doubled focus ring. When nil the chip allocates and owns its
	// own clickable, which survives every theme emission.
	Clickable *widget.Clickable

	// OnClick is called when the chip is activated by click or Space/Enter.
	// The gtx argument is the layout.Context active on the frame the
	// activation is processed in, so a consumer may emit
	// mvu.MessageOp{Message: …}.Add(gtx.Ops) from inside it.
	OnClick func(gtx layout.Context)

	// Message, if non-nil, is emitted as mvu.MessageOp into gtx.Ops on
	// activation — the MVU path, where OnClick is the FRP one. Both fire when
	// both are set, and they fire from the one place the activation is
	// noticed: the chip polls its clickable once per frame, so a click and a
	// Space both arrive through the same branch and neither can dispatch
	// twice.
	Message any

	// Shaper is an explicit per-instance override of the text shaper. Leave
	// it nil in normal use: the chip then shapes with the theme's shaper
	// (tokens.Typography.Shaper()), built once for the process and shared by
	// every component reading that typography. Set it only when this chip
	// must shape with a different one — a golden test pinning its faces.
	Shaper *text.Shaper
}

// resolvedTokens is the concrete per-emission snapshot the widget closure
// draws from: the whole theme flattened to the values one frame needs.
type resolvedTokens struct {
	color   tokens.ColorTokens
	label   tokens.TextStyle // the LabelLarge role
	spacing tokens.SpacingScale
	radius  tokens.RadiusScale
	density tokens.Density
	shaper  *text.Shaper
}

// Chip returns an rx.Observable[layout.Widget] emitting a new widget whenever
// the theme changes. It is the live face of [Render]: the same pill, drawn
// from the theme rather than from tokens handed in, with the three things the
// pure path cannot carry — the pointer area, the keyboard, and the dispatch.
//
// The pointer target is extended to the density's [tokens.Density.MinHitTarget]
// (44 dp, WCAG 2.5.5) on both axes, centred on the drawn pill, exactly as
// components/button extends its own: the chip draws at the density's control
// height and what the pointer may land on does not shrink with it. The widget
// still reports the pill's size, so a row of chips is laid out at the pill's
// scale and the slop overhangs the air around it — unless [Props.Pin] asks
// for the box instead, in which case the slop travels with the pill it was
// centred on.
//
// Keyboard activation is gioui.org/widget.Clickable's: the chip is focusable,
// Space and Enter activate it, and gtx.Focused drives [RenderState.Focused] —
// so a focused chip wears the ring the package doc describes, derived against
// its own storey. Both integration paths are supported and both are read off
// the one poll of the clickable:
//   - FRP: set Props.OnClick.
//   - MVU: set Props.Message; the chip emits mvu.MessageOp on activation.
//
// Widget state — hover, press, focus — lives in the rx.Defer scope and
// survives every theme emission for the life of the subscription.
func Chip(th rx.Observable[theme.Theme], props Props) rx.Observable[layout.Widget] {
	// Flatten the nested theme observables into one concrete snapshot. The
	// typography emission carries both the LabelLarge role the chip is set in
	// and the theme's cached shaper (ADR-003: the theme owns the typeface).
	resolved := rx.SwitchMap(th, func(t theme.Theme) rx.Observable[resolvedTokens] {
		return rx.Map(
			rx.CombineLatest5(t.Color, t.Typography, t.Spacing, t.Radius, t.Density),
			func(n rx.Tuple5[tokens.ColorTokens, tokens.Typography, tokens.SpacingScale, tokens.RadiusScale, tokens.Density]) resolvedTokens {
				typ := n.Second
				return resolvedTokens{
					color:   n.First,
					label:   typ.LabelLarge,
					spacing: n.Third,
					radius:  n.Fourth,
					density: n.Fifth,
					shaper:  typ.Shaper(),
				}
			},
		)
	})

	return rx.Defer(func() rx.Observable[layout.Widget] {
		// Allocated once per subscription, so hover, press and focus survive
		// every theme emission. Used only when the caller supplies none.
		var ownClick widget.Clickable

		return rx.Map(resolved, func(tok resolvedTokens) layout.Widget {
			shaper := props.Shaper
			if shaper == nil {
				shaper = tok.shaper
			}
			desc := props.Description
			if desc == "" {
				desc = props.Label
			}

			return func(gtx layout.Context) layout.Dimensions {
				click := props.Clickable
				if click == nil {
					click = &ownClick
				}

				// One poll, one dispatch: Clicked reports a pointer click and
				// a Space or Enter alike, so both paths leave from here.
				if click.Clicked(gtx) {
					if props.OnClick != nil {
						props.OnClick(gtx)
					}
					if props.Message != nil {
						mvu.MessageOp{Message: props.Message}.Add(gtx.Ops)
					}
				}

				s := RenderState{
					Ground:  props.Ground,
					Hovered: click.Hovered(),
					Pressed: click.Pressed(),
					Focused: gtx.Focused(click),
				}

				return props.Pin.layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return hit.Extend(gtx, gtx.Dp(unit.Dp(tok.density.MinHitTarget())), click.Layout,
						func(gtx layout.Context) layout.Dimensions {
							semantic.ClassOp(semantic.Button).Add(gtx.Ops)
							semantic.LabelOp(props.Label).Add(gtx.Ops)
							semantic.DescriptionOp(desc).Add(gtx.Ops)
							semantic.EnabledOp(true).Add(gtx.Ops)
							return draw(gtx, shaper, props.Label, props.Icon, tok.color,
								tok.spacing, tok.radius, tok.label, tok.density, s, true)
						})
				})
			}
		})
	})
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
	h := max(labelDims.Size.Y+2*padV, minH)
	w = min(w, gtx.Constraints.Max.X)
	h = min(h, gtx.Constraints.Max.Y)
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
	offX := max((w-content)/2, padH)
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

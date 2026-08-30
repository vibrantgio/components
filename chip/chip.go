package chip

import (
	"image/color"

	"gioui.org/io/semantic"
	"gioui.org/layout"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"

	"github.com/reactivego/rx"

	"github.com/vibrantgio/mvu"
	"github.com/vibrantgio/theme/theme"
	"github.com/vibrantgio/theme/tokens"

	"github.com/vibrantgio/components/internal/chipface"
	"github.com/vibrantgio/components/internal/hit"
	"github.com/vibrantgio/components/picker"
)

// Face is which member of the chip family a widget draws. The geometry is one
// geometry — the measured fill, the two-sided rim, the density's height and
// padding, the state walk, the pointer target — and the face names what varies
// on top of it.
//
// [FaceChip] and [FaceAnchor] are the two a caller may ask for. The badge is a
// face too, but it is not selectable: it takes no input, so it has no live
// path to select it in, and [RenderBadge] is the whole of it.
type Face uint8

const (
	// FaceChip is the pill: the scale's Full radius, the caller's own glyph,
	// and the zero value, so a [Props] that says nothing draws the chip.
	FaceChip Face = Face(chipface.FaceChip)

	// FaceAnchor is the pull-down anchor: the same chip at the button's own
	// rounded-rect radius, with the down chevron drawn by the component
	// instead of a caller's glyph.
	//
	// Deprecated: the pull-down anchor is components/picker's, where it is
	// one of two triggers over a shared menu. Use picker.Anchor.
	FaceAnchor Face = Face(chipface.FaceAnchor)
)

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
	// the input to every colour the chip resolves: the fill is the measured
	// step over that storey, and the rim is the neutral rung that clears the
	// graphic floor against both. A dialog at tokens.Level2 passes Level2.
	// The zero value is tokens.Level0, the window ground.
	Ground tokens.ElevationLevel

	Hovered bool
	Pressed bool
	Focused bool
}

// Fill is the chip's ground: the measured step over the surface it stands
// on, walked by the interaction state. The package doc states the step and
// which half of it is measured.
//
// It is exported because a container that draws behind or beside a chip — a
// header band deciding what its own seam should clear, a test measuring the
// pill — needs the same answer the chip drew with, and re-deriving it at the
// call site is how two answers appear.
func Fill(c tokens.ColorTokens, ground tokens.ElevationLevel, state tokens.State) color.NRGBA {
	return chipface.Fill(c, ground, state)
}

// Rim is the chip's edge, and whether it has one: the rung of the neutral ramp
// that reaches the graphic floor against BOTH of the edge's neighbours — the
// ground outside it and the chip's own fill inside it — or no rim at all when
// no rung can reach both, which is the case where the fill is carrying its own
// edge. The package doc states why one side is not enough.
func Rim(c tokens.ColorTokens, ground tokens.ElevationLevel, state tokens.State) (color.NRGBA, bool) {
	return chipface.Rim(c, ground, state)
}

// Ink is the colour something reads in when it is drawn on a chip's fill: the
// Text pin while that pin clears floor against fill, and otherwise the rung of
// the neutral ramp nearest its mid-value step that does.
//
// Pass tokens.TextFloor for the label and tokens.GraphicFloor for the glyph.
func Ink(c tokens.ColorTokens, fill color.NRGBA, floor float64) color.NRGBA {
	return chipface.Ink(c, fill, floor)
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
// chip handed on to a container that centres whatever it is given cannot: the
// reserved cap and the drawn pill then part company by half the slack, and
// the only place both widths are known is inside the chip. A pin says it
// there, once.
//
// It costs the container the drawn rect, which is the whole box as far as it
// can tell, so say it only where nothing upstream needs that rect. A
// container that aligns what it is given needs no pin at all.
type Pin uint8

const (
	// PinNone is the zero value and the chip's own habit: the widget reports
	// the pill it drew and no more, so a row of chips is laid out at the
	// pills' own scale and the box around them is the container's business.
	PinNone Pin = Pin(chipface.PinNone)

	// PinLeading draws the pill at the leading edge of the offered box.
	PinLeading Pin = Pin(chipface.PinLeading)

	// PinTrailing draws the pill at the trailing edge of the offered box.
	PinTrailing Pin = Pin(chipface.PinTrailing)
)

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

	// Face is which member of the family this widget draws: [FaceChip], the
	// zero value and the pill, or [FaceAnchor], the pull-down anchor. The
	// anchor draws its own chevron and ignores Icon.
	Face Face

	// Icon is the mark drawn after the label, in the label's own line box. A
	// nil Icon draws no mark and the chip is label-only. It is named Icon
	// rather than Glyph to match components/button's Props, so a caller
	// moving between the two components writes the same field name.
	//
	// Ignored when Face is [FaceAnchor]: that face's mark is the component's
	// own chevron, which is the whole reason the face exists.
	Icon Glyph

	// Description is the screen-reader label. Falls back to Label when empty.
	Description string

	// Ground is the elevation storey of the surface hosting the chip, copied
	// straight into [RenderState.Ground] on every frame: the chip fills the
	// measured step over it and derives its rim, its inks and its focus ring
	// from the pair. A dialog at tokens.Level2 passes Level2. The zero value
	// is tokens.Level0, the window ground. See [RenderState.Ground].
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

				s := chipface.State{
					Ground:  props.Ground,
					Hovered: click.Hovered(),
					Pressed: click.Pressed(),
					Focused: gtx.Focused(click),
				}

				return chipface.Pin(props.Pin).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return hit.Extend(gtx, gtx.Dp(unit.Dp(tok.density.MinHitTarget())), click.Layout,
						func(gtx layout.Context) layout.Dimensions {
							semantic.ClassOp(semantic.Button).Add(gtx.Ops)
							semantic.LabelOp(props.Label).Add(gtx.Ops)
							semantic.DescriptionOp(desc).Add(gtx.Ops)
							semantic.EnabledOp(true).Add(gtx.Ops)
							return chipface.Draw(gtx, shaper, props.Label, chipface.Glyph(props.Icon),
								tok.color, tok.spacing, tok.radius, tok.label, tok.density,
								s, chipface.Face(props.Face))
						})
				})
			}
		})
	})
}

// Render produces a layout.Widget drawing the interactive chip face in an
// explicit visual state, without event processing: the pill filled the
// measured step over s.Ground and walked by the pointer, its one-dp rim, the
// label in the ink that clears the text floor on that fill, and the glyph in
// the ink that clears the graphic floor. When s.Focused, the focus ring —
// measured against that fill — takes the rim's place at the chip's edge, two
// dp instead of one.
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
		return chipface.Draw(gtx, shaper, label, chipface.Glyph(glyph), colors, sp, rad,
			labelStyle, d, chipface.State(s), chipface.FaceChip)
	}
}

// RenderAnchor produces a layout.Widget drawing the ANCHOR face.
//
// Deprecated: the pull-down anchor is components/picker's — the chrome
// register's trigger over the menu the picker's other trigger shares. Use
// picker.RenderAnchor, which this forwards to; it draws the same control from
// the same geometry.
func RenderAnchor(
	shaper *text.Shaper,
	label string,
	colors tokens.ColorTokens,
	sp tokens.SpacingScale,
	rad tokens.RadiusScale,
	labelStyle tokens.TextStyle,
	d tokens.Density,
	s RenderState,
) layout.Widget {
	return picker.RenderAnchor(shaper, label, colors, sp, rad, labelStyle, d, picker.AnchorState(s))
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
		return chipface.Draw(gtx, shaper, label, chipface.Glyph(glyph), colors, sp, rad,
			labelStyle, d, chipface.State{Ground: ground}, chipface.FaceBadge)
	}
}

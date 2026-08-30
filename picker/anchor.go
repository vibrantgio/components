package picker

import (
	"gioui.org/io/semantic"
	"gioui.org/layout"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"

	"github.com/reactivego/rx"
	"github.com/vibrantgio/components/internal/chipface"
	"github.com/vibrantgio/components/internal/hit"
	"github.com/vibrantgio/mvu"
	"github.com/vibrantgio/theme/theme"
	"github.com/vibrantgio/theme/tokens"
)

// Pin is the edge of the box an anchor is offered that its control is pinned
// to.
//
// It is a placement, not a stretch: the anchor stays sized to its value and
// what changes is where in the offered box it is drawn and how much of that
// box the widget reports having used. Only the horizontal axis is pinned,
// because the vertical one is already settled by whatever row the anchor
// stands in.
//
// Say it only where the box is a cap the caller sized and something between
// the caller and the anchor does the placing. An anchor its container lays out
// needs nothing here — the container knows both boxes and offsets the widget.
// An anchor handed to patterns/popover does: the popover measures its anchor
// and centres it in the canvas it was given, so the reserved cap and the drawn
// control part company by half the slack, and the only place both widths are
// known is inside the anchor.
type Pin uint8

const (
	// PinNone is the zero value and the anchor's own habit: the widget
	// reports the control it drew and no more, so the box around it is the
	// container's business.
	PinNone Pin = iota

	// PinLeading draws the control at the leading edge of the offered box.
	PinLeading

	// PinTrailing draws the control at the trailing edge of the offered box.
	PinTrailing
)

// AnchorState holds the explicit visual state a static anchor render draws in.
// The zero value is a resting anchor on the window ground.
//
// Intended for golden-image testing and static rendering; production code
// obtains the interaction half from the Gio event system via [Anchor].
type AnchorState struct {
	// Ground is the elevation storey of the surface hosting the anchor, in
	// the same vocabulary the host names its own fill (tokens.SurfaceAt). It
	// is the input to every colour the anchor resolves: the fill is the
	// measured step over this storey, and the rim is the neutral rung that
	// clears the graphic floor against both sides of the edge. A dialog at
	// tokens.Level2 passes Level2. The zero value is tokens.Level0, the
	// window ground.
	Ground tokens.ElevationLevel

	Hovered bool
	Pressed bool
	Focused bool
}

// AnchorProps configures an [Anchor] instance.
type AnchorProps struct {
	// Value is the text the control carries: the choice the picker currently
	// holds, because a picker's trigger shows its value.
	Value string

	// Description is the screen-reader label. Falls back to Value when empty.
	Description string

	// Ground is the elevation storey of the surface hosting the anchor,
	// copied straight into [AnchorState.Ground] on every frame. A dialog at
	// tokens.Level2 passes Level2. The zero value is tokens.Level0, the
	// window ground. See [AnchorState.Ground].
	Ground tokens.ElevationLevel

	// Pin is the edge of the offered box the control is drawn at. The zero
	// value is [PinNone] and the anchor reports the control alone. See [Pin].
	Pin Pin

	// Clickable, if non-nil, is used instead of an internally-allocated one.
	// The caller then owns &Clickable as the anchor's focus tag — usable with
	// key.FocusCmd, key.Filter{Focus: …} and an external Tab cycle — and may
	// detect activation via Clickable.Clicked(gtx). This is what lets a
	// container that drives focus itself — a popover anchored on this control
	// — avoid a doubled focus ring. When nil the anchor allocates and owns its
	// own clickable, which survives every theme emission.
	Clickable *widget.Clickable

	// OnClick is called when the anchor is activated by click or Space/Enter.
	// The gtx argument is the layout.Context active on the frame the
	// activation is processed in, so a consumer may emit
	// mvu.MessageOp{Message: …}.Add(gtx.Ops) from inside it.
	OnClick func(gtx layout.Context)

	// Message, if non-nil, is emitted as mvu.MessageOp into the frame's ops on
	// activation — the MVU path, where OnClick is the FRP one. Both fire when
	// both are set, and they fire from the one place the activation is
	// noticed: the anchor polls its clickable once per frame, so a click and a
	// Space both arrive through the same branch and neither can dispatch
	// twice.
	Message any

	// Shaper is an explicit per-instance override of the text shaper. Leave
	// it nil in normal use: the anchor then shapes with the theme's shaper
	// (tokens.Typography.Shaper()), built once for the process and shared by
	// every component reading that typography. Set it only when this anchor
	// must shape with a different one — a golden test pinning its faces.
	Shaper *text.Shaper
}

// Anchor returns an rx.Observable[layout.Widget] emitting the chrome
// register's trigger: the platform's pull-down control, at the button's
// rounded-rect corner with the single down chevron drawn by the component.
//
// It has no menu of its own — a chrome-register menu floats against the window
// and patterns/popover places it, so the caller hands this widget to the
// popover as its anchor and a [Menu] as its content. [Field] is the trigger
// that drops its own menu.
//
// The mark commits the caller to that placement: the chevron says a menu opens
// BELOW this control, so the popover it is handed to must place the menu below
// it. See the package doc for what a trigger the menu stands over would need.
//
// The pointer target is extended to the density's tokens.Density.MinHitTarget
// (44 dp, WCAG 2.5.5) on both axes, centred on the drawn control, exactly as
// components/button extends its own: the anchor draws at the density's control
// height and what the pointer may land on does not shrink with it. The widget
// still reports the control's size — unless [AnchorProps.Pin] asks for the box
// instead, in which case the slop travels with the control it was centred on.
//
// Keyboard activation is gioui.org/widget.Clickable's: the anchor is
// focusable, Space and Enter activate it, and gtx.Focused drives
// [AnchorState.Focused]. Both integration paths are supported and both are
// read off the one poll of the clickable:
//   - FRP: set AnchorProps.OnClick.
//   - MVU: set AnchorProps.Message; the anchor emits mvu.MessageOp on
//     activation.
//
// Widget state — hover, press, focus — lives in the rx.Defer scope and
// survives every theme emission for the life of the subscription.
func Anchor(th rx.Observable[theme.Theme], props AnchorProps) rx.Observable[layout.Widget] {
	// The typography emission carries both the LabelLarge role the anchor is
	// set in — the chip family's role, because the anchor is one of its faces
	// — and the theme's cached shaper (the theme owns the typeface).
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
				desc = props.Value
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
							semantic.LabelOp(props.Value).Add(gtx.Ops)
							semantic.DescriptionOp(desc).Add(gtx.Ops)
							semantic.EnabledOp(true).Add(gtx.Ops)
							return chipface.Draw(gtx, shaper, props.Value, nil, tok.color,
								tok.spacing, tok.radius, tok.label, tok.density, s, chipface.FaceAnchor)
						})
				})
			}
		})
	})
}

// RenderAnchor produces a layout.Widget drawing the chrome register's trigger
// in an explicit visual state, without event processing: the control filled
// the measured step over s.Ground and walked by the pointer, its one-dp rim,
// the value in the ink that clears the text floor on that fill, and the down
// chevron in the ink that clears the graphic floor. When s.Focused, the focus
// ring — measured against that fill — takes the rim's place at the control's
// edge, two dp instead of one.
//
// It takes no glyph, and that is the point rather than an omission: the mark
// on a pull-down anchor is not the caller's to choose, and it does not change
// when the menu opens. The platform's anchor says "a menu opens below this" and
// never "this is open"; a caller that flipped the chevron would be saying the
// second thing in a vocabulary the platform reserves for a disclosure triangle.
//
// labelStyle is the whole text style the value is set in; pass
// tokens.DefaultTypography.LabelLarge with tokens.Comfortable for the default
// desktop control. The anchor is sized to its content, clamped to the
// constraints it is handed, and asks for the pointer cursor. Extending its
// pointer area to tokens.MinHitTarget is the live path's job — see [Anchor].
func RenderAnchor(
	shaper *text.Shaper,
	value string,
	colors tokens.ColorTokens,
	sp tokens.SpacingScale,
	rad tokens.RadiusScale,
	labelStyle tokens.TextStyle,
	d tokens.Density,
	s AnchorState,
) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		return chipface.Draw(gtx, shaper, value, nil, colors, sp, rad, labelStyle, d,
			chipface.State(s), chipface.FaceAnchor)
	}
}

package button

import (
	"image"
	"image/color"

	"gioui.org/font"
	"gioui.org/io/pointer"
	"gioui.org/io/semantic"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"

	"github.com/reactivego/rx"
	"github.com/vibrantgio/mvu"
	"github.com/vibrantgio/prism/internal/hit"
	"github.com/vibrantgio/spectrum/theme"
	"github.com/vibrantgio/spectrum/tokens"
)

// RenderState holds explicit visual interaction state for static rendering.
// All fields default to false (normal/idle state).
// Intended for golden-image testing; production code obtains state from the
// Gio event system via Button.
type RenderState struct {
	Hovered  bool
	Focused  bool
	Pressed  bool
	Disabled bool
}

// Props configures a Button instance.
type Props struct {
	// Label is the text rendered inside the button.
	Label string

	// Description is the screen-reader label. Falls back to Label when empty.
	Description string

	// Icon, when non-nil and Label is empty, renders the button as a compact
	// icon-only affordance: a square the density's control height on a side
	// with the glyph centred, instead of a fill-width text label (the pointer
	// target stays at least the 44 dp square). The painter draws into
	// a sizePx×sizePx box at the current origin in colour col, via
	// clip.Path / clip.Stroke, so output stays golden-deterministic (no font or
	// SVG rasterisation). prism/icon is the registry for named glyphs;
	// determinism-sensitive callers pass a clip.Path painter directly.
	Icon func(gtx layout.Context, sizePx int, col color.NRGBA)

	// Disabled, if non-nil, disables the button when it emits true.
	// A nil Disabled means always enabled.
	Disabled rx.Observable[bool]

	// OnClick is called when the button is activated by click or Space/Enter.
	// This is the FRP callback path. The gtx argument is the layout.Context
	// active on the frame when the click is processed, allowing consumers to
	// emit mvu.MessageOp{Message: ...}.Add(gtx.Ops) inside the callback.
	OnClick func(gtx layout.Context)

	// Message, if non-nil, causes the button to emit mvu.MessageOp{Message}
	// into gtx.Ops on activation. This is the MVU integration path.
	Message any

	// Clickable, if non-nil, is used instead of an internally-allocated one.
	// The caller then owns &Clickable as the button's focus tag — usable with
	// key.FocusCmd, key.Filter{Focus: …} and an external Tab cycle — and may
	// detect activation via Clickable.Clicked(gtx). This lets a container (e.g.
	// cadence/modal) drive focus and trap Tab without a doubled focus ring.
	// When nil the button allocates and owns its own clickable.
	Clickable *widget.Clickable

	// Shaper is an explicit per-instance override of the text shaper. Leave it
	// nil in normal use: the button then shapes its label with the theme's
	// shaper (Typography.Shaper()), which is built once and cached inside the
	// theme's Typography value. Set it only when this button must shape with
	// a different shaper than the theme provides.
	Shaper *text.Shaper
}

// resolvedTokens is the concrete per-emission snapshot consumed by the widget closure.
type resolvedTokens struct {
	color   tokens.ColorTokens
	label   tokens.TextStyle // the LabelLarge role: typeface, weight, size, line height
	spacing tokens.SpacingScale
	radius  tokens.RadiusScale
	density tokens.Density // control height and inner padding (E1.3)
	shaper  *text.Shaper   // the theme's shaper; nil in the Render/RenderIcon path
}

// Button returns an rx.Observable[layout.Widget] that emits a new widget
// whenever the theme or disabled state changes. Widget state (clickable, hover,
// focus, press) lives in the rx.Defer scope and persists across emissions.
//
// Both integration paths are supported:
//   - FRP: set Props.OnClick; FRP consumers wrap with rx.NewSubject if needed.
//   - MVU: set Props.Message; the component emits mvu.MessageOp on activation.
func Button(th rx.Observable[theme.Theme], props Props) rx.Observable[layout.Widget] {
	disabled := props.Disabled
	if disabled == nil {
		disabled = rx.Of(false)
	}

	// Flatten the nested theme observables into a concrete snapshot. The
	// typography emission supplies both the LabelLarge text style and the
	// theme's cached shaper (ADR-003: the theme owns the typeface).
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

	inputs := rx.CombineLatest2(resolved, disabled)

	return rx.Defer(func() rx.Observable[layout.Widget] {
		// Allocated once per subscription — survives all theme and disabled
		// emissions for the lifetime of this button instance. Used only when
		// the caller does not supply Props.Clickable.
		var ownClick widget.Clickable

		return rx.Map(inputs, func(next rx.Tuple2[resolvedTokens, bool]) layout.Widget {
			tok, dis := next.First, next.Second

			// Props.Shaper is an explicit override; the theme's shaper is
			// the default.
			shaper := props.Shaper
			if shaper == nil {
				shaper = tok.shaper
			}

			return func(gtx layout.Context) layout.Dimensions {
				if dis {
					gtx = gtx.Disabled()
				}

				// The caller may own the clickable (and thus the focus tag);
				// otherwise use the per-subscription one.
				click := props.Clickable
				if click == nil {
					click = &ownClick
				}

				// Process events; Clicked also handles Space/Enter via widget.Clickable.
				if click.Clicked(gtx) {
					if props.OnClick != nil {
						props.OnClick(gtx)
					}
					if props.Message != nil {
						mvu.MessageOp{Message: props.Message}.Add(gtx.Ops)
					}
				}

				hov := click.Hovered()
				prs := click.Pressed()
				foc := !dis && gtx.Focused(click)

				desc := props.Description
				if desc == "" {
					desc = props.Label
				}

				iconOnly := props.Icon != nil && props.Label == ""

				// The clickable's pointer area is at least MinHitTarget
				// (44 dp) on each axis, centred on the visual control:
				// density shrinks the drawn button, never the hit target.
				return hit.Extend(gtx, gtx.Dp(unit.Dp(tok.density.MinHitTarget())), click.Layout, func(gtx layout.Context) layout.Dimensions {
					semantic.ClassOp(semantic.Button).Add(gtx.Ops)
					semantic.LabelOp(props.Label).Add(gtx.Ops)
					semantic.DescriptionOp(desc).Add(gtx.Ops)
					semantic.EnabledOp(!dis).Add(gtx.Ops)
					state := RenderState{
						Hovered:  hov,
						Focused:  foc,
						Pressed:  prs,
						Disabled: dis,
					}
					if iconOnly {
						return drawIconButton(gtx, props.Icon, tok, state)
					}
					return drawButton(gtx, shaper, props.Label, tok, state)
				})
			}
		})
	})
}

// Render produces a layout.Widget for a button in an explicit visual state,
// without any event processing or rx machinery. Intended for golden-image
// testing and static demonstrations; production code should use Button, which
// takes the shaper and the LabelLarge text style from the theme's Typography.
// The TypeScale parameter contributes only the LabelLarge size; typeface,
// weight and line height stay at the shaper's defaults. Density is not a
// parameter (the signature predates E1.3): the static path renders at
// tokens.Comfortable; density-aware rendering goes through Button.
func Render(
	shaper *text.Shaper,
	label string,
	colors tokens.ColorTokens,
	sp tokens.SpacingScale,
	rad tokens.RadiusScale,
	ts tokens.TypeScale,
	s RenderState,
) layout.Widget {
	tok := resolvedTokens{color: colors, spacing: sp, radius: rad, label: tokens.TextStyle{Size: ts.LabelLarge}, density: tokens.Comfortable}
	return func(gtx layout.Context) layout.Dimensions {
		return drawButton(gtx, shaper, label, tok, s)
	}
}

// RenderIcon produces a layout.Widget for a compact icon-only button in an
// explicit visual state, without event processing or rx machinery. The glyph is
// drawn by icon into a square the control height of tokens.Comfortable on a
// side (the signature predates E1.3; density-aware rendering goes through
// Button). Intended for golden-image testing and static demonstrations;
// production code should use Button with Props.Icon (and, when a container
// drives focus, Props.Clickable).
func RenderIcon(
	icon func(gtx layout.Context, sizePx int, col color.NRGBA),
	colors tokens.ColorTokens,
	sp tokens.SpacingScale,
	rad tokens.RadiusScale,
	ts tokens.TypeScale,
	s RenderState,
) layout.Widget {
	tok := resolvedTokens{color: colors, spacing: sp, radius: rad, label: tokens.TextStyle{Size: ts.LabelLarge}, density: tokens.Comfortable}
	return func(gtx layout.Context) layout.Dimensions {
		return drawIconButton(gtx, icon, tok, s)
	}
}

// drawButton renders the button visual into gtx. All visual state comes from s;
// no event queries are performed here.
func drawButton(gtx layout.Context, shaper *text.Shaper, label string, tok resolvedTokens, s RenderState) layout.Dimensions {
	// E1.3 sizing rule: button height = Density.ControlHeight (36 dp
	// Comfortable, 28 dp Compact), inner padding = Density.PaddingX/PaddingY
	// (16/8 and 12/6). The 44 dp of the pre-density button was the WCAG hit
	// floor, not a control height; the pointer target keeps it via hit.Extend
	// in the live path.
	padH := gtx.Dp(unit.Dp(tok.density.PaddingX))
	padV := gtx.Dp(unit.Dp(tok.density.PaddingY))
	minH := gtx.Dp(unit.Dp(tok.density.ControlHeight))
	rad := gtx.Dp(unit.Dp(tok.radius.Md)) // 6 dp corner radius

	bg, fg := buttonColors(tok.color, s)

	// Record the text material (fg color op) — replayed inside the label layout.
	mColor := op.Record(gtx.Ops)
	paint.ColorOp{Color: fg}.Add(gtx.Ops)
	textMaterial := mColor.Stop()

	// Record the label render to obtain its size before drawing the background.
	labelGtx := gtx
	labelGtx.Constraints.Min = image.Pt(0, 0)
	maxLabelW := gtx.Constraints.Max.X - 2*padH
	if maxLabelW > 0 {
		labelGtx.Constraints.Max.X = maxLabelW
	}
	// Shape with the LabelLarge role's typeface, weight, size and line height.
	// Zero fields (the legacy Render path synthesizes a size-only style) fall
	// back to the shaper's defaults.
	style := tok.label
	f := font.Font{Typeface: font.Typeface(style.Typeface)}
	if style.Weight != 0 {
		f.Weight = tokens.FontWeight(style.Weight)
	}
	wl := widget.Label{MaxLines: 1}
	if style.LineHeight != 0 {
		wl.LineHeight = unit.Sp(style.LineHeight)
		wl.LineHeightScale = 1
	}
	mLabel := op.Record(gtx.Ops)
	labelDims := wl.Layout(labelGtx, shaper, f, unit.Sp(style.Size), label, textMaterial)
	labelCall := mLabel.Stop()

	// Button dimensions: fill available width, enforce the density's control
	// height as the minimum.
	btnW := gtx.Constraints.Max.X
	if btnW < labelDims.Size.X+2*padH {
		btnW = labelDims.Size.X + 2*padH
	}
	btnH := labelDims.Size.Y + 2*padV
	if btnH < minH {
		btnH = minH
	}
	btnSize := image.Pt(btnW, btnH)

	// Background fill.
	rrect := clip.RRect{Rect: image.Rectangle{Max: btnSize}, SE: rad, SW: rad, NE: rad, NW: rad}
	paint.FillShape(gtx.Ops, bg, rrect.Op(gtx.Ops))

	// Focus ring (2 dp stroke on the background boundary).
	if s.Focused {
		paint.FillShape(gtx.Ops, tok.color.FocusRing(), clip.Stroke{
			Path:  rrect.Path(gtx.Ops),
			Width: float32(gtx.Dp(2)),
		}.Op())
	}

	// Replay the label centered within the button.
	offX := (btnW - labelDims.Size.X) / 2
	offY := (btnH - labelDims.Size.Y) / 2
	st := op.Offset(image.Pt(offX, offY)).Push(gtx.Ops)
	labelCall.Add(gtx.Ops)
	st.Pop()

	if !s.Disabled {
		pointer.CursorPointer.Add(gtx.Ops)
	}

	return layout.Dimensions{Size: btnSize}
}

// drawIconButton renders a compact, square icon-only button: a square the
// density's control height on a side, filled with the button background, the
// focus ring when focused, and the glyph (drawn by icon) centred inside the
// padding. Shares buttonColors with the text button so
// hover/press/focus/disabled treatments match. All visual state comes from s;
// no event queries are performed here.
func drawIconButton(gtx layout.Context, icon func(gtx layout.Context, sizePx int, col color.NRGBA), tok resolvedTokens, s RenderState) layout.Dimensions {
	// E1.3 sizing rule: side = Density.ControlHeight, glyph inset =
	// Density.PaddingY, so the glyph gets ControlHeight − 2·PaddingY — the
	// same content-box rule icon.Size documents (20 dp Comfortable, 16 dp
	// Compact). The pointer target stays the 44 dp square via hit.Extend in
	// the live path.
	pad := gtx.Dp(unit.Dp(tok.density.PaddingY))
	side := gtx.Dp(unit.Dp(tok.density.ControlHeight))
	rad := gtx.Dp(unit.Dp(tok.radius.Md)) // 6 dp corner radius
	sz := image.Pt(side, side)

	bg, fg := buttonColors(tok.color, s)

	// Background fill.
	rrect := clip.RRect{Rect: image.Rectangle{Max: sz}, SE: rad, SW: rad, NE: rad, NW: rad}
	paint.FillShape(gtx.Ops, bg, rrect.Op(gtx.Ops))

	// Focus ring (2 dp stroke on the background boundary), matching drawButton.
	if s.Focused {
		paint.FillShape(gtx.Ops, tok.color.FocusRing(), clip.Stroke{
			Path:  rrect.Path(gtx.Ops),
			Width: float32(gtx.Dp(2)),
		}.Op())
	}

	// Glyph, centred within the padded square.
	if icon != nil {
		glyph := side - 2*pad
		if glyph < 1 {
			glyph, pad = side, 0
		}
		off := op.Offset(image.Pt(pad, pad)).Push(gtx.Ops)
		icon(gtx, glyph, fg)
		off.Pop()
	}

	if !s.Disabled {
		pointer.CursorPointer.Add(gtx.Ops)
	}
	return layout.Dimensions{Size: sz}
}

// buttonColors returns the background and foreground colors for the given
// state. The background is the Primary solid fill resolved through the D2.3
// state walk (ADR-007: hover and pressed step the pin toward the 900 end of
// the primary ramp; focus keeps the fill and draws the ring); the foreground
// is OnPrimary, faded to DisabledOpacity when disabled.
func buttonColors(c tokens.ColorTokens, s RenderState) (bg, fg color.NRGBA) {
	fg = c.OnPrimary
	state := tokens.StateNormal
	switch {
	case s.Disabled:
		state = tokens.StateDisabled
		fg = tokens.Disabled(fg)
	case s.Pressed:
		state = tokens.StatePressed
	case s.Hovered:
		state = tokens.StateHover
	case s.Focused:
		state = tokens.StateFocus
	}
	bg = c.SolidStateColor(tokens.RolePrimary, state)
	return
}

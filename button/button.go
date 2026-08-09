package button

import (
	"fmt"
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
	"github.com/vibrantgio/spectrum/typeset"
)

// Emphasis is the visual weight register a button wears — how loudly it
// competes for attention on the surface it sits on. It is a colour property
// and nothing else: the drawn control keeps the density's size, the pointer
// target keeps its 44 dp floor, and the focus ring is identical in every
// register. Keyboard visibility is not an emphasis property.
//
// Every desktop system carries this axis under its own names — MD3 has
// filled, tonal, outlined and text; Fluent primary, standard and subtle;
// Apple prominent, regular and plain — and the three registers below are
// the set all of them agree on: the two ends plus the tinted middle.
// Outlined is deliberately not a fourth register. A border is a property of
// a surface rather than a rung on a loudness scale, and prism already
// carries its two border weights as ramp steps 500 and 300.
type Emphasis int

const (
	// Filled is the loudest register and the zero value: the role's pinned
	// solid fill carrying its on-colour. One per surface — the action the
	// screen is about. Being the zero value is what makes this axis
	// additive: every Props and RenderState written before it existed
	// renders exactly as it did.
	Filled Emphasis = iota

	// Tonal is the middle register: a tinted fill off the role's own ramp
	// with the ramp's text shade on top (ADR-007's 100–300 tinted fills and
	// 700–900 text over them). It reads as an action without claiming the
	// surface's one loud slot — the register for a secondary action, and
	// the one a row of equals wears.
	Tonal

	// Ghost is the quietest register: no ground at rest, the label or glyph
	// in the neutral ramp's low-contrast text shade, and a neutral wash
	// only while the pointer is on it. For affordances that must be present
	// without being the subject — a dialog's close X, a toolbar of icons, a
	// tertiary "Learn more". A ghost is quiet, not small: it keeps the full
	// pointer target and the full focus ring.
	Ghost
)

// String returns the register's name in the vocabulary the design system
// uses everywhere else — the same three words the token sheet's CSS classes
// and the gallery's captions carry.
func (e Emphasis) String() string {
	switch e {
	case Filled:
		return "filled"
	case Tonal:
		return "tonal"
	case Ghost:
		return "ghost"
	}
	return fmt.Sprintf("Emphasis(%d)", int(e))
}

// RenderState holds the explicit visual state a static render draws in: the
// emphasis register the button wears and the interaction state it is in.
// The zero value is the filled register at rest, so RenderState{} is exactly
// today's default button.
//
// Intended for golden-image testing and static rendering; production code
// obtains the interaction half from the Gio event system via Button, which
// copies the register straight off Props.Emphasis.
type RenderState struct {
	// Emphasis is the visual weight register. It is a property of the
	// button rather than of the pointer, and it lives here because Render
	// and RenderIcon take exactly one parameter that is not a token and
	// this is it. Zero is Filled.
	Emphasis Emphasis

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

	// Emphasis is the visual weight register: Filled (the zero value) for
	// the one action a surface is about, Tonal for a secondary action,
	// Ghost for an affordance that must be present without being the
	// subject. It changes colour only — never the drawn size, never the
	// pointer target, never the focus ring. Composes with Icon: a ghost
	// icon button is a quiet glyph over a full 44 dp square.
	Emphasis Emphasis

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
	// shaper (Typography.Shaper()), which is built once for the process and
	// shared by every component reading that typography — the cache lives
	// behind the Typography value, so it survives the copy this component's
	// map function makes of it (spectrum F5.1). Set it only when this button
	// must shape with a different shaper than the theme provides.
	//
	// A shaper is not safe to use from two goroutines; Gio lays the widget
	// forest out on the one goroutine that runs the event loop, which is what
	// makes sharing it correct. See spectrum/tokens.Typography.Shaper.
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
						Emphasis: props.Emphasis,
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
// reads both of the parameters below off the theme.
//
// labelStyle is the LabelLarge role's whole text style and d is the density
// the button draws at (control height and inner padding). Pass
// tokens.DefaultTypography.LabelLarge and tokens.Comfortable for the default
// desktop look. s carries the emphasis register alongside the interaction
// state; its zero value is the filled button, so a call written before the
// register existed draws what it always drew.
//
// All four properties of the style are honoured, and line height is honoured
// in the sense a design system means: the label box is labelStyle.LineHeight
// tall, leading split evenly above and below the glyphs, so the button's
// height derives from the type role rather than from which letters the label
// happens to contain. Handing the number to gioui.org/widget.Label does not
// achieve that — it changes nothing on a single line — so the layout goes
// through spectrum/typeset, which is where that discrepancy is documented.
//
// The drawn height is therefore max(d.ControlHeight, LineHeight + 2×d.PaddingY),
// and the second term wins for Compact at any of the label roles: 20 + 12 = 32
// against a 28 dp control height. [tokens.Density.ControlHeight] is a floor,
// not a height.
func Render(
	shaper *text.Shaper,
	label string,
	colors tokens.ColorTokens,
	sp tokens.SpacingScale,
	rad tokens.RadiusScale,
	labelStyle tokens.TextStyle,
	d tokens.Density,
	s RenderState,
) layout.Widget {
	tok := resolvedTokens{color: colors, spacing: sp, radius: rad, label: labelStyle, density: d}
	return func(gtx layout.Context) layout.Dimensions {
		return drawButton(gtx, shaper, label, tok, s)
	}
}

// RenderIcon produces a layout.Widget for a compact icon-only button in an
// explicit visual state, without event processing or rx machinery. The glyph
// is drawn by icon into a square d.ControlHeight on a side, inset by
// d.PaddingY, in the register s carries — a ghost icon button is the quiet
// glyph cadence/modal's close affordance wants, over the same square and
// the same pointer target as a filled one.
// Pass tokens.Comfortable for the default desktop look. Intended
// for golden-image testing and static demonstrations; production code should
// use Button with Props.Icon (and, when a container drives focus,
// Props.Clickable).
//
// It takes no text style: an icon-only button draws no text, so unlike
// [Render] there is nothing for a tokens.TextStyle to reach.
func RenderIcon(
	icon func(gtx layout.Context, sizePx int, col color.NRGBA),
	colors tokens.ColorTokens,
	sp tokens.SpacingScale,
	rad tokens.RadiusScale,
	d tokens.Density,
	s RenderState,
) layout.Widget {
	tok := resolvedTokens{color: colors, spacing: sp, radius: rad, density: d}
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
	// Zero fields fall back to the shaper's defaults. typeset.Layout, not
	// widget.Label.Layout, because the role's line height has to be the height
	// of the label box and Gio alone reports the glyph ink instead — see
	// spectrum/typeset.
	style := tok.label
	f := typeset.Font(style, font.Normal)
	wl := typeset.Label(style, 1)
	mLabel := op.Record(gtx.Ops)
	labelDims := typeset.Layout(labelGtx, shaper, wl, f, unit.Sp(style.Size), label, textMaterial)
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

	// Focus ring (2 dp stroke on the background boundary). Identical in
	// every emphasis register, by construction: FocusRing is not one of the
	// colours buttonColors resolves. Keyboard visibility is not a loudness
	// property, so a ghost button's ring is exactly a filled one's.
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
// padding. Shares buttonColors with the text button so the register and the
// hover/press/focus/disabled treatments match. All visual state comes from s;
// no event queries are performed here.
func drawIconButton(gtx layout.Context, icon func(gtx layout.Context, sizePx int, col color.NRGBA), tok resolvedTokens, s RenderState) layout.Dimensions {
	// E1.3 sizing rule: side = Density.ControlHeight, glyph inset =
	// Density.PaddingY, so the glyph gets ControlHeight − 2·PaddingY — the
	// same content-box rule icon.Size documents (20 dp Comfortable, 16 dp
	// Compact). The pointer target stays the 44 dp square via hit.Extend in
	// the live path — in every register. Emphasis reaches the colours and
	// stops there: the glyph quiets, the square does not shrink.
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

// Ramp steps the quieter registers resolve against, in ADR-007's vocabulary.
// Every one of them is a step the ADR already names, so a register is a
// choice of rungs on the existing ramps rather than a second colour model.
//
// Both grounds are step 200, and the APCA gate is what fixes them there
// rather than taste. A tinted ground presses two steps deeper, and 200 is
// the only rung of ADR-007's 100–300 tinted band whose press still carries
// the 900 text: measured on the default seed, primary 900 over primary 400
// is Lc 62.8 light and −84.5 dark, and neutral 900 over neutral 400 is
// Lc 63.4 and −84.6 — all above the ADR's Lc 60 floor. A ground of 300
// would press onto 500 and take its label to Lc 47.5, unreadable, in
// exchange for a slightly louder button.
const (
	// tonalGround is the tinted fill (ADR-007's 100–300 band) a tonal
	// button rests on, and the ground its hover and press walk from — one
	// step to 300, two to 400, exactly as any tinted surface walks. It is
	// the step a card sits on, which is the relationship a tonal button
	// wants: a tinted card over the app background.
	tonalGround = 200
	// tonalText is the text shade over that tinted fill: step 900, the
	// stop the APCA gate holds at Lc ≥ 90 over the 100 and 200 grounds.
	tonalText = 900

	// ghostGround is the ground a ghost's wash walks from. A ghost paints
	// nothing at rest, so it has no ground of its own and must assume one;
	// it assumes the surface step, and performs that surface's own hover
	// (300) and press (400). Assuming the app background instead would
	// wash to 200, which is invisible on the card a ghost most often sits
	// on — a modal's close X, a toolbar over a panel. Assuming 200 costs
	// only that the wash reads one step strong on the bare background,
	// which is the harmless direction of the same error.
	ghostGround = 200
	// ghostText is the resting label shade: neutral step 700, ADR-007's
	// low-contrast text (Lc ≥ 60) — the resolution the deleted
	// OnSurfaceVariant alias carried.
	ghostText = 700
	// ghostTextOnWash is the label shade once a wash appears under it.
	// The ground walks toward the 900 end, so the label walks with it and
	// keeps its headroom instead of spending it.
	ghostTextOnWash = 900
)

// buttonColors returns the background and foreground colours for the given
// register and interaction state.
//
// Filled — the zero register — is the treatment prism has always drawn: the
// Primary solid fill resolved through the D2.3 state walk (ADR-007: hover
// and pressed step the pin toward the 900 end of the primary ramp; focus
// keeps the fill and draws the ring) under OnPrimary, faded to
// DisabledOpacity when disabled. Tonal and Ghost resolve through the same
// two entry points on the same ramps, only from different rungs: tonal is
// the tinted-ground walk on the primary ramp, ghost the same walk on the
// neutral one with the resting step painted as nothing at all.
//
// Ghost's wash is neutral rather than role-tinted on purpose. A ghost claims
// no role colour — that is what makes it the quiet register — and tinting
// one under the pointer would hand the brand hue to the very affordance that
// was chosen for not carrying it.
//
// The focus ring is not resolved here and does not vary by register: it is
// FocusRing in every one of them, drawn by the two draw functions.
func buttonColors(c tokens.ColorTokens, s RenderState) (bg, fg color.NRGBA) {
	state := interactionState(s)

	switch s.Emphasis {
	case Tonal:
		fg = c.Ramps.Primary.Step(tonalText)
		if s.Disabled {
			fg = tokens.Disabled(fg)
		}
		bg = c.StateColor(tokens.RolePrimary, tonalGround, state)

	case Ghost:
		switch state {
		case tokens.StateHover, tokens.StatePressed:
			fg = c.Ramps.Neutral.Step(ghostTextOnWash)
			bg = c.StateColor(tokens.RoleNeutral, ghostGround, state)
		default:
			// Rest, focus and disabled paint no ground: the surface behind
			// shows through untouched. A fully transparent fill is a no-op
			// over any ground, which is the whole point of the register.
			fg = c.Ramps.Neutral.Step(ghostText)
			if s.Disabled {
				fg = tokens.Disabled(fg)
			}
			bg = color.NRGBA{}
		}

	default: // Filled
		fg = c.OnPrimary
		if s.Disabled {
			fg = tokens.Disabled(fg)
		}
		bg = c.SolidStateColor(tokens.RolePrimary, state)
	}
	return
}

// interactionState collapses the four RenderState booleans into the one
// tokens.State the ramp walks take, in the precedence the component has
// always applied: disabled outranks pressed, pressed outranks hover, hover
// outranks focus.
func interactionState(s RenderState) tokens.State {
	switch {
	case s.Disabled:
		return tokens.StateDisabled
	case s.Pressed:
		return tokens.StatePressed
	case s.Hovered:
		return tokens.StateHover
	case s.Focused:
		return tokens.StateFocus
	}
	return tokens.StateNormal
}

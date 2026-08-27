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
	"github.com/vibrantgio/components/internal/focus"
	"github.com/vibrantgio/components/internal/hit"
	"github.com/vibrantgio/mvu"
	"github.com/vibrantgio/theme/theme"
	"github.com/vibrantgio/theme/tokens"
	"github.com/vibrantgio/theme/typeset"
)

// Emphasis is the visual weight register a button wears — how loudly it
// competes for attention on the surface it sits on. It is a colour property
// and nothing else: the drawn control keeps the density's size, the pointer
// target keeps its 44 dp floor, and the focus ring keeps its shape, width and
// place in every register — only its rung moves, and only so far as the
// ground under it moved. Keyboard visibility is not an emphasis property.
//
// Every desktop system carries this axis under its own names — MD3 has
// filled, tonal, outlined and text; Fluent primary, standard and subtle;
// Apple prominent, regular and plain — and the three registers below are
// the set all of them agree on: the two ends plus the tinted middle.
// Outlined is deliberately not a fourth register. A border is a property of
// a surface rather than a rung on a loudness scale, and components already
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
	// only while the pointer is on it. A ghost's wash is its host
	// surface's own one-rung walk — it derives from the local ground the
	// button sits on (RenderState.Ground), not the window ground, so a
	// ghost on a raised surface washes one rung past that surface's own
	// storey. For affordances that must be present
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

	// Ground is the elevation storey of the surface hosting the button —
	// the local ground the Ghost register's hover and press washes walk
	// from, in the same vocabulary the host names its own fill
	// (tokens.SurfaceAt). A ghost's wash is its host surface's own
	// one-rung walk: a dialog at tokens.Level2 passes Level2 and its
	// ghost washes one rung past the level's step, whichever storey that
	// is. The zero value is tokens.Level0, the window ground, which
	// resolves to exactly the walk the register always performed — so
	// every state written before this field existed keeps its colours.
	// Filled and Tonal ignore it: they carry their own grounds.
	Ground tokens.ElevationLevel

	// Fill and OnFill pin the Filled register's ground and the ink over it
	// to a pair the scheme does not carry: a colour fixed from outside the
	// palette, which a change of scheme must not move. The case they exist
	// for is an action whose colour is not the theme's to choose — a
	// destructive confirmation carrying the one red its platform pins for
	// that meaning, in both schemes, where a status role would hand it the
	// scheme's own idea of red instead.
	//
	// Nothing else about the register changes. The ground still walks
	// toward the 900 end under the pointer (tokens.PinnedStateColor: hover
	// one rung, press two, at the pin's own hue and chroma), still keeps
	// the pin at rest and under focus, still fades to the disabled opacity
	// with the ink; and the focus ring is still the rung that reads against
	// whatever ground came back, so a pinned fill is measured against
	// exactly as the primary one is.
	//
	// The two are one pin and are honoured together. Leave either half
	// unset — the zero value, alpha zero, which is no colour a fill could
	// use — and the register resolves from the primary role exactly as it
	// always has: that is what makes the pair invisible to every state
	// written before it existed, and it means a half-written pin renders
	// the stock button rather than an invisible label. Tonal and Ghost
	// ignore both. A register that paints a tint, or paints no ground at
	// all, has no solid fill to pin.
	Fill   color.NRGBA
	OnFill color.NRGBA

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

	// Ground is the elevation storey of the surface hosting the button,
	// copied straight into RenderState.Ground on every frame: the local
	// ground a Ghost's hover and press washes walk from. A container that
	// raises its surface (patterns/modal's level-2 dialog hosting its
	// close X) passes its own storey here; the zero value is the window
	// ground and keeps exactly the colours the register has always had.
	// See RenderState.Ground.
	Ground tokens.ElevationLevel

	// Fill and OnFill pin the Filled register's ground and its ink to a
	// pair the scheme does not carry, copied straight into RenderState on
	// every frame — for the action whose colour is not the theme's to
	// choose. They are one pin: set both or neither, and the zero value
	// keeps exactly the colours the register has always had. The hover,
	// press, focus and disabled treatments are the register's own either
	// way. See RenderState.Fill.
	Fill   color.NRGBA
	OnFill color.NRGBA

	// Icon, when non-nil and Label is empty, renders the button as a compact
	// icon-only affordance: a square the density's control height on a side
	// with the glyph centred, instead of a fill-width text label (the pointer
	// target stays at least the 44 dp square). The painter draws into
	// a sizePx×sizePx box at the current origin in colour col, via
	// clip.Path / clip.Stroke, so output stays golden-deterministic (no font or
	// SVG rasterisation). components/icon is the registry for named glyphs;
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
	// patterns/modal) drive focus and trap Tab without a doubled focus ring.
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
	// makes sharing it correct. See theme/tokens.Typography.Shaper.
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
						Ground:   props.Ground,
						Fill:     props.Fill,
						OnFill:   props.OnFill,
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
// through theme/typeset, which is where that discrepancy is documented.
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
// glyph patterns/modal's close affordance wants, over the same square and
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
	// theme/typeset.
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

	// Focus ring: the button's outermost 2 dp, in the primary rung that reads
	// against the ground it circles (focus.Ring). Same shape, same width and
	// same place in every emphasis register — keyboard visibility is not a
	// loudness property, so a ghost button's ring is exactly a filled one's.
	// The rung differs where the ground does, which is the only way the ring
	// can clear its floor over a filled ground as well as an empty one.
	if s.Focused {
		drawFocusRing(gtx, btnSize, rad, focus.Ring(tok.color, ringGround(tok.color, bg, s)))
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

	// Focus ring, matching drawButton.
	if s.Focused {
		drawFocusRing(gtx, sz, rad, focus.Ring(tok.color, ringGround(tok.color, bg, s)))
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

// drawFocusRing paints the focus ring of a button of size size and corner
// radius rad: a focus.Width stroke lying inside the button's own boundary,
// its own width clear of it, so the whole ring falls on the ground it was
// measured against with that ground on both sides of it.
//
// Inside, rather than centred on the boundary the way the grey ring was. A
// stroke centred on the edge spends half its width on the surface behind the
// button and half on the button's own fill, and those two grounds are never
// the same colour — so half the ring always dissolved into one of them and a
// 2 dp ring was never wider than 1 dp anywhere.
//
// Clear of the edge rather than flush with it, for the same reason the
// checkbox's ring is clear of its box: a band flush with a boundary is read
// as that boundary — a bevel, a seam, a slightly different edge — and not as
// a ring. It is a filled button that proves it. Its ring is the pale rung,
// the only one that reads against the primary fill it lies on, and against
// the page outside it measures 1.65:1: flush with the edge it merges with the
// page on one side and looks like the button's own rim. Held clear, it has
// the fill on both sides of it at the full measured ratio and reads as what
// it is. The gap costs the label nothing — a button is at least the density's
// control height and the ring rides in the padding.
func drawFocusRing(gtx layout.Context, size image.Point, rad int, ring color.NRGBA) {
	w := gtx.Dp(focus.Width)
	inset := w + w/2 // stroke centreline: the band spans w..2w inside the edge
	r := rad - inset
	if r < 0 {
		r = 0
	}
	rrect := clip.RRect{
		Rect: image.Rectangle{
			Min: image.Pt(inset, inset),
			Max: image.Pt(size.X-inset, size.Y-inset),
		},
		SE: r, SW: r, NE: r, NW: r,
	}
	paint.FillShape(gtx.Ops, ring, clip.Stroke{
		Path:  rrect.Path(gtx.Ops),
		Width: float32(w),
	}.Op())
}

// ringGround is the ground a focused button's ring circles: the register's own
// background. A ghost paints none at rest, so what its ring circles is the
// host surface showing through it — the storey s.Ground names, resolved by
// focus.Ground, which is the one rule every control in the library hands its
// ring and which lands on exactly the rung ghostGroundStep walks the ghost's
// wash from.
func ringGround(c tokens.ColorTokens, bg color.NRGBA, s RenderState) color.NRGBA {
	if bg.A == 0 {
		return focus.Ground(c, s.Ground)
	}
	return bg
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

	// ghostGround is the ground a ghost's wash walks from when its host
	// names no storey of its own. A ghost paints nothing at rest, so it
	// has no ground of its own and performs its host surface's walk
	// instead — a wash derives from the local ground it sits on, not the
	// window ground. A host that knows its storey says so through
	// RenderState.Ground and the wash walks from that storey's own step
	// (see ghostGroundStep); this constant is the assumption for the
	// window ground (tokens.Level0), whose Background pin is off-ramp
	// and cannot be walked. It assumes the level-1 surface step, so the
	// bare-background wash reads one step strong — the harmless
	// direction of the error, where assuming 100 would make the wash on
	// the card a ghost most often sits on invisible.
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
// Filled — the zero register — is the treatment components has always drawn: the
// Primary solid fill resolved through the D2.3 state walk (ADR-007: hover
// and pressed step the pin toward the 900 end of the primary ramp; focus
// keeps the fill and draws the ring) under OnPrimary, faded to
// DisabledOpacity when disabled. Tonal and Ghost resolve through the same
// two entry points on the same ramps, only from different rungs: tonal is
// the tinted-ground walk on the primary ramp, ghost the same walk on the
// neutral one with the resting step painted as nothing at all. The rung a
// ghost walks from is its host surface's — a ghost's wash is that
// surface's own one-rung walk, taken from whichever storey s.Ground names
// (ghostGroundStep), with the on-wash text riding at the ramp's 900 end,
// where the walk itself clamps.
//
// Filled is also the one register that takes a pin from the caller. A
// RenderState carrying both halves of a fill pair (RenderState.Fill and
// OnFill) wears that pair in place of the primary one and keeps everything
// else: the same walk toward the 900 end, now laddered on the scheme's own
// lightness scale because a caller's colour belongs to no role
// (tokens.PinnedStateColor), the same disabled opacity over both halves,
// and the same ring, which is measured against the ground this function
// returns and therefore against the pin. Half a pair is no pair; the
// register resolves from the primary role, exactly as every state written
// before the pair existed does.
//
// Ghost's wash is neutral rather than role-tinted on purpose. A ghost claims
// no role colour — that is what makes it the quiet register — and tinting
// one under the pointer would hand the brand hue to the very affordance that
// was chosen for not carrying it.
//
// The focus ring is not resolved here. Its shape, width and position are the
// same in every register (see drawFocusRing); its colour is the primary rung
// measured against the background this function returned, which is the only
// way one ring can read over a filled ground and an empty one alike.
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
			bg = c.StateColor(tokens.RoleNeutral, ghostGroundStep(s.Ground), state)
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
		if pinnedFill(s) {
			// The caller's pair replaces the role's, and replaces nothing
			// else: the walk, the opacity and the ring are the register's.
			fg = s.OnFill
			bg = c.PinnedStateColor(s.Fill, state)
		} else {
			fg = c.OnPrimary
			bg = c.SolidStateColor(tokens.RolePrimary, state)
		}
		if s.Disabled {
			fg = tokens.Disabled(fg)
		}
	}
	return
}

// pinnedFill reports whether the state carries a fill pin the Filled
// register should wear instead of the primary pair. Both halves must be
// there: a fill is no fill at alpha zero, and an ink at alpha zero would
// draw a label nobody can read, so a half-written pin is no pin — the
// register falls back to the role it has always resolved from rather than
// rendering something the caller cannot have meant.
func pinnedFill(s RenderState) bool {
	return s.Fill.A != 0 && s.OnFill.A != 0
}

// ghostGroundStep resolves the neutral-ramp step a ghost's wash walks from:
// the surface step of the hosting storey, so the wash is the host surface's
// own one-rung walk. Level 0 has no ramp step to walk — its fill is the
// Background pin, off-ramp by design — so the window ground keeps the
// register's long-standing level-1 assumption (ghostGround), which is also
// what makes the zero value colour-identical to every render made before
// the field existed.
func ghostGroundStep(level tokens.ElevationLevel) int {
	if step := tokens.Elevation.SurfaceStep(level); step != 0 {
		return step
	}
	return ghostGround
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

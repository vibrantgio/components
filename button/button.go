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
	"github.com/vibrantgio/components/internal/control"
	"github.com/vibrantgio/components/internal/focus"
	"github.com/vibrantgio/components/internal/surface"
	"github.com/vibrantgio/components/internal/toolbarface"
	"github.com/vibrantgio/mvu"
	vgcolor "github.com/vibrantgio/theme/color"
	"github.com/vibrantgio/theme/theme"
	"github.com/vibrantgio/theme/tokens"
	"github.com/vibrantgio/theme/typeset"
)

// Emphasis is how pronounced a button is — how strongly it competes for
// attention on the surface it sits on. It is a colour property and nothing
// else: the drawn control keeps the density's size, the pointer target keeps
// that size with it, and the focus ring keeps its shape, width and place in
// every variant. Keyboard visibility is not an emphasis property.
//
// The three variants are the three buttons the platform draws: the default
// action it fills with the accent, the ordinary button beside it, and the
// borderless kind that carries no fill at all.
type Emphasis int

const (
	// Filled is the most pronounced variant and the zero value: the
	// platform's default action, filled with the accent and labelled in the
	// foreground the platform pairs with an accent fill. One per surface —
	// the action the screen is about.
	Filled Emphasis = iota

	// Tonal is the middle variant: the platform's ordinary button — the
	// push button's own measured fill under the platform's control text,
	// inside the platform's hairline. The variant for a secondary action,
	// and the one a row of equals wears.
	Tonal

	// Ghost is the least pronounced variant: the platform's borderless
	// button — no fill and no hairline, the label or glyph in the
	// platform's control text. For affordances that must be present without
	// being the subject — a dialog's close X, a toolbar of icons, a "Learn
	// more". A ghost is less pronounced, not small: it draws the same square
	// as a filled button, so it offers the same pointer target, and it keeps
	// the full focus ring.
	Ghost
)

// String returns the name of the emphasis in the vocabulary the design
// system uses everywhere else — the same three words the gallery's captions
// carry.
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

// Variant is where the button stands — a setting, never a behaviour and never
// a prominence. The zero value is [Form].
//
// It is the Variant the picker and the search field carry under the same two
// names: one control drawn in two places, and the place settles what the
// platform draws it as.
type Variant uint8

const (
	// Form is the button among content: the platform's push button, at the
	// density's control height, wearing its [Emphasis].
	Form Variant = iota

	// Chrome is the button standing in a chrome region — a toolbar, a navbar.
	// A button whose label is a SYMBOL is drawn there as the platform's
	// bordered toolbar control: a capsule at
	// [tokens.Density.ToolbarControlHeight] with the symbol centred in it,
	// wearing the toolbar control's own fill, its rim where the platform
	// draws one, and the drop shadow it casts on its band — the same box the
	// picker's chrome trigger is drawn from, through
	// components/internal/toolbarface.
	//
	// It reaches the symbol path alone. A button carrying TEXT draws the form
	// variant whatever this says: every symbol-labelled control in the stored
	// toolbar bands is bordered, and no stored band holds a toolbar button
	// with a word in it to measure one from.
	//
	// The fill, the rim and the shadow are the platform's for that control,
	// so [Emphasis] reaches nothing here — there is one bordered toolbar
	// control and the platform draws it one way.
	Chrome
)

// String returns the name of the variant in the vocabulary the design system
// uses everywhere else.
func (v Variant) String() string {
	switch v {
	case Form:
		return "form"
	case Chrome:
		return "chrome"
	}
	return fmt.Sprintf("Variant(%d)", int(v))
}

// RenderState holds the explicit visual state a static render draws in: the
// emphasis the button wears and the interaction state it is in. The zero
// value is Filled emphasis at rest, so RenderState{} is exactly today's
// default button.
//
// Intended for golden-image testing and static rendering; production code
// obtains the interaction half from the Gio event system via Button, which
// copies the emphasis straight off Props.Emphasis.
type RenderState struct {
	// Emphasis is how pronounced the button is. It is a property of the
	// button rather than of the pointer, and it lives here because Render
	// and RenderIcon take exactly one parameter that is not a token and
	// this is it. Zero is Filled.
	Emphasis Emphasis

	// Variant is where the button stands: [Form] (the zero value) among
	// content, [Chrome] in a chrome region. It is read on the symbol path
	// alone — see [Chrome].
	Variant Variant

	// Fill and Foreground pin the Filled emphasis' fill and the foreground over
	// it to a pair the scheme does not carry: a colour fixed from outside the
	// palette, which a change of scheme must not move. The case they exist
	// for is an action whose colour is not the theme's to choose — a
	// destructive confirmation carrying the one red its platform pins for
	// that meaning, in both schemes, where a status role would hand it the
	// scheme's own idea of red instead.
	//
	// Nothing else about the emphasis changes. The pin still takes the
	// platform's press overlay while the button is held, still keeps the
	// pin at rest, under the pointer and under focus, and still gives way
	// to the platform's disabled pair when the button is disabled; and the
	// focus ring is the platform's own, which composites over whatever fill
	// came back.
	//
	// The two are one pin and are honoured together. Leave either half
	// unset — the zero value, alpha zero, which is no colour a fill could
	// use — and the emphasis takes the platform's accent pair instead, so a
	// half-written pin renders the stock button rather than an invisible
	// label. Tonal and Ghost
	// ignore both: neither carries a fill of its own to pin.
	Fill       color.NRGBA
	Foreground color.NRGBA

	// Surface is the opaque fill the button stands on. The platform's press
	// overlay, its seam, its control text and its disabled text all carry a
	// coverage rather than a colour, so what each lands as depends on what is
	// under it; the button flattens them and hands Gio opaque fills. It
	// matters wherever the button carries no fill of its own — a Ghost
	// button's label and a held Ghost button's tint stand straight on it. The
	// zero value — no colour — is the window's own plane.
	Surface color.NRGBA

	Hovered  bool
	Focused  bool
	Pressed  bool
	Disabled bool

	// Checked is the persistent state of a control that records a yes — the
	// toolbar toggle that says whether the pane it governs stands. It is
	// read on the chrome symbol path alone, where the platform draws it as
	// the chosen segment of a segmented control: a patch inside the
	// control's own box, laid over whatever fill the control carries. The
	// form variants carry no such drawing and ignore it.
	Checked bool
}

// Props configures a Button instance.
type Props struct {
	// Label is the text rendered inside the button.
	Label string

	// Description is the screen-reader label. Falls back to Label when empty.
	Description string

	// Emphasis is how pronounced the button is: Filled (the zero value) for
	// the one action a surface is about, Tonal for a secondary action,
	// Ghost for an affordance that must be present without being the
	// subject. It changes colour only — never the drawn size, never the
	// pointer target, never the focus ring. Composes with Icon: a ghost
	// icon button is a less pronounced glyph over the same square.
	Emphasis Emphasis

	// Variant is where the button stands: [Form] (the zero value) among
	// content, [Chrome] in a chrome region, copied straight into RenderState
	// on every frame. A chrome button whose label is a symbol is drawn as the
	// platform's bordered toolbar control; one carrying text draws the form
	// variant. See [Chrome].
	Variant Variant

	// Fill and Foreground pin the Filled emphasis' fill and its foreground to a
	// pair the scheme does not carry, copied straight into RenderState on
	// every frame — for the action whose colour is not the theme's to
	// choose. They are one pin: set both or neither, and the zero value
	// keeps exactly the colours the emphasis has always had. The hover,
	// press, focus and disabled treatments are the emphasis' own either
	// way. See RenderState.Fill.
	Fill       color.NRGBA
	Foreground color.NRGBA

	// Surface is the opaque fill the button stands on, copied straight into
	// RenderState on every frame. Set it where the button does not stand on
	// the window's own plane — on a card, a selected row, a coloured fill —
	// so that the platform's press overlay, seam and text land as they do
	// there. See RenderState.Surface.
	Surface color.NRGBA

	// Icon, when non-nil and Label is empty, renders the button as a compact
	// icon-only affordance: a square the density's control height on a side
	// with the glyph centred, instead of a fill-width text label; that square
	// is the pointer target. The painter draws into
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
	// map function makes of it. Set it only when this button must shape with
	// a different shaper than the theme provides.
	//
	// A shaper is not safe to use from two goroutines; Gio lays the layout
	// tree out on the one goroutine that runs the event loop, which is what
	// makes sharing it correct. See theme/tokens.Typography.Shaper.
	Shaper *text.Shaper
}

// resolvedTokens is the concrete per-emission snapshot consumed by the
// layout.Widget closure.
type resolvedTokens struct {
	platform tokens.PlatformColors
	label    tokens.TextStyle // the LabelLarge role: typeface, weight, size, line height
	spacing  tokens.SpacingScale
	radius   tokens.RadiusScale
	density  tokens.Density // control height and inner padding
	shaper   *text.Shaper   // the theme's shaper; nil in the Render/RenderIcon path
}

// Button returns an rx.Observable[layout.Widget] that emits a new
// layout.Widget whenever the theme or disabled state changes. Interaction
// state (clickable, hover, focus, press) lives in the rx.Defer scope and
// persists across emissions.
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
	// theme's cached shaper — the theme owns the typeface.
	resolved := rx.SwitchMap(th, func(t theme.Theme) rx.Observable[resolvedTokens] {
		return rx.Map(
			rx.CombineLatest5(t.Platform, t.Typography, t.Spacing, t.Radius, t.Density),
			func(n rx.Tuple5[tokens.PlatformColors, tokens.Typography, tokens.SpacingScale, tokens.RadiusScale, tokens.Density]) resolvedTokens {
				typ := n.Second
				return resolvedTokens{
					platform: n.First,
					label:    typ.LabelLarge,
					spacing:  n.Third,
					radius:   n.Fourth,
					density:  n.Fifth,
					shaper:   typ.Shaper(),
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

				state := RenderState{
					Emphasis:   props.Emphasis,
					Variant:    props.Variant,
					Fill:       props.Fill,
					Foreground: props.Foreground,
					Surface:    props.Surface,
					Hovered:    hov,
					Focused:    foc,
					Pressed:    prs,
					Disabled:   dis,
				}
				chrome := iconOnly && state.Variant == Chrome

				// The clickable covers the drawn button exactly: a
				// control's pointer target is the control, so density
				// moves the target with the pixels.
				body := func(gtx layout.Context) layout.Dimensions {
					return click.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						semantic.ClassOp(semantic.Button).Add(gtx.Ops)
						semantic.LabelOp(props.Label).Add(gtx.Ops)
						semantic.DescriptionOp(desc).Add(gtx.Ops)
						semantic.EnabledOp(!dis).Add(gtx.Ops)
						if chrome {
							return drawChromeIcon(gtx, props.Icon, tok, state)
						}
						if iconOnly {
							return drawIconButton(gtx, props.Icon, tok, state)
						}
						return drawButton(gtx, shaper, props.Label, tok, state)
					})
				}
				if chrome {
					// The shadow the chrome variant casts falls outside the
					// control's own box, and a Clickable clips what it wraps
					// to that box, so it is cast around the clickable.
					return toolbarface.Cast(gtx, chromeShadow(tok.platform, state), body)
				}
				return body(gtx)
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
// tokens.PlatformLight, tokens.DefaultTypography.LabelLarge and
// tokens.Comfortable for the default desktop look. s carries the emphasis alongside the interaction state; its
// zero value is the filled button, so a call written before emphasis existed
// draws what it always drew.
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
// and the second term wins for Compact at any of the label roles: LabelLarge's
// 20 dp line box against a 19 dp control height.
// [tokens.Density.ControlHeight] is a floor, not a height.
func Render(
	shaper *text.Shaper,
	label string,
	p tokens.PlatformColors,
	sp tokens.SpacingScale,
	rad tokens.RadiusScale,
	labelStyle tokens.TextStyle,
	d tokens.Density,
	s RenderState,
) layout.Widget {
	tok := resolvedTokens{platform: p, spacing: sp, radius: rad, label: labelStyle, density: d}
	return func(gtx layout.Context) layout.Dimensions {
		return drawButton(gtx, shaper, label, tok, s)
	}
}

// RenderIcon produces a layout.Widget for a compact icon-only button in an
// explicit visual state, without event processing or rx machinery. The glyph
// is drawn by icon into a square d.ControlHeight on a side, inset by
// d.PaddingY, in the emphasis s carries — a ghost icon button is the less
// pronounced glyph patterns/modal's close affordance wants, over the same
// square as a filled one, which is the pointer target in both.
// Pass tokens.Comfortable for the default desktop look. Intended
// for golden-image testing and static demonstrations; production code should
// use Button with Props.Icon (and, when a container drives focus,
// Props.Clickable).
//
// It takes no text style: an icon-only button draws no text, so unlike
// [Render] there is nothing for a tokens.TextStyle to reach.
func RenderIcon(
	icon func(gtx layout.Context, sizePx int, col color.NRGBA),
	p tokens.PlatformColors,
	sp tokens.SpacingScale,
	rad tokens.RadiusScale,
	d tokens.Density,
	s RenderState,
) layout.Widget {
	tok := resolvedTokens{platform: p, spacing: sp, radius: rad, density: d}
	return func(gtx layout.Context) layout.Dimensions {
		return drawIconButton(gtx, icon, tok, s)
	}
}

// drawButton renders the button visual into gtx. All visual state comes from s;
// no event queries are performed here.
func drawButton(gtx layout.Context, shaper *text.Shaper, label string, tok resolvedTokens, s RenderState) layout.Dimensions {
	// Sizing rule: button height = Density.ControlHeight (24 dp
	// Comfortable, 19 dp Compact — the platform's regular and small push
	// button), inner padding = Density.PaddingX/PaddingY. The drawn box is
	// also the pointer target: the live path's clickable covers it and
	// nothing more.
	padH := gtx.Dp(unit.Dp(tok.density.PaddingX))
	padV := gtx.Dp(unit.Dp(tok.density.PaddingY))
	minH := gtx.Dp(unit.Dp(tok.density.ControlHeight))
	rad := gtx.Dp(unit.Dp(tok.radius.Md)) // 6 dp corner radius

	bg, edge, fg, ring := buttonColors(tok.platform, s)

	// Record the label's paint material — replayed inside the label layout.
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
	// of the label box and Gio alone reports the drawn glyph extent instead — see
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

	// The fill, then the hairline the platform draws around an ordinary
	// button. Both are opaque: buttonColors has already laid the platform's
	// press overlay into the fill and the seam onto that, so nothing here
	// asks Gio to composite a coverage.
	rrect := clip.RRect{Rect: image.Rectangle{Max: btnSize}, SE: rad, SW: rad, NE: rad, NW: rad}
	paint.FillShape(gtx.Ops, bg, rrect.Op(gtx.Ops))
	if edge.A != 0 {
		strokeRRect(gtx, btnSize, rad, edge)
	}

	// Focus ring: the button's outermost 2 dp, inset in its own background.
	// Same shape, same width and same place in every emphasis — keyboard
	// visibility is not a prominence property, so a ghost button's ring is
	// exactly a filled one's.
	if s.Focused {
		drawFocusRing(gtx, btnSize, rad, ring)
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
// padding. Shares buttonColors with the text button so the emphasis and the
// press, focus and disabled treatments match. All visual state comes from s;
// no event queries are performed here.
func drawIconButton(gtx layout.Context, icon func(gtx layout.Context, sizePx int, col color.NRGBA), tok resolvedTokens, s RenderState) layout.Dimensions {
	// Sizing rule: side = Density.ControlHeight, glyph inset =
	// Density.PaddingY, so the glyph gets ControlHeight − 2·PaddingY — the
	// same content-box rule icon.Size documents. That square is the pointer
	// target in every emphasis: emphasis reaches the colours and stops
	// there, so the glyph grows less pronounced and the square does not
	// shrink.
	pad := gtx.Dp(unit.Dp(tok.density.PaddingY))
	side := gtx.Dp(unit.Dp(tok.density.ControlHeight))
	rad := gtx.Dp(unit.Dp(tok.radius.Md)) // 6 dp corner radius
	sz := image.Pt(side, side)

	bg, edge, fg, ring := buttonColors(tok.platform, s)

	rrect := clip.RRect{Rect: image.Rectangle{Max: sz}, SE: rad, SW: rad, NE: rad, NW: rad}
	paint.FillShape(gtx.Ops, bg, rrect.Op(gtx.Ops))
	if edge.A != 0 {
		strokeRRect(gtx, sz, rad, edge)
	}

	// Focus ring, matching drawButton.
	if s.Focused {
		drawFocusRing(gtx, sz, rad, ring)
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
// its own width clear of it, so the whole ring falls on the button's own fill
// with that fill on both sides of it.
//
// Inside, rather than centred on the boundary. A stroke centred on the edge
// spends half its width on the surface behind the button and half on the
// button's own fill, and those two are never the same colour — so half the
// ring dissolves into one of them and a 2 dp ring is nowhere wider than 1 dp.
//
// Clear of the edge rather than flush with it, for the same reason the
// checkbox's ring is clear of its box: a band flush with a boundary is read
// as that boundary — a bevel, a seam, a slightly different edge — and not as
// a ring.
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

// strokeRRect draws a one-hair line around a rounded rectangle of size size,
// wholly inside it: the button's painted footprint is size, not size plus
// half a line.
//
// A stroke is centred on the path it follows, so a hairline drawn at its own
// width spends half of itself outside the box the button reports. The
// platform's hairline carries a coverage rather than a colour, so half laid
// on the button's fill and half on whatever stands behind it is two
// different lines; and at 1x, where the hairline is one device pixel, the
// outside half is a half-covered row above the button and another below it,
// which measures a Tonal button 2 px taller than the Filled one beside it.
// Insetting the path cannot fix that at 1x — half of one pixel is not an
// integer coordinate — so the line is stroked at twice its width under a
// clip of the button's own shape, which takes the outside half away. The
// same idiom the picker's focus band and patterns' group hairline are drawn
// with; the checkbox reaches the same place by insetting one fill inside
// another.
func strokeRRect(gtx layout.Context, size image.Point, rad int, col color.NRGBA) {
	w := gtx.Dp(unit.Dp(1))
	if w < 1 {
		w = 1
	}
	rrect := clip.RRect{Rect: image.Rectangle{Max: size}, SE: rad, SW: rad, NE: rad, NW: rad}
	stroke := clip.Stroke{Path: rrect.Path(gtx.Ops), Width: float32(2 * w)}.Op()
	area := rrect.Push(gtx.Ops)
	paint.FillShape(gtx.Ops, col, stroke)
	area.Pop()
}

// buttonColors returns what the button paints for the given variant and
// interaction state: the fill, the hairline around it, the foreground of the
// label or glyph, and the ring a focused button wears. An unused part comes
// back at alpha zero, which is no colour a fill could use.
//
// Every one of them is opaque. The platform's press overlay, seam, control
// text and disabled text each carry a coverage, and the platform composites
// those in encoded sRGB where Gio's rasterizer would composite them in
// linear light, so each is flattened here onto the fill it actually lands
// on — the button's own where it has one and RenderState.Surface where it
// does not — and Gio is handed a colour rather than a coverage.
//
// Every one of them is a platform name:
//
//	Filled   the accent fill under the foreground the platform pairs with it
//	Tonal    the push button's measured fill and the platform's control
//	         text, inside its hairline
//	Ghost    no fill and no hairline, the platform's control text
//
// Under the pointer the button takes the platform's hover overlay over
// whatever fill it carries, and held down its press overlay over that same
// fill; a press wins, because the two do not stack. MEASURED,
// control-hover-{light,dark}.png: a Finder toolbar pop-up, which carries no
// fill of its own at rest, reads #f2f2f2 on the #ffffff band light and
// #384146 on the #242d32 band dark — the overlay landing on the surface the
// control stands on, which is what a Ghost button does here. MEASURED,
// control-pressed-{light,dark}.png: a held push button reads #d5d5d5 and
// #474d52, the press overlay straight over the push button's fill with no
// hover beneath it. No capture holds a push button under the pointer and not
// held, so nothing measures one as exempt and the overlay is applied here
// like everywhere else.
//
// Disabled fades the control toward the surface it stands on: the fill and
// the hairline at the platform's measured disabled coverage (control.Faded),
// the foreground at the platform's disabled control text. The platform draws
// a disabled default action as an ordinary disabled button, so every variant
// that carries a fill falls back to the push button's fill first and fades
// from there — which is what parts a disabled button from an enabled Tonal
// one, the two having been the same pixel before. MEASURED,
// save-dialog-{light,dark}.png: the switched-off "Options:" checkbox reads
// #f2f2f2 and #2e3439 against the enabled pop-up's #ececec and #333a3f
// seventeen rows above it on the same sheet.
//
// Filled is the one variant that takes a pin from the caller. A RenderState
// carrying both halves of a fill pair (RenderState.Fill and Foreground) wears
// that pair in place of the accent's and keeps everything else: the same
// press overlay, the same disabled pair, and the same ring, which carries
// its own coverage and composites over whatever fill is there. Half a pair
// is no pair.
//
// Focus is a persistent state and not a treatment that replaces another: in
// every variant it keeps the resting fill and adds the ring.
func buttonColors(p tokens.PlatformColors, s RenderState) (bg, edge, fg, ring color.NRGBA) {
	standsOn := surface.Or(s.Surface, p.WindowBackground)

	var edged bool
	switch s.Emphasis {
	case Tonal:
		bg, edged, fg = p.PushButtonFill, true, p.ControlText

	case Ghost:
		fg = p.ControlText

	default: // Filled
		if pinnedFill(s) {
			bg, fg = s.Fill, s.Foreground
		} else {
			bg, fg = p.ControlAccent, p.AlternateSelectedControlText
		}
	}

	if s.Disabled {
		fg = p.DisabledControlText
		if s.Emphasis != Ghost {
			bg, edged = control.Faded(p.PushButtonFill, standsOn), true
		}
	}

	// The pointer overlays go onto whatever fill the variant carries, and
	// straight onto the surface where it carries none — which is how a held
	// or hovered Ghost button gets a fill at all. A press wins over a hover:
	// the two are one answer and not two laid on each other. Everything the
	// button draws over that result is then flattened onto it, hairline
	// included: the platform's seam over a held button is over the held fill.
	if !s.Disabled {
		switch {
		case s.Pressed:
			bg = vgcolor.Flatten(p.PressOverlay, fillOr(bg, standsOn))
		case s.Hovered:
			bg = vgcolor.Flatten(p.HoverOverlay, fillOr(bg, standsOn))
		}
	}
	beneath := fillOr(bg, standsOn)
	if edged {
		seam := p.Separator
		if s.Disabled {
			seam = vgcolor.Fade(seam, tokens.DisabledCoverage)
		}
		edge = vgcolor.Flatten(seam, beneath)
	}
	return bg, edge, vgcolor.Flatten(fg, beneath), focus.Ring(p, beneath)
}

// fillOr returns the button's own fill, or what it stands on where it has
// none: alpha zero is no fill, so the surface is what lies under the part
// being drawn.
func fillOr(bg, standsOn color.NRGBA) color.NRGBA {
	if bg.A == 0 {
		return standsOn
	}
	return bg
}

// pinnedFill reports whether the state carries a fill pin the Filled
// emphasis should wear instead of the platform's accent pair. Both halves
// must be there: a fill is no fill at alpha zero, and a foreground at alpha
// zero would draw a label nobody can read, so a half-written pin is no pin.
func pinnedFill(s RenderState) bool {
	return s.Fill.A != 0 && s.Foreground.A != 0
}

package picker

import (
	"image"
	"image/color"

	"gioui.org/f32"
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
	"github.com/vibrantgio/components/internal/hit"
	"github.com/vibrantgio/mvu"
	"github.com/vibrantgio/theme/theme"
	"github.com/vibrantgio/theme/tokens"
	"github.com/vibrantgio/theme/typeset"
)

// fieldChevron is the box the field trigger's mark is drawn in: a solid
// downward triangle, the form register's own mark rather than the chrome
// register's paired chevrons. It is a fixed size because the trigger's other
// numbers come from the density and the mark is what tells the two registers
// apart at a glance.
const fieldChevron = unit.Dp(16)

// Drop is the direction an open [Field] stacks its menu in.
//
// It answers a question only the caller can see: whether the room beneath the
// trigger is room the menu may have. A field at the foot of a dialog whose
// action row is drawn after the body has none — a menu dropped there is
// painted over by the row — so that caller says [DropUp] and the menu stands
// above instead.
//
// Either way the open field is the trigger plus the menu, stacked, and the
// widget reports both; what changes is the order. [DropUp] therefore puts the
// TRIGGER at the bottom of the reported box, so a caller placing an upward
// field aligns that box's BOTTOM edge with the row the trigger stands in —
// record the widget, read the height it reports, and offset by it.
type Drop uint8

const (
	// DropDown is the zero value: the menu stands directly beneath the
	// trigger, which is what a form's select does.
	DropDown Drop = iota

	// DropUp stands the menu directly above the trigger.
	DropUp
)

// FieldState holds the explicit visual state a static field render draws in.
// All fields default to their zero values (normal, closed, idle, on the window
// ground).
//
// Intended for golden-image testing; production code obtains state from the
// Gio event system via [Field].
type FieldState struct {
	Open     bool
	Focused  bool
	Disabled bool
	Selected int
	Options  []string

	// Drop is which way the open menu stacks. The zero value is [DropDown],
	// beneath the trigger. See [Drop].
	Drop Drop

	// Ground is the elevation storey of the surface hosting the trigger —
	// the local ground its resting border is derived against, in the same
	// vocabulary the host names its own fill (tokens.SurfaceAt). A dialog at
	// tokens.Level2 passes Level2 and the border takes whichever neutral
	// rung clears the floor over that storey. The zero value is
	// tokens.Level0, the window ground. It governs the trigger only: the
	// open menu is its own plane, a level-3 overlay that draws no edge and
	// separates by fill (see optionRowColors).
	Ground tokens.ElevationLevel
}

// FieldProps configures a [Field] instance.
type FieldProps struct {
	// Description is the screen-reader label.
	Description string

	// Options is the list of selectable items.
	Options []string

	// Selected is the initial selected index established on subscribe.
	Selected int

	// Drop is which way the open menu stacks, copied straight into
	// [FieldState.Drop] on every frame. The zero value is [DropDown]. A caller
	// with no room beneath the trigger says [DropUp] and then places the
	// widget by its bottom edge. See [Drop].
	Drop Drop

	// Ground is the elevation storey of the surface hosting the field, copied
	// straight into [FieldState.Ground] on every frame: the local ground the
	// trigger's resting border is derived against. A container that raises its
	// surface passes its own storey here; the zero value is the window ground.
	// See [FieldState.Ground].
	Ground tokens.ElevationLevel

	// Disabled, if non-nil, disables the field when it emits true.
	Disabled rx.Observable[bool]

	// OnSelect is called with the newly selected index on every selection.
	// This is the FRP path. The gtx argument is the layout.Context active on
	// the frame the selection is processed in, so a consumer may emit
	// mvu.MessageOp{Message: …}.Add(gtx.Ops) from inside it.
	OnSelect func(gtx layout.Context, index int)

	// Message, if non-nil, is emitted as mvu.MessageOp into the frame's ops on
	// every selection — the MVU path, where OnSelect is the FRP one.
	Message any

	// Shaper is an explicit per-instance override of the text shaper. Leave it
	// nil in normal use: the field then shapes its text with the theme's
	// shaper (Typography.Shaper()), which is built once for the process and
	// shared by every component reading that typography — the cache lives
	// behind the Typography value, so it survives the copy this component's
	// map function makes of it. Set it only when this field must shape with a
	// different shaper than the theme provides.
	//
	// A shaper is not safe to use from two goroutines; Gio lays the widget
	// forest out on the one goroutine that runs the event loop, which is what
	// makes sharing it correct. See theme/tokens.Typography.Shaper.
	Shaper *text.Shaper
}

// Field returns an rx.Observable[layout.Widget] that emits a new widget
// whenever the theme or disabled state changes: the form register's picker —
// the flat trigger bar and, while it is open, the [Menu] it stacks against
// itself, beneath by default and above under [DropUp]. Widget state
// (open/closed, selected index, focus) lives in the rx.Defer scope and
// persists across emissions.
//
// A trigger that must stand under a floating menu instead — one placed against
// the window by patterns/popover — is [Anchor]; see the package doc.
//
// Both integration paths are supported:
//   - FRP: set FieldProps.OnSelect.
//   - MVU: set FieldProps.Message; the field emits mvu.MessageOp on selection.
//
// Keyboard reach through the open menu is [Menu]'s, and its doc states what it
// covers and what it does not.
func Field(th rx.Observable[theme.Theme], props FieldProps) rx.Observable[layout.Widget] {
	disabled := props.Disabled
	if disabled == nil {
		disabled = rx.Of(false)
	}

	resolved := rx.SwitchMap(th, func(t theme.Theme) rx.Observable[resolvedTokens] {
		return rx.Map(
			rx.CombineLatest5(t.Color, t.Typography, t.Spacing, t.Radius, t.Density),
			func(n rx.Tuple5[tokens.ColorTokens, tokens.Typography, tokens.SpacingScale, tokens.RadiusScale, tokens.Density]) resolvedTokens {
				typ := n.Second
				return resolvedTokens{
					color:   n.First,
					body:    typ.BodyLarge,
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
		var trigger widget.Clickable
		optClicks := make([]widget.Clickable, len(props.Options))
		var open bool
		selected := props.Selected

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

				for trigger.Clicked(gtx) {
					open = !open
				}
				for i := range optClicks {
					for optClicks[i].Clicked(gtx) {
						selected = i
						open = false
						if props.OnSelect != nil {
							props.OnSelect(gtx, i)
						}
						if props.Message != nil {
							mvu.MessageOp{Message: props.Message}.Add(gtx.Ops)
						}
					}
				}

				foc := !dis && gtx.Focused(&trigger)

				return layoutFieldLive(gtx, shaper, &trigger, optClicks, tok, props.Description, FieldState{
					Open:     open,
					Focused:  foc,
					Disabled: dis,
					Selected: selected,
					Options:  props.Options,
					Drop:     props.Drop,
					Ground:   props.Ground,
				})
			}
		})
	})
}

// RenderField produces a layout.Widget for the form register's picker in an
// explicit visual state, without any event processing or rx machinery.
// Intended for golden-image testing and static demonstrations; production code
// should use [Field], which reads both of the parameters below off the theme.
//
// body is the BodyLarge role's whole text style — typeface, weight, size and
// line height all reach the shaper — and d is the density the trigger and the
// option rows draw at. Pass tokens.DefaultTypography.BodyLarge and
// tokens.Comfortable for the default desktop look.
func RenderField(
	shaper *text.Shaper,
	colors tokens.ColorTokens,
	sp tokens.SpacingScale,
	rad tokens.RadiusScale,
	body tokens.TextStyle,
	d tokens.Density,
	s FieldState,
) layout.Widget {
	tok := resolvedTokens{color: colors, spacing: sp, radius: rad, body: body, density: d}
	return func(gtx layout.Context) layout.Dimensions {
		return drawField(gtx, shaper, tok, s)
	}
}

// layoutFieldLive lays out the interactive field with Clickable hit areas.
func layoutFieldLive(gtx layout.Context, shaper *text.Shaper, trigger *widget.Clickable, optClicks []widget.Clickable, tok resolvedTokens, desc string, s FieldState) layout.Dimensions {
	// The trigger's pointer area is at least MinHitTarget (44 dp) on each
	// axis, centred on the visual bar: density shrinks the drawn trigger,
	// never the hit target. The menu's rows are not extended — see
	// layoutMenuLive.
	triggerMacro := op.Record(gtx.Ops)
	triggerDims := hit.Extend(gtx, gtx.Dp(unit.Dp(tok.density.MinHitTarget())), trigger.Layout, func(gtx layout.Context) layout.Dimensions {
		semantic.Button.Add(gtx.Ops)
		if desc != "" {
			semantic.DescriptionOp(desc).Add(gtx.Ops)
		}
		return drawTrigger(gtx, shaper, tok, s)
	})
	triggerCall := triggerMacro.Stop()

	if !s.Open || len(s.Options) == 0 {
		triggerCall.Add(gtx.Ops)
		return triggerDims
	}

	menuMacro := op.Record(gtx.Ops)
	menuDims := layoutMenuLive(gtx, shaper, optClicks, tok, MenuState{Options: s.Options, Selected: s.Selected})
	menuCall := menuMacro.Stop()

	return stackOpen(gtx, s.Drop, triggerCall, triggerDims, menuCall, menuDims)
}

// stackOpen places the recorded trigger and menu in the [Drop]'s order and
// reports the whole stack, because what was drawn is what a container has to
// make room for. Under [DropUp] the trigger is the LOWER half, which is what
// lets an upward field be placed by the bottom edge of the box it reports.
func stackOpen(gtx layout.Context, d Drop, trigger op.CallOp, triggerDims layout.Dimensions, menu op.CallOp, menuDims layout.Dimensions) layout.Dimensions {
	triggerY, menuY := 0, triggerDims.Size.Y
	if d == DropUp {
		triggerY, menuY = menuDims.Size.Y, 0
	}
	off := op.Offset(image.Pt(0, triggerY)).Push(gtx.Ops)
	trigger.Add(gtx.Ops)
	off.Pop()
	off = op.Offset(image.Pt(0, menuY)).Push(gtx.Ops)
	menu.Add(gtx.Ops)
	off.Pop()
	return layout.Dimensions{Size: image.Pt(gtx.Constraints.Max.X, triggerDims.Size.Y+menuDims.Size.Y)}
}

// drawField renders the static field — the trigger and, when open, the menu
// under it — for golden-image testing.
func drawField(gtx layout.Context, shaper *text.Shaper, tok resolvedTokens, s FieldState) layout.Dimensions {
	triggerMacro := op.Record(gtx.Ops)
	triggerDims := drawTrigger(gtx, shaper, tok, s)
	triggerCall := triggerMacro.Stop()

	if !s.Open || len(s.Options) == 0 {
		triggerCall.Add(gtx.Ops)
		return triggerDims
	}

	menuMacro := op.Record(gtx.Ops)
	menuDims := drawMenu(gtx, shaper, tok, MenuState{Options: s.Options, Selected: s.Selected})
	menuCall := menuMacro.Stop()

	return stackOpen(gtx, s.Drop, triggerCall, triggerDims, menuCall, menuDims)
}

// drawTrigger renders the field trigger bar (the closed face).
func drawTrigger(gtx layout.Context, shaper *text.Shaper, tok resolvedTokens, s FieldState) layout.Dimensions {
	// The trigger follows the text field's sizing rule — height =
	// Density.ControlHeight, vertical padding = Density.PaddingY, horizontal
	// padding a static spacing.S3 (12 dp).
	padH := gtx.Dp(unit.Dp(tok.spacing.S3))
	padV := gtx.Dp(unit.Dp(tok.density.PaddingY))
	rad := gtx.Dp(unit.Dp(tok.radius.Md))
	// Shape with the BodyLarge role's typeface, weight, size and line height.
	f, wl, textSize := bodyLabel(tok)
	minH := gtx.Dp(unit.Dp(tok.density.ControlHeight))
	fieldW := gtx.Constraints.Max.X
	chevronSz := gtx.Dp(fieldChevron)

	selectedText := ""
	if len(s.Options) > 0 && s.Selected >= 0 && s.Selected < len(s.Options) {
		selectedText = s.Options[s.Selected]
	}

	textCol := tok.color.Text
	if s.Disabled {
		textCol = tokens.Disabled(textCol)
	}

	// Reserve space for chevron: padH on the right side plus chevron width.
	innerW := fieldW - 2*padH - chevronSz - padH
	if innerW < 1 {
		innerW = 1
	}
	innerGtx := gtx
	innerGtx.Constraints = layout.Constraints{
		Min: image.Pt(0, 0),
		Max: image.Pt(innerW, gtx.Constraints.Max.Y),
	}

	mTextCol := op.Record(gtx.Ops)
	paint.ColorOp{Color: textCol}.Add(gtx.Ops)
	textMat := mTextCol.Stop()

	mLabel := op.Record(gtx.Ops)
	labelDims := typeset.Layout(innerGtx, shaper, wl, f, textSize, selectedText, textMat)
	labelCall := mLabel.Stop()

	triggerH := labelDims.Size.Y + 2*padV
	if triggerH < minH {
		triggerH = minH
	}
	triggerSize := image.Pt(fieldW, triggerH)

	// The trigger's own fill is the storey it is raised to over the ground it
	// stands on (control.Fill), the same walk the field, the box and the radio
	// take. Disabled fades that fill rather than naming a second one, so the
	// state follows the storey wherever the trigger was put.
	bg := control.Fill(tok.color, s.Ground)
	if s.Disabled {
		bg = tokens.Disabled(bg)
	}
	// Focus promotes the trigger's border to the focus ring, the one idiom
	// every control in the library wears: the primary rung measured to clear
	// focus.Floor against the storey the trigger stands on — the same ground
	// the resting border below is measured against, so promoting the edge
	// changes its hue and not the ground it answers to. A trigger opened
	// inside a level-3 menu is the case that makes this matter: measured
	// against the trigger's own fill the ring comes out 2.14:1 against the
	// popover it is standing on (focus.Ground).
	// At rest the trigger's border is the neutral rung the ramp measures as
	// clearing the graphic floor against that same storey, the same edge the
	// field and the radio wear (control.Border).
	borderCol := control.Border(tok.color, s.Ground)
	if s.Focused {
		borderCol = focus.Ring(tok.color, focus.Ground(tok.color, s.Ground))
	}
	if s.Disabled {
		borderCol = tokens.Disabled(control.Border(tok.color, s.Ground))
	}
	borderPx := gtx.Dp(1)
	if s.Focused {
		borderPx = gtx.Dp(focus.Width)
	}
	innerRad := rad - borderPx
	if innerRad < 0 {
		innerRad = 0
	}

	rrectOuter := clip.RRect{Rect: image.Rectangle{Max: triggerSize}, SE: rad, SW: rad, NE: rad, NW: rad}
	paint.FillShape(gtx.Ops, borderCol, rrectOuter.Op(gtx.Ops))
	rrectInner := clip.RRect{
		Rect: image.Rectangle{
			Min: image.Pt(borderPx, borderPx),
			Max: image.Pt(triggerSize.X-borderPx, triggerSize.Y-borderPx),
		},
		SE: innerRad, SW: innerRad, NE: innerRad, NW: innerRad,
	}
	paint.FillShape(gtx.Ops, bg, rrectInner.Op(gtx.Ops))

	// Text label: vertically centered.
	offY := (triggerH - labelDims.Size.Y) / 2
	st := op.Offset(image.Pt(padH, offY)).Push(gtx.Ops)
	labelCall.Add(gtx.Ops)
	st.Pop()

	// Chevron: downward triangle aligned to the right.
	cx := fieldW - padH - chevronSz/2
	cy := triggerH / 2
	chevronCol := tok.color.Ramps.Neutral.Step(700) // low-contrast glyph
	if s.Disabled {
		chevronCol = tokens.Disabled(chevronCol)
	}
	drawChevron(gtx, cx, cy, chevronSz, chevronCol)

	return layout.Dimensions{Size: triggerSize}
}

// drawChevron draws a downward-pointing solid triangle centered at (cx, cy)
// with overall width sz.
func drawChevron(gtx layout.Context, cx, cy, sz int, col color.NRGBA) {
	half := float32(sz) / 2
	quarter := float32(sz) / 4
	fcx := float32(cx)
	fcy := float32(cy)

	var p clip.Path
	p.Begin(gtx.Ops)
	p.MoveTo(f32.Pt(fcx-half, fcy-quarter))
	p.LineTo(f32.Pt(fcx+half, fcy-quarter))
	p.LineTo(f32.Pt(fcx, fcy+quarter))
	p.Close()
	paint.FillShape(gtx.Ops, col, clip.Outline{Path: p.End()}.Op())
}

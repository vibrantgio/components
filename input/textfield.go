package input

import (
	"image"
	"image/color"

	"gioui.org/font"
	"gioui.org/io/event"
	"gioui.org/io/key"
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
	"github.com/vibrantgio/mvu"
	"github.com/vibrantgio/theme/theme"
	"github.com/vibrantgio/theme/tokens"
	"github.com/vibrantgio/theme/typeset"
)

// RenderState holds explicit visual interaction state for static rendering.
// All fields default to false (normal/idle state).
// Intended for golden-image testing; production code obtains state from the
// Gio event system via TextField.
type RenderState struct {
	Focused  bool
	Disabled bool

	// Level is the level of the surface the field stands on — the field has
	// no level of its own — and its resting border is derived against that
	// surface, in the same vocabulary the host names its own fill
	// (tokens.SurfaceAt). A dialog at tokens.Level2 passes Level2 and the
	// border takes whichever neutral step clears the floor over that surface.
	// The zero value is tokens.Level0, the window's own surface. A focused
	// field ignores it: its
	// border is promoted to the focus ring, which derives against the fill
	// inside the border instead.
	Level tokens.ElevationLevel
	// Text, when non-empty, is rendered in place of the placeholder using the
	// text colour. It models a field that holds user input for the static
	// render path; it has no effect on the live TextField, whose text is held
	// by the inner widget.Editor.
	Text string
}

// TextFieldProps configures a TextField instance.
type TextFieldProps struct {
	// Placeholder is shown when the field is empty and unfocused.
	Placeholder string

	// Description is the screen-reader label. Falls back to Placeholder when empty.
	Description string

	// Level is the level of the surface the field stands on — the field has
	// no level of its own — copied straight into RenderState.Level on every
	// frame: what the resting border is derived against. A container that
	// raises its surface (a level-2 dialog carrying a form) passes its own
	// level here; the zero value is the window's own surface. See
	// RenderState.Level.
	Level tokens.ElevationLevel

	// Seed, when non-empty, pre-fills the editor when the field instance is
	// created, so an existing value can be edited rather than retyped. The
	// field stays uncontrolled: later Seed values have no effect on a live
	// instance — rebuild the field (e.g. keyed on an epoch, the modal-form
	// pattern) to reseed it.
	Seed string

	// FocusTag, if non-nil, is called once when the field instance is
	// created, with the editor's focus tag — for callers that manage a
	// focus cycle (e.g. patterns/modal's Tab trap) and need to include the
	// field in it. A rebuilt field (new epoch) calls it again with the new
	// instance's tag.
	FocusTag func(tag event.Tag)

	// Mask, when non-zero, hides the entered text by rendering every rune as
	// this one (e.g. '•' for a password or secret field). The unmasked value is
	// still delivered through OnChange/Message and the editor's Text(); only the
	// on-screen display is obscured. The placeholder is never masked.
	Mask rune

	// Disabled, if non-nil, disables the field when it emits true.
	Disabled rx.Observable[bool]

	// OnChange is called with the new value on every text change.
	// This is the FRP callback path. The gtx argument is the layout.Context
	// active on the frame when the change is processed, allowing consumers to
	// emit mvu.MessageOp{Message: ...}.Add(gtx.Ops) inside the callback.
	OnChange func(gtx layout.Context, text string)

	// Message, if non-nil, causes the field to emit mvu.MessageOp{Message}
	// on every text change. This is the MVU integration path.
	Message any

	// Submit, when true, configures the inner widget.Editor to translate
	// carriage-return key presses into widget.SubmitEvent rather than
	// inserting newlines. Required for chat-style inputs.
	Submit bool

	// SubmitMessage, if non-nil, is invoked on each submit and its return
	// value is wrapped in mvu.MessageOp and emitted on the current frame.
	SubmitMessage func(text string) any

	// OnSubmit, if non-nil, is invoked on each submit with the editor's
	// current text. After both callbacks run the editor is cleared.
	// The gtx argument is the layout.Context active on the frame when the
	// submit is processed, allowing consumers to emit
	// mvu.MessageOp{Message: ...}.Add(gtx.Ops) inside the callback.
	OnSubmit func(gtx layout.Context, text string)

	// Shaper is an explicit per-instance override of the text shaper. Leave it
	// nil in normal use: the field then shapes its text with the theme's
	// shaper (Typography.Shaper()), which is built once for the process and
	// shared by every component reading that typography — the cache lives
	// behind the Typography value, so it survives the copy this component's
	// map function makes of it. Set it only when this field
	// must shape with a different shaper than the theme provides.
	//
	// A shaper is not safe to use from two goroutines; Gio lays the layout
	// tree out on the one goroutine that runs the event loop, which is what
	// makes sharing it correct. See theme/tokens.Typography.Shaper.
	Shaper *text.Shaper
}

// resolvedTokens is the concrete per-emission snapshot consumed by the
// layout.Widget closure.
type resolvedTokens struct {
	color   tokens.ColorTokens
	body    tokens.TextStyle // the BodyLarge role: typeface, weight, size, line height
	spacing tokens.SpacingScale
	radius  tokens.RadiusScale
	density tokens.Density // control height and inner padding
	shaper  *text.Shaper   // the theme's shaper; nil in the Render* paths
}

// bodyLabel derives the Gio font, a single-line label and the text size from
// the BodyLarge role carried in tok. Zero fields fall back to the shaper's
// defaults. Lay the returned label out with typeset.Layout rather than
// widget.Label.Layout: the role's line height is the height of the line box,
// which Gio does not give a single line on its own.
func bodyLabel(tok resolvedTokens) (font.Font, widget.Label, unit.Sp) {
	style := tok.body
	return typeset.Font(style, font.Normal), typeset.Label(style, 1), unit.Sp(style.Size)
}

// TextField returns an rx.Observable[layout.Widget] that emits a new widget
// whenever the theme or disabled state changes. Interaction state (editor content,
// focus) lives in the rx.Defer scope and persists across emissions.
//
// Both integration paths are supported:
//   - FRP: set TextFieldProps.OnChange; FRP consumers wrap with rx.NewSubject if needed.
//   - MVU: set TextFieldProps.Message; the component emits mvu.MessageOp on text change.
func TextField(th rx.Observable[theme.Theme], props TextFieldProps) rx.Observable[layout.Widget] {
	disabled := props.Disabled
	if disabled == nil {
		disabled = rx.Of(false)
	}

	// Flatten the nested theme observables into a concrete snapshot. The
	// typography emission supplies both the BodyLarge text style and the
	// theme's cached shaper — the theme owns the typeface.
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
		// Allocated once per subscription — survives all theme and disabled
		// emissions for the lifetime of this TextField instance.
		editor := &widget.Editor{SingleLine: true, Submit: props.Submit, Mask: props.Mask}
		// hitTag identifies the extended pointer-target area: the
		// visual field is the density's control height tall, but a press
		// anywhere in the ≥44 dp hit rectangle focuses the editor.
		hitTag := new(int)
		if props.Seed != "" {
			editor.SetText(props.Seed)
		}
		if props.FocusTag != nil {
			props.FocusTag(editor)
		}

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

				// Drain editor events; fire callbacks on text change and submit.
				for {
					ev, ok := editor.Update(gtx)
					if !ok {
						break
					}
					switch ev := ev.(type) {
					case widget.ChangeEvent:
						val := editor.Text()
						if props.OnChange != nil {
							props.OnChange(gtx, val)
						}
						if props.Message != nil {
							mvu.MessageOp{Message: props.Message}.Add(gtx.Ops)
						}
					case widget.SubmitEvent:
						if props.SubmitMessage != nil {
							mvu.MessageOp{Message: props.SubmitMessage(ev.Text)}.Add(gtx.Ops)
						}
						if props.OnSubmit != nil {
							props.OnSubmit(gtx, ev.Text)
						}
						editor.SetText("")
					}
				}

				// A press in the extended hit area (beyond the visual
				// field but inside the ≥44 dp pointer target) focuses
				// the editor; presses on the text line itself are also
				// seen by the editor's own area, which handles caret
				// placement.
				for {
					ev, ok := gtx.Event(pointer.Filter{Target: hitTag, Kinds: pointer.Press})
					if !ok {
						break
					}
					if _, isPtr := ev.(pointer.Event); isPtr && !dis {
						gtx.Execute(key.FocusCmd{Tag: editor})
					}
				}

				desc := props.Description
				if desc == "" {
					desc = props.Placeholder
				}

				foc := !dis && gtx.Focused(editor)
				showPh := !foc && editor.Len() == 0

				return drawTextFieldLive(gtx, shaper, editor, hitTag, props.Placeholder, desc, tok, RenderState{
					Focused:  foc,
					Disabled: dis,
					Level:    props.Level,
				}, showPh)
			}
		})
	})
}

// Render produces a layout.Widget for a text field in an explicit visual state,
// without any event processing or rx machinery. Intended for golden-image
// testing and static demonstrations; production code should use TextField,
// which reads both of the parameters below off the theme.
//
// body is the BodyLarge role's whole text style — typeface, weight, size and
// line height all reach the shaper — and d is the density the field draws at
// (control height and vertical padding; horizontal padding stays spacing.S3).
// Pass tokens.DefaultTypography.BodyLarge and tokens.Comfortable for the
// default desktop look.
func Render(
	shaper *text.Shaper,
	placeholder string,
	colors tokens.ColorTokens,
	sp tokens.SpacingScale,
	rad tokens.RadiusScale,
	body tokens.TextStyle,
	d tokens.Density,
	s RenderState,
) layout.Widget {
	tok := resolvedTokens{color: colors, spacing: sp, radius: rad, body: body, density: d}
	return func(gtx layout.Context) layout.Dimensions {
		return drawTextFieldStatic(gtx, shaper, placeholder, tok, s)
	}
}

// drawTextFieldLive renders a live text field containing a widget.Editor.
func drawTextFieldLive(gtx layout.Context, shaper *text.Shaper, editor *widget.Editor, hitTag *int, placeholder, desc string, tok resolvedTokens, s RenderState, showPlaceholder bool) layout.Dimensions {
	// Sizing rule: field height = Density.ControlHeight (36 dp
	// Comfortable, 28 dp Compact — shadcn's h-9 input), vertical padding =
	// Density.PaddingY. Horizontal padding stays spacing.S3 (12 dp): shadcn
	// keeps px-3 on inputs across sizes, so it does not follow density.
	padH := gtx.Dp(unit.Dp(tok.spacing.S3))
	padV := gtx.Dp(unit.Dp(tok.density.PaddingY))
	minH := gtx.Dp(unit.Dp(tok.density.ControlHeight))
	rad := gtx.Dp(unit.Dp(tok.radius.Md))
	// Shape with the BodyLarge role's typeface, weight, size and line height.
	f, wl, textSize := bodyLabel(tok)

	bg, textColor, borderColor, phColor := textFieldColors(tok.color, s)

	fieldW := gtx.Constraints.Max.X
	innerW := fieldW - 2*padH
	if innerW < 1 {
		innerW = 1
	}

	innerGtx := gtx
	innerGtx.Constraints = layout.Constraints{
		Min: image.Pt(0, 0),
		Max: image.Pt(innerW, gtx.Constraints.Max.Y),
	}

	// Semantic accessibility ops.
	semantic.ClassOp(semantic.Editor).Add(gtx.Ops)
	semantic.DescriptionOp(desc).Add(gtx.Ops)
	semantic.EnabledOp(!s.Disabled).Add(gtx.Ops)

	// Measure content height via recorded label layout so we can size the
	// field before drawing the background. Replay if placeholder is needed.
	mPhColor := op.Record(gtx.Ops)
	paint.ColorOp{Color: phColor}.Add(gtx.Ops)
	phMat := mPhColor.Stop()

	mPh := op.Record(gtx.Ops)
	contentDims := typeset.Layout(innerGtx, shaper, wl, f, textSize, placeholder, phMat)
	phCall := mPh.Stop()

	fieldH := contentDims.Size.Y + 2*padV
	if fieldH < minH {
		fieldH = minH
	}
	fieldSize := image.Pt(fieldW, fieldH)

	// Border as nested fills: outer rect in border color, inner rect in
	// background color. Avoids clip.Stroke anti-aliasing variance in tests.
	borderPx := gtx.Dp(1)
	if s.Focused {
		borderPx = gtx.Dp(focus.Width)
	}
	innerRad := rad - borderPx
	if innerRad < 0 {
		innerRad = 0
	}
	rrectOuter := clip.RRect{Rect: image.Rectangle{Max: fieldSize}, SE: rad, SW: rad, NE: rad, NW: rad}
	paint.FillShape(gtx.Ops, borderColor, rrectOuter.Op(gtx.Ops))
	rrectInner := clip.RRect{
		Rect: image.Rectangle{
			Min: image.Pt(borderPx, borderPx),
			Max: image.Pt(fieldSize.X-borderPx, fieldSize.Y-borderPx),
		},
		SE: innerRad, SW: innerRad, NE: innerRad, NW: innerRad,
	}
	paint.FillShape(gtx.Ops, bg, rrectInner.Op(gtx.Ops))

	offY := (fieldH - contentDims.Size.Y) / 2

	// Placeholder overlay (only when empty and unfocused).
	if showPlaceholder {
		st := op.Offset(image.Pt(padH, offY)).Push(gtx.Ops)
		phCall.Add(gtx.Ops)
		st.Pop()
	}

	// Editor materials.
	mText := op.Record(gtx.Ops)
	paint.ColorOp{Color: textColor}.Add(gtx.Ops)
	textMat := mText.Stop()

	mSel := op.Record(gtx.Ops)
	paint.ColorOp{Color: withAlpha(tok.color.Primary, 0x40)}.Add(gtx.Ops)
	selMat := mSel.Stop()

	// Editor — always laid out so it receives pointer/keyboard events.
	// Min.X = innerW ensures the event clip covers the full field width even
	// when the editor is empty (otherwise the clip shrinks to the caret width).
	editorGtx := innerGtx
	editorGtx.Constraints = layout.Constraints{
		Min: image.Pt(innerW, 0),
		Max: image.Pt(innerW, contentDims.Size.Y),
	}
	// The editor shapes with the same BodyLarge role as the placeholder label
	// so measured content height and caret metrics line up.
	if tok.body.LineHeight != 0 {
		editor.LineHeight = unit.Sp(tok.body.LineHeight)
		editor.LineHeightScale = 1
	}
	// The placeholder above went through typeset.Layout, which centres its
	// glyphs within the role's line box: half the line-height deficit sits above
	// the glyphs. The editor is a raw widget.Editor, and Gio baselines an
	// editor's first line at its own ascent — its share of the line height
	// all lands below the glyphs. Drawn at the same offY, the visible text would
	// rise by half the deficit the moment the editor takes over (focus, or a
	// first keystroke); offsetting the editor down by that same half-deficit
	// keeps the text, the caret and the selection where the placeholder's text
	// was, and inside the line box the field was sized from.
	textShift := editorTextShift(innerGtx, shaper, wl, f, textSize, placeholder, phMat, contentDims.Size.Y)
	st := op.Offset(image.Pt(padH, offY+textShift)).Push(gtx.Ops)
	editor.Layout(editorGtx, shaper, f, textSize, textMat, selMat)
	st.Pop()

	if !s.Disabled {
		pointer.CursorText.Add(gtx.Ops)
	}

	// Pointer-target extension: the drawn field may be shorter than
	// the WCAG 2.5.5 floor, but the pointer target never is. Register a
	// pass-through input area over the hit rectangle — max(field, 44 dp) per
	// axis, centred on the field, extending beyond its bounds — whose press
	// events the TextField component turns into a FocusCmd for the editor. The
	// pass op keeps the editor's own area receiving the presses that land on
	// the text line.
	hitPx := gtx.Dp(unit.Dp(tok.density.MinHitTarget()))
	hitW, hitH := fieldW, fieldH
	if hitW < hitPx {
		hitW = hitPx
	}
	if hitH < hitPx {
		hitH = hitPx
	}
	hitRect := image.Rect(-(hitW-fieldW)/2, -(hitH-fieldH)/2, fieldW+(hitW-fieldW+1)/2, fieldH+(hitH-fieldH+1)/2)
	cl := clip.Rect(hitRect).Push(gtx.Ops)
	pass := pointer.PassOp{}.Push(gtx.Ops)
	event.Op(gtx.Ops, hitTag)
	pass.Pop()
	cl.Pop()

	return layout.Dimensions{Size: fieldSize}
}

// editorTextShift returns how far down the live editor must draw so its first
// line's glyphs land where typeset.Layout put the placeholder's: half the
// line-height deficit the placeholder's line box carries above its glyphs.
//
// corrected is the box typeset.Layout reported for txt; the natural glyph
// height is re-measured here from the same single-line layout — a hit in the
// shaper's cache, and measured against the text rather than the face, for the
// reason the typeset package documents: a line holding a fallback run is
// taller than its primary face and needs less added. The measurement drops
// the caller's vertical constraints exactly as typeset does, so the two
// halves of the deficit agree; the recorded ops are discarded.
//
// The shift is zero whenever typeset's correction is: no positive absolute
// line height, a LineHeightScale other than 1, or text already as tall as its
// box. It is never negative, so the editor never draws above the field's
// content offset.
func editorTextShift(gtx layout.Context, sh *text.Shaper, lbl widget.Label, f font.Font, size unit.Sp, txt string, material op.CallOp, corrected int) int {
	if gtx.Sp(lbl.LineHeight) <= 0 || lbl.LineHeightScale != 1 {
		return 0
	}
	m := gtx
	m.Constraints.Min.Y = 0
	m.Constraints.Max.Y = 1 << 20
	rec := op.Record(m.Ops)
	dims := lbl.Layout(m, sh, f, size, txt, material)
	rec.Stop()
	if d := corrected - dims.Size.Y; d > 0 {
		return d / 2
	}
	return 0
}

// drawTextFieldStatic renders a static text field for golden-image testing.
// It always shows the placeholder text; there is no live editor.
func drawTextFieldStatic(gtx layout.Context, shaper *text.Shaper, placeholder string, tok resolvedTokens, s RenderState) layout.Dimensions {
	// Same sizing rules as drawTextFieldLive.
	padH := gtx.Dp(unit.Dp(tok.spacing.S3))
	padV := gtx.Dp(unit.Dp(tok.density.PaddingY))
	minH := gtx.Dp(unit.Dp(tok.density.ControlHeight))
	rad := gtx.Dp(unit.Dp(tok.radius.Md))
	// Shape with the BodyLarge role's typeface, weight, size and line height.
	f, wl, textSize := bodyLabel(tok)

	bg, textColor, borderColor, phColor := textFieldColors(tok.color, s)

	fieldW := gtx.Constraints.Max.X
	innerW := fieldW - 2*padH
	if innerW < 1 {
		innerW = 1
	}

	innerGtx := gtx
	innerGtx.Constraints = layout.Constraints{
		Min: image.Pt(0, 0),
		Max: image.Pt(innerW, gtx.Constraints.Max.Y),
	}

	// Record the inner label for measurement and deferred rendering. When
	// RenderState.Text is non-empty it stands in for editor content (text
	// colour); otherwise the placeholder is drawn.
	labelText := placeholder
	labelColor := phColor
	if s.Text != "" {
		labelText = s.Text
		labelColor = textColor
	}

	mLabelColor := op.Record(gtx.Ops)
	paint.ColorOp{Color: labelColor}.Add(gtx.Ops)
	labelMat := mLabelColor.Stop()

	mLabel := op.Record(gtx.Ops)
	labelDims := typeset.Layout(innerGtx, shaper, wl, f, textSize, labelText, labelMat)
	labelCall := mLabel.Stop()

	fieldH := labelDims.Size.Y + 2*padV
	if fieldH < minH {
		fieldH = minH
	}
	fieldSize := image.Pt(fieldW, fieldH)

	// Border as nested fills: outer rect in border color, inner rect in
	// background color. Avoids clip.Stroke anti-aliasing variance in tests.
	borderPx := gtx.Dp(1)
	if s.Focused {
		borderPx = gtx.Dp(focus.Width)
	}
	innerRad := rad - borderPx
	if innerRad < 0 {
		innerRad = 0
	}
	rrectOuter := clip.RRect{Rect: image.Rectangle{Max: fieldSize}, SE: rad, SW: rad, NE: rad, NW: rad}
	paint.FillShape(gtx.Ops, borderColor, rrectOuter.Op(gtx.Ops))
	rrectInner := clip.RRect{
		Rect: image.Rectangle{
			Min: image.Pt(borderPx, borderPx),
			Max: image.Pt(fieldSize.X-borderPx, fieldSize.Y-borderPx),
		},
		SE: innerRad, SW: innerRad, NE: innerRad, NW: innerRad,
	}
	paint.FillShape(gtx.Ops, bg, rrectInner.Op(gtx.Ops))

	// Placeholder label centered vertically.
	offY := (fieldH - labelDims.Size.Y) / 2
	st := op.Offset(image.Pt(padH, offY)).Push(gtx.Ops)
	labelCall.Add(gtx.Ops)
	st.Pop()

	return layout.Dimensions{Size: fieldSize}
}

// textFieldColors returns (bg, text, border, placeholder) colors for the
// given state: the field's own raised fill (controlFill), body text, the
// resting border the neutral ramp measures against the surface the field
// stands on (controlBorder) and the control family's own prompt foreground
// (control.Placeholder) — the same step components/picker's field trigger
// draws its prompt in, named once so the two cannot drift. Disabled fades
// each to DisabledOpacity; focus promotes the border to the focus ring.
//
// The fill is walked from the surface the field was handed (controlFill)
// rather than a named surface colour, so the field stays lighter than
// whatever it lies on in both colour schemes and stays lighter as the host
// rises.
//
// The border is derived from the neutral ramp against the surface the field
// stands on (controlBorder) rather than a named ramp step, so the field wears
// the same edge the checkbox and the radio do, on whatever level it is put.
//
// The focused border's colour is focus.Ring — the scheme's one focus colour,
// the same on every level and on every control, so promoting the edge changes
// its hue and not what it has to answer to. That surface lies immediately
// outside
// the promoted band and that is the side the ring's floor is measured to.
func textFieldColors(c tokens.ColorTokens, s RenderState) (bg, text, border, placeholder color.NRGBA) {
	bg = controlFill(c, s.Level)
	text = c.Text
	border = controlBorder(c, s.Level)
	placeholder = control.Placeholder(c)
	switch {
	case s.Disabled:
		bg = tokens.Disabled(bg)
		text = tokens.Disabled(text)
		border = tokens.Disabled(border)
		placeholder = tokens.Disabled(placeholder)
	case s.Focused:
		border = focus.Ring(c)
	}
	return
}

// withAlpha returns c with its alpha scaled by factor a (0–255).
func withAlpha(c color.NRGBA, a uint8) color.NRGBA {
	c.A = uint8(uint16(c.A) * uint16(a) / 255)
	return c
}

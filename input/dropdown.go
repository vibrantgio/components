package input

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
	"github.com/vibrantgio/mvu"
	"github.com/vibrantgio/prism/internal/hit"
	"github.com/vibrantgio/spectrum/theme"
	"github.com/vibrantgio/spectrum/tokens"
)

// DropdownRenderState holds explicit visual state for static rendering.
// All fields default to their zero values (normal/closed/idle).
// Intended for golden-image testing; production code obtains state from the
// Gio event system via Dropdown.
type DropdownRenderState struct {
	Open     bool
	Focused  bool
	Disabled bool
	Selected int
	Options  []string
}

// DropdownProps configures a Dropdown instance.
type DropdownProps struct {
	// Description is the screen-reader label.
	Description string

	// Options is the list of selectable items.
	Options []string

	// Selected is the initial selected index established on subscribe.
	Selected int

	// Disabled, if non-nil, disables the dropdown when it emits true.
	Disabled rx.Observable[bool]

	// OnSelect is called with the newly selected index on every selection.
	// This is the FRP callback path. The gtx argument is the layout.Context
	// active on the frame when the selection is processed, allowing consumers to
	// emit mvu.MessageOp{Message: ...}.Add(gtx.Ops) inside the callback.
	OnSelect func(gtx layout.Context, index int)

	// Message, if non-nil, causes the dropdown to emit mvu.MessageOp{Message}
	// on every selection. This is the MVU integration path.
	Message any

	// Shaper is an explicit per-instance override of the text shaper. Leave it
	// nil in normal use: the dropdown then shapes its text with the theme's
	// shaper (Typography.Shaper()), which is built once and cached inside the
	// theme's Typography value. Set it only when this dropdown must shape with
	// a different shaper than the theme provides.
	Shaper *text.Shaper
}

// Dropdown returns an rx.Observable[layout.Widget] that emits a new widget
// whenever the theme or disabled state changes. Widget state (open/closed,
// selected index, focus) lives in the rx.Defer scope and persists across emissions.
//
// Both integration paths are supported:
//   - FRP: set DropdownProps.OnSelect.
//   - MVU: set DropdownProps.Message; the component emits mvu.MessageOp on selection.
func Dropdown(th rx.Observable[theme.Theme], props DropdownProps) rx.Observable[layout.Widget] {
	disabled := props.Disabled
	if disabled == nil {
		disabled = rx.Of(false)
	}

	// Flatten the nested theme observables into a concrete snapshot. The
	// typography emission supplies both the BodyLarge text style and the
	// theme's cached shaper (ADR-003: the theme owns the typeface).
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

				return layoutDropdownLive(gtx, shaper, &trigger, optClicks, tok, props.Description, DropdownRenderState{
					Open:     open,
					Focused:  foc,
					Disabled: dis,
					Selected: selected,
					Options:  props.Options,
				})
			}
		})
	})
}

// RenderDropdown produces a layout.Widget for a dropdown in an explicit visual
// state, without any event processing or rx machinery. Intended for golden-image
// testing and static demonstrations; production code should use Dropdown, which
// takes the shaper and the BodyLarge text style from the theme's Typography.
// The TypeScale parameter contributes only the BodyLarge size; typeface,
// weight and line height stay at the shaper's defaults.
func RenderDropdown(
	shaper *text.Shaper,
	colors tokens.ColorTokens,
	sp tokens.SpacingScale,
	rad tokens.RadiusScale,
	ts tokens.TypeScale,
	s DropdownRenderState,
) layout.Widget {
	// Density is not a parameter (the signature predates E1.3): the static
	// path renders at tokens.Comfortable; density-aware rendering goes
	// through Dropdown.
	tok := resolvedTokens{color: colors, spacing: sp, radius: rad, body: tokens.TextStyle{Size: ts.BodyLarge}, density: tokens.Comfortable}
	return func(gtx layout.Context) layout.Dimensions {
		return drawDropdown(gtx, shaper, tok, s)
	}
}

// layoutDropdownLive lays out the interactive dropdown with Clickable hit areas.
func layoutDropdownLive(gtx layout.Context, shaper *text.Shaper, trigger *widget.Clickable, optClicks []widget.Clickable, tok resolvedTokens, desc string, s DropdownRenderState) layout.Dimensions {
	// The trigger's pointer area is at least MinHitTarget (44 dp) on each
	// axis, centred on the visual bar: density shrinks the drawn trigger,
	// never the hit target. Option rows are NOT extended: they stack
	// directly against each other, so a ≥44 dp slop per row would overlap
	// the neighbouring rows' targets; the rows rely on their full-row width
	// instead.
	triggerDims := hit.Extend(gtx, gtx.Dp(unit.Dp(tok.density.MinHitTarget())), trigger.Layout, func(gtx layout.Context) layout.Dimensions {
		semantic.Button.Add(gtx.Ops)
		if desc != "" {
			semantic.DescriptionOp(desc).Add(gtx.Ops)
		}
		return drawTrigger(gtx, shaper, tok, s)
	})

	if !s.Open || len(s.Options) == 0 {
		return triggerDims
	}

	fieldW := gtx.Constraints.Max.X
	totalH := triggerDims.Size.Y
	for i := range optClicks {
		off := op.Offset(image.Pt(0, totalH)).Push(gtx.Ops)
		optGtx := gtx
		optGtx.Constraints = layout.Constraints{
			Min: image.Pt(0, 0),
			Max: image.Pt(fieldW, gtx.Constraints.Max.Y),
		}
		idx := i
		label := s.Options[idx]
		optDims := optClicks[idx].Layout(optGtx, func(gtx layout.Context) layout.Dimensions {
			semantic.Button.Add(gtx.Ops)
			return drawOptionRow(gtx, shaper, tok, idx == s.Selected, label)
		})
		off.Pop()
		totalH += optDims.Size.Y
	}

	return layout.Dimensions{Size: image.Pt(fieldW, totalH)}
}

// drawDropdown renders the static dropdown for golden-image testing.
func drawDropdown(gtx layout.Context, shaper *text.Shaper, tok resolvedTokens, s DropdownRenderState) layout.Dimensions {
	triggerDims := drawTrigger(gtx, shaper, tok, s)

	if !s.Open || len(s.Options) == 0 {
		return triggerDims
	}

	fieldW := gtx.Constraints.Max.X
	totalH := triggerDims.Size.Y
	for i, opt := range s.Options {
		off := op.Offset(image.Pt(0, totalH)).Push(gtx.Ops)
		optGtx := gtx
		optGtx.Constraints = layout.Constraints{
			Min: image.Pt(0, 0),
			Max: image.Pt(fieldW, gtx.Constraints.Max.Y),
		}
		optDims := drawOptionRow(optGtx, shaper, tok, i == s.Selected, opt)
		off.Pop()
		totalH += optDims.Size.Y
	}

	return layout.Dimensions{Size: image.Pt(fieldW, totalH)}
}

// drawTrigger renders the dropdown trigger bar (the closed face).
func drawTrigger(gtx layout.Context, shaper *text.Shaper, tok resolvedTokens, s DropdownRenderState) layout.Dimensions {
	// E1.3 sizing rule: the trigger follows the text field — height =
	// Density.ControlHeight, vertical padding = Density.PaddingY, horizontal
	// padding a static spacing.S3 (12 dp, shadcn's px-3).
	padH := gtx.Dp(unit.Dp(tok.spacing.S3))
	padV := gtx.Dp(unit.Dp(tok.density.PaddingY))
	rad := gtx.Dp(unit.Dp(tok.radius.Md))
	// Shape with the BodyLarge role's typeface, weight, size and line height.
	f, wl, textSize := bodyLabel(tok)
	minH := gtx.Dp(unit.Dp(tok.density.ControlHeight))
	fieldW := gtx.Constraints.Max.X
	chevronSz := gtx.Dp(unit.Dp(16))

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
	labelDims := wl.Layout(innerGtx, shaper, f, textSize, selectedText, textMat)
	labelCall := mLabel.Stop()

	triggerH := labelDims.Size.Y + 2*padV
	if triggerH < minH {
		triggerH = minH
	}
	triggerSize := image.Pt(fieldW, triggerH)

	bg := tok.color.Surface
	if s.Disabled {
		bg = tokens.Disabled(bg)
	}
	borderCol := tok.color.Ramps.Neutral.Step(500) // strong border
	if s.Focused {
		borderCol = tok.color.Primary
	}
	if s.Disabled {
		borderCol = tokens.Disabled(tok.color.Ramps.Neutral.Step(500))
	}
	borderPx := gtx.Dp(1)
	if s.Focused {
		borderPx = gtx.Dp(2)
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

// drawOptionRow renders a single option row in the open dropdown list.
func drawOptionRow(gtx layout.Context, shaper *text.Shaper, tok resolvedTokens, selected bool, label string) layout.Dimensions {
	// E1.3 sizing rule: option rows are list rows — row height =
	// Density.ControlHeight exactly (list.RowHeight's rule: 36 dp
	// Comfortable, 28 dp Compact).
	padH := gtx.Dp(unit.Dp(tok.spacing.S3))
	padV := gtx.Dp(unit.Dp(tok.density.PaddingY))
	// Shape with the BodyLarge role's typeface, weight, size and line height.
	f, wl, textSize := bodyLabel(tok)
	minH := gtx.Dp(unit.Dp(tok.density.ControlHeight))
	fieldW := gtx.Constraints.Max.X

	textCol := tok.color.Text
	innerW := fieldW - 2*padH
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
	labelDims := wl.Layout(innerGtx, shaper, f, textSize, label, textMat)
	labelCall := mLabel.Stop()

	rowH := labelDims.Size.Y + 2*padV
	if rowH < minH {
		rowH = minH
	}
	rowSize := image.Pt(fieldW, rowH)

	bg := tok.color.Surface
	if selected {
		// Selected is a D2.3 state walk: two steps past the Surface ground
		// (Neutral 200), landing on Neutral 400.
		bg = tok.color.StateColor(tokens.RoleNeutral, 200, tokens.StateSelected)
	}
	paint.FillShape(gtx.Ops, bg, clip.Rect{Max: rowSize}.Op())

	offY := (rowH - labelDims.Size.Y) / 2
	st := op.Offset(image.Pt(padH, offY)).Push(gtx.Ops)
	labelCall.Add(gtx.Ops)
	st.Pop()

	return layout.Dimensions{Size: rowSize}
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

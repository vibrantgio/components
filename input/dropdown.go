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
	"github.com/vibrantgio/components/internal/hit"
	"github.com/vibrantgio/mvu"
	"github.com/vibrantgio/theme/theme"
	"github.com/vibrantgio/theme/tokens"
	"github.com/vibrantgio/theme/typeset"
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
	// shaper (Typography.Shaper()), which is built once for the process and
	// shared by every component reading that typography — the cache lives
	// behind the Typography value, so it survives the copy this component's
	// map function makes of it (spectrum F5.1). Set it only when this dropdown
	// must shape with a different shaper than the theme provides.
	//
	// A shaper is not safe to use from two goroutines; Gio lays the widget
	// forest out on the one goroutine that runs the event loop, which is what
	// makes sharing it correct. See theme/tokens.Typography.Shaper.
	Shaper *text.Shaper
}

// Dropdown returns an rx.Observable[layout.Widget] that emits a new widget
// whenever the theme or disabled state changes. Widget state (open/closed,
// selected index, focus) lives in the rx.Defer scope and persists across emissions.
//
// Both integration paths are supported:
//   - FRP: set DropdownProps.OnSelect.
//   - MVU: set DropdownProps.Message; the component emits mvu.MessageOp on selection.
//
// # Keyboard reach
//
// F4.7 checked the open menu against the gap it fixed in patterns/sidebar —
// keyboard traversal built on per-row focus tags in a virtualised region,
// which reaches only the rows a frame laid out — and the menu does not have
// it. It is not virtualised: layoutDropdownLive walks every option, so while
// the menu is open every option row exists in the op tree with its own
// widget.Clickable focus tag, and Tab plus Enter/Space reaches all of them.
// There is no unreachable option because there is no offscreen option.
//
// Two things follow, and they are worth writing down rather than
// rediscovering. First, that guarantee is bounded by the option count: the
// menu draws its full height and would run off the window before it ran out of
// focus tags, so an options list long enough to need virtualising must move to
// components/list's LayoutSelectable, not grow per-row tags. Second, Tab-per-option
// is not the menu behaviour a listbox implies — arrow keys should move a
// highlight within the open menu and Escape should close it. That is a real
// gap, but it is a menu-semantics gap, not the virtualisation one, and it is
// not what F4.7 was scoped to fix.
func Dropdown(th rx.Observable[theme.Theme], props DropdownProps) rx.Observable[layout.Widget] {
	disabled := props.Disabled
	if disabled == nil {
		disabled = rx.Of(false)
	}

	// Flatten the nested theme observables into a concrete snapshot. The
	// typography emission supplies both the BodyLarge text style and the
	// theme's cached shaper (ADR-003: the theme owns the typeface). The
	// elevation scale joins in a second combine (CombineLatest tops out at
	// five inputs): the open menu resolves its surface through the ladder.
	resolved := rx.SwitchMap(th, func(t theme.Theme) rx.Observable[resolvedTokens] {
		base := rx.Map(
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
		return rx.Map(
			rx.CombineLatest2(base, t.Elevation),
			func(n rx.Tuple2[resolvedTokens, tokens.ElevationScale]) resolvedTokens {
				tok := n.First
				tok.elevation = n.Second
				return tok
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
// reads both of the parameters below off the theme.
//
// body is the BodyLarge role's whole text style — typeface, weight, size and
// line height all reach the shaper — and d is the density the trigger and the
// option rows draw at. Pass tokens.DefaultTypography.BodyLarge and
// tokens.Comfortable for the default desktop look.
func RenderDropdown(
	shaper *text.Shaper,
	colors tokens.ColorTokens,
	sp tokens.SpacingScale,
	rad tokens.RadiusScale,
	body tokens.TextStyle,
	d tokens.Density,
	s DropdownRenderState,
) layout.Widget {
	// Elevation is not a parameter (the signature predates E2.3): the static
	// path renders on the default tokens.Elevation scale; elevation-aware
	// rendering goes through Dropdown.
	tok := resolvedTokens{color: colors, spacing: sp, radius: rad, body: body, density: d, elevation: tokens.Elevation}
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
	// instead. What they measure, at 1:1, is 40 dp Comfortable and 36 dp
	// Compact — BodyLarge's 24 dp line box plus 2×PaddingY, which wins over
	// the ControlHeight floor in both densities — so both clear WCAG 2.5.8
	// Target Size (Minimum), the 24 dp AA criterion these rows are held to.
	// See tokens.MinHitTarget for why 2.5.5's 44 dp is not that criterion.
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
	labelDims := typeset.Layout(innerGtx, shaper, wl, f, textSize, selectedText, textMat)
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
	labelDims := typeset.Layout(innerGtx, shaper, wl, f, textSize, label, textMat)
	labelCall := mLabel.Stop()

	rowH := labelDims.Size.Y + 2*padV
	if rowH < minH {
		rowH = minH
	}
	rowSize := image.Pt(fieldW, rowH)

	// The open menu is a floating transient overlay — an unscrimmed,
	// shadowless plane like patterns/popover — so its rows fill at Level3
	// on the elevation ladder (Neutral 400), not the flat Surface the
	// closed trigger keeps (E2.3; see the package doc).
	bg := tok.color.SurfaceAt(tokens.Level3)
	if selected {
		// Selected is a D2.3 state walk composed on the menu's elevation
		// ground (E2.1): two steps past the level-3 step (Neutral 400),
		// landing on Neutral 600.
		bg = tok.color.StateColor(tokens.RoleNeutral, tok.elevation.SurfaceStep(tokens.Level3), tokens.StateSelected)
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

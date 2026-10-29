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
	"github.com/vibrantgio/components/internal/focus"
	"github.com/vibrantgio/components/internal/surface"
	"github.com/vibrantgio/mvu"
	vgcolor "github.com/vibrantgio/theme/color"
	"github.com/vibrantgio/theme/theme"
	"github.com/vibrantgio/theme/tokens"
	"github.com/vibrantgio/theme/typeset"
)

// checkboxBoxSize is the visual side length of the checkbox square: 16 dp,
// MEASURED off save-dialog-light.png and save-dialog-dark.png, the "Options:"
// checkbox at y 372–387, x 264–279 in both appearances, and recorded in the
// organization's macOS reference.
const checkboxBoxSize = unit.Dp(16)

// checkboxCornerRadius is the corner the box is drawn with: 5 dp, MEASURED
// off save-dialog-light.png and save-dialog-dark.png.
//
// Both switched-off "Options:" boxes carry a one-pixel antialiased ramp at
// each of their four corners. A circular fit to that ramp's per-row coverage,
// the box's own extremes pinned — the fit CG4.8 made to the sidebar recess's
// ends — answers r = 5.04, rms 0.038 px over 8 rows, in the light appearance
// and r = 5.34, rms 0.070 px over 8 rows, in the dark; the four corners of
// each box and the two boxes of each sheet agree to the hundredth. The
// coverage missing from each corner says the same: 5.54 px² light and 6.21
// px² dark against r²(1 − π/4), which is r = 5.08 and r = 5.38.
//
// The platform's corner is a continuous curve, which is what puts a circular
// fit above the radius the corner is drawn at — the sidebar recess's 14 fits
// at 14.7 and the sidebar pill's 8 at 7.9 — and the dark reading sits above
// the light one because the dark sheet and fill are ten of 255 apart against
// the light pair's thirteen, so its coverage is read on a coarser step. 5 is
// what a circular corner draws.
const checkboxCornerRadius = unit.Dp(5)

// checkboxRingOver is how much of the focus ring lies over the checkbox's
// box: 1 dp, with the other 3 dp of the band past the box.
//
// MEASURED, checkbox-focused-{light,dark}.png: the box runs x 264-279 and
// the ring covers x 261-264 and x 279-282 across y=380, and y 369-372 and
// y 387-390 down x=271, in both schemes. x=264 is the box's first column, so
// one column of the band is over the box's fill and three are past it.
const checkboxRingOver = unit.Dp(1)

// controlTitleGap is the leading gap a checkbox's square — or a radio's
// circle — spends on the title beside it.
//
// MEASURED, save-dialog-{light,dark}.png: both "Options:" squares end at
// column 279, so their edge is 280, and both titles' first covered column is
// 286 — six columns clear, in both appearances and both rows.
//
// It is spent whole, where [control.TextLeadDp] spends its reading less the
// capture's own first letter's bearing. The difference is what the capture
// shows: the field's selection fill stands behind its value, so the platform's
// own origin is there to be read and the bearing can be told from it. Here
// only the covered columns are visible, so the six is spent to the first of
// them and the capture's own titles — which begin with an S the body role's
// face bears no column on — are covered column for column. A title whose
// first glyph does carry a bearing stands one column further out, as it
// would on the platform.
const controlTitleGap = unit.Dp(6)

// The check mark is the platform's white mark, MEASURED off
// keyboard-shortcuts-sheet-{light,dark}.png's checked "Mission Control" box:
// 10 × 10 px, 3 px in from each side of the 16 px box, a stroke 2.06 px wide
// with its long arm at 56.6° from the horizontal.
//
// checkLine is the mark's centre line in the box's own units, 16 to a side:
// in from the left, down to the turn, then up and out to the right. It is the
// least-squares fit of a round-capped, round-joined stroke 2.06 wide to the
// mean of both captures' coverage over the box. The squared coverage error
// summed over the mark's 100 pixels is 0.04. The long arm runs at 56.6°, the short arm at
// 48.5°. Gio's clip.Stroke caps and joins round, so the stroke is the centre
// line plus checkStroke.
const (
	checkGrid   = 16.0
	checkStroke = 2.06
)

var checkLine = [3]f32.Point{
	{X: 4.07, Y: 8.80},
	{X: 6.93, Y: 12.03},
	{X: 11.73, Y: 4.75},
}

// CheckboxRenderState holds explicit visual state for static rendering.
// The zero value is an unchecked, idle, enabled box, so CheckboxRenderState{}
// is exactly today's default checkbox.
// Intended for golden-image testing; production code obtains state from the
// Gio event system via Checkbox.
type CheckboxRenderState struct {
	Checked  bool
	Focused  bool
	Disabled bool

	// Title is the control's own text, drawn beside the box. Empty draws
	// the box alone.
	Title string

	// Surface is the opaque fill the control stands on. Its focus ring rides
	// in the slack around the symbol, so the platform's keyboard focus
	// indicator — a coverage rather than a colour — lands on this. The
	// unchecked box's fill is a step apart from this colour. The zero value —
	// no colour — is the window's own plane.
	Surface color.NRGBA

	// AccentSelection is true where Surface is a selected row in the
	// accent fill with white text. On it the unchecked box is that white at
	// accentSelectionCoverage over that fill, and the title is that white.
	AccentSelection bool
}

// CheckboxProps configures a Checkbox instance.
type CheckboxProps struct {
	// Title is the control's own text, drawn beside the box and part of the
	// control: the whole row operates the box. Empty draws the box alone.
	Title string

	// Description is the screen-reader label. Empty falls back to Title.
	Description string

	// Checked is the initial checked state established on subscribe.
	Checked bool

	// Disabled, if non-nil, disables the checkbox when it emits true.
	Disabled rx.Observable[bool]

	// OnChange is called with the new checked value on every toggle.
	// This is the FRP callback path. The gtx argument is the layout.Context
	// active on the frame when the toggle is processed, allowing consumers to
	// emit mvu.MessageOp{Message: ...}.Add(gtx.Ops) inside the callback.
	OnChange func(gtx layout.Context, checked bool)

	// Message, if non-nil, causes the checkbox to emit mvu.MessageOp{Message}
	// on every toggle. This is the MVU integration path.
	Message any

	// Surface answers the opaque fill the checkbox stands on. The checkbox
	// calls it on every frame, so a row that becomes selected moves the
	// box's fill with it. Leave it nil where the checkbox stands on the
	// window's own plane.
	Surface func(tokens.PlatformColors) color.NRGBA

	// AccentSelection answers, on every frame, whether Surface is a
	// selected row in the accent fill with white text. Nil answers false.
	AccentSelection func() bool
}

// Checkbox returns an rx.Observable[layout.Widget] that emits a new widget
// whenever the theme or disabled state changes. Interaction state (checked value,
// focus) lives in the rx.Defer scope and persists across emissions.
//
// Both integration paths are supported:
//   - FRP: set CheckboxProps.OnChange.
//   - MVU: set CheckboxProps.Message; the component emits mvu.MessageOp on toggle.
func Checkbox(th rx.Observable[theme.Theme], props CheckboxProps) rx.Observable[layout.Widget] {
	disabled := props.Disabled
	if disabled == nil {
		disabled = rx.Of(false)
	}

	// Flatten the nested theme observables into a concrete snapshot. The
	// title is drawn in the body role, so the typography emission supplies
	// the text style, its cap band and the theme's cached shaper; a checkbox
	// without a title never reaches them.
	resolved := rx.SwitchMap(th, func(t theme.Theme) rx.Observable[resolvedTokens] {
		return rx.Map(
			rx.CombineLatest4(t.Platform, t.Typography, t.Spacing, t.Density),
			func(n rx.Tuple4[tokens.PlatformColors, tokens.Typography, tokens.SpacingScale, tokens.Density]) resolvedTokens {
				typ := n.Second
				return resolvedTokens{
					platform: n.First,
					body:     typ.SystemFont,
					capBand:  typ.FaceMetrics(typ.SystemFont).CapHeight,
					spacing:  n.Third,
					density:  n.Fourth,
					shaper:   typ.Shaper(),
				}
			},
		)
	})

	inputs := rx.CombineLatest2(resolved, disabled)

	return rx.Defer(func() rx.Observable[layout.Widget] {
		// Allocated once per subscription — survives all theme and disabled
		// emissions for the lifetime of this checkbox instance.
		var b widget.Bool
		b.Value = props.Checked

		return rx.Map(inputs, func(next rx.Tuple2[resolvedTokens, bool]) layout.Widget {
			tok, dis := next.First, next.Second

			return func(gtx layout.Context) layout.Dimensions {
				if dis {
					gtx = gtx.Disabled()
				}

				// Update before Layout so we can fire callbacks on this frame.
				// b.Layout re-drains the event queue (safe — second call finds nothing).
				if b.Update(gtx) {
					if props.OnChange != nil {
						props.OnChange(gtx, b.Value)
					}
					if props.Message != nil {
						mvu.MessageOp{Message: props.Message}.Add(gtx.Ops)
					}
				}

				foc := !dis && gtx.Focused(&b)

				// The pointer area is the whole control — the
				// footprint the symbol is centred in plus the title
				// beside it, both of which operate the box, as they
				// do on the platform.
				// The focus band rings the symbol's box and straddles it,
				// and a Clickable clips what it wraps, so the band is
				// collected inside and painted here, outside it.
				return focus.Around(gtx, func(gtx layout.Context) layout.Dimensions {
					return b.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						semantic.CheckBox.Add(gtx.Ops)
						if desc := props.Description; desc != "" || props.Title != "" {
							if desc == "" {
								desc = props.Title
							}
							semantic.DescriptionOp(desc).Add(gtx.Ops)
						}
						return drawCheckbox(gtx, tok, CheckboxRenderState{
							Checked:         b.Value,
							Focused:         foc,
							Disabled:        dis,
							Title:           props.Title,
							Surface:         standsOn(props.Surface, tok.platform),
							AccentSelection: props.AccentSelection != nil && props.AccentSelection(),
						})
					})
				})
			}
		})
	})
}

// RenderCheckbox produces a layout.Widget for a checkbox in an explicit visual
// state, without any event processing or rx machinery. Intended for golden-image
// testing and static demonstrations; production code should use Checkbox,
// which reads both of the parameters below off the theme.
//
// shaper and body draw CheckboxRenderState.Title — the whole text style, so
// typeface, weight, size and line height all reach the shaper. Pass
// tokens.DefaultTypography.SystemFont and a shaper for the default desktop
// look; a state with no title reaches neither, and a nil shaper is then
// harmless.
//
// Density is not a parameter: the static path always renders at
// tokens.Comfortable; density-aware rendering goes through Checkbox. Neither
// is the radius scale: the box is drawn at the corner it measures,
// checkboxCornerRadius, and the radio's disc has no corner at all.
func RenderCheckbox(
	shaper *text.Shaper,
	p tokens.PlatformColors,
	sp tokens.SpacingScale,
	body tokens.TextStyle,
	s CheckboxRenderState,
) layout.Widget {
	tok := resolvedTokens{
		platform: p,
		body:     body,
		capBand:  body.FaceMetrics().CapHeight,
		spacing:  sp,
		density:  tokens.Comfortable,
		shaper:   shaper,
	}
	return func(gtx layout.Context) layout.Dimensions {
		return focus.Around(gtx, func(gtx layout.Context) layout.Dimensions {
			return drawCheckbox(gtx, tok, s)
		})
	}
}

// titleBeside draws title in fg to the right of a symbol whose right edge is at
// symbolRight, in a row rowH px tall, and answers the width it took — the
// measured gap plus the title's own — or zero when there is no title. It is
// the one title a checkbox and a radio draw, so the two cannot drift.
//
// The cap band, baseline up to the cap height, is centred on the symbol's row.
// MEASURED, save-dialog-{light,dark}.png: "Show startup screen" caps run
// y 375–385 against a square of y 372–387, a band centre of 380.0 against the
// square's 379.5, and "Stay open after run handler" agrees — the same
// half-pixel-low rounding the field's prompt takes. The offset is
// capBandOffset's and may be negative: the body role's line box is taller
// than the measured 22 px row, so the line box hangs above the row while the
// band sits where the platform puts it.
func titleBeside(gtx layout.Context, tok resolvedTokens, symbolRight, rowH int, title string, fg color.NRGBA) int {
	if title == "" {
		return 0
	}
	gap := gtx.Dp(controlTitleGap)

	inner := gtx
	inner.Constraints = layout.Constraints{Max: image.Pt(gtx.Constraints.Max.X-symbolRight-gap, gtx.Constraints.Max.Y)}
	if inner.Constraints.Max.X < 1 {
		inner.Constraints.Max.X = 1
	}

	mTitle := op.Record(gtx.Ops)
	dims := typeset.Text(inner, tok.shaper, title, tok.body, fg, 1)
	call := mTitle.Stop()

	st := op.Offset(image.Pt(symbolRight+gap, capBandOffset(gtx, tok, rowH, dims))).Push(gtx.Ops)
	call.Add(gtx.Ops)
	st.Pop()

	return gap + dims.Size.X
}

// drawCheckbox renders the checkbox into gtx. All visual state comes from s;
// no event queries are performed here.
func drawCheckbox(gtx layout.Context, tok resolvedTokens, s CheckboxRenderState) layout.Dimensions {
	// Sizing rule: the visual symbol keeps its measured 16 dp box at every
	// density; the footprint (the row the symbol is centred in) is the
	// density's checkbox row, and the footprint is the pointer target —
	// the platform gives a pointer the checkbox's row, never its symbol.
	boxSz := gtx.Dp(checkboxBoxSize)
	ctlSz := gtx.Dp(unit.Dp(tok.density.CheckboxRowHeight))
	if ctlSz < boxSz {
		ctlSz = boxSz
	}

	offX := (ctlSz - boxSz) / 2
	offY := (ctlSz - boxSz) / 2

	boxRect := image.Rectangle{
		Min: image.Pt(offX, offY),
		Max: image.Pt(offX+boxSz, offY+boxSz),
	}
	boxRad := gtx.Dp(checkboxCornerRadius)
	rrectOuter := clip.RRect{Rect: boxRect, SE: boxRad, SW: boxRad, NE: boxRad, NW: boxRad}

	// Every name the box draws that carries a coverage is flattened onto
	// what lies under it: the surface for the unchecked fill and the ring,
	// and the fill for the check.
	standsOn := surface.Or(s.Surface, tok.platform.WindowBackground)

	switch {
	case s.Disabled:
		// A switched-off box is one fill and no edge at all: the
		// platform's switched-off box, measured. MEASURED,
		// save-panel-untinted-{light,dark}.png: the two switched-off
		// "Options:" checkboxes read #f2f2f2 light and #292929 dark on the
		// sheet, with no edge column in either scheme.
		fill := tok.platform.DisabledControlFill
		paint.FillShape(gtx.Ops, fill, rrectOuter.Op(gtx.Ops))
		if s.Checked {
			// No stored capture holds a switched-off CHECKED box, so the
			// mark takes the colour the switched-off title takes beside it:
			// the platform's tertiary label, which reproduces the measured
			// #bdbdbd light and #595f62 dark of both checkbox titles on the
			// sheet. The capture is on the reference's list. No contrast
			// floor applies to either: the platform chose to draw its
			// switched-off controls below any floor, and the title stands
			// there as measured — |Lc| 35.6 light and -16.1 dark on the
			// sheet. On this fill the mark reads |Lc| 33 light and 17 dark,
			// the same order, which is the platform's relation between its
			// switched-off drawings recorded rather than derived away.
			drawCheck(gtx, boxRect, boxSz, vgcolor.Flatten(tok.platform.TertiaryLabel, fill))
		}

	case s.Checked:
		// The fill says a value has been set; the check says what setting it
		// means. Without the mark the checked state is a swatch, and a list
		// of them carries completion in hue alone — which is the one channel
		// a reader may not have. So the box draws a check, in the foreground
		// the platform names for text on a fill its accent paints, in the
		// platform's measured shape. The fill is the accent the platform
		// draws, not controlAccentColor: MEASURED, the Keyboard Shortcuts
		// sheet's checked box reads #157bf5 light and #1b81fb dark.
		paint.FillShape(gtx.Ops, tok.platform.AccentFill, rrectOuter.Op(gtx.Ops))
		drawCheck(gtx, boxRect, boxSz, tok.platform.AlternateSelectedControlText)

	default:
		// An unchecked box is one fill and no edge: a filled square a step
		// apart from the fill it stands on, darker in light and lighter in
		// dark. The step is the platform's separator flattened onto that
		// fill, so the box stays seen on any surface. MEASURED, the find
		// bar's Replace box: #e6e6e6 on #ffffff light, #343434 on #1e1e1e
		// dark, #393e42 on the tinted #23292e, each the separator to the
		// byte. MEASURED, the Keyboard Shortcuts sheet's list: #d8d8d8 on the
		// row's #efefef light, #4a4d50 on #363a3d dark, the separator again.
		// The box's sides lie on whole pixels with no edge column.
		paint.FillShape(gtx.Ops, uncheckedFill(tok.platform, standsOn, s.AccentSelection), rrectOuter.Op(gtx.Ops))
	}

	// The focus ring, across the 16 dp box's own outline: the same band
	// the button, the chip and the field have, at the checkbox's measured
	// placement, checkboxRingOver over the box and the rest past it. The box
	// keeps its fill under it, whatever it is doing.
	//
	// It is in the slack the density's footprint holds around the symbol,
	// so nothing about the control moves when it takes the keyboard. At the
	// comfortable footprint's 22 dp that slack is 3 dp on each side and the
	// ring uses all 3; tighter, the band reaches past the footprint, which
	// is why focus.RingOver defers it rather than drawing it in place.
	if s.Focused && !s.Disabled {
		// What the band's part over the box lands on: the box's own fill,
		// the accent where it is checked. The box has no edge, so no field
		// edge is under the band. MEASURED, checkbox-focused-{light,dark}.png:
		// the band's column over the box reads the indicator over the box's
		// #e6e6e6 light and #343434 dark, at the coverage it has over the
		// sheet.
		boxFill := uncheckedFill(tok.platform, standsOn, s.AccentSelection)
		if s.Checked {
			boxFill = tok.platform.AccentFill
		}
		focus.RingOver(gtx, checkboxRingOver, boxRect, boxRad,
			focus.RingColor(tok.platform, standsOn), focus.RingColor(tok.platform, boxFill))
	}

	// The title is part of the control, as it is on the platform: it stands
	// at the measured gap after the square, in the platform's label colour
	// over the surface the box stands on, and it fades with the box.
	// MEASURED, save-dialog-{light,dark}.png: an enabled title on that sheet
	// reads #272727 and #dddfdf, which is what the platform's label lands on
	// over it, where both switched-off checkbox titles read #bdbdbd and
	// #595f62.
	titleFg := vgcolor.Flatten(tok.platform.Label, standsOn)
	switch {
	case s.Disabled:
		titleFg = vgcolor.Flatten(tok.platform.TertiaryLabel, standsOn)
	case s.AccentSelection:
		titleFg = tok.platform.AlternateSelectedControlText
	}
	w := ctlSz
	if beside := titleBeside(gtx, tok, boxRect.Max.X, ctlSz, s.Title, titleFg); beside > 0 {
		// The control ends at the title's last column; the row's height
		// stays the measured one whatever the title's line box is.
		w = boxRect.Max.X + beside
	}

	return layout.Dimensions{Size: image.Pt(w, ctlSz)}
}

// accentSelectionCoverage is the coverage of the unchecked box on a selected
// row in the accent fill: the selection's white text colour,
// AlternateSelectedControlText, at 51/255 over that fill, in both schemes.
//
// MEASURED, keyboard-shortcuts-row-selected-{light,dark}.png: the box reads
// #4086e5 on the selection's #1068de light and #3e7dd8 on #0e5dce dark, with
// no edge. Solved per channel, white's coverage is 0.201, 0.199 and 0.212
// light, 0.199, 0.198 and 0.204 dark. On an unselected row the same box
// is the separator.
const accentSelectionCoverage = 51

// uncheckedFill is the fill of an enabled unchecked box on the surface
// standsOn: the platform's separator flattened onto that surface, or the
// selection's text colour at accentSelectionCoverage flattened onto it where
// the surface is a selected row in the accent fill.
func uncheckedFill(p tokens.PlatformColors, standsOn color.NRGBA, accentSelection bool) color.NRGBA {
	if accentSelection {
		white := p.AlternateSelectedControlText
		white.A = accentSelectionCoverage
		return vgcolor.Flatten(white, standsOn)
	}
	return vgcolor.Flatten(p.Separator, standsOn)
}

// drawCheck strokes the check into the box at boxRect, boxSz px on a side, in
// the given foreground. One drawing serves every state that carries a mark:
// the check's geometry does not move when the box is switched off, only the
// colour it is stroked in.
func drawCheck(gtx layout.Context, boxRect image.Rectangle, boxSz int, foreground color.NRGBA) {
	scale := float32(boxSz) / checkGrid
	org := f32.Pt(float32(boxRect.Min.X), float32(boxRect.Min.Y))
	var check clip.Path
	check.Begin(gtx.Ops)
	for i, u := range checkLine {
		at := org.Add(f32.Pt(u.X*scale, u.Y*scale))
		if i == 0 {
			check.MoveTo(at)
		} else {
			check.LineTo(at)
		}
	}
	paint.FillShape(gtx.Ops, foreground, clip.Stroke{
		Path:  check.End(),
		Width: checkStroke * scale,
	}.Op())
}

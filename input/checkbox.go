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
	"github.com/vibrantgio/components/internal/control"
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

// controlEdgeWidth is the width a control draws its own edge at: 1 dp,
// MEASURED off the save dialog's "Tags:" text field, the sheet's one
// unfocused enabled control that draws an edge at all.
//
// The field's box runs x 264–495 and y 243–269. A run across its straight
// side gives one column at x=264 and one at x=495; a run down it gives one
// row at y=243 and one at y=269. Each is a single pixel of #f3f3f3 light and
// #2c3338 dark — [tokens.PlatformColors.FieldEdge] to the byte — with the
// sheet on one side of it and the field's interior on the other. No stored
// capture holds an enabled checkbox or an unselected radio, so the box and
// the disc spend the field's hairline; that capture is on the reference's
// list.
const controlEdgeWidth = unit.Dp(1)

// controlLabelGap is the leading gap a checkbox's square — or a radio's
// circle — spends on the label beside it.
//
// MEASURED, save-dialog-{light,dark}.png: both "Options:" squares end at
// column 279, so their edge is 280, and both labels' first covered column is
// 286 — six columns clear, in both appearances and both rows.
//
// It is spent whole, where [control.TextLeadDp] spends its reading less the
// capture's own first letter's bearing. The difference is what the capture
// shows: the field's selection fill stands behind its value, so the platform's
// own origin is there to be read and the bearing can be told from it. Here
// only the covered columns are visible, so the six is spent to the first of
// them and the capture's own labels — which begin with an S the body role's
// face bears no column on — are covered column for column. A label whose
// first glyph does carry a bearing stands one column further out, as it
// would on the platform.
const controlLabelGap = unit.Dp(6)

// The check mark is drawn on components/icons' grid rather than on one of
// its own, so that the library has a single answer to "what does a stroke
// weigh". checkGrid is that grid — 24 units, whatever size the drawing is
// realized at — and checkBandUnits is its diagonal band measure: 2 units,
// the compensation a 45-degree edge needs to cover device pixels whole,
// against 1.5 for an axis-aligned one. The check is nothing but two
// 45-degree arms, so it takes the diagonal measure throughout.
const (
	checkGrid      = 24.0
	checkBandUnits = 2.0
)

// checkLine is the check's centre line on that grid: in from the left, down
// to the turn, then up and out to the right. Both arms run at exactly 45
// degrees and every corner sits on the icon set's 1.5 sub-grid. Stroked, the
// figure spans 17 by 12.5 units, inside the 20-unit allowance a diagonal
// form is given — a diagonal drawing fills a square keyline less than a
// square one does, which is why that allowance is the wider of the two.
//
// The set's own files draw their caps and turns as an explicit closed
// contour because their SVG backend cannot ask for a line cap or a join.
// Here nothing is going through that backend, and Gio's clip.Stroke caps and
// joins round on its own, so the same figure is the centre line plus a
// width. It is the identical drawing, arrived at with three points instead
// of fourteen.
var checkLine = [3]f32.Point{
	{X: 4.5, Y: 12},
	{X: 9, Y: 16.5},
	{X: 19.5, Y: 6},
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

	// Label is the control's own text, drawn beside the box. Empty draws
	// the box alone.
	Label string

	// Surface is the opaque fill the control stands on. Its focus ring rides
	// in the slack around the glyph, so the platform's keyboard focus
	// indicator — a coverage rather than a colour — lands on this, and so
	// does the glyph's own edge, which is drawn as a shape the fill is inset
	// inside. The zero value — no colour — is the window's own plane.
	Surface color.NRGBA
}

// CheckboxProps configures a Checkbox instance.
type CheckboxProps struct {
	// Label is the control's own text, drawn beside the box and part of the
	// control: the whole row operates the box. Empty draws the box alone.
	Label string

	// Description is the screen-reader label. Empty falls back to Label.
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
	// label is drawn in the body role, so the typography emission supplies
	// the text style, its cap band and the theme's cached shaper; a checkbox
	// without a label never reaches them.
	resolved := rx.SwitchMap(th, func(t theme.Theme) rx.Observable[resolvedTokens] {
		return rx.Map(
			rx.CombineLatest4(t.Platform, t.Typography, t.Spacing, t.Density),
			func(n rx.Tuple4[tokens.PlatformColors, tokens.Typography, tokens.SpacingScale, tokens.Density]) resolvedTokens {
				typ := n.Second
				return resolvedTokens{
					platform: n.First,
					body:     typ.BodyLarge,
					capBand:  typ.FaceMetrics(typ.BodyLarge).CapHeight,
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
				// footprint the glyph is centred in plus the label
				// beside it, both of which operate the box, as they
				// do on the platform.
				return b.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					semantic.CheckBox.Add(gtx.Ops)
					if desc := props.Description; desc != "" || props.Label != "" {
						if desc == "" {
							desc = props.Label
						}
						semantic.DescriptionOp(desc).Add(gtx.Ops)
					}
					return drawCheckbox(gtx, tok, CheckboxRenderState{
						Checked:  b.Value,
						Focused:  foc,
						Disabled: dis,
						Label:    props.Label,
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
// shaper and body draw CheckboxRenderState.Label — the whole text style, so
// typeface, weight, size and line height all reach the shaper. Pass
// tokens.DefaultTypography.BodyLarge and a shaper for the default desktop
// look; a state with no label reaches neither, and a nil shaper is then
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
		return drawCheckbox(gtx, tok, s)
	}
}

// labelBeside draws label in fg to the right of a glyph whose right edge is at
// glyphRight, in a row rowH px tall, and answers the width it took — the
// measured gap plus the label's own — or zero when there is no label. It is
// the one label a checkbox and a radio draw, so the two cannot drift.
//
// The cap band, baseline up to the cap height, is centred on the glyph's row.
// MEASURED, save-dialog-{light,dark}.png: "Show startup screen" caps run
// y 375–385 against a square of y 372–387, a band centre of 380.0 against the
// square's 379.5, and "Stay open after run handler" agrees — the same
// half-pixel-low rounding the field's prompt takes. The offset is
// capBandOffset's and may be negative: the body role's line box is taller
// than the measured 22 px row, so the line box hangs above the row while the
// band sits where the platform puts it.
func labelBeside(gtx layout.Context, tok resolvedTokens, glyphRight, rowH int, label string, fg color.NRGBA) int {
	if label == "" {
		return 0
	}
	gap := gtx.Dp(controlLabelGap)
	f, wl, textSize := bodyLabel(tok)

	inner := gtx
	inner.Constraints = layout.Constraints{Max: image.Pt(gtx.Constraints.Max.X-glyphRight-gap, gtx.Constraints.Max.Y)}
	if inner.Constraints.Max.X < 1 {
		inner.Constraints.Max.X = 1
	}

	mMat := op.Record(gtx.Ops)
	paint.ColorOp{Color: fg}.Add(gtx.Ops)
	mat := mMat.Stop()

	mLabel := op.Record(gtx.Ops)
	dims := typeset.Layout(inner, tok.shaper, wl, f, textSize, label, mat)
	call := mLabel.Stop()

	st := op.Offset(image.Pt(glyphRight+gap, capBandOffset(gtx, tok, rowH, dims))).Push(gtx.Ops)
	call.Add(gtx.Ops)
	st.Pop()

	return gap + dims.Size.X
}

// drawCheckbox renders the checkbox into gtx. All visual state comes from s;
// no event queries are performed here.
func drawCheckbox(gtx layout.Context, tok resolvedTokens, s CheckboxRenderState) layout.Dimensions {
	// Sizing rule: the visual glyph keeps its measured 16 dp box at every
	// density; the footprint (the row the glyph is centred in) is the
	// density's checkbox row, and the footprint is the pointer target —
	// the platform gives a pointer the checkbox's row, never its glyph.
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
	// what lies under it: the surface for the edge and the ring, which are
	// shapes the fill is inset inside, and the fill for the check.
	standsOn := surface.Or(s.Surface, tok.platform.WindowBackground)

	switch {
	case s.Disabled:
		// A switched-off box is one fill and no edge at all. MEASURED,
		// save-dialog-light.png and save-dialog-dark.png: the two
		// switched-off "Options:" checkboxes read #f2f2f2 light and #2e3439
		// dark, and seventeen rows above them on the same sheet the enabled
		// "File Format:" pop-up reads the push button's own #ececec and
		// #333a3f. That is the platform's control fill at
		// tokens.DisabledCoverage over the sheet — exactly in light, one
		// 255th over on dark green and blue — which is what the push button
		// does when it is switched off. The box draws no edge column in
		// either appearance: its rim is a one-pixel antialiased ramp from
		// this fill to the sheet.
		fill := control.Faded(tok.platform.PushButtonFill, standsOn)
		paint.FillShape(gtx.Ops, fill, rrectOuter.Op(gtx.Ops))
		if s.Checked {
			// No stored capture holds a switched-off CHECKED box, so the
			// mark takes the colour the switched-off label takes beside it:
			// the platform's tertiary label, which reproduces the measured
			// #bdbdbd light and #595f62 dark of both checkbox labels on the
			// sheet. The capture is on the reference's list. No contrast
			// floor applies to either: the platform chose to draw its
			// switched-off controls below any floor, and the label stands
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
		// the platform names for text on a fill its accent paints, at the
		// icon set's weight.
		paint.FillShape(gtx.Ops, tok.platform.ControlAccent, rrectOuter.Op(gtx.Ops))
		drawCheck(gtx, boxRect, boxSz, tok.platform.AlternateSelectedControlText)

	default:
		// Edge as nested fills: outer rect in the edge colour, inner rect in
		// the box's own fill. Avoids clip.Stroke anti-aliasing variance in
		// tests. The edge is one pixel of the platform's field hairline —
		// controlEdgeWidth's reading — because no stored capture holds an
		// enabled checkbox and the field beside it on the same sheet is what
		// the platform draws an edge on.
		borderPx := gtx.Dp(controlEdgeWidth)
		innerRad := boxRad - borderPx
		if innerRad < 0 {
			innerRad = 0
		}
		innerRect := image.Rectangle{
			Min: image.Pt(offX+borderPx, offY+borderPx),
			Max: image.Pt(offX+boxSz-borderPx, offY+boxSz-borderPx),
		}
		rrectInner := clip.RRect{Rect: innerRect, SE: innerRad, SW: innerRad, NE: innerRad, NW: innerRad}
		paint.FillShape(gtx.Ops, control.Border(tok.platform), rrectOuter.Op(gtx.Ops))
		paint.FillShape(gtx.Ops, control.Fill(tok.platform), rrectInner.Op(gtx.Ops))
	}

	// The focus ring: focus.Width around the box, clear of it, in the one
	// colour focus.Ring answers for the scheme — the same pixel the button,
	// the chip and the field beside it draw. It rides in the slack between
	// the 16 dp glyph and the density's footprint, so taking focus moves
	// nothing and the ring is the same ring whatever the box is doing.
	//
	// The ring must sit outside the glyph with clear space on both sides: a
	// ring that touches the edge it marks cannot be read separately from it,
	// and a checked box has no free edge to promote at all — its edge is the
	// accent fill that says it is checked.
	if s.Focused && !s.Disabled {
		w := gtx.Dp(focus.Width)
		out := w + w/2 // stroke centreline: the band spans w..2w clear of the box
		r := boxRad + out
		ring := clip.RRect{
			Rect: image.Rectangle{
				Min: boxRect.Min.Sub(image.Pt(out, out)),
				Max: boxRect.Max.Add(image.Pt(out, out)),
			},
			SE: r, SW: r, NE: r, NW: r,
		}
		paint.FillShape(gtx.Ops, focus.Ring(tok.platform, standsOn), clip.Stroke{
			Path:  ring.Path(gtx.Ops),
			Width: float32(w),
		}.Op())
	}

	// The label is part of the control, as it is on the platform: it stands
	// at the measured gap after the square, in the platform's label colour
	// over the surface the box stands on, and it fades with the box.
	// MEASURED, save-dialog-{light,dark}.png: an enabled label on that sheet
	// reads #272727 and #dddfdf, which is what the platform's label lands on
	// over it, where both switched-off checkbox labels read #bdbdbd and
	// #595f62.
	label := vgcolor.Flatten(tok.platform.Label, standsOn)
	if s.Disabled {
		label = vgcolor.Flatten(tok.platform.TertiaryLabel, standsOn)
	}
	w := ctlSz
	if beside := labelBeside(gtx, tok, boxRect.Max.X, ctlSz, s.Label, label); beside > 0 {
		// The control ends at the label's last column; the row's height
		// stays the measured one whatever the label's line box is.
		w = boxRect.Max.X + beside
	}

	return layout.Dimensions{Size: image.Pt(w, ctlSz)}
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
		Width: checkBandUnits * scale,
	}.Op())
}

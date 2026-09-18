package input

import (
	"image"
	"image/color"

	"gioui.org/io/semantic"
	"gioui.org/layout"
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
)

// radioCircleSize is the outer diameter of the radio circle, and
// radioDotSize the diameter of the dot inside it. The circle is the
// checkbox's measured side length: the two controls stand beside each other
// in one form, so one of them being wider than the other is a defect, and
// the dot is half of it. The platform draws the same 16 —
// system-settings-grouped-box-light.png and -dark.png, the selected
// "Automatically based on mouse or trackpad" radio at x 253–268, y 696–711,
// both appearances agreeing to the pixel.
const (
	radioCircleSize = checkboxBoxSize
	radioDotSize    = checkboxBoxSize / 2
)

// RadioRenderState holds explicit visual state for static rendering.
// All fields default to false (normal/unselected/idle).
// Intended for golden-image testing; production code obtains state from the
// Gio event system via Radio.
type RadioRenderState struct {
	Selected bool
	Focused  bool
	Disabled bool

	// Label is the option's own text, drawn beside the disc. Empty draws
	// the disc alone.
	Label string

	// Surface is the opaque fill the control stands on. Its focus ring rides
	// in the slack around the glyph, so the platform's keyboard focus
	// indicator — a coverage rather than a colour — lands on this, and so
	// does the glyph's own edge, which is drawn as a shape the fill is inset
	// inside. The zero value — no colour — is the window's own plane.
	Surface color.NRGBA
}

// RadioProps configures a Radio instance.
type RadioProps struct {
	// Label is the option's own text, drawn beside the disc and part of the
	// control: the whole row operates it. Empty draws the disc alone.
	Label string

	// Description is the screen-reader label. Empty falls back to Label.
	Description string

	// Selected is the initial selected state established on subscribe.
	Selected bool

	// Disabled, if non-nil, disables the radio when it emits true.
	Disabled rx.Observable[bool]

	// OnChange is called with the new selected value on every toggle.
	// This is the FRP callback path. The gtx argument is the layout.Context
	// active on the frame when the toggle is processed, allowing consumers to
	// emit mvu.MessageOp{Message: ...}.Add(gtx.Ops) inside the callback.
	OnChange func(gtx layout.Context, selected bool)

	// Message, if non-nil, causes the radio to emit mvu.MessageOp{Message}
	// on every toggle. This is the MVU integration path.
	Message any
}

// Radio returns an rx.Observable[layout.Widget] that emits a new widget
// whenever the theme or disabled state changes. Interaction state (selected value,
// focus) lives in the rx.Defer scope and persists across emissions.
//
// Both integration paths are supported:
//   - FRP: set RadioProps.OnChange.
//   - MVU: set RadioProps.Message; the component emits mvu.MessageOp on toggle.
func Radio(th rx.Observable[theme.Theme], props RadioProps) rx.Observable[layout.Widget] {
	disabled := props.Disabled
	if disabled == nil {
		disabled = rx.Of(false)
	}

	// Flatten the nested theme observables into a concrete snapshot. The
	// label is drawn in the body role, so the typography emission supplies
	// the text style, its cap band and the theme's cached shaper; a radio
	// without a label never reaches them.
	resolved := rx.SwitchMap(th, func(t theme.Theme) rx.Observable[resolvedTokens] {
		return rx.Map(
			rx.CombineLatest5(t.Platform, t.Typography, t.Spacing, t.Radius, t.Density),
			func(n rx.Tuple5[tokens.PlatformColors, tokens.Typography, tokens.SpacingScale, tokens.RadiusScale, tokens.Density]) resolvedTokens {
				typ := n.Second
				return resolvedTokens{
					platform: n.First,
					body:     typ.BodyLarge,
					capBand:  typ.FaceMetrics(typ.BodyLarge).CapHeight,
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
		var b widget.Bool
		b.Value = props.Selected

		return rx.Map(inputs, func(next rx.Tuple2[resolvedTokens, bool]) layout.Widget {
			tok, dis := next.First, next.Second

			return func(gtx layout.Context) layout.Dimensions {
				if dis {
					gtx = gtx.Disabled()
				}

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
				// beside it, both of which operate it, as they do on
				// the platform.
				return b.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					semantic.RadioButton.Add(gtx.Ops)
					if desc := props.Description; desc != "" || props.Label != "" {
						if desc == "" {
							desc = props.Label
						}
						semantic.DescriptionOp(desc).Add(gtx.Ops)
					}
					return drawRadio(gtx, tok, RadioRenderState{
						Selected: b.Value,
						Focused:  foc,
						Disabled: dis,
						Label:    props.Label,
					})
				})
			}
		})
	})
}

// RenderRadio produces a layout.Widget for a radio button in an explicit visual
// state, without any event processing or rx machinery. Intended for golden-image
// testing and static demonstrations; production code should use Radio.
//
// The parameters are [RenderCheckbox]'s and mean the same things: shaper and
// body draw RadioRenderState.Label, and density is not among them.
func RenderRadio(
	shaper *text.Shaper,
	p tokens.PlatformColors,
	sp tokens.SpacingScale,
	rad tokens.RadiusScale,
	body tokens.TextStyle,
	s RadioRenderState,
) layout.Widget {
	tok := resolvedTokens{
		platform: p,
		body:     body,
		capBand:  body.FaceMetrics().CapHeight,
		spacing:  sp,
		radius:   rad,
		density:  tokens.Comfortable,
		shaper:   shaper,
	}
	return func(gtx layout.Context) layout.Dimensions {
		return drawRadio(gtx, tok, s)
	}
}

// drawRadio renders the radio button into gtx. All visual state comes from s;
// no event queries are performed here.
func drawRadio(gtx layout.Context, tok resolvedTokens, s RadioRenderState) layout.Dimensions {
	// Sizing rule: the visual glyph keeps its 16 dp circle at every density;
	// the footprint (the row the glyph is centred in) is the density's
	// checkbox row, which the radio stands in beside the box, and the
	// footprint is the pointer target — the platform gives a pointer the
	// button's row, never its circle.
	circleSz := gtx.Dp(radioCircleSize)
	ctlSz := gtx.Dp(unit.Dp(tok.density.CheckboxRowHeight))
	if ctlSz < circleSz {
		ctlSz = circleSz
	}

	cx := ctlSz / 2
	cy := ctlSz / 2

	outerRect := image.Rectangle{
		Min: image.Pt(cx-circleSz/2, cy-circleSz/2),
		Max: image.Pt(cx+circleSz/2, cy+circleSz/2),
	}

	// Three drawings, not one with a colour swapped. Unselected, the glyph is
	// an edge with the control's own fill inside it — the same pair the box
	// and the field wear at rest. Selected, the whole circle is the accent
	// the platform paints a chosen control in, with the dot in the
	// foreground that colour is named against. Switched off, it is one fill
	// and no edge, as the box beside it is. Nested fills throughout:
	// clip.Stroke's anti-aliasing varies between GPU context initialisations
	// and these are golden-tested.
	// Every name the glyph draws that carries a coverage is flattened onto
	// what lies under it: the surface for the edge and the ring, which are
	// shapes the fill is inset inside, and the fill for the dot.
	standsOn := surface.Or(s.Surface, tok.platform.WindowBackground)

	dotRect := func() image.Rectangle {
		dotSz := gtx.Dp(radioDotSize)
		return image.Rectangle{
			Min: image.Pt(cx-dotSz/2, cy-dotSz/2),
			Max: image.Pt(cx+dotSz/2, cy+dotSz/2),
		}
	}

	switch {
	case s.Disabled:
		// The switched-off drawing the checkbox measures, for the reason
		// given in drawCheckbox: the platform's control fill at
		// tokens.DisabledCoverage over the surface the glyph stands on, and
		// no edge. The Save dialog holds no switched-off radio, so this is
		// the checkbox's reading carried across — the two controls stand
		// beside each other in one form and the platform draws them as one
		// family.
		fill := control.Faded(tok.platform.PushButtonFill, standsOn)
		paint.FillShape(gtx.Ops, fill, clip.Ellipse(outerRect).Op(gtx.Ops))
		if s.Selected {
			// No stored capture holds a switched-off SELECTED radio, so the
			// dot takes the colour the switched-off label takes beside it,
			// as the check does. The capture is on the reference's list. No
			// contrast floor applies here either: the platform chose to draw
			// its switched-off controls below any floor, and the label stands
			// there as measured — |Lc| 35.6 light and -16.1 dark on the
			// sheet.
			dot := vgcolor.Flatten(tok.platform.TertiaryLabel, fill)
			paint.FillShape(gtx.Ops, dot, clip.Ellipse(dotRect()).Op(gtx.Ops))
		}

	case s.Selected:
		paint.FillShape(gtx.Ops, tok.platform.ControlAccent, clip.Ellipse(outerRect).Op(gtx.Ops))
		paint.FillShape(gtx.Ops, tok.platform.AlternateSelectedControlText, clip.Ellipse(dotRect()).Op(gtx.Ops))

	default:
		// No capture holds an unselected enabled radio, so the edge and the
		// interior stand as they are and the capture is on the reference's
		// list.
		borderPx := gtx.Dp(2)
		innerRect := image.Rectangle{
			Min: image.Pt(outerRect.Min.X+borderPx, outerRect.Min.Y+borderPx),
			Max: image.Pt(outerRect.Max.X-borderPx, outerRect.Max.Y-borderPx),
		}
		paint.FillShape(gtx.Ops, control.Border(tok.platform), clip.Ellipse(outerRect).Op(gtx.Ops))
		paint.FillShape(gtx.Ops, control.Fill(tok.platform), clip.Ellipse(innerRect).Op(gtx.Ops))
	}

	// The focus ring, drawn exactly as the checkbox draws it: focus.Width
	// around the glyph, clear of it, in the one colour focus.Ring answers for
	// the scheme, riding in the slack between the 16 dp circle and the
	// density's footprint.
	//
	// A selected radio is why the ring cannot be the circle's own edge. That
	// edge is already the accent — it is what says the radio is chosen — so
	// recolouring it on focus would leave a focused chosen radio looking like
	// an unfocused one. Clear of the glyph, the ring is a mark the glyph does
	// not already carry, in any state.
	if s.Focused && !s.Disabled {
		w := gtx.Dp(focus.Width)
		out := w + w/2 // stroke centreline: the band spans w..2w clear of the circle
		ring := image.Rectangle{
			Min: outerRect.Min.Sub(image.Pt(out, out)),
			Max: outerRect.Max.Add(image.Pt(out, out)),
		}
		paint.FillShape(gtx.Ops, focus.Ring(tok.platform, standsOn), clip.Stroke{
			Path:  clip.Ellipse(ring).Path(gtx.Ops),
			Width: float32(w),
		}.Op())
	}

	// The label is part of the control, as it is on the platform, drawn
	// exactly as the checkbox draws it: the measured gap after the circle,
	// the platform's label colour over the surface the glyph stands on, and
	// faded with the glyph.
	label := vgcolor.Flatten(tok.platform.Label, standsOn)
	if s.Disabled {
		label = vgcolor.Flatten(tok.platform.TertiaryLabel, standsOn)
	}
	w := ctlSz
	if beside := labelBeside(gtx, tok, outerRect.Max.X, ctlSz, s.Label, label); beside > 0 {
		w = outerRect.Max.X + beside
	}

	return layout.Dimensions{Size: image.Pt(w, ctlSz)}
}

package input

import (
	"image"

	"gioui.org/io/semantic"
	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"

	"github.com/reactivego/rx"
	"github.com/vibrantgio/components/internal/hit"
	"github.com/vibrantgio/mvu"
	"github.com/vibrantgio/theme/theme"
	"github.com/vibrantgio/theme/tokens"
)

// radioCircleSize is the outer diameter of the radio circle.
const radioCircleSize = unit.Dp(20)

// radioDotSize is the diameter of the inner selected dot.
const radioDotSize = unit.Dp(10)

// RadioRenderState holds explicit visual state for static rendering.
// All fields default to false (normal/unselected/idle).
// Intended for golden-image testing; production code obtains state from the
// Gio event system via Radio.
type RadioRenderState struct {
	Selected bool
	Focused  bool
	Disabled bool
}

// RadioProps configures a Radio instance.
type RadioProps struct {
	// Description is the screen-reader label.
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
// whenever the theme or disabled state changes. Widget state (selected value,
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
	// radio draws no text, so unlike TextField/Dropdown it does not
	// subscribe to the theme's Typography and leaves the snapshot's body
	// style and shaper zero.
	resolved := rx.SwitchMap(th, func(t theme.Theme) rx.Observable[resolvedTokens] {
		return rx.Map(
			rx.CombineLatest4(t.Color, t.Spacing, t.Radius, t.Density),
			func(n rx.Tuple4[tokens.ColorTokens, tokens.SpacingScale, tokens.RadiusScale, tokens.Density]) resolvedTokens {
				return resolvedTokens{color: n.First, spacing: n.Second, radius: n.Third, density: n.Fourth}
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

				// The pointer area is at least MinHitTarget (44 dp) on
				// each axis, centred on the visual footprint: density
				// shrinks the drawn control, never the hit target.
				return hit.Extend(gtx, gtx.Dp(unit.Dp(tok.density.MinHitTarget())), b.Layout, func(gtx layout.Context) layout.Dimensions {
					semantic.RadioButton.Add(gtx.Ops)
					if props.Description != "" {
						semantic.DescriptionOp(props.Description).Add(gtx.Ops)
					}
					return drawRadio(gtx, tok, RadioRenderState{
						Selected: b.Value,
						Focused:  foc,
						Disabled: dis,
					})
				})
			}
		})
	})
}

// RenderRadio produces a layout.Widget for a radio button in an explicit visual
// state, without any event processing or rx machinery. Intended for golden-image
// testing and static demonstrations; production code should use Radio.
func RenderRadio(
	colors tokens.ColorTokens,
	sp tokens.SpacingScale,
	rad tokens.RadiusScale,
	s RadioRenderState,
) layout.Widget {
	// Density is not a parameter (the signature predates E1.3): the static
	// path renders at tokens.Comfortable; density-aware rendering goes
	// through Radio.
	tok := resolvedTokens{color: colors, spacing: sp, radius: rad, density: tokens.Comfortable}
	return func(gtx layout.Context) layout.Dimensions {
		return drawRadio(gtx, tok, s)
	}
}

// drawRadio renders the radio button into gtx. All visual state comes from s;
// no event queries are performed here.
func drawRadio(gtx layout.Context, tok resolvedTokens, s RadioRenderState) layout.Dimensions {
	// E1.3 sizing rule: the visual glyph keeps its 20 dp circle at every
	// density; the footprint (the touch row the glyph is centred in) is the
	// density's control height — 36 dp Comfortable, 28 dp Compact. The
	// glyph's circle is never the pointer target: the live path extends the
	// hit area to at least 44 dp around this footprint via hit.Extend.
	circleSz := gtx.Dp(radioCircleSize)
	ctlSz := gtx.Dp(unit.Dp(tok.density.ControlHeight))
	if ctlSz < circleSz {
		ctlSz = circleSz
	}

	cx := ctlSz / 2
	cy := ctlSz / 2

	outerRect := image.Rectangle{
		Min: image.Pt(cx-circleSz/2, cy-circleSz/2),
		Max: image.Pt(cx+circleSz/2, cy+circleSz/2),
	}

	borderPx := gtx.Dp(2)

	// Focus ring: 2dp stroke centred on the circle boundary; the outer half
	// (1dp) is visible outside the circle fill and stays within the
	// footprint. Draw first so the circle overdrawing covers only the inner half.
	if s.Focused {
		paint.FillShape(gtx.Ops, tok.color.FocusRing(), clip.Stroke{
			Path:  clip.Ellipse(outerRect).Path(gtx.Ops),
			Width: float32(gtx.Dp(2)),
		}.Op())
	}

	if s.Selected {
		fill := tok.color.Primary
		if s.Disabled {
			fill = tokens.Disabled(fill)
		}
		// Outer ring in primary, surface gap, inner dot in primary.
		// Nested-fill technique avoids clip.Stroke anti-aliasing variance.
		paint.FillShape(gtx.Ops, fill, clip.Ellipse(outerRect).Op(gtx.Ops))
		innerRect := image.Rectangle{
			Min: image.Pt(outerRect.Min.X+borderPx, outerRect.Min.Y+borderPx),
			Max: image.Pt(outerRect.Max.X-borderPx, outerRect.Max.Y-borderPx),
		}
		paint.FillShape(gtx.Ops, tok.color.Surface, clip.Ellipse(innerRect).Op(gtx.Ops))
		dotSz := gtx.Dp(radioDotSize)
		dotRect := image.Rectangle{
			Min: image.Pt(cx-dotSz/2, cy-dotSz/2),
			Max: image.Pt(cx+dotSz/2, cy+dotSz/2),
		}
		paint.FillShape(gtx.Ops, fill, clip.Ellipse(dotRect).Op(gtx.Ops))
	} else {
		// Border as nested fills: outer ellipse in border colour, inner
		// ellipse in surface colour. Avoids clip.Stroke anti-aliasing
		// variance in tests.
		border := tok.color.Ramps.Neutral.Step(500) // strong border
		if s.Disabled {
			border = tokens.Disabled(border)
		}
		paint.FillShape(gtx.Ops, border, clip.Ellipse(outerRect).Op(gtx.Ops))
		innerRect := image.Rectangle{
			Min: image.Pt(outerRect.Min.X+borderPx, outerRect.Min.Y+borderPx),
			Max: image.Pt(outerRect.Max.X-borderPx, outerRect.Max.Y-borderPx),
		}
		paint.FillShape(gtx.Ops, tok.color.Surface, clip.Ellipse(innerRect).Op(gtx.Ops))
	}

	return layout.Dimensions{Size: image.Pt(ctlSz, ctlSz)}
}

package input

import (
	"image"
	"image/color"

	"gioui.org/io/semantic"
	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"

	"github.com/reactivego/rx"
	"github.com/vibrantgio/components/internal/control"
	"github.com/vibrantgio/components/internal/focus"
	"github.com/vibrantgio/components/internal/hit"
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
// the dot is half of it.
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

	// Surface is the opaque fill the control stands on. Its focus ring rides
	// in the slack around the glyph, so the platform's keyboard focus
	// indicator — a coverage rather than a colour — lands on this, and so
	// does the glyph's own edge, which is drawn as a shape the fill is inset
	// inside. The zero value — no colour — is the window's own plane.
	Surface color.NRGBA
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
	// radio draws no text, so unlike TextField it does not subscribe to the
	// theme's Typography and leaves the snapshot's body style and shaper
	// zero.
	resolved := rx.SwitchMap(th, func(t theme.Theme) rx.Observable[resolvedTokens] {
		return rx.Map(
			rx.CombineLatest4(t.Platform, t.Spacing, t.Radius, t.Density),
			func(n rx.Tuple4[tokens.PlatformColors, tokens.SpacingScale, tokens.RadiusScale, tokens.Density]) resolvedTokens {
				return resolvedTokens{platform: n.First, spacing: n.Second, radius: n.Third, density: n.Fourth}
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
	p tokens.PlatformColors,
	sp tokens.SpacingScale,
	rad tokens.RadiusScale,
	s RadioRenderState,
) layout.Widget {
	// Density is not a parameter: the static path always renders at
	// tokens.Comfortable; density-aware rendering goes through Radio.
	tok := resolvedTokens{platform: p, spacing: sp, radius: rad, density: tokens.Comfortable}
	return func(gtx layout.Context) layout.Dimensions {
		return drawRadio(gtx, tok, s)
	}
}

// drawRadio renders the radio button into gtx. All visual state comes from s;
// no event queries are performed here.
func drawRadio(gtx layout.Context, tok resolvedTokens, s RadioRenderState) layout.Dimensions {
	// Sizing rule: the visual glyph keeps its 16 dp circle at every density;
	// the footprint (the row the glyph is centred in) is the density's
	// control height. The glyph's circle is never the pointer target: the
	// live path extends the hit area to at least 44 dp around this footprint
	// via hit.Extend.
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
	innerRect := image.Rectangle{
		Min: image.Pt(outerRect.Min.X+borderPx, outerRect.Min.Y+borderPx),
		Max: image.Pt(outerRect.Max.X-borderPx, outerRect.Max.Y-borderPx),
	}

	// Two drawings, not one with a colour swapped. Unselected, the glyph is
	// an edge with the control's own fill inside it — the same pair the box
	// and the field wear at rest. Selected, the whole circle is the accent
	// the platform paints a chosen control in, with the dot in the
	// foreground that colour is named against. Nested fills throughout:
	// clip.Stroke's anti-aliasing varies between GPU context initialisations
	// and these are golden-tested.
	// Every name the glyph draws that carries a coverage is flattened onto
	// what lies under it: the surface for the edge and the ring, which are
	// shapes the fill is inset inside, and the fill for the dot.
	standsOn := surface.Or(s.Surface, tok.platform.WindowBackground)

	if s.Selected {
		fill := tok.platform.ControlAccent
		dot := tok.platform.AlternateSelectedControlText
		if s.Disabled {
			fill = vgcolor.Flatten(tokens.Disabled(fill), standsOn)
			dot = vgcolor.Flatten(tok.platform.DisabledControlText, fill)
		}
		paint.FillShape(gtx.Ops, fill, clip.Ellipse(outerRect).Op(gtx.Ops))

		dotSz := gtx.Dp(radioDotSize)
		dotRect := image.Rectangle{
			Min: image.Pt(cx-dotSz/2, cy-dotSz/2),
			Max: image.Pt(cx+dotSz/2, cy+dotSz/2),
		}
		paint.FillShape(gtx.Ops, dot, clip.Ellipse(dotRect).Op(gtx.Ops))
	} else {
		edge := control.Border(tok.platform)
		if s.Disabled {
			edge = vgcolor.Flatten(tok.platform.DisabledControlText, standsOn)
		}
		paint.FillShape(gtx.Ops, edge, clip.Ellipse(outerRect).Op(gtx.Ops))
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

	return layout.Dimensions{Size: image.Pt(ctlSz, ctlSz)}
}

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
	"github.com/vibrantgio/components/internal/focus"
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

	// Level is the level of the surface the radio stands on — the radio has
	// no level of its own — and the ring is derived against that surface, in
	// the same vocabulary the host names its own fill (tokens.SurfaceAt). A
	// dialog at tokens.Level2 passes Level2 and the unselected ring takes
	// whichever neutral step clears the floor over that surface. The zero
	// value is tokens.Level0, the window's own surface. A selected radio's
	// ring is the primary ink measured against that same surface
	// ([tokens.ColorTokens.InkOn]) rather than the bare accent pin, so it too
	// answers to the host it stands on.
	Level tokens.ElevationLevel
}

// RadioProps configures a Radio instance.
type RadioProps struct {
	// Description is the screen-reader label.
	Description string

	// Selected is the initial selected state established on subscribe.
	Selected bool

	// Level is the level of the surface the radio stands on — the radio has
	// no level of its own — copied straight into RadioRenderState.Level on
	// every frame: what the unselected ring is derived against. A container
	// that raises its surface passes its own level here; the zero value is
	// the window's own surface. See RadioRenderState.Level.
	Level tokens.ElevationLevel

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
						Level:    props.Level,
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
	// Density is not a parameter: the static path always renders at
	// tokens.Comfortable; density-aware rendering goes through Radio.
	tok := resolvedTokens{color: colors, spacing: sp, radius: rad, density: tokens.Comfortable}
	return func(gtx layout.Context) layout.Dimensions {
		return drawRadio(gtx, tok, s)
	}
}

// selectedRadioEdge is the colour a selected radio's edge is drawn in: the
// primary pin while it clears the graphic floor against the surface at level
// — the same surface controlBorder measures the resting edge against — and
// otherwise the step of the primary ramp that does
// ([tokens.ColorTokens.InkOn]). The bare Primary pin is not used directly
// because a pastel seed's primary can fall under the graphic floor against
// some host surfaces.
func selectedRadioEdge(c tokens.ColorTokens, level tokens.ElevationLevel) color.NRGBA {
	return c.InkOn(tokens.RolePrimary, c.SurfaceAt(level), tokens.GraphicFloor)
}

// drawRadio renders the radio button into gtx. All visual state comes from s;
// no event queries are performed here.
func drawRadio(gtx layout.Context, tok resolvedTokens, s RadioRenderState) layout.Dimensions {
	// Sizing rule: the visual glyph keeps its 20 dp circle at every density;
	// the footprint (the touch row the glyph is centred in) is the
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
	innerRect := image.Rectangle{
		Min: image.Pt(outerRect.Min.X+borderPx, outerRect.Min.Y+borderPx),
		Max: image.Pt(outerRect.Max.X-borderPx, outerRect.Max.Y-borderPx),
	}

	// Outer ellipse in the edge colour, the gap, and — when selected — the
	// dot. Nested fills avoid clip.Stroke anti-aliasing variance in tests.
	// The unselected ring is the radio's whole statement, so it is derived
	// rather than named: the neutral step that clears the graphic floor
	// against the surface the radio stands on (controlBorder). The gap inside
	// it is the glyph's own interior, so it takes the fill the glyph is
	// raised to (controlFill) — the same fill the box, the field and the
	// trigger carry, in the chosen state as much as the resting one: the dot
	// is drawn on the radio's surface, not on the host's.
	//
	// The selected edge is [selectedRadioEdge], measured against that same
	// surface.
	edge := controlBorder(tok.color, s.Level)
	if s.Selected {
		edge = selectedRadioEdge(tok.color, s.Level)
	}
	if s.Disabled {
		edge = tokens.Disabled(edge)
	}
	paint.FillShape(gtx.Ops, edge, clip.Ellipse(outerRect).Op(gtx.Ops))
	paint.FillShape(gtx.Ops, controlFill(tok.color, s.Level), clip.Ellipse(innerRect).Op(gtx.Ops))
	if s.Selected {
		fill := tok.color.Primary
		if s.Disabled {
			fill = tokens.Disabled(fill)
		}
		dotSz := gtx.Dp(radioDotSize)
		dotRect := image.Rectangle{
			Min: image.Pt(cx-dotSz/2, cy-dotSz/2),
			Max: image.Pt(cx+dotSz/2, cy+dotSz/2),
		}
		paint.FillShape(gtx.Ops, fill, clip.Ellipse(dotRect).Op(gtx.Ops))
	}

	// The focus ring, drawn exactly as the checkbox draws it: focus.Width
	// around the glyph, clear of it, in the one colour focus.Ring answers for
	// the scheme, riding in the slack between the 20 dp circle and the
	// density's footprint.
	//
	// A selected radio is why the ring cannot be the circle's own edge. That
	// edge is already primary — it is what says the radio is chosen — so
	// recolouring it on focus moves primary to a neighbouring step of primary
	// — 1.48:1 apart on the default palette — and a focused chosen radio comes
	// out indistinguishable from an unfocused one. Clear of the glyph, the
	// ring is a mark the glyph does not already carry, in any state.
	if s.Focused && !s.Disabled {
		w := gtx.Dp(focus.Width)
		out := w + w/2 // stroke centreline: the band spans w..2w clear of the circle
		ring := image.Rectangle{
			Min: outerRect.Min.Sub(image.Pt(out, out)),
			Max: outerRect.Max.Add(image.Pt(out, out)),
		}
		paint.FillShape(gtx.Ops, focus.Ring(tok.color), clip.Stroke{
			Path:  clip.Ellipse(ring).Path(gtx.Ops),
			Width: float32(w),
		}.Op())
	}

	return layout.Dimensions{Size: image.Pt(ctlSz, ctlSz)}
}

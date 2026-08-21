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
	"github.com/vibrantgio/components/internal/focus"
	"github.com/vibrantgio/components/internal/hit"
	"github.com/vibrantgio/mvu"
	"github.com/vibrantgio/theme/theme"
	"github.com/vibrantgio/theme/tokens"
)

// checkboxBoxSize is the visual side length of the checkbox square.
const checkboxBoxSize = unit.Dp(20)

// CheckboxRenderState holds explicit visual state for static rendering.
// All fields default to false (normal/unchecked/idle).
// Intended for golden-image testing; production code obtains state from the
// Gio event system via Checkbox.
type CheckboxRenderState struct {
	Checked  bool
	Focused  bool
	Disabled bool
}

// CheckboxProps configures a Checkbox instance.
type CheckboxProps struct {
	// Description is the screen-reader label.
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
// whenever the theme or disabled state changes. Widget state (checked value,
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
	// checkbox draws no text, so unlike TextField/Dropdown it does not
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

				// The pointer area is at least MinHitTarget (44 dp) on
				// each axis, centred on the visual footprint: density
				// shrinks the drawn control, never the hit target.
				return hit.Extend(gtx, gtx.Dp(unit.Dp(tok.density.MinHitTarget())), b.Layout, func(gtx layout.Context) layout.Dimensions {
					semantic.CheckBox.Add(gtx.Ops)
					if props.Description != "" {
						semantic.DescriptionOp(props.Description).Add(gtx.Ops)
					}
					return drawCheckbox(gtx, tok, CheckboxRenderState{
						Checked:  b.Value,
						Focused:  foc,
						Disabled: dis,
					})
				})
			}
		})
	})
}

// RenderCheckbox produces a layout.Widget for a checkbox in an explicit visual
// state, without any event processing or rx machinery. Intended for golden-image
// testing and static demonstrations; production code should use Checkbox.
func RenderCheckbox(
	colors tokens.ColorTokens,
	sp tokens.SpacingScale,
	rad tokens.RadiusScale,
	s CheckboxRenderState,
) layout.Widget {
	// Density is not a parameter (the signature predates E1.3): the static
	// path renders at tokens.Comfortable; density-aware rendering goes
	// through Checkbox.
	tok := resolvedTokens{color: colors, spacing: sp, radius: rad, density: tokens.Comfortable}
	return func(gtx layout.Context) layout.Dimensions {
		return drawCheckbox(gtx, tok, s)
	}
}

// drawCheckbox renders the checkbox into gtx. All visual state comes from s;
// no event queries are performed here.
func drawCheckbox(gtx layout.Context, tok resolvedTokens, s CheckboxRenderState) layout.Dimensions {
	// E1.3 sizing rule: the visual glyph keeps its 20 dp box at every
	// density; the footprint (the touch row the glyph is centred in) is the
	// density's control height — 36 dp Comfortable, 28 dp Compact. The
	// glyph's box is never the pointer target: the live path extends the hit
	// area to at least 44 dp around this footprint via hit.Extend.
	boxSz := gtx.Dp(checkboxBoxSize)
	ctlSz := gtx.Dp(unit.Dp(tok.density.ControlHeight))
	if ctlSz < boxSz {
		ctlSz = boxSz
	}

	offX := (ctlSz - boxSz) / 2
	offY := (ctlSz - boxSz) / 2

	boxRect := image.Rectangle{
		Min: image.Pt(offX, offY),
		Max: image.Pt(offX+boxSz, offY+boxSz),
	}
	boxRad := gtx.Dp(unit.Dp(tok.radius.Sm))

	// Border as nested fills: outer rect in border colour, inner rect in
	// surface colour. Avoids clip.Stroke anti-aliasing variance in tests.
	borderPx := gtx.Dp(2)
	innerRad := boxRad - borderPx
	if innerRad < 0 {
		innerRad = 0
	}
	rrectOuter := clip.RRect{Rect: boxRect, SE: boxRad, SW: boxRad, NE: boxRad, NW: boxRad}
	innerRect := image.Rectangle{
		Min: image.Pt(offX+borderPx, offY+borderPx),
		Max: image.Pt(offX+boxSz-borderPx, offY+boxSz-borderPx),
	}
	rrectInner := clip.RRect{Rect: innerRect, SE: innerRad, SW: innerRad, NE: innerRad, NW: innerRad}

	if s.Checked {
		fill := tok.color.Primary
		if s.Disabled {
			fill = tokens.Disabled(fill)
		}
		paint.FillShape(gtx.Ops, fill, rrectOuter.Op(gtx.Ops))
	} else {
		border := tok.color.Ramps.Neutral.Step(500) // strong border
		if s.Disabled {
			border = tokens.Disabled(border)
		}
		paint.FillShape(gtx.Ops, border, rrectOuter.Op(gtx.Ops))
		paint.FillShape(gtx.Ops, tok.color.Surface, rrectInner.Op(gtx.Ops))
	}

	// The focus ring: focus.Width around the box, clear of it, in the primary
	// rung that reads against the surface it lies on. It rides in the slack
	// between the 20 dp glyph and the density's footprint, so taking focus
	// moves nothing and the ring is the same ring whatever the box is doing.
	//
	// Clear of the box, and that gap is the fix. The ring used to be stroked
	// on the box's own boundary in neutral step 500 — the exact colour of the
	// unchecked border it landed on, 1.00:1 against the edge it was circling —
	// and the box then overdrew its inner half, leaving one device pixel of
	// grey against grey. A ring that touches the edge it marks cannot be read
	// separately from it, and a checked box has no free edge to promote at
	// all: its border is the primary fill that says it is checked. So the ring
	// sits outside the glyph with clear ground on both sides of it, which is
	// the one placement that reads in all four states.
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
		paint.FillShape(gtx.Ops, focus.Ring(tok.color, tok.color.Surface), clip.Stroke{
			Path:  ring.Path(gtx.Ops),
			Width: float32(w),
		}.Op())
	}

	return layout.Dimensions{Size: image.Pt(ctlSz, ctlSz)}
}

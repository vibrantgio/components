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
	"github.com/vibrantgio/mvu"
	"github.com/vibrantgio/spectrum/theme"
	"github.com/vibrantgio/spectrum/tokens"
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
			rx.CombineLatest3(t.Color, t.Spacing, t.Radius),
			func(n rx.Tuple3[tokens.ColorTokens, tokens.SpacingScale, tokens.RadiusScale]) resolvedTokens {
				return resolvedTokens{color: n.First, spacing: n.Second, radius: n.Third}
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

				return b.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
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
	tok := resolvedTokens{color: colors, spacing: sp, radius: rad}
	return func(gtx layout.Context) layout.Dimensions {
		return drawCheckbox(gtx, tok, s)
	}
}

// drawCheckbox renders the checkbox into gtx. All visual state comes from s;
// no event queries are performed here.
func drawCheckbox(gtx layout.Context, tok resolvedTokens, s CheckboxRenderState) layout.Dimensions {
	boxSz := gtx.Dp(checkboxBoxSize)
	hitSz := gtx.Dp(minHeight)
	if hitSz < boxSz {
		hitSz = boxSz
	}

	offX := (hitSz - boxSz) / 2
	offY := (hitSz - boxSz) / 2

	boxRect := image.Rectangle{
		Min: image.Pt(offX, offY),
		Max: image.Pt(offX+boxSz, offY+boxSz),
	}
	boxRad := gtx.Dp(unit.Dp(tok.radius.Sm))

	// Focus ring: 2dp stroke centred on the box boundary; the outer half (1dp)
	// is visible outside the box fill and stays within the 44dp hit target.
	// Draw first so the box overdrawing covers only the inner half.
	if s.Focused {
		rrect := clip.RRect{Rect: boxRect, SE: boxRad, SW: boxRad, NE: boxRad, NW: boxRad}
		paint.FillShape(gtx.Ops, tok.color.Primary, clip.Stroke{
			Path:  rrect.Path(gtx.Ops),
			Width: float32(gtx.Dp(2)),
		}.Op())
	}

	if s.Checked {
		fill := tok.color.Primary
		if s.Disabled {
			fill = withAlpha(fill, 0x61)
		}
		rrect := clip.RRect{Rect: boxRect, SE: boxRad, SW: boxRad, NE: boxRad, NW: boxRad}
		paint.FillShape(gtx.Ops, fill, rrect.Op(gtx.Ops))
	} else {
		// Border as nested fills: outer rect in border colour, inner rect in
		// surface colour. Avoids clip.Stroke anti-aliasing variance in tests.
		border := tok.color.Outline
		if s.Disabled {
			border = withAlpha(border, 0x61)
		}
		borderPx := gtx.Dp(2)
		innerRad := boxRad - borderPx
		if innerRad < 0 {
			innerRad = 0
		}
		rrectOuter := clip.RRect{Rect: boxRect, SE: boxRad, SW: boxRad, NE: boxRad, NW: boxRad}
		paint.FillShape(gtx.Ops, border, rrectOuter.Op(gtx.Ops))
		innerRect := image.Rectangle{
			Min: image.Pt(offX+borderPx, offY+borderPx),
			Max: image.Pt(offX+boxSz-borderPx, offY+boxSz-borderPx),
		}
		rrectInner := clip.RRect{Rect: innerRect, SE: innerRad, SW: innerRad, NE: innerRad, NW: innerRad}
		paint.FillShape(gtx.Ops, tok.color.Surface, rrectInner.Op(gtx.Ops))
	}

	return layout.Dimensions{Size: image.Pt(hitSz, hitSz)}
}

package input

import (
	"image"

	"gioui.org/f32"
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
	"github.com/vibrantgio/mvu"
	"github.com/vibrantgio/theme/theme"
	"github.com/vibrantgio/theme/tokens"
)

// checkboxBoxSize is the visual side length of the checkbox square: 16 dp,
// MEASURED off save-dialog-light.png and save-dialog-dark.png, the "Options:"
// checkbox at y 372–387, x 264–279 in both appearances, and recorded in the
// organization's macOS reference.
const checkboxBoxSize = unit.Dp(16)

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
	// checkbox draws no text, so unlike TextField it does not subscribe to
	// the theme's Typography and leaves the snapshot's body style and shaper
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
	p tokens.PlatformColors,
	sp tokens.SpacingScale,
	rad tokens.RadiusScale,
	s CheckboxRenderState,
) layout.Widget {
	// Density is not a parameter: the static path always renders at
	// tokens.Comfortable; density-aware rendering goes through Checkbox.
	tok := resolvedTokens{platform: p, spacing: sp, radius: rad, density: tokens.Comfortable}
	return func(gtx layout.Context) layout.Dimensions {
		return drawCheckbox(gtx, tok, s)
	}
}

// drawCheckbox renders the checkbox into gtx. All visual state comes from s;
// no event queries are performed here.
func drawCheckbox(gtx layout.Context, tok resolvedTokens, s CheckboxRenderState) layout.Dimensions {
	// Sizing rule: the visual glyph keeps its measured 16 dp box at every
	// density; the footprint (the row the glyph is centred in) is the
	// density's control height. The glyph's box is never the pointer target:
	// the live path extends the hit area to at least 44 dp around this
	// footprint via hit.Extend.
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

	// Edge as nested fills: outer rect in the edge colour, inner rect in the
	// box's own fill. Avoids clip.Stroke anti-aliasing variance in tests.
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
		// The fill says a value has been set; the check says what setting it
		// means. Without the mark the checked state is a swatch, and a list
		// of them carries completion in hue alone — which is the one channel
		// a reader may not have. So the box draws a check, in the foreground
		// the platform names for text on a fill its accent paints, at the
		// icon set's weight.
		fill := tok.platform.ControlAccent
		foreground := tok.platform.AlternateSelectedControlText
		if s.Disabled {
			fill = tokens.Disabled(fill)
			foreground = tok.platform.DisabledControlText
		}
		paint.FillShape(gtx.Ops, fill, rrectOuter.Op(gtx.Ops))

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
	} else {
		edge := control.Border(tok.platform)
		if s.Disabled {
			edge = tok.platform.DisabledControlText
		}
		paint.FillShape(gtx.Ops, edge, rrectOuter.Op(gtx.Ops))
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
		paint.FillShape(gtx.Ops, focus.Ring(tok.platform), clip.Stroke{
			Path:  ring.Path(gtx.Ops),
			Width: float32(w),
		}.Op())
	}

	return layout.Dimensions{Size: image.Pt(ctlSz, ctlSz)}
}

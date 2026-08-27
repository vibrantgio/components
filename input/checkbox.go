package input

import (
	"image"
	"image/color"

	"gioui.org/f32"
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

// graphicFloor is WCAG 1.4.11's contrast floor for a graphic that carries
// meaning without being text — 3:1. A checkbox is entirely such a graphic:
// nothing about its state is spelled out, so its edge owes the page this
// much and its mark owes the fill it is drawn on the same. The same is true
// of every other control in this package that says what it is with an edge
// — the radio, the text field, the dropdown trigger — which is why one
// floor serves the row (see controlBorder).
const graphicFloor = 3.0

// The check mark is drawn on components/icons' grid rather than on one of
// its own, so that the library has a single answer to "what does a stroke
// weigh". checkGrid is that grid — 24 units, whatever size the drawing is
// realized at — and checkBandUnits is its DIAGONAL band measure: 2 units,
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

// controlBorder is the ink of a control's resting edge — the unchecked box,
// the unselected radio, the text field, the dropdown trigger: the rung of
// the neutral ramp nearest its mid-value step that reaches graphicFloor
// against the ground the control stands on.
//
// It used to be neutral step 500, named once and drawn in both schemes at
// each of those four sites, and that is a pairing rather than a colour. The
// neutral ramps are paired — light and dark are realized at the same
// perceptual depths from opposite ends — so step 500 is the one rung that
// barely moves between schemes while the ground under it moves the whole
// way. The result was a border measuring 6.63:1 in the dark and 2.67:1 in
// the light, under the floor in the scheme most people read in, from a line
// of code that looks scheme-neutral. Asking the ramp which rung clears the
// floor answers 600 in the light scheme and 500 in the dark and needs to
// know nothing about either.
//
// ground is the storey the control is standing on, and the walk is taken
// against that storey's own fill rather than against the window's. It used
// to be level 0 unconditionally — the one ground a component that is never
// told where it was put can be sure of — and that assumption is what failed
// as soon as a control was placed inside a dialog: the light scheme's rung
// measures 2.94:1 over a level-2 plane and 2.15:1 over a level-3 one, both
// under the floor, from a derivation that was itself correct and merely
// aimed at the wrong ground. Handed the level, the same walk answers a
// deeper rung where the ground is deeper and the control keeps its edge
// wherever it stands.
//
// The control's own interior is the edge's other side, and every rung this
// walk answers clears the floor against it too: the interior is Surface —
// the level-1 storey — and the deeper the outer ground, the deeper the ink,
// so the inner pairing only widens. Measured over the four storeys the
// light scheme lands 3.55 / 3.55 / 5.46 / 5.46:1 inside and the dark 5.94:1
// throughout.
func controlBorder(c tokens.ColorTokens, ground tokens.ElevationLevel) color.NRGBA {
	return c.MarkOn(tokens.RoleNeutral, c.SurfaceAt(ground), graphicFloor)
}

// CheckboxRenderState holds explicit visual state for static rendering.
// The zero value is an unchecked, idle, enabled box on the window ground —
// so CheckboxRenderState{} is exactly today's default checkbox.
// Intended for golden-image testing; production code obtains state from the
// Gio event system via Checkbox.
type CheckboxRenderState struct {
	Checked  bool
	Focused  bool
	Disabled bool

	// Ground is the elevation storey of the surface hosting the checkbox —
	// the local ground its unchecked edge is derived against, in the same
	// vocabulary the host names its own fill (tokens.SurfaceAt). A dialog
	// at tokens.Level2 passes Level2 and the edge takes whichever neutral
	// rung clears the floor over that storey. The zero value is
	// tokens.Level0, the window ground, which resolves to exactly the walk
	// the border always performed — so every state written before this
	// field existed keeps its colours. A checked box ignores it: it
	// carries the accent fill, which is its own ground.
	Ground tokens.ElevationLevel
}

// CheckboxProps configures a Checkbox instance.
type CheckboxProps struct {
	// Description is the screen-reader label.
	Description string

	// Checked is the initial checked state established on subscribe.
	Checked bool

	// Ground is the elevation storey of the surface hosting the checkbox,
	// copied straight into CheckboxRenderState.Ground on every frame: the
	// local ground the unchecked box's edge is derived against. A
	// container that raises its surface (a level-2 dialog hosting a
	// "remember this" toggle) passes its own storey here; the zero value
	// is the window ground and keeps exactly the colours the border has
	// always had. See CheckboxRenderState.Ground.
	Ground tokens.ElevationLevel

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
						Ground:   props.Ground,
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
		// The fill says a colour has been applied; the check says what was
		// applied means. Without the mark the checked state is a swatch, and
		// a list of them carries completion in hue alone — which is the one
		// channel a reader may not have. So the box draws a check, in the
		// on-colour the fill is paired with, at the icon set's weight.
		fill := tok.color.Primary
		ink := tok.color.OnPrimary
		if s.Disabled {
			fill = tokens.Disabled(fill)
			ink = tokens.Disabled(ink)
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
		paint.FillShape(gtx.Ops, ink, clip.Stroke{
			Path:  check.End(),
			Width: checkBandUnits * scale,
		}.Op())
	} else {
		border := controlBorder(tok.color, s.Ground)
		if s.Disabled {
			border = tokens.Disabled(border)
		}
		paint.FillShape(gtx.Ops, border, rrectOuter.Op(gtx.Ops))
		paint.FillShape(gtx.Ops, tok.color.Surface, rrectInner.Op(gtx.Ops))
	}

	// The focus ring: focus.Width around the box, clear of it, in the primary
	// rung that reads against the storey the box stands on — s.Ground through
	// focus.Ground, the same walk the unchecked edge above takes, so a box in
	// a dialog wears an edge and a ring that were both measured against the
	// dialog. It rides in the slack
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
		paint.FillShape(gtx.Ops, focus.Ring(tok.color, focus.Ground(tok.color, s.Ground)), clip.Stroke{
			Path:  ring.Path(gtx.Ops),
			Width: float32(w),
		}.Op())
	}

	return layout.Dimensions{Size: image.Pt(ctlSz, ctlSz)}
}

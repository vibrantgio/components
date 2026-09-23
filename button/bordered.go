package button

import (
	"image"
	"image/color"

	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/unit"

	vgcolor "github.com/vibrantgio/theme/color"
	"github.com/vibrantgio/theme/tokens"

	"github.com/vibrantgio/components/internal/control"
	"github.com/vibrantgio/components/internal/controlface"
	"github.com/vibrantgio/components/internal/surface"
	"github.com/vibrantgio/components/pointershape"
)

// RenderBordered produces a layout.Widget for the platform's bordered
// control labelled with a symbol, drawn in an explicit visual state without
// event processing or rx machinery.
//
// The glyph is drawn by icon into a square the control's variant names,
// centred in a shape that variant names too — the chrome variant's capsule at
// d.ToolbarControlHeight, the form variant's push button at d.ControlHeight
// and rad.Md — and that shape is the pointer target. It takes no text style:
// there is no text to shape. The button doc's Variant table carries the
// height, face, shadow and shape each variant draws.
//
// Intended for golden-image testing and static rendering; production code
// uses Button with Props.Variant set to [Chrome] and Props.Icon set, or —
// where the caller owns the control's widget.Clickable — [BorderedShadow]
// around [BorderedFace].
func RenderBordered(
	icon func(gtx layout.Context, sizePx int, col color.NRGBA),
	p tokens.PlatformColors,
	rad tokens.RadiusScale,
	d tokens.Density,
	s RenderState,
) layout.Widget {
	face := BorderedFace(icon, p, rad, d, s)
	return func(gtx layout.Context) layout.Dimensions {
		return BorderedShadow(gtx, p, s, face)
	}
}

// BorderedFace draws the bordered control's own box and the symbol in it, and
// nothing outside that box. [RenderState.Variant] says which of the two boxes
// it is.
//
// It is separate from the shadow because gioui.org/widget's Clickable clips
// whatever it wraps to the box its layout.Widget reports, and the shadow falls
// outside the box: a caller that owns the control's Clickable lays this out
// INSIDE it and calls [BorderedShadow] AROUND it. [RenderBordered] is the two
// composed for a caller that owns neither.
func BorderedFace(
	icon func(gtx layout.Context, sizePx int, col color.NRGBA),
	p tokens.PlatformColors,
	rad tokens.RadiusScale,
	d tokens.Density,
	s RenderState,
) layout.Widget {
	tok := resolvedTokens{platform: p, radius: rad, density: d}
	return func(gtx layout.Context) layout.Dimensions {
		return drawBorderedSymbol(gtx, icon, tok, s)
	}
}

// BorderedShadow lays w out and paints under it the drop shadow the bordered
// control w drew casts on the band it stands on, sized to the box w reported
// and at the coverage s carries — the platform's own, faded with the control
// while it is switched off.
//
// w is [BorderedFace] or something wrapping it, a widget.Clickable included.
// A control of the [Form] variant casts none: the shadow was measured on a
// band and belongs to the chrome variant, so this lays w out and paints
// nothing under it.
func BorderedShadow(gtx layout.Context, p tokens.PlatformColors, s RenderState, w layout.Widget) layout.Dimensions {
	if s.Variant != Chrome {
		return w(gtx)
	}
	return controlface.Cast(gtx, borderedShadow(p, s), s.Focused && !s.Disabled, w)
}

// drawBorderedSymbol renders the platform's bordered control whose label is a
// symbol, drawn through the same internal/controlface the picker's chrome
// trigger is drawn through, with the symbol centred in it.
//
// MEASURED, the toolbar bands of mail-window.png, notes-toolbar.png and
// voicememos-window.png: a control carrying one symbol and nothing else is
// [control.ChromeMarkDp] of mark with [control.ChromeMarkSideDp] clear on each
// side, at the toolbar control's own height. All visual state comes from s; no
// event queries are performed here.
//
// THE VARIANT SETTLES THE SHAPE AND THE MARK'S ROOM. A chrome control is the
// capsule every bordered control in a stored toolbar band is drawn as. A form
// control is the push button's rounded rectangle at the measured
// [tokens.RadiusScale.Md] — save-dialog-{light,dark}.png's "Cancel" fits
// r = 6.11 light and 6.17 dark — and its mark stands in
// [control.FormMarkBox], the band the pop-up on that same sheet leaves its
// own mark, rather than in the band's 24 dp box, which in a 24 dp control
// would leave the symbol no room at all.
func drawBorderedSymbol(gtx layout.Context, icon func(gtx layout.Context, sizePx int, col color.NRGBA), tok resolvedTokens, s RenderState) layout.Dimensions {
	p := tok.platform
	h := min(gtx.Dp(unit.Dp(s.Variant.Height(tok.density))), gtx.Constraints.Max.Y)
	mark, markTop := gtx.Dp(control.ChromeMarkDp), 0
	if s.Variant == Chrome {
		markTop = (h - mark) / 2
	} else {
		mark, markTop = control.FormMarkBox(gtx, h)
	}
	w := mark + 2*gtx.Dp(control.ChromeMarkSideDp)
	w = min(w, gtx.Constraints.Max.X)
	size := image.Pt(w, h)
	box := image.Rectangle{Max: size}

	fill, fg := borderedColors(p, s)
	faceState := controlface.State{
		Hovered:  s.Hovered && !s.Disabled,
		Pressed:  s.Pressed && !s.Disabled,
		Focused:  s.Focused && !s.Disabled,
		Checked:  s.Checked,
		StandsOn: surface.Or(s.Surface, p.SidebarMaterial),
	}
	var outer clip.RRect
	if s.Variant == Chrome {
		outer = controlface.Capsule(gtx, box, p, fill, faceState)
	} else {
		outer = controlface.PushButton(gtx, box, gtx.Dp(unit.Dp(tok.radius.Md)), p, fill, faceState)
	}

	// The symbol centred in the box, clipped to the control's own shape so a
	// mark wider than the control is cut by the control rather than drawn
	// past it.
	if icon != nil && mark > 0 {
		area := outer.Push(gtx.Ops)
		off := op.Offset(image.Pt((w-mark)/2, markTop)).Push(gtx.Ops)
		icon(gtx, mark, fg)
		off.Pop()
		area.Pop()
	}

	if !s.Disabled {
		pointershape.OverSize(gtx.Ops, size, pointer.CursorPointer)
	}
	return layout.Dimensions{Size: size}
}

// borderedColors returns what a bordered control paints inside its own box:
// the fill it carries in its interaction state and the foreground its symbol
// reads in. The shadow the chrome variant casts outside that box is
// [borderedShadow].
//
// Emphasis reaches none of them. There is one bordered control per variant and
// the platform draws each one way, so what varies here is the pointer and the
// switched-off state and nothing else.
//
// Switched off, the control fades toward the surface it stands on at the
// platform's measured coverage, the way every other switched-off control in
// this library does.
func borderedColors(p tokens.PlatformColors, s RenderState) (fill, fg color.NRGBA) {
	if s.Disabled {
		standsOn := surface.Or(s.Surface, p.SidebarMaterial)
		fill = control.Faded(variantRestingFill(p, s.Variant), standsOn)
		return fill, vgcolor.Flatten(p.DisabledControlText, fill)
	}
	fill = variantFill(p, s.Variant, controlState(s))
	return fill, variantForeground(p, s.Variant, fill)
}

// variantFill is the fill a bordered control carries where it stands: the
// toolbar control's own in a chrome region, the push button's in a form.
func variantFill(p tokens.PlatformColors, v Variant, st tokens.State) color.NRGBA {
	if v == Chrome {
		return controlface.Fill(p, st)
	}
	return controlface.FormFill(p, st)
}

// variantRestingFill is that fill at rest — what a switched-off control fades
// from toward the surface it stands on.
func variantRestingFill(p tokens.PlatformColors, v Variant) color.NRGBA {
	return variantFill(p, v, tokens.StateNormal)
}

// variantForeground is the colour a bordered control's mark and its wording
// read in where it stands: the toolbar's own label in a chrome region, the
// platform's control text in a form.
func variantForeground(p tokens.PlatformColors, v Variant, fill color.NRGBA) color.NRGBA {
	if v == Chrome {
		return controlface.Mark(p, fill)
	}
	return controlface.FormForeground(p, fill)
}

// borderedShadow is the shadow the chrome variant casts on its band — the
// platform's own coverage with the reach and the offset its appearance
// measures — faded with the control while it is switched off: a control that
// is not offering itself does not stand off its band as one that is.
//
// A form control casts none — see [BorderedShadow] — so this is not reached
// there.
func borderedShadow(p tokens.PlatformColors, s RenderState) tokens.DropShadow {
	sh := controlface.Shadow(p)
	if s.Disabled {
		return control.FadedShadow(sh)
	}
	return sh
}

// controlState is the token vocabulary's name for the interaction the control
// is in. Press wins over hover, because a pressed control is under the pointer
// by definition and the deeper walk is the one that has something to say.
func controlState(s RenderState) tokens.State {
	switch {
	case s.Pressed:
		return tokens.StatePressed
	case s.Hovered:
		return tokens.StateHover
	case s.Focused:
		return tokens.StateFocus
	}
	return tokens.StateNormal
}

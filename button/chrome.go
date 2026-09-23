package button

import (
	"image"
	"image/color"

	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"

	vgcolor "github.com/vibrantgio/theme/color"
	"github.com/vibrantgio/theme/tokens"

	"github.com/vibrantgio/components/internal/control"
	"github.com/vibrantgio/components/internal/surface"
	"github.com/vibrantgio/components/internal/toolbarface"
	"github.com/vibrantgio/components/pointershape"
)

// RenderChrome produces a layout.Widget for a button standing in a chrome
// region whose label is a symbol: the platform's bordered toolbar control,
// drawn in an explicit visual state without event processing or rx machinery.
//
// The glyph is drawn by icon into a square the control's place names, centred
// in a shape that place names too — the band's capsule at
// d.ToolbarControlHeight, the body's push button at d.ControlHeight and
// rad.Md — and that shape is the pointer target. It takes no text style:
// there is no text to shape. Intended for golden-image testing and static
// rendering; production code uses Button with Props.Variant set to [Chrome]
// and Props.Icon set, or — where the caller owns the control's
// widget.Clickable — [ChromeShadow] around [ChromeFace].
func RenderChrome(
	icon func(gtx layout.Context, sizePx int, col color.NRGBA),
	p tokens.PlatformColors,
	rad tokens.RadiusScale,
	d tokens.Density,
	s RenderState,
) layout.Widget {
	face := ChromeFace(icon, p, rad, d, s)
	return func(gtx layout.Context) layout.Dimensions {
		return ChromeShadow(gtx, p, s, face)
	}
}

// ChromeFace draws the bordered toolbar control's own box and the symbol in
// it, and nothing outside that box.
//
// It is separate from the shadow because gioui.org/widget's Clickable clips
// whatever it wraps to the box its layout.Widget reports, and the shadow falls
// outside the box: a caller that owns the control's Clickable lays this out
// INSIDE it and calls [ChromeShadow] AROUND it. [RenderChrome] is the two
// composed for a caller that owns neither.
func ChromeFace(
	icon func(gtx layout.Context, sizePx int, col color.NRGBA),
	p tokens.PlatformColors,
	rad tokens.RadiusScale,
	d tokens.Density,
	s RenderState,
) layout.Widget {
	tok := resolvedTokens{platform: p, radius: rad, density: d}
	s.Variant = Chrome
	return func(gtx layout.Context) layout.Dimensions {
		return drawChromeIcon(gtx, icon, tok, s)
	}
}

// ChromeShadow lays w out and paints under it the drop shadow the bordered
// toolbar control w drew casts on the band it stands on, sized to the box w
// reported and at the coverage s carries — the platform's own, faded with the
// control while it is switched off.
//
// w is [ChromeFace] or something wrapping it, a widget.Clickable included.
// A control whose [RenderState.Place] is [Body] casts none: the shadow was
// measured on a band and belongs to the chrome variant, so this lays w out
// and paints nothing under it.
func ChromeShadow(gtx layout.Context, p tokens.PlatformColors, s RenderState, w layout.Widget) layout.Dimensions {
	if s.Place == Body {
		return w(gtx)
	}
	return toolbarface.Cast(gtx, chromeShadow(p, s), s.Focused && !s.Disabled, w)
}

// drawChromeIcon renders a chrome button whose label is a symbol: the
// platform's bordered toolbar control, drawn through the same
// internal/toolbarface the picker's chrome trigger is drawn through, with the
// symbol centred in it.
//
// MEASURED, the toolbar bands of mail-window.png, notes-toolbar.png and
// voicememos-window.png: a control carrying one symbol and nothing else is
// [control.ChromeMarkDp] of mark with [control.ChromeMarkSideDp] clear on each
// side, at the toolbar control's own height. All visual state comes from s; no
// event queries are performed here.
//
// THE PLACE SETTLES THE SHAPE AND THE MARK'S ROOM. A band control is the
// capsule every bordered control in a stored toolbar band is drawn as. A body
// control is the form's own shape, the push button's rounded rectangle at the
// measured [tokens.RadiusScale.Md] — save-dialog-{light,dark}.png's "Cancel"
// fits r = 6.11 light and 6.17 dark — and its mark stands in
// [control.BodyMarkBox], the band the pop-up on that same sheet leaves its
// own mark, rather than in the band's 24 dp box, which in a 24 dp control
// would leave the symbol no room at all.
func drawChromeIcon(gtx layout.Context, icon func(gtx layout.Context, sizePx int, col color.NRGBA), tok resolvedTokens, s RenderState) layout.Dimensions {
	p := tok.platform
	h := min(gtx.Dp(unit.Dp(s.Place.Height(tok.density))), gtx.Constraints.Max.Y)
	mark, markTop := gtx.Dp(control.ChromeMarkDp), 0
	radius := h / 2
	if s.Place == Body {
		mark, markTop = control.BodyMarkBox(gtx, h)
		radius = gtx.Dp(unit.Dp(tok.radius.Md))
	} else {
		markTop = (h - mark) / 2
	}
	w := mark + 2*gtx.Dp(control.ChromeMarkSideDp)
	w = min(w, gtx.Constraints.Max.X)
	size := image.Pt(w, h)
	box := image.Rectangle{Max: size}

	fill, fg := chromeColors(p, s)
	outer := toolbarface.Capsule(gtx, box, radius, p, fill, toolbarface.State{
		Hovered:  s.Hovered && !s.Disabled,
		Pressed:  s.Pressed && !s.Disabled,
		Focused:  s.Focused && !s.Disabled,
		Checked:  s.Checked,
		Form:     s.Place == Body,
		StandsOn: surface.Or(s.Surface, p.SidebarMaterial),
	})

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

// chromeColors returns what the chrome variant paints inside its own box: the
// fill the control carries in its interaction state and the foreground its
// symbol reads in. The shadow it casts outside that box is [chromeShadow].
//
// Emphasis reaches none of them. There is one bordered toolbar control and the
// platform draws it one way, so what varies here is the pointer and the
// switched-off state and nothing else.
//
// Switched off, the control fades toward the band it stands on at the
// platform's measured coverage, the way every other switched-off control in
// this library does.
func chromeColors(p tokens.PlatformColors, s RenderState) (fill, fg color.NRGBA) {
	if s.Disabled {
		standsOn := surface.Or(s.Surface, p.SidebarMaterial)
		fill = control.Faded(placeRestingFill(p, s.Place), standsOn)
		return fill, vgcolor.Flatten(p.DisabledControlText, fill)
	}
	fill = placeFill(p, s.Place, chromeState(s))
	return fill, placeForeground(p, s.Place, fill)
}

// placeFill is the fill a bordered control carries where it stands: the
// toolbar control's own on a band, the push button's in a body.
func placeFill(p tokens.PlatformColors, pl Place, st tokens.State) color.NRGBA {
	if pl == Body {
		return toolbarface.FormFill(p, st)
	}
	return toolbarface.Fill(p, st)
}

// placeRestingFill is that fill at rest — what a switched-off control fades
// from toward the surface it stands on.
func placeRestingFill(p tokens.PlatformColors, pl Place) color.NRGBA {
	return placeFill(p, pl, tokens.StateNormal)
}

// placeForeground is the colour a bordered control's mark and its wording
// read in where it stands: the toolbar's own label on a band, the platform's
// control text in a body.
func placeForeground(p tokens.PlatformColors, pl Place, fill color.NRGBA) color.NRGBA {
	if pl == Body {
		return toolbarface.FormForeground(p, fill)
	}
	return toolbarface.Mark(p, fill)
}

// chromeShadow is the shadow the control casts on its band — the platform's
// own coverage with the reach and the offset its appearance measures — faded
// with the control while it is switched off: a control that is not offering
// itself does not stand off its band as one that is.
//
// A body control casts none — see [ChromeShadow] — so this is not reached
// there.
func chromeShadow(p tokens.PlatformColors, s RenderState) tokens.DropShadow {
	sh := toolbarface.Shadow(p)
	if s.Disabled {
		return control.FadedShadow(sh)
	}
	return sh
}

// chromeState is the token vocabulary's name for the interaction the control
// is in. Press wins over hover, because a pressed control is under the pointer
// by definition and the deeper walk is the one that has something to say.
func chromeState(s RenderState) tokens.State {
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

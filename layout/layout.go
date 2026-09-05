package layout

import (
	"image"

	gio "gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/unit"
)

// Inset returns a uniform inset with dp device-independent pixels on all sides.
func Inset(dp float32) gio.Inset {
	return gio.UniformInset(unit.Dp(dp))
}

// InsetXY returns an inset with h horizontal (left+right) and v vertical
// (top+bottom) device-independent padding.
func InsetXY(h, v float32) gio.Inset {
	return gio.Inset{Left: unit.Dp(h), Right: unit.Dp(h), Top: unit.Dp(v), Bottom: unit.Dp(v)}
}

// HSpacer returns a layout.Widget that occupies dp device-independent pixels horizontally
// and no vertical space. Use as a gap between children inside a Row.
func HSpacer(dp float32) gio.Widget {
	s := gio.Spacer{Width: unit.Dp(dp)}
	return func(gtx gio.Context) gio.Dimensions {
		return s.Layout(gtx)
	}
}

// VSpacer returns a layout.Widget that occupies dp device-independent pixels vertically
// and no horizontal space. Use as a gap between children inside a Col.
func VSpacer(dp float32) gio.Widget {
	s := gio.Spacer{Height: unit.Dp(dp)}
	return func(gtx gio.Context) gio.Dimensions {
		return s.Layout(gtx)
	}
}

// Pill returns a rounded-rect clip op whose corner radius is clamped to
// min(w,h)/2. clip.RRect does not clamp corner radii to the rect, so a
// token.Radius.Full sentinel (9999 dp) passed directly to clip.RRect sprays
// paint across the entire frame. Pill centralises the clamp so callers
// cannot reintroduce that bug.
func Pill(ops *op.Ops, rect image.Rectangle, rad int) clip.Op {
	if maxRad := min(rect.Dx(), rect.Dy()) / 2; rad > maxRad {
		rad = maxRad
	}
	return clip.RRect{Rect: rect, SE: rad, SW: rad, NE: rad, NW: rad}.Op(ops)
}

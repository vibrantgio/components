package control

import (
	"image"
	"image/color"

	"gioui.org/f32"
	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"

	vgcolor "github.com/vibrantgio/theme/color"
)

// The drop shadow a BORDERED TOOLBAR CONTROL casts on the band it stands in.
//
// It is what tells a light control from its band: the platform draws no edge
// there and fills the control with the band's own white, so the shadow is the
// whole of the step. Dark it is one 255th deep and the control is told by its
// fill and its rim instead — one drawing, and the coverage the appearance
// carries is what makes it show or not.
//
// MEASURED, finder-window-light.png, the view pop-up at x 694-742, y 8-43 on a
// band flat at #ffffff. The band reads 244 in the row under the control, 250
// in the row over it and 248 in the columns beside it, and it recovers to
// #ffffff 39 rows below, 31 columns beside and — the window's own top edge
// cutting the reading off at 8 rows, where it still stands two 255ths down —
// about 15 rows above. So the shadow is not symmetric about the control: it is
// heavier and longer below it than above, which is one rectangle sunk below
// the control and not a second shadow.
//
// [ToolbarShadowReachDp] and [ToolbarShadowOffsetDp] are that rectangle fitted
// to the reading, with the peak in tokens.PlatformColors.ToolbarControlShadow.
//
// The ramp is the one effects/depth draws for a floating surface — a linear
// falloff from the peak at the shape's edge to nothing at the reach, in eight
// gradient tiles around one interior fill — and it is drawn here rather than
// called there because the module graph runs the other way: effects imports
// components.
const (
	// ToolbarShadowReachDp is how far the shadow carries past the rectangle
	// that casts it.
	ToolbarShadowReachDp unit.Dp = 23

	// ToolbarShadowOffsetDp is how far below the control that rectangle
	// sits, which is what makes the shadow heavier under the control than
	// over it.
	ToolbarShadowOffsetDp unit.Dp = 9
)

// bezierCircle is the cubic-Bézier control-point ratio that best approximates
// a quarter circle: 4/3·(√2−1).
const bezierCircle = 0.55228475

// DrawToolbarShadow paints the shadow the control occupying bounds casts,
// in shadow at its own coverage at the sunk rectangle's edge and falling
// linearly to nothing [ToolbarShadowReachDp] away.
//
// radius rounds the shadow's corners, in pixels: a caller passes the radius it
// rounds its own fill to, so the interior cannot show through the rounding as
// square wedges. The caller paints its fill over this, which is what covers
// the interior.
//
// A zero coverage, or a reach that rounds to nothing at the current metric,
// paints nothing.
func DrawToolbarShadow(gtx layout.Context, bounds image.Rectangle, radius int, shadow color.NRGBA) {
	extent := gtx.Dp(ToolbarShadowReachDp)
	if extent <= 0 || shadow.A == 0 {
		return
	}

	shadowBounds := bounds.Add(image.Pt(0, gtx.Dp(ToolbarShadowOffsetDp)))
	if radius < 0 {
		radius = 0
	}
	if m := min(shadowBounds.Dx(), shadowBounds.Dy()) / 2; radius > m {
		radius = m
	}

	// Gio blends a coverage it is handed in linear light where the platform
	// blends the encoded byte, so the ramp goes over at the coverage fitted
	// to Gio's blend — the same correction effects/depth spends, and at a
	// coverage this low it lands the platform's byte to within one 255th.
	inner := shadow
	inner.A = vgcolor.LinearCoverage(shadow.A)
	outer := color.NRGBA{R: shadow.R, G: shadow.G, B: shadow.B}

	if radius == 0 {
		paint.FillShape(gtx.Ops, inner, clip.Rect(shadowBounds).Op())
	} else {
		paint.FillShape(gtx.Ops, inner, clip.RRect{
			Rect: shadowBounds,
			SE:   radius, SW: radius, NE: radius, NW: radius,
		}.Op(gtx.Ops))
	}

	r := extent
	rho := radius
	bMin, bMax := shadowBounds.Min, shadowBounds.Max

	// Edge bands, shortened by the corner radius so the corner tiles own the
	// rounded ends. The inner stop is flush with the rectangle's edge, which
	// is where the interior fill leaves off; the outer stop is the reach.
	shadowTile(gtx,
		image.Rect(bMin.X+rho, bMin.Y-r, bMax.X-rho, bMin.Y),
		f32.Pt(0, float32(bMin.Y-r)), outer,
		f32.Pt(0, float32(bMin.Y)), inner,
	)
	shadowTile(gtx,
		image.Rect(bMin.X+rho, bMax.Y, bMax.X-rho, bMax.Y+r),
		f32.Pt(0, float32(bMax.Y)), inner,
		f32.Pt(0, float32(bMax.Y+r)), outer,
	)
	shadowTile(gtx,
		image.Rect(bMin.X-r, bMin.Y+rho, bMin.X, bMax.Y-rho),
		f32.Pt(float32(bMin.X-r), 0), outer,
		f32.Pt(float32(bMin.X), 0), inner,
	)
	shadowTile(gtx,
		image.Rect(bMax.X, bMin.Y+rho, bMax.X+r, bMax.Y-rho),
		f32.Pt(float32(bMax.X), 0), inner,
		f32.Pt(float32(bMax.X+r), 0), outer,
	)

	shadowCorner(gtx, image.Pt(bMin.X, bMin.Y), -1, -1, rho, r, inner, outer)
	shadowCorner(gtx, image.Pt(bMax.X, bMin.Y), +1, -1, rho, r, inner, outer)
	shadowCorner(gtx, image.Pt(bMin.X, bMax.Y), -1, +1, rho, r, inner, outer)
	shadowCorner(gtx, image.Pt(bMax.X, bMax.Y), +1, +1, rho, r, inner, outer)
}

// shadowCorner fills one corner of the penumbra. corner is the shadow
// rectangle's square corner point and sx, sy its outward direction (±1 each).
// With rho > 0 the tile grows inward to the (rho+r)-sided square anchored at
// the corner circle's centre, minus the quarter disc the rounded interior
// already painted, so the ramp has no transparent bite at each rounded corner.
//
// The diagonal gradient's inner stop sits rho/2 inside the square corner and
// its outer stop half the reach outside it, which is what the seam constraint
// against both adjacent (shortened) edge bands solves to.
func shadowCorner(gtx layout.Context, corner image.Point, sx, sy, rho, r int, inner, outer color.NRGBA) {
	h := float32(r) / 2
	fx, fy := float32(sx), float32(sy)
	kx, ky := float32(corner.X), float32(corner.Y)
	s1 := f32.Pt(kx-fx*float32(rho)/2, ky-fy*float32(rho)/2)
	s2 := f32.Pt(s1.X+fx*h, s1.Y+fy*h)

	if rho == 0 {
		shadowTile(gtx,
			image.Rectangle{Min: corner, Max: corner.Add(image.Pt(sx*r, sy*r))}.Canon(),
			s1, inner, s2, outer,
		)
		return
	}

	rf := float32(rho)
	ext := rf + float32(r)
	c := f32.Pt(kx-fx*rf, ky-fy*rf) // the corner circle's centre
	k := bezierCircle * rf

	var p clip.Path
	p.Begin(gtx.Ops)
	p1 := f32.Pt(c.X+fx*rf, c.Y) // the arc's end on the horizontal axis through the centre
	p5 := f32.Pt(c.X, c.Y+fy*rf) // the arc's end on the vertical axis through the centre
	p.MoveTo(p1)
	p.LineTo(f32.Pt(c.X+fx*ext, c.Y))
	p.LineTo(f32.Pt(c.X+fx*ext, c.Y+fy*ext))
	p.LineTo(f32.Pt(c.X, c.Y+fy*ext))
	p.LineTo(p5)
	p.CubeTo(
		f32.Pt(p5.X+fx*k, p5.Y),
		f32.Pt(p1.X, p1.Y+fy*k),
		p1,
	)
	p.Close()
	defer clip.Outline{Path: p.End()}.Op().Push(gtx.Ops).Pop()
	paint.LinearGradientOp{Stop1: s1, Color1: inner, Stop2: s2, Color2: outer}.Add(gtx.Ops)
	paint.PaintOp{}.Add(gtx.Ops)
}

func shadowTile(gtx layout.Context, rect image.Rectangle, stop1 f32.Point, c1 color.NRGBA, stop2 f32.Point, c2 color.NRGBA) {
	if rect.Dx() <= 0 || rect.Dy() <= 0 {
		return
	}
	defer clip.Rect(rect).Push(gtx.Ops).Pop()
	paint.LinearGradientOp{Stop1: stop1, Color1: c1, Stop2: stop2, Color2: c2}.Add(gtx.Ops)
	paint.PaintOp{}.Add(gtx.Ops)
}

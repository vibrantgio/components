package control

import (
	"image"
	"image/color"

	"gioui.org/f32"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"

	vgcolor "github.com/vibrantgio/theme/color"
	"github.com/vibrantgio/theme/tokens"
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
// tokens.PlatformColors.ToolbarControlShadow carries that rectangle fitted to
// the reading, per appearance: the peak with the reach and the offset it was
// fitted at, which is why this package owns no geometry of its own.
//
// The ramp is the one effects/depth draws for a floating surface — a linear
// falloff from the peak at the shape's edge to nothing at the reach, in eight
// gradient tiles around one interior fill — and it is drawn here rather than
// called there because the module graph runs the other way: effects imports
// components.

// ToolbarShadowOf is the shadow a bordered toolbar control casts under p's
// appearance: the reading p carries, whole.
//
// Which reading answers is the platform's own behaviour and not an appearance
// this code tests for. Where the platform gives the control an edge — the
// hairline [ToolbarControlRim] answers a colour for — the control is told
// from its band by that edge and its fill, and the shadow measures a hint
// sunk under it. Where it gives none, the control's fill IS the band's own
// white and the shadow is the whole of the step, so it carries far.
func ToolbarShadowOf(p tokens.PlatformColors) tokens.DropShadow {
	return p.ToolbarControlShadow
}

// FadedShadow is the shadow a switched-off control casts: the same geometry
// at the platform's measured disabled coverage. A control that is not
// offering itself does not stand off its band as one that is.
func FadedShadow(s tokens.DropShadow) tokens.DropShadow {
	return s.WithPeak(vgcolor.Fade(s.Peak, tokens.DisabledCoverage))
}

// DrawFloatingShadow paints the shadow a floating surface casts around
// bounds: the reading tokens.PlatformColors.FloatingShadow carries, at its own
// coverage at the surface's edge and falling linearly to nothing its reach
// away.
//
// It takes the same ramp [DrawToolbarShadow] draws, because there is one ramp
// and a [tokens.DropShadow] is the value it is handed. What differs is the
// offset the reading carries: a floating surface's shadow is centred on it,
// where a bordered toolbar control's is sunk below it, which is what those
// captures measure.
//
// radius rounds the shadow's corners, in pixels: a caller passes the radius it
// rounds its own fill to, so the interior cannot show through the rounding as
// square wedges, and 0 keeps the square geometry.
func DrawFloatingShadow(gtx layout.Context, bounds image.Rectangle, radius int, p tokens.PlatformColors) {
	DrawToolbarShadow(gtx, bounds, radius, p.FloatingShadow)
}

// bezierCircle is the cubic-Bézier control-point ratio that best approximates
// a quarter circle: 4/3·(√2−1).
const bezierCircle = 0.55228475

// DrawToolbarShadow paints the shadow the control occupying bounds casts: sh
// at its own coverage at the sunk rectangle's edge, falling linearly to
// nothing sh.Reach away.
//
// radius rounds the shadow's corners, in pixels: a caller passes the radius it
// rounds its own fill to, so the interior cannot show through the rounding as
// square wedges. The caller paints its fill over this, which is what covers
// the interior.
//
// A zero coverage, or a reach that rounds to nothing at the current metric,
// paints nothing.
func DrawToolbarShadow(gtx layout.Context, bounds image.Rectangle, radius int, sh tokens.DropShadow) {
	extent := gtx.Dp(unit.Dp(sh.Reach))
	if extent <= 0 || sh.Peak.A == 0 {
		return
	}

	shadowBounds := bounds.Add(image.Pt(0, gtx.Dp(unit.Dp(sh.Offset))))
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
	inner := sh.Peak
	inner.A = vgcolor.LinearCoverage(sh.Peak.A)
	outer := color.NRGBA{R: sh.Peak.R, G: sh.Peak.G, B: sh.Peak.B}

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

// DrawToolbarShadowAround paints the same shadow as [DrawToolbarShadow] with
// cut removed from it, for a caller painting the shadow AFTER the control
// rather than under it.
//
// A band paints its controls' shadows in one pass after the window's columns
// have laid out — otherwise a shadow is covered over one column and left
// standing over another — and by then the control is already drawn. The
// cut-out is what makes the later pass land what the earlier one landed: the
// control's fill is opaque, so a shadow under it and a shadow with its box
// removed are the same image. What the two differ by is the antialiased ring
// at the control's own edge, where the coverage is spent twice: at the
// steepest phase, half a pixel, the difference is the shadow's peak coverage
// times a quarter of the step from the band to the fill — under one 255th at
// either appearance's measured numbers.
//
// cut is the control's own box at rest, rounded to cutRadius. While the
// control is FOCUSED it is the footprint the focus halo covers instead, which
// straddles that box: the halo is painted before this pass and the ramp would
// otherwise run over it, which reads as a shadow lying on the band rather than
// under it.
func DrawToolbarShadowAround(gtx layout.Context, bounds image.Rectangle, radius int, cut image.Rectangle, cutRadius int, sh tokens.DropShadow) {
	extent := gtx.Dp(unit.Dp(sh.Reach))
	if extent <= 0 || sh.Peak.A == 0 {
		return
	}
	if radius < 0 {
		radius = 0
	}
	if m := min(bounds.Dx(), bounds.Dy()) / 2; radius > m {
		radius = m
	}
	if cutRadius < 0 {
		cutRadius = 0
	}
	if m := min(cut.Dx(), cut.Dy()) / 2; cutRadius > m {
		cutRadius = m
	}
	// The whole drawing's extent: the sunk rectangle, the control's own box
	// and whatever is cut out of it, grown by the reach. One pixel of slack
	// keeps the ramp's last column inside the outer contour.
	reachAll := bounds.Union(cut).Union(bounds.Add(image.Pt(0, gtx.Dp(unit.Dp(sh.Offset))))).Inset(-extent - 1)
	defer clip.Outline{Path: ringPath(gtx.Ops, reachAll, cut, cutRadius)}.Op().Push(gtx.Ops).Pop()
	DrawToolbarShadow(gtx, bounds, radius, sh)
}

// ringPath is outer with the rounded rectangle box cut out of it: the outer
// contour wound one way and the inner the other, so the inner is a hole under
// the non-zero winding rule Gio fills outlines by. Corners are the
// quarter-circle Bézier, the same approximation the penumbra's own corners
// are drawn with.
func ringPath(ops *op.Ops, outer, box image.Rectangle, radius int) clip.PathSpec {
	var p clip.Path
	p.Begin(ops)

	// The outer contour, clockwise on a screen whose y runs down.
	p.MoveTo(f32.Pt(float32(outer.Min.X), float32(outer.Min.Y)))
	p.LineTo(f32.Pt(float32(outer.Max.X), float32(outer.Min.Y)))
	p.LineTo(f32.Pt(float32(outer.Max.X), float32(outer.Max.Y)))
	p.LineTo(f32.Pt(float32(outer.Min.X), float32(outer.Max.Y)))
	p.Close()

	// The control's box, counter-clockwise: down the leading side, along the
	// foot, up the trailing side and back across the top.
	x0, y0 := float32(box.Min.X), float32(box.Min.Y)
	x1, y1 := float32(box.Max.X), float32(box.Max.Y)
	r := float32(radius)
	k := bezierCircle * r
	p.MoveTo(f32.Pt(x0, y0+r))
	p.LineTo(f32.Pt(x0, y1-r))
	p.CubeTo(f32.Pt(x0, y1-r+k), f32.Pt(x0+r-k, y1), f32.Pt(x0+r, y1))
	p.LineTo(f32.Pt(x1-r, y1))
	p.CubeTo(f32.Pt(x1-r+k, y1), f32.Pt(x1, y1-r+k), f32.Pt(x1, y1-r))
	p.LineTo(f32.Pt(x1, y0+r))
	p.CubeTo(f32.Pt(x1, y0+r-k), f32.Pt(x1-r+k, y0), f32.Pt(x1-r, y0))
	p.LineTo(f32.Pt(x0+r, y0))
	p.CubeTo(f32.Pt(x0+r-k, y0), f32.Pt(x0, y0+r-k), f32.Pt(x0, y0+r))
	p.Close()

	return p.End()
}

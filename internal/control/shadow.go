package control

import (
	"image"
	"image/color"
	"math"

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
// The ramp is a linear falloff from the peak at the shape's edge to nothing at
// the reach. effects/depth draws a falloff of the same kind for a floating
// surface. This package draws its own because the module graph runs the other
// way: effects imports components.

// ToolbarShadowRiseDp bounds how far ABOVE a bordered toolbar control its
// shadow is drawn. The fitted rectangle carries its ramp 14 px up, which no
// capture holds: the band gives the control 8 px above it and the window's
// own top edge cuts the reading there.
//
// MEASURED at 1x on four controls in two light captures, the band flat at
// #ffffff. finder-window-light.png, the view pop-up at x 694-742, y 8-43,
// and mail-three-column-light.png, three of its bordered controls at y 8-43,
// all read the same run above the control's top edge, 255ths of the band by
// distance: 5, 5, 4, 4, 4, 3, 3, 2 — the eighth row being the window's own
// top row, where the band still stands two 255ths down. Below those same
// controls both captures read 11, 11, 10, 10, 10, 9, 9, 8, 8, 8, 7, 7, 6, 6,
// 6, 5, 5, 5, 4, 4, 4, 3, 3, 3, 3, 2 and recover to #ffffff 39 rows down, so
// the shadow carries onto whatever lies under the band and is bounded only
// there.
//
// The ramp is cut at this distance rather than carried to the reach, which
// leaves a step of the two 255ths the eighth row measures.
const ToolbarShadowRiseDp unit.Dp = 8

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

	r := extent
	rho := radius
	bMin, bMax := shadowBounds.Min, shadowBounds.Max

	// The interior less its four corner squares, as rectangles on whole
	// pixels. The corner fans paint the quarter discs, so no curve has to
	// meet another curve.
	fillRect(gtx, image.Rect(bMin.X+rho, bMin.Y, bMax.X-rho, bMax.Y), inner)
	fillRect(gtx, image.Rect(bMin.X, bMin.Y+rho, bMin.X+rho, bMax.Y-rho), inner)
	fillRect(gtx, image.Rect(bMax.X-rho, bMin.Y+rho, bMax.X, bMax.Y-rho), inner)

	// Edge bands, shortened by the corner radius so the corner fans own the
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

// cornerFanError bounds, in pixels, how far a corner fan's distance falls
// short of the straight-line distance from the corner circle's centre. At the
// light toolbar shadow's peak and reach, 0.1 px is 0.04 of a 255th.
const cornerFanError = 0.1

// shadowCorner fills one corner of the penumbra and the quarter disc of the
// interior inside it. corner is the shadow rectangle's square corner point
// and sx, sy its outward direction (±1 each).
//
// The darkening falls with the straight-line distance from the corner
// circle's centre. So a line of equal darkening is an arc on that centre,
// which is the shape the stored captures measure at a toolbar control's
// corners.
//
// Gio has no radial gradient, so a fan of wedges draws the arcs. Each wedge is
// a triangle from the centre, with a linear gradient along its bisector from
// the peak at rho to nothing at rho+r. A pixel at distance d, an angle φ off
// the bisector, gets the distance d·cos φ. The fan has enough wedges to keep
// d·(1−cos φ) under cornerFanError out to rho+r. Inside rho the gradient holds
// the peak, which paints the quarter disc.
func shadowCorner(gtx layout.Context, corner image.Point, sx, sy, rho, r int, inner, outer color.NRGBA) {
	ext := float64(rho + r)
	n := int(math.Ceil(math.Pi / 4 / math.Acos(1-cornerFanError/ext)))
	step := math.Pi / 2 / float64(n)
	// The chord of each triangle lies ext from the centre at its nearest
	// point, so the fan covers the whole quarter disc of radius ext.
	far := ext / math.Cos(step/2)
	fx, fy := float64(sx), float64(sy)
	cx, cy := float64(corner.X-sx*rho), float64(corner.Y-sy*rho)
	at := func(a, d float64) f32.Point {
		return f32.Pt(float32(cx+fx*d*math.Cos(a)), float32(cy+fy*d*math.Sin(a)))
	}
	for i := range n {
		a0, a1 := float64(i)*step, float64(i+1)*step
		mid := (a0 + a1) / 2

		var p clip.Path
		p.Begin(gtx.Ops)
		p.MoveTo(at(0, 0))
		p.LineTo(at(a0, far))
		p.LineTo(at(a1, far))
		p.Close()
		st := clip.Outline{Path: p.End()}.Op().Push(gtx.Ops)
		paint.LinearGradientOp{
			Stop1: at(mid, float64(rho)), Color1: inner,
			Stop2: at(mid, ext), Color2: outer,
		}.Add(gtx.Ops)
		paint.PaintOp{}.Add(gtx.Ops)
		st.Pop()
	}
}

func fillRect(gtx layout.Context, rect image.Rectangle, c color.NRGBA) {
	if rect.Dx() <= 0 || rect.Dy() <= 0 {
		return
	}
	paint.FillShape(gtx.Ops, c, clip.Rect(rect).Op())
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
// control is FOCUSED it is the footprint the focus ring covers instead, which
// straddles that box: the ring is painted before this pass and the ramp would
// otherwise run over it, which reads as a shadow lying on the band rather than
// under it.
//
// The drawing is bounded above at [ToolbarShadowRiseDp] from the control's own
// top edge — the rows the stored captures close — so the ramp reaches the
// toolbar's top edge and no further. Below it is bounded by the reach alone:
// the platform's shadow carries onto whatever lies under the band, which
// finder-window-light.png reads four 255ths down fifteen rows past the band's
// own foot.
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
	// Bounded above at the measured rise, never inside whatever the cut
	// itself reaches: a hole lying outside the outer contour would be wound
	// into the fill rather than out of it.
	if top := min(bounds.Min.Y-gtx.Dp(ToolbarShadowRiseDp), cut.Min.Y); reachAll.Min.Y < top {
		reachAll.Min.Y = top
	}
	defer clip.Outline{Path: ringPath(gtx.Ops, reachAll, cut, cutRadius)}.Op().Push(gtx.Ops).Pop()
	DrawToolbarShadow(gtx, bounds, radius, sh)
}

// ringPath is outer with the rounded rectangle box cut out of it: the outer
// contour wound one way and the inner the other, so the inner is a hole under
// the non-zero winding rule Gio fills outlines by. Corners are the
// quarter-circle Bézier.
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

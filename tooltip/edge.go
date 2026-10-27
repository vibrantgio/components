package tooltip

import (
	"image"
	"image/color"
	"math"
	"sync"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"

	"github.com/vibrantgio/theme/tokens"
)

// casting holds every value the image of a surface's edge and shadow
// depends on. It is comparable, so it is the cache key of that image.
type casting struct {
	size    image.Point // the surface's bounds, in px
	radius  int         // the surface's corner radius, in px
	edgePx  int         // the edge's width and each rim pixel's width, in px
	offset  int         // how far below the bounds the shadow's rectangle is, in px
	scale   float32     // px per dp: the curve's index is a distance in dp
	edge    tokens.Edge
	shadow  tokens.ShadowCurve
	surface color.NRGBA // the opaque fill the tooltip stands on
}

// drawEdgeAndShadow paints the help tag's edge and shadow around the
// surface bounds at the origin, size wide and tall, before the surface's
// own fill. It paints nothing inside the bounds' fill.
func drawEdgeAndShadow(gtx layout.Context, size image.Point, radius int, p tokens.PlatformColors, stands color.NRGBA) {
	c := casting{
		size:    size,
		radius:  radius,
		edgePx:  max(gtx.Dp(1), 1),
		offset:  gtx.Dp(unit.Dp(p.HelpTagShadow.Offset)),
		scale:   gtx.Metric.PxPerDp,
		edge:    p.HelpTagEdge,
		shadow:  p.HelpTagShadow,
		surface: stands,
	}
	if c.scale <= 0 {
		c.scale = 1
	}
	iop, margin := c.image()
	defer op.Offset(image.Pt(-margin, -margin)).Push(gtx.Ops).Pop()
	defer clip.Rect{Max: size.Add(image.Pt(2*margin, 2*margin))}.Push(gtx.Ops).Pop()
	iop.Add(gtx.Ops)
	paint.PaintOp{}.Add(gtx.Ops)
}

// casts keeps the images already built. A tooltip draws the same image on
// every frame it stands, and a new ImageOp uploads its image again.
var casts = struct {
	sync.Mutex
	m map[casting]castImage
}{m: map[casting]castImage{}}

type castImage struct {
	op     paint.ImageOp
	margin int
}

// image returns the image op of c and the margin the image extends past
// the bounds on each side.
func (c casting) image() (paint.ImageOp, int) {
	casts.Lock()
	defer casts.Unlock()
	if ci, ok := casts.m[c]; ok {
		return ci.op, ci.margin
	}
	if len(casts.m) >= 16 {
		clear(casts.m)
	}
	img, margin := c.draw()
	iop := paint.NewImageOp(img)
	iop.Filter = paint.FilterNearest
	casts.m[c] = castImage{op: iop, margin: margin}
	return iop, margin
}

// draw builds the image: black on every pixel, at the coverage that lands
// the platform's pixel on c.surface.
//
// The platform composites each layer in encoded sRGB. Gio blends in linear
// light, so a coverage handed to Gio as it is lands a lighter pixel. Each
// pixel's coverage is therefore the one whose linear blend over c.surface
// lands the encoded composite of shadow and edge, to half a 255th. Over
// another fill the pixel is a darkening of that fill, not that byte.
func (c casting) draw() (*image.NRGBA, int) {
	reach := int(math.Ceil(float64(c.shadow.Reach()) * float64(c.scale)))
	margin := max(c.edgePx, reach) + c.offset
	img := image.NewNRGBA(image.Rectangle{Max: c.size.Add(image.Pt(2*margin, 2*margin))})
	fit := newCoverageFit(c.surface)
	for y := -margin; y < c.size.Y+margin; y++ {
		for x := -margin; x < c.size.X+margin; x++ {
			fill, ring := c.cover(x, y)
			if fill >= 1 {
				continue
			}
			s := c.shadowAt(x, y)
			e := float64(c.edgeAt(x, y)) / 255
			// Coverage of the pixel's part outside the fill: the edge over
			// the shadow on the ring, the shadow alone outside it.
			out := 1 - fill - ring
			cov := (ring*(1-(1-e)*(1-s)) + out*s) / (1 - fill)
			if cov <= 0 {
				continue
			}
			img.SetNRGBA(x+margin, y+margin, color.NRGBA{A: fit.coverage(cov)})
		}
	}
	return img, margin
}

// cover returns the part of the pixel at (x, y) inside the rounded bounds,
// and the part inside the edge's ring around them. With no radius both are
// whole pixels.
func (c casting) cover(x, y int) (fill, ring float64) {
	b := image.Rectangle{Max: c.size}
	outer := b.Inset(-c.edgePx)
	if c.radius == 0 {
		p := image.Pt(x, y)
		switch {
		case p.In(b):
			return 1, 0
		case p.In(outer):
			return 0, 1
		}
		return 0, 0
	}
	const n = 4
	in, edge := 0, 0
	for j := range n {
		for i := range n {
			px := float64(x) + (float64(i)+0.5)/n
			py := float64(y) + (float64(j)+0.5)/n
			switch {
			case inRRect(px, py, b, c.radius):
				in++
			case inRRect(px, py, outer, c.radius+c.edgePx):
				edge++
			}
		}
	}
	return float64(in) / (n * n), float64(edge) / (n * n)
}

// inRRect reports whether the point (px, py) is inside r with corners of
// radius rad.
func inRRect(px, py float64, r image.Rectangle, rad int) bool {
	if px < float64(r.Min.X) || px >= float64(r.Max.X) || py < float64(r.Min.Y) || py >= float64(r.Max.Y) {
		return false
	}
	f := float64(rad)
	cx := math.Max(float64(r.Min.X)+f, math.Min(px, float64(r.Max.X)-f))
	cy := math.Max(float64(r.Min.Y)+f, math.Min(py, float64(r.Max.Y)-f))
	dx, dy := px-cx, py-cy
	return dx*dx+dy*dy <= f*f
}

// edgeAt returns the edge's coverage at a pixel of its ring, out of 255.
// With no radius the corner pixels take the measured corner value. With a
// radius the ring's own coverage at the rounding makes its corners lighter,
// so every ring pixel takes the side value.
func (c casting) edgeAt(x, y int) uint8 {
	if c.radius == 0 {
		cornerX := x < 0 || x >= c.size.X
		cornerY := y < 0 || y >= c.size.Y
		if cornerX && cornerY {
			return c.edge.Corner.A
		}
	}
	return c.edge.Side.A
}

// shadowAt returns the shadow's coverage at the pixel (x, y), out of 1.
//
// The shadow is cast from the bounds moved c.offset down. The measured
// corner rule, Alpha[dx]·Alpha[dy]/(2·Alpha[0]), makes the shadow a product
// of one fraction per axis, of the peak 2·Alpha[0]. Each fraction is
// Alpha[d]/(2·Alpha[0]) at d pixels out from the rectangle's edge pixel. A
// pixel d inside that edge pixel takes 1 minus the fraction d out, the
// symmetry of a blurred edge. Far inside it is 1, so a side is the curve
// itself. One inside gives 2·Alpha[0] − Alpha[1]: light 33 and dark 41,
// which the reference's fit gives under the bottom edge, 32.9 and 41.0.
func (c casting) shadowAt(x, y int) float64 {
	a0 := float64(c.shadow.Alpha[0].A)
	if a0 == 0 {
		return 0
	}
	top, bottom := c.offset, c.size.Y-1+c.offset
	dx := max(-x, x-(c.size.X-1))
	dy := max(top-y, y-bottom)
	return 2 * a0 * c.fraction(dx) * c.fraction(dy) / 255
}

// fraction is the shadow's part of its peak at d pixels out from the
// rectangle's edge pixel along one axis; a negative d is inside it.
func (c casting) fraction(d int) float64 {
	if d < 0 {
		return 1 - c.fraction(-d)
	}
	i := int(float32(d) / c.scale)
	if i >= tokens.ShadowCurveLen {
		return 0
	}
	return float64(c.shadow.Alpha[i].A) / (2 * float64(c.shadow.Alpha[0].A))
}

// coverageFit finds, for one opaque surface, the 8-bit coverage of black
// whose blend in linear light lands what black at a coverage composited in
// encoded sRGB lands.
type coverageFit struct {
	surface [3]float64      // the surface's channels, encoded, out of 255
	lands   [256][3]float64 // what each coverage lands in linear light, encoded, out of 255
	memo    map[float64]uint8
}

func newCoverageFit(s color.NRGBA) *coverageFit {
	f := &coverageFit{
		surface: [3]float64{float64(s.R), float64(s.G), float64(s.B)},
		memo:    map[float64]uint8{},
	}
	for k := range f.lands {
		keep := 1 - float64(k)/255
		for ch, v := range f.surface {
			f.lands[k][ch] = 255 * encode(keep*decode(v/255))
		}
	}
	return f
}

// coverage returns the 8-bit coverage for the encoded coverage cov, out
// of 1: the one whose worst channel misses the encoded composite least.
func (f *coverageFit) coverage(cov float64) uint8 {
	if k, ok := f.memo[cov]; ok {
		return k
	}
	best, bestMiss := 0, math.Inf(1)
	for k, lands := range f.lands {
		miss := 0.0
		for ch, v := range f.surface {
			miss = math.Max(miss, math.Abs(lands[ch]-(1-cov)*v))
		}
		if miss < bestMiss {
			best, bestMiss = k, miss
		}
	}
	f.memo[cov] = uint8(best)
	return uint8(best)
}

// decode and encode are the sRGB transfer function, both ways, on the unit
// interval.
func decode(v float64) float64 {
	if v <= 0.04045 {
		return v / 12.92
	}
	return math.Pow((v+0.055)/1.055, 2.4)
}

func encode(v float64) float64 {
	if v <= 0.0031308 {
		return v * 12.92
	}
	return 1.055*math.Pow(v, 1/2.4) - 0.055
}

package icons

import (
	"math"

	"gioui.org/f32"
	"golang.org/x/image/math/fixed"

	"github.com/vibrantgio/svg"
)

// Widening a mark's bands is an OFFSET of what it draws, not a second shape
// laid over it. Two shapes painted one after the other composite as
// a+b-ab, so a band's antialiased edge pixel would read up to a tenth of a
// pixel over what the geometry gives; the offset is computed here instead and
// filled once, so what the tests read off a render is the arithmetic the
// mark's own numbers give.
//
// The offset moves every drawn edge outward by half the widening, which
// leaves each band's centreline exactly where the grid puts it and grows the
// band symmetrically about it — what a heavier stroke in the same box is. A
// hole is wound against the contour that contains it, so the same signed
// offset shrinks it and the band between the two grows by the whole widening.

// flattenChord is the longest straight segment a curve is cut into, in device
// pixels. A quarter-pixel chord leaves a sagitta under a hundredth of a pixel
// on every curve this set draws, which stands inside the coverage its tests
// read.
const flattenChord = 0.25

// mitreLimit is how far a corner's offset point may stand out from the corner
// itself, as a multiple of the offset. Beyond it the corner is cut square
// rather than run to a spike.
const mitreLimit = 4

// contour is one closed loop of a mark's drawing, flattened to straight
// segments in device pixels.
type contour []f32.Point

// flattenPath cuts the buffered path into closed contours of straight
// segments. Every contour a mark draws is a fill, so an unclosed one is
// closed here.
func flattenPath(path []svg.Operation) []contour {
	var (
		out   []contour
		cur   contour
		start f32.Point
		at    f32.Point
	)
	flush := func() {
		if len(cur) > 2 {
			if same(cur[0], cur[len(cur)-1]) {
				cur = cur[:len(cur)-1]
			}
		}
		if len(cur) > 2 {
			out = append(out, cur)
		}
		cur = nil
	}
	add := func(p f32.Point) {
		if n := len(cur); n > 0 && same(cur[n-1], p) {
			return
		}
		cur = append(cur, p)
	}
	for _, o := range path {
		switch o := o.(type) {
		case svg.OpMoveTo:
			flush()
			at = pt(fixed.Point26_6(o))
			start = at
			cur = contour{at}
		case svg.OpLineTo:
			at = pt(fixed.Point26_6(o))
			add(at)
		case svg.OpQuadTo:
			b, c := pt(o[0]), pt(o[1])
			n := steps(span(at, b) + span(b, c))
			for i := 1; i <= n; i++ {
				add(quadAt(at, b, c, float32(i)/float32(n)))
			}
			at = c
		case svg.OpCubicTo:
			b, c, d := pt(o[0]), pt(o[1]), pt(o[2])
			n := steps(span(at, b) + span(b, c) + span(c, d))
			for i := 1; i <= n; i++ {
				add(cubeAt(at, b, c, d, float32(i)/float32(n)))
			}
			at = d
		case svg.OpClose:
			at = start
		}
	}
	flush()
	return out
}

// offsetContours grows what a path fills by d device pixels on every edge.
//
// Which way that is cannot be read off a contour's winding alone: under the
// non-zero rule a lone loop fills the same region whichever way it is
// traversed, and the set's files wind them both ways. So each contour is
// asked two questions instead — which way it turns, and whether it stands
// inside another contour's fill. A loop standing on its own grows away from
// what it encloses; a loop standing inside a fill is a hole, and it grows the
// fill by shrinking.
func offsetContours(cs []contour, d float64) []contour {
	out := make([]contour, 0, len(cs))
	for i, c := range cs {
		n := len(c)
		if n < 3 {
			continue
		}
		dir := d
		if signedArea(c) < 0 {
			dir = -dir
		}
		if enclosed(cs, i) {
			dir = -dir
		}
		o := make(contour, 0, n+8)
		for i := range c {
			here := c[i]
			n1, ok1 := edgeNormal(c[(i-1+n)%n], here)
			n2, ok2 := edgeNormal(here, c[(i+1)%n])
			switch {
			case !ok1 && !ok2:
				continue
			case !ok1:
				n1 = n2
			case !ok2:
				n2 = n1
			}
			cos := float64(n1.X*n2.X + n1.Y*n2.Y)
			if 1+cos > 1e-6 {
				m := f32.Point{
					X: float32(float64(n1.X+n2.X) / (1 + cos)),
					Y: float32(float64(n1.Y+n2.Y) / (1 + cos)),
				}
				if math.Hypot(float64(m.X), float64(m.Y)) <= mitreLimit {
					o = append(o, along(here, m, dir))
					continue
				}
			}
			o = append(o, along(here, n1, dir), along(here, n2, dir))
		}
		if len(o) > 2 {
			out = append(out, o)
		}
	}
	return out
}

// signedArea is twice the area a contour encloses, signed by the way it
// turns: positive where edgeNormal's normal points away from what it encloses.
func signedArea(c contour) float64 {
	var a float64
	for i := range c {
		p, q := c[i], c[(i+1)%len(c)]
		a += float64(p.X)*float64(q.Y) - float64(q.X)*float64(p.Y)
	}
	return a
}

// enclosed reports whether contour i stands inside what the rest of the path
// fills, which is what makes it a hole. The point it is asked about is a
// hair inside the contour's own longest edge, so it lies in the contour
// whatever shape the rest of it takes.
func enclosed(cs []contour, i int) bool {
	p, ok := justInside(cs[i])
	if !ok {
		return false
	}
	var w int
	for j, c := range cs {
		if j != i {
			w += winding(c, p)
		}
	}
	return w != 0
}

// justInside is a point a hundredth of a pixel inside the contour, taken off
// the midpoint of its longest edge.
func justInside(c contour) (f32.Point, bool) {
	var (
		best float64
		at   f32.Point
		ok   bool
	)
	for i := range c {
		p, q := c[i], c[(i+1)%len(c)]
		l := span(p, q)
		if l <= best {
			continue
		}
		n, fine := edgeNormal(p, q)
		if !fine {
			continue
		}
		in := -0.01
		if signedArea(c) < 0 {
			in = -in
		}
		best, at, ok = l, along(f32.Point{X: (p.X + q.X) / 2, Y: (p.Y + q.Y) / 2}, n, in), true
	}
	return at, ok
}

// winding is how many times the contour wraps the point, counted by the
// crossings of the ray running from it along +x.
func winding(c contour, p f32.Point) int {
	var w int
	for i := range c {
		a, b := c[i], c[(i+1)%len(c)]
		switch {
		case a.Y <= p.Y && b.Y > p.Y:
			if side(a, b, p) > 0 {
				w++
			}
		case a.Y > p.Y && b.Y <= p.Y:
			if side(a, b, p) < 0 {
				w--
			}
		}
	}
	return w
}

// side is which side of the line a→b the point stands on.
func side(a, b, p f32.Point) float64 {
	return float64(b.X-a.X)*float64(p.Y-a.Y) - float64(p.X-a.X)*float64(b.Y-a.Y)
}

// edgeNormal is the unit normal of the segment a→b that points away from what
// a positively wound contour encloses.
func edgeNormal(a, b f32.Point) (f32.Point, bool) {
	dx, dy := float64(b.X-a.X), float64(b.Y-a.Y)
	l := math.Hypot(dx, dy)
	if l < 1e-6 {
		return f32.Point{}, false
	}
	return f32.Point{X: float32(dy / l), Y: float32(-dx / l)}, true
}

func along(p, dir f32.Point, d float64) f32.Point {
	return f32.Point{X: p.X + float32(float64(dir.X)*d), Y: p.Y + float32(float64(dir.Y)*d)}
}

func same(a, b f32.Point) bool {
	return math.Abs(float64(a.X-b.X)) < 1e-5 && math.Abs(float64(a.Y-b.Y)) < 1e-5
}

func span(a, b f32.Point) float64 {
	return math.Hypot(float64(b.X-a.X), float64(b.Y-a.Y))
}

// steps is how many straight segments a curve of the given control-polygon
// length is cut into.
func steps(l float64) int {
	n := int(math.Ceil(l / flattenChord))
	return min(max(n, 1), 256)
}

func quadAt(a, b, c f32.Point, t float32) f32.Point {
	u := 1 - t
	return f32.Point{
		X: u*u*a.X + 2*u*t*b.X + t*t*c.X,
		Y: u*u*a.Y + 2*u*t*b.Y + t*t*c.Y,
	}
}

func cubeAt(a, b, c, d f32.Point, t float32) f32.Point {
	u := 1 - t
	return f32.Point{
		X: u*u*u*a.X + 3*u*u*t*b.X + 3*u*t*t*c.X + t*t*t*d.X,
		Y: u*u*u*a.Y + 3*u*u*t*b.Y + 3*u*t*t*c.Y + t*t*t*d.Y,
	}
}

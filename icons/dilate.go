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

// knockoutBands is how far a hole may stand from the contour around it and
// still be a band's inner edge, in bands. A band's inner edge lies exactly
// one band from its outer edge; the farthest point of a figure knocked out
// of a solid one lies two bands and more away in this set, so the two are
// told apart at one and a half.
const knockoutBands = 1.5

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

// bandedPath reports whether every hole this path draws is a band's inner
// edge, which is the only thing the widening is spent on.
//
// A hole standing a band from the contour around it IS that band's inner
// edge: the same signed offset shrinks it while the outer edge grows, and
// the band between the two comes out the device width its size's capture
// gives. A hole standing further away is a FIGURE knocked out of a solid one
// — the folder-plus badge's plus, the xmark-circle mark's cross — where the figure
// and the hole are both measured extents rather than a weight, and the
// offset would thicken the one while closing the other. Such a path is drawn
// at the extents the grid gives it at every size.
//
// The judgement is the DRAWING's rather than the mark's: one mark carries
// both kinds, the folder-plus's bands widening while its badge holds its
// measured 8 units and its plus the measured span knocked out of it.
func bandedPath(cs []contour, band float64) bool {
	if band <= 0 {
		return true
	}
	for i := range cs {
		if !enclosed(cs, i) {
			continue
		}
		for _, p := range cs[i] {
			if clearance(cs, i, p) > band*knockoutBands {
				return false
			}
		}
	}
	return true
}

// holdClear opens the air between a path and a figure knocked out of a solid
// one to need, moving only what stands closer than that.
//
// It is what keeps that air open at every size. A band's edges are offset
// outward by half the widening, and the cut a band is drawn with round such a
// figure is one of those edges: offset, it moves INWARD, and the air closes by
// that much while the figure itself holds the extents the grid gives it. Run
// before the offset with need standing half the widening outside the air that
// is to be left, the cut comes out of the offset exactly that far from the
// figure, and the corners where it meets the rest of the drawing are mitred by
// the offset as they are everywhere else.
//
// A point closer than need is moved out to need along the line from the
// nearest point of the figure, which on a round figure is the round grown. A
// SEGMENT running from outside need to inside it is cut where it crosses,
// and the crossing is kept: without it the segment's far end would be dragged
// along with the point, tilting an edge the length of the mark away from
// where the grid puts it.
func holdClear(cs []contour, from []contour, need float64) ([]contour, bool) {
	var moved bool
	out := make([]contour, 0, len(cs))
	for _, c := range cs {
		o := make(contour, 0, len(c)+8)
		for i := range c {
			a, b := c[i], c[(i+1)%len(c)]
			pa, ina := hold(from, a, need)
			o = append(o, pa)
			moved = moved || ina
			if _, inb := hold(from, b, need); ina != inb {
				o = append(o, crossing(from, a, b, need))
			}
		}
		out = append(out, o)
	}
	return out, moved
}

// hold is p held at need from the nearest point of the figure, and whether it
// stood closer than that.
func hold(from []contour, p f32.Point, need float64) (f32.Point, bool) {
	// Air that meets the need to within a twentieth of a device pixel IS the
	// drawn air, and the path is laid down as the file gives it. That is the
	// precision the drawing arrives at rather than a tolerance chosen here:
	// the walk hands its points over on a 64th of a pixel and cuts every
	// curve into chords, which puts a point of the set's one cut up to a
	// fiftieth of a pixel off the air the file's own numbers give. What this
	// opens falls short by a tenth of a pixel and more.
	const slack = 0.05
	q, d := nearestPoint(from, p)
	if d < slack || d >= need-slack {
		return p, false
	}
	return f32.Point{
		X: q.X + float32(float64(p.X-q.X)*need/d),
		Y: q.Y + float32(float64(p.Y-q.Y)*need/d),
	}, true
}

// crossing is the point of the segment a-b standing need from the figure,
// bisected: the distance to a figure runs continuously along the segment and
// one end stands inside need and the other outside.
func crossing(from []contour, a, b f32.Point, need float64) f32.Point {
	lo, hi := 0.0, 1.0
	at := func(t float64) f32.Point {
		return f32.Point{X: a.X + float32(float64(b.X-a.X)*t), Y: a.Y + float32(float64(b.Y-a.Y)*t)}
	}
	_, da := nearestPoint(from, a)
	for range 40 {
		mid := (lo + hi) / 2
		_, d := nearestPoint(from, at(mid))
		if (d < need) == (da < need) {
			lo = mid
		} else {
			hi = mid
		}
	}
	return at((lo + hi) / 2)
}

// nearestPoint is the closest point of any of the contours to p, and how far
// off it stands.
func nearestPoint(cs []contour, p f32.Point) (f32.Point, float64) {
	var (
		at   f32.Point
		best = math.Inf(1)
	)
	for _, c := range cs {
		for i := range c {
			a, b := c[i], c[(i+1)%len(c)]
			q := onSegment(p, a, b)
			if d := span(p, q); d < best {
				best, at = d, q
			}
		}
	}
	return at, best
}

// onSegment is the point of the segment a-b closest to p.
func onSegment(p, a, b f32.Point) f32.Point {
	dx, dy := float64(b.X-a.X), float64(b.Y-a.Y)
	l := dx*dx + dy*dy
	t := 0.0
	if l > 0 {
		t = (float64(p.X-a.X)*dx + float64(p.Y-a.Y)*dy) / l
		t = min(max(t, 0), 1)
	}
	return f32.Point{X: a.X + float32(t*dx), Y: a.Y + float32(t*dy)}
}

// clearance is how far p stands from the nearest point of every contour in
// the path but the one it was taken from.
func clearance(cs []contour, i int, p f32.Point) float64 {
	best := math.Inf(1)
	for j, c := range cs {
		if j == i {
			continue
		}
		for k := range c {
			if d := toSegment(p, c[k], c[(k+1)%len(c)]); d < best {
				best = d
			}
		}
	}
	return best
}

// toSegment is the distance from p to the segment a-b.
func toSegment(p, a, b f32.Point) float64 { return span(p, onSegment(p, a, b)) }

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

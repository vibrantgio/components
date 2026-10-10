package icons

import (
	"gioui.org/f32"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"golang.org/x/image/math/fixed"

	"github.com/vibrantgio/svg"
	svgdriver "github.com/vibrantgio/svg/driver"
)

// drawing returns the ops for one mark at one pixel size, building them on
// first ask and replaying the recorded macro afterwards. The recorded ops
// carry geometry and per-path opacity only; the painter supplies the colour
// outside them, which is why the size and the widening are the whole key.
//
// widen is how much wider than the grid's 1.4 units the band is drawn here,
// in device pixels, keyed in thousandths so that two frames at one size build
// one drawing.
func (s *Set) drawing(entry string, px, widen int) op.CallOp {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := cacheKey{entry: entry, px: px, widen: widen}
	if call, ok := s.cache[key]; ok {
		return call
	}

	// The parsed drawing is resized in place, so it is only ever touched
	// under this lock and never handed out: one mark drawn at two sizes in
	// the same frame would otherwise fight over the transform.
	parsed, _ := s.reg.Icon(entry)
	mark := parsed.SVG()

	macro := op.Record(s.ops)
	mark.SetTarget(mark.ViewBox.AspectMeet(float64(px), float64(px), 0.5, 0.5))
	rec := &recorder{
		ops:   s.ops,
		widen: float64(widen) / 1000,
		band:  bandUnitsAt24 * float64(px) / gridUnits,
		clear: clearancePx(px),
	}
	svgdriver.Draw(rec, mark, 1)
	rec.flush()
	call := macro.Stop()

	s.cache[key] = call
	return call
}

// recorder is the drawing backend the set builds its marks through: a sink
// that turns the svg package's own drawing walk into Gio clip and paint ops.
//
// It exists because the shipped Gio backend picks the colour out of the file
// and bakes it into the ops. A mark takes the control's colour, not its own,
// and ops with a colour in them would have to be rebuilt whenever that colour
// moved — so this backend emits coverage and leaves the colour to whatever is
// selected when the ops are replayed. Per-path opacity is kept, as an opacity
// layer, because a faint element has to stay faint against any colour.
type recorder struct {
	ops    *op.Ops
	path   []svg.Operation
	stroke svgdriver.StrokeOptions
	// held is every path the walk handed over, kept until the walk is done:
	// what a band is held clear of is another path of the same mark, and the
	// drawing is not known to be complete until the last of them is in.
	held []held
	// widen is how much wider than the grid's own 1.4 units every band this
	// drawing lays down is, in device pixels. It is spent as an offset of
	// half its own width on every edge, so a band grows by the whole of it
	// and keeps its centreline.
	widen float64
	// band is the grid's own band at this size, in device pixels. It is what
	// tells a band's inner edge from a figure knocked out of a solid one, so
	// that a path drawing the second takes no widening.
	band float64
	// clear is the air a band keeps from a figure knocked out of a solid one,
	// in device pixels at this size.
	clear float64
}

// held is one path of the mark as the walk handed it over, with the coverage
// it is to be filled or stroked at.
type held struct {
	path    []svg.Operation
	stroke  svgdriver.StrokeOptions
	alpha   float32
	stroked bool
}

// assert interface conformance
var _ svgdriver.DrawerNG = (*recorder)(nil)

func (r *recorder) Clear() {
	r.path = r.path[:0]
	r.stroke = svgdriver.StrokeOptions{}
}

// SetWinding is ignored: Gio's outline fills by the non-zero rule and offers
// no other, so a mark's holes are wound rather than declared.
func (r *recorder) SetWinding(useNonZeroWinding bool) {}

func (r *recorder) SetStrokeOptions(o svgdriver.StrokeOptions) { r.stroke = o }

func (r *recorder) Start(a fixed.Point26_6) { r.path = append(r.path, svg.OpMoveTo(a)) }
func (r *recorder) Line(b fixed.Point26_6)  { r.path = append(r.path, svg.OpLineTo(b)) }

func (r *recorder) QuadBezier(b, c fixed.Point26_6) { r.path = append(r.path, svg.OpQuadTo{b, c}) }

func (r *recorder) CubeBezier(b, c, d fixed.Point26_6) {
	r.path = append(r.path, svg.OpCubicTo{b, c, d})
}

func (r *recorder) Close() { r.path = append(r.path, svg.OpClose{}) }

func (r *recorder) Fill(col svg.Pattern, opacity float64) {
	r.hold(held{alpha: coverage(col, opacity)})
}

// Stroke is here for completeness — marks are authored as outlines, because
// the backend cannot set line cap or join — and renders butt-capped and
// mitred when a file uses it anyway.
func (r *recorder) Stroke(col svg.Pattern, opacity float64) {
	r.hold(held{stroke: r.stroke, alpha: coverage(col, opacity), stroked: true})
}

// hold keeps one path of the walk, copied because the buffer is reused.
func (r *recorder) hold(h held) {
	h.path = append([]svg.Operation(nil), r.path...)
	r.held = append(r.held, h)
}

// flush lays down every path the walk handed over, in the order it drew them.
//
// It is one step rather than a path at a time because the widening is spent
// against the WHOLE drawing: a path that takes the widening is held clear of
// every path that takes none, and the one it is held clear of may not have
// been walked yet when it is itself laid down.
func (r *recorder) flush() {
	var (
		paths     = make([][]contour, len(r.held))
		banded    = make([]bool, len(r.held))
		knockouts []contour
	)
	for i, h := range r.held {
		if h.stroked {
			continue
		}
		paths[i] = flattenPath(h.path)
		banded[i] = bandedPath(paths[i], r.band)
		if !banded[i] {
			knockouts = append(knockouts, paths[i]...)
		}
	}
	for i, h := range r.held {
		if h.stroked {
			r.cover(clip.Stroke{
				Path:  r.replay(h.path),
				Width: float32(h.stroke.LineWidth) / 64,
			}.Op(), h.alpha)
			continue
		}
		r.cover(clip.Outline{Path: r.spec(h.path, paths[i], knockouts, banded[i])}.Op(), h.alpha)
	}
}

// cover paints the shape with whatever colour is selected at replay time,
// under an opacity layer when the path asked to be fainter than the rest.
func (r *recorder) cover(shape clip.Op, alpha float32) {
	if alpha <= 0 {
		return
	}
	if alpha < 1 {
		defer paint.PushOpacity(r.ops, alpha).Pop()
	}
	defer shape.Push(r.ops).Pop()
	paint.PaintOp{}.Add(r.ops)
}

// spec is one filled path as it is laid down at this size.
//
// The widening is spent PER PATH, on the paths that draw bands: a path whose
// hole is a figure knocked out of a solid one is laid down as the grid gives
// it, because an offset there would thicken the figure while closing the hole
// and both are measured extents.
//
// A path that does take the widening is first held clear of the knocked-out
// figures of the same mark, so the air between the two is never closed by it:
// the cut a band is drawn with round such a figure GROWS with the widening,
// and grows further where the grid's own band of air comes to less than a
// device pixel at this size. The clearance cannot be drawn in the file
// because it is a device width and the file is one drawing for every size, so
// the widening step computes it here.
func (r *recorder) spec(path []svg.Operation, cs, knockouts []contour, banded bool) clip.PathSpec {
	if !banded {
		return r.replay(path)
	}
	var moved bool
	if len(knockouts) > 0 {
		cs, moved = holdClear(cs, knockouts, r.clear+r.widen/2)
	}
	switch {
	case r.widen > 0:
		return r.flatSpec(offsetContours(cs, r.widen/2))
	case moved:
		return r.flatSpec(cs)
	}
	return r.replay(path)
}

// replay builds a fresh clip.Path from the operations as the file draws them,
// which is what the single-use clip.PathSpec forces: a path filled and
// stroked is built twice.
func (r *recorder) replay(path []svg.Operation) clip.PathSpec {
	var p clip.Path
	p.Begin(r.ops)
	for _, o := range path {
		switch o := o.(type) {
		case svg.OpMoveTo:
			p.MoveTo(pt(fixed.Point26_6(o)))
		case svg.OpLineTo:
			p.LineTo(pt(fixed.Point26_6(o)))
		case svg.OpQuadTo:
			p.QuadTo(pt(o[0]), pt(o[1]))
		case svg.OpCubicTo:
			p.CubeTo(pt(o[0]), pt(o[1]), pt(o[2]))
		case svg.OpClose:
			p.Close()
		}
	}
	return p.End()
}

// flatSpec fills the flattened contours as one path, so the coverage a render
// reads is that geometry's own and not two shapes composited over each other.
func (r *recorder) flatSpec(cs []contour) clip.PathSpec {
	var p clip.Path
	p.Begin(r.ops)
	for _, c := range cs {
		p.MoveTo(c[0])
		for _, q := range c[1:] {
			p.LineTo(q)
		}
		p.Close()
	}
	return p.End()
}

// coverage is how much of the control's colour a path gets: the style's own
// opacity, times the alpha of the fill it was authored with. The fill's hue is
// discarded — a mark is monochrome, and a gradient is painted as flat coverage
// rather than as colours the control cannot override.
func coverage(col svg.Pattern, opacity float64) float32 {
	if c, ok := col.(svg.PlainColor); ok {
		opacity *= float64(c.A) / 0xff
	}
	switch {
	case opacity <= 0:
		return 0
	case opacity >= 1:
		return 1
	}
	return float32(opacity)
}

func pt(p fixed.Point26_6) f32.Point {
	return f32.Point{X: float32(p.X) / 64, Y: float32(p.Y) / 64}
}

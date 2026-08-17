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
// outside them, which is why the size is the whole key.
func (s *Set) drawing(entry string, px int) op.CallOp {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := cacheKey{entry: entry, px: px}
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
	svgdriver.Draw(&recorder{ops: s.ops}, mark, 1)
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
	r.cover(clip.Outline{Path: r.spec()}.Op(), coverage(col, opacity))
}

// Stroke is here for completeness — marks are authored as outlines, because
// the backend cannot set line cap or join — and renders butt-capped and
// mitred when a file uses it anyway.
func (r *recorder) Stroke(col svg.Pattern, opacity float64) {
	r.cover(clip.Stroke{
		Path:  r.spec(),
		Width: float32(r.stroke.LineWidth) / 64,
	}.Op(), coverage(col, opacity))
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

// spec replays the buffered path into a fresh clip.Path, which is what the
// single-use clip.PathSpec forces: a path filled and stroked is built twice.
func (r *recorder) spec() clip.PathSpec {
	var p clip.Path
	p.Begin(r.ops)
	for _, o := range r.path {
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

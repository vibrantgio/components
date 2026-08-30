// Package bench provides the shared benchmark harness for Components components.
//
// BenchFrame standardises per-frame measurement across components: it drives
// widget(gtx) for b.N frames against synthesized constraints, resets the op
// buffer each frame, and enables b.ReportAllocs so both wall-clock (ns/op) and
// per-frame allocation (B/op) can be compared against recorded baselines.
// A change is a regression once either metric moves more than 5% for a
// component.
package bench

import (
	"image"
	"testing"

	"gioui.org/io/input"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"
)

// DefaultSize is the synthesized frame size used when WithSize is not supplied.
// 300×60 px matches the constraints the button and text-field benchmarks have
// always run under.
var DefaultSize = image.Pt(300, 60)

type config struct {
	size   image.Point
	router *input.Router
}

// Option customises a BenchFrame run.
type Option func(*config)

// WithSize overrides the synthesized frame constraints (default 300×60 px).
// Components with a meaningful viewport size — e.g. a virtual list — set their
// own.
func WithSize(p image.Point) Option {
	return func(c *config) { c.size = p }
}

// WithRouter routes input through r for the duration of the benchmark:
// gtx.Source is set to r.Source() and r.Frame is called after each widget(gtx).
// It is required for components whose rendering depends on input state — e.g. a
// focused widget.Editor that draws a blinking caret only while gtx.Focused is
// true. The caller is responsible for delivering any focus/queue events (and
// pumping the pre-roll frames) on r before handing it to BenchFrame; the timed
// loop then re-renders that established state.
func WithRouter(r *input.Router) Option {
	return func(c *config) { c.router = r }
}

// BenchFrame drives w(gtx) for b.N frames against synthesized constraints,
// resetting the op buffer each frame and reporting allocations. It is the
// standard measurement path every Components component benchmark plugs into.
func BenchFrame(b *testing.B, w layout.Widget, opts ...Option) {
	b.Helper()
	cfg := config{size: DefaultSize}
	for _, opt := range opts {
		opt(&cfg)
	}

	var ops op.Ops
	gtx := layout.Context{
		Metric:      unit.Metric{PxPerDp: 1, PxPerSp: 1},
		Constraints: layout.Exact(cfg.size),
		Ops:         &ops,
	}
	if cfg.router != nil {
		gtx.Source = cfg.router.Source()
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		ops.Reset()
		w(gtx)
		if cfg.router != nil {
			cfg.router.Frame(&ops)
		}
	}
}

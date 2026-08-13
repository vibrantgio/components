// Package cache provides FrameCache, a per-widget op-recording cache for
// animation-heavy widgets in Gio layouts.
//
// # Problem
//
// Gio widgets re-record their draw commands every frame. When a widget's
// visual state is unchanged — a settled panel, a read-only data grid, a
// static label — that re-recording is wasted work. Indicator.Compute-style
// allocations inside the widget closure run even when nothing changed.
//
// # Solution
//
// FrameCache wraps the op.Record / call.Add pattern from gioui.org/op.
// The caller records the widget once into a per-instance op.Ops buffer;
// on subsequent frames where dirty is false, only call.Add replays the
// recorded commands into the frame ops. The widget body (and its
// allocations) is bypassed entirely.
//
// # When to use
//
// A FrameCache is appropriate when:
//   - The widget is expensive to render (≥ 5 µs or ≥ 5 allocs per call).
//   - The widget can derive a "dirty" signal from its inputs — a changed
//     data version, a cursor move, a theme change — without rendering.
//   - The cache lifetime spans multiple frames (scoped to an rx.Defer
//     or similar long-lived closure).
//
// A FrameCache is NOT appropriate for widgets that change every frame
// (physics-driven animation, live streaming data). In those cases the
// cache check is pure overhead — equilibrium is never reached.
//
// # Threading
//
// FrameCache is not safe for concurrent use. All calls must occur on the
// Gio frame goroutine (the goroutine that receives layout.Context).
//
// # Invalidation semantics
//
// The dirty flag is the caller's responsibility. Compute it in screen-pixel
// space rather than physics- or model-coordinate space: a sub-pixel move
// does not warrant a re-record. The rule is empirical — validation
// Experiment B established it; that experiment's record predates the
// organization and is not preserved, and its surviving measurements are
// reproduced below.
//
// # Performance reference (Experiment B, Apple M1, darwin/arm64)
//
// At equilibrium (dirty=false, all cache hits):
//   - call.Add replay: ~9 µs for 200 nodes vs ~211 µs naive — 23× faster.
//
// At non-equilibrium (dirty=true, all cache misses):
//   - Same cost as naive re-record: op.Record overhead is negligible.
//
// Break-even vs a dedicated scene primitive: N ≈ 300,000 entities.
// For realistic graph-scale UIs (N ≤ 10,000) the op-cache is sufficient.
package cache

import "gioui.org/op"

// FrameCache holds a single recorded op.Ops subtree. Allocate one
// FrameCache per logically distinct widget region, typically inside an
// rx.Defer or long-lived subscription closure so the buffer survives
// across frames.
type FrameCache struct {
	ops   *op.Ops
	call  op.CallOp
	valid bool
}

// New returns a FrameCache backed by a fresh op.Ops buffer.
func New() *FrameCache {
	return &FrameCache{ops: new(op.Ops)}
}

// Draw replays cached draw commands into dst when dirty is false and the
// cache has been recorded at least once. Otherwise draw is called to
// re-record, the result is stored, and the commands are replayed into dst.
//
// draw must record exactly the drawing commands for this widget region into
// the op.Ops it receives. It must not retain that *op.Ops across calls.
func (c *FrameCache) Draw(dst *op.Ops, dirty bool, draw func(*op.Ops)) {
	if !dirty && c.valid {
		c.call.Add(dst)
		return
	}
	c.ops.Reset()
	macro := op.Record(c.ops)
	draw(c.ops)
	c.call = macro.Stop()
	c.valid = true
	c.call.Add(dst)
}

// Invalidate marks the cache as stale so the next Draw call re-records.
// Use this when the widget's visual state changes outside the normal dirty
// path — e.g., on a theme change or after an explicit resize.
func (c *FrameCache) Invalidate() {
	c.valid = false
}

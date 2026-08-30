package cache_test

import (
	"image"
	"image/color"
	"testing"

	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"

	"github.com/vibrantgio/components/cache"
)

// sinkData prevents the compiler from eliding heap allocations in indicatorCompute.
var sinkData []float64

// indicatorCompute simulates one call to a typical indicator-computation
// function (EMA, SMA, Bollinger, etc.). The //go:noinline directive plus
// sinkData assignment ensure make([]float64, 500) escapes to the heap,
// producing exactly 1 heap alloc per call.
//
//go:noinline
func indicatorCompute() []float64 {
	s := make([]float64, 500) // matches a typical chart benchmark's candle count
	s[0] = 1.0
	sinkData = s
	return s
}

// simulateRender models a chart pane's Render call: n indicator-compute calls
// (each 1 alloc) followed by clip/paint op encoding (0 allocs).
func simulateRender(ops *op.Ops, n int) {
	for range n {
		indicatorCompute()
	}
	r := clip.Rect(image.Rect(0, 0, 100, 80)).Push(ops)
	paint.Fill(ops, color.NRGBA{R: 40, G: 80, B: 180, A: 255})
	r.Pop()
}

// BenchmarkNaive re-records every pane every frame without caching.
// Calibrated to ~120 allocs/op (12 panes × 10 allocs each), close to a
// measured baseline of 115 allocs/op.
func BenchmarkNaive(b *testing.B) {
	const (
		numPanes    = 12
		workPerPane = 10
	)
	ops := new(op.Ops)
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		ops.Reset()
		for range numPanes {
			simulateRender(ops, workPerPane)
		}
	}
}

// BenchmarkCached uses FrameCache with a "mostly static" dirty pattern:
// 4 of 12 panes are dirty per frame (live-updating panes), 8 are settled.
// Cached panes pay only call.Add replay cost — no allocs from simulateRender.
func BenchmarkCached(b *testing.B) {
	const (
		numPanes    = 12
		dirtyPanes  = 4 // 1/3 dirty → 2/3 cache hits per frame
		workPerPane = 10
	)
	caches := make([]*cache.FrameCache, numPanes)
	for i := range caches {
		caches[i] = cache.New()
	}
	ops := new(op.Ops)
	b.ReportAllocs()
	b.ResetTimer()
	for frame := range b.N {
		ops.Reset()
		for i, c := range caches {
			dirty := i < dirtyPanes || frame == 0
			c.Draw(ops, dirty, func(o *op.Ops) {
				simulateRender(o, workPerPane)
			})
		}
	}
}

// TestAllocationRegression verifies that FrameCache achieves at least 30%
// fewer allocations per frame than naive re-recording.
//
// Reference: measured baseline = 115 allocs/op.
// Scenario: 4/12 dirty panes → expected cached = ~40 allocs (67% reduction).
func TestAllocationRegression(t *testing.T) {
	const (
		numPanes    = 12
		dirtyPanes  = 4 // matches BenchmarkCached dirty pattern
		workPerPane = 10
		runs        = 1000
	)

	ops := new(op.Ops)

	naiveAllocs := testing.AllocsPerRun(runs, func() {
		ops.Reset()
		for range numPanes {
			simulateRender(ops, workPerPane)
		}
	})

	caches := make([]*cache.FrameCache, numPanes)
	for i := range caches {
		caches[i] = cache.New()
	}
	// Prime all caches on the first frame so subsequent runs hit the clean path.
	ops.Reset()
	for _, c := range caches {
		c.Draw(ops, true, func(o *op.Ops) { simulateRender(o, workPerPane) })
	}

	cachedAllocs := testing.AllocsPerRun(runs, func() {
		ops.Reset()
		for i, c := range caches {
			dirty := i < dirtyPanes
			c.Draw(ops, dirty, func(o *op.Ops) {
				simulateRender(o, workPerPane)
			})
		}
	})

	threshold := naiveAllocs * 0.70 // cached must reach ≥ 30% reduction

	t.Logf("naive  allocs/call: %.1f  (G−1.6 baseline: 115)", naiveAllocs)
	t.Logf("cached allocs/call: %.1f  (target ≤ %.1f, −30%% of naive)", cachedAllocs, threshold)

	if naiveAllocs == 0 {
		t.Fatal("naive reported 0 allocs — indicatorCompute elision not prevented")
	}
	if cachedAllocs > threshold {
		t.Errorf("FrameCache missed 30%% allocation target: cached=%.1f naive=%.1f threshold=%.1f",
			cachedAllocs, naiveAllocs, threshold)
	}
}

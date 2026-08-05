package input_test

import (
	"image"
	"testing"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"

	"github.com/vibrantgio/prism/input"
	"github.com/vibrantgio/spectrum/tokens"
)

// BenchmarkRadioRender exercises widget(gtx) for b.N synthetic frames,
// per DESIGN §"Performance — Profiling". b.ReportAllocs is enabled so CI
// can gate on per-frame allocation regressions (>5% threshold).
func BenchmarkRadioRender(b *testing.B) {
	w := input.RenderRadio(
		tokens.DefaultLight, tokens.Spacing, tokens.Radius,
		input.RadioRenderState{},
	)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		var ops op.Ops
		gtx := layout.Context{
			Metric:      unit.Metric{PxPerDp: 1, PxPerSp: 1},
			Constraints: layout.Exact(image.Pt(44, 44)),
			Ops:         &ops,
		}
		w(gtx)
	}
}

// BenchmarkRadioRenderSelected benchmarks the selected state which draws a
// three-layer nested fill (outer ring, surface gap, inner dot).
func BenchmarkRadioRenderSelected(b *testing.B) {
	w := input.RenderRadio(
		tokens.DefaultLight, tokens.Spacing, tokens.Radius,
		input.RadioRenderState{Selected: true},
	)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		var ops op.Ops
		gtx := layout.Context{
			Metric:      unit.Metric{PxPerDp: 1, PxPerSp: 1},
			Constraints: layout.Exact(image.Pt(44, 44)),
			Ops:         &ops,
		}
		w(gtx)
	}
}

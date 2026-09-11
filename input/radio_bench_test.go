package input_test

import (
	"image"
	"testing"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"

	"github.com/vibrantgio/components/input"
	"github.com/vibrantgio/theme/tokens"
)

// BenchmarkRadioRender exercises the layout.Widget for b.N synthetic frames.
// b.ReportAllocs is enabled so CI can gate on per-frame allocation
// regressions (>5% threshold).
func BenchmarkRadioRender(b *testing.B) {
	w := input.RenderRadio(
		tokens.PlatformLight, tokens.Spacing, tokens.Radius,
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

// BenchmarkRadioRenderSelected benchmarks the selected state, which draws the
// accent disc and the dot on it rather than an edge and a gap.
func BenchmarkRadioRenderSelected(b *testing.B) {
	w := input.RenderRadio(
		tokens.PlatformLight, tokens.Spacing, tokens.Radius,
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

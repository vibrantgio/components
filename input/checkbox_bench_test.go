package input_test

import (
	"image"
	"testing"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"

	"github.com/vibrantgio/prism/input"
	"github.com/vibrantgio/theme/tokens"
)

// BenchmarkCheckboxRender exercises widget(gtx) for b.N synthetic frames,
// per DESIGN §"Performance — Profiling". b.ReportAllocs is enabled so CI
// can gate on per-frame allocation regressions (>5% threshold).
func BenchmarkCheckboxRender(b *testing.B) {
	w := input.RenderCheckbox(
		tokens.DefaultLight, tokens.Spacing, tokens.Radius,
		input.CheckboxRenderState{},
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

// BenchmarkCheckboxRenderChecked benchmarks the checked state which draws a
// solid primary fill instead of a bordered box.
func BenchmarkCheckboxRenderChecked(b *testing.B) {
	w := input.RenderCheckbox(
		tokens.DefaultLight, tokens.Spacing, tokens.Radius,
		input.CheckboxRenderState{Checked: true},
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

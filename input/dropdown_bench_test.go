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

// BenchmarkDropdownRender exercises the closed dropdown widget(gtx) for b.N
// synthetic frames, per DESIGN §"Performance — Profiling". b.ReportAllocs is
// enabled so CI can gate on per-frame allocation regressions (>5% threshold).
func BenchmarkDropdownRender(b *testing.B) {
	shaper := tokens.DefaultTypography.Shaper()
	w := input.RenderDropdown(
		shaper,
		tokens.DefaultLight, tokens.Spacing, tokens.Radius, tokens.DefaultTypography.BodyLarge, tokens.Comfortable,
		input.DropdownRenderState{Options: []string{"Option A", "Option B", "Option C"}},
	)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		var ops op.Ops
		gtx := layout.Context{
			Metric:      unit.Metric{PxPerDp: 1, PxPerSp: 1},
			Constraints: layout.Exact(image.Pt(200, 44)),
			Ops:         &ops,
		}
		w(gtx)
	}
}

// BenchmarkDropdownRenderOpen benchmarks the open state which additionally
// draws the option list below the trigger.
func BenchmarkDropdownRenderOpen(b *testing.B) {
	shaper := tokens.DefaultTypography.Shaper()
	opts := []string{"Option A", "Option B", "Option C"}
	openH := 44 + len(opts)*44
	w := input.RenderDropdown(
		shaper,
		tokens.DefaultLight, tokens.Spacing, tokens.Radius, tokens.DefaultTypography.BodyLarge, tokens.Comfortable,
		input.DropdownRenderState{Open: true, Options: opts, Selected: 0},
	)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		var ops op.Ops
		gtx := layout.Context{
			Metric:      unit.Metric{PxPerDp: 1, PxPerSp: 1},
			Constraints: layout.Exact(image.Pt(200, openH)),
			Ops:         &ops,
		}
		w(gtx)
	}
}

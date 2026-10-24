package chip_test

import (
	"image"
	"testing"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"

	"github.com/vibrantgio/components/chip"
	"github.com/vibrantgio/theme/tokens"
)

// benchFrame drives w for b.N synthetic frames, with b.ReportAllocs enabled
// so the per-frame allocations are reported.
func benchFrame(b *testing.B, w layout.Widget) {
	b.Helper()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var ops op.Ops
		gtx := layout.Context{
			Metric:      unit.Metric{PxPerDp: 1, PxPerSp: 1},
			Constraints: layout.Constraints{Max: image.Pt(220, 56)},
			Ops:         &ops,
		}
		w(gtx)
	}
}

// BenchmarkChipRender is the resting chip: one shaped title, one leading mark,
// the outline and the body.
func BenchmarkChipRender(b *testing.B) {
	benchFrame(b, chip.Render(
		tokens.DefaultTypography.DeterministicShaper(), "Claude · Opus 5", chip.Assist, chevron,
		tokens.PlatformLight, tokens.Spacing, tokens.Radius,
		tokens.DefaultTypography.LabelLarge, tokens.Comfortable, chip.RenderState{}))
}

// BenchmarkChipRenderSelected is the other rest: the container derived against
// the level, the walk's stop, and the checkmark the component strokes itself.
func BenchmarkChipRenderSelected(b *testing.B) {
	benchFrame(b, chip.Render(
		tokens.DefaultTypography.DeterministicShaper(), "Claude · Opus 5", chip.Filter, nil,
		tokens.PlatformLight, tokens.Spacing, tokens.Radius,
		tokens.DefaultTypography.LabelLarge, tokens.Comfortable,
		chip.RenderState{Selected: true}))
}

// BenchmarkChipRenderFocused adds the focus ring, which is the one shape in
// the chip drawn as a stroked path rather than as a fill.
func BenchmarkChipRenderFocused(b *testing.B) {
	benchFrame(b, chip.Render(
		tokens.DefaultTypography.DeterministicShaper(), "Claude · Opus 5", chip.Assist, chevron,
		tokens.PlatformLight, tokens.Spacing, tokens.Radius,
		tokens.DefaultTypography.LabelLarge, tokens.Comfortable,
		chip.RenderState{Focused: true}))
}

package button_test

import (
	"testing"

	"github.com/vibrantgio/prism/bench"
	"github.com/vibrantgio/prism/button"
	"github.com/vibrantgio/spectrum/tokens"
)

// BenchmarkButtonRender exercises widget(gtx) for b.N synthetic frames via the
// shared bench.BenchFrame harness (DESIGN §"Performance — Methodology"). The
// harness enables b.ReportAllocs so per-frame allocation regressions (>5%
// threshold) are measurable. This is the idle render: default unfocused state.
func BenchmarkButtonRender(b *testing.B) {
	shaper := tokens.DefaultTypography.Shaper()
	w := button.Render(
		shaper, "Benchmark",
		tokens.DefaultLight, tokens.Spacing, tokens.Radius, tokens.DefaultTypeScale,
		button.RenderState{},
	)
	bench.BenchFrame(b, w)
}

// BenchmarkButtonRenderFocused benchmarks the focused state which draws an
// extra clip.Stroke path for the focus ring.
func BenchmarkButtonRenderFocused(b *testing.B) {
	shaper := tokens.DefaultTypography.Shaper()
	w := button.Render(
		shaper, "Benchmark",
		tokens.DefaultLight, tokens.Spacing, tokens.Radius, tokens.DefaultTypeScale,
		button.RenderState{Focused: true},
	)
	bench.BenchFrame(b, w)
}

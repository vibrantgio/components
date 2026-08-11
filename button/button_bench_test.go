package button_test

import (
	"context"
	"testing"

	"gioui.org/layout"

	"github.com/reactivego/rx"
	"github.com/vibrantgio/prism/bench"
	"github.com/vibrantgio/prism/button"
	"github.com/vibrantgio/theme/theme"
	"github.com/vibrantgio/theme/tokens"
)

// BenchmarkButtonRender exercises widget(gtx) for b.N synthetic frames via the
// shared bench.BenchFrame harness (DESIGN §"Performance — Methodology"). The
// harness enables b.ReportAllocs so per-frame allocation regressions (>5%
// threshold) are measurable. This is the idle render: default unfocused state.
func BenchmarkButtonRender(b *testing.B) {
	shaper := tokens.DefaultTypography.DeterministicShaper()
	w := button.Render(
		shaper, "Benchmark",
		tokens.DefaultLight, tokens.Spacing, tokens.Radius, tokens.DefaultTypography.LabelLarge, tokens.Comfortable,
		button.RenderState{},
	)
	bench.BenchFrame(b, w)
}

// BenchmarkButtonRenderFocused benchmarks the focused state which draws an
// extra clip.Stroke path for the focus ring.
func BenchmarkButtonRenderFocused(b *testing.B) {
	shaper := tokens.DefaultTypography.DeterministicShaper()
	w := button.Render(
		shaper, "Benchmark",
		tokens.DefaultLight, tokens.Spacing, tokens.Radius, tokens.DefaultTypography.LabelLarge, tokens.Comfortable,
		button.RenderState{Focused: true},
	)
	bench.BenchFrame(b, w)
}

// BenchmarkButtonThemeEmission measures one theme emission end to end: a fresh
// subscription against a fresh default theme, which runs Button's
// SwitchMap/CombineLatest5 map function exactly once. That map function is
// where the component pulls a tokens.Typography *value* out of the rx tuple and
// asks it for the theme's shaper, so this benchmark is the direct measure of
// whether the shaper cache survives that copy (F5.1).
//
// It is deliberately not a BenchFrame: the cost under test is the emission, not
// the paint. Every dark-mode toggle, density change and first subscription in a
// running application pays exactly this.
func BenchmarkButtonThemeEmission(b *testing.B) {
	props := button.Props{Label: "Benchmark"}
	b.ReportAllocs()
	for b.Loop() {
		var w layout.Widget
		err := button.Button(rx.Of(theme.Default()), props).Subscribe(
			context.Background(),
			func(next layout.Widget, _ error, done bool) {
				if !done && next != nil {
					w = next
				}
			},
		).Wait()
		if err != nil {
			b.Fatalf("subscribe: %v", err)
		}
		if w == nil {
			b.Fatal("component did not emit a widget")
		}
	}
}

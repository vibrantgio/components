package richtext_test

import (
	"image"
	"testing"

	"github.com/vibrantgio/prism/bench"
	"github.com/vibrantgio/prism/richtext"
	"github.com/vibrantgio/spectrum/tokens"
)

// benchSize fits the mixed test paragraph wrapped over two lines — a
// representative document paragraph rather than the 300×60 default.
var benchSize = image.Pt(400, 120)

// BenchmarkRichtextRender exercises widget(gtx) for b.N synthetic frames via
// the shared bench.BenchFrame harness (DESIGN §"Performance — Methodology").
// This is the idle render of the canonical mixed paragraph (regular, bold,
// italic, mono, one link): span resolution, per-span shaping (served from the
// shaper's layout cache after the first frame), line wrapping, glyph
// painting, and the link underline.
func BenchmarkRichtextRender(b *testing.B) {
	shaper := tokens.DefaultTypography.DeterministicShaper()
	style := richtext.FromTokens(tokens.DefaultLight, tokens.DefaultTypography.BodyLarge)
	w := richtext.Render(shaper, style, mixedSpans(), richtext.Idle())
	bench.BenchFrame(b, w, bench.WithSize(benchSize))
}

// BenchmarkRichtextRenderFocused benchmarks the focused-link state, which
// additionally draws the focus-ring stroke path.
func BenchmarkRichtextRenderFocused(b *testing.B) {
	shaper := tokens.DefaultTypography.DeterministicShaper()
	style := richtext.FromTokens(tokens.DefaultLight, tokens.DefaultTypography.BodyLarge)
	w := richtext.Render(shaper, style, mixedSpans(),
		richtext.RenderState{HoveredLink: richtext.NoLink, FocusedLink: 0})
	bench.BenchFrame(b, w, bench.WithSize(benchSize))
}

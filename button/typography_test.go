package button_test

import (
	"image"
	"testing"

	"gioui.org/layout"

	"github.com/vibrantgio/prism/button"
	golden "github.com/vibrantgio/prism/golden"
	"github.com/vibrantgio/spectrum/tokens"
)

// specimen is the string every typography golden draws. It is deliberately
// dull: ASCII only, so the pinned Roboto and Roboto Mono faces both carry
// every rune and no symbol face is involved (F4.2 keeps symbols out of stored
// images). It is also deliberately chosen to make the four properties legible
// — "Il1" and "Wm" separate a proportional face from a monospaced one at a
// glance, and the "g" descender shows where the line box sits.
const specimen = "Il1 Wm gj 018"

// typographyCases are the typography goldens: a baseline in the shipped
// LabelLarge role, then one variant per property the role promises, each
// differing from the baseline in exactly that one field.
//
// Why button.Render is the subject. F3.3 re-cut every static Render signature
// in this repo from a bare size onto a whole tokens.TextStyle, and
// button.Render is the canonical consumer of the result: drawButton reads
// Typeface, Weight, Size and LineHeight off the style and puts all four into
// the shaping call. That re-cut moved zero pixels, because no golden drew
// enough text to notice. These do.
//
// Three of the four properties are here, not four. LineHeight is absent
// because it cannot move a pixel on this path and a golden that pretended
// otherwise would be a lie: gioui.org/text's calculateYOffsets baselines the
// first line at that line's own ascent and only spends the line height on the
// gap to the next one, and widget.Label reports the glyph ink bounds as its
// size. A MaxLines:1 label — which every prism control is — therefore renders
// identically at any LineHeight. The property is pinned where it is
// observable, on a wrapped multi-line label, in cadence/feature.
var typographyCases = []struct {
	name  string
	style tokens.TextStyle
}{
	// LabelLarge as shipped: Roboto, weight 500, 14 dp.
	{"type-label-large", tokens.DefaultTypography.LabelLarge},
	// Typeface only: the same metrics on Roboto Mono. This is the monospace
	// case — Code's face, the newest role's — and the one image where every
	// advance is uniform.
	{"type-typeface-mono", withTypeface(tokens.DefaultTypography.LabelLarge, tokens.DefaultTypography.Code.Typeface)},
	// Weight only: 400 instead of 500. Thinner stems.
	{"type-weight-regular", withWeight(tokens.DefaultTypography.LabelLarge, tokens.WeightRegular)},
	// Size only: 11 dp instead of 14 — LabelSmall's size on LabelLarge's face
	// and weight, so nothing but the scale differs.
	{"type-size-small", withSize(tokens.DefaultTypography.LabelLarge, 11)},
}

func withTypeface(s tokens.TextStyle, face string) tokens.TextStyle {
	s.Typeface = face
	return s
}

func withWeight(s tokens.TextStyle, w int) tokens.TextStyle {
	s.Weight = w
	return s
}

func withSize(s tokens.TextStyle, size float32) tokens.TextStyle {
	s.Size = size
	return s
}

// typographySize is the capture window every typography golden uses. It is
// the same for all of them so the stored images are directly comparable.
var typographySize = image.Pt(300, 60)

// typographyWidget renders the specimen string as a button label in style, on
// the light theme with sharp corners so nothing but the text moves between
// cases.
func typographyWidget(t *testing.T, style tokens.TextStyle) layout.Widget {
	t.Helper()
	return button.Render(
		defaultShaper(t), specimen,
		tokens.DefaultLight, tokens.Spacing, tokens.RadiusScale{}, style, tokens.Comfortable,
		button.RenderState{},
	)
}

// TestTypographyGolden records or diffs one image per typography property.
// Together they are the first goldens in this repo that would fail if a role's
// typeface, weight, size or line height changed.
func TestTypographyGolden(t *testing.T) {
	for _, tc := range typographyCases {
		t.Run(tc.name, func(t *testing.T) {
			golden.Render(t, tc.name, typographySize, typographyWidget(t, tc.style))
		})
	}
}

// TestTypographyPropertiesAreDetectable proves the goldens above are an
// instrument and not decoration: changing exactly one field of the role must
// change pixels. If a property were dropped on the way from tokens.TextStyle
// into the shaping call, its variant would render identically to the baseline
// and this test — not a stale golden — says which one.
func TestTypographyPropertiesAreDetectable(t *testing.T) {
	base := golden.Capture(t, typographySize, typographyWidget(t, tokens.DefaultTypography.LabelLarge))
	if base == nil {
		return // headless unavailable; Capture called t.Skip
	}
	for _, tc := range typographyCases[1:] {
		t.Run(tc.name, func(t *testing.T) {
			img := golden.Capture(t, typographySize, typographyWidget(t, tc.style))
			if img == nil {
				return
			}
			if n := golden.PixelDiff(base, img); n == 0 {
				t.Errorf("%s renders identically to the baseline role; that property never reaches the shaper", tc.name)
			}
		})
	}
}

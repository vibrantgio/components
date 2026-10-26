package button_test

import (
	"image"
	"testing"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"

	"github.com/vibrantgio/components/button"
	golden "github.com/vibrantgio/components/golden"
	"github.com/vibrantgio/theme/tokens"
)

// specimen is the string every typography golden draws. It is deliberately
// dull: ASCII only, so the pinned Roboto and Roboto Mono faces both carry
// every rune and no symbol face is involved. It is also deliberately chosen
// to make the four properties legible
// — "Il1" and "Wm" separate a proportional face from a monospaced one at a
// glance, and the "g" descender shows where the line box sits.
const specimen = "Il1 Wm gj 018"

// typographyCases are the typography goldens: a baseline in the shipped
// SystemFont role, then one variant per property the role promises, each
// differing from the baseline in exactly that one field.
//
// button.Render is the subject because it is the canonical consumer of a
// whole tokens.TextStyle: drawButton reads Typeface, Weight, Size and
// LineHeight off the style and puts all four into the shaping call.
//
// All four properties are exercised here, LineHeight included.
// gioui.org/text's calculateYOffsets baselines the first line at that line's
// own ascent and spends the line height only on the gap to the next one, and
// widget.Label reports the drawn glyph extent as its size — so a MaxLines:1 title
// rendered through widget.Label is identical at any LineHeight. The title
// box must instead be sized as the role's line box via theme/typeset, so the
// property is observable exactly where it is documented to arrive.
var typographyCases = []struct {
	name  string
	style tokens.TextStyle
}{
	// SystemFont as shipped: Roboto, weight 400, 13 dp on a 16 dp line.
	{"type-system-font", tokens.DefaultTypography.SystemFont},
	// Typeface only: the same metrics on Roboto Mono — the monospace case,
	// Code's face — and the one image where every advance is uniform.
	//
	// This is also the case that exposes a hazard in measuring a role's
	// natural line by shaping the empty string: an empty string has no run
	// in the face it names, so the probe falls back to the shaper's default
	// face and reads that face's line, not Roboto Mono's. Measuring the text
	// actually being laid out puts the title box back on the role's line
	// height.
	{"type-typeface-mono", withTypeface(tokens.DefaultTypography.SystemFont, tokens.DefaultTypography.Code.Typeface)},
	// Weight only: 700 instead of 400. Thicker stems.
	{"type-weight-bold", withWeight(tokens.DefaultTypography.SystemFont, tokens.WeightBold)},
	// Size only: 11 dp instead of 13 — SmallSystemFont's size on SystemFont's
	// face and weight, so nothing but the scale differs.
	{"type-size-small", withSize(tokens.DefaultTypography.SystemFont, 11)},
	// Line height only: 32 dp instead of 16, at the same 13 dp size. The
	// glyphs are identical to the baseline's — same face, same weight, same
	// scale — and everything that moves moves because the line box grew: the
	// title box is 32 px rather than 16, so the button is 32 + 2×PaddingY
	// tall rather than the density's floor, and the text sits lower by the
	// extra leading above it.
	{"type-line-height-loose", withLineHeight(tokens.DefaultTypography.SystemFont, 32)},
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

func withLineHeight(s tokens.TextStyle, lh float32) tokens.TextStyle {
	s.LineHeight = lh
	return s
}

// typographySize is the capture window every typography golden uses. It is
// the same for all of them so the stored images are directly comparable.
var typographySize = image.Pt(300, 60)

// typographyWidget renders the specimen string as a button title in style, on
// the light theme with sharp corners so nothing but the text moves between
// cases.
func typographyWidget(t *testing.T, style tokens.TextStyle) layout.Widget {
	t.Helper()
	return button.Render(
		defaultShaper(t), specimen,
		tokens.PlatformLight, tokens.Spacing, tokens.RadiusScale{}, style, tokens.Comfortable,
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

// TestLineHeightSizesTheTitleBox is the numeric half of the line-height
// constraint: gioui.org/widget.Label reports glyph ink as its size, and
// gioui.org/text spends the line height only on the gap to a next line that
// a MaxLines:1 title never has, so a naive title box does not move with
// LineHeight at all. The title box must instead be sized as the role's line
// box, so the button's height is the line height plus its padding — and
// every number below is derived from the tokens rather than from the
// letters in the title.
//
// The golden above shows the same fact in pixels; this one says which number
// is wrong when it moves.
func TestLineHeightSizesTheTitleBox(t *testing.T) {
	shaper := defaultShaper(t)
	d := tokens.Comfortable

	var heights []int
	for _, lh := range []float32{20, 32, 64} {
		var ops op.Ops
		gtx := layout.Context{
			Metric:      unit.Metric{PxPerDp: 1, PxPerSp: 1},
			Constraints: layout.Exact(image.Pt(300, 200)),
			Ops:         &ops,
		}
		style := withLineHeight(tokens.DefaultTypography.SystemFont, lh)
		dims := button.Render(
			shaper, specimen,
			tokens.PlatformLight, tokens.Spacing, tokens.Radius, style, d,
			button.RenderState{},
		)(gtx)

		want := int(lh + 2*d.PaddingY)
		if f := int(d.ControlHeight); want < f {
			want = f
		}
		if dims.Size.Y != want {
			t.Errorf("line height %v: button %d px tall, want %d (line box + 2×PaddingY, floored at ControlHeight)",
				lh, dims.Size.Y, want)
		}
		heights = append(heights, dims.Size.Y)
	}

	for i, h := range heights[1:] {
		if h == heights[i] {
			t.Fatalf("button heights %v: two line heights render the same box, so the role's line height reaches the shaper and changes nothing", heights)
		}
	}
}

// TestTypographyPropertiesAreDetectable proves the goldens above are an
// instrument and not decoration: changing exactly one field of the role must
// change pixels. If a property were dropped on the way from tokens.TextStyle
// into the shaping call, its variant would render identically to the baseline
// and this test — not a stale golden — says which one.
func TestTypographyPropertiesAreDetectable(t *testing.T) {
	base := golden.Capture(t, typographySize, typographyWidget(t, tokens.DefaultTypography.SystemFont))
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

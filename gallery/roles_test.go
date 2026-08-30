package main

import (
	"image"
	stdcolor "image/color"
	"math"
	"testing"

	"github.com/vibrantgio/components/golden"
	vgcolor "github.com/vibrantgio/theme/color"
	"github.com/vibrantgio/theme/tokens"
)

// The role swatches, restated. The page draws them from a table of its own,
// and this is that table written down a second time on purpose: a test that
// asked the page what it draws could only ever agree with it. Here each chip
// is named with the pair the tokens prescribe for it, and the render is
// measured against that.
//
// Order matters — the chips are read out of the capture by position — so a
// row reordered in the page and not here fails loudly rather than silently
// measuring the wrong chip.
func roleChips(c tokens.ColorTokens) []struct {
	name     string
	fill, on stdcolor.NRGBA
} {
	return []struct {
		name     string
		fill, on stdcolor.NRGBA
	}{
		{"Primary", c.Primary, c.OnPrimary},
		{"Secondary", c.Secondary, c.OnSecondary},
		{"Tertiary", c.Tertiary, c.OnTertiary},
		{"Info", c.Info, c.OnInfo},
		{"Success", c.Success, c.OnSuccess},
		{"Warning", c.Warning, c.OnWarning},
		{"Error", c.Error, c.OnError},
		{"Background", c.Background, c.Text},
		{"Surface", c.Surface, c.Text},
		{"Divider", c.Divider, c.Text},
		{"Inverse", c.InverseSurface, c.OnInverseSurface},
	}
}

// chipH is the height of a role chip, and the height the row is captured at
// so that nothing under the chips — their names — is in the frame. The
// chips' width and spacing are not assumed: a column is as wide as the wider
// of its chip and its name, so "Background" makes a wider column than
// "Error" does, and a fixed pitch would walk off the row. They are found in
// the capture instead (see chipRuns).
const chipH = 40

// inkCoverageTolerance is how far from the ink a role names the darkest (or
// lightest) pixel of its chip's label may land, as a fraction of the
// distance from the chip's fill to that ink. Some slack is needed because a
// label is antialiased: a 13sp glyph's stems are about a pixel wide, so even
// the pixel most covered by ink is a blend and lands a little short. It is a
// tolerance rather than a floor because both directions are defects — a chip
// drawing a weaker ink than its role names is the one this gate was written
// for, and a chip drawing a stronger one is painting a pairing the tokens do
// not prescribe just the same. Every chip in both schemes measures 0.99 of
// the way.
const inkCoverageTolerance = 0.15

// TestRoleSwatchesPaintTheirTokenPairs measures the roles row off the
// rendered page, in both schemes: every chip's fill is the colour its role
// names, the label on it is the ink that role names, and the two measure at
// or above WCAG AA for body text.
//
// It is a render-side gate and not a token-side one: a token gate checks the
// derivation, not what the page painted, so it cannot confirm or refute a
// chip drawn with the wrong ink. The question is asked of the pixels
// instead: the fill is compared to the token exactly, and the label's
// most-covered pixel is measured for how far it travelled from that fill
// toward the token's ink. A chip painting an ink its role does not name
// lands short of the floor; a chip painting the right ink lands within a
// pixel's antialiasing of it, whichever way round the scheme puts the pair.
func TestRoleSwatchesPaintTheirTokenPairs(t *testing.T) {
	for _, sc := range schemes() {
		sc := sc
		t.Run(sc.name, func(t *testing.T) {
			inv := testInventory(t)
			row := inv.Foundations(sc.colors)[0]
			chips := roleChips(sc.colors)
			img := golden.Capture(t, image.Pt(pageWidth, chipH),
				ground(sc.colors, row.Body))
			runs := chipRuns(img, sc.colors.Ramps.Neutral.Step(400))
			if len(runs) != len(chips) {
				t.Fatalf("the row drew %d chips, want %d — the roles row and this test disagree about what it shows",
					len(runs), len(chips))
			}
			for i, chip := range chips {
				// The chip interior, inside the hairline border.
				at := image.Rect(runs[i][0]+1, 1, runs[i][1], chipH-1)
				fill, ink, coverage := chipInk(img, at, chip.on)
				ratio := vgcolor.ContrastRatio(ink, fill)
				t.Logf("%s: fill %v, label reaches %v (%.2f of the way to the token's %v), %.2f:1",
					chip.name, fill, ink, coverage, chip.on, ratio)
				if fill != chip.fill {
					t.Errorf("%s: the chip is filled %v, want the role's %v", chip.name, fill, chip.fill)
				}
				if math.Abs(coverage-1) > inkCoverageTolerance {
					t.Errorf("%s: the label's most-covered pixel %v lands %.2f of the way from the fill %v to the ink %v the role names — the chip is not painting its own on-colour",
						chip.name, ink, coverage, fill, chip.on)
				}
				if ratio < wcagAA {
					t.Errorf("%s: the label measures %.2f:1 on the chip, under the %.1f:1 floor",
						chip.name, ratio, wcagAA)
				}
			}
		})
	}
}

// wcagAA is WCAG 2's body-text ratio, the floor a label on a swatch has to
// reach.
const wcagAA = 4.5

// chipRuns finds the chips in a capture of the roles row by their top edge:
// every chip is drawn with a hairline border in the neutral ramp's step 400,
// so the top row of the capture is a run of that colour per chip with the
// page's own ground between them. It returns each run as a half-open x
// range. Reading the row rather than assuming a pitch is what lets the test
// measure the chip it names when a long role name widens its column.
func chipRuns(img *image.RGBA, edge stdcolor.NRGBA) [][2]int {
	var runs [][2]int
	in := false
	for x := img.Bounds().Min.X; x < img.Bounds().Max.X; x++ {
		c := img.RGBAAt(x, 0)
		at := stdcolor.NRGBA{c.R, c.G, c.B, 0xff} == edge
		switch {
		case at && !in:
			runs = append(runs, [2]int{x, x + 1})
			in = true
		case at:
			runs[len(runs)-1][1] = x + 1
		default:
			in = false
		}
	}
	return runs
}

// chipInk reads one chip out of a capture: the colour it is filled with —
// the one most of it is — the pixel of the label most covered by ink, and
// how far that pixel travelled from the fill toward want, as a fraction of
// the whole distance. Coverage is measured on relative luminance rather than
// per channel, because that is what makes a label readable and what the
// ratio beside it is computed from.
func chipInk(img *image.RGBA, at image.Rectangle, want stdcolor.NRGBA) (fill, ink stdcolor.NRGBA, coverage float64) {
	counts := map[stdcolor.NRGBA]int{}
	for y := at.Min.Y; y < at.Max.Y; y++ {
		for x := at.Min.X; x < at.Max.X; x++ {
			c := img.RGBAAt(x, y)
			counts[stdcolor.NRGBA{c.R, c.G, c.B, 0xff}]++
		}
	}
	best := 0
	for c, n := range counts {
		// Ties go to the darker colour so the reading is deterministic;
		// a fill is thousands of pixels and a tie cannot arise in practice.
		if n > best || (n == best && vgcolor.RelativeLuminance(c) < vgcolor.RelativeLuminance(fill)) {
			fill, best = c, n
		}
	}
	fl := vgcolor.RelativeLuminance(fill)
	span := vgcolor.RelativeLuminance(want) - fl
	ink = fill
	for c := range counts {
		if math.Abs(vgcolor.RelativeLuminance(c)-fl) > math.Abs(vgcolor.RelativeLuminance(ink)-fl) {
			ink = c
		}
	}
	if span == 0 {
		return fill, ink, 1
	}
	return fill, ink, (vgcolor.RelativeLuminance(ink) - fl) / span
}

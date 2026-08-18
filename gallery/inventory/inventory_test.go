package inventory

import (
	"image"
	"image/color"
	"testing"

	"github.com/vibrantgio/components/golden"
	"github.com/vibrantgio/theme/tokens"
)

// sampleWidth is the width a section is captured at here — wide enough that
// the shell's three columns and the pricing tiers lay out in a shape a window
// would actually show them in.
const sampleWidth = 900

// Two brand colours far enough apart that nothing derived from one survives
// the other: opposite sides of the hue circle, both vivid.
var (
	seedA = color.NRGBA{R: 0x1e, G: 0x66, B: 0xff, A: 0xff}
	seedB = color.NRGBA{R: 0xff, G: 0x6a, B: 0x00, A: 0xff}
)

// testInventory builds the inventory a test draws from, with the shaper
// resolving no system fonts and the control marks pinned to one platform, so
// the same bytes come out on any machine.
func testInventory(t *testing.T) *Inventory {
	t.Helper()
	return NewForOS(tokens.DefaultTypography.DeterministicShaper(), "darwin")
}

// shot captures one section's slot exactly as the column lays it out: the
// scheme's ground, the section's own height, and nothing else on it.
func shot(t *testing.T, c tokens.ColorTokens, s Section) *image.RGBA {
	t.Helper()
	return golden.Capture(t, image.Pt(sampleWidth, int(s.Height)+40), sectionBody(c, s))
}

// changed reports what share of a section's pixels moved between two
// palettes, as a percentage.
func changed(a, b *image.RGBA) float64 {
	bounds := a.Bounds()
	return 100 * float64(golden.PixelDiff(a, b)) / float64(bounds.Dx()*bounds.Dy())
}

// TestNoSectionIsPinnedToAScheme is the standing hunt for a surface that
// draws itself out of something other than the tokens it was handed.
//
// Light and dark of one seed invert the neutral ramps every section stands
// on, so a section that follows its tokens cannot come out looking similar in
// the two. One that does — a popup drawn from a default palette, a fill
// remembered from a previous frame — is the defect this looks for, and it is
// invisible on a page where everything around it changed correctly.
func TestNoSectionIsPinnedToAScheme(t *testing.T) {
	inv := testInventory(t)
	light, dark := tokens.FromSeed(seedA)
	lit, drk := inv.Groups(light), inv.Groups(dark)
	for g := range lit {
		for i := range lit[g].Sections {
			s := lit[g].Sections[i]
			t.Run(s.Name, func(t *testing.T) {
				pct := changed(shot(t, light, s), shot(t, dark, drk[g].Sections[i]))
				t.Logf("%s: %.3f%% of the slot follows the scheme", s.Name, pct)
				if pct < schemeFloor {
					t.Errorf("%s changed %.1f%% between light and dark, want at least %.0f%% — part of it is drawn from something other than the tokens it was handed",
						s.Name, pct, schemeFloor)
				}
			})
		}
	}
}

// schemeFloor is the share of a section's slot that must move when the scheme
// flips. Every section measures over 99.98% today — the ground itself
// inverts, so all a section leaves standing is the odd antialiased pixel
// where two edges happen to blend to the same byte — and the floor is set
// just under the whole slot rather than at some cautious fraction. A fraction
// is what lets a pinned surface hide: the recorded case was a popup drawn
// light on a dark page, and its panel is a fifth of the slot it sits in, so
// anything below about 80% would have called it well themed.
const schemeFloor = 99.5

// TestTheSeedReachesEveryGroup: a brand colour is meant to reach all four
// groups, not only the palette swatches at the top. Each group is captured
// with its section bodies alone — no banner, whose fill is the seed's primary
// and would carry the whole assertion by itself.
func TestTheSeedReachesEveryGroup(t *testing.T) {
	inv := testInventory(t)
	a, _ := tokens.FromSeed(seedA)
	b, _ := tokens.FromSeed(seedB)
	for g, grp := range inv.Groups(a) {
		other := inv.Groups(b)[g]
		t.Run(grp.Name, func(t *testing.T) {
			moved := 0.0
			for i, s := range grp.Sections {
				moved += changed(shot(t, a, s), shot(t, b, other.Sections[i]))
			}
			t.Logf("%s: %.1f%% summed across %d sections", grp.Name, moved, len(grp.Sections))
			if moved == 0 {
				t.Errorf("group %q drew identically under two brand colours — the seed does not reach it", grp.Name)
			}
		})
	}
}

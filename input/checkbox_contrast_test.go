// The checkbox's two pairings, measured rather than eyeballed: the edge an
// unchecked box draws against the ground it stands on, and the check mark
// against the fill it is drawn on. Both are internal because both are
// derivations rather than fields — checkboxBorder walks the neutral ramp, and
// what is worth holding is the ratio it lands on, not the rung.
package input

import (
	"image/color"
	"testing"

	themecolor "github.com/vibrantgio/theme/color"
	"github.com/vibrantgio/theme/tokens"
)

// checkboxSeeds is the spread the pairings are held over, because a palette
// is generated and the defaults are only one of its outputs: the default
// seed, the six saturated corners of sRGB, a seed with no chroma at all, and
// the two ends of the lightness range.
var checkboxSeeds = []color.NRGBA{
	{R: 0x6c, G: 0x3a, B: 0xd4, A: 0xff}, // the default seed
	{R: 0xff, A: 0xff},
	{G: 0xff, A: 0xff},
	{B: 0xff, A: 0xff},
	{R: 0xff, G: 0xff, A: 0xff},
	{G: 0xff, B: 0xff, A: 0xff},
	{R: 0xff, G: 0x80, A: 0xff},
	{R: 0x80, G: 0x80, B: 0x80, A: 0xff},
	{R: 0xff, G: 0xff, B: 0xff, A: 0xff},
	{A: 0xff},
}

// TestCheckboxBorderClearsTheGraphicFloor is the defect this derivation
// replaced, written down. The border was neutral step 500 in both schemes,
// and the neutral ramps are paired — light and dark are realized at the same
// perceptual depths from opposite ends — so step 500 is the one rung that
// barely moves while the ground under it moves the whole way: 6.63:1 against
// the dark background and 2.67:1 against the light one, under WCAG 1.4.11's
// floor in the scheme most people read in. Nothing in that line of code
// looked scheme-dependent, which is exactly why the light half went
// unnoticed.
//
// Two grounds are measured, because the border has two sides. Outside it is
// the level-0 plane, the ground the component is guaranteed against; inside
// it is the box's own Surface interior, which the component paints itself
// and therefore always knows.
func TestCheckboxBorderClearsTheGraphicFloor(t *testing.T) {
	for _, sc := range []struct {
		name   string
		colors tokens.ColorTokens
	}{
		{"light", tokens.DefaultLight},
		{"dark", tokens.DefaultDark},
	} {
		t.Run(sc.name, func(t *testing.T) {
			c := sc.colors
			border := checkboxBorder(c)
			for _, g := range []struct {
				name   string
				ground color.NRGBA
			}{
				{"the level-0 ground it stands on", c.SurfaceAt(tokens.Level0)},
				{"its own Surface interior", c.Surface},
			} {
				got := themecolor.ContrastRatio(border, g.ground)
				t.Logf("border %s against %s %s: %.2f:1", hex(border), g.name, hex(g.ground), got)
				if got < graphicFloor {
					t.Errorf("border %s against %s %s = %.2f:1, want at least %.1f:1",
						hex(border), g.name, hex(g.ground), got, graphicFloor)
				}
			}
		})
	}
}

// TestCheckboxBorderClearsTheFloorForEverySeed walks the same pairing over a
// spread of seeds and both contrast variants. The neutral ramps carry the
// seed's tint, so the measurements move a little from seed to seed; what may
// not move is the verdict.
func TestCheckboxBorderClearsTheFloorForEverySeed(t *testing.T) {
	worst := 99.0
	for _, seed := range checkboxSeeds {
		light, dark := tokens.FromSeed(seed)
		lightHC, darkHC := tokens.FromSeedHighContrast(seed)
		for _, sc := range []struct {
			name   string
			colors tokens.ColorTokens
		}{
			{"light", light},
			{"dark", dark},
			{"light high-contrast", lightHC},
			{"dark high-contrast", darkHC},
		} {
			c := sc.colors
			border := checkboxBorder(c)
			for _, g := range []struct {
				name   string
				ground color.NRGBA
			}{
				{"level-0 ground", c.SurfaceAt(tokens.Level0)},
				{"Surface interior", c.Surface},
			} {
				got := themecolor.ContrastRatio(border, g.ground)
				if got < worst {
					worst = got
				}
				if got < graphicFloor {
					t.Errorf("seed %s %s: border %s against the %s %s = %.2f:1, want at least %.1f:1",
						hex(seed), sc.name, hex(border), g.name, hex(g.ground), got, graphicFloor)
				}
			}
		}
	}
	t.Logf("worst border pairing over the sweep: %.2f:1", worst)
}

// TestCheckboxCheckClearsTheGraphicFloor measures the other pairing: the
// check mark against the Primary fill it is drawn on. The mark takes
// OnPrimary — the on-colour the pin is derived with, which is what
// components/button's filled label and patterns/tag's filled register both
// take over the same fill — so a checkbox agrees with every other thing in
// the library that puts a mark on the accent.
func TestCheckboxCheckClearsTheGraphicFloor(t *testing.T) {
	for _, sc := range []struct {
		name   string
		colors tokens.ColorTokens
	}{
		{"light", tokens.DefaultLight},
		{"dark", tokens.DefaultDark},
	} {
		t.Run(sc.name, func(t *testing.T) {
			c := sc.colors
			got := themecolor.ContrastRatio(c.OnPrimary, c.Primary)
			t.Logf("check %s on the fill %s: %.2f:1", hex(c.OnPrimary), hex(c.Primary), got)
			if got < graphicFloor {
				t.Errorf("check %s on the fill %s = %.2f:1, want at least %.1f:1",
					hex(c.OnPrimary), hex(c.Primary), got, graphicFloor)
			}
		})
	}

	worst := 99.0
	for _, seed := range checkboxSeeds {
		light, dark := tokens.FromSeed(seed)
		lightHC, darkHC := tokens.FromSeedHighContrast(seed)
		for _, c := range []tokens.ColorTokens{light, dark, lightHC, darkHC} {
			got := themecolor.ContrastRatio(c.OnPrimary, c.Primary)
			if got < worst {
				worst = got
			}
			if got < graphicFloor {
				t.Errorf("seed %s: check %s on the fill %s = %.2f:1, want at least %.1f:1",
					hex(seed), hex(c.OnPrimary), hex(c.Primary), got, graphicFloor)
			}
		}
	}
	t.Logf("worst check pairing over the sweep: %.2f:1", worst)
}

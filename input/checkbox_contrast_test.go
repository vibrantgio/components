// The control row's two pairings, measured rather than eyeballed: the edge a
// resting control draws against the surface it stands on, and the check mark
// against the fill it is drawn on. Both are internal because both are
// derivations rather than fields — controlBorder walks the neutral ramp, and
// what is worth holding is the ratio it lands on, not the step.
//
// One border sweep covers four controls. The unchecked box, the unselected
// radio, the text field and the dropdown trigger all take their resting edge
// from controlBorder against a Level in the same four-level vocabulary, so
// measuring the walk over every level measures every placement any of them
// admits; nothing is left for a per-component copy of this test to find.
package input

import (
	"fmt"
	"image/color"
	"testing"

	themecolor "github.com/vibrantgio/theme/color"
	"github.com/vibrantgio/theme/tokens"
)

func hex(c color.NRGBA) string {
	return fmt.Sprintf("#%02x%02x%02x", c.R, c.G, c.B)
}

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

// TestControlBorderClearsTheGraphicFloor asserts controlBorder clears WCAG
// 1.4.11 against every level a control can be handed, not merely the window's
// own surface: a border derivation that only clears the floor against level 0
// can still fail against a level-2 or level-3 host, so the sweep runs every
// level and derives the border against each in turn.
//
// Two surfaces are measured per level, because the border has two sides.
// Outside it is the surface the control stands on; inside it is the control's
// own interior (controlFill, one step above that same surface), which the
// component paints itself and which no surface the control is handed can
// excuse it from clearing. Both sides move together as the surface changes,
// so both are asked rather than one being assumed to stand still.
func TestControlBorderClearsTheGraphicFloor(t *testing.T) {
	for _, sc := range []struct {
		name   string
		colors tokens.ColorTokens
	}{
		{"light", tokens.DefaultLight},
		{"dark", tokens.DefaultDark},
	} {
		t.Run(sc.name, func(t *testing.T) {
			c := sc.colors
			for _, level := range controlLevels {
				border := controlBorder(c, level.level)
				for _, g := range []struct {
					name    string
					surface color.NRGBA
				}{
					{"the " + level.name + " surface it stands on", c.SurfaceAt(level.level)},
					{"its own raised interior", controlFill(c, level.level)},
				} {
					got := themecolor.ContrastRatio(border, g.surface)
					t.Logf("%s border %s against %s %s: %.2f:1", level.name, hex(border), g.name, hex(g.surface), got)
					if got < graphicFloor {
						t.Errorf("%s border %s against %s %s = %.2f:1, want at least %.1f:1",
							level.name, hex(border), g.name, hex(g.surface), got, graphicFloor)
					}
				}
			}
		})
	}
}

// controlLevels is every level, which is the whole set of surfaces a control
// can be handed: every Level field in this package — CheckboxRenderState's,
// RadioRenderState's, RenderState's, DropdownRenderState's — names a
// tokens.ElevationLevel and there are exactly four, so a sweep over these four
// is a sweep over every placement those fields admit.
var controlLevels = []struct {
	name  string
	level tokens.ElevationLevel
}{
	{"level-0", tokens.Level0},
	{"level-1", tokens.Level1},
	{"level-2", tokens.Level2},
	{"level-3", tokens.Level3},
}

// TestControlBorderClearsTheFloorForEverySeed walks the same pairings, on
// every level, over a spread of seeds and both contrast variants. The
// neutral ramps carry the seed's tint, so the measurements move a little from
// seed to seed; what may not move is the verdict.
func TestControlBorderClearsTheFloorForEverySeed(t *testing.T) {
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
			for _, level := range controlLevels {
				border := controlBorder(c, level.level)
				for _, g := range []struct {
					name    string
					surface color.NRGBA
				}{
					{level.name + " surface", c.SurfaceAt(level.level)},
					{"raised interior", controlFill(c, level.level)},
				} {
					got := themecolor.ContrastRatio(border, g.surface)
					if got < worst {
						worst = got
					}
					if got < graphicFloor {
						t.Errorf("seed %s %s: %s border %s against the %s %s = %.2f:1, want at least %.1f:1",
							hex(seed), sc.name, level.name, hex(border), g.name, hex(g.surface), got, graphicFloor)
					}
				}
			}
		}
	}
	t.Logf("worst border pairing over the sweep: %.2f:1", worst)
}

// TestCheckboxCheckClearsTheGraphicFloor measures the other pairing: the
// check mark against the Primary fill it is drawn on. The mark takes
// OnPrimary — the on-colour the pin is derived with, which is what
// components/button's filled label takes over the same fill — so a checkbox
// agrees with every other thing in the library that puts a mark on the accent.
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

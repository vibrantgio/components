// An internal test: it measures the colour pairing optionRowColors returns
// rather than the pixels it ends up as, and that function is unexported on
// purpose — a row's fill and its ink are chosen together, and the pair is what
// is worth holding.
package picker

import (
	"fmt"
	"image/color"
	"testing"

	themecolor "github.com/vibrantgio/theme/color"
	"github.com/vibrantgio/theme/tokens"
)

// wcagText is WCAG 1.4.3's AA floor for text below 18 pt: an option row's
// label is BodyLarge, so it owes this over whatever ground its row paints.
const wcagText = 4.5

// wcagIndicator is WCAG 1.4.11's floor for a non-text indicator — here the
// difference in fill that says which row is the selected one.
const wcagIndicator = 3.0

func hex(c color.NRGBA) string {
	return fmt.Sprintf("#%02x%02x%02x", c.R, c.G, c.B)
}

// TestMenuOptionRowContrast measures all three option-row pairings in both
// schemes and records the numbers in the test log.
//
// The coloured rows are the pairings that keep being lost, and always the same
// way: a highlight is picked, and the ink on it is not picked with it. A
// neutral state walk past the menu's own level-3 ground under the scheme's
// body text read 4.27:1 in the light scheme — a mid-grey ground is where no
// neutral ink reaches the text floor at all — and 1.75:1 in the dark, where
// the walk runs toward the LIGHT end and the ink does not move. Asking each
// ramp for the rung that clears its own ground is what cannot make that
// mistake, and this test is the pair being measured as a pair.
func TestMenuOptionRowContrast(t *testing.T) {
	for _, sc := range []struct {
		name   string
		colors tokens.ColorTokens
	}{
		{"light", tokens.DefaultLight},
		{"dark", tokens.DefaultDark},
	} {
		t.Run(sc.name, func(t *testing.T) {
			restFill, restInk := optionRowColors(sc.colors, false, false)
			hovFill, hovInk := optionRowColors(sc.colors, false, true)
			selFill, selInk := optionRowColors(sc.colors, true, false)

			for _, row := range []struct {
				name string
				fill color.NRGBA
				ink  color.NRGBA
			}{
				{"unselected", restFill, restInk},
				{"hovered", hovFill, hovInk},
				{"selected", selFill, selInk},
			} {
				got := themecolor.ContrastRatio(row.ink, row.fill)
				t.Logf("%s label on its own row %.2f:1 (fill %s, ink %s)", row.name, got, hex(row.fill), hex(row.ink))
				if got < wcagText {
					t.Errorf("%s label on its own row = %.2f:1, want at least %.1f:1", row.name, got, wcagText)
				}
			}

			// A row that reads is only half of it: the selected row has to
			// stay the one that looks selected, which is its fill against
			// the fill every other row paints — and against the wash the
			// pointer leaves, which is the same accent family and must not
			// be mistakable for the answer.
			for _, sep := range []struct {
				name   string
				ground color.NRGBA
			}{
				{"the menu's own fill", restFill},
				{"a hovered row", hovFill},
			} {
				got := themecolor.ContrastRatio(selFill, sep.ground)
				t.Logf("selected fill against %s %.2f:1", sep.name, got)
				if got < wcagIndicator {
					t.Errorf("selected fill against %s = %.2f:1, want at least %.1f:1", sep.name, got, wcagIndicator)
				}
			}

			// Hover owes no separation floor of its own: it is not a mark,
			// it says nothing the pointer does not already say, and it is
			// gone the moment the pointer is. The number is logged because
			// a wash nobody can see is still worth knowing about.
			t.Logf("hovered fill against the menu's own fill %.2f:1 (no floor: hover is not a mark)",
				themecolor.ContrastRatio(hovFill, restFill))
		})
	}
}

// TestMenuOptionRowContrastHoldsForEverySeed walks the pairing over a
// spread of seeds, because a palette is generated and the defaults are only
// one of its outputs. The neutral ramps carry the seed's tint, so the
// measurements move a little from seed to seed; what may not move is the
// verdict.
func TestMenuOptionRowContrastHoldsForEverySeed(t *testing.T) {
	seeds := []color.NRGBA{
		{R: 0x6c, G: 0x3a, B: 0xd4, A: 0xff}, // the default seed
		{R: 0xff, G: 0x00, B: 0x00, A: 0xff},
		{R: 0x00, G: 0xff, B: 0x00, A: 0xff},
		{R: 0x00, G: 0x00, B: 0xff, A: 0xff},
		{R: 0xff, G: 0xff, B: 0x00, A: 0xff},
		{R: 0x00, G: 0xff, B: 0xff, A: 0xff},
		{R: 0xff, G: 0x80, B: 0x00, A: 0xff},
		{R: 0x80, G: 0x80, B: 0x80, A: 0xff}, // a seed with no chroma at all
	}
	worstText, worstSep := 99.0, 99.0
	for _, seed := range seeds {
		light, dark := tokens.FromSeed(seed)
		for _, sc := range []struct {
			name   string
			colors tokens.ColorTokens
		}{
			{"light", light},
			{"dark", dark},
		} {
			restFill, restInk := optionRowColors(sc.colors, false, false)
			hovFill, hovInk := optionRowColors(sc.colors, false, true)
			selFill, selInk := optionRowColors(sc.colors, true, false)
			for _, row := range []struct {
				name string
				fill color.NRGBA
				ink  color.NRGBA
			}{
				{"unselected", restFill, restInk},
				{"hovered", hovFill, hovInk},
				{"selected", selFill, selInk},
			} {
				got := themecolor.ContrastRatio(row.ink, row.fill)
				if got < worstText {
					worstText = got
				}
				if got < wcagText {
					t.Errorf("seed %s %s: %s label on its own row = %.2f:1, want at least %.1f:1",
						hex(seed), sc.name, row.name, got, wcagText)
				}
			}
			for _, sep := range []struct {
				name   string
				ground color.NRGBA
			}{
				{"the menu's own fill", restFill},
				{"a hovered row", hovFill},
			} {
				got := themecolor.ContrastRatio(selFill, sep.ground)
				if got < worstSep {
					worstSep = got
				}
				if got < wcagIndicator {
					t.Errorf("seed %s %s: selected fill against %s = %.2f:1, want at least %.1f:1",
						hex(seed), sc.name, sep.name, got, wcagIndicator)
				}
			}
		}
	}
	t.Logf("worst label over its row %.2f:1; worst selected fill against what it stands beside %.2f:1", worstText, worstSep)
}

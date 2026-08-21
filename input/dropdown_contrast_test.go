// This file is the one internal test in the package: it measures the colour
// pairing optionRowColors returns rather than the pixels it ends up as, and
// that function is unexported on purpose — a row's fill and its ink are chosen
// together, and the pair is what is worth holding.
package input

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

func rowTokens(c tokens.ColorTokens) resolvedTokens {
	return resolvedTokens{color: c, elevation: tokens.Elevation}
}

// TestDropdownOptionRowContrast measures both option-row pairings in both
// schemes and records the numbers in the test log.
//
// The selected row is the pairing that was lost. Its highlight flipped with
// the scheme — a neutral state walk two steps past the menu's own level-3
// ground — while its ink stayed the scheme's body text, so the light scheme
// read dark ink on a mid-grey highlight at 4.27:1 and the dark scheme read
// light ink on a light-grey highlight at 1.75:1: a selected row whose label
// all but vanished in the dark. A highlight and its ink are one decision, and
// this is the test that says so.
func TestDropdownOptionRowContrast(t *testing.T) {
	for _, sc := range []struct {
		name   string
		colors tokens.ColorTokens
	}{
		{"light", tokens.DefaultLight},
		{"dark", tokens.DefaultDark},
	} {
		t.Run(sc.name, func(t *testing.T) {
			tok := rowTokens(sc.colors)
			restFill, restInk := optionRowColors(tok, false)
			selFill, selInk := optionRowColors(tok, true)

			for _, row := range []struct {
				name string
				fill color.NRGBA
				ink  color.NRGBA
			}{
				{"unselected", restFill, restInk},
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
			// the fill every other row paints.
			sep := themecolor.ContrastRatio(selFill, restFill)
			t.Logf("selected fill against the menu's own fill %.2f:1", sep)
			if sep < wcagIndicator {
				t.Errorf("selected fill against the menu's own fill = %.2f:1, want at least %.1f:1", sep, wcagIndicator)
			}
		})
	}
}

// TestDropdownOptionRowContrastHoldsForEverySeed walks the pairing over a
// spread of seeds, because a palette is generated and the defaults are only
// one of its outputs. The neutral ramps carry the seed's tint, so the
// measurements move a little from seed to seed; what may not move is the
// verdict.
func TestDropdownOptionRowContrastHoldsForEverySeed(t *testing.T) {
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
			tok := rowTokens(sc.colors)
			restFill, restInk := optionRowColors(tok, false)
			selFill, selInk := optionRowColors(tok, true)
			for _, row := range []struct {
				name string
				fill color.NRGBA
				ink  color.NRGBA
			}{
				{"unselected", restFill, restInk},
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
			sep := themecolor.ContrastRatio(selFill, restFill)
			if sep < worstSep {
				worstSep = sep
			}
			if sep < wcagIndicator {
				t.Errorf("seed %s %s: selected fill against the menu's own fill = %.2f:1, want at least %.1f:1",
					hex(seed), sc.name, sep, wcagIndicator)
			}
		}
	}
	t.Logf("worst label over its row %.2f:1; worst selected fill against the menu %.2f:1", worstText, worstSep)
}

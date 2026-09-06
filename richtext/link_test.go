package richtext_test

import (
	"fmt"
	stdcolor "image/color"
	"math/rand"
	"testing"

	"github.com/vibrantgio/theme/color"
	"github.com/vibrantgio/theme/tokens"

	"github.com/vibrantgio/components/richtext"
)

// linkSweepSeeds is the seed population this package reads its link-colour
// claims against, the same one theme/tokens sweeps its derivation with: the
// default seed, the nine macOS system accents, both ends of the tonal axis,
// three pastels stated at a dark scheme's tone, and four hundred random
// colours from a fixed source.
//
// The three pastels are stated at a dark scheme's tone: a palette published
// for a dark scheme states its accents high on the tonal axis, and a brand
// seeded with one of them derives a light scheme whose primary pin sits a
// whisper off the content — the case a link coloured with the bare pin would
// fail contrast on.
func linkSweepSeeds() []stdcolor.NRGBA {
	rng := rand.New(rand.NewSource(20260818))
	seeds := []stdcolor.NRGBA{
		tokens.DefaultSeed,
		{0xff, 0x3b, 0x30, 0xff}, {0xff, 0x95, 0x00, 0xff}, {0xff, 0xcc, 0x00, 0xff},
		{0x28, 0xcd, 0x41, 0xff}, {0x00, 0x7a, 0xff, 0xff}, {0xaf, 0x52, 0xde, 0xff},
		{0xff, 0x2d, 0x55, 0xff}, {0x8e, 0x8e, 0x93, 0xff}, {0x00, 0x00, 0x00, 0xff},
		{0xff, 0xff, 0xff, 0xff},
		{0x89, 0xb4, 0xfa, 0xff}, {0xcb, 0xa6, 0xf7, 0xff}, {0xa6, 0xe3, 0xa1, 0xff},
	}
	for i := 0; i < 400; i++ {
		seeds = append(seeds, stdcolor.NRGBA{
			uint8(rng.Intn(256)), uint8(rng.Intn(256)), uint8(rng.Intn(256)), 0xff})
	}
	return seeds
}

func linkHex(c stdcolor.NRGBA) string { return fmt.Sprintf("#%02X%02X%02X", c.R, c.G, c.B) }

// TestLinkColorClearsTheTextFloorForEverySeed is the composition-level gate: a
// link in a paragraph is words on a page, so whatever a caller seeds the
// palette with, [richtext.FromTokens] hands back a link colour that reaches
// WCAG AA against the surface the paragraph is set on. It is read over both
// schemes and both derivations, because the seed's depth decides the light
// scheme's answer and nothing decides the dark one.
func TestLinkColorClearsTheTextFloorForEverySeed(t *testing.T) {
	worstLight, worstDark := 99.0, 99.0
	var worstLightAt, worstDarkAt string
	for _, seed := range linkSweepSeeds() {
		light, dark := tokens.FromSeed(seed)
		hcLight, hcDark := tokens.FromSeedHighContrast(seed)
		for _, s := range []struct {
			name  string
			tok   tokens.ColorTokens
			light bool
		}{
			{"FromSeed light", light, true},
			{"FromSeed dark", dark, false},
			{"FromSeedHighContrast light", hcLight, true},
			{"FromSeedHighContrast dark", hcDark, false},
		} {
			style := richtext.FromTokens(s.tok, tokens.DefaultTypography.BodyLarge)
			surface := s.tok.SurfaceAt(tokens.Level0)
			got := color.ContrastRatio(style.LinkColor, surface)
			if got < tokens.TextFloor {
				t.Errorf("seed %s: %s: link colour %s on surface %s measures %.2f:1, under the %.1f:1 text floor",
					linkHex(seed), s.name, linkHex(style.LinkColor), linkHex(surface), got, tokens.TextFloor)
			}
			if s.light && got < worstLight {
				worstLight, worstLightAt = got, linkHex(seed)
			}
			if !s.light && got < worstDark {
				worstDark, worstDarkAt = got, linkHex(seed)
			}
		}
	}
	t.Logf("over %d seeds: worst light link %.2f:1 (%s), worst dark link %.2f:1 (%s)",
		len(linkSweepSeeds()), worstLight, worstLightAt, worstDark, worstDarkAt)
}

// TestTheCanonicalSeedsLinkColorIsThePrimaryPin asserts that deriving the
// link colour via ForegroundOnAtFloor costs no stored golden image: on the
// seed every golden is rendered from, the brand's own colour already clears
// the floor, so it is what the paragraph gets.
func TestTheCanonicalSeedsLinkColorIsThePrimaryPin(t *testing.T) {
	for _, s := range []struct {
		name string
		tok  tokens.ColorTokens
	}{
		{"DefaultLight", tokens.DefaultLight},
		{"DefaultDark", tokens.DefaultDark},
	} {
		style := richtext.FromTokens(s.tok, tokens.DefaultTypography.BodyLarge)
		if style.LinkColor != s.tok.Primary {
			t.Errorf("%s: link colour is %s, not the Primary pin %s — a golden moved",
				s.name, linkHex(style.LinkColor), linkHex(s.tok.Primary))
		}
	}
}

// TestAPastelSeedsLinkColorLeavesThePin covers a light scheme seeded with a
// dark scheme's accent, where the bare primary pin fails the text floor.
func TestAPastelSeedsLinkColorLeavesThePin(t *testing.T) {
	seed := stdcolor.NRGBA{0x89, 0xb4, 0xfa, 0xff}
	light, dark := tokens.FromSeed(seed)

	lightSurface := light.SurfaceAt(tokens.Level0)
	if bare := color.ContrastRatio(light.Primary, lightSurface); bare >= tokens.TextFloor {
		t.Fatalf("this seed's bare light pin now measures %.2f:1 — the test no longer reads the shape it was written for", bare)
	}
	lightLink := richtext.FromTokens(light, tokens.DefaultTypography.BodyLarge).LinkColor
	if lightLink == light.Primary {
		t.Errorf("light link colour is still the bare pin %s", linkHex(light.Primary))
	}

	darkSurface := dark.SurfaceAt(tokens.Level0)
	darkLink := richtext.FromTokens(dark, tokens.DefaultTypography.BodyLarge).LinkColor
	if darkLink != dark.Primary {
		t.Errorf("dark link colour walked to %s; the dark pin %s clears its surface and should stand",
			linkHex(darkLink), linkHex(dark.Primary))
	}
	t.Logf("seed %s: light link %s on %s %.2f:1 (bare pin %s %.2f:1); dark link %s on %s %.2f:1",
		linkHex(seed), linkHex(lightLink), linkHex(lightSurface), color.ContrastRatio(lightLink, lightSurface),
		linkHex(light.Primary), color.ContrastRatio(light.Primary, lightSurface),
		linkHex(darkLink), linkHex(darkSurface), color.ContrastRatio(darkLink, darkSurface))
}

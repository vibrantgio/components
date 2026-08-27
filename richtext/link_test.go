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

// linkSweepSeeds is the seed population this package reads its link-ink
// claims against, the same one theme/tokens sweeps its derivation with: the
// default seed, the nine macOS system accents, both ends of the tonal axis,
// three pastels stated at a dark scheme's tone, and four hundred random
// colours from a fixed source.
//
// The three pastels are the shape that produced the defect this file gates.
// A palette published for a dark scheme states its accents high on the tonal
// axis, and a brand seeded with one of them derives a light scheme whose
// primary pin sits a whisper off the paper — which is exactly what a link
// coloured with the bare pin used to be.
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

// TestLinkInkClearsTheTextFloorForEverySeed is the composition-level gate: a
// link in a paragraph is words on a page, so whatever a caller seeds the
// palette with, [richtext.FromTokens] hands back a link colour that reaches
// WCAG AA against the ground the paragraph is set on. It is read over both
// schemes and both derivations, because the seed's depth decides the light
// scheme's answer and nothing decides the dark one.
func TestLinkInkClearsTheTextFloorForEverySeed(t *testing.T) {
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
			ground := s.tok.SurfaceAt(tokens.Level0)
			got := color.ContrastRatio(style.LinkColor, ground)
			if got < tokens.TextFloor {
				t.Errorf("seed %s: %s: link ink %s on ground %s measures %.2f:1, under the %.1f:1 text floor",
					linkHex(seed), s.name, linkHex(style.LinkColor), linkHex(ground), got, tokens.TextFloor)
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

// TestTheCanonicalSeedsLinkInkIsThePrimaryPin states what this repair costs
// every stored image in the design system, which is nothing: on the seed
// every golden is rendered from, the brand's own colour clears the floor and
// is what the paragraph gets, exactly as before.
func TestTheCanonicalSeedsLinkInkIsThePrimaryPin(t *testing.T) {
	for _, s := range []struct {
		name string
		tok  tokens.ColorTokens
	}{
		{"DefaultLight", tokens.DefaultLight},
		{"DefaultDark", tokens.DefaultDark},
	} {
		style := richtext.FromTokens(s.tok, tokens.DefaultTypography.BodyLarge)
		if style.LinkColor != s.tok.Primary {
			t.Errorf("%s: link ink is %s, not the Primary pin %s — a golden moved",
				s.name, linkHex(style.LinkColor), linkHex(s.tok.Primary))
		}
	}
}

// TestAPastelSeedsLinkInkLeavesThePin is the regression itself, read on the
// shape that produced it: a light scheme seeded with a dark scheme's accent.
// Before the gate this paragraph's links were the bare pin at 1.95:1.
func TestAPastelSeedsLinkInkLeavesThePin(t *testing.T) {
	seed := stdcolor.NRGBA{0x89, 0xb4, 0xfa, 0xff}
	light, dark := tokens.FromSeed(seed)

	lightGround := light.SurfaceAt(tokens.Level0)
	if bare := color.ContrastRatio(light.Primary, lightGround); bare >= tokens.TextFloor {
		t.Fatalf("this seed's bare light pin now measures %.2f:1 — the test no longer reads the shape it was written for", bare)
	}
	lightLink := richtext.FromTokens(light, tokens.DefaultTypography.BodyLarge).LinkColor
	if lightLink == light.Primary {
		t.Errorf("light link ink is still the bare pin %s", linkHex(light.Primary))
	}

	darkGround := dark.SurfaceAt(tokens.Level0)
	darkLink := richtext.FromTokens(dark, tokens.DefaultTypography.BodyLarge).LinkColor
	if darkLink != dark.Primary {
		t.Errorf("dark link ink walked to %s; the dark pin %s clears its ground and should stand",
			linkHex(darkLink), linkHex(dark.Primary))
	}
	t.Logf("seed %s: light link %s on %s %.2f:1 (bare pin %s %.2f:1); dark link %s on %s %.2f:1",
		linkHex(seed), linkHex(lightLink), linkHex(lightGround), color.ContrastRatio(lightLink, lightGround),
		linkHex(light.Primary), color.ContrastRatio(light.Primary, lightGround),
		linkHex(darkLink), linkHex(darkGround), color.ContrastRatio(darkLink, darkGround))
}

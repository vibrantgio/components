package input

// This file is an internal test (package input, not input_test) so it can
// exercise selectedRadioEdge directly, the way
// theme/tokens/foreground_test.go exercises ColorTokens.ForegroundOnAtFloor
// and components/paragraph/link_test.go exercises paragraph.FromTokens's
// LinkColor field. drawRadio has no exported field to read the drawn edge
// colour back off of, so the derivation itself is the seam this file
// measures.

import (
	"fmt"
	stdcolor "image/color"
	"math/rand"
	"testing"

	"github.com/vibrantgio/theme/color"
	"github.com/vibrantgio/theme/tokens"
)

// radioEdgeSweepSeeds is the seed population this file reads the selected
// radio edge's colour claims against, the same one theme/tokens and
// components/paragraph sweep their derivations with: the default seed, the
// nine macOS system accents, both ends of the tonal axis, three pastels
// stated at a dark scheme's tone, and four hundred random colours from a
// fixed source.
//
// The three pastels represent a palette published for a dark scheme that
// states its accents high on the tonal axis: a brand seeded with one of
// them derives a light scheme whose primary pin sits a whisper off its own
// level, which is the shape this sweep needs to exercise.
func radioEdgeSweepSeeds() []stdcolor.NRGBA {
	rng := rand.New(rand.NewSource(20260827))
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
			R: uint8(rng.Intn(256)), G: uint8(rng.Intn(256)), B: uint8(rng.Intn(256)), A: 0xff})
	}
	return seeds
}

func radioEdgeHex(c stdcolor.NRGBA) string { return fmt.Sprintf("#%02X%02X%02X", c.R, c.G, c.B) }

// radioEdgeSweepSchemes yields every palette the sweep reads a seed as:
// both derivations, both schemes.
func radioEdgeSweepSchemes(seed stdcolor.NRGBA) []struct {
	name  string
	tok   tokens.ColorTokens
	light bool
} {
	light, dark := tokens.FromSeed(seed)
	hcLight, hcDark := tokens.FromSeedHighContrast(seed)
	return []struct {
		name  string
		tok   tokens.ColorTokens
		light bool
	}{
		{"FromSeed light", light, true},
		{"FromSeed dark", dark, false},
		{"FromSeedHighContrast light", hcLight, true},
		{"FromSeedHighContrast dark", hcDark, false},
	}
}

// radioEdgeLevels are every level RadioRenderState.Level can name: the
// window's own surface and the three raised fills a host can stand a radio on.
var radioEdgeLevels = []tokens.ElevationLevel{
	tokens.Level0, tokens.Level1, tokens.Level2, tokens.Level3,
}

// TestSelectedRadioEdgeClearsTheGraphicFloorForEverySeed is the site-level
// gate: whatever a caller seeds the palette with, and whatever
// level hosts the radio, a selected radio's edge reaches WCAG 1.4.11
// against that surface's own fill.
func TestSelectedRadioEdgeClearsTheGraphicFloorForEverySeed(t *testing.T) {
	worstLight, worstDark := 99.0, 99.0
	var worstLightAt, worstDarkAt string
	for _, seed := range radioEdgeSweepSeeds() {
		for _, s := range radioEdgeSweepSchemes(seed) {
			for _, level := range radioEdgeLevels {
				host := s.tok.SurfaceAt(level)
				edge := selectedRadioEdge(s.tok, level)
				got := color.ContrastRatio(edge, host)
				if got < tokens.GraphicFloor {
					t.Errorf("seed %s: %s: level %v: selected edge %s on host %s measures %.2f:1, under the %.1f:1 graphic floor",
						radioEdgeHex(seed), s.name, level, radioEdgeHex(edge), radioEdgeHex(host), got, tokens.GraphicFloor)
				}
				if s.light && got < worstLight {
					worstLight, worstLightAt = got, radioEdgeHex(seed)
				}
				if !s.light && got < worstDark {
					worstDark, worstDarkAt = got, radioEdgeHex(seed)
				}
			}
		}
	}
	t.Logf("over %d seeds: worst light selected edge %.2f:1 (%s), worst dark selected edge %.2f:1 (%s)",
		len(radioEdgeSweepSeeds()), worstLight, worstLightAt, worstDark, worstDarkAt)
}

// TestTheCanonicalSeedsSelectedRadioEdgeIsThePrimaryPin asserts that on the
// seed every golden is rendered from, the brand's own colour clears the
// floor on every host level, so a selected radio's edge is exactly the
// Primary pin there.
func TestTheCanonicalSeedsSelectedRadioEdgeIsThePrimaryPin(t *testing.T) {
	for _, s := range []struct {
		name string
		tok  tokens.ColorTokens
	}{
		{"DefaultLight", tokens.DefaultLight},
		{"DefaultDark", tokens.DefaultDark},
	} {
		for _, level := range radioEdgeLevels {
			if edge := selectedRadioEdge(s.tok, level); edge != s.tok.Primary {
				t.Errorf("%s level %v: selected edge is %s, not the Primary pin %s — a golden moved",
					s.name, level, radioEdgeHex(edge), radioEdgeHex(s.tok.Primary))
			}
		}
	}
}

// TestAPastelSeedsSelectedRadioEdgeLeavesThePin exercises a light scheme
// seeded with a dark scheme's accent, where the seed's own primary pin
// falls under the graphic floor against the window level: it asserts that
// selectedRadioEdge does not return the bare pin in that case, while a
// scheme whose primary already clears its host keeps that pin.
func TestAPastelSeedsSelectedRadioEdgeLeavesThePin(t *testing.T) {
	seed := stdcolor.NRGBA{0x89, 0xb4, 0xfa, 0xff}
	light, dark := tokens.FromSeed(seed)

	lightHost := light.SurfaceAt(tokens.Level0)
	if bare := color.ContrastRatio(light.Primary, lightHost); bare >= tokens.GraphicFloor {
		t.Fatalf("this seed's bare light pin now measures %.2f:1 on the window level — the test no longer reads the shape it was written for", bare)
	}
	lightEdge := selectedRadioEdge(light, tokens.Level0)
	if lightEdge == light.Primary {
		t.Errorf("light selected edge is still the bare pin %s", radioEdgeHex(light.Primary))
	}

	darkEdge := selectedRadioEdge(dark, tokens.Level0)
	if darkEdge != dark.Primary {
		t.Errorf("dark selected edge walked to %s; the dark pin %s clears its host and should stand",
			radioEdgeHex(darkEdge), radioEdgeHex(dark.Primary))
	}
}

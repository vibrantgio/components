package focus_test

import (
	stdcolor "image/color"
	"math/rand"
	"testing"

	"github.com/vibrantgio/components/internal/focus"
	"github.com/vibrantgio/theme/color"
	"github.com/vibrantgio/theme/tokens"
)

// sweepSeeds is the seed population the ring rule's whole-palette claims are
// asserted over: the same eleven chosen colours theme's own contrast gates
// use — the default seed, the nine macOS system accents and both ends of the
// tonal axis — plus four hundred random ones from a fixed source, so the run
// is wide and identical every time.
func sweepSeeds() []stdcolor.NRGBA {
	rng := rand.New(rand.NewSource(20260818))
	seeds := []stdcolor.NRGBA{
		tokens.DefaultSeed,
		{0xff, 0x3b, 0x30, 0xff}, {0xff, 0x95, 0x00, 0xff}, {0xff, 0xcc, 0x00, 0xff},
		{0x28, 0xcd, 0x41, 0xff}, {0x00, 0x7a, 0xff, 0xff}, {0xaf, 0x52, 0xde, 0xff},
		{0xff, 0x2d, 0x55, 0xff}, {0x8e, 0x8e, 0x93, 0xff}, {0x00, 0x00, 0x00, 0xff},
		{0xff, 0xff, 0xff, 0xff},
	}
	for i := 0; i < 400; i++ {
		seeds = append(seeds, stdcolor.NRGBA{
			uint8(rng.Intn(256)), uint8(rng.Intn(256)), uint8(rng.Intn(256)), 0xff})
	}
	return seeds
}

// grounds names, for one scheme, every ground a control in this library draws
// its focus ring over — one entry per control and per state that changes what
// the ring circles. Adding a focusable control to the library means adding its
// ground here; that is what makes this file the gate for the whole idiom
// rather than for the four packages that happen to exist.
func grounds(c tokens.ColorTokens) []struct {
	name   string
	ground stdcolor.NRGBA
} {
	return []struct {
		name   string
		ground stdcolor.NRGBA
	}{
		// A button's ring circles its own background, in each register and
		// each interaction state that walks it.
		{"button filled", c.SolidStateColor(tokens.RolePrimary, tokens.StateFocus)},
		{"button filled hovered", c.SolidStateColor(tokens.RolePrimary, tokens.StateHover)},
		{"button filled pressed", c.SolidStateColor(tokens.RolePrimary, tokens.StatePressed)},
		{"button tonal", c.StateColor(tokens.RolePrimary, 200, tokens.StateFocus)},
		{"button tonal hovered", c.StateColor(tokens.RolePrimary, 200, tokens.StateHover)},
		{"button tonal pressed", c.StateColor(tokens.RolePrimary, 200, tokens.StatePressed)},
		// A ghost paints no ground, so its ring circles the host storey's
		// surface — every storey the elevation ladder carries.
		{"button ghost on level 0/1", c.Ramps.Neutral.Step(200)},
		{"button ghost on level 2", c.Ramps.Neutral.Step(tokens.Elevation.SurfaceStep(2))},
		{"button ghost on level 3", c.Ramps.Neutral.Step(tokens.Elevation.SurfaceStep(3))},
		{"button ghost hovered", c.StateColor(tokens.RoleNeutral, 200, tokens.StateHover)},
		// The promoted-border family: the ring is the control's own edge, and
		// the ground it lies on is the surface the field is filled with.
		{"text field", c.Surface},
		{"dropdown trigger", c.Surface},
		// The clear-of-the-glyph family: the ring rides in the surface beside
		// the glyph, in every checked or chosen state, and a link's ring in
		// the paragraph ground it is padded clear into.
		{"checkbox", c.Surface},
		{"radio", c.Surface},
		{"link", c.Surface},
	}
}

// TestRingClearsTheFloorOnEveryGroundItCircles is the whole idiom's gate: for
// every ground any control in this library draws a focus ring over, in both
// schemes, the ring reaches focus.Floor against that ground.
//
// Both schemes and not just the default one, because the two are not each
// other's mirror image: a light scheme's ring walks down its ramp from the
// mid-value rung and a dark scheme's walks up, and the grounds they walk
// against are built by different halves of the derivation.
func TestRingClearsTheFloorOnEveryGroundItCircles(t *testing.T) {
	for _, s := range []struct {
		name string
		tok  tokens.ColorTokens
	}{
		{"light", tokens.DefaultLight},
		{"dark", tokens.DefaultDark},
	} {
		for _, g := range grounds(s.tok) {
			ring := focus.Ring(s.tok, g.ground)
			got := color.ContrastRatio(ring, g.ground)
			if got < focus.Floor {
				t.Errorf("%s: %s: ring %v measures %.2f:1 against %v, under the %.1f:1 floor",
					s.name, g.name, ring, got, g.ground, focus.Floor)
			}
		}
	}
}

// TestRingClearsTheFloorForEverySeed extends the same gate over the seed
// sweep and both derivations, which is where the rule earns its keep. Primary
// is the seed a caller chose, so "focus is Primary" is a claim about a colour
// nobody in this library has seen: over this population bare Primary measures
// under the floor against the light surface for more than half the seeds and
// bottoms out at 1.00:1. Walking the primary ramp to the rung that reads has
// no such gap — and the walk lands on one rung per ground per scheme for
// every seed there is, which is why one ring colour looks like one idiom
// rather than a colour that wanders with the palette.
func TestRingClearsTheFloorForEverySeed(t *testing.T) {
	worstLight, worstDark := 99.0, 99.0
	for _, seed := range sweepSeeds() {
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
			for _, g := range grounds(s.tok) {
				ring := focus.Ring(s.tok, g.ground)
				got := color.ContrastRatio(ring, g.ground)
				if got < focus.Floor {
					t.Errorf("seed %v: %s: %s: ring %v measures %.2f:1 against %v, under the %.1f:1 floor",
						seed, s.name, g.name, ring, got, g.ground, focus.Floor)
				}
				if s.light {
					if got < worstLight {
						worstLight = got
					}
				} else if got < worstDark {
					worstDark = got
				}
			}
		}
	}
	t.Logf("over %d seeds and both derivations: worst light ring %.2f:1, worst dark %.2f:1",
		len(sweepSeeds()), worstLight, worstDark)
}

// TestRingIsARungOfThePrimaryRamp holds the "primary-coloured" half of the
// rule the floor cannot see. A ring that met its floor by reaching for a
// neutral, an inverse surface or an invented colour would satisfy every
// contrast assertion above and would not be the brand's focus ring.
func TestRingIsARungOfThePrimaryRamp(t *testing.T) {
	for _, seed := range sweepSeeds() {
		light, dark := tokens.FromSeed(seed)
		for _, c := range []tokens.ColorTokens{light, dark} {
			for _, g := range grounds(c) {
				ring := focus.Ring(c, g.ground)
				onRamp := false
				for _, rung := range c.Ramps.Primary {
					if rung == ring {
						onRamp = true
						break
					}
				}
				if !onRamp {
					t.Fatalf("seed %v: %s: ring %v is not a rung of the primary ramp", seed, g.name, ring)
				}
			}
		}
	}
}

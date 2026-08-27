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
// its focus ring over — one entry per control and per state or storey that
// changes what the ring circles. Adding a focusable control to the library
// means adding its ground here; that is what makes this file the gate for the
// whole idiom rather than for the four packages that happen to exist.
//
// Most controls appear once per storey rather than once. A control's host is
// a rung of the elevation ladder, the ring lies on that host's fill, and the
// fill moves with the ladder — so "which ground does this control's ring
// circle" has one answer per storey and a gate that asked only the first
// would have passed the level-2 and level-3 readings that were under the
// floor for as long as the ring was measured against a fixed surface. Since
// ADR-022 the ladder carries a storey under the paper as well, so the list
// walks [storeys] rather than naming three of them: a control on a sidebar
// stands on the furniture floor and its ring is measured there too.
func grounds(c tokens.ColorTokens) []struct {
	name   string
	ground stdcolor.NRGBA
} {
	type entry = struct {
		name   string
		ground stdcolor.NRGBA
	}
	out := []entry{
		// A button's ring circles its own background, in each register and
		// each interaction state that walks it.
		{"button filled", c.SolidStateColor(tokens.RolePrimary, tokens.StateFocus)},
		{"button filled hovered", c.SolidStateColor(tokens.RolePrimary, tokens.StateHover)},
		{"button filled pressed", c.SolidStateColor(tokens.RolePrimary, tokens.StatePressed)},
		{"button tonal", c.StateColor(tokens.RolePrimary, 200, tokens.StateFocus)},
		{"button tonal hovered", c.StateColor(tokens.RolePrimary, 200, tokens.StateHover)},
		{"button tonal pressed", c.StateColor(tokens.RolePrimary, 200, tokens.StatePressed)},
		// A link's ring rides in the paragraph ground it is padded clear
		// into. Prose carries no storey of its own, so a paragraph lies on
		// the paper — richtext.FromTokens derives the ring against level 0
		// for exactly that reason.
		{"link", focus.Ground(c, tokens.Level0)},
	}
	for _, level := range storeys {
		at := storeyName(level)
		// A ghost paints no ground, so its ring circles the host storey's
		// surface. Hovered and pressed it paints that storey's own walk and
		// the ring circles the wash instead — since ADR-022 the wash is
		// taken from the storey's fill rather than from a ramp step, so
		// there are as many washes as there are storeys.
		out = append(out,
			entry{"button ghost " + at, focus.Ground(c, level)},
			entry{"button ghost hovered " + at, c.StateAt(level, tokens.StateHover)},
			entry{"button ghost pressed " + at, c.StateAt(level, tokens.StatePressed)},
			// The promoted-border family: the ring is the control's own
			// edge, so the band has the field's fill inside it and the host
			// storey outside. The walk is taken against the storey; the
			// fill is asserted as the neighbour that walk must also satisfy
			// (TestPromotedBorderRingClearsBothItsNeighbours).
			entry{"text field " + at, focus.Ground(c, level)},
			entry{"dropdown trigger " + at, focus.Ground(c, level)},
			// The clear-of-the-glyph family: the ring rides in the host
			// storey's surface beside the glyph, in every checked or chosen
			// state.
			entry{"checkbox " + at, focus.Ground(c, level)},
			entry{"radio " + at, focus.Ground(c, level)},
		)
	}
	return out
}

// storeyName spells a storey the way this file's failure messages want to
// read it, so a report names the rung a developer would put a control on
// rather than an integer that counts from the paper.
func storeyName(level tokens.ElevationLevel) string {
	if level == tokens.LevelFloor {
		return "on the furniture floor"
	}
	return "on level " + string(rune('0'+int(level)))
}

// storeys is the elevation ladder every control in this library can be put
// on, named as a host says it: a control that is told nothing stands on
// tokens.Level0, and a control on a sidebar, a rail or a toolbar stands on
// the furniture floor beneath it (ADR-022).
var storeys = []tokens.ElevationLevel{
	tokens.LevelFloor, tokens.Level0, tokens.Level1, tokens.Level2, tokens.Level3,
}

// TestGroundIsTheStoreysOwnFill holds the resolution [focus.Ground] performs,
// which the contrast gates above cannot see: they would pass just as happily
// if Ground answered one fixed colour for every storey.
//
// Every storey, with no exception written into the rule. Level 0 used to be
// one — its fill is the Background pin, off the neutral ramp, so a resolution
// that answered ramp steps had no step to walk and fell back to Surface. A
// resolution that answers fills has nothing to fall back from, and the floor
// beneath the paper is asked for on the same terms as the storeys above it.
func TestGroundIsTheStoreysOwnFill(t *testing.T) {
	for _, s := range []struct {
		name string
		tok  tokens.ColorTokens
	}{
		{"light", tokens.DefaultLight},
		{"dark", tokens.DefaultDark},
	} {
		for _, level := range []tokens.ElevationLevel{
			tokens.LevelFloor, tokens.Level0, tokens.Level1, tokens.Level2, tokens.Level3,
		} {
			if got, want := focus.Ground(s.tok, level), s.tok.SurfaceAt(level); got != want {
				t.Errorf("%s: Ground(%d) = %v, want the storey's own fill %v", s.name, level, got, want)
			}
		}
	}
}

// TestLevelZeroGroundMovesNoPixel holds the claim the Level 0 case rests on
// now that it has stopped being an exception: the ring a control on the page
// draws is the ring it drew when Ground answered Surface there. Both grounds
// are asked for the rung, over the whole sweep and both derivations, and they
// answer the same rung every time — so the repair is a change of reasoning
// and not a repaint.
func TestLevelZeroGroundMovesNoPixel(t *testing.T) {
	for _, seed := range sweepSeeds() {
		light, dark := tokens.FromSeed(seed)
		hcLight, hcDark := tokens.FromSeedHighContrast(seed)
		for _, c := range []tokens.ColorTokens{light, dark, hcLight, hcDark} {
			if got, want := focus.Ring(c, focus.Ground(c, tokens.Level0)), focus.Ring(c, c.Surface); got != want {
				t.Fatalf("seed %v: level 0 ring %v against the pin, %v against Surface — the pre-storey ground and the paper no longer answer one rung", seed, got, want)
			}
		}
	}
}

// TestPromotedBorderRingClearsBothItsNeighbours holds the claim [focus.Ground]
// rests on for the text field and the dropdown trigger: their ring is the
// control's outermost band, so it has two neighbours — the field's own Surface
// fill on the inside, the host storey on the outside — and one walk has to
// satisfy them both. Deriving against the storey does; deriving against the
// fill did not, which is the defect this replaces.
func TestPromotedBorderRingClearsBothItsNeighbours(t *testing.T) {
	for _, seed := range sweepSeeds() {
		light, dark := tokens.FromSeed(seed)
		hcLight, hcDark := tokens.FromSeedHighContrast(seed)
		for _, c := range []tokens.ColorTokens{light, dark, hcLight, hcDark} {
			for _, level := range storeys {
				ring := focus.Ring(c, focus.Ground(c, level))
				if got := color.ContrastRatio(ring, c.Surface); got < focus.Floor {
					t.Fatalf("seed %v: level %d: ring %v measures %.2f:1 against the field's own fill %v, under the %.1f:1 floor",
						seed, level, ring, got, c.Surface, focus.Floor)
				}
			}
		}
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

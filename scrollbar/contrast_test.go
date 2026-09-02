// The thumb's one pairing, measured over the composite rather than off the
// ramp. An overlay thumb is translucent, so it has no colour until it is
// drawn: what a reader sees is the ink mixed with the ground in linear light,
// and that mix is the only thing a contrast floor can be applied to. The ink's
// own ratio says nothing — the low-contrast-text step at 39% coverage measures
// 6.19:1 as an ink and 1.49:1 composited over the light page.
package scrollbar

import (
	"image/color"
	"testing"

	tcolor "github.com/vibrantgio/theme/color"
	"github.com/vibrantgio/theme/tokens"
)

// thumbSeeds is the spread the pairing is held over, because a palette is
// generated and the defaults are only one of its outputs: the default seed,
// the six saturated corners of sRGB, a seed with no chroma at all, and the
// two ends of the lightness range.
var thumbSeeds = []color.NRGBA{
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

// thumbGrounds are the levels an overlay bar rides: the window's own page in
// a document, and the chrome level the panes are filled at everywhere else. Both are measured for both states, because the derivation
// answers whichever of the two asks more and the other has to come out no
// worse. The chrome level is the harder of the two in both schemes: the
// thumb's ink is dark, and chrome is the darker of the two surfaces.
var thumbGrounds = []struct {
	name  string
	level tokens.ElevationLevel
}{
	{"level 0, the page", tokens.Level0},
	{"the chrome level, a pane", tokens.LevelChrome},
}

// TestTheThumbClearsItsFloorOnEveryGroundItRides asserts the floors where they
// apply — on the composite, in both states and on both grounds. The rest state
// owes WCAG 1.4.11's 3:1 as a graphic that carries meaning without being text;
// the hover and drag state owes 4.5:1, because a thumb the pointer is on is a
// target being aimed at.
func TestTheThumbClearsItsFloorOnEveryGroundItRides(t *testing.T) {
	for _, scheme := range []struct {
		name string
		c    tokens.ColorTokens
	}{{"light", tokens.DefaultLight}, {"dark", tokens.DefaultDark}} {
		s := FromTokens(scheme.c)
		for _, st := range []struct {
			name  string
			ink   color.NRGBA
			floor float64
		}{
			{"rest", s.ThumbColor, restFloor},
			{"hover/drag", s.ThumbHoverColor, activeFloor},
		} {
			for _, g := range thumbGrounds {
				ground := scheme.c.SurfaceAt(g.level)
				composite := tcolor.Over(st.ink, ground)
				got := tcolor.ContrastRatio(composite, ground)
				if got < st.floor {
					t.Errorf("%s %s: %v over %s composites to %v and measures %.2f:1, under the %.1f:1 floor",
						scheme.name, st.name, st.ink, g.name, composite, got, st.floor)
				}
				t.Logf("%s %s: %v over %s -> %v, %.2f:1", scheme.name, st.name, st.ink, g.name, composite, got)
			}
		}
	}
}

// TestTheThumbClearsItsFloorForEverySeed holds the same two floors over the
// generated population rather than the two schemes the defaults happen to
// be, and over the increased-contrast variant as well, which asks more of
// every other pairing and may not ask less of this one.
func TestTheThumbClearsItsFloorForEverySeed(t *testing.T) {
	worstRest, worstActive := 99.0, 99.0
	for _, seed := range thumbSeeds {
		light, dark := tokens.FromSeed(seed)
		hcLight, hcDark := tokens.FromSeedHighContrast(seed)
		for _, scheme := range []struct {
			name string
			c    tokens.ColorTokens
		}{
			{"light", light}, {"dark", dark},
			{"increased-contrast light", hcLight}, {"increased-contrast dark", hcDark},
		} {
			s := FromTokens(scheme.c)
			for _, st := range []struct {
				name  string
				ink   color.NRGBA
				floor float64
				worst *float64
			}{
				{"rest", s.ThumbColor, restFloor, &worstRest},
				{"hover/drag", s.ThumbHoverColor, activeFloor, &worstActive},
			} {
				for _, g := range thumbGrounds {
					ground := scheme.c.SurfaceAt(g.level)
					got := tcolor.ContrastRatio(tcolor.Over(st.ink, ground), ground)
					if got < st.floor {
						t.Errorf("seed %v %s %s: %v over %s measures %.2f:1, under the %.1f:1 floor",
							seed, scheme.name, st.name, st.ink, g.name, got, st.floor)
					}
					if got < *st.worst {
						*st.worst = got
					}
				}
			}
		}
	}
	t.Logf("over %d seeds in four schemes: worst resting thumb %.2f:1, worst hovered %.2f:1",
		len(thumbSeeds), worstRest, worstActive)
}

// TestTheThumbIsAsTranslucentAsItsFloorAllows is the other half of the
// derivation: the point of an overlay bar is that content shows through it, so
// the thumb owes its floor and owes the reader everything past it. The gate is
// minimality in the two dials, in the order the derivation spends them — one
// step less coverage must fail, and at the coverage it settled on, the rung
// one step shallower must fail too.
func TestTheThumbIsAsTranslucentAsItsFloorAllows(t *testing.T) {
	clears := func(c tokens.ColorTokens, ink color.NRGBA, floor float64) bool {
		for _, g := range thumbGrounds {
			ground := c.SurfaceAt(g.level)
			if tcolor.ContrastRatio(tcolor.Over(ink, ground), ground) < floor {
				return false
			}
		}
		return true
	}
	for _, scheme := range []struct {
		name string
		c    tokens.ColorTokens
	}{{"light", tokens.DefaultLight}, {"dark", tokens.DefaultDark}} {
		s := FromTokens(scheme.c)
		for _, st := range []struct {
			name     string
			ink      color.NRGBA
			floor    float64
			coverage uint8
		}{
			{"rest", s.ThumbColor, restFloor, restCoverage},
			{"hover/drag", s.ThumbHoverColor, activeFloor, activeCoverage},
		} {
			// Coverage never rises above the overlay's intent without
			// cause: either it is still the intended coverage, or one
			// step less of it fails the floor.
			if st.ink.A > st.coverage {
				thinner := st.ink
				thinner.A--
				if clears(scheme.c, thinner, st.floor) {
					t.Errorf("%s %s: coverage %d clears the floor, so %d is more than the thumb needed",
						scheme.name, st.name, thinner.A, st.ink.A)
				}
			}
			// The ink is never deeper than it had to be at that coverage.
			step := rungOf(t, scheme.c, st.ink)
			if step > inkStep {
				shallower := scheme.c.Ramps.Neutral.Step(step - 100)
				shallower.A = st.ink.A
				if clears(scheme.c, shallower, st.floor) {
					t.Errorf("%s %s: step %d clears the floor at coverage %d, so step %d is deeper than the thumb needed",
						scheme.name, st.name, step-100, st.ink.A, step)
				}
			}
		}
	}
}

// TestTheTwoStatesComeFromOneRecipe: hover and drag are the same walk as rest
// against a higher floor, not a second colour picked to look right beside the
// first. So the active thumb is never weaker than the resting one on any
// ground it rides.
func TestTheTwoStatesComeFromOneRecipe(t *testing.T) {
	for _, scheme := range []struct {
		name string
		c    tokens.ColorTokens
	}{{"light", tokens.DefaultLight}, {"dark", tokens.DefaultDark}} {
		s := FromTokens(scheme.c)
		if s.ThumbHoverColor.A < s.ThumbColor.A {
			t.Errorf("%s: the hovered thumb is covered %d against the resting %d",
				scheme.name, s.ThumbHoverColor.A, s.ThumbColor.A)
		}
		if rungOf(t, scheme.c, s.ThumbHoverColor) < rungOf(t, scheme.c, s.ThumbColor) {
			t.Errorf("%s: the hovered thumb's ink is shallower than the resting thumb's", scheme.name)
		}
		for _, g := range thumbGrounds {
			ground := scheme.c.SurfaceAt(g.level)
			rest := tcolor.ContrastRatio(tcolor.Over(s.ThumbColor, ground), ground)
			active := tcolor.ContrastRatio(tcolor.Over(s.ThumbHoverColor, ground), ground)
			if active <= rest {
				t.Errorf("%s on %s: the hovered thumb measures %.2f:1 against the resting thumb's %.2f:1",
					scheme.name, g.name, active, rest)
			}
		}
	}
}

// rungOf returns the neutral-ramp step ink was taken from, failing the test
// if it was taken from anywhere else: the thumb is a ramp colour at some
// coverage, and a hand-mixed one would slip past every ratio above.
func rungOf(t *testing.T, c tokens.ColorTokens, ink color.NRGBA) int {
	t.Helper()
	for step := 100; step <= 900; step += 100 {
		if rung := c.Ramps.Neutral.Step(step); rung.R == ink.R && rung.G == ink.G && rung.B == ink.B {
			return step
		}
	}
	t.Fatalf("thumb ink %v is not a step of the neutral ramp", ink)
	return 0
}

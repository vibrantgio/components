// The three pairings this geometry answers, measured rather than eyeballed:
// the rim against the surface the control stands on and against the control's
// own fill, the label against that fill, and the mark against it. All four
// colours come out of derivations rather than fields, so what is held is the
// ratio each lands on and not the step it picked.
//
// Every sweep here runs every level — five of them is every
// placement State.Level admits — and every interaction state, because the
// fill walks under the pointer and the inks are resolved against the fill
// actually drawn.
package toolbarface

import (
	"image/color"
	"testing"

	vgcolor "github.com/vibrantgio/theme/color"
	"github.com/vibrantgio/theme/tokens"

	"github.com/vibrantgio/components/internal/focus"
)

// focusFloor is what the ring owes whatever it lies on — the same 3:1 the
// focus package derives it to, named here so the assertion and the derivation
// cannot drift apart.
const focusFloor = focus.Floor

// seeds is the spread the pairings are held over, because a palette is
// generated and the defaults are only one of its outputs: the default seed,
// the six saturated corners of sRGB, a seed with no chroma at all, and the two
// ends of the lightness range.
var seeds = []color.NRGBA{
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

// levels is every level — every surface the control can be handed.
var levels = []struct {
	name  string
	level tokens.ElevationLevel
}{
	{"chrome", tokens.LevelChrome},
	{"level-0", tokens.Level0},
	{"level-1", tokens.Level1},
	{"level-2", tokens.Level2},
	{"level-3", tokens.Level3},
}

// states is the walk the interactive face takes. The badge face draws only
// the first of them, so measuring all three measures both faces.
var states = []struct {
	name  string
	state tokens.State
}{
	{"at rest", tokens.StateNormal},
	{"hovered", tokens.StateHover},
	{"pressed", tokens.StatePressed},
}

func hex(c color.NRGBA) string {
	const digits = "0123456789abcdef"
	return string([]byte{'#',
		digits[c.R>>4], digits[c.R&0xf],
		digits[c.G>>4], digits[c.G&0xf],
		digits[c.B>>4], digits[c.B&0xf],
	})
}

// edgeHolds is the whole claim this edge makes, in one function: on every
// surface, in every state, the control's boundary is legible — either the
// rim clears the graphic floor against both the surface outside it and the fill
// inside it, or there is no rim and the fill itself clears the floor against
// the surface.
//
// The two halves must be asserted together: asserting only the drawn rim would
// be satisfied by a derivation that dropped the rim whenever it got hard, and
// asserting only the fill would be satisfied by the light scheme's 1.02:1
// whisper.
func edgeHolds(t *testing.T, label string, c tokens.ColorTokens, level tokens.ElevationLevel, state tokens.State) float64 {
	t.Helper()
	below := c.SurfaceAt(level)
	fill := Fill(c, level, state)
	rim, rimmed := Rim(c, level, state)
	if !rimmed {
		got := vgcolor.ContrastRatio(fill, below)
		if got < tokens.GraphicFloor {
			t.Errorf("%s: no rim and the fill %s only reaches %.2f:1 against the surface %s, want at least %.1f:1",
				label, hex(fill), got, hex(below), tokens.GraphicFloor)
		}
		return got
	}
	worst := vgcolor.ContrastRatio(rim, below)
	if worst < tokens.GraphicFloor {
		t.Errorf("%s: rim %s against the surface %s = %.2f:1, want at least %.1f:1",
			label, hex(rim), hex(below), worst, tokens.GraphicFloor)
	}
	if got := vgcolor.ContrastRatio(rim, fill); got < tokens.GraphicFloor {
		t.Errorf("%s: rim %s against its own fill %s = %.2f:1, want at least %.1f:1",
			label, hex(rim), hex(fill), got, tokens.GraphicFloor)
	} else if got < worst {
		worst = got
	}
	return worst
}

// TestEdgeHoldsOnEveryLevelAndState measures the edge on both of its sides,
// on every surface and in every state the fill walks through.
func TestEdgeHoldsOnEveryLevelAndState(t *testing.T) {
	for _, sc := range []struct {
		name   string
		colors tokens.ColorTokens
	}{
		{"light", tokens.DefaultLight},
		{"dark", tokens.DefaultDark},
	} {
		t.Run(sc.name, func(t *testing.T) {
			c := sc.colors
			for _, lv := range levels {
				for _, st := range states {
					name := lv.name + " " + st.name
					got := edgeHolds(t, name, c, lv.level, st.state)
					if rim, rimmed := Rim(c, lv.level, st.state); rimmed {
						t.Logf("%s: rim %s on %s over %s, worst side %.2f:1", name, hex(rim),
							hex(Fill(c, lv.level, st.state)), hex(c.SurfaceAt(lv.level)), got)
					} else {
						t.Logf("%s: no rim — the fill %s carries its own edge over %s at %.2f:1", name,
							hex(Fill(c, lv.level, st.state)), hex(c.SurfaceAt(lv.level)), got)
					}
				}
			}
		})
	}
}

// TestInksClearTheirFloors measures the label and the mark against the fill
// they are drawn on, in every state and on every surface. The label owes WCAG
// 1.4.3's 4.5:1 because it is words; the mark owes 1.4.11's 3:1 because it is
// a mark. What this gates is that [Ink] does not hand back the Text pin once
// that pin has stopped reading.
func TestInksClearTheirFloors(t *testing.T) {
	for _, sc := range []struct {
		name   string
		colors tokens.ColorTokens
	}{
		{"light", tokens.DefaultLight},
		{"dark", tokens.DefaultDark},
	} {
		t.Run(sc.name, func(t *testing.T) {
			c := sc.colors
			for _, lv := range levels {
				for _, st := range states {
					fill := Fill(c, lv.level, st.state)
					for _, ink := range []struct {
						name  string
						col   color.NRGBA
						floor float64
					}{
						{"label", Ink(c, fill, tokens.TextFloor), tokens.TextFloor},
						{"glyph", Ink(c, fill, tokens.GraphicFloor), tokens.GraphicFloor},
					} {
						got := vgcolor.ContrastRatio(ink.col, fill)
						t.Logf("%s %s %s %s on the fill %s: %.2f:1",
							lv.name, st.name, ink.name, hex(ink.col), hex(fill), got)
						if got < ink.floor {
							t.Errorf("%s %s %s %s on the fill %s = %.2f:1, want at least %.1f:1",
								lv.name, st.name, ink.name, hex(ink.col), hex(fill), got, ink.floor)
						}
					}
				}
			}
		})
	}
}

// TestPairingsHoldForEverySeed walks all three pairings over the seed
// spread and both contrast variants. The ramps carry the seed's tint, so the
// measurements move from seed to seed; the verdict may not.
func TestPairingsHoldForEverySeed(t *testing.T) {
	worstRim, worstLabel, worstGlyph := 99.0, 99.0, 99.0
	for _, seed := range seeds {
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
			for _, lv := range levels {
				for _, st := range states {
					fill := Fill(c, lv.level, st.state)
					if got := edgeHolds(t, "seed "+hex(seed)+" "+sc.name+" "+lv.name+" "+st.name,
						c, lv.level, st.state); got < worstRim {
						worstRim = got
					}
					labelInk := Ink(c, fill, tokens.TextFloor)
					if got := vgcolor.ContrastRatio(labelInk, fill); got < worstLabel {
						worstLabel = got
						if got < tokens.TextFloor {
							t.Errorf("seed %s %s: %s %s label %s on its fill = %.2f:1, want at least %.1f:1",
								hex(seed), sc.name, lv.name, st.name, hex(labelInk), got, tokens.TextFloor)
						}
					}
					glyphInk := Ink(c, fill, tokens.GraphicFloor)
					if got := vgcolor.ContrastRatio(glyphInk, fill); got < worstGlyph {
						worstGlyph = got
						if got < tokens.GraphicFloor {
							t.Errorf("seed %s %s: %s %s glyph %s on its fill = %.2f:1, want at least %.1f:1",
								hex(seed), sc.name, lv.name, st.name, hex(glyphInk), got, tokens.GraphicFloor)
						}
					}
				}
			}
		}
	}
	t.Logf("worst over the sweep: edge %.2f:1, label %.2f:1, glyph %.2f:1", worstRim, worstLabel, worstGlyph)
}

// TestFocusRingClearsItsFloor measures the ring against the side of the band
// that owes it a floor. A focused control's ring takes the rim's place, so the
// surface it stands on lies immediately outside it and its own fill
// immediately inside; the surface is the side that is the same for every control
// on that surface and the side the ring is read against.
//
// The fill inside is not measured, and the pressed control is why: it walks up
// to 20 L* off its surface, so no one colour could clear both it and the surface
// it lies on. Derived against that fill instead — as this geometry once did —
// the walk answered the fill rather than the scheme, and a control resting on
// a card came out 19 L* from the button beside it.
func TestFocusRingClearsItsFloor(t *testing.T) {
	worst := 99.0
	for _, seed := range seeds {
		light, dark := tokens.FromSeed(seed)
		lightHC, darkHC := tokens.FromSeedHighContrast(seed)
		for _, c := range []tokens.ColorTokens{light, dark, lightHC, darkHC} {
			ring := focus.Ring(c)
			for _, lv := range levels {
				surface := c.SurfaceAt(lv.level)
				if got := vgcolor.ContrastRatio(ring, surface); got < worst {
					worst = got
					if got < focusFloor {
						t.Errorf("seed %s: %s focus ring %s on the surface %s = %.2f:1, want at least %.1f:1",
							hex(seed), lv.name, hex(ring), hex(surface), got, focusFloor)
					}
				}
			}
		}
	}
	t.Logf("worst focus-ring pairing over the sweep: %.2f:1", worst)
}

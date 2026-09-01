// The chip's pairings, measured rather than eyeballed: the outline against the
// ground the chip stands on, the label and the marks against the body actually
// drawn, and — for a selected chip, which has no outline — the body against
// that ground. Every colour comes out of a derivation rather than a field, so
// what is held is the ratio each lands on and not the rung it picked.
//
// Every sweep here runs the whole elevation ladder — five rungs is every
// placement RenderState.Ground admits — both selections and every interaction
// state, because the body walks under the pointer and everything drawn on it is
// resolved against the body actually drawn.
package chip

import (
	"image/color"
	"testing"

	vgcolor "github.com/vibrantgio/theme/color"
	"github.com/vibrantgio/theme/tokens"

	"github.com/vibrantgio/components/internal/focus"
)

// chipSeeds is the spread the pairings are held over, because a palette is
// generated and the defaults are only one of its outputs: the default seed,
// the six saturated corners of sRGB, a seed with no chroma at all, and the two
// ends of the lightness range.
var chipSeeds = []color.NRGBA{
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

// chipStoreys is the whole ladder — every ground a chip can be handed.
var chipStoreys = []struct {
	name  string
	level tokens.ElevationLevel
}{
	{"floor", tokens.LevelFloor},
	{"level-0", tokens.Level0},
	{"level-1", tokens.Level1},
	{"level-2", tokens.Level2},
	{"level-3", tokens.Level3},
}

// chipStates is the walk, and it is walked from both rests: an unselected chip
// starts on the storey and a selected one on its container.
var chipStates = []struct {
	name     string
	hovered  bool
	pressed  bool
	selected bool
}{
	{name: "at rest"},
	{name: "hovered", hovered: true},
	{name: "pressed", pressed: true},
	{name: "selected", selected: true},
	{name: "selected hovered", selected: true, hovered: true},
	{name: "selected pressed", selected: true, pressed: true},
}

// chipSchemes is both schemes of one palette; the seed sweep adds the two
// high-contrast variants on top.
var chipSchemes = []struct {
	name   string
	colors tokens.ColorTokens
}{
	{"light", tokens.DefaultLight},
	{"dark", tokens.DefaultDark},
}

func hex(c color.NRGBA) string {
	const digits = "0123456789abcdef"
	return string([]byte{'#',
		digits[c.R>>4], digits[c.R&0xf],
		digits[c.G>>4], digits[c.G&0xf],
		digits[c.B>>4], digits[c.B&0xf],
	})
}

// stateOf builds the render state one sweep row asks for. Filter is the intent
// every row uses, because it is the only one that can be selected and the only
// one whose rest is therefore both derivations.
func stateOf(level tokens.ElevationLevel, row int) RenderState {
	s := chipStates[row]
	return RenderState{Ground: level, Selected: s.selected, Hovered: s.hovered, Pressed: s.pressed}
}

// bodyIsFindable is the claim a chip's boundary makes, in one function, and it
// has two halves because the chip has two rests. An unselected chip is found by
// its outline, so the outline is measured against the ground outside it. A
// selected chip has no outline at all, so its FILL is what must be found, and
// what it owes the ground is [tokens.ContainerFloor] — the threshold for
// seeing a filled region at all, not the graphic floor a mark owes, because a
// container carries no shape.
//
// Asserting only one half would be satisfied by a derivation that dropped the
// outline whenever it got hard, or by a fill that vanished into the storey.
func bodyIsFindable(t *testing.T, label string, c tokens.ColorTokens, s RenderState) float64 {
	t.Helper()
	ground := c.SurfaceAt(s.Ground)
	col := Resolve(c, Filter, s)
	if !col.Outlined {
		got := vgcolor.ContrastRatio(col.Fill, ground)
		if got < tokens.ContainerFloor {
			t.Errorf("%s: a selected chip draws no outline and its fill %s reaches only %.2f:1 against the ground %s, want at least %.2f:1",
				label, hex(col.Fill), got, hex(ground), tokens.ContainerFloor)
		}
		return got
	}
	got := vgcolor.ContrastRatio(col.Outline, ground)
	if got < tokens.GraphicFloor {
		t.Errorf("%s: outline %s against the ground %s = %.2f:1, want at least %.1f:1",
			label, hex(col.Outline), hex(ground), got, tokens.GraphicFloor)
	}
	return got
}

// inksClearTheirFloors measures the label and the marks against the body they
// are drawn on. The label owes WCAG 1.4.3's 4.5:1 because it is words; the
// marks — the leading checkmark, the leading glyph, the dismiss cross — owe
// 1.4.11's 3:1 because they are shapes that must be resolved.
//
// It is the gate on the one thing a re-derivation can quietly stop doing:
// handing back a token that has stopped reading on the body under it.
func inksClearTheirFloors(t *testing.T, label string, c tokens.ColorTokens, i Intent, s RenderState) (float64, float64) {
	t.Helper()
	col := Resolve(c, i, s)
	gotLabel := vgcolor.ContrastRatio(col.Label, col.Fill)
	if gotLabel < tokens.TextFloor {
		t.Errorf("%s: label %s on the body %s = %.2f:1, want at least %.1f:1",
			label, hex(col.Label), hex(col.Fill), gotLabel, tokens.TextFloor)
	}
	gotMark := vgcolor.ContrastRatio(col.Mark, col.Fill)
	if gotMark < tokens.GraphicFloor {
		t.Errorf("%s: mark %s on the body %s = %.2f:1, want at least %.1f:1",
			label, hex(col.Mark), hex(col.Fill), gotMark, tokens.GraphicFloor)
	}
	return gotLabel, gotMark
}

// chipIntents is every intent, because the ink derivation is not the same for
// all four: Assist reads in the full-strength Text pin and the other three in
// the muted rung, and both have to hold their floor on every body.
var chipIntents = []struct {
	name string
	i    Intent
}{
	{"assist", Assist},
	{"filter", Filter},
	{"input", Input},
	{"suggestion", Suggestion},
}

// TestBodyIsFindableOnEveryStoreyAndState walks the default palette's two
// schemes over the whole ladder in every state and both selections.
func TestBodyIsFindableOnEveryStoreyAndState(t *testing.T) {
	for _, sc := range chipSchemes {
		t.Run(sc.name, func(t *testing.T) {
			for _, storey := range chipStoreys {
				for row, st := range chipStates {
					s := stateOf(storey.level, row)
					name := storey.name + " " + st.name
					got := bodyIsFindable(t, name, sc.colors, s)
					col := Resolve(sc.colors, Filter, s)
					if col.Outlined {
						t.Logf("%s: outline %s over %s at %.2f:1", name, hex(col.Outline),
							hex(sc.colors.SurfaceAt(storey.level)), got)
					} else {
						t.Logf("%s: fill %s over %s at %.2f:1", name, hex(col.Fill),
							hex(sc.colors.SurfaceAt(storey.level)), got)
					}
				}
			}
		})
	}
}

// TestInksClearTheirFloorsOnEveryIntent measures every intent's label and mark
// on every storey, in every state and both selections.
func TestInksClearTheirFloorsOnEveryIntent(t *testing.T) {
	for _, sc := range chipSchemes {
		t.Run(sc.name, func(t *testing.T) {
			for _, in := range chipIntents {
				for _, storey := range chipStoreys {
					for row, st := range chipStates {
						if st.selected && !in.i.Selectable() {
							continue
						}
						name := in.name + " " + storey.name + " " + st.name
						lab, mark := inksClearTheirFloors(t, name, sc.colors, in.i, stateOf(storey.level, row))
						t.Logf("%s: label %.2f:1, mark %.2f:1", name, lab, mark)
					}
				}
			}
		})
	}
}

// TestChipPairingsHoldForEverySeed walks every pairing over the seed spread and
// both contrast variants. The ramps carry the seed's tint, so the measurements
// move from seed to seed; the verdict may not.
func TestChipPairingsHoldForEverySeed(t *testing.T) {
	worstEdge, worstLabel, worstMark := 99.0, 99.0, 99.0
	for _, seed := range chipSeeds {
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
			for _, in := range chipIntents {
				for _, storey := range chipStoreys {
					for row, st := range chipStates {
						if st.selected && !in.i.Selectable() {
							continue
						}
						name := "seed " + hex(seed) + " " + sc.name + " " + in.name + " " +
							storey.name + " " + st.name
						s := stateOf(storey.level, row)
						if in.i == Filter {
							if got := bodyIsFindable(t, name, sc.colors, s); got < worstEdge {
								worstEdge = got
							}
						}
						lab, mark := inksClearTheirFloors(t, name, sc.colors, in.i, s)
						if lab < worstLabel {
							worstLabel = lab
						}
						if mark < worstMark {
							worstMark = mark
						}
					}
				}
			}
		}
	}
	t.Logf("worst over the sweep: boundary %.2f:1, label %.2f:1, mark %.2f:1",
		worstEdge, worstLabel, worstMark)
}

// TestFocusRingClearsItsFloor measures the ring against the side of the band
// that owes it a floor. A focused chip's ring takes the outline's place, so the
// storey the chip stands on lies immediately outside it; the storey is the side
// that is the same for every control on it and the side the ring is read
// against.
//
// The body inside is not measured, and the pressed chip is why: it walks off
// its storey, so no one colour could clear both it and the ground the chip lies
// on. Derived against that body instead, the walk would answer the body rather
// than the scheme, and a chip resting on a card would come out rungs away from
// the button beside it.
func TestFocusRingClearsItsFloor(t *testing.T) {
	worst := 99.0
	for _, seed := range chipSeeds {
		light, dark := tokens.FromSeed(seed)
		lightHC, darkHC := tokens.FromSeedHighContrast(seed)
		for _, c := range []tokens.ColorTokens{light, dark, lightHC, darkHC} {
			ring := focus.Ring(c)
			for _, storey := range chipStoreys {
				ground := c.SurfaceAt(storey.level)
				if got := vgcolor.ContrastRatio(ring, ground); got < worst {
					worst = got
					if got < focus.Floor {
						t.Errorf("seed %s: %s focus ring %s on the storey %s = %.2f:1, want at least %.1f:1",
							hex(seed), storey.name, hex(ring), hex(ground), got, focus.Floor)
					}
				}
			}
		}
	}
	t.Logf("worst focus-ring pairing over the sweep: %.2f:1", worst)
}

// TestOnlyFilterCanBeSelected is the anatomy rule where a derivation could
// quietly break it: three of the four intents must resolve identically whether
// or not the caller set Selected, because they have no selection to draw.
func TestOnlyFilterCanBeSelected(t *testing.T) {
	for _, sc := range chipSchemes {
		for _, in := range chipIntents {
			if in.i.Selectable() {
				continue
			}
			for _, storey := range chipStoreys {
				plain := Resolve(sc.colors, in.i, RenderState{Ground: storey.level})
				marked := Resolve(sc.colors, in.i, RenderState{Ground: storey.level, Selected: true})
				if plain != marked {
					t.Errorf("%s %s on %s: Selected changed the colours to %+v from %+v; only a filter chip is selectable",
						sc.name, in.name, storey.name, marked, plain)
				}
			}
		}
	}
}

// TestAssistIsTheLoudestInk pins the one difference between the intents' inks:
// Assist reads in the page's full-strength text colour and the other three in
// the muted rung, so on a resting chip the assist label measures at least as
// much as theirs and, where the ramp has room, more.
func TestAssistIsTheLoudestInk(t *testing.T) {
	for _, sc := range chipSchemes {
		for _, storey := range chipStoreys {
			s := RenderState{Ground: storey.level}
			assist := Resolve(sc.colors, Assist, s)
			quiet := Resolve(sc.colors, Suggestion, s)
			loud := vgcolor.ContrastRatio(assist.Label, assist.Fill)
			muted := vgcolor.ContrastRatio(quiet.Label, quiet.Fill)
			if loud < muted {
				t.Errorf("%s %s: the assist label measures %.2f:1 and the muted one %.2f:1; assist is the full-strength ink",
					sc.name, storey.name, loud, muted)
			}
		}
	}
}

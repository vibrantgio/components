// The badge's pairings, measured rather than eyeballed. There are only two —
// the ink against the storey it stands on, and the close mark against that
// same storey in each state it walks through — because the badge has no fill
// and therefore no second ground. Both are derivations rather than fields, so
// what is held is the ratio each lands on and not the rung it picked.
//
// Every sweep runs the whole elevation ladder, five rungs being every
// placement RenderState.Ground admits, and all five variants, because the four
// hued ones read off pinned bases and the neutral one reads off a walk.
package badge

import (
	"image/color"
	"testing"

	vgcolor "github.com/vibrantgio/theme/color"
	"github.com/vibrantgio/theme/tokens"
)

// badgeSeeds is the spread the pairings are held over, because a palette is
// generated and the defaults are only one of its outputs: the default seed,
// the six saturated corners of sRGB, a seed with no chroma at all, and the two
// ends of the lightness range.
var badgeSeeds = []color.NRGBA{
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

// badgeStoreys is the whole ladder — every ground a badge can be handed.
var badgeStoreys = []struct {
	name  string
	level tokens.ElevationLevel
}{
	{"floor", tokens.LevelFloor},
	{"level-0", tokens.Level0},
	{"level-1", tokens.Level1},
	{"level-2", tokens.Level2},
	{"level-3", tokens.Level3},
}

var badgeVariants = []struct {
	name string
	v    Variant
}{
	{"neutral", Neutral},
	{"success", Success},
	{"warning", Warning},
	{"error", Error},
	{"info", Info},
}

func hex(c color.NRGBA) string {
	const digits = "0123456789abcdef"
	return string([]byte{'#',
		digits[c.R>>4], digits[c.R&0xf],
		digits[c.G>>4], digits[c.G&0xf],
		digits[c.B>>4], digits[c.B&0xf],
	})
}

// TestInkClearsItsFloorOnEveryStorey measures each variant's ink against the
// storey it stands on. There is no fill in between, so the storey is the whole
// of the pairing, and the floor is the text one for every utterance: a badge
// that says its word as a sign is the same utterance at the same weight.
func TestInkClearsItsFloorOnEveryStorey(t *testing.T) {
	for _, sc := range []struct {
		name   string
		colors tokens.ColorTokens
	}{
		{"light", tokens.DefaultLight},
		{"dark", tokens.DefaultDark},
	} {
		t.Run(sc.name, func(t *testing.T) {
			c := sc.colors
			for _, storey := range badgeStoreys {
				below := c.SurfaceAt(storey.level)
				for _, va := range badgeVariants {
					ink := Ink(c, va.v, storey.level)
					got := vgcolor.ContrastRatio(ink, below)
					t.Logf("%s %s ink %s on %s: %.2f:1",
						storey.name, va.name, hex(ink), hex(below), got)
					if got < tokens.TextFloor {
						t.Errorf("%s %s ink %s on the storey %s = %.2f:1, want at least %.1f:1",
							storey.name, va.name, hex(ink), hex(below), got, tokens.TextFloor)
					}
				}
			}
		})
	}
}

// TestTheCloseMarkNeverFallsBelowItsFloor measures the mark in every state it
// walks through. The mark rides the badge's own ink and owes GraphicFloor; the
// walk moves it toward the ramp's 900 end, which is away from a light ground
// and away from a dark one, so no state may take it below the floor the
// resting mark already cleared.
func TestTheCloseMarkNeverFallsBelowItsFloor(t *testing.T) {
	states := []struct {
		name string
		s    RenderState
	}{
		{"at rest", RenderState{}},
		{"hovered", RenderState{DismissHovered: true}},
		{"pressed", RenderState{DismissPressed: true}},
	}
	for _, sc := range []struct {
		name   string
		colors tokens.ColorTokens
	}{
		{"light", tokens.DefaultLight},
		{"dark", tokens.DefaultDark},
	} {
		t.Run(sc.name, func(t *testing.T) {
			c := sc.colors
			for _, storey := range badgeStoreys {
				below := c.SurfaceAt(storey.level)
				for _, va := range badgeVariants {
					ink := Ink(c, va.v, storey.level)
					for _, st := range states {
						s := st.s
						s.Ground = storey.level
						markInk := c.PinnedStateColor(ink, s.state())
						got := vgcolor.ContrastRatio(markInk, below)
						t.Logf("%s %s close mark %s %s on %s: %.2f:1",
							storey.name, va.name, st.name, hex(markInk), hex(below), got)
						if got < tokens.GraphicFloor {
							t.Errorf("%s %s close mark %s %s on the storey %s = %.2f:1, want at least %.1f:1",
								storey.name, va.name, st.name, hex(markInk), hex(below), got, tokens.GraphicFloor)
						}
					}
				}
			}
		})
	}
}

// TestThePointerMovesTheCloseMark is the acknowledgement, in pixels rather
// than in prose: the one thing on a badge that answers a pointer has to look
// different when the pointer is on it. Both schemes, because the walk heads
// toward the ramp's 900 end and that is a different direction on each.
func TestThePointerMovesTheCloseMark(t *testing.T) {
	for _, sc := range []struct {
		name   string
		colors tokens.ColorTokens
	}{
		{"light", tokens.DefaultLight},
		{"dark", tokens.DefaultDark},
	} {
		c := sc.colors
		for _, va := range badgeVariants {
			ink := Ink(c, va.v, tokens.Level0)
			rest := c.PinnedStateColor(ink, tokens.StateNormal)
			hover := c.PinnedStateColor(ink, tokens.StateHover)
			press := c.PinnedStateColor(ink, tokens.StatePressed)
			if hover == rest {
				t.Errorf("%s %s: a hovered close mark is the resting colour %s", sc.name, va.name, hex(rest))
			}
			if press == hover {
				t.Errorf("%s %s: a pressed close mark is the hovered colour %s", sc.name, va.name, hex(hover))
			}
		}
	}
}

// TestBadgePairingsHoldForEverySeed walks the ink over the seed spread and
// both contrast variants. The ramps carry the seed's tint, so the measurements
// move from seed to seed; the verdict may not.
func TestBadgePairingsHoldForEverySeed(t *testing.T) {
	worstInk, worstMark := 99.0, 99.0
	for _, seed := range badgeSeeds {
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
			for _, storey := range badgeStoreys {
				below := c.SurfaceAt(storey.level)
				for _, va := range badgeVariants {
					word := Ink(c, va.v, storey.level)
					if got := vgcolor.ContrastRatio(word, below); got < worstInk {
						worstInk = got
						if got < tokens.TextFloor {
							t.Errorf("seed %s %s: %s %s ink %s on %s = %.2f:1, want at least %.1f:1",
								hex(seed), sc.name, storey.name, va.name, hex(word), hex(below), got, tokens.TextFloor)
						}
					}
					for _, st := range []tokens.State{tokens.StateNormal, tokens.StateHover, tokens.StatePressed} {
						markInk := c.PinnedStateColor(word, st)
						if got := vgcolor.ContrastRatio(markInk, below); got < worstMark {
							worstMark = got
							if got < tokens.GraphicFloor {
								t.Errorf("seed %s %s: %s %s close mark %s on %s = %.2f:1, want at least %.1f:1",
									hex(seed), sc.name, storey.name, va.name, hex(markInk), hex(below), got, tokens.GraphicFloor)
							}
						}
					}
				}
			}
		}
	}
	t.Logf("worst over the sweep: ink %.2f:1, close mark %.2f:1", worstInk, worstMark)
}

// TestTheFourStatusesAreFourColours is what makes a set of badges a set: the
// statuses differ from each other and from the neutral one, on every storey
// and in both schemes. A palette that collapsed two roles into one rung would
// leave a reader two words in one colour.
func TestTheFourStatusesAreFourColours(t *testing.T) {
	for _, sc := range []struct {
		name   string
		colors tokens.ColorTokens
	}{
		{"light", tokens.DefaultLight},
		{"dark", tokens.DefaultDark},
	} {
		c := sc.colors
		for _, storey := range badgeStoreys {
			seen := map[color.NRGBA]string{}
			for _, va := range badgeVariants {
				ink := Ink(c, va.v, storey.level)
				if prev, ok := seen[ink]; ok {
					t.Errorf("%s %s: %s and %s both read in %s",
						sc.name, storey.name, prev, va.name, hex(ink))
				}
				seen[ink] = va.name
			}
		}
	}
}

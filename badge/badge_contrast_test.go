// The badge's pairings, measured rather than eyeballed. There are four, and
// which of them apply depends on the utterance: a worded or counted badge
// wears a container, so its fill against the surface and its foreground
// against that fill are both live, while a glyph badge stands bare and pairs
// its foreground straight with the surface. The close mark's states walk the
// fill under it on the one and its own colour on the other. Every pairing is
// a derivation rather than a field, so what is held is the ratio each lands
// on and not the step it picked.
//
// Every sweep runs every level, five of them being every
// placement RenderState.Level admits, and all five variants, because the four
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

// badgeLevels is every level — every surface a badge can be handed.
var badgeLevels = []struct {
	name  string
	level tokens.ElevationLevel
}{
	{"chrome", tokens.LevelChrome},
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

// badgeSchemes is the pair every measurement is taken in. A badge's whole
// palette is derived, and the two schemes derive from opposite ends.
var badgeSchemes = []struct {
	name   string
	colors tokens.ColorTokens
}{
	{"light", tokens.DefaultLight},
	{"dark", tokens.DefaultDark},
}

// badgeStates is the close mark through everything a pointer puts it in.
var badgeStates = []struct {
	name string
	s    tokens.State
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

// TestBareForegroundClearsItsFloorOnEveryLevel measures the glyph utterance,
// the one that stands without a container: there is nothing in between, so the
// surface is the whole of the pairing. The floor is the text one even though
// the content is a sign — a badge that says its word as a sign is the same
// utterance at the same weight.
func TestBareForegroundClearsItsFloorOnEveryLevel(t *testing.T) {
	for _, sc := range badgeSchemes {
		t.Run(sc.name, func(t *testing.T) {
			c := sc.colors
			for _, lv := range badgeLevels {
				below := c.SurfaceAt(lv.level)
				for _, va := range badgeVariants {
					fg := BareForeground(c, va.v, lv.level)
					got := vgcolor.ContrastRatio(fg, below)
					t.Logf("%s %s bare foreground %s on %s: %.2f:1",
						lv.name, va.name, hex(fg), hex(below), got)
					if got < tokens.TextFloor {
						t.Errorf("%s %s bare foreground %s on the surface %s = %.2f:1, want at least %.1f:1",
							lv.name, va.name, hex(fg), hex(below), got, tokens.TextFloor)
					}
				}
			}
		})
	}
}

// TestTheFillSeparatesFromEveryLevel is the seam: the container's own edge
// against the surface it is placed on. A fill a reader cannot see is not a
// quiet second channel, it is no second channel — which is the whole reason
// the badge wears one.
//
// The bound is [tokens.ContainerFloor] rather than a WCAG criterion, because
// WCAG has none for a field. 1.4.11's 3:1 governs a mark that must be resolved
// as a shape; a container carries no shape, and gating it at 3:1 would make
// five badges read as five filled controls.
func TestTheFillSeparatesFromEveryLevel(t *testing.T) {
	for _, sc := range badgeSchemes {
		t.Run(sc.name, func(t *testing.T) {
			c := sc.colors
			for _, lv := range badgeLevels {
				below := c.SurfaceAt(lv.level)
				for _, va := range badgeVariants {
					fill := Fill(c, va.v, lv.level)
					got := vgcolor.ContrastRatio(fill, below)
					t.Logf("%s %s fill %s on %s: %.3f:1",
						lv.name, va.name, hex(fill), hex(below), got)
					if got < tokens.ContainerFloor {
						t.Errorf("%s %s fill %s on the surface %s = %.3f:1, want at least %.2f:1",
							lv.name, va.name, hex(fill), hex(below), got, tokens.ContainerFloor)
					}
				}
			}
		})
	}
}

// TestForegroundClearsItsFloorOnTheFill is the second half of the same seam:
// the content over the field, at the text floor, in the role's own hue. It is
// the pairing a container exists to keep readable, and the one that would
// silently rot if the fill were ever re-derived without re-deriving the
// foreground on it.
func TestForegroundClearsItsFloorOnTheFill(t *testing.T) {
	for _, sc := range badgeSchemes {
		t.Run(sc.name, func(t *testing.T) {
			c := sc.colors
			for _, lv := range badgeLevels {
				for _, va := range badgeVariants {
					fill := Fill(c, va.v, lv.level)
					fg := Foreground(c, va.v, lv.level)
					got := vgcolor.ContrastRatio(fg, fill)
					t.Logf("%s %s foreground %s on the fill %s: %.2f:1",
						lv.name, va.name, hex(fg), hex(fill), got)
					if got < tokens.TextFloor {
						t.Errorf("%s %s foreground %s on the fill %s = %.2f:1, want at least %.1f:1",
							lv.name, va.name, hex(fg), hex(fill), got, tokens.TextFloor)
					}
				}
			}
		})
	}
}

// TestTheCloseMarkNeverFallsBelowItsFloor measures the mark against whatever
// is actually behind it in every state it walks through. On a badge wearing a
// container that is the fill walked one step and two, with the mark
// re-derived against it; on a bare one it is the surface, with the mark's own
// colour walking instead. Both owe GraphicFloor: an affordance cannot be
// harder to see than the utterance beside it.
//
// The re-derivation is what this test is for. Holding the resting colour over
// a walked fill measured 2.3:1 pressed on every surface and both schemes — a
// state that made the affordance harder to see the more the reader committed
// to it.
func TestTheCloseMarkNeverFallsBelowItsFloor(t *testing.T) {
	for _, sc := range badgeSchemes {
		t.Run(sc.name, func(t *testing.T) {
			c := sc.colors
			for _, lv := range badgeLevels {
				below := c.SurfaceAt(lv.level)
				for _, va := range badgeVariants {
					fill := Fill(c, va.v, lv.level)
					bare := BareForeground(c, va.v, lv.level)
					for _, st := range badgeStates {
						zone := c.PinnedStateColor(fill, st.s)
						onFill := ForegroundOver(c, va.v, zone)
						if got := vgcolor.ContrastRatio(onFill, zone); got < tokens.GraphicFloor {
							t.Errorf("%s %s close mark %s %s on the walked fill %s = %.2f:1, want at least %.1f:1",
								lv.name, va.name, st.name, hex(onFill), hex(zone), got, tokens.GraphicFloor)
						} else {
							t.Logf("%s %s close mark %s %s on the walked fill %s: %.2f:1",
								lv.name, va.name, st.name, hex(onFill), hex(zone), got)
						}
						mark := c.PinnedStateColor(bare, st.s)
						if got := vgcolor.ContrastRatio(mark, below); got < tokens.GraphicFloor {
							t.Errorf("%s %s bare close mark %s %s on the surface %s = %.2f:1, want at least %.1f:1",
								lv.name, va.name, st.name, hex(mark), hex(below), got, tokens.GraphicFloor)
						}
					}
				}
			}
		})
	}
}

// TestThePointerMovesTheCloseMark is the acknowledgement, in pixels rather
// than in prose: the one thing on a badge that answers a pointer has to look
// different when the pointer is on it. On a badge with a container what moves
// is the region under the mark, which is the reason the container was worth
// having there — a colour-only walk on an 8 dp x was the smallest possible
// answer to the largest possible target. Both schemes, because the walk heads
// toward the ramp's 900 end and that is a different direction on each.
func TestThePointerMovesTheCloseMark(t *testing.T) {
	for _, sc := range badgeSchemes {
		c := sc.colors
		for _, va := range badgeVariants {
			fill := Fill(c, va.v, tokens.Level0)
			rest := c.PinnedStateColor(fill, tokens.StateNormal)
			hover := c.PinnedStateColor(fill, tokens.StateHover)
			press := c.PinnedStateColor(fill, tokens.StatePressed)
			if rest != fill {
				t.Errorf("%s %s: a resting close region is not the fill it sits in", sc.name, va.name)
			}
			if hover == rest {
				t.Errorf("%s %s: a hovered close region is the resting fill %s", sc.name, va.name, hex(rest))
			}
			if press == hover {
				t.Errorf("%s %s: a pressed close region is the hovered fill %s", sc.name, va.name, hex(hover))
			}
			bare := BareForeground(c, va.v, tokens.Level0)
			if c.PinnedStateColor(bare, tokens.StateHover) == bare {
				t.Errorf("%s %s: a hovered bare close mark is the resting colour %s", sc.name, va.name, hex(bare))
			}
		}
	}
}

// TestBadgePairingsHoldForEverySeed walks all four pairings over the seed
// spread and both contrast variants. The ramps carry the seed's tint, so the
// measurements move from seed to seed; the verdicts may not.
func TestBadgePairingsHoldForEverySeed(t *testing.T) {
	worstBare, worstSeam, worstFg, worstMark := 99.0, 99.0, 99.0, 99.0
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
			for _, lv := range badgeLevels {
				below := c.SurfaceAt(lv.level)
				for _, va := range badgeVariants {
					bare := BareForeground(c, va.v, lv.level)
					if got := vgcolor.ContrastRatio(bare, below); got < worstBare {
						worstBare = got
						if got < tokens.TextFloor {
							t.Errorf("seed %s %s: %s %s bare foreground %s on %s = %.2f:1, want at least %.1f:1",
								hex(seed), sc.name, lv.name, va.name, hex(bare), hex(below), got, tokens.TextFloor)
						}
					}
					fill := Fill(c, va.v, lv.level)
					if got := vgcolor.ContrastRatio(fill, below); got < worstSeam {
						worstSeam = got
						if got < tokens.ContainerFloor {
							t.Errorf("seed %s %s: %s %s fill %s on %s = %.3f:1, want at least %.2f:1",
								hex(seed), sc.name, lv.name, va.name, hex(fill), hex(below), got, tokens.ContainerFloor)
						}
					}
					word := Foreground(c, va.v, lv.level)
					if got := vgcolor.ContrastRatio(word, fill); got < worstFg {
						worstFg = got
						if got < tokens.TextFloor {
							t.Errorf("seed %s %s: %s %s foreground %s on the fill %s = %.2f:1, want at least %.1f:1",
								hex(seed), sc.name, lv.name, va.name, hex(word), hex(fill), got, tokens.TextFloor)
						}
					}
					for _, st := range badgeStates {
						zone := c.PinnedStateColor(fill, st.s)
						cappedMark := ForegroundOver(c, va.v, zone)
						if got := vgcolor.ContrastRatio(cappedMark, zone); got < worstMark {
							worstMark = got
							if got < tokens.GraphicFloor {
								t.Errorf("seed %s %s: %s %s close mark %s on the walked fill %s = %.2f:1, want at least %.1f:1",
									hex(seed), sc.name, lv.name, va.name, hex(cappedMark), hex(zone), got, tokens.GraphicFloor)
							}
						}
						bareMark := c.PinnedStateColor(bare, st.s)
						if got := vgcolor.ContrastRatio(bareMark, below); got < tokens.GraphicFloor {
							t.Errorf("seed %s %s: %s %s bare close mark %s on %s = %.2f:1, want at least %.1f:1",
								hex(seed), sc.name, lv.name, va.name, hex(bareMark), hex(below), got, tokens.GraphicFloor)
						}
					}
				}
			}
		}
	}
	t.Logf("worst over the sweep: bare foreground %.2f:1, fill seam %.3f:1, foreground on the fill %.2f:1, close mark on the walked fill %.2f:1",
		worstBare, worstSeam, worstFg, worstMark)
}

// TestTheFiveVariantsAreFiveBadges is what makes a set of badges a set: no two
// variants land on one fill and no two on one foreground, on every surface and
// in both schemes. Both halves are asserted because either collapse loses a
// variant — two roles sharing a fill leaves a reader two words in one field,
// and two sharing a foreground leaves them two fields in one word.
func TestTheFiveVariantsAreFiveBadges(t *testing.T) {
	for _, sc := range badgeSchemes {
		c := sc.colors
		for _, lv := range badgeLevels {
			fgs, fills := map[color.NRGBA]string{}, map[color.NRGBA]string{}
			for _, va := range badgeVariants {
				fg := Foreground(c, va.v, lv.level)
				if prev, ok := fgs[fg]; ok {
					t.Errorf("%s %s: %s and %s both read in %s",
						sc.name, lv.name, prev, va.name, hex(fg))
				}
				fgs[fg] = va.name

				fill := Fill(c, va.v, lv.level)
				if prev, ok := fills[fill]; ok {
					t.Errorf("%s %s: %s and %s both sit on %s",
						sc.name, lv.name, prev, va.name, hex(fill))
				}
				fills[fill] = va.name
			}
		}
	}
}

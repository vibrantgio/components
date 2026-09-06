package focus_test

import (
	stdcolor "image/color"
	"math/rand"
	"testing"

	"github.com/vibrantgio/components/internal/control"
	"github.com/vibrantgio/components/internal/focus"
	"github.com/vibrantgio/components/internal/toolbarface"
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

// palettes is every palette one seed produces: both schemes of both
// derivations. The two are not each other's mirror image — a light scheme's
// ramp is walked from the opposite end and its levels climb the other way —
// so a claim asserted on one says nothing about the other.
func palettes(seed stdcolor.NRGBA) []tokens.ColorTokens {
	light, dark := tokens.FromSeed(seed)
	hcLight, hcDark := tokens.FromSeedHighContrast(seed)
	return []tokens.ColorTokens{light, dark, hcLight, hcDark}
}

// levels is every elevation level a control in this library can be put
// on, named as a host says it: a control that is told nothing stands on
// tokens.Level0, and a control on a sidebar, a rail or a toolbar stands at
// the chrome level beneath it.
var levels = []tokens.ElevationLevel{
	tokens.LevelChrome, tokens.Level0, tokens.Level1, tokens.Level2, tokens.Level3,
}

// levelName spells a level the way this file's failure messages want to
// read it, so a report names the level a developer would put a control on
// rather than an integer that counts from the content.
func levelName(level tokens.ElevationLevel) string {
	if level == tokens.LevelChrome {
		return "at the chrome level"
	}
	return "on level " + string(rune('0'+int(level)))
}

// drawn names, for one scheme, the ring every focusable control in this
// library actually draws, restating each caller's own call. Adding a focusable
// control means adding its ring here; that is what makes this file the gate for
// the whole idiom rather than for the packages that happen to exist.
//
// Every entry names a level, because a control's host is an elevation level
// and a gate that asked only one would miss a ring that
// moved on another. The point of the list is that the answer does not move: it
// is the census the single-colour claim is asserted over.
func drawn(c tokens.ColorTokens) []struct {
	name string
	ring stdcolor.NRGBA
} {
	type entry = struct {
		name string
		ring stdcolor.NRGBA
	}
	// A link's ring rides in the paragraph surface it is padded clear into.
	// Prose carries no level of its own, so a paragraph lies on the content.
	out := []entry{{"link", focus.Ring(c)}}
	for _, level := range levels {
		at := levelName(level)
		out = append(out,
			// The clear-of-the-glyph family: the ring rides in the host
			// level's surface beside the glyph, in every checked or chosen
			// state.
			entry{"checkbox " + at, focus.Ring(c)},
			entry{"radio " + at, focus.Ring(c)},
			// The promoted-border family: the ring is the control's own
			// outermost band, with the host level immediately outside it.
			entry{"text field " + at, focus.Ring(c)},
			entry{"dropdown trigger " + at, focus.Ring(c)},
			// The toolbar trigger trades its rim for the ring, in every state its
			// own fill walks to.
			entry{"toolbar " + at, focus.Ring(c)},
			// A ghost paints no fill of its own at rest, so its ring lies on the host
			// level showing through it.
			entry{"button ghost " + at, focus.RingOn(c, stdcolor.NRGBA{})},
		)
	}
	return out
}

// TestEveryControlDrawsOneColour is the ruling this package exists to hold:
// the ring's colour depends on the scheme and on nothing else. Every control
// in [drawn], on every elevation level, over the whole seed sweep and
// both derivations, draws the same pixel.
//
// Asserted per palette rather than per level, because that is the shape of
// the claim: a page carries controls standing on several levels at once, and
// what a keyboard user must not see is two of them ringing differently.
func TestEveryControlDrawsOneColour(t *testing.T) {
	for _, seed := range sweepSeeds() {
		for _, c := range palettes(seed) {
			want := focus.Ring(c)
			for _, d := range drawn(c) {
				if d.ring != want {
					t.Fatalf("seed %v: %s draws ring %v, but the scheme's ring is %v — two focus colours on one page",
						seed, d.name, d.ring, want)
				}
			}
		}
	}
}

// TestRingClearsTheFloorOnEveryLevel holds the promise the single colour is
// bought with: one step that reaches [focus.Floor] against every surface
// elevation carries, so a control keeps its ring wherever it is put.
// This is the outer side of every band in the library — the side that is the
// same for every control on a level, and the side a ring is read against.
func TestRingClearsTheFloorOnEveryLevel(t *testing.T) {
	worst := 99.0
	for _, seed := range sweepSeeds() {
		for _, c := range palettes(seed) {
			ring := focus.Ring(c)
			for _, level := range levels {
				surface := c.SurfaceAt(level)
				got := color.ContrastRatio(ring, surface)
				if got < focus.Floor {
					t.Fatalf("seed %v: ring %v measures %.2f:1 against the level %s %v, under the %.1f:1 floor",
						seed, ring, got, levelName(level), surface, focus.Floor)
				}
				if got < worst {
					worst = got
				}
			}
		}
	}
	t.Logf("worst level of the scheme's ring over the sweep: %.2f:1", worst)
}

// TestRingSeparatesFromTheRestingBorder holds the ring's second channel: on
// every level, the ring differs in luminance from the neutral resting border a
// control on that level draws — the line a text field, a checkbox, a radio and
// a picker's field trigger wear when they are not focused, and the line the
// ring replaces or sits beside — by at least [focus.BorderSeparation].
//
// The pairing is asserted against components/internal/control.Border, the
// derivation the controls actually call, rather than against the walk
// focus.Ring measures: what a reader compares a focused control with is the
// unfocused one beside it, and this is the colour that draws it.
//
// Contrast ratio is the whole of the assertion on purpose. It is a luminance
// metric and nothing else, so it is what survives macOS Differentiate Without
// Color, Windows forced-colors and a greyscale display — the environments in
// which a ring that parts from its border in hue alone stops being a ring.
func TestRingSeparatesFromTheRestingBorder(t *testing.T) {
	worst := 99.0
	for _, seed := range sweepSeeds() {
		for _, c := range palettes(seed) {
			ring := focus.Ring(c)
			for _, level := range levels {
				border := control.Border(c, level)
				got := color.ContrastRatio(ring, border)
				if got < focus.BorderSeparation {
					t.Fatalf("seed %v: ring %v measures %.3f:1 against the resting border %s %v, under the %.2f:1 separation — focus would be spelled in hue alone",
						seed, ring, got, levelName(level), border, focus.BorderSeparation)
				}
				if got < worst {
					worst = got
				}
			}
		}
	}
	t.Logf("worst ring-to-resting-border separation over the sweep: %.3f:1", worst)
}

// TestRingClearsTheRestingFillsInsideIt measures the other side of the band —
// the control's own fill, where the control has one — at rest, which is the
// worst pairing the ring is asked to hold there.
//
// Two fills answer for every control that fills a box: the step above the
// level, which the text field and the dropdown trigger fill with
// (control.Fill), and the toolbar trigger's measured step over the level, which is
// neither a level nor on the ramp. A pressed fill is left out on purpose: it
// walks up to 20 L* off its level, no one colour could clear both it and the
// level, and it is where the control's own state is spoken rather than where
// focus is.
func TestRingClearsTheRestingFillsInsideIt(t *testing.T) {
	worst := 99.0
	for _, seed := range sweepSeeds() {
		for _, c := range palettes(seed) {
			ring := focus.Ring(c)
			for _, level := range levels {
				for _, side := range []struct {
					name string
					fill stdcolor.NRGBA
				}{
					{"a field's own fill", control.Fill(c, level)},
					{"a toolbar trigger's resting fill", toolbarface.Fill(c, level, tokens.StateFocus)},
				} {
					got := color.ContrastRatio(ring, side.fill)
					if got < focus.Floor {
						t.Fatalf("seed %v: %s: ring %v measures %.2f:1 against %s %v, under the %.1f:1 floor",
							seed, levelName(level), ring, got, side.name, side.fill, focus.Floor)
					}
					if got < worst {
						worst = got
					}
				}
			}
		}
	}
	t.Logf("worst resting fill inside the ring over the sweep: %.2f:1", worst)
}

// TestRingIsNeverTheAccentFill holds the third clause of the pick: the ring is
// never the colour a control paints for a state that is not focus. c.Primary
// is what a checked box, a chosen radio and a filled button paint at rest, and
// a dark scheme realizes it exactly on a step of the ramp the ring is picked
// from, so without the exclusion the two collide byte for byte and focus is
// announced in the colour the control was already speaking.
func TestRingIsNeverTheAccentFill(t *testing.T) {
	for _, seed := range sweepSeeds() {
		for _, c := range palettes(seed) {
			if ring := focus.Ring(c); ring == c.Primary {
				t.Fatalf("seed %v: the ring is the accent fill %v — focus and checked would be one colour", seed, ring)
			}
		}
	}
}

// TestRingIsAStepOfThePrimaryRamp holds the "primary-coloured" half of the rule
// the floor cannot see. A ring that met its floor by reaching for a neutral, an
// inverse surface or an invented colour would satisfy every contrast assertion
// above and would not be the brand's focus ring.
func TestRingIsAStepOfThePrimaryRamp(t *testing.T) {
	for _, seed := range sweepSeeds() {
		for _, c := range palettes(seed) {
			ring := focus.Ring(c)
			onRamp := false
			for _, step := range c.Ramps.Primary {
				if step == ring {
					onRamp = true
					break
				}
			}
			if !onRamp {
				t.Fatalf("seed %v: ring %v is not a step of the primary ramp", seed, ring)
			}
		}
	}
}

// TestRingOnAnswersTheSchemesRingWhereverItReads holds [focus.RingOn]'s first
// clause, which is what keeps the button inside the idiom rather than beside
// it: a band inset in a fill of the control's own takes the scheme's ring
// unchanged wherever that ring reaches the floor on that fill. A ghost's
// transparent rest fill is no fill at all and always takes it.
func TestRingOnAnswersTheSchemesRingWhereverItReads(t *testing.T) {
	for _, seed := range sweepSeeds() {
		for _, c := range palettes(seed) {
			ring := focus.Ring(c)
			if got := focus.RingOn(c, stdcolor.NRGBA{}); got != ring {
				t.Fatalf("seed %v: a ghost's transparent rest fill drew %v, want the scheme's ring %v", seed, got, ring)
			}
			for _, level := range levels {
				for _, fill := range []stdcolor.NRGBA{
					c.StateAt(level, tokens.StateHover),
					c.StateAt(level, tokens.StatePressed),
				} {
					if color.ContrastRatio(ring, fill) < focus.Floor {
						continue
					}
					if got := focus.RingOn(c, fill); got != ring {
						t.Fatalf("seed %v: fill %v reads the scheme's ring at %.2f:1 yet drew %v, want %v",
							seed, fill, color.ContrastRatio(ring, fill), got, ring)
					}
				}
			}
		}
	}
}

// TestRingOnClearsTheFillItLiesOn holds the second clause. A solid primary fill
// is a step of the very ramp the ring is a step of, so no scheme's ring can
// read on it — over the sweep the two land on the same colour, 1.00:1 — and a
// ring nobody can see is not a ring. There the band is walked against the fill
// it lies on, and this asserts the walk lands somewhere legible.
func TestRingOnClearsTheFillItLiesOn(t *testing.T) {
	worst, shared, walked := 99.0, 0, 0
	for _, seed := range sweepSeeds() {
		for _, c := range palettes(seed) {
			ring := focus.Ring(c)
			for _, fill := range []stdcolor.NRGBA{
				c.SolidStateColor(tokens.RolePrimary, tokens.StateFocus),
				c.SolidStateColor(tokens.RolePrimary, tokens.StateHover),
				c.SolidStateColor(tokens.RolePrimary, tokens.StatePressed),
				c.StateColor(tokens.RolePrimary, 200, tokens.StateFocus),
				c.StateColor(tokens.RolePrimary, 200, tokens.StateHover),
				c.StateColor(tokens.RolePrimary, 200, tokens.StatePressed),
			} {
				got := focus.RingOn(c, fill)
				if got == ring {
					shared++
				} else {
					walked++
				}
				r := color.ContrastRatio(got, fill)
				if r < focus.Floor {
					t.Fatalf("seed %v: a button's ring %v measures %.2f:1 on its own fill %v, under the %.1f:1 floor",
						seed, got, r, fill, focus.Floor)
				}
				if r < worst {
					worst = r
				}
			}
		}
	}
	t.Logf("button bands: %d take the scheme's ring, %d are walked against their own fill; worst %.2f:1",
		shared, walked, worst)
}

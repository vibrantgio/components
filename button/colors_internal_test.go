package button

import (
	"image/color"
	"testing"

	"github.com/vibrantgio/theme/tokens"
)

// The emphasis registers are a choice of rungs on ADR-007's ramps, and this
// is the table that says which rungs. It is asserted in both schemes because
// the light and dark ramps are paired scales — the same step keeps the same
// job — so the register table must be written once and hold in both. If a
// register ever needs a mode-specific rule, this test is where that shows up.
func TestEmphasisResolvesTheDocumentedRampSteps(t *testing.T) {
	schemes := []struct {
		name string
		c    tokens.ColorTokens
	}{
		{"light", tokens.DefaultLight},
		{"dark", tokens.DefaultDark},
	}

	for _, sc := range schemes {
		c := sc.c
		transparent := color.NRGBA{}

		cases := []struct {
			name   string
			state  RenderState
			wantBG color.NRGBA
			wantFG color.NRGBA
		}{
			// Filled: unchanged from before the axis existed — the pinned
			// solid fill walked by tokens.SolidStateColor, under OnPrimary.
			{"filled/normal", RenderState{},
				c.SolidStateColor(tokens.RolePrimary, tokens.StateNormal), c.OnPrimary},
			{"filled/hovered", RenderState{Hovered: true},
				c.SolidStateColor(tokens.RolePrimary, tokens.StateHover), c.OnPrimary},
			{"filled/pressed", RenderState{Pressed: true},
				c.SolidStateColor(tokens.RolePrimary, tokens.StatePressed), c.OnPrimary},
			{"filled/focused", RenderState{Focused: true},
				c.SolidStateColor(tokens.RolePrimary, tokens.StateFocus), c.OnPrimary},
			{"filled/disabled", RenderState{Disabled: true},
				c.SolidStateColor(tokens.RolePrimary, tokens.StateDisabled), tokens.Disabled(c.OnPrimary)},

			// Tonal: the primary ramp's tinted 200 ground, walked one step
			// on hover and two on press, under the ramp's 900 text shade.
			{"tonal/normal", RenderState{Emphasis: Tonal},
				c.Ramps.Primary.Step(200), c.Ramps.Primary.Step(900)},
			{"tonal/hovered", RenderState{Emphasis: Tonal, Hovered: true},
				c.Ramps.Primary.Step(300), c.Ramps.Primary.Step(900)},
			{"tonal/pressed", RenderState{Emphasis: Tonal, Pressed: true},
				c.Ramps.Primary.Step(400), c.Ramps.Primary.Step(900)},
			{"tonal/focused", RenderState{Emphasis: Tonal, Focused: true},
				c.Ramps.Primary.Step(200), c.Ramps.Primary.Step(900)},
			{"tonal/disabled", RenderState{Emphasis: Tonal, Disabled: true},
				tokens.Disabled(c.Ramps.Primary.Step(200)), tokens.Disabled(c.Ramps.Primary.Step(900))},

			// Ghost: no ground at rest, focused or disabled; the neutral
			// surface's own hover and press wash under the pointer, with the
			// label walking from low-contrast text to high-contrast text as
			// its ground deepens.
			{"ghost/normal", RenderState{Emphasis: Ghost},
				transparent, c.Ramps.Neutral.Step(700)},
			{"ghost/hovered", RenderState{Emphasis: Ghost, Hovered: true},
				c.Ramps.Neutral.Step(300), c.Ramps.Neutral.Step(900)},
			{"ghost/pressed", RenderState{Emphasis: Ghost, Pressed: true},
				c.Ramps.Neutral.Step(400), c.Ramps.Neutral.Step(900)},
			{"ghost/focused", RenderState{Emphasis: Ghost, Focused: true},
				transparent, c.Ramps.Neutral.Step(700)},
			{"ghost/disabled", RenderState{Emphasis: Ghost, Disabled: true},
				transparent, tokens.Disabled(c.Ramps.Neutral.Step(700))},

			// Ghost on a raised host (I3.1): the wash is the host surface's
			// own one-rung walk. Level 1's step is the window-ground
			// assumption itself, so naming it changes nothing; levels 2 and
			// 3 walk from their own steps (300, 400). Rest stays transparent
			// on every storey — the ground is the host's to paint.
			{"ghost/level1/hovered", RenderState{Emphasis: Ghost, Ground: tokens.Level1, Hovered: true},
				c.Ramps.Neutral.Step(300), c.Ramps.Neutral.Step(900)},
			{"ghost/level2/normal", RenderState{Emphasis: Ghost, Ground: tokens.Level2},
				transparent, c.Ramps.Neutral.Step(700)},
			{"ghost/level2/hovered", RenderState{Emphasis: Ghost, Ground: tokens.Level2, Hovered: true},
				c.Ramps.Neutral.Step(400), c.Ramps.Neutral.Step(900)},
			{"ghost/level2/pressed", RenderState{Emphasis: Ghost, Ground: tokens.Level2, Pressed: true},
				c.Ramps.Neutral.Step(500), c.Ramps.Neutral.Step(900)},
			{"ghost/level3/hovered", RenderState{Emphasis: Ghost, Ground: tokens.Level3, Hovered: true},
				c.Ramps.Neutral.Step(500), c.Ramps.Neutral.Step(900)},
			{"ghost/level3/pressed", RenderState{Emphasis: Ghost, Ground: tokens.Level3, Pressed: true},
				c.Ramps.Neutral.Step(600), c.Ramps.Neutral.Step(900)},

			// The other registers carry their own grounds and ignore the
			// host's: a filled or tonal button on a level-2 surface renders
			// exactly as on the window ground.
			{"filled/level2/hovered", RenderState{Ground: tokens.Level2, Hovered: true},
				c.SolidStateColor(tokens.RolePrimary, tokens.StateHover), c.OnPrimary},
			{"tonal/level2/hovered", RenderState{Emphasis: Tonal, Ground: tokens.Level2, Hovered: true},
				c.Ramps.Primary.Step(300), c.Ramps.Primary.Step(900)},
		}

		for _, tc := range cases {
			t.Run(sc.name+"/"+tc.name, func(t *testing.T) {
				bg, fg := buttonColors(c, tc.state)
				if bg != tc.wantBG {
					t.Errorf("background = %v, want %v", bg, tc.wantBG)
				}
				if fg != tc.wantFG {
					t.Errorf("foreground = %v, want %v", fg, tc.wantFG)
				}
			})
		}
	}
}

// A transparent ground is the ghost register's whole mechanism: alpha zero
// composites as a no-op, so the surface behind the button survives it. Any
// non-zero alpha here would mean a ghost tints whatever it sits on.
func TestGhostRestingGroundIsFullyTransparent(t *testing.T) {
	for _, s := range []RenderState{
		{Emphasis: Ghost},
		{Emphasis: Ghost, Focused: true},
		{Emphasis: Ghost, Disabled: true},
	} {
		if bg, _ := buttonColors(tokens.DefaultLight, s); bg.A != 0 {
			t.Errorf("ghost %+v: background alpha = %d, want 0", s, bg.A)
		}
	}
}

// The zero value must be the filled register: that is what makes every Props
// and RenderState written before this axis existed render unchanged.
func TestZeroEmphasisIsFilled(t *testing.T) {
	var e Emphasis
	if e != Filled {
		t.Errorf("zero Emphasis = %v, want Filled", e)
	}
	if got, want := e.String(), "filled"; got != want {
		t.Errorf("Emphasis(0).String() = %q, want %q", got, want)
	}
	if got, want := Tonal.String()+" "+Ghost.String(), "tonal ghost"; got != want {
		t.Errorf("register names = %q, want %q", got, want)
	}
}

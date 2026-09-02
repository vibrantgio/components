package button

import (
	"image/color"
	"testing"

	"github.com/vibrantgio/theme/tokens"
)

// pinFill and pinInk are a pair no scheme carries: a fixed red of the kind a
// caller pins when the meaning of an action, rather than the palette, chooses
// its colour, and the ink that reads over it (white measures 6.5:1 there).
var (
	pinFill = color.NRGBA{0xb3, 0x26, 0x1e, 0xff}
	pinInk  = color.NRGBA{0xff, 0xff, 0xff, 0xff}
)

// The emphasis registers are a choice of steps on the design system's ramps,
// and this is the table that says which steps. It is asserted in both schemes because
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

			// Tonal: the primary ramp's tinted 200 fill, walked one step
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

			// Ghost: no fill at rest, focused or disabled; the host
			// surface's own hover and press wash under the pointer, with the
			// label walking from low-contrast text to high-contrast text as
			// that fill deepens.
			//
			// A ghost that is told nothing stands on the paper, so its wash
			// is the paper's own walk — tokens.ColorTokens.StateAt at level
			// 0, which lands on neutral 200 in both schemes.
			{"ghost/normal", RenderState{Emphasis: Ghost},
				transparent, c.Ramps.Neutral.Step(700)},
			{"ghost/hovered", RenderState{Emphasis: Ghost, Hovered: true},
				c.StateAt(tokens.Level0, tokens.StateHover), c.Ramps.Neutral.Step(900)},
			{"ghost/pressed", RenderState{Emphasis: Ghost, Pressed: true},
				c.StateAt(tokens.Level0, tokens.StatePressed), c.Ramps.Neutral.Step(900)},
			{"ghost/focused", RenderState{Emphasis: Ghost, Focused: true},
				transparent, c.Ramps.Neutral.Step(700)},
			{"ghost/disabled", RenderState{Emphasis: Ghost, Disabled: true},
				transparent, tokens.Disabled(c.Ramps.Neutral.Step(700))},

			// Ghost on a host that is not the paper: the wash is that host
			// surface's own one-step walk, on every level — the furniture
			// floor below the paper included, which is where a rail's or a
			// toolbar's ghost buttons actually stand. Rest stays transparent
			// on every level; the fill is the host's to paint.
			{"ghost/floor/normal", RenderState{Emphasis: Ghost, Level: tokens.LevelBackdrop},
				transparent, c.Ramps.Neutral.Step(700)},
			{"ghost/floor/hovered", RenderState{Emphasis: Ghost, Level: tokens.LevelBackdrop, Hovered: true},
				c.StateAt(tokens.LevelBackdrop, tokens.StateHover), c.Ramps.Neutral.Step(900)},
			{"ghost/level1/hovered", RenderState{Emphasis: Ghost, Level: tokens.Level1, Hovered: true},
				c.StateAt(tokens.Level1, tokens.StateHover), c.Ramps.Neutral.Step(900)},
			{"ghost/level2/normal", RenderState{Emphasis: Ghost, Level: tokens.Level2},
				transparent, c.Ramps.Neutral.Step(700)},
			{"ghost/level2/hovered", RenderState{Emphasis: Ghost, Level: tokens.Level2, Hovered: true},
				c.StateAt(tokens.Level2, tokens.StateHover), c.Ramps.Neutral.Step(900)},
			{"ghost/level2/pressed", RenderState{Emphasis: Ghost, Level: tokens.Level2, Pressed: true},
				c.StateAt(tokens.Level2, tokens.StatePressed), c.Ramps.Neutral.Step(900)},
			{"ghost/level3/hovered", RenderState{Emphasis: Ghost, Level: tokens.Level3, Hovered: true},
				c.StateAt(tokens.Level3, tokens.StateHover), c.Ramps.Neutral.Step(900)},
			{"ghost/level3/pressed", RenderState{Emphasis: Ghost, Level: tokens.Level3, Pressed: true},
				c.StateAt(tokens.Level3, tokens.StatePressed), c.Ramps.Neutral.Step(900)},

			// The other registers carry their own fills and ignore the
			// host's: a filled or tonal button on a level-2 surface renders
			// exactly as on the window's own surface.
			{"filled/level2/hovered", RenderState{Level: tokens.Level2, Hovered: true},
				c.SolidStateColor(tokens.RolePrimary, tokens.StateHover), c.OnPrimary},
			{"tonal/level2/hovered", RenderState{Emphasis: Tonal, Level: tokens.Level2, Hovered: true},
				c.Ramps.Primary.Step(300), c.Ramps.Primary.Step(900)},

			// A pinned pair takes the place of the role's, and of nothing
			// else: the same walk toward the 900 end, the same untouched pin
			// at rest and under focus, the same opacity over both halves —
			// the register's treatments, applied to the caller's colours.
			{"filled/pinned/normal", RenderState{Fill: pinFill, OnFill: pinInk},
				pinFill, pinInk},
			{"filled/pinned/hovered", RenderState{Fill: pinFill, OnFill: pinInk, Hovered: true},
				c.PinnedStateColor(pinFill, tokens.StateHover), pinInk},
			{"filled/pinned/pressed", RenderState{Fill: pinFill, OnFill: pinInk, Pressed: true},
				c.PinnedStateColor(pinFill, tokens.StatePressed), pinInk},
			{"filled/pinned/focused", RenderState{Fill: pinFill, OnFill: pinInk, Focused: true},
				pinFill, pinInk},
			{"filled/pinned/disabled", RenderState{Fill: pinFill, OnFill: pinInk, Disabled: true},
				tokens.Disabled(pinFill), tokens.Disabled(pinInk)},

			// Half a pair is no pair. A fill with no ink would draw a label
			// nobody can read and an ink with no fill has nothing to read
			// against, so either alone leaves the register exactly where it
			// has always resolved from.
			{"filled/fill-without-ink", RenderState{Fill: pinFill},
				c.SolidStateColor(tokens.RolePrimary, tokens.StateNormal), c.OnPrimary},
			{"filled/ink-without-fill", RenderState{OnFill: pinInk},
				c.SolidStateColor(tokens.RolePrimary, tokens.StateNormal), c.OnPrimary},
			{"filled/fill-without-ink/hovered", RenderState{Fill: pinFill, Hovered: true},
				c.SolidStateColor(tokens.RolePrimary, tokens.StateHover), c.OnPrimary},

			// The less pronounced registers ignore the pair outright: a tint
			// and an absent fill are not solid fills, so there is nothing in
			// them to pin.
			{"tonal/pinned/normal", RenderState{Emphasis: Tonal, Fill: pinFill, OnFill: pinInk},
				c.Ramps.Primary.Step(200), c.Ramps.Primary.Step(900)},
			{"tonal/pinned/hovered", RenderState{Emphasis: Tonal, Fill: pinFill, OnFill: pinInk, Hovered: true},
				c.Ramps.Primary.Step(300), c.Ramps.Primary.Step(900)},
			{"ghost/pinned/normal", RenderState{Emphasis: Ghost, Fill: pinFill, OnFill: pinInk},
				transparent, c.Ramps.Neutral.Step(700)},
			{"ghost/pinned/hovered", RenderState{Emphasis: Ghost, Fill: pinFill, OnFill: pinInk, Hovered: true},
				c.StateAt(tokens.Level0, tokens.StateHover), c.Ramps.Neutral.Step(900)},
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

// A transparent fill is the ghost register's whole mechanism: alpha zero
// composites as a no-op, so the surface behind the button survives it. Any
// non-zero alpha here would mean a ghost tints whatever it sits on.
func TestGhostRestingFillIsFullyTransparent(t *testing.T) {
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

// A pinned pair is the same colour in both schemes while the role it stands
// in for is not — which is the whole reason the pair exists. A theme carries
// its red as a paired scale, so the scheme decides how deep that red is; a
// caller whose colour is fixed from outside the palette needs it not to.
func TestPinnedFillIsSchemeStableWhereTheRoleIsNot(t *testing.T) {
	pinned := RenderState{Fill: pinFill, OnFill: pinInk}
	lightBG, lightFG := buttonColors(tokens.DefaultLight, pinned)
	darkBG, darkFG := buttonColors(tokens.DefaultDark, pinned)
	if lightBG != darkBG || lightFG != darkFG {
		t.Errorf("pinned pair moved between schemes: light %v on %v, dark %v on %v",
			lightFG, lightBG, darkFG, darkBG)
	}
	if lightBG != pinFill || lightFG != pinInk {
		t.Errorf("pinned pair resolved to %v on %v, want the caller's %v on %v",
			lightFG, lightBG, pinInk, pinFill)
	}

	// The control: the register's own pair does move, so the assertion above
	// is about the pin rather than about the two schemes being alike.
	stockLight, _ := buttonColors(tokens.DefaultLight, RenderState{})
	stockDark, _ := buttonColors(tokens.DefaultDark, RenderState{})
	if stockLight == stockDark {
		t.Fatal("the stock filled fill is the same in both schemes; this test proves nothing")
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

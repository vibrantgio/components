package button

import (
	"fmt"
	"image/color"
	"testing"

	vgcolor "github.com/vibrantgio/theme/color"
	"github.com/vibrantgio/theme/tokens"
)

// pinFill and pinForeground are a pair no scheme carries: a fixed red of the
// kind a caller pins when the meaning of an action, rather than the palette,
// chooses its colour, and the foreground that reads over it (white measures
// 6.5:1 there).
var (
	pinFill       = color.NRGBA{0xb3, 0x26, 0x1e, 0xff}
	pinForeground = color.NRGBA{0xff, 0xff, 0xff, 0xff}
)

// tonalRest and tonalForeground are the badge's tint recipe written out for the
// table below: the floored, surface-aware container over the level the
// button stands on, and the role's own foreground at the text floor over
// whatever the fill has walked to. components/badge draws with exactly these
// two calls; TestTonalWearsTheBadgesTint holds the two spellings together.
func tonalRest(c tokens.ColorTokens, level tokens.ElevationLevel) color.NRGBA {
	return c.StatusContainerOn(tokens.RolePrimary, c.SurfaceAt(level))
}

func tonalForeground(c tokens.ColorTokens, fill color.NRGBA) color.NRGBA {
	return c.ForegroundOn(tokens.RolePrimary, fill)
}

// The emphasis variants are a choice of derivations off the design system's
// ramps, and this is the table that says which. It is asserted in both
// schemes because the light and dark ramps are paired scales — the same step
// keeps the same job — so the table must be written once and hold in both. If
// a variant ever needs a scheme-specific rule, this test is where that shows
// up.
func TestEmphasisResolvesTheDocumentedColours(t *testing.T) {
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

			// Tonal: the badge's tint recipe, spelled out here in the two
			// token calls it is made of rather than taken from the badge —
			// so this table is an independent statement of the recipe and
			// not a copy of the code under test. The fill walks under the
			// pointer and the foreground is re-derived against wherever the
			// walk landed.
			{"tonal/normal", RenderState{Emphasis: Tonal},
				tonalRest(c, tokens.Level0), tonalForeground(c, tonalRest(c, tokens.Level0))},
			{"tonal/hovered", RenderState{Emphasis: Tonal, Hovered: true},
				c.PinnedStateColor(tonalRest(c, tokens.Level0), tokens.StateHover),
				tonalForeground(c, c.PinnedStateColor(tonalRest(c, tokens.Level0), tokens.StateHover))},
			{"tonal/pressed", RenderState{Emphasis: Tonal, Pressed: true},
				c.PinnedStateColor(tonalRest(c, tokens.Level0), tokens.StatePressed),
				tonalForeground(c, c.PinnedStateColor(tonalRest(c, tokens.Level0), tokens.StatePressed))},
			{"tonal/focused", RenderState{Emphasis: Tonal, Focused: true},
				tonalRest(c, tokens.Level0), tonalForeground(c, tonalRest(c, tokens.Level0))},
			{"tonal/disabled", RenderState{Emphasis: Tonal, Disabled: true},
				tokens.Disabled(tonalRest(c, tokens.Level0)),
				tokens.Disabled(tonalForeground(c, tonalRest(c, tokens.Level0)))},

			// Ghost: no fill at rest, focused or disabled; the host
			// surface's own hover and press fill under the pointer, with the
			// label walking from low-contrast text to high-contrast text as
			// that fill deepens.
			//
			// A ghost that is told nothing stands on the content, so its state
			// fill is the content's own walk — tokens.ColorTokens.StateAt at level
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

			// Ghost on a host that is not the content: the fill is that host
			// surface's own one-step walk, on every level — the chrome level
			// below the content included, which is where a rail's or a
			// toolbar's ghost buttons actually stand. Rest stays transparent
			// on every level; the fill is the host's to paint.
			{"ghost/chrome/normal", RenderState{Emphasis: Ghost, Level: tokens.LevelChrome},
				transparent, c.Ramps.Neutral.Step(700)},
			{"ghost/chrome/hovered", RenderState{Emphasis: Ghost, Level: tokens.LevelChrome, Hovered: true},
				c.StateAt(tokens.LevelChrome, tokens.StateHover), c.Ramps.Neutral.Step(900)},
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

			// Filled carries its own solid fill and ignores the host's: a
			// filled button on a level-2 surface renders exactly as on the
			// window's own surface. Tonal does not — its tint is derived
			// against whatever it stands on, which is what makes it the
			// badge's recipe and not a fixed step.
			{"filled/level2/hovered", RenderState{Level: tokens.Level2, Hovered: true},
				c.SolidStateColor(tokens.RolePrimary, tokens.StateHover), c.OnPrimary},
			{"tonal/level2/normal", RenderState{Emphasis: Tonal, Level: tokens.Level2},
				tonalRest(c, tokens.Level2), tonalForeground(c, tonalRest(c, tokens.Level2))},
			{"tonal/level2/hovered", RenderState{Emphasis: Tonal, Level: tokens.Level2, Hovered: true},
				c.PinnedStateColor(tonalRest(c, tokens.Level2), tokens.StateHover),
				tonalForeground(c, c.PinnedStateColor(tonalRest(c, tokens.Level2), tokens.StateHover))},
			{"tonal/chrome/normal", RenderState{Emphasis: Tonal, Level: tokens.LevelChrome},
				tonalRest(c, tokens.LevelChrome), tonalForeground(c, tonalRest(c, tokens.LevelChrome))},

			// A pinned pair takes the place of the role's, and of nothing
			// else: the same walk toward the 900 end, the same untouched pin
			// at rest and under focus, the same opacity over both halves —
			// the emphasis' treatments, applied to the caller's colours.
			{"filled/pinned/normal", RenderState{Fill: pinFill, OnFill: pinForeground},
				pinFill, pinForeground},
			{"filled/pinned/hovered", RenderState{Fill: pinFill, OnFill: pinForeground, Hovered: true},
				c.PinnedStateColor(pinFill, tokens.StateHover), pinForeground},
			{"filled/pinned/pressed", RenderState{Fill: pinFill, OnFill: pinForeground, Pressed: true},
				c.PinnedStateColor(pinFill, tokens.StatePressed), pinForeground},
			{"filled/pinned/focused", RenderState{Fill: pinFill, OnFill: pinForeground, Focused: true},
				pinFill, pinForeground},
			{"filled/pinned/disabled", RenderState{Fill: pinFill, OnFill: pinForeground, Disabled: true},
				tokens.Disabled(pinFill), tokens.Disabled(pinForeground)},

			// Half a pair is no pair. A fill with no foreground would draw a
			// label nobody can read and a foreground with no fill has nothing
			// to read against, so either alone leaves the emphasis exactly
			// where it has always resolved from.
			{"filled/fill-without-foreground", RenderState{Fill: pinFill},
				c.SolidStateColor(tokens.RolePrimary, tokens.StateNormal), c.OnPrimary},
			{"filled/foreground-without-fill", RenderState{OnFill: pinForeground},
				c.SolidStateColor(tokens.RolePrimary, tokens.StateNormal), c.OnPrimary},
			{"filled/fill-without-foreground/hovered", RenderState{Fill: pinFill, Hovered: true},
				c.SolidStateColor(tokens.RolePrimary, tokens.StateHover), c.OnPrimary},

			// The less pronounced variants ignore the pair outright: a tint
			// and an absent fill are not solid fills, so there is nothing in
			// them to pin.
			{"tonal/pinned/normal", RenderState{Emphasis: Tonal, Fill: pinFill, OnFill: pinForeground},
				tonalRest(c, tokens.Level0), tonalForeground(c, tonalRest(c, tokens.Level0))},
			{"tonal/pinned/hovered", RenderState{Emphasis: Tonal, Fill: pinFill, OnFill: pinForeground, Hovered: true},
				c.PinnedStateColor(tonalRest(c, tokens.Level0), tokens.StateHover),
				tonalForeground(c, c.PinnedStateColor(tonalRest(c, tokens.Level0), tokens.StateHover))},
			{"ghost/pinned/normal", RenderState{Emphasis: Ghost, Fill: pinFill, OnFill: pinForeground},
				transparent, c.Ramps.Neutral.Step(700)},
			{"ghost/pinned/hovered", RenderState{Emphasis: Ghost, Fill: pinFill, OnFill: pinForeground, Hovered: true},
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

// A transparent fill is the ghost emphasis' whole mechanism: alpha zero
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
	pinned := RenderState{Fill: pinFill, OnFill: pinForeground}
	lightBG, lightFG := buttonColors(tokens.DefaultLight, pinned)
	darkBG, darkFG := buttonColors(tokens.DefaultDark, pinned)
	if lightBG != darkBG || lightFG != darkFG {
		t.Errorf("pinned pair moved between schemes: light %v on %v, dark %v on %v",
			lightFG, lightBG, darkFG, darkBG)
	}
	if lightBG != pinFill || lightFG != pinForeground {
		t.Errorf("pinned pair resolved to %v on %v, want the caller's %v on %v",
			lightFG, lightBG, pinForeground, pinFill)
	}

	// The control: the emphasis' own pair does move, so the assertion above
	// is about the pin rather than about the two schemes being alike.
	stockLight, _ := buttonColors(tokens.DefaultLight, RenderState{})
	stockDark, _ := buttonColors(tokens.DefaultDark, RenderState{})
	if stockLight == stockDark {
		t.Fatal("the stock filled fill is the same in both schemes; this test proves nothing")
	}
}

// The zero value must be Filled emphasis: that is what makes every Props
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
		t.Errorf("variant names = %q, want %q", got, want)
	}
}

// ghostLevels are the surfaces a ghost is placed on: the chrome
// below the content, the content itself, and the three raised levels — the
// level-2 one being where patterns/modal stands its close mark.
var ghostLevels = []struct {
	name  string
	level tokens.ElevationLevel
}{
	{"chrome", tokens.LevelChrome},
	{"content", tokens.Level0},
	{"raised", tokens.Level1},
	{"floating", tokens.Level2},
	{"top", tokens.Level3},
}

// ghostSweepSeeds is the spread a generated palette is judged over: a
// palette is derived, and the defaults are one of its outputs.
var ghostSweepSeeds = []color.NRGBA{
	{R: 0x6c, G: 0x3a, B: 0xd4, A: 0xff},
	{R: 0xff, G: 0x00, B: 0x00, A: 0xff},
	{R: 0x00, G: 0xff, B: 0x00, A: 0xff},
	{R: 0x00, G: 0x00, B: 0xff, A: 0xff},
	{R: 0xff, G: 0xff, B: 0x00, A: 0xff},
	{R: 0x00, G: 0xff, B: 0xff, A: 0xff},
	{R: 0xff, G: 0x80, B: 0x00, A: 0xff},
	{R: 0x80, G: 0x80, B: 0x80, A: 0xff}, // a seed with no chroma at all
}

func lstar(c color.NRGBA) float64 {
	l, _, _ := vgcolor.LabFromNRGBA(c)
	return l
}

// TestGhostStateFillClearsThePerceptibilityFloor gates every ghost affordance in
// the system at once: the label button and the icon button resolve their
// colours through the one buttonColors, and patterns/modal's close mark is
// an icon ghost naming tokens.Level2, so the table below is the whole
// family.
//
// Three things are pinned. The state fill separates from the surface the
// ghost stands on by at least tokens.StateFloor, on every level and in both
// schemes of both derivations — the defect this replaced put the dark
// content's hover at 1.12:1, a signal nobody could see. Press lies beyond
// hover, so the two states stay two. And the label that fill is read
// against still clears the text floor, for as long as the fill is shallower
// than the neutral ramp's mid-value step: past that step no neutral shade
// reaches 4.5:1 over it from either side, which is a question about a
// ceiling on the fill rather than about the floor under it, and the two
// deep levels of the dark scheme sit there already — unmoved by the floor,
// and recorded here rather than silently skipped.
func TestGhostStateFillClearsThePerceptibilityFloor(t *testing.T) {
	worstStateFill, worstStateFillAt := 99.0, ""
	worstStep, worstStepAt := 99.0, ""
	worstText, worstTextAt := 99.0, ""
	pastMid := 0
	for _, seed := range ghostSweepSeeds {
		light, dark := tokens.FromSeed(seed)
		hcLight, hcDark := tokens.FromSeedHighContrast(seed)
		for _, sc := range []struct {
			name string
			c    tokens.ColorTokens
		}{
			{"light", light}, {"dark", dark},
			{"hc light", hcLight}, {"hc dark", hcDark},
		} {
			c := sc.c
			// Which way the scheme's scale runs: a walk always heads
			// toward the ramp's 900 end, darker in light and lighter in
			// dark, so one sign covers both.
			toward := 1.0
			if lstar(c.Ramps.Neutral.Step(900)) > lstar(c.Ramps.Neutral.Step(100)) {
				toward = -1
			}
			midL := lstar(c.Ramps.Neutral.Step(500))
			for _, lv := range ghostLevels {
				surface := c.SurfaceAt(lv.level)
				hoverBG, hoverFG := buttonColors(c, RenderState{
					Emphasis: Ghost, Level: lv.level, Hovered: true})
				pressBG, pressFG := buttonColors(c, RenderState{
					Emphasis: Ghost, Level: lv.level, Pressed: true})

				for _, w := range []struct {
					name   string
					bg, fg color.NRGBA
				}{{"hover", hoverBG, hoverFG}, {"press", pressBG, pressFG}} {
					where := fmt.Sprintf("seed %v %s %s %s", seed, sc.name, lv.name, w.name)
					got := vgcolor.ContrastRatio(w.bg, surface)
					if got < tokens.StateFloor {
						t.Errorf("%s: state fill %v on the surface %v measures %.3f:1, under the %.2f:1 floor",
							where, w.bg, surface, got, tokens.StateFloor)
					} else if got < worstStateFill {
						worstStateFill, worstStateFillAt = got, where
					}
					text := vgcolor.ContrastRatio(w.fg, w.bg)
					if toward*lstar(w.bg) > toward*midL {
						if text < tokens.TextFloor {
							t.Errorf("%s: label %v on the state fill %v measures %.3f:1, under the %.1f:1 text floor",
								where, w.fg, w.bg, text, tokens.TextFloor)
						} else if text < worstText {
							worstText, worstTextAt = text, where
						}
					} else {
						pastMid++
					}
				}
				if lstar(pressBG) == lstar(hoverBG) ||
					toward*lstar(pressBG) > toward*lstar(hoverBG) {
					t.Errorf("seed %v %s %s: press %v does not lie beyond hover %v",
						seed, sc.name, lv.name, pressBG, hoverBG)
				} else if step := vgcolor.ContrastRatio(pressBG, hoverBG); step < worstStep {
					worstStep, worstStepAt = step, fmt.Sprintf("seed %v %s %s", seed, sc.name, lv.name)
				}
			}
		}
	}
	t.Logf("over %d seeds, both derivations, both schemes, five levels: worst state fill %.3f:1 (floor %.2f, %s), worst press-over-hover %.3f:1 (%s), worst label %.3f:1 (%s); %d state fills lie at or past the ramp's mid step, where no neutral label reaches the text floor",
		len(ghostSweepSeeds), worstStateFill, tokens.StateFloor, worstStateFillAt,
		worstStep, worstStepAt, worstText, worstTextAt, pastMid)
}

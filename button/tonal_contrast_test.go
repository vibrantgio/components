// The Tonal variant's pairings, measured rather than eyeballed. Tonal wears
// the status badge's tint — one recipe, told apart from a badge by behaviour
// and not by colour — so what it owes is the badge's own two floors: the fill
// separates from the surface it stands on by at least tokens.ContainerFloor,
// and the content reads over that fill at tokens.TextFloor. Both are
// derivations rather than fields, so what is held is the ratio each lands on
// and not the step it picked.
//
// Every sweep runs all five levels, because a Tonal button derives against
// the surface it is placed on, and both schemes of both derivations, because
// the two schemes derive from opposite ends.
package button

import (
	"fmt"
	"image/color"
	"testing"

	"github.com/vibrantgio/components/badge"
	vgcolor "github.com/vibrantgio/theme/color"
	"github.com/vibrantgio/theme/tokens"
)

func hexOf(c color.NRGBA) string { return fmt.Sprintf("#%02x%02x%02x", c.R, c.G, c.B) }

// TestTonalWearsTheBadgesTint is the shared-recipe invariant itself: the
// tinted button and the status badge resolve their fill and their content
// colour through the same two calls, and the ONLY thing that differs between
// them is the role passed in. A second spelling of either half — a pinned
// ramp step here, a walk there — fails this test, which is the whole point of
// keeping one recipe.
//
// Both halves are checked from both ends. Each badge variant is held to the
// two token calls, so the recipe is stated independently of components/badge;
// then Tonal is held to the same two calls at the accent role, so a drift in
// either component is caught by the other's statement of the rule.
func TestTonalWearsTheBadgesTint(t *testing.T) {
	variants := []struct {
		name string
		v    badge.Variant
		role tokens.Role
	}{
		{"neutral", badge.Neutral, tokens.RoleNeutral},
		{"success", badge.Success, tokens.RoleSuccess},
		{"warning", badge.Warning, tokens.RoleWarning},
		{"error", badge.Error, tokens.RoleError},
		{"info", badge.Info, tokens.RoleInfo},
	}
	for _, sc := range []struct {
		name string
		c    tokens.ColorTokens
	}{{"light", tokens.DefaultLight}, {"dark", tokens.DefaultDark}} {
		c := sc.c
		for _, lv := range ghostLevels {
			surface := c.SurfaceAt(lv.level)

			for _, va := range variants {
				fill := c.StatusContainerOn(va.role, surface)
				if got := badge.Fill(c, va.v, lv.level); got != fill {
					t.Errorf("%s %s: badge %s fill %s is not the shared container %s",
						sc.name, lv.name, va.name, hexOf(got), hexOf(fill))
				}
				fg := c.ForegroundOn(va.role, fill)
				if got := badge.Foreground(c, va.v, lv.level); got != fg {
					t.Errorf("%s %s: badge %s foreground %s is not the shared foreground %s",
						sc.name, lv.name, va.name, hexOf(got), hexOf(fg))
				}
			}

			fill := c.StatusContainerOn(tokens.RolePrimary, surface)
			fg := c.ForegroundOn(tokens.RolePrimary, fill)
			bg, foreground := buttonColors(c, RenderState{Emphasis: Tonal, Level: lv.level})
			if bg != fill {
				t.Errorf("%s %s: tonal fill %s is not the shared container %s",
					sc.name, lv.name, hexOf(bg), hexOf(fill))
			}
			if foreground != fg {
				t.Errorf("%s %s: tonal foreground %s is not the shared foreground %s",
					sc.name, lv.name, hexOf(foreground), hexOf(fg))
			}
		}
	}
}

// TestTonalClearsTheBadgesFloors holds the two seams the tint recipe exists
// to keep: the fill against the surface it is placed on, and the content
// against that fill.
//
// The fill's bound is tokens.ContainerFloor and not a WCAG criterion, because
// WCAG has none for a field: 1.4.11's 3:1 governs a mark that must be
// resolved as a shape, and gating a tint at 3:1 would make it a solid. The
// pinned step this replaced cleared no floor at all — measured on the default
// seed against the paper, 1.133:1 light and 1.110:1 dark, a fill the reader
// could not see was there.
func TestTonalClearsTheBadgesFloors(t *testing.T) {
	for _, sc := range []struct {
		name string
		c    tokens.ColorTokens
	}{{"light", tokens.DefaultLight}, {"dark", tokens.DefaultDark}} {
		c := sc.c
		for _, lv := range ghostLevels {
			surface := c.SurfaceAt(lv.level)
			bg, fg := buttonColors(c, RenderState{Emphasis: Tonal, Level: lv.level})

			seam := vgcolor.ContrastRatio(bg, surface)
			text := vgcolor.ContrastRatio(fg, bg)
			t.Logf("%s %s: fill %s on the surface %s %.3f:1, foreground %s on the fill %.2f:1",
				sc.name, lv.name, hexOf(bg), hexOf(surface), seam, hexOf(fg), text)
			if seam < tokens.ContainerFloor {
				t.Errorf("%s %s: fill %s on the surface %s = %.3f:1, want at least %.2f:1",
					sc.name, lv.name, hexOf(bg), hexOf(surface), seam, tokens.ContainerFloor)
			}
			if text < tokens.TextFloor {
				t.Errorf("%s %s: foreground %s on the fill %s = %.2f:1, want at least %.1f:1",
					sc.name, lv.name, hexOf(fg), hexOf(bg), text, tokens.TextFloor)
			}
		}
	}
}

// TestTonalFloorsHoldForEverySeed is the same two seams over the generated
// spread, because a palette is derived and the defaults are one of its
// outputs. The ramps carry the seed's tint, so the measurements move from
// seed to seed; the verdicts may not.
func TestTonalFloorsHoldForEverySeed(t *testing.T) {
	worstSeam, worstSeamAt := 99.0, ""
	worstText, worstTextAt := 99.0, ""
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
			for _, lv := range ghostLevels {
				where := fmt.Sprintf("seed %s %s %s", hexOf(seed), sc.name, lv.name)
				surface := c.SurfaceAt(lv.level)
				bg, fg := buttonColors(c, RenderState{Emphasis: Tonal, Level: lv.level})

				seam := vgcolor.ContrastRatio(bg, surface)
				if seam < tokens.ContainerFloor {
					t.Errorf("%s: fill %s on the surface %s = %.3f:1, under the %.2f:1 floor",
						where, hexOf(bg), hexOf(surface), seam, tokens.ContainerFloor)
				} else if seam < worstSeam {
					worstSeam, worstSeamAt = seam, where
				}

				text := vgcolor.ContrastRatio(fg, bg)
				if text < tokens.TextFloor {
					t.Errorf("%s: foreground %s on the fill %s = %.3f:1, under the %.1f:1 text floor",
						where, hexOf(fg), hexOf(bg), text, tokens.TextFloor)
				} else if text < worstText {
					worstText, worstTextAt = text, where
				}
			}
		}
	}
	t.Logf("over %d seeds, both derivations, both schemes, five levels: worst fill-vs-surface %.4f:1 (floor %.2f, %s), worst foreground-vs-fill %.4f:1 (floor %.1f, %s)",
		len(ghostSweepSeeds), worstSeam, tokens.ContainerFloor, worstSeamAt,
		worstText, tokens.TextFloor, worstTextAt)
}

// TestTonalsWalkedLabelIsTheOpenGap records what the tint recipe does NOT
// hold, so the gap is a number under test rather than a surprise.
//
// The tinted walk carries a floor by construction and no ceiling. Hover and
// press move the fill along the neutral ramp and the label is re-derived
// against wherever it landed — the badge's own rule, and the best either can
// do — but past a certain depth no step of the role's ramp reaches
// tokens.TextFloor over the walked fill, so the label lands on the closest
// step there is. Over the sweep below the pressed label bottoms out at
// 4.208:1 and the default seed's own pressed label at 4.261:1, both under
// the 4.5:1 the resting pair clears everywhere.
//
// The bound here is a fence and not a floor: it fails if the gap widens,
// which is what a recorded defect owes. Closing it is a ceiling on the walk,
// which is the state-fill recipe's question and not this variant's.
func TestTonalsWalkedLabelIsTheOpenGap(t *testing.T) {
	// The measured worst, minus a hair: a derivation change that makes the
	// pressed label harder to read than it already is fails here.
	const recordedWorst = 4.20

	worstText, worstTextAt := 99.0, ""
	worstStep, worstStepAt := 99.0, ""
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
			for _, lv := range ghostLevels {
				rest, _ := buttonColors(c, RenderState{Emphasis: Tonal, Level: lv.level})
				hoverBG, hoverFG := buttonColors(c, RenderState{
					Emphasis: Tonal, Level: lv.level, Hovered: true})
				pressBG, pressFG := buttonColors(c, RenderState{
					Emphasis: Tonal, Level: lv.level, Pressed: true})

				for _, w := range []struct {
					name   string
					bg, fg color.NRGBA
				}{{"hover", hoverBG, hoverFG}, {"press", pressBG, pressFG}} {
					where := fmt.Sprintf("seed %s %s %s %s", hexOf(seed), sc.name, lv.name, w.name)
					if text := vgcolor.ContrastRatio(w.fg, w.bg); text < worstText {
						worstText, worstTextAt = text, where
					}
				}

				// The walk itself is visible on every level and both
				// schemes, and press lies beyond hover: the fill the badge
				// hands the pointer answers with a field, and the two states
				// stay two. Held at tokens.StateFloor even though the tinted
				// walk carries no floor of its own — the collapse onto the
				// container family is what put it there, and it is worth
				// knowing if it ever leaves.
				if got := vgcolor.ContrastRatio(hoverBG, rest); got < tokens.StateFloor {
					t.Errorf("seed %s %s %s: hover %s on the resting fill %s = %.3f:1, under the %.2f:1 floor",
						hexOf(seed), sc.name, lv.name, hexOf(hoverBG), hexOf(rest), got, tokens.StateFloor)
				} else if got < worstStep {
					worstStep, worstStepAt = got, fmt.Sprintf("seed %s %s %s hover", hexOf(seed), sc.name, lv.name)
				}
				if got := vgcolor.ContrastRatio(pressBG, hoverBG); got < tokens.StateFloor {
					t.Errorf("seed %s %s %s: press %s on the hovered fill %s = %.3f:1, under the %.2f:1 floor",
						hexOf(seed), sc.name, lv.name, hexOf(pressBG), hexOf(hoverBG), got, tokens.StateFloor)
				} else if got < worstStep {
					worstStep, worstStepAt = got, fmt.Sprintf("seed %s %s %s press", hexOf(seed), sc.name, lv.name)
				}
			}
		}
	}
	if worstText < recordedWorst {
		t.Errorf("the walked label now bottoms out at %.4f:1 (%s), worse than the %.2f:1 recorded here",
			worstText, worstTextAt, recordedWorst)
	}
	t.Logf("over %d seeds, both derivations, both schemes, five levels: worst walked label %.4f:1 (%s, under the %.1f:1 text floor and left open), worst state step %.4f:1 (%s)",
		len(ghostSweepSeeds), worstText, worstTextAt, tokens.TextFloor, worstStep, worstStepAt)
}

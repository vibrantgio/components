// Package control holds the colours every Vibrant Gio form control derives
// for the box it paints: the interior it fills, the edge it draws around it,
// and the ink of the prompt it shows where a value is not there yet. They
// live here rather than in components/input because components/picker's field
// trigger is the same box under a different package — one control register,
// one derivation, and no second answer to keep in step.
package control

import (
	"image/color"

	vgcolor "github.com/vibrantgio/theme/color"
	"github.com/vibrantgio/theme/tokens"
)

// GraphicFloor is WCAG 1.4.11's contrast floor for a graphic that carries
// meaning without being text — 3:1. A control that says what it is with an
// edge is entirely such a graphic: nothing about its state is spelled out, so
// its edge owes the page this much and its mark owes the fill it is drawn on
// the same. One floor serves the whole register — the box, the radio, the text
// field, the picker's field trigger.
const GraphicFloor = 3.0

// Border is the ink of a control's resting edge — the unchecked box, the
// unselected radio, the text field, the picker's field trigger: the rung of
// the neutral ramp nearest its mid-value step that reaches [GraphicFloor]
// against BOTH sides of that edge, the storey the control stands on outside it
// and the control's own interior ([Fill]) inside it.
//
// Asking the ramp which rung clears the floor is what keeps the edge legible
// in both schemes. A named step cannot: the neutral ramps are paired — light
// and dark realized at the same perceptual depths from opposite ends — so a
// fixed rung barely moves between schemes while the ground under it moves the
// whole way.
//
// ground is the storey the control is standing on, and the walk is taken
// against that storey's own fill rather than against the window's. Aimed at
// level 0 unconditionally, the same derivation measures the light scheme's
// rung at 2.94:1 over a level-2 plane and 2.15:1 over a level-3 one, both
// under the floor. Handed the level, the same walk answers a deeper rung
// where the ground is deeper and the control keeps its edge wherever it
// stands.
//
// An edge has two sides and one colour, so a walk aimed at one of them is a
// promise about the other, and where the ramp carries a rung BETWEEN the two
// neighbours the walk stops on it and breaks that promise: aimed at the ground
// alone the rung comes back 2.62:1 against the interior it encloses at a dark
// scheme's level 1, on every seed. So both candidates are derived — the rung
// clear of the ground outside and the rung clear of the interior inside — and
// the first that clears both is the edge. Over the seed sweep, both
// derivations and every storey, one of the two always does, and the worst side
// of the kept rung measures 3.07:1.
func Border(c tokens.ColorTokens, ground tokens.ElevationLevel) color.NRGBA {
	outside, inside := c.SurfaceAt(ground), Fill(c, ground)
	against := c.MarkOn(tokens.RoleNeutral, outside, GraphicFloor)
	for _, cand := range [...]color.NRGBA{
		against,
		c.MarkOn(tokens.RoleNeutral, inside, GraphicFloor),
	} {
		if vgcolor.ContrastRatio(cand, outside) >= GraphicFloor &&
			vgcolor.ContrastRatio(cand, inside) >= GraphicFloor {
			return cand
		}
	}
	return against
}

// Fill is the interior of a control that paints a box of its own — the
// unchecked box, the unselected radio's gap ring, the text field, the picker's
// field trigger: the fill of the storey one rung nearer the viewer than the
// ground the control stands on.
//
// It is a rung walked from the ground the control was handed, never an
// absolute step: a field on the paper fills at level 1, the same field inside
// a level-2 dialog fills at level 3, and one on a sidebar fills at the paper's
// storey. Level 3 is the ceiling and a control already there stays there.
//
// The Surface alias cannot serve here, and that is a pairing rather than a
// colour in exactly the way the named border step was. Surface is the neutral
// ramp's step 200, and the paired ramps realize step 200 at the same
// perceptual depth from opposite ends: in the dark scheme that lands on the
// raised storey by coincidence — #222222 is both — and in the light scheme it
// lands on nothing the ladder carries. A field filled #E8E8E8 on its #F6F6F6
// page is painted below the paper it lies on, and the paper is the darkest
// thing in the window. A surface nearer the viewer is lighter in both
// schemes, and a control that fills a box on its host is raised on it.
//
// In the light scheme the rungs above the pin are whispers — #F8F8F8 over
// #F6F6F6 — so what says where the control is, is the [Border] hairline and
// the corner radius; that trade is the ladder's, stated in full in the tokens
// package's elevation header.
func Fill(c tokens.ColorTokens, ground tokens.ElevationLevel) color.NRGBA {
	return c.SurfaceAt(ground.Raised())
}

// Placeholder is the ink of a control's prompt: the wording a text field or a
// picker's field trigger draws in the space its value will occupy, while
// there is no value there yet.
//
// It is the neutral ramp's low-contrast-text rung, the one step the ramp
// carries for words the reader is allowed to skip. The prompt has two things
// to do at once and they pull against each other — it has to be readable,
// because it says what the control is for, and it has to be visibly NOT a
// value, or an empty field reads as a filled one. A rung is what settles
// that: it is the same distance from the ink beside it in both schemes,
// where an alpha over the fill would fade with whatever storey the control
// was put on.
func Placeholder(c tokens.ColorTokens) color.NRGBA {
	return c.Ramps.Neutral.Step(700)
}

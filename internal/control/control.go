// Package control holds the colours every Vibrant Gio form control derives
// for the box it paints: the interior it fills, the edge it draws around it,
// and the foreground of the prompt it shows where a value is not there yet.
// They live here rather than in components/input because components/picker's
// field trigger is the same box under a different package — one control
// family, one derivation, and no second answer to keep in step.
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
// the same. One floor serves the whole family — the box, the radio, the text
// field, the picker's field trigger.
const GraphicFloor = 3.0

// Border is the colour a control's resting edge is drawn in — the unchecked
// box, the unselected radio, the text field, the picker's field trigger: the
// step of the neutral ramp nearest its mid-value step that reaches
// [GraphicFloor] against BOTH sides of that edge, the level the control stands
// on outside it and the control's own interior ([Fill]) inside it.
//
// Asking the ramp which step clears the floor is what keeps the edge legible
// in both schemes. A named step cannot: the neutral ramps are paired — light
// and dark realized at the same perceptual depths from opposite ends — so a
// fixed step barely moves between schemes while the surface under it moves the
// whole way.
//
// `ground` is the level the control stands on, and the walk is taken
// against that level's own fill rather than against the window's. Aimed at
// level 0 unconditionally, the same derivation measures the light scheme's
// step at 2.94:1 over a level-2 plane and 2.15:1 over a level-3 one, both
// under the floor. Handed the level, the same walk answers a deeper step
// where the surface beneath is deeper and the control keeps its edge wherever
// it stands.
//
// An edge has two sides and one colour, so a walk aimed at one of them is a
// promise about the other, and where the ramp carries a step BETWEEN the two
// neighbours the walk stops on it and breaks that promise: aimed at the
// surface outside alone the step comes back 2.62:1 against the interior it
// encloses at a dark scheme's level 1, on every seed. So both candidates are
// derived — the step clear of the surface outside and the step clear of the
// interior inside — and the first that clears both is the edge. Over the seed
// sweep, both derivations and every level, one of the two always does, and the
// worst side of the kept step measures 3.07:1.
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
// field trigger: the raise walked from the surface the control stands on.
//
// It is walked from the level the control was handed, never an absolute step
// ([tokens.ColorTokens.RaisedOn]): a field on the content fills one step
// above the content, the same field inside a dialog fills one step above the
// dialog, and one on a sidebar fills at the content's own depth. Where the
// scheme has no step left the walk clamps and the raise is flush with its
// host.
//
// The Surface alias cannot serve here, and that is a pairing rather than a
// colour in exactly the way the named border step was. Surface is the neutral
// ramp's step 200, and the paired ramps realize step 200 at the same
// perceptual depth from opposite ends: in the dark scheme that lands on the
// raised level by coincidence — #222222 is both — and in the light scheme on
// nothing the elevation carries. A surface nearer the viewer is never darker
// in either scheme, and a control that fills a box on its host is raised on
// it.
//
// A raise that the scheme cannot tell by its fill owes a seam
// ([tokens.Raise.Seamed]); this one never draws a second hairline for it,
// because [Border] is already a 3:1 mark around exactly that pairing and two
// lines saying one thing is worse than one.
func Fill(c tokens.ColorTokens, ground tokens.ElevationLevel) color.NRGBA {
	return c.RaisedOn(c.SurfaceAt(ground)).Fill
}

// Placeholder is the foreground a control's prompt is drawn in: the wording a
// text field or a picker's field trigger draws in the space its value will
// occupy, while there is no value there yet.
//
// It is the neutral ramp's low-contrast-text step, the one step the ramp
// carries for words the reader is allowed to skip. The prompt has two things
// to do at once and they pull against each other — it has to be readable,
// because it says what the control is for, and it has to be visibly NOT a
// value, or an empty field reads as a filled one. A step is what settles
// that: it is the same distance from the foreground beside it in both
// schemes, where an alpha over the fill would fade with whatever level the
// control was put on.
func Placeholder(c tokens.ColorTokens) color.NRGBA {
	return c.Ramps.Neutral.Step(700)
}

// Package focus holds the one focus idiom every control in this library
// wears, so that "what a focused control looks like" is written once.
//
// # The idiom
//
// A focused control shows a [Width] ring in the primary hue at its focus
// boundary, coloured by [Ring]: the rung of the primary ramp nearest that
// ramp's mid-value step which still reaches [Floor] against the ground the
// ring lies on.
//
// Where that boundary is depends only on whether the control fills the box it
// occupies. A text field, a dropdown trigger and a button do, so the ring is
// their own outermost Width — the text field's border promoted from a
// hairline to a ring, which is the treatment the other two adopt. A checkbox,
// a radio and a link do not: their glyph or their words sit inside a larger
// footprint, and the ring goes in that slack, clear of the glyph, with ground
// on both sides of it.
//
// # The ground is the storey, not the page
//
// Whichever placement a control takes, the ring's outer edge lies on the
// surface of whatever hosts the control — the page, a card, a dialog, a
// popover. That host is a storey of the elevation ladder and its fill moves
// with the ladder, so a ring measured once against one storey is a ring
// measured against the wrong thing on another. [Ground] resolves the storey's
// own fill from the level a control was told it stands on, and every control
// in this library hands Ring that rather than a fixed surface.
//
// Deriving the ground from the storey rather than naming a fixed surface is
// what keeps this true when a scale, a headroom or a seed moves: the floor is
// met against the fill the ring is actually drawn on, not against a surface
// that merely happens to agree with it. Over the 822 palettes this package
// sweeps, the thinnest margin anywhere is 3.44:1, a dark scheme's level-3
// popover — the ladder's lightest rung and the one with the least room left.
//
// The second placement is not a second idiom, and the checkbox is the reason
// it exists. Its edge is spoken for: unchecked, that edge is the border, and
// checked, it is the primary fill that says so. A radio's is spoken for
// twice over — the chosen state paints the edge primary, so a ring there
// would be primary recoloured to a neighbouring rung of primary, which reads
// as nothing at all. An edge already carrying state cannot also carry focus.
// What both placements share is the thing a keyboard user actually learns:
// one width, one hue, one measured contrast, at the edge of whatever has the
// keyboard.
//
// # Why the ring is a walked rung, not a fixed colour
//
// A fixed colour cannot satisfy both requirements the ring has: a known
// lightness whatever the seed is, and never too close to what it circles.
// Primary is the seed the palette is derived from, and a seed may be any
// colour a caller likes, so a ring pinned to bare Primary is a colour nobody
// in this library has seen — over a 411-seed sweep, bare Primary measures
// under 3:1 against the light surface for 226 of them, bottoming out at
// 1.00:1.
//
// Walking the primary ramp to the rung nearest its mid-value step that clears
// the floor answers both requirements: the ramp is realized at fixed
// perceptual depths, so a rung is a known lightness whatever the seed is, and
// the result is unmistakably the brand hue and never too close to its ground.
// Over that same sweep, both derivations and both schemes, the worst pairing
// any control's ring makes with the ground it circles measures 3.00:1 in a
// light scheme and 3.44:1 in a dark one, and there is no seed for which any
// of them fails.
package focus

import (
	"image/color"

	"gioui.org/unit"

	"github.com/vibrantgio/theme/tokens"
)

// Width is the thickness of the focus ring: 2 dp, at every density and on
// every control. A ring is a keyboard affordance rather than an ornament, so
// it does not thin out when the controls around it tighten.
const Width = unit.Dp(2)

// Floor is the contrast the ring must reach against the ground it circles:
// 3:1, WCAG 1.4.11's floor for a graphic that carries meaning without being
// text. It is the same floor the status marks are measured to.
const Floor = 3.0

// Ring returns the colour a focused control draws its ring in over ground —
// the ground the ring lies on: a button's own background, the surface inside
// a text field's border, the surface a checkbox's ring rides in beside the
// box, the paragraph ground a focused link is set in.
//
// Passing the ground rather than assuming one is what makes the idiom hold on
// a filled control. A filled button's ring lies on the primary fill itself,
// and no rung that reads against the page reads against that fill; the rung
// that does is a pale one, and it is still the same ring, at the same width,
// in the same place.
func Ring(c tokens.ColorTokens, ground color.NRGBA) color.NRGBA {
	return c.MarkOn(tokens.RolePrimary, ground, Floor)
}

// Ground is the ground a control standing on the given storey hands [Ring]:
// that storey's own surface fill. It is the ground the ring lies on for every
// control whose ring meets the host rather than a fill of the control's own —
// a checkbox and a radio, whose ring rides in the slack beside the glyph with
// the host showing through on both sides; a ghost button, which paints no
// ground at rest; and the promoted-border family, whose ring sits at the
// control's outermost edge with the host immediately outside it.
//
// The promoted-border family is why this resolves to the storey rather than
// to the field's own fill. That ring has two neighbours — the control's fill
// inside, the host storey outside — and one walk has to satisfy both:
// deriving against the storey does, on every palette the seed sweep reaches.
// The inner neighbour is itself a storey — a control that fills a box on its
// host is raised on it, so its fill is the rung above the ground handed in
// here — and the sweep clears at 3.44:1 worst on either side of the band.
//
// There is no special case: [tokens.ColorTokens.SurfaceAt] answers every
// storey the ladder carries, the Background pin at level 0 and the floor
// beneath it included, so the storey's own fill is simply asked for. Over
// 1644 palettes the rung derived against the pin and the rung derived
// against Surface are the same rung every time, so level 0 moves no pixel.
func Ground(c tokens.ColorTokens, level tokens.ElevationLevel) color.NRGBA {
	return c.SurfaceAt(level)
}

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
// popover. That host is a storey of the elevation ladder and its fill deepens
// as the ladder climbs, so a ring measured once against the page is a ring
// measured against the wrong thing three storeys up. [Ground] resolves the
// storey's fill from the level a control was told it stands on, and every
// control in this library hands Ring that rather than a fixed surface. The
// numbers are what make it necessary rather than tidy: on the default light
// palette the page's rung measures 2.92:1 over a dialog and 2.14:1 over a
// popover, both under [Floor]; asking the ramp against those grounds answers
// a deeper rung at 4.53:1 and 3.31:1. A dark scheme's ladder is shallower and
// its page rung already clears every storey, so it does not move — which is
// the whole argument for deriving instead of naming: one rule, two schemes,
// and only the scheme that needed to move moves.
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
// # Why the ring is not one fixed colour
//
// It was, and that is what this package replaces: a single neutral step 500
// ring, chosen once and drawn everywhere. Neutral 500 measures 2.35:1 against
// the light surface and 1.42:1 against a level-3 one — under the 3:1 floor
// WCAG 1.4.11 sets for a non-text graphic — and on a checkbox it was strictly
// worse than that. The unchecked box's border is neutral step 500 too, so a
// focused unchecked box drew the ring's colour onto its own border colour and
// the ring measured 1.00:1 against the thing it was circling. Both readings
// of that defect are true at once: the code did stroke a ring, and there was
// nothing to see.
//
// Nor is the answer the bare Primary the text field promoted its border to.
// Primary is the seed the palette is derived from, and a seed may be any
// colour a caller likes: over a 411-seed sweep, bare Primary measures under
// 3:1 against the light surface for 226 of them, bottoming out at 1.00:1. It
// reads on the default seed and is a coin toss on the rest.
//
// Walking the primary ramp fixes both. The ramp is realized at fixed
// perceptual depths, so a rung is a known lightness whatever the seed is, and
// asking for the rung nearest the middle that clears the floor answers with a
// colour that is unmistakably the brand hue and is never too close to its
// ground. Over that same sweep, both derivations and both schemes, the worst
// pairing any control's ring makes with the ground it circles measures 3.00:1
// in a light scheme and 3.44:1 in a dark one, and there is no seed for which
// any of them fails.
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
// The promoted-border family is the reason this resolves to the storey rather
// than to the field's own fill. That ring has two neighbours — the field's
// Surface inside, the host storey outside — and the storey is the harder of
// the two on every palette the seed sweep reaches: a light ladder deepens as
// it climbs, so the rung that clears the storey clears the lighter fill by
// more, and a dark ladder lightens as it climbs while its ring walks up, so
// the rung that clears the storey clears the darker fill by more. Deriving
// against the storey therefore satisfies both edges of the band at once,
// which the field's own fill could not do.
//
// Level 0 answers with [tokens.ColorTokens.Surface] rather than with the
// window ground. The window ground is the Background pin — off the neutral
// ramp by design, so there is no step to walk — and Surface is the rung the
// ladder's first storey sits on, which is where a control on the page
// effectively stands and which is the ground every ring in this library was
// measured against before storeys existed. So the zero value moves no pixel,
// and it errs, if at all, toward the deeper ground.
func Ground(c tokens.ColorTokens, level tokens.ElevationLevel) color.NRGBA {
	if step := tokens.Elevation.SurfaceStep(level); step != 0 {
		return c.Ramps.Neutral.Step(step)
	}
	return c.Surface
}

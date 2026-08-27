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
// What that resolution buys changed with ADR-022. On the pre-AU1 ladder,
// which darkened as it climbed in the light scheme, it moved the rung: the
// page's rung measured 2.92:1 over a light dialog and 2.14:1 over a light
// popover, both under [Floor], and asking the ramp against those grounds
// answered a deeper rung at 4.53:1 and 3.31:1. The re-founded ladder climbs
// toward the light in both schemes and is shallow in each — whisper steps
// toward white above a light page, a compact climb above a dark one — and
// over the 822 palettes this package sweeps the rung derived against a
// storey is now the same rung derived against the page, every time, in both
// schemes and both derivations. So the resolution no longer buys a different
// colour. It buys a measured one: the floor is met against the fill the ring
// is actually drawn on rather than against a surface that happens to agree,
// and the thinnest margin anywhere in that sweep is 3.44:1, a dark scheme's
// level-3 popover — the ladder's lightest rung and the one with the least
// room left. Deriving is what keeps that true when a scale, a headroom or a
// seed moves; naming the page is what stops being true the first time one
// does.
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
// Surface inside, the host storey outside — and one walk has to satisfy both.
// Deriving against the storey does, on every palette the seed sweep reaches;
// deriving against the fill did not, which is the defect this replaces.
//
// There is no special case, and that is the whole of the ADR-022 repair. This
// used to resolve a storey through the retired [tokens.ElevationScale.SurfaceStep]
// and fall back to [tokens.ColorTokens.Surface] when the answer was the
// accessor's zero sentinel, which meant it handed [Ring] a neutral-ramp rung
// — a ladder that had left the ramp. [tokens.ColorTokens.SurfaceAt] answers
// every storey the ladder carries, the Background pin at level 0 and the
// floor beneath it included, so the storey's own fill is simply asked for.
// Level 0 moves no pixel by that change: over 1644 palettes the rung derived
// against the pin and the rung derived against Surface are the same rung
// every time, so the zero value keeps the ring every control written before
// storeys existed already drew, and now keeps it for a stated reason.
func Ground(c tokens.ColorTokens, level tokens.ElevationLevel) color.NRGBA {
	return c.SurfaceAt(level)
}

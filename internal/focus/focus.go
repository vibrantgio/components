// Package focus holds the one focus idiom every control in this library wears,
// so that "what a focused control looks like" is written once.
//
// # The idiom
//
// A focused control shows a [Width] ring in the primary hue at its focus
// boundary, in [Ring]: one colour per scheme, drawn by every control, on every
// level, in every state. A keyboard user learns one width, one hue and one
// value, and learns it once.
//
// # One colour, not one per ground
//
// A walk aimed at a ground answers the ground, and the grounds controls stand
// on differ: two controls whose fills lie three units apart on one level come
// back rungs 19 L* apart, because the ramp carries a rung between them. Two
// purples for one state on one page is not an idiom, so the ground is out of
// the derivation: [Ring] takes the scheme and nothing else.
//
// One colour that reads everywhere is possible because the ring only ever has
// to clear [Floor] against the surfaces elevation carries, and there are
// five levels, not the whole scheme. The rung nearest the
// primary ramp's mid-value step that clears every one of those five is the
// ring. Over 1644 palettes — 411 seeds, both schemes, both derivations — such
// a rung exists for every palette and the thinnest margin anywhere is 3.33:1.
//
// # Where the ring goes, and what lies inside it
//
// Where the boundary is depends only on whether the control fills the box it
// occupies. A text field, a dropdown trigger and a chip do, so the ring is
// their own outermost band — the field's border promoted from a hairline to a
// ring, the chip's rim traded for one. A checkbox, a radio and a link do not:
// their glyph or their words sit inside a larger footprint, and the ring goes
// in that slack, clear of the glyph. A button is the one control that rings
// inside its own fill rather than at its boundary, which is what [RingOn] is
// for.
//
// Either way the level the control stands on lies immediately outside the
// ring, and that is the side the floor is owed to — the side that is the same
// for every control on that level, and the side a ring is read against. What
// lies inside the ring is the control's own fill, which the control moves as
// its own state moves: a chip's fill is a whisper over the level at rest and
// climbs 20 L* above it under a press. A band measured to the floor on both
// sides could not be one colour, because those two sides span most of the
// scheme; measured to the level it can. The resting fills are close enough
// that the ring clears them anyway — over the same 1644 palettes the ring
// measures 3.17:1 at worst against a resting fill — and a pressed fill is
// where the control's own state is spoken, not where focus is.
//
// The second placement is not a second idiom, and the checkbox is the reason it
// exists. Its edge is spoken for: unchecked, that edge is the border, and
// checked, it is the primary fill that says so. A radio's is spoken for twice
// over — the chosen state paints the edge primary, so a ring there would be
// primary recoloured to a neighbouring rung of primary, which reads as nothing
// at all. An edge already carrying state cannot also carry focus.
//
// # Why the ring is a walked rung, not a fixed colour
//
// A fixed colour cannot satisfy both requirements the ring has: a known
// lightness whatever the seed is, and never too close to what it circles.
// Primary is the seed the palette is derived from, and a seed may be any colour
// a caller likes, so a ring pinned to bare Primary is a colour nobody in this
// library has seen — over a 411-seed sweep, bare Primary measures under 3:1
// against the light surface for 226 of them, bottoming out at 1.00:1.
//
// The ramp is realized at fixed perceptual depths, so a rung of it is a known
// lightness whatever the seed is, and it is unmistakably the brand hue. Aiming
// the pick at the ramp's mid-value step is what keeps the ring from drifting to
// an end of the ramp when a shallow stack would allow it.
package focus

import (
	"image/color"

	"gioui.org/unit"

	vgcolor "github.com/vibrantgio/theme/color"
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

// midStep is the index of the primary ramp's step 500, the mid-value rung the
// pick is aimed at. tokens.Ramp is nine rungs, 100 through 900.
const midStep = 4

// levels is every elevation level a control in this library can be put on,
// and so every surface a focus ring can lie on. A control that is told nothing
// stands on tokens.Level0; one on a sidebar, a rail or a toolbar stands on the
// furniture floor beneath it; a menu or a popover stands at the ceiling.
var levels = [...]tokens.ElevationLevel{
	tokens.LevelBackdrop, tokens.Level0, tokens.Level1, tokens.Level2, tokens.Level3,
}

// Ring is the colour every focused control in this library draws its ring in:
// the rung of the primary ramp nearest that ramp's mid-value step which reaches
// [Floor] against every elevation level.
//
// Every level rather than one, because the ring is one colour and a
// control may stand anywhere on it. Asking every level at once is what makes
// the answer the scheme's rather than the caller's: a chip on a card and the
// button beside it hand this the same argument and get the same pixel.
//
// The ramp is walked from its middle out, so where several rungs clear the
// whole set the ring is the one nearest the ramp's mid-value depth — the
// depth the brand hue is most itself at, and the one furthest from both ends.
//
// A ring has to be drawn whatever it measures, so a palette no rung cleared
// on every level would take the rung that comes closest. Over 1644 palettes there
// is no such palette, and the worst level of the kept rung measures 3.33:1.
func Ring(c tokens.ColorTokens) color.NRGBA {
	pick, dist := -1, len(c.Ramps.Primary)
	widest, widestAt := -1.0, 0
	for i, rung := range c.Ramps.Primary {
		worst := 99.0
		for _, level := range levels {
			if got := vgcolor.ContrastRatio(rung, c.SurfaceAt(level)); got < worst {
				worst = got
			}
		}
		if worst > widest {
			widest, widestAt = worst, i
		}
		if worst < Floor {
			continue
		}
		d := i - midStep
		if d < 0 {
			d = -d
		}
		if d < dist {
			pick, dist = i, d
		}
	}
	if pick < 0 {
		return c.Ramps.Primary[widestAt]
	}
	return c.Ramps.Primary[pick]
}

// RingOn is [Ring] for a band that lies inside a fill of the control's own,
// with that fill on both sides of it and no level anywhere near it.
// components/button is the only such band in this library: its ring is inset in
// its own background, its own width clear of the edge, because a band flush
// with a boundary reads as that boundary.
//
// It answers [Ring] wherever the ring reads on that fill, which is every
// register whose ground is a surface or a container tint. It cannot answer it
// on a solid primary fill, and no derivation could: the ring is a rung of the
// primary ramp and so is that fill, and over the seed sweep the two land on the
// same rung — 1.00:1, the same colour twice. A ring nobody can see is not a
// ring, so on a fill the scheme's ring cannot read against, the band is walked
// against the fill it lies on instead.
//
// A transparent fill is no fill: a ghost button paints no ground at rest, so
// what its ring lies on is the level showing through it, and the level is
// what [Ring] already answers.
func RingOn(c tokens.ColorTokens, fill color.NRGBA) color.NRGBA {
	ring := Ring(c)
	if fill.A == 0 || vgcolor.ContrastRatio(ring, fill) >= Floor {
		return ring
	}
	return c.MarkOn(tokens.RolePrimary, fill, Floor)
}

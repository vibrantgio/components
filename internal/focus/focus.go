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
// # One colour, not one per surface
//
// A walk aimed at one surface answers that surface, and the surfaces controls
// stand on differ: two controls whose fills lie three units apart on one level
// come back steps 19 L* apart, because the ramp carries a step between them.
// Two purples for one state on one page is not an idiom, so the surface a
// control stands on is out of the derivation: [Ring] takes the scheme and
// nothing else.
//
// One colour that reads everywhere is possible because the ring only ever has
// to clear [Floor] against the five surfaces the levels carry, not against the
// whole scheme. The step nearest the primary ramp's mid-value
// step that clears [Floor] against every one of those five, clears
// [BorderSeparation] against every resting border drawn on them, and is not
// the accent fill itself, is the ring. Over 1644 palettes — 411 seeds, both
// schemes, both derivations — such a step exists for every palette; the
// thinnest margin anywhere is 5.44:1 against a level and 1.53:1 against a
// border.
//
// # Two channels, because one of them can be switched off
//
// Colour is the only channel this ring has: its width, its placement and its
// radius are the same in every state, so if the colour stops speaking the ring
// stops speaking. That makes the neutral resting border the pairing that
// matters most — it is the very line a text field, a checkbox, a radio and a
// picker's field trigger swap for the ring, so a ring at the border's own
// luminance says nothing but hue, and hue is what macOS Differentiate Without
// Color, Windows forced-colors and any greyscale display take away.
// [BorderSeparation] is the second channel: the pick asks the neutral ramp a
// question as well as the surfaces, and keeps only a step that is a different
// grey from every resting border the scheme draws.
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
// measures 5.49:1 at worst against a resting fill — and a pressed fill is
// where the control's own state is spoken, not where focus is.
//
// The second placement is not a second idiom, and the checkbox is the reason it
// exists. Its edge is spoken for: unchecked, that edge is the border, and
// checked, it is the primary fill that says so. A radio's is spoken for twice
// over — the chosen state paints the edge primary, so a ring there would be
// primary recoloured to a neighbouring step of primary, which reads as nothing
// at all. An edge already carrying state cannot also carry focus.
//
// # Why the ring is a walked step, not a fixed colour
//
// A fixed colour cannot satisfy both requirements the ring has: a known
// lightness whatever the seed is, and never too close to what it circles.
// Primary is the seed the palette is derived from, and a seed may be any colour
// a caller likes, so a ring pinned to bare Primary is a colour nobody in this
// library has seen — over a 411-seed sweep, bare Primary measures under 3:1
// against the light surface for 226 of them, bottoming out at 1.00:1.
//
// The ramp is realized at fixed perceptual depths, so a step of it is a known
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

// Floor is the contrast the ring must reach against the surface it is drawn
// on: 3:1, WCAG 1.4.11's floor for a graphic that carries meaning without
// being text. It is the same floor the status marks are measured to.
const Floor = 3.0

// BorderSeparation is the least luminance separation the ring owes the neutral
// resting border of a control standing on the same level — the line a text
// field, a checkbox, a radio and a picker's field trigger draw when they are
// not focused, and the line the ring replaces or sits beside. A ring at that
// border's own luminance is spelled in hue and nothing else, and hue is the
// channel an assistive display removes.
//
// 1.25:1 is measured rather than picked. Over the seed sweep — 411 seeds, both
// schemes, both derivations, every level — the separations a step of the
// primary ramp can reach while still clearing [Floor] fall in two bands with a
// wide empty stretch between them: 1.00–1.01, where the ring and the border
// are one grey, and 1.53 upward — 1.53 in the light scheme, 1.72 in the dark,
// 2.38 and 4.24 in the two high-contrast variants — where the ramp's next step
// is a different grey. The threshold goes in the empty stretch between 1.01
// and 1.53, so it rejects the collisions and only those, and no seed sits near
// enough to it to flip on a rounding.
//
// It lands on the same number as tokens.ContainerFloor and tokens.StateFloor,
// independently: all three ask when two colours off one lightness sweep stop
// being one colour, and either may move on its own evidence.
const BorderSeparation = 1.25

// midStep is the index of the primary ramp's step 500, the mid-value step the
// pick is aimed at. tokens.Ramp is nine steps, 100 through 900.
const midStep = 4

// levels is every elevation level a control in this library can be put on,
// and so every surface a focus ring can lie on. A control that is told
// nothing stands on tokens.Level0; one on a sidebar, a rail or a toolbar
// stands at the chrome level beneath it; a menu or a popover stands at the
// ceiling. The backdrop is not among them: nothing stands on it.
var levels = [...]tokens.ElevationLevel{
	tokens.LevelChrome, tokens.Level0, tokens.Level1, tokens.Level2, tokens.Level3,
}

// restingBorder is the neutral resting border a control standing on level
// draws: the neutral step that clears [Floor] against that level's own
// surface, which is components/internal/control.Border's first candidate.
//
// Taken over every level, the set this returns covers every border that
// derivation can answer, including its second candidate: that candidate is
// the same walk against the control's interior, and the interior is the
// surface of the level one nearer the viewer, which is itself a member of
// [levels]. So a ring separated from this on all five levels is separated
// from every resting border the scheme draws.
func restingBorder(c tokens.ColorTokens, level tokens.ElevationLevel) color.NRGBA {
	return c.MarkOn(tokens.RoleNeutral, c.SurfaceAt(level), Floor)
}

// Ring is the colour every focused control in this library draws its ring in:
// the step of the primary ramp nearest that ramp's mid-value step which
// reaches [Floor] against every level, reaches [BorderSeparation] against
// every level's neutral resting border, and is not the accent fill itself.
//
// Every level rather than one, because the ring is one colour and a
// control may stand anywhere on it. Asking every level at once is what makes
// the answer the scheme's rather than the caller's: a chip on a card and the
// button beside it hand this the same argument and get the same pixel.
//
// The accent fill is excluded rather than measured, because what it owes the
// ring has no scale: c.Primary is what a checked box, a chosen radio and a
// filled button paint at rest, and a dark scheme realizes it exactly on a step
// of this ramp. A ring drawn in it would announce focus in the colour the
// control was already speaking, and nothing — a reader, a screenshot, this
// package's own pixel gates — could tell the two apart. It is the rule the
// checkbox's placement already follows, read off the ramp instead of the
// geometry: an edge already carrying state cannot also carry focus.
//
// The ramp is walked from its middle out, so where several steps clear the
// whole set the ring is the one nearest the ramp's mid-value depth — the
// depth the brand hue is most itself at, and the one furthest from both ends.
//
// A ring has to be drawn whatever it measures, so a palette no step satisfied
// all three on would take the step that comes closest against the levels.
// Over 1644 palettes there is no such palette, and the worst level of the kept
// step measures 5.44:1.
func Ring(c tokens.ColorTokens) color.NRGBA {
	pick, dist := -1, len(c.Ramps.Primary)
	widest, widestAt := -1.0, 0
	for i, step := range c.Ramps.Primary {
		worst, worstBorder := 99.0, 99.0
		for _, level := range levels {
			if got := vgcolor.ContrastRatio(step, c.SurfaceAt(level)); got < worst {
				worst = got
			}
			if got := vgcolor.ContrastRatio(step, restingBorder(c, level)); got < worstBorder {
				worstBorder = got
			}
		}
		if worst > widest {
			widest, widestAt = worst, i
		}
		if worst < Floor || worstBorder < BorderSeparation || step == c.Primary {
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
// variant whose fill is a surface or a container tint. It cannot answer it
// on a solid primary fill, and no derivation could: the ring is a step of the
// primary ramp and so is that fill, and over the seed sweep the two land on the
// same step — 1.00:1, the same colour twice. A ring nobody can see is not a
// ring, so on a fill the scheme's ring cannot read against, the band is walked
// against the fill it lies on instead.
//
// A transparent fill is no fill: a ghost button paints no fill of its own at
// rest, so what its ring lies on is the level showing through it, and the
// level is what [Ring] already answers.
func RingOn(c tokens.ColorTokens, fill color.NRGBA) color.NRGBA {
	ring := Ring(c)
	if fill.A == 0 || vgcolor.ContrastRatio(ring, fill) >= Floor {
		return ring
	}
	return c.MarkOn(tokens.RolePrimary, fill, Floor)
}

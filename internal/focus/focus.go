// Package focus holds the one focus idiom every control in this library
// wears, so that "what a focused control looks like" is written once.
//
// # The idiom
//
// A focused control shows a [Width] ring in [Ring] at its focus boundary:
// the platform's keyboard focus indicator, one value per appearance, drawn
// by every control in every state. A keyboard user learns one width and one
// colour, and learns it once.
//
// The colour is the platform's and nothing is derived from it. The platform
// publishes keyboardFocusIndicatorColor as the accent at half coverage, so
// the ring composites over whatever lies under it and reads there without
// being measured against it. The caller says what that is; see [Ring].
//
// # Where the ring goes, and what lies inside it
//
// Where the boundary is depends only on whether the control fills the box it
// occupies. A text field, a pop-up trigger and a chip do, so the ring is
// their own outermost band — the field's hairline promoted to a ring, the
// chip's rim traded for one. A checkbox, a radio and a link do not: their
// glyph or their words sit inside a larger footprint, and the ring goes in
// that slack, clear of the glyph. A button rings inside its own fill rather
// than at its boundary, because a band flush with a boundary reads as that
// boundary.
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

// Ring is the colour a focused control draws its ring in: the platform's
// keyboard focus indicator, flattened over beneath — the opaque fill the
// ring's band lies on, which is the control's own fill where the band is
// inside it and the surface the control stands on where it is not.
//
// One name for every control on every surface. The platform's value carries
// its coverage, so what the ring lands as is the coverage and the fill under
// it, and nothing else: no control needs a second answer.
func Ring(p tokens.PlatformColors, beneath color.NRGBA) color.NRGBA {
	return vgcolor.Flatten(p.KeyboardFocusIndicator, beneath)
}

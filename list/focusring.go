package list

import (
	"image"
	"image/color"

	"gioui.org/layout"

	"github.com/vibrantgio/components/internal/focus"
	"github.com/vibrantgio/theme/tokens"
)

// FocusRing lays w out and draws the keyboard focus ring around the box w
// filled while state holds the keyboard. w is the list's own layout —
// [LayoutSelectable] or [LayoutSelectableScrollbar] over state — so the box
// the ring is drawn around is the list's viewport.
//
// A list at the front of a dialog is a focusable like a field: it holds the
// keyboard when the dialog opens, and what tells the reader so is the ring,
// not the tag. This is the one place that ring is drawn for a list, so every
// list wrapped in it shows the same ring at the same width in the same
// colour.
//
// # The caller says the place
//
// A list shows its focus as the platform does for the place it is in, and
// this wrapper is where that place is said: a list in the content or at the
// front of a dialog is wrapped in FocusRing, and a list anywhere the platform
// answers differently is not. A chrome rail says it in the pill's colour
// (patterns/sidebar) and a menu in the held row (components/picker), so
// neither wraps, and neither draws a ring.
//
// It is a wrapper rather than something the layout entry points do
// themselves for exactly that reason: a list is not always a focusable, and
// a ring around a rail or a menu would name a focus that place does not
// draw.
//
// # Where the ring is drawn
//
// The ring is drawn around the list's outer edge, outside its content: its
// inner edge is the box's outer edge and it is [RingWidth] wide past it, so
// no ring pixel is drawn over a row. The ring is drawn after w and in w's own
// coordinate space, outside the dimensions FocusRing reports; the caller's
// frame keeps [RingWidth] clear on every side of the list so nothing clips
// the ring or draws over it.
//
// standsOn is the opaque fill the list itself stands on, which is the fill
// under every pixel of the ring. The ring's colour is the platform's keyboard
// focus indicator at its own coverage over that fill, flattened rather than
// stroked translucent: see [focus.RingColor].
//
// No sink of its own is published: a ring a control inside a row draws must
// stay inside the list's viewport, and collecting it here would draw it past
// the viewport's edge.
func FocusRing(
	gtx layout.Context,
	state *State,
	p tokens.PlatformColors,
	standsOn color.NRGBA,
	w layout.Widget,
) layout.Dimensions {
	dims := w(gtx)
	if !gtx.Focused(state.Focus()) {
		return dims
	}
	box := image.Rectangle{Max: dims.Size}
	ring := focus.RingColor(p, standsOn)
	// The band is stroked centred on its outline, so an outline half the
	// band's width past the box puts the band's inner edge on the box's
	// outer edge. The half is rounded up: at a density where the band is an
	// odd number of pixels, the half pixel left over is left clear outside
	// the box rather than drawn over a row.
	focus.Ring(gtx, focus.Footprint(gtx, box), 0, ring, ring)
	return dims
}

// RingWidth is the width of the focus ring [FocusRing] draws past the list's
// box: the platform's measured 4 dp. A caller's frame keeps this much clear
// around the list on every side.
const RingWidth = focus.Width

package list

import (
	"image"
	"image/color"

	"gioui.org/layout"

	"github.com/vibrantgio/components/internal/focus"
	"github.com/vibrantgio/theme/tokens"
)

// Halo lays w out and draws the keyboard focus halo on the box w filled
// while state holds the keyboard. w is the list's own layout —
// [LayoutSelectable] or [LayoutSelectableScrollbar] over state — so the box
// the band straddles is the list's viewport.
//
// A list at the front of a dialog is a focusable like a field: it holds the
// keyboard when the dialog opens, and what tells the reader so is the band,
// not the tag. This is the one place that band is drawn for a list, so every
// list wrapped in it wears the same ring at the same width in the same
// colour.
//
// It is a wrapper rather than something the layout entry points do
// themselves because a list is not always a focusable: a menu that floats
// and a chrome rail's rows carry a selection without ever being the control
// the keyboard stands on, and a band around either would name a focus that
// is not there.
//
// standsOn is the opaque fill the list stands on. Both halves of the band
// take it: the list paints no fill of its own and its rows stand on that
// same fill, so the half over the list's outermost [focus.Width]/2 lies on
// it exactly as the half past the box does. A selected row reaching the
// box's edge is the one place the half over the box lies on the selection's
// fill instead.
//
// The band is drawn after w and in w's own coordinate space, so it lands
// over the rows and no sink of its own is published: a halo a control inside
// a row draws must stay inside the list's viewport, and collecting it here
// would carry it out past the viewport's edge.
func Halo(
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
	ring := focus.Ring(p, standsOn)
	focus.Halo(gtx, image.Rectangle{Max: dims.Size}, 0, ring, ring)
	return dims
}

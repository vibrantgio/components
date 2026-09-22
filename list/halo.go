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
// # The caller says the place
//
// A list shows its focus as the platform does for the place it stands in,
// and this wrapper is where that place is said: a list standing in the
// content or at the front of a dialog is wrapped in Halo, and a list
// standing anywhere the platform answers differently is not. A chrome rail
// says it in the pill's colour (patterns/sidebar) and a menu in the held
// row (components/picker), so neither wraps, and neither draws a ring.
//
// It is a wrapper rather than something the layout entry points do
// themselves for exactly that reason: a list is not always a focusable, and
// a band around a rail or a menu would name a focus that place does not
// draw.
//
// # What the band lands on
//
// standsOn is the opaque fill the list itself stands on. The band's half
// past the box lies on it, as every control's does; the half over the box
// lies on whatever each of its pixels stands on, which is the fill the row
// under it paints. rowFill answers that per row — it is handed a row's index
// into the caller's item slice and returns the opaque fill that row paints,
// or a colour with a zero alpha where the row paints none and standsOn shows
// through. A nil rowFill is a list whose rows paint nothing.
//
// Every row is asked, not the selected one alone: a list that fills a row
// under the pointer as well as under the selection puts two fills under one
// band, and the band's over half must land on whichever of them the row it
// crosses actually paints. A row reaching the box's edge is what puts a
// second fill under the band at all — the band's sides then run down rows of
// both, and a filled row at the head or the foot of the viewport carries the
// band across it as well.
//
// Every colour the band takes is the platform's keyboard focus indicator at
// its own coverage over the fill beneath it, flattened rather than stroked
// translucent: see [focus.Fill].
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
	rowFill func(row int) color.NRGBA,
	w layout.Widget,
) layout.Dimensions {
	dims := w(gtx)
	if !gtx.Focused(state.Focus()) {
		return dims
	}
	box := image.Rectangle{Max: dims.Size}
	ring := focus.Ring(p, standsOn)
	var fills []focus.Fill
	if rowFill != nil {
		for _, r := range state.rowBoxes() {
			fill := rowFill(r.index)
			if fill.A == 0 || fill == standsOn {
				continue
			}
			clipped := r.box.Intersect(box)
			if clipped.Empty() {
				continue
			}
			fills = append(fills, focus.Fill{Box: clipped, Band: focus.Ring(p, fill)})
		}
	}
	focus.Halo(gtx, box, 0, ring, ring, fills...)
	return dims
}

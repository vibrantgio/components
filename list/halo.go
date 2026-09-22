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
// standsOn is the opaque fill the list stands on and selected the fill the
// caller paints under the selected row. The band's half past the box lies on
// standsOn, as every control's does; the half over the box lies on whatever
// each of its pixels stands on — the selection's fill where it crosses the
// selected row, standsOn everywhere else, the list painting no fill of its
// own. A row reaching the box's edge is what puts two fills under one band:
// the band's sides then run down rows of both, and a selected row at the
// head or the foot of the viewport carries the band across it as well.
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
	standsOn, selected color.NRGBA,
	w layout.Widget,
) layout.Dimensions {
	dims := w(gtx)
	if !gtx.Focused(state.Focus()) {
		return dims
	}
	box := image.Rectangle{Max: dims.Size}
	ring := focus.Ring(p, standsOn)
	var fills []focus.Fill
	if row, ok := state.selectedBox(); ok {
		if row = row.Intersect(box); !row.Empty() {
			fills = append(fills, focus.Fill{Box: row, Band: focus.Ring(p, selected)})
		}
	}
	focus.Halo(gtx, box, 0, ring, ring, fills...)
	return dims
}

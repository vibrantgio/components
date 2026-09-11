// Package surface answers what a component stands on, so that every
// component in this library spells the answer the same way.
//
// The platform's labels, seams, overlays and focus ring carry a coverage
// rather than a colour: one recorded value that reads correctly on every
// fill because it composites over whatever is beneath it. The platform takes
// that composite in encoded sRGB, where Gio's rasterizer takes it in linear
// light, so a component flattens the name against the surface it stands on
// (theme/color.Flatten) and hands Gio an opaque fill. Which surface that is,
// is the one thing the component cannot work out for itself: a caller that
// put it on a card, a selected row or a coloured fill says so through the
// component's Surface property, and everything else stands on the level the
// component belongs to.
package surface

import "image/color"

// Or returns the surface the caller stated, or plane — the fill of the level
// the component stands at unless the caller moved it — when it stated none.
//
// A surface is an opaque fill, so alpha zero is no answer rather than a
// transparent one, and alpha zero is the zero value a Surface property is
// left at. On macOS 26 the window's plane, the content's fill and the chrome
// material carry one value in the light appearance, so an unstated surface
// is wrong only where something was deliberately put between.
func Or(stated, plane color.NRGBA) color.NRGBA {
	if stated.A == 0 {
		return plane
	}
	return stated
}

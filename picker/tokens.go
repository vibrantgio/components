package picker

import (
	"gioui.org/text"

	"github.com/vibrantgio/components/scrollbar"
	"github.com/vibrantgio/theme/tokens"
)

// resolvedTokens is the concrete per-emission snapshot the layout.Widget closures
// draw from: the whole theme flattened to the values one frame needs.
//
// Two text styles, because the two triggers are drawn for two variants. The
// field and the menu rows take body and the toolbar trigger takes title. Both
// are the System font role, the role that sets text on a regular control.
type resolvedTokens struct {
	platform tokens.PlatformColors
	body     tokens.TextStyle // SystemFont: the field and the menu rows
	title    tokens.TextStyle // SystemFont: the toolbar trigger
	spacing  tokens.SpacingScale
	radius   tokens.RadiusScale
	density  tokens.Density // control height and inner padding
	shaper   *text.Shaper   // the theme's shaper; nil in the Render* paths
	// bar is the scrollbar a capped menu scrolls its rows under, already
	// resolved for this frame: the theme's own bar after the caller has
	// spent the platform's "Show scroll bars" setting on it. See menuBar
	// and [MenuProps.Scrollbar].
	bar scrollbar.Style
}

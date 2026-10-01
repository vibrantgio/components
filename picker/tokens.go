package picker

import (
	"gioui.org/text"

	"github.com/vibrantgio/theme/tokens"
)

// resolvedTokens is the concrete per-emission snapshot the layout.Widget closures
// draw from: the whole theme flattened to the values one frame needs.
//
// Two text styles, because the two triggers are drawn for two variants. The
// field and the menu rows are BodyLarge, the style the form controls beside
// them are set in; the toolbar trigger is LabelLarge, the style a control that
// names a value rather than accepting one is set in.
type resolvedTokens struct {
	platform tokens.PlatformColors
	body     tokens.TextStyle // BodyLarge: the field and the menu rows
	title    tokens.TextStyle // LabelLarge: the toolbar trigger
	spacing  tokens.SpacingScale
	radius   tokens.RadiusScale
	density  tokens.Density // control height and inner padding
	shaper   *text.Shaper   // the theme's shaper; nil in the Render* paths
}

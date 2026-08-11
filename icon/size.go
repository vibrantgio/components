package icon

import (
	"gioui.org/unit"

	"github.com/vibrantgio/theme/tokens"
)

// Size returns the default icon glyph size for a density: the control's inner
// content box, ControlHeight − 2·PaddingY — 20 dp at tokens.Comfortable
// (36 − 2·8), 16 dp at tokens.Compact (28 − 2·6).
//
// The rule comes from the E1.1 metrics world. shadcn/ui draws 16 px icons
// ([&_svg]:size-4) inside its h-9 py-2 buttons, whose content box is
// 36 − 2·8 = 20 px; MD3 draws 24 dp icons inside its 40 dp buttons. Sizing
// the glyph to the content box keeps the icon in lockstep with the control
// across densities — an icon that stays put while its control shrinks is the
// tell that density is only half-wired — and it is exactly the glyph size
// components/button gives an icon-only button (side ControlHeight, inset PaddingY).
func Size(d tokens.Density) unit.Dp {
	return unit.Dp(d.ControlHeight - 2*d.PaddingY)
}

// DefaultSize is the icon size at the default density, Size(tokens.Comfortable):
// 20 dp. Callers outside a density-aware pipeline (legacy static render paths)
// use it the way they use tokens.Comfortable itself.
var DefaultSize = Size(tokens.Comfortable)

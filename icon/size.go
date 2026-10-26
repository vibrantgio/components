package icon

import (
	"gioui.org/unit"

	"github.com/vibrantgio/theme/tokens"
)

// Size returns the size of a symbol in a control for a density:
// [tokens.Density.SymbolSize], 20 dp at tokens.Comfortable and 19 dp at
// tokens.Compact. It is a value of its own, and no padding derives it. A
// former padding derivation gave these values. A symbol's size in a control
// waits for a measurement, which replaces them. It is exactly the symbol size
// components/button gives an icon-only button, centred in a ControlHeight
// square.
func Size(d tokens.Density) unit.Dp {
	return unit.Dp(d.SymbolSize)
}

// DefaultSize is the icon size at the default density, Size(tokens.Comfortable):
// 20 dp. Callers outside a density-aware pipeline (legacy static render paths)
// use it the way they use tokens.Comfortable itself.
var DefaultSize = Size(tokens.Comfortable)

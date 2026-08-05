package tokens

import "github.com/vibrantgio/spectrum/tokens"

// Type aliases — downstream code sees types identical to spectrum's.
type (
	// ColorScale is an alias for [tokens.ColorScale].
	ColorScale = tokens.ColorScale
	// ColorTokens is an alias for [tokens.ColorTokens].
	ColorTokens = tokens.ColorTokens
	// TypeScale is an alias for [tokens.TypeScale].
	TypeScale = tokens.TypeScale
	// SpacingScale is an alias for [tokens.SpacingScale].
	SpacingScale = tokens.SpacingScale
	// RadiusScale is an alias for [tokens.RadiusScale].
	RadiusScale = tokens.RadiusScale
	// ElevationScale is an alias for [tokens.ElevationScale].
	ElevationScale = tokens.ElevationScale
	// ElevationLevel is an alias for [tokens.ElevationLevel].
	ElevationLevel = tokens.ElevationLevel
	// Bezier is an alias for [tokens.Bezier].
	Bezier = tokens.Bezier
	// MotionScale is an alias for [tokens.MotionScale].
	MotionScale = tokens.MotionScale
)

// Elevation level constants, re-exported from spectrum's tokens package.
const (
	Level0 = tokens.Level0
	Level1 = tokens.Level1
	Level2 = tokens.Level2
	Level3 = tokens.Level3
	Level4 = tokens.Level4
	Level5 = tokens.Level5
)

// Variable re-exports. These are init-time copies of spectrum's values; the
// scales are plain value-typed structs that both packages document as
// read-only, so the copies are indistinguishable in correct use.
var (
	// Slate re-exports [tokens.Slate].
	Slate = tokens.Slate
	// Blue re-exports [tokens.Blue].
	Blue = tokens.Blue
	// Red re-exports [tokens.Red].
	Red = tokens.Red
	// White re-exports [tokens.White].
	White = tokens.White
	// DefaultLight re-exports [tokens.DefaultLight].
	DefaultLight = tokens.DefaultLight
	// DefaultDark re-exports [tokens.DefaultDark].
	DefaultDark = tokens.DefaultDark
	// DefaultTypeScale re-exports [tokens.DefaultTypeScale].
	DefaultTypeScale = tokens.DefaultTypeScale
	// Spacing re-exports [tokens.Spacing].
	Spacing = tokens.Spacing
	// Radius re-exports [tokens.Radius].
	Radius = tokens.Radius
	// Elevation re-exports [tokens.Elevation].
	Elevation = tokens.Elevation
	// Motion re-exports [tokens.Motion].
	Motion = tokens.Motion
)

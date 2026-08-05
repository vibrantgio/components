package theme

import "github.com/vibrantgio/spectrum/theme"

// Theme is an alias for [theme.Theme] — downstream code sees a type identical
// to spectrum's.
type Theme = theme.Theme

// Default re-exports [theme.Default].
var Default = theme.Default

// AutoLightDark re-exports [theme.AutoLightDark].
var AutoLightDark = theme.AutoLightDark

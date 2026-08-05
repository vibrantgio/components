package a11y

import "github.com/vibrantgio/spectrum/a11y"

// Type aliases — downstream code sees types identical to spectrum's.
type (
	// A11yPrefs is an alias for [a11y.A11yPrefs].
	A11yPrefs = a11y.A11yPrefs
	// Source is an alias for [a11y.Source].
	Source = a11y.Source
)

// Function re-exports. These are the same functions, not wrappers, so an
// observable built through this path is indistinguishable from one built
// through spectrum's.
var (
	// FromSource re-exports [a11y.FromSource].
	FromSource = a11y.FromSource
	// Live re-exports [a11y.Live].
	Live = a11y.Live
)

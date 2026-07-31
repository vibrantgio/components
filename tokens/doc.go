// Package tokens holds the typed design values the whole system is styled
// from: eleven-stop colour scales taken verbatim from Tailwind, the semantic
// ColorTokens pairs built out of them, the Material Design 3 type scale in dp,
// and the 4-pt spacing, radius, elevation and motion scales.
//
// Reach for it when you draw something yourself and want a value that matches
// the components around it — a pane background, a gap, a corner radius, an
// animation duration — instead of inventing a number. Components do not import
// this package for their values: they read the observables on a theme.Theme,
// and DefaultLight, DefaultDark, DefaultTypeScale, Spacing, Radius, Elevation
// and Motion are what those observables carry by default.
//
// Every scale is a plain comparable struct of float32 device-independent
// pixels, except MotionScale, whose duration stops are time.Duration. The
// package-level instances are variables rather than constants, so treat them
// as read-only: mutating one changes it for every consumer in the process.
// Copy and edit a value instead, which is how a custom theme is built.
package tokens

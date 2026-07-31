// Package theme carries the design tokens a component reads while it lays
// out. A Theme is one rx.Observable per token category — colour, type,
// motion, spacing, radius, elevation — so a consumer subscribes to only the
// categories it actually reads and rebuilds only when those change.
//
// Reach for it whenever you build an observable prism component: button.Button,
// input.TextField and their siblings all take an rx.Observable[theme.Theme] as
// their first argument, and a window that emits a new Theme follows the
// appearance change with no application code. Default() returns a static light
// theme, emitted once, which is what tests and static rendering want.
//
// Two assumptions are worth knowing. Theme is a plain struct of observables
// and owns no state, so a Theme you assemble field by field must have every
// field set — a missing one is a nil observable, not a default. And
// AutoLightDark() reads the wall clock, not the OS: hours 7 through 17 are
// light and the rest dark. For real OS appearance and accent tracking use
// spectrum's system.LiveTheme instead.
//
// This contract moves down into github.com/vibrantgio/spectrum in a later
// release, so that the theme runtime sits beneath the components it themes;
// aliases will keep this import path working. See the repository README.
package theme

// Package a11y publishes the operating system's accessibility display
// preferences — reduce motion, increased contrast, larger text — as an
// rx.Observable[A11yPrefs] that emits only when a value actually changes.
//
// Reach for it when behaviour should follow the user's system settings rather
// than an application toggle: gate an animation on ReduceMotion, pick a
// higher-contrast token set on HighContrast. Subscribe once near the top of a
// layer and share the observable downstream; pass your own Source to
// FromSource to stub the OS out in a test.
//
// It assumes polling. Nothing here is push-based, so Live takes an interval
// and one second is the intended default — the platform caches these
// properties and will not report a toggle much faster. macOS and Windows
// report real preferences; Linux returns the zero A11yPrefs, all false,
// because there is no reliable cross-desktop API without depending on a
// particular desktop environment. A11yPrefs is deliberately all comparable
// fields: change detection is struct equality, so adding a slice or map field
// would break it.
package a11y

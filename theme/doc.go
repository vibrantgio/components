// Package theme is a forwarding shim for github.com/vibrantgio/spectrum/theme,
// which is where the theme contract now lives: the token and theme contract
// moved down from prism into spectrum (ADR-001) so the theme runtime sits
// beneath the components it themes. Every identifier here is a type alias or
// re-export of its spectrum counterpart, so existing imports of this path keep
// compiling unchanged for one release cycle; the shim is removed in the next
// major release of prism.
//
// Deprecated: use github.com/vibrantgio/spectrum/theme instead.
package theme

// Package tokens is a forwarding shim for github.com/vibrantgio/spectrum/tokens,
// which is where the typed design values of the system now live: the token and
// theme contract moved down from prism into spectrum (ADR-001) so the theme
// runtime sits beneath the components it themes. Every identifier here is a
// type alias or re-export of its spectrum counterpart, so existing imports of
// this path keep compiling unchanged for one release cycle; the shim is
// removed in the next major release of prism.
//
// Deprecated: use github.com/vibrantgio/spectrum/tokens instead.
package tokens

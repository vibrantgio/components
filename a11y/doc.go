// Package a11y is a forwarding shim for github.com/vibrantgio/spectrum/a11y,
// which is where the OS accessibility-preference observables now live: the
// source moved down from prism into spectrum (ADR-001, E3.2) so the theme
// runtime that composes reduce-motion and high-contrast into its emissions
// sits beneath the components it themes. Every identifier here is a type
// alias or re-export of its spectrum counterpart, so existing imports of this
// path keep compiling unchanged for one release cycle; the shim is removed in
// the next major release of prism.
//
// Deprecated: use github.com/vibrantgio/spectrum/a11y instead.
package a11y

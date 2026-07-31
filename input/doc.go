// Package input provides the VibrantGio form controls — TextField, Checkbox,
// Radio and Dropdown — on the same contract as prism/button: an
// rx.Observable[theme.Theme] and a props struct in, an
// rx.Observable[layout.Widget] out, with a matching pure Render, RenderCheckbox,
// RenderRadio and RenderDropdown path that takes resolved tokens and an explicit
// render state and draws one frame without handling events.
//
// Reach for it for form input inside an MVU or FRP layer. Every control
// reports both ways: through a Props callback — OnChange, OnSubmit — which is
// handed the frame's layout.Context so it can emit a message from inside the
// callback, or through Props.Message, which adds an mvu.MessageOp to the
// frame's ops for the runtime to deliver to Update.
//
// The text field is uncontrolled. Props.Seed pre-fills a newly created
// instance so an existing value can be edited rather than retyped, but a later
// Seed does not touch a live instance — rebuild the field, keyed on an epoch,
// to reseed it. TextField and Dropdown draw text and therefore need
// Props.Shaper: leave it nil and they silently build a Go-fonts shaper for
// themselves and render in the wrong typeface. Checkbox and Radio draw no text
// and need no shaper.
package input

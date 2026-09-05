// Package input provides the Vibrant Gio form controls — TextField, Checkbox
// and Radio — on the same contract as components/button: an
// rx.Observable[theme.Theme] and a props struct in, an
// rx.Observable[layout.Widget] out, with a matching pure Render, RenderCheckbox
// and RenderRadio path that takes resolved tokens and an explicit render state
// and draws one frame without handling events.
//
// Reach for it for form input inside an MVU or FRP layer. Every control
// reports both ways: through a Props callback — OnChange, OnSubmit — which is
// handed the frame's layout.Context so it can emit a message from inside the
// callback, or through Props.Message, which adds an mvu.MessageOp to the
// frame's ops for the runtime to deliver to Update.
//
// Pick-one-from-many is components/picker's, where one menu serves both a
// form-variant trigger and a chrome-variant one. Dropdown and RenderDropdown
// forward to picker.Field and picker.RenderField and are deprecated;
// DropdownProps and DropdownRenderState are aliases of picker's, so props and
// states written against either name are the same values.
//
// The text field, checkbox and radio inner fills paint the level above the
// surface they stand on and sit in the page plane; none of them raises a plane
// above the page.
//
// The text field is uncontrolled. Props.Seed pre-fills a newly created
// instance so an existing value can be edited rather than retyped, but a later
// Seed does not touch a live instance — rebuild the field, keyed on an epoch,
// to reseed it. TextField draws text in the theme's BodyLarge role, shaped
// with the theme's shaper (Typography.Shaper()); Props.Shaper is an explicit
// per-instance override for the rare case where one control must shape with a
// different shaper than the theme provides. Checkbox and Radio draw no text
// and need no shaper.
package input

// Package input provides the Vibrant Gio form controls — TextField, Checkbox,
// Radio and Dropdown — on the same contract as components/button: an
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
// Elevation (goal G-E2): the controls themselves are flat — the text
// field, checkbox and radio inner fills and the closed dropdown trigger
// paint the plain Surface token, sitting in the page plane. The one
// raised plane in the package is the dropdown's open option menu, a
// floating unscrimmed, shadowless transient overlay: its rows fill at
// level 3 on the ladder (Neutral step 400 on the default scale), the same
// rung patterns/popover takes. The selected row leaves the ladder — it is
// the theme's inverse pair, the one plane in the menu built from the
// counterpart scheme, because a state walk on a mid-grey ground leaves no
// ink able to read on it; optionRowColors carries the measurements.
//
// The text field is uncontrolled. Props.Seed pre-fills a newly created
// instance so an existing value can be edited rather than retyped, but a later
// Seed does not touch a live instance — rebuild the field, keyed on an epoch,
// to reseed it. TextField and Dropdown draw text in the theme's BodyLarge
// role, shaped with the theme's shaper (Typography.Shaper()); Props.Shaper is
// an explicit per-instance override for the rare case where one control must
// shape with a different shaper than the theme provides. Checkbox and Radio
// draw no text and need no shaper.
package input

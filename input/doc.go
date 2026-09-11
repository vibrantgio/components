// Package input provides the Vibrant Gio form controls — TextField,
// SearchField, Checkbox and Radio — on the same contract as
// components/button: an rx.Observable[theme.Theme] and a props struct in, an
// rx.Observable[layout.Widget] out, with a matching pure Render,
// RenderSearch, RenderCheckbox and RenderRadio path that takes resolved
// tokens and an explicit render state and draws one frame without handling
// events.
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
// Every colour these controls draw is a platform colour taken by name. A
// field and an unchecked box are filled with the platform's text background
// and edged with its measured field edge; a checked box and a chosen radio
// are its accent, marked in the foreground it names for text on that accent;
// a disabled control's foreground is its disabled control text; and a focused
// control wears its keyboard focus indicator. None of them measures, walks or
// derives a colour of its own.
//
// A text field takes the density's field height rather than its control
// height: the platform draws a field shorter than the button standing beside
// it, and that is the one place in the library where the two heights part.
//
// The search field is the text field with two slots added to its structure:
// the looking glass that names it, drawn leading, and the clear mark that
// empties it, drawn trailing while there is a query to take back. Both are
// spent out of the field's own width, so a search field and a text field of
// the same width are the same size and the text does not reflow when the
// mark appears. Both are drawn in the platform's secondary label. Clearing
// reports the empty query through the same OnChange and Message a keystroke
// does, which is what lets a consumer's highlight die with the query that
// caused it; what is found is the consumer's, and the fill it marks a match
// with is the platform's find highlight, which leaves the text it covers its
// own colour.
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

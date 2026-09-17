package control

import "gioui.org/unit"

// TextLeadDp is the leading inset a control spends on the text standing in
// it: its inner edge to the origin that text is laid from. The text field,
// the picker's field trigger and the rows of the menu that trigger drops all
// spend it, which is what puts a picker's value on the same column as the
// value of a text field beside it.
//
// MEASURED, save-dialog-{light,dark}.png, the "Save As:" field at 1x: the
// field's box runs x 264–495, the columns the unfocused "Tags:" field below
// it runs, so its fill begins at x=265. The selection standing behind the
// value fills from x=271 and the first covered pixel of "Untitled" is at
// x=272 — the text's origin six columns in from the inner edge and its first
// covered pixel seven, the U carrying one column of left side bearing in the
// face the platform sets the field in. Both appearances give both columns.
//
// The rule the seven is spent by: what a control spends is the origin, and a
// face adds its own first glyph's bearing to whatever origin it is given, so
// the origin is the measured seven less the bearing the capture's own first
// letter carries — six here, and a face whose glyph bears one column then
// covers the platform's column exactly.
//
// It is spent at the control's resting edge whatever the control's state, so
// focus — which draws a wider ring in the edge's place — does not move the
// text.
const TextLeadDp unit.Dp = 6

// TextTrailDp is the trailing inset, [TextLeadDp] mirrored.
//
// MIRRORED, not measured: no stored capture holds a value that reaches a
// field's trailing edge. The save dialog's only filled field is the focused
// "Save As:", whose "Untitled" ends at x=317 against an inner edge at x=494;
// its "Tags:" field is empty; and the trailing mark of the File Format
// pop-up beside them is a chevron the control draws rather than a value. A
// control is symmetric end to end until a capture says otherwise, and the
// capture that would say so is on the reference's capture list.
const TextTrailDp unit.Dp = 6

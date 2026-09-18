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

// PopupLeadDp is the leading inset the platform's pop-up button spends on its
// own label — a different control from the text field above, and a deeper
// inset. The picker's form trigger is drawn as that pop-up, so it spends this
// rather than [TextLeadDp]; the rows of the menu it drops are not pop-ups and
// keep the field's.
//
// MEASURED, save-dialog-{light,dark}.png, the "File Format:" pop-up at 1x: its
// fill runs x 264–451 with no edge column — a run down x=350 gives #ececec
// light and #333a3f dark from the control's first row to its last, so its
// outer edge and its inner edge are one — and the first covered column of its
// label, "Script", is x=276 in both appearances: twelve columns in, five
// further than the field's seven.
//
// The same rule the field's inset is spent by: what a control spends is the
// origin, and a face adds its own first glyph's bearing to it, so the origin
// is the measured twelve less the one column of bearing the capture's face
// carries — eleven. The bearing is taken from the "Untitled" of the field
// above, where the selection behind the value exposes the origin (x=271) one
// column before the first covered pixel (x=272, 90% covered): the "Script"
// label opens on a capital S, a curve whose own leading column carries only a
// 16% fringe light and 28% dark — that places where the glyph starts but
// cannot separate the origin from the bearing on its own.
const PopupLeadDp unit.Dp = 11

// PopupMarkTrailDp is the clear room a pop-up leaves between the last pixel of
// its mark and its inner trailing edge. It is the trigger's trailing inset:
// what a pop-up puts at that end is the mark, not a value.
//
// MEASURED, save-dialog-{light,dark}.png: the "File Format:" pop-up's chevron
// pair spans x 435–442 against a fill ending at x=451, nine clear columns.
// Read twice over: the same nine stands between a Mail toolbar pull-down's
// chevron and its own trailing edge in a control 29 px tall, so the room is a
// fixed one and not a ratio of the control's height.
const PopupMarkTrailDp unit.Dp = 9

// ChromeMarkDp is the box a symbol standing alone in a toolbar control is
// drawn in — the size the chrome variant hands its mark, and the top of the
// range components/icons is authored for.
//
// MEASURED at 1x, the toolbar bands of finder-window-light.png,
// notes-toolbar.png and voicememos-window.png. A symbol's covered extent
// there reads 18 × 18 px (Finder's group pull-down), 19 × 19 (its tag),
// 17 × 12 (its list), 16 × 17 (its magnifier), 17 × 17 (Notes' compose) and
// 19 × 15 (Voice Memos' sidebar toggle): a square form fills about 18 px and
// a round or diagonal one about 17. That is the icon set's own grid at 24 —
// a square form to its 18-unit keyline and a round one to 20 units on a
// 24-unit grid — so 24 is the size at which the set draws the platform's
// measured extent, and the size at which every unit of that grid lands on a
// whole pixel.
//
// The weight is the set's one weight and stands with its miss stated. The
// axis-aligned band of those same symbols measures 1.12 px (Finder's list
// bar), 1.15 to 1.22 (its magnifier's circle), 1.26 (Notes' compose) and 1.39
// (Voice Memos' sidebar rectangle), against the 1.5 px the set's 1.5-unit
// band draws at 24. The set is a sixth heavier, and a lighter one is not
// available: 1.25 units falls to 0.83 px at the 16 dp end of the range, below
// one device pixel, where an antialiased line is drawn grey rather than in
// the control's colour.
const ChromeMarkDp unit.Dp = 24

// ChromeMarkSideDp is the clear room a toolbar control leaves on each side of
// that box, and so the width the chrome variant draws: [ChromeMarkDp] plus
// twice this.
//
// MEASURED at 1x: a toolbar control carrying one symbol and nothing else
// measures 38 px wide against its 36 px height in mail-window.png (the
// compose control, x 404–441), 37 in notes-toolbar.png (x 8–44) and 40 in
// voicememos-window.png (the sidebar toggle, x 96–135); Mail's three-segment
// group divides to 37.3 a segment. Around a 24 px box those leave 7, 6.5 and
// 8 columns a side. Seven is the middle reading and lands Mail's control
// exactly; the platform's own spread across four readings is three columns.
const ChromeMarkSideDp unit.Dp = 7

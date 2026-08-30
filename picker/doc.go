// Package picker provides the Vibrant Gio pick-one-from-many affordance: a
// trigger that names the current value and a menu that offers the alternatives.
//
// It is one component with two triggers and one open surface, because the two
// triggers differ only in which register they are drawn for:
//
//	[Field]   the FORM register — the flat bar that stands in a form beside a
//	          text field and a checkbox, at the same control height, with the
//	          same bezel and the same focus ring
//	[Anchor]  the CHROME register — the platform's pop-up control, at the
//	          button's rounded-rect corner with the paired up/down chevrons,
//	          for a toolbar, a header row or any other furniture
//	[Menu]    the surface both of them stand under: the level-3 rows, the
//	          accent selection and the accent's quieter wash under the
//	          pointer
//
// Each has a live path and a pure one, the contract every component in this
// library keeps: an rx.Observable[theme.Theme] and a props struct in, an
// rx.Observable[layout.Widget] out, with a matching [RenderField],
// [RenderAnchor] and [RenderMenu] that take resolved tokens and an explicit
// render state and draw one frame without handling events.
//
// # Single choice by contract
//
// The trigger shows the value, so there is exactly one. Few-of-few selection
// is not this component's — it is a filter, drawn as a row of chips a reader
// can see the state of at a glance, and a picker that summarised several
// choices on its trigger would be describing a set through a control shaped
// like a scalar.
//
// # Where the menu is placed
//
// [Field] stacks its menu against its own trigger, which is what a form's
// select does and what its [FieldState.Open] draws — beneath by default, and
// above it when the caller says [DropUp] because the room below is somebody
// else's. Either way the open field is one widget reporting one box, and an
// upward field is placed by that box's bottom edge. [Anchor] does not:
// a chrome-register menu is a floating surface placed against the window, and
// placing it is patterns/popover's job — a component may not reach up into a
// pattern. So an anchor's caller hands the anchor to the popover as the anchor
// slot and a [Menu] as the content slot, and the two meet there.
//
// Either way the surface is the same one: a floating, unscrimmed, shadowless
// transient plane whose rows fill at level 3 on the elevation ladder and whose
// coloured rows are the accent — the selection at the weight text is held to
// against the plane, the pointer's wash at the accent's own container depth.
// [Menu]'s optionRowColors carries the measurements.
//
// Who draws the plane's EDGE depends on who placed the plane. [Field] draws it
// around the menu it stacks, because inline there is nobody else to; a [Menu]
// handed to patterns/popover is circled by that pattern's surface and draws
// none of its own, which is the only arrangement in which the plane wears one
// line.
//
// # The chrome register's trigger
//
// [Anchor] is a face of the chip family rather than a restyle of one: it keeps
// every one of the chip's own answers — the measured fill, the two-sided rim,
// the walked inks, the focus ring that replaces that rim, the density's height
// and padding, the 44 dp pointer target, the pin — and changes exactly two
// things. components/chip's package doc carries the derivations those answers
// come from.
//
// THE CORNER. The scale's Md stop, the same one components/button reads for
// every register it draws. The platform's pop-up control is a rounded
// rectangle where its toolbar capsules are pills, and the rounded rectangle
// this system already owns is the button's; the anchor reads that stop rather
// than naming a number, so the two cannot drift.
//
// THE MARK. The paired up/down chevrons, drawn by the component, not passed
// in. Two consequences, both deliberate. A caller cannot put a different mark
// on a pop-up anchor, because the mark is what says the control pops up. And
// a caller cannot FLIP it: on this platform the pair says "this pops up" and
// never "this is open", so the trigger carries no open state and offers
// nowhere to hang one. An anchor whose glyph turned over when its menu stood
// would be speaking the vocabulary the platform reserves for a disclosure
// triangle. It is also why the two triggers do not share one state: the
// field's mark and its menu are one widget, and the anchor's menu is
// somewhere else on the window.
//
// The chevrons' proportions are measured off the stored macOS reference and
// their internal spacing is not; the constants say which is which, along with
// the capture the numbers were read from and the one figure the reference
// cannot answer.
//
// # Uncontrolled
//
// The live paths keep their own selection, seeded by props on subscribe: a
// later Selected does not move a running instance. Rebuild the subscription,
// keyed on whatever the value is a function of, to reseed one — the idiom
// components/input's text field states for the same reason.
//
// Shaper is not optional in the pure paths. Pass the theme's —
// tokens.Typography.Shaper() — or, in a golden test, its DeterministicShaper.
package picker

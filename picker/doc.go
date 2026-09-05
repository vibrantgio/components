// Package picker provides the Vibrant Gio pick-one-from-many affordance: a
// trigger that names the current value and a menu that offers the alternatives.
//
// It is one component with two triggers and one open surface, because the two
// triggers differ only in which variant they are drawn for:
//
//	[Field]    the FORM variant — the flat bar that stands in a form beside a
//	           text field and a checkbox, at the same control height, with the
//	           same bezel and the same focus ring
//	[Toolbar]  the CHROME variant — the platform's pull-down control, at the
//	           button's rounded-rect corner with a single down chevron, for a
//	           toolbar, a header row or any other furniture
//	[Menu]     the surface both of them stand under: the level-3 rows, the
//	           accent selection and the accent's less pronounced state fill under
//	           the pointer
//
// Each has a live path and a pure one, the contract every component in this
// library keeps: an rx.Observable[theme.Theme] and a props struct in, an
// rx.Observable[layout.Widget] out, with a matching [RenderField],
// [RenderToolbar] and [RenderMenu] that take resolved tokens and an explicit
// render state and draw one frame without handling events.
//
// # Single choice by contract
//
// The trigger shows the value, so there is exactly one. Few-of-few selection
// is not this component's — it is the Filter chip's, drawn as a row a reader
// can see the state of at a glance, and a picker that summarised several
// choices on its trigger would be describing a set through a control shaped
// like a scalar.
//
// # Where the menu is placed
//
// [Field] stacks its menu against its own trigger, which is what a form's
// select does and what its [FieldState.Open] draws — beneath by default, and
// above it when the caller says [DropUp] because the room below is somebody
// else's. Either way the open field is one component reporting one box, and an
// upward field is placed by that box's bottom edge, with its trigger's triangle
// pointing the way its menu goes. [Toolbar] does not:
// a chrome-variant menu is a floating surface placed against the window, and
// placing it is patterns/popover's job — a component may not reach up into a
// pattern. So a toolbar trigger's caller hands the trigger to the popover as
// the anchor slot and a [Menu] as the content slot, and the two meet there.
//
// Either way the surface is the same one: a floating, unscrimmed, shadowless
// transient plane whose rows fill at level 3, the top of the elevation, and
// whose coloured rows are the accent — the selection at the weight text is held
// to against the plane, the pointer's state fill at the accent's own container
// depth.
// [Menu]'s optionRowColors carries the measurements.
//
// Who draws the plane's EDGE depends on who placed the plane. [Field] draws it
// around the menu it stacks, because inline there is nobody else to; a [Menu]
// handed to patterns/popover is circled by that pattern's surface and draws
// none of its own, which is the only arrangement in which the plane wears one
// line.
//
// # The chrome variant's trigger
//
// [Toolbar] is the platform's pull-down control, drawn from the measured
// geometry components/internal/toolbarface holds: the fill a measured step over
// the surface it stands on and walked by the pointer, the rim derived against
// both of its sides, the foregrounds resolved against the fill actually drawn, the
// focus ring that replaces that rim, the density's height and padding, the
// 44 dp pointer target, the pin. [ToolbarFill] is that fill, for a caller that
// must clear it. Two things are the toolbar trigger's own.
//
// THE CORNER. The scale's Md stop, the same one components/button reads for
// every variant it draws. The platform draws its pop-up control as a rounded
// rectangle, and the rounded rectangle this system already owns is the
// button's; the trigger reads that stop rather than naming a number, so the two
// cannot drift.
//
// THE MARK. A single down chevron, drawn by the component, not passed in.
// Three consequences, all deliberate. A caller cannot put a different mark on
// a toolbar trigger, because the mark is what says the control opens a menu. A caller
// cannot FLIP it: on this platform a pull-down's chevron says "a menu opens
// below this" and never "this is open", so the trigger carries no open state
// and offers nowhere to hang one; a trigger whose glyph turned over when its
// menu stood would be speaking the vocabulary the platform reserves for a
// disclosure triangle. And the mark is a claim about PLACEMENT, so the caller
// owes it: whatever places the menu — patterns/popover, in the only
// arrangement this component has — must place it below the trigger, because a
// mark that announces a direction the menu does not take is a defect and not a
// style. It is also why the two triggers do not share one state: the field's
// mark and its menu are one component, and the toolbar's menu is somewhere else on
// the window.
//
// THE PAIRED CHEVRONS, and what they would take. The platform's OTHER
// menu-bearing control is the pop-up button: its menu stands OVER the trigger
// with the selected row aligned on it, and it wears an up chevron over a down
// one to say the choice can move either way. That is a different control, so
// it is a different face rather than a flag on this one, and it may only be
// drawn once something places a menu that way — no caller does. Two things are
// owed before it can be: a placement that stands the menu over the trigger with
// the selected row on it, and a macOS pop-up-button capture in the platform
// reference. The stored reference holds only single-chevron pull-downs, so the
// air between the pair's halves — whether the platform narrows or flattens
// each half rather than spacing them — is a number nothing here can answer.
//
// The chevron's proportions are measured off that stored reference; the
// constants in components/internal/toolbarface carry the capture and the figures.
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

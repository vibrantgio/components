// Package picker provides the Vibrant Gio pick-one-from-many affordance: a
// trigger that names the current value and a menu that offers the alternatives.
//
// It is one component with two triggers and one open surface, because the two
// triggers differ only in which variant they are drawn for:
//
//	[Field]    the FORM variant — the platform's pop-up button, standing in a
//	           form beside a text field and a checkbox at the same control
//	           height, with the same focus ring and the platform's stacked
//	           chevron pair as its mark
//	[Toolbar]  the CHROME variant — the same pop-up drawn for a toolbar, a
//	           header row or any other chrome region: a capsule carrying its
//	           own fill, with the same stacked chevron pair
//	[Menu]     the surface both of them stand under: the platform's menu
//	           rows, with its selection fill on the chosen row and on the row
//	           under the pointer
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
// [Field] floats its menu against its own trigger, which is what a form's
// select does and what its [FieldState.Open] draws — beneath by default, and
// above it when the caller says [DropUp] because the room below is somebody
// else's. Either way the box the field reports is the trigger's alone, so an
// open field is placed exactly where a closed one is, and the trigger draws
// the same mark either way — see [Drop]. [Toolbar] does not:
// a chrome-variant menu is a floating surface placed against the window, and
// placing it is patterns/popover's job — a component may not reach up into a
// pattern. So a toolbar trigger's caller hands the trigger to the popover as
// the anchor slot and a [Menu] as the content slot, and the two meet there.
//
// The menu the field places itself is a floating surface, so it is drawn like
// one and it costs like one: the trigger stays where the caller put it and the
// menu goes through op.Defer, painting and hit-testing above everything the
// window lays out after the field's slot — patterns/popover's package doc
// states the idiom — while the field asks its container for the trigger's
// height and no more. It goes with its trigger too: a scroll that carries the
// field away takes the menu with it, the way a press elsewhere and Escape do.
//
// # The available room
//
// A floating surface has to land whole, so the field fits its menu to the room
// there actually is. [FieldProps.AvailableRoom] is how it learns that room —
// the pixels above the trigger's top edge and below its bottom edge, inside
// the container that laid the field out — and it is the container's to answer,
// because Gio hands a component its constraints and nothing about where its
// ancestors put it. Given the room, [Drop] becomes a preference: a menu that
// cannot be seen whole on the preferred side while the other side holds more
// of it flips to that other side, and either way the plane is capped to what
// the chosen side leaves and the rows scroll inside the cap.
// [FieldProps.MaxHeight] is a preference of the same kind — the room may
// tighten it and can never loosen it. A caller that reports no room is
// bounded by the window alone, which is what a floating surface is bounded by
// when nobody says otherwise, and a catalogue opened in one wants a MaxHeight.
//
// Either way the surface is the same one: a floating, unscrimmed, shadowless
// transient plane whose rows take the platform's control background under its
// label, and whose chosen row — and the row under the pointer, which this
// platform marks the same way — takes its emphasized selection fill.
// [Menu]'s optionRowColors names them.
//
// Who draws the plane's EDGE depends on who placed the plane. [Field] draws it
// around the menu it places itself, because there is nobody else to; a [Menu]
// handed to patterns/popover is circled by that pattern's surface and draws
// none of its own, which is the only arrangement in which the plane wears one
// line.
//
// # The chrome variant's trigger
//
// [Toolbar] is the same pop-up drawn for a chrome region, from the measured
// geometry components/internal/toolbarface holds: the platform's own toolbar
// control fill at rest and that fill under the platform's overlays under the
// pointer and while held, the rim of its seam, the value and the mark both in
// its control text, the focus ring that replaces that rim, the density's
// height, the pointer target that control is, the pin. [ToolbarFill] is the
// fill the trigger draws, for a caller that must know. Two things are the
// toolbar trigger's own.
//
// THE FILL. The platform gives a control standing in a toolbar a fill of its
// own, lighter than the band it stands on, so the control is a figure on the
// band and not part of it. MEASURED, finder-window-untinted-dark.png, a
// frontmost window: every bordered control in its toolbar reads #262626
// against a band of #1e1e1e; the light value is the same control's #ffffff in
// finder-window-light.png. It does not depend on what the control stands on,
// which is why the trigger takes no surface.
//
// THE CORNER. Fully rounded — half the control's height, where the form
// trigger takes the button's rounded rectangle. MEASURED off the same
// captures: the toolbar's controls are capsules, and the dialog's pop-up is
// not. It is the one place the two variants differ in shape.
//
// THE MARK is NOT the toolbar trigger's own. Both variants wear the
// platform's pop-up mark, the stacked chevron pair, at the one size the
// platform draws it: 8 px by 11 in a 24 px control and the same 8 by 11 in a
// 36 px one, so the mark is sized by its point size and not by the control.
// components/internal/control holds the measurement and the drawing.
//
// The single chevron the platform draws beside it belongs to a PULL-DOWN — a
// menu of actions rather than a choice — and this component has none: a
// picker is single-choice by contract, so neither trigger wears it. The mark
// is drawn by the component and not passed in, and a caller cannot flip it:
// a pair of chevrons pointing opposite ways says the control holds one of
// several values and cannot say a direction, so the trigger carries no open
// state and offers nowhere to hang one.
//
// What the pop-up still owes is its PLACEMENT. On the platform a pop-up's
// menu stands OVER the trigger with the selected row aligned on it, and this
// field drops its menu beneath or above instead. That is a placement nothing
// here offers yet and no caller asks for.
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

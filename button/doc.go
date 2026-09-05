// Package button provides the Vibrant Gio button: a text or icon-only
// affordance carrying hover, focus, press and disabled treatments, activation
// by click or by Space and Enter, a screen-reader label, and a pointer target
// of at least 44 dp on each axis regardless of density (the drawn control is
// the theme Density's control height; the hit area extends beyond it when the
// control is smaller).
//
// Button is the observable path — an rx.Observable[theme.Theme] and a Props
// in, an rx.Observable[layout.Widget] out, rebuilt whenever the theme changes
// — and activations leave through either Props.OnClick, which is handed the
// frame's layout.Context, or Props.Message, which adds an mvu.MessageOp to the
// frame's ops for an MVU runtime to deliver to Update. Render and RenderIcon
// are the pure path: resolved tokens plus an explicit RenderState in, one
// frame out, no event handling — that is what the golden-image tests drive and
// what static rendering should use.
//
// # Emphasis
//
// A button also carries a visual weight — the [Filled] variant, the default,
// [Tonal] and [Ghost] — set through Props.Emphasis or, on the pure path,
// through RenderState.Emphasis. It is a colour axis and only a colour axis:
// filled is the accent's pinned solid fill under its on-colour; tonal is the
// accent's tint over the surface the button stands on, under the accent's
// own colour at the text floor — never a neutral label, and the same recipe
// a status badge wears, because two near-identical tints would be one recipe
// spelled twice; ghost paints no fill at all under the neutral ramp's 700
// text. The drawn size and the 44 dp pointer floor are identical in all
// three, and so is the focus ring's shape, width and place — only its step
// moves, and only so far as the fill under it moved. Focus is a persistent
// state in every variant: the resting fill stays and the ring is added to
// it. The least pronounced variant is not the smallest one and is no harder
// to see with a keyboard. The zero value is Filled, so nothing written
// before the axis existed renders differently.
//
// Tonal reads the level it is given (Props.Level, RenderState.Level), as a
// ghost does and a filled button does not: its tint is derived against the
// surface underneath, so a Tonal button in a dialog is tinted against the
// dialog rather than against the window.
//
// Emphasis says how important an action is on the surface it sits on, and
// nothing more. Marking a choice is never a button's job, whatever its
// emphasis: a persistent selection is the Filter chip's purpose
// (components/chip). No variant here records a picked state and none is
// asked to stand for one.
//
// # A pinned fill
//
// The filled variant alone will take a fill from its caller. Set both halves
// of Props.Fill and Props.OnFill — RenderState.Fill and OnFill on the pure
// path — and the button wears that fill under that foreground in place of the
// primary pair, keeping every treatment the variant has: the walk toward the
// 900 end under the pointer, the disabled opacity over both halves, and a
// focus ring measured against the fill that came back. It is for the action
// whose colour is not the theme's to choose — a meaning that fixes its own
// shade, which a paired status ramp would restate one way in light and
// another in dark. The two are one pin, honoured only together, and their
// zero value changes nothing: a button that names neither half is the button
// that was there before.
//
// Three things it assumes. Interaction state is allocated inside the
// component's rx.Defer scope, so press and focus survive the view rebuilds an
// MVU loop drives; pass Props.Clickable when an enclosing container such as a
// modal must own the focus tag instead. A button fills the width it is given
// and is at least the density's control height tall (36 dp Comfortable, 28 dp
// Compact), so a fixed-size button is laid out inside a constrained box. And
// Props.Shaper is not optional today — leave it nil and
// the button builds a Go-fonts shaper for itself, with no warning, and renders
// in the wrong typeface.
package button

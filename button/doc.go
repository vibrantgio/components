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
// A button also carries a visual weight register — [Filled], the default,
// [Tonal] and [Ghost] — set through Props.Emphasis or, on the pure path,
// through RenderState.Emphasis. It is a colour axis and only a colour axis:
// each register resolves its ground and its label from ADR-007's ramps (the
// pinned solid fill for filled, the primary ramp's tinted 200 ground under
// its 900 text for tonal, no ground at all under the neutral ramp's 700 text
// for ghost, with the tinted walk supplying hover and press in each). The
// drawn size and the 44 dp pointer floor are identical in all three, and so
// is the focus ring's shape, width and place — only its rung moves, and only
// so far as the ground under it moved. A quiet button is quiet, not small,
// and not harder to see with a keyboard. The zero value is Filled, so nothing written before the axis
// existed renders differently.
//
// # A pinned fill
//
// The filled register alone will take a fill from its caller. Set both halves
// of Props.Fill and Props.OnFill — RenderState.Fill and OnFill on the pure
// path — and the button wears that ground under that ink in place of the
// primary pair, keeping every treatment the register has: the walk toward the
// 900 end under the pointer, the disabled opacity over both halves, and a
// focus ring measured against the ground that came back. It is for the action
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

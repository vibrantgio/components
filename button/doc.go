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

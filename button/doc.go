// Package button provides the Vibrant Gio button: a text or icon-only
// affordance carrying focus, press and disabled treatments, activation
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
// through RenderState.Emphasis. It is a colour axis and only a colour axis,
// and each variant is one of the three buttons the platform draws: filled is
// the platform's default action, its accent under the foreground the platform
// pairs with an accent fill; tonal is the platform's ordinary button, its
// measured fill under its control text inside its hairline; ghost is the
// platform's borderless kind, no fill and no hairline under the same control
// text. The drawn size and the 44 dp pointer floor are identical in all
// three, and so is the focus ring's shape, width, place and colour. Focus is
// a persistent state in every variant: the resting fill stays and the ring is
// added to it. The least pronounced variant is not the smallest one and is no
// harder to see with a keyboard. The zero value is Filled.
//
// No variant tints under the pointer. A push button does not on this platform
// — the reference records a Finder toolbar button that does and a Save
// dialog's push button that does not — so hover changes nothing and press is
// the platform's own overlay laid over whatever fill the variant has.
// Disabled is the platform's answer rather than a fading of the resting pair:
// the platform draws a disabled default action as an ordinary disabled
// button, so a fill falls back to the push button's fill and the foreground
// becomes
// the disabled control text.
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
// platform's accent pair, keeping every treatment the variant has: the press
// overlay, the platform's disabled pair, and the focus ring, which carries a
// coverage of its own and composites over whatever fill came back. It is for
// the action whose colour is not the platform's to choose — a meaning that
// fixes its own shade in both appearances. The two are one pin, honoured only together, and their
// zero value changes nothing: a button that names neither half is the button
// that was there before.
//
// Three things it assumes. Interaction state is allocated inside the
// component's rx.Defer scope, so press and focus survive the view rebuilds an
// MVU loop drives; pass Props.Clickable when an enclosing container such as a
// modal must own the focus tag instead. A button fills the width it is given
// and is at least the density's control height tall — the platform's regular
// and small push button, 24 dp and 19 dp — so a fixed-size button is laid out
// inside a constrained box. And
// Props.Shaper is not optional today — leave it nil and
// the button builds a Go-fonts shaper for itself, with no warning, and renders
// in the wrong typeface.
package button

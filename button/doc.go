// Package button provides the Vibrant Gio button: a text or icon-only
// affordance carrying focus, press and disabled treatments, activation
// by click or by Space and Enter, a screen-reader label, and a pointer target
// that is the drawn control itself — the theme Density's control height, so
// the target moves with the pixels when density changes.
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
// measured fill under its control text, with no edge around it; ghost is the
// platform's borderless kind, no fill at all under the same control text. The drawn size, and with it the pointer target, are identical in all
// three, and so is the focus ring's shape, width, place and colour. Focus is
// a persistent state in every variant: the resting fill stays and the ring is
// added to it. The least pronounced variant is not the smallest one and is no
// harder to see with a keyboard. The zero value is Filled.
//
// Under the pointer every variant takes the platform's hover overlay over
// whatever fill it carries, and held down its press overlay over that same
// fill; a press wins, the two not being states that stack. The overlays are
// the platform's own, each read off the state captures in the organization's
// macOS reference. Disabled fades the control toward the surface it stands
// on: the fill at the platform's measured disabled coverage, the foreground
// at its disabled control text. A fill falls back to
// the push button's fill first and fades from there, because the platform
// draws a disabled default action as an ordinary disabled button.
//
// Emphasis says how important an action is on the surface it sits on, and
// nothing more. Marking a choice is never a button's job, whatever its
// emphasis: a persistent selection is the Filter chip's purpose
// (components/chip). No variant here records a picked state and none is
// asked to stand for one.
//
// # Where it stands
//
// A button also carries a [Variant] — where it stands, never how it behaves
// and never how pronounced it is. [Form] is the default: the button among
// content, at the density's control height, wearing its emphasis. [Chrome] is
// the button in a chrome region, and there a button whose label is a SYMBOL is
// the platform's bordered toolbar control: a capsule at the toolbar control's
// measured height with the symbol centred in it, its own fill, its rim where
// the platform draws one, and the drop shadow it casts on the band it stands
// on — the box the picker's chrome trigger is drawn from. The emphasis reaches
// nothing there: the platform draws one bordered toolbar control, one way.
//
// The chrome variant reaches the symbol path alone; a chrome button carrying
// text draws the form variant, because no stored toolbar band holds a control
// with a word in it to measure one from.
//
// # What each variant draws
//
// The platform draws its bordered control — the symbol button
// ([RenderBordered], [BorderedFace]) and the segmented control
// ([BorderedSegments]) — in both variants, and the variant settles every one
// of these at once:
//
//	              form                        chrome
//	height        d.ControlHeight, 24         d.ToolbarControlHeight, 36
//	face          PushButtonFill under the    ToolbarControlFill under the
//	              platform's control text,    toolbar's own label colour,
//	              no rim                      its measured rim in the dark
//	                                          appearance and none in the light
//	shadow        none                        the drop shadow the control
//	                                          casts on its band
//	shape         the push button's rounded   the capsule, half the control's
//	              rectangle at rad.Md, 6      own height
//	mark's room   seven rows above and six    the band's 24 dp box, centred
//	              below, the room the         in the control
//	              Save dialog's pop-up
//	              leaves its own
//
// Every number there is measured: 36 px in every stored toolbar band against
// the 24 the same machine's Save dialog draws a push button and a pop-up at,
// the capsule against "Cancel"'s r = 6.11 light and 6.17 dark, and the mark's
// room off the "File Format:" pop-up on that same sheet, where the band's
// 24 dp mark box in a 24 dp control would leave none. The drop shadow goes
// with the chrome variant and not with borderedness: it was measured on a
// band, and it is what tells a control from a band the platform fills with
// the same #ffffff.
//
// [RenderBordered] is the whole control on the pure path, and [BorderedShadow]
// around [BorderedFace] is the pair a caller that owns the control's own
// widget.Clickable composes, a Clickable clipping what it wraps to the box its
// layout.Widget reports while the shadow falls outside it. A form control
// needs no wrapper: it casts nothing.
//
// # A pinned fill
//
// The filled variant alone will take a fill from its caller. Set both halves
// of Props.Fill and Props.Foreground — RenderState.Fill and Foreground on the pure
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
// inside a constrained box. That box is a budget and not a cap: a label that
// does not fit it widens its own button by the label's measure rather than
// being elided into it, the way the platform sizes a push button to its
// label and holds a minimum under it. And
// Props.Shaper is not optional today — leave it nil and
// the button builds a Go-fonts shaper for itself, with no warning, and renders
// in the wrong typeface.
package button

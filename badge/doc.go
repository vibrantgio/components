// Package badge provides the Vibrant Gio badge: the system's own word about a
// thing, set inline at the size of its type and coloured by the role it
// speaks in.
//
// [Render] is the pure path: resolved tokens plus a [RenderState] naming the
// level the badge stands on, one frame out, no event handling.
// [RenderDismissible] is the same badge carrying its close mark, for a caller
// that owns the clickable. [Badge] is the live path — a theme observable and
// [Props] in, a layout.Widget out on every theme emission, with the close mark's
// pointer target and the dismissal dispatch the pure path cannot carry.
//
// # Not a control
//
// A badge is read, never used. It is off the control-height scale entirely: its
// height is its own type's line box and [tokens.Density.ControlHeight] does
// not reach it, so a badge beside a control is a fraction of that control's
// height and is meant to be. It draws no boundary, takes no pointer state on
// its body, and never grows to the width it is offered.
//
// It does wear a container, and the container is what says it is not a
// control: a pale fill, the role's hue tinted down until it is a field, with
// the same hue at reading strength on top of it. A saturated fill under
// knocked-out white content is the emphasis interaction speaks in — the filled
// button's — and a badge that borrowed it would be claiming to do something.
// One hue, two strengths, and the more pronounced pairing is not available
// here.
//
// The one thing that answers a pointer is the close mark, and what it removes
// is the label. A badge that switched something off would be a control wearing
// a badge's clothes; the affordance says "stop telling me this" and nothing
// else, and the caller owns what happens next.
//
// A control that a user operates is components/button or components/chip. A
// token the user themselves entered is a chip; a badge is applied ABOUT the
// thing by the system or the developer, which is the whole of the difference and
// the reason the two look nothing alike.
//
// # Three utterances, one structure
//
// A badge speaks as
//
//	a word    "Popular", "Beta", "Deprecated"
//	a count   "9", "128" — a word made of digits, not a second component
//	a glyph   a check, a cross, a key — the sign that stands for the verdict
//
// and there is one structure underneath all three: the type's line box tall,
// sized to what it says, in the colours its role resolves to. A count is a
// [Props.Label] of digits and needs no field of its own. A glyph is
// [Props.Glyph] with no label, drawn in the line box's own square. A glyph set
// beside a label leads it across the spacing scale's S1 stop — the sign comes
// before the word it stands for, which is also what keeps the badge from
// reading as a chip, whose mark trails its label.
//
// The utterance picks the structure in one place, and it is the only branch in
// the component: anything with words in it wears the container, and a sign on
// its own stands bare. See [Fill] for why — and for the obligation that
// carries, which is that a set of glyph badges must differ in shape, because
// a sign repeated in two variants is two hues and nothing else.
//
// # Colour: one hue at two strengths
//
// Five variants and they differ in hue alone: [Neutral] for a plain category
// label, [Success], [Warning], [Error] and [Info] for the four statuses. There
// is no emphasis axis and there will not be one — emphasis belongs where
// interaction does, and nothing here is interactive.
//
// A worded or counted badge draws two colours of one hue:
//
//	fill         a pale tint of the role's hue, relative to the level the
//	             badge stands on: the container chroma, at whatever depth
//	             separates it from that level ([Fill])
//	foreground   the role's pinned base while that base clears the text floor
//	             over that fill, and otherwise the step of the role's own
//	             ramp nearest the mid-value 500 that does ([Foreground])
//
// [Neutral] has no pinned base — the neutral ramp carries no pin — so its
// foreground takes the walk directly ([tokens.ColorTokens.MarkOn]), and its
// fill comes back as depth alone, the neutral ramp carrying no chroma to tint
// with. Which is why a Neutral badge measures its own floor rather than
// inheriting a caption token's, and why it is a badge rather than prose.
//
// A glyph badge has no container, so its foreground is derived against the
// level instead ([BareForeground]). Both derivations are one function over
// two surfaces — [ForegroundOver] — and that is deliberate: the three
// utterances read at one weight only if they are floored the same way over
// whatever each of them actually stands on.
//
// The floor is WCAG 1.4.3's 4.5:1 ([tokens.TextFloor]) for all three
// utterances. A badge that says its word as a sign instead is the same
// utterance at the same weight, so it is derived at the same floor;
// 1.4.11's 3:1 ([tokens.GraphicFloor]) governs a mark that must be resolved as
// a shape, and neither the fill nor the word on it is one. The fill's own seam
// against the level is gated at [tokens.ContainerFloor], the threshold for
// seeing a field at all.
//
// # Geometry
//
// Height is the type role's line box and nothing else:
//
//	height = style.LineHeight
//
// with no vertical padding, no minimum and no floor. The fill is drawn at that
// height and needs no padding of its own: the line box carries its own leading
// — 16 dp of box around a 12 sp face — so the fill already stands about 3 dp
// clear of the label's cap and descender, and adding to it would take the
// badge off its line.
//
// Horizontally the badge is its content between two S2 stops: the padding, the
// glyph's square, the S1 gap, the shaped label, the S1 gap, the close mark and
// the padding, each present only when it has something to draw. S2 rather than
// S1 because the gap inside the utterance and the gap to its edge must not be
// the same number, or the sign and the word stop reading as one thing in one
// box. A glyph badge, wearing no container, has no padding either: its whole
// box is the line box square.
//
// The corner is the radius scale's Base stop, clamped to half the height.
// Deliberately not Full: the pill is components/chip's shape, and a chip is
// the thing a badge must not be confused with — same rough size, same inline
// placement, opposite originator.
//
// The badge reports its label's baseline, so a row carrying a badge beside
// words in a larger role can be set on one line with layout.Baseline. A glyph
// badge reports none; a sign has no baseline to offer.
//
// The type role is the density's, one step less pronounced than the chip's:
//
//	density      role          size   line box   drawn height
//	-------      ----          ----   --------   ------------
//	Comfortable  LabelMedium   12 sp  16 dp      16
//	Compact      LabelSmall    11 sp  16 dp      16
//
// [Style] is that choice, stated once so a container reserving room for a
// badge asks the same question the badge answers. The two roles share a line
// box, so density moves the type's size and not the badge's height; that is a
// property of the type scale, and the badge reports what it draws either way.
//
// The glyph's square is the line box, the rule components/chip states for an
// inline mark: a mark on a line belongs to that line rather than to a control
// around it.
//
// # The close mark
//
// A badge with a non-nil [Props.OnDismiss] draws a small close mark after its
// label; one without draws none and registers no pointer area at all.
//
// The mark is half the line box — 8 dp on a 16 dp line — and the pointer
// target under it is [CloseHitDp] square, centred on the mark and free to
// overhang the badge on every side. The badge itself is unchanged by that
// target: it reports the text it drew, so a row of badges is laid out at the
// scale of the words in it and the slop overhangs the air around them.
//
// What answers the pointer is the fill's right cap: from the middle of the gap
// that separates the mark from the label out to the fill's own edge and
// corner, walked one step on hover and two on press
// ([tokens.ColorTokens.PinnedStateColor]) toward the ramp's 900 end, which is
// away from a light surface and away from a dark one alike. At rest it is the
// fill and cannot be told from it; under the pointer the badge grows a visibly
// deeper end. A region rather than the mark's own colour, because an 8 dp x
// changing colour is the smallest possible answer to a 24 dp target — the
// affordance was there and nothing showed where.
//
// The mark re-derives its foreground against the walked cap rather than
// holding the resting one, which is the same rule as everywhere else in this
// package: a colour is derived against what is actually under it. Holding it
// measured 4.5:1 at rest and 2.3:1 pressed, the state making the affordance
// harder to see the more the reader committed to it.
//
// A bare glyph badge has no cap to walk, so there the mark's own colour walks
// instead.
//
// 24 dp is WCAG 2.5.8 Target Size (Minimum), the criterion that governs at AA,
// and deliberately not the 44 dp of [tokens.MinHitTarget]: 44 is this system's
// floor for a standalone control with space around it, and a 44 dp target on a
// 16 dp badge would reach into whatever is set beside it.
//
// Shaper is not optional. Pass the theme's — tokens.Typography.Shaper() — or,
// in a golden test, its DeterministicShaper.
package badge

// Package badge provides the Vibrant Gio badge: the system's own word about a
// thing, set inline at the size of its type and coloured by the role it
// speaks in.
//
// [Render] is the pure path: resolved tokens plus a [RenderState] naming the
// storey the badge stands on, one frame out, no event handling.
// [RenderDismissible] is the same badge carrying its close mark, for a caller
// that owns the clickable. [Badge] is the live path — a theme observable and
// [Props] in, a widget out on every theme emission, with the close mark's
// pointer target and the dismissal dispatch the pure path cannot carry.
//
// # Not a control
//
// A badge is read, never used. It is off the control ladder entirely: its
// height is its own type's line box and [tokens.Density.ControlHeight] does
// not reach it, so a badge beside a control is a fraction of that control's
// height and is meant to be. It has no fill, no corner and no boundary, takes
// no pointer state on its body, and never grows to the width it is offered.
//
// The one thing that answers a pointer is the close mark, and what it removes
// is the label. A badge that switched something off would be a control wearing
// a badge's clothes; the affordance says "stop telling me this" and nothing
// else, and the caller owns what happens next.
//
// A control that a user operates is components/button or components/chip. A
// token the user themselves entered is a chip; a badge is applied ABOUT the
// thing by the system or the author, which is the whole of the difference and
// the reason the two look nothing alike.
//
// # Three utterances, one anatomy
//
// A badge speaks as
//
//	a word    "Popular", "Beta", "Deprecated"
//	a count   "9", "128" — a word made of digits, not a second component
//	a glyph   a check, a cross, a key — the sign that stands for the verdict
//
// and there is one anatomy underneath all three: the type's line box tall,
// sized to what it says, in the one ink its role resolves to. A count is a
// [Props.Label] of digits and needs no field of its own. A glyph is
// [Props.Glyph] with no label, drawn in the line box's own square. A glyph set
// beside a label leads it across the spacing scale's S1 stop — the sign comes
// before the word it stands for, which is also what keeps the badge from
// reading as a chip, whose mark trails its label.
//
// # Colour: the role's own hue, derived against the ground
//
// Five variants and they differ in hue alone: [Neutral] for a plain category
// label, [Success], [Warning], [Error] and [Info] for the four statuses. There
// is no emphasis axis and there will not be one — emphasis belongs where
// interaction does, and nothing here is interactive.
//
// Every colour is derived against the ground the badge stands on, because
// there is no fill for it to be derived against:
//
//	ink    the role's pinned base while that base clears the floor over the
//	       storey, and otherwise the rung of the role's own ramp nearest the
//	       mid-value 500 that does
//
// which is [tokens.ColorTokens.InkOn], one call, for the four hued variants.
// [Neutral] has no pinned base — the neutral ramp carries no pin — so it takes
// the walk directly ([tokens.ColorTokens.MarkOn]), which is why a Neutral
// badge measures its floor rather than inheriting a caption token's.
//
// The floor is WCAG 1.4.3's 4.5:1 ([tokens.TextFloor]) for all three
// utterances. A badge that says its word as a sign instead is the same
// utterance at the same weight, so it is derived at the same floor;
// 1.4.11's 3:1 ([tokens.GraphicFloor]) governs a boundary, and a badge draws
// none. Deriving a glyph badge at the lower floor would make the three
// utterances read at three weights, which is the one thing a single anatomy is
// for.
//
// The close mark rides that same ink, which clears GraphicFloor by
// construction — an affordance cannot be harder to see than the utterance it
// sits beside. Under the pointer it walks —
// [tokens.ColorTokens.PinnedStateColor], one rung on hover and two on press,
// at the ink's own hue and chroma — so the mark comes forward in both schemes,
// toward the ramp's 900 end and therefore away from a light ground and away
// from a dark one alike.
//
// # Geometry
//
// Height is the type role's line box and nothing else:
//
//	height = style.LineHeight
//
// with no vertical padding, no minimum and no floor. Horizontally the badge is
// its content: the glyph's square, the S1 gap, the shaped label, the S1 gap and
// the close mark, each present only when it has something to draw. There is no
// horizontal padding either — a badge is a run of text, and a run of text is
// spaced by whatever sets it.
//
// The type role is the density's, one rung quieter than the chip's:
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
// 24 dp is WCAG 2.5.8 Target Size (Minimum), the criterion that governs at AA,
// and deliberately not the 44 dp of [tokens.MinHitTarget]: 44 is this system's
// floor for a standalone control with space around it, and a 44 dp target on a
// 16 dp badge would reach into whatever is set beside it.
//
// Shaper is not optional. Pass the theme's — tokens.Typography.Shaper() — or,
// in a golden test, its DeterministicShaper.
package badge

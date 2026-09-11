// Package badge provides the Vibrant Gio badge: the system's own word about a
// thing, set inline at the size of its type and coloured in the platform's
// system colour for its status. The developer gives a badge one of the four
// statuses — [Error],
// [Success], [Warning], [Info] — or no status; a badge given no status is
// [Neutral].
//
// [Render] is the pure path: resolved tokens plus a [RenderState], one frame
// out, no event handling.
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
// It does wear a fill, and the fill is the platform's own answer for a badge:
// the status colour solid with its content knocked out in white, which is how
// the platform draws the count badge on an icon. What says it is not a
// control is not the colour but the size and the box — it is a fraction of a
// control's height, it draws no hairline, and it never grows to the width it
// is offered.
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
// A badge says its piece as
//
//	a word    "Popular", "Beta", "Deprecated"
//	a count   "9", "128" — a word made of digits, not a second component
//	a glyph   a check, a cross, a key — the sign that stands for the verdict
//
// and there is one structure underneath all three: the type's line box tall,
// sized to what it says, in the platform's colours for its status. A count is a
// [Props.Label] of digits and needs no field of its own. A glyph is
// [Props.Glyph] with no label, drawn in the line box's own square. A glyph set
// beside a label leads it across the spacing scale's S1 stop — the sign comes
// before the word it stands for, which is also what keeps the badge from
// reading as a chip, whose mark trails its label.
//
// The utterance picks the structure in one place, and it is the only branch in
// the component: anything with words in it wears the fill, and a sign on
// its own stands bare. See [Fill] for why — and for the obligation that
// carries, which is that a set of glyph badges must differ in shape, because
// a sign repeated under two statuses is two hues and nothing else.
//
// # The disc
//
// A glyph badge may be asked to stand on its status's fill instead of bare:
// [RenderState.Disc], or [Props.Disc] on the live path. The disc wears the
// system colour itself — systemGreen under a white check — and never a tint
// of it: it is the same [Fill] a word wears and the sign reads in the same
// [Foreground] over it, drawn as a circle inscribed in the glyph's own
// line-box square:
//
//	diameter = style.LineHeight
//
// which is the box a labelled badge's line already reserves at that density.
// So the disc costs the badge nothing — same reported size, same baseline of
// none — and a row that held a bare sign holds a disc without moving. The
// [Neutral] disc takes the badge's Neutral fill, systemGray, like every other
// Neutral fill in this package.
//
// The bare sign is the default and stays it. A verdict standing beside a
// field reads as a report on that field precisely because it has no box of
// its own; the disc is for where a sign has to hold its own against what is
// set around it, and it is asked for rather than assumed.
//
// A label ignores the disc. The fill a worded badge already wears IS the
// fill the disc would add, so there is still exactly one structure branch,
// and a labelled badge that also asked for a disc would be a badge inside a
// badge.
//
// The sign itself is handed the square inscribed in that circle, centred on
// it, rather than the line box the bare sign gets:
//
//	sign box = round down to the diameter's parity of (diameter / √2)
//
// which is the largest box whose every point is inside the circle. A [Glyph]
// is a painter this package cannot inspect and the contract it is written to
// is that a sign spans most of the box it is handed, so handing one the disc's
// full square puts a check's tip on the antialiased rim and a sign drawn
// corner to corner outside it altogether. The inscribed square is the only
// size that holds for a painter the badge has not seen. Rounding to the
// diameter's own parity is what keeps the sign centred on whole pixels.
//
// A caller passes the same [Glyph] either way: the sign is smaller inside a
// disc than standing bare, and the badge's box is the same size in both.
//
// # Colour: the platform's system colours
//
// Five values and they differ in hue alone: [Neutral] for a plain category
// label carrying no status, [Success], [Warning], [Error] and [Info] for the
// four statuses. There is no emphasis axis and there will not be one —
// emphasis belongs where interaction does, and nothing here is interactive.
//
// Every colour is one of the platform's own names:
//
//	status     fill and bare sign
//	------     ------------------
//	Success    systemGreen
//	Warning    systemOrange
//	Error      systemRed
//	Info       systemBlue
//	Neutral    systemGray filled, secondaryLabelColor bare
//
// A worded or counted badge is that colour filled, with its content in
// alternateSelectedControlTextColor — white in both appearances, the
// foreground the platform pairs with a fill it paints in a system colour
// ([Fill], [Foreground]). A glyph badge standing bare draws its sign in the
// system colour itself ([BareForeground]); [Neutral] has no system colour of
// its own, so bare it reads in the platform's secondary label, the strength
// the platform gives a word that is not the subject.
//
// Nothing is derived and nothing is measured against what the badge stands
// on. A system colour is the platform's one answer for a hue that must read
// on every fill a window carries, and the white on it is the platform's one
// answer for what stands on such a fill.
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
// box. A glyph badge, wearing no fill, has no padding either: its whole
// box is the line box square.
//
// The corner is the radius scale's Base stop, clamped to half the height.
// Deliberately not Full: the pill is components/chip's shape, and a chip is
// the thing a badge must not be confused with — same rough size, same inline
// placement, opposite originator. The platform draws its own count badge as
// a full capsule; here the silhouette is doing work the platform's badge does
// not have to do, since nothing sits beside a Dock icon's badge that could be
// mistaken for it.
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
// [Style] is that choice, stated once so a caller reserving room for a
// badge asks the same question the badge answers. The two roles share a line
// box, so density moves the type's size and not the badge's height; that is a
// property of the type scale, and the badge reports what it draws either way.
//
// The glyph's square is the line box, the rule components/chip states for an
// inline mark: a mark on a line belongs to that line rather than to a control
// around it. The disc is that same square's inscribed circle, so a glyph badge
// measures the line box square whether it stands bare or on a disc.
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
// What answers the pointer is a region and not the 8 dp x inside it: on a
// badge that wears a fill, that fill's trailing cap — from the middle of the
// gap that separates the mark from the label out to the fill's own edge and
// corner; on a bare badge, the mark's own square. Under the pointer the
// region takes the platform's hover overlay and held it takes the press
// overlay, each a coverage of black in the light appearance and white in the
// dark one, laid over whatever is beneath. A region rather than the mark's
// own colour, because an 8 dp x changing colour is the smallest possible
// answer to a 24 dp target — the affordance was there and nothing showed
// where.
//
// The mark itself is [Foreground] on a filled badge and the platform's
// secondary label on a bare one, and it does not move with the pointer: an
// overlay composites, so the white on a filled cap and the grey on a bare
// square both keep reading through it.
//
// 24 dp is WCAG 2.5.8 Target Size (Minimum), the criterion that governs at AA,
// and deliberately not the 44 dp of [tokens.MinHitTarget]: 44 is this system's
// floor for a standalone control with space around it, and a 44 dp target on a
// 16 dp badge would reach into whatever is set beside it.
//
// Shaper is not optional. Pass the theme's — tokens.Typography.Shaper() — or,
// in a golden test, its DeterministicShaper.
package badge

// Package chip provides the Vibrant Gio chip: a small pill that carries a
// data-bearing summary — a label and, optionally, one glyph — filled a
// measured step above the ground it rests on.
//
// It has two faces and one geometry. [Render] draws the CHIP, which is
// clickable: it takes a [RenderState] carrying hover, press, focus and the
// storey it stands on, walks its fill under the pointer and asks for the
// pointer cursor. [RenderBadge] draws the BADGE, the same face held still —
// for a mark that keeps a fill but takes no input, with no state walk, no
// focus ring and no cursor. Both are the pure path: resolved tokens plus an
// explicit state in, one frame out, no event handling.
//
// [Chip] is the live face of the clickable one: a theme observable and [Props]
// in, a widget out on every theme emission, with the pointer area, the
// keyboard and the activation dispatch the pure path cannot carry. The badge
// has no live twin, because a face that takes no input has no state to keep —
// [RenderBadge] and a ground is the whole of it.
//
// # The anchor face has moved
//
// The pop-up anchor — the same geometry at the button's rounded-rect corner
// with the paired up/down chevrons — is components/picker's, where it is the
// chrome register's trigger over the menu the form register's trigger shares.
// A pop-up control is not a summary that happens to be clickable; it names a
// choice and stands under a list of the alternatives, and that is a picker.
// [RenderAnchor] forwards to picker.RenderAnchor and is deprecated; the live
// path is picker.Anchor. The geometry both draw from is one geometry, so
// nothing about the drawn control changed with the address.
//
// # What the chip is not
//
// There is no emphasis axis. A chip is not a quiet button and a button is not
// a loud chip: selection rides components/button's emphasis register (Tonal
// when picked, Ghost when not), and a button's label stays a verb. The chip is
// where a data-bearing summary with a glyph lives — the model a chat is using,
// the branch a pane is on, the filter a list is under — and it has exactly one
// weight because that summary has exactly one job.
//
// Nor is it patterns/tag. That pill is a status vocabulary drawn in a role's
// own hue — Success, Warning, Error — and it does not move under the pointer.
// This one is neutral by construction, wears the elevation ladder rather than
// a ramp, and is the only one of the two that can be clicked.
//
// # Colour: everything is walked, nothing is mixed
//
// Five colours, five derivations, no literal in the package:
//
//	fill   the ground lifted by the scheme's measured step, walked by the
//	       pointer
//	rim    the neutral rung that clears GraphicFloor against the ground AND
//	       against the fill — or none, when no rung reaches both
//	label  the Text pin, or the neutral rung that clears TextFloor on the fill
//	glyph  the Text pin, or the neutral rung that clears GraphicFloor on the fill
//	ring   focus.Ring against the fill the band lies on
//
// The fill is relative to the ground the caller names and never an absolute
// step, which is what ADR-022 prescribes for anything raised: a chip on the
// paper, the same chip inside a dialog and one on the furniture floor each
// stand the same distance over what they lie on. Hover and press are that
// fill's own state walk — tokens.ColorTokens.PinnedStateColor, the walk
// tokens.ColorTokens.StateAt takes from a storey, taken from the chip's
// resting fill instead — which still heads toward the ramp's 900 end, so a
// chip DARKENS under the pointer on paper and lightens on slate. The two
// directions are not a mirror: the ladder answers to the linchpin, feedback
// does not.
//
// # The fill's two steps
//
// The step itself is a MEASUREMENT, in CIELAB L\*, one number per scheme —
// documented here the way theme/tokens documents the floor's own two
// measurements, and read the same way: off the direction of the neutral
// surface band, never off a mode flag. A scheme whose band climbs away from
// its 100 stop has its pin as the darkest surface the ramp carries and takes
// the first number; one whose band descends takes the second.
//
// THE DARK STEP IS 1.28 L\*, AND IT IS MEASURED. Through the ladder's first
// year the chip filled at ground.Raised() — a whole storey. That was right as
// depth and wrong as loudness: on the dark scheme's paper it put the pill at
// #222222 over #181818, +10.0 luminance, where the platform's own toolbar
// pop-up capsules — a chevron beside a label, the chip's exact role — stand
// +2.65 over the band they are drawn on. Four times the platform's step, in
// the one role the platform draws as a near-hairline outline rather than a
// filled block. The reading, from the stored macOS reference
// (reference/macos/mail-window.png in the org's .github repository, indexed by
// ADR-019; window-bounded capture, macOS 26.5.2, dark appearance):
//
//	Mail's unified toolbar band          #232A2E   L* 16.555   luminance 40.80
//	its pop-up capsules on that band     #242D32   L* 17.837   luminance 43.45
//	                                               step 1.28   step +2.65
//
// so 1.28 L\* is the platform's step in the ladder's own unit, and on the
// default dark palette it realizes #1B1B1B over the #181818 paper — +3.0
// luminance, the platform's whisper rather than the block.
//
// THE LIGHT STEP IS 0.70 L\*, AND IT IS DERIVED — the gap is real and is
// recorded here rather than papered over. ADR-019's whole sweep was taken in
// the DARK appearance; there is no light-appearance capture in
// reference/macos to measure a light toolbar capsule off, so this half cannot
// be obtained the way the dark half was. What it takes instead is the number
// the light scheme already spends on its first storey over the paper: 0.70
// L\*, Level1 over Level0 on the default light palette. The reason it is not
// simply the dark measurement is the light scheme's headroom. That scheme has
// 3.12 L\* in total between its #F6F6F6 paper and the tonal axis, and the
// ladder spends all three of its storeys inside it; a chip taking the
// platform's 1.28 L\* there would spend 41% of the whole ladder on one pill
// and stand above where a dialog sits.
//
// The two schemes differing is the platform's own precedent, not an
// inconsistency. theme/tokens' floor takes 4.89 L\* down in the light scheme
// and 1.48 L\* down in the dark one, because a light window separates its
// furniture with a step the ramp happens to carry and a dark window with a
// whisper. The same asymmetry governs here, read off the same band, and each
// half says plainly which of the two it is: the dark number is a
// measurement, the light number is a derivation awaiting one. A light-
// appearance capture of a macOS toolbar's pop-up capsules would close it, and
// should be added to ADR-019 rather than measured privately.
//
// The rim is why the chip is legible at all in the light scheme. There the
// ladder has almost no headroom above its paper, so the storey step is a
// fraction of an L* — a chip filled one storey over the light paper measures
// 1.02:1 against it, which is a pill with no pill in it. The fill is correct
// and it is not the thing that can carry the edge; the rim is.
//
// An edge has two sides and one colour, though, and the chip's inner side
// moves: the fill walks one and two rungs under the pointer, and in the dark
// scheme those rungs are long. So the rim is derived against both neighbours
// rather than against one of them — components/input can aim a field's bezel
// at the ground alone because a field's interior never moves, and a chip's
// does. Aimed at the ground alone the rim landed exactly ON the pressed fill
// at level 1 in the dark scheme, the same colour twice; aimed at the fill
// alone it would vanish into the light scheme's paper at rest. Both candidate
// rungs are derived and the one that clears the floor on both sides is kept.
//
// When no rung reaches both, the chip draws NO rim, and that is not a
// concession. The two neighbours are then further apart than twice the floor,
// which means the fill is separating from its ground on its own — the
// condition patterns/tag states for its own pill, translated into the
// elevation ladder's vocabulary: a fill that stands off its page needs no
// outline, and a fill that cannot never will. It happens in the dark scheme
// only, on a pressed chip at level 2 and above and on a hovered one at level
// 3, where the walk carries the fill most of the way to the ramp's light end;
// the chip reads there as a solid block under the finger. In the light
// scheme, on every storey and in every state, the rim is always drawn. The
// quieter fill above moved that boundary rather than removing it: at level 1
// the rim now clears both sides in every state, where a pressed chip used to
// lose it.
//
// The label and the glyph are inks over the fill, and they are derived the way
// [tokens.ColorTokens.InkOn] derives one — the pinned base while it clears the
// floor against the ground it is drawn on, and the walked rung otherwise. InkOn
// itself refuses RoleNeutral, which has no pin; a neutral ink's pin is the Text
// pin, so that is the base this package hands the same rule. The label owes
// WCAG 1.4.3's 4.5:1 ([tokens.TextFloor]) because it is words; the glyph owes
// 1.4.11's 3:1 ([tokens.GraphicFloor]) because it is a mark. They are resolved
// against the fill actually drawn, state included, so a chip whose fill walks
// far enough to threaten its own label re-derives it rather than keeping a
// colour that no longer reads.
//
// The focus ring is components/internal/focus's, measured against the chip's
// own storey — its fill, in the state it is drawn in. That is the ground
// focus.Ring asks for and the one components/button hands it for a filled
// register. Measured against the storey UNDER the chip instead, the ring came
// out at 1.01:1 on a pressed chip in the dark scheme, where the fill has walked
// and the storey has not.
//
// A focused chip's edge IS the ring: it takes the rim's place, two dp where the
// rim was one, rather than being drawn inside it. Drawn inside, the two made a
// three-line sandwich — hairline, one pixel of fill, then the ring — which a
// reviewer handed the rendering called a dirty grey halo around a purple
// outline before naming anything else. It is the same reason components/button
// holds its ring clear of its own boundary: a band beside a boundary is read as
// part of that boundary. A button has no rim to collide with and can put air
// between the two; a chip's rim is a drawn line, so the chip spends it instead.
// The pill measures the same box focused as at rest and the label does not
// shift — what changes is which colour draws the edge.
//
// # Geometry: the density table, not a new set of numbers
//
// Height is the rule theme/tokens' density header states for every control in
// the system — a control height is a floor, not a height:
//
//	height = max(d.ControlHeight, labelStyle.LineHeight + 2×d.PaddingY)
//
// which for the two roles a chip is set in comes out as:
//
//	density      role          line box   + 2×PaddingY   ControlHeight   drawn
//	-------      ----          --------   ------------   -------------   -----
//	Comfortable  LabelLarge    20         36             36              36
//	Compact      LabelLarge    20         32             28              32
//	Comfortable  LabelMedium   16         32             36              36
//	Compact      LabelMedium   16         28             28              28
//
// The last row is the chip this component replaces: mindchat's model picker
// was a hand-rolled 28 dp pill set in LabelMedium with 12 dp of side padding,
// which is exactly Compact × LabelMedium out of the table above, down to the
// padding. Nothing had to be invented to reproduce it; the numbers were
// already in the tokens.
//
// Horizontal padding is d.PaddingX (16 dp Comfortable, 12 dp Compact) at each
// end. The gap between the label and the glyph is the spacing scale's S2 stop,
// the same 8 dp patterns/tag gives its close mark and for the same reason: at
// S1 a trailing mark reads as the end of the word rather than as something of
// its own.
//
// The glyph is the label's own line box — 20 dp in LabelLarge, 16 dp in
// LabelMedium — because an inline mark belongs to the line it sits on rather
// than to the control around it. That rule agrees with components/icon's
// [icon.Size] exactly on the pairing each density was derived for:
// icon.Size(Comfortable) is 36 − 2×8 = 20, LabelLarge's line box; and
// icon.Size(Compact) is 28 − 2×6 = 16, LabelMedium's. The two rules meet where
// they should and the chip needs no third number.
//
// The corner radius is the scale's Full stop, clamped to half the pill's
// shorter side by components/layout's Pill — a chip is a pill, which is what
// separates it at a glance from a button's 6 dp Md corner. The rim is one dp
// at every density, the width every other derived edge in this library draws,
// and it is painted as nested fills rather than as a stroke on the pill's
// path: a stroke is centred on the path, so half of it would fall outside the
// box the chip reports and every pixel of it would be a blend of the two
// colours rather than either.
//
// A chip is sized to its content and does not fill the width it is given —
// that is the other half of what makes it not a button. It clamps to the
// constraints it is handed, so a chip in a box narrower than its label is
// truncated by the box rather than overflowing it.
//
// # Pinning the pill to an edge
//
// A box wider than the pill therefore has slack in it, and by default the
// pill sits at the box's leading edge with the slack behind it and the widget
// reports the pill alone. [Props.Pin] moves the pill to a named edge of that
// box and reports the box — [PinLeading] or [PinTrailing], the horizontal axis
// only. It is a placement, not a stretch: the drawn pill is the same pill at
// the same width, and a chip that pins nothing is unchanged.
//
// Say it only where the box is a cap the caller sized. A chip laid out Rigid
// in a flex is offered the whole row, and a pin would have it report the whole
// row; the container that reserved a cap for it is the one with something to
// pin to.
//
// The seam is for the case where the container cannot do the placing itself.
// A chip its container lays out needs nothing here — the container knows both
// boxes and offsets the widget. A chip handed to a pattern does: patterns/
// popover measures its anchor and centres it in the canvas it was given, so
// the reserved cap and the pill part company by half the slack, and the only
// place the two widths are both known is inside the chip. A model picker in
// a window's chrome row is that case — a cap at the trailing edge of the row,
// whose pill has to land on the content column's own edge rather than a few
// pixels inboard of it.
//
// # The pointer target
//
// The drawn pill is the density's control height and a standalone control owes
// its pointer 44 dp on each axis ([tokens.MinHitTarget], WCAG 2.5.5). The pure
// path draws and does not register pointer areas, so the extension belongs to
// the live path, exactly as it does for components/button: [Render] reports the
// pill's own size and the caller that wires input extends it. [Chip] is that
// caller — it spends the difference as slop centred on the pill, so the row
// the chip stands in is laid out at the pill's size and the target overhangs
// the air around it.
//
// Shaper is not optional. Pass the theme's — tokens.Typography.Shaper() — or,
// in a golden test, its DeterministicShaper.
package chip

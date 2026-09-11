// Package chip provides the Vibrant Gio chip: the small control that content
// sprouts, drawn as the platform draws an ordinary small control.
//
// A chip is defined by its purpose and by nothing else. [Assist] offers a
// contextual action, [Filter] narrows a set and is the one that toggles,
// [Input] is a token the reader entered and carries a dismiss mark, and
// [Suggestion] is a generated prompt. The four are the whole variation: one
// structure, one silhouette, one height, four purposes.
//
// [Render] is the pure path: resolved tokens plus a [RenderState] carrying the
// selection and the pointer, one frame out, no event handling.
// [Chip] is the live one — a theme observable and [Props] in, a layout.Widget out on
// every theme emission, with the pointer areas, the keyboard, the filter's own
// selection and the activation dispatch the pure path cannot carry.
//
// # The structure
//
//	[mark] label [dismiss]
//
// Both brackets are optional and both are decided by the purpose rather than by
// a flag. The leading slot holds the caller's [Props.Icon] in the label's cap
// band ([MarkDp]); on an [Input] chip it is the avatar slot instead,
// [AvatarDp] behind a full-round corner, because what leads a token the reader
// entered is a picture of the thing rather than a sign for it. A selected
// [Filter] chip draws the checkmark there — the mark that says it is selected
// takes the slot rather than standing beside a second one. Only [Input]
// carries the trailing dismiss mark, and it always carries it.
//
// The marks and the words are ONE LINE OF TEXT. A mark stands in the band the
// label's capitals stand in — baseline to cap height, read off the face the
// label is set in — and is stroked at the width of the label's own stem, so
// nothing inside a chip is heavier or taller than the words it is beside. See
// [MarkDp] and [MarkStrokeDp] for the measurement that fixes both.
//
// The parts are set an S2 stop apart, the spacing scale's own answer for two
// things that belong to one utterance without being one word, and the row is
// laid from the leading padding rather than centred: the structure is read from
// its leading edge, so a chip clamped narrower than its content loses its
// trailing end and not both ends at once. The one exception to that padding is
// the avatar, which is set in from the leading edge by the same clearance it
// has above and below itself: a round picture sits in a square well, not at
// the far end of a text inset.
//
// # What the chip is not
//
// Not a less pronounced button. A button is a fixture — placed by the
// developer, always there, always offering the same action; a chip appears out of content and
// context. Marking a choice is the [Filter] chip's job and no button's,
// whatever the button's emphasis.
//
// Not components/badge. A badge is the system's word ABOUT a thing and is read
// rather than used; it is off the control family, sized to its type, and does
// not move under the pointer. The line between the two families is read/use.
//
// Not the picker's toolbar trigger. A control that names a choice and stands
// over a list of the alternatives is components/picker's, which draws the
// platform's own measured pop-up capsule.
//
// # Colour
//
// Every colour is one of the platform's own names and nothing is measured or
// mixed. [Resolve] carries the table: a resting chip is the platform's
// Control behind a Separator rim with ControlText words, a selected [Filter]
// chip is SelectedContentBackground with AlternateSelectedControlText and no
// rim, and a held chip wears PressOverlay over whichever body it started
// from. The trailing dismiss mark is SecondaryLabel, which is the one part of
// a chip the platform draws weaker than the words beside it.
//
// The four purposes are one colour. What separates them is behaviour and
// structure — which one selects, which carries the trailing mark, what stands
// in the leading slot — and the platform draws all four alike.
//
// The pointer moves no colour at rest. A push-button-shaped control does not
// change colour under the pointer on macOS 26, measured against the stored
// captures, so hover is the cursor and nothing else; press is the only pointer
// state that paints. The overlay is laid over the body as a second fill rather than
// blended into it, because the body itself is translucent in the dark scheme
// and a coverage folded into a translucent fill is not the pixel the platform
// draws.
//
// The focus ring is components/internal/focus's, the platform's keyboard
// focus indicator, one colour per appearance on every surface. A focused
// chip's edge IS the ring — it takes the rim's place, two dp where the rim
// was one, rather than being drawn inside it. Drawn inside, the two make a
// three-line sandwich that reads as a smeared halo; it is the same reason
// components/button holds its ring clear of its own boundary. The chip
// measures the same box focused as at rest and the label does not shift.
//
// # Geometry
//
// The height is [tokens.Density.ChipHeight] — the density's control height
// less the system's chip drop, 20 dp Comfortable and 15 dp Compact — under the
// label's own line box, which the chip takes wherever that box is taller. It
// is not a floor over the padding rule, which is the rule for controls in the
// control family the chip has just left: a chip is smaller than a button by
// construction and spends no padding on the axis. At Compact the line box is
// taller than the chip height in both label roles, so a Compact chip measures
// its label; a box shorter than the words in it is not a chip either.
//
// Horizontal padding is d.PaddingX at each end. The silhouette is the radius
// scale's Lg stop, 8 dp, clamped to half the height — a rounded rectangle, not
// a capsule. The edge is one dp at every density, the width every other
// edge in this library draws, and it is painted as nested fills rather
// than as a stroke on the shape's path: a stroke is centred on the path, so
// half of it would fall outside the box the chip reports and every pixel of it
// would be a blend of the two colours rather than either.
//
// A chip is sized to its content and does not fill the width it is given. It
// clamps to the constraints it is handed, so a chip in a box narrower than its
// label is truncated by the box rather than overflowing it.
//
// # Pinning the chip to an edge
//
// A box wider than the chip therefore has slack in it, and by default the chip
// sits at the box's leading edge with the slack behind it and the component
// reports the chip alone. [Props.Pin] moves it to a named edge of that box and
// reports the box — [PinLeading] or [PinTrailing], the horizontal axis only.
// It is a placement, not a stretch: the drawn chip is the same chip at the
// same width, and a chip that pins nothing is unchanged.
//
// Say it only where the box is a cap the caller sized. A chip laid out Rigid
// in a flex is offered the whole row, and a pin would have it report the whole
// row; the container that reserved a cap for it is the one with something to
// pin to. The seam is for the case where the container cannot do the placing
// itself — a pattern that measures what it is given and centres it in a box of
// its own has the two widths part company by half the slack, and the only place both
// are known is inside the chip.
//
// # The pointer targets
//
// The drawn chip is the density's chip height and a standalone control owes
// its pointer 44 dp on each axis ([tokens.MinHitTarget], WCAG 2.5.5). The pure
// path draws and does not register pointer areas, so the extension belongs to
// the live path, exactly as it does for components/button: [Render] reports
// the chip's own size and the caller that wires input extends it. [Chip] is
// that caller — it spends the difference as slop centred on the chip, so the
// row the chip stands in is laid out at the chip's size and the target
// overhangs the air around it.
//
// The dismiss mark registers a second target of its own, [DismissHitDp] — WCAG
// 2.5.8's AA minimum rather than 44 dp, because a 44 dp target centred on the
// mark would reach past both ends of the chip carrying it. It lies over the
// body's target and takes the pointer where they overlap, so the body reads
// the mark's hover as its own: a chip whose mark is under the finger is a chip
// under the finger.
//
// Shaper is not optional in the pure path. Pass the theme's —
// tokens.Typography.Shaper() — or, in a golden test, its DeterministicShaper.
package chip

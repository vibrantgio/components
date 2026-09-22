// Package icons is the design system's own icon set: marks drawn on one grid
// at one weight, addressed by a name that says what the control does, and
// resolved to the host operating system's drawing at run time.
//
// A name yields a painter with the signature the library's controls take for
// an icon slot — func(gtx, sizePx, col) — so it drops straight into one:
//
//	mark := icons.Mark(icons.Sidebar)
//	if mark != nil {
//		mark(gtx, gtx.Dp(20), fg)
//	}
//
// The painter draws into a square of sizePx at the current origin, the drawing
// centred and its aspect preserved. Mark returns nil for a name the set does
// not carry, which is also what a control's icon slot reads as "no icon", so a
// missing mark degrades rather than panics; Has answers the same question
// without building a painter.
//
// # Names
//
// A name says what the mark depicts, never what the control using it does:
// sidebar, disclosure, history-back — not panel-with-lines, triangle,
// left-chevron. One picture therefore carries one name however many actions
// it serves: the folder a row IS and the control that chooses which folder a
// window shows are both drawn with a closed folder, so both ask for folder,
// and what the control does is its label's to say. The drawing behind a name
// is free to change, and does change between platforms; the name is the part
// call sites store, so it has to survive that. Names are lowercase ASCII with
// words joined by "-", the qualifier last (history-back, history-forward),
// and they are stable once published.
//
// # One name, one drawing per platform
//
// Marks live in this package's marks directory as SVG files, and the file name
// is the whole registration mechanism:
//
//	marks/<name>.svg          the drawing that serves every platform
//	marks/<name>.<goos>.svg   the drawing for one operating system
//
// where <goos> is a value of runtime.GOOS. Adding a mark means adding a file.
// A lookup asks for the platform's own drawing first and falls back to the
// plain name when that platform has none — so a platform whose idiom differs
// gets its own picture, and every other platform gets the fallback rather than
// somebody else's idiom by accident. Every mark must carry the fallback file;
// a name that resolves anywhere resolves everywhere, and the package's tests
// enforce it.
//
// Nothing here is selected by build tag. The whole set is compiled into every
// binary and the choice happens at run time, so New can be handed any GOOS
// string — which is how a test asserts what another platform would see, and
// how a preview shows the set as another platform draws it. Resolve reports
// which file answered.
//
// The set registers into components/icon's Registry rather than inventing a
// second lookup: the keys are the name for the fallback and name@goos for a
// platform's drawing. Register fills a registry an application already owns
// with its own freshly parsed copies of every mark.
//
// # The grid
//
// Every mark is drawn on a 24×24 grid — viewBox "0 0 24 24" — and what a form
// is drawn TO depends on what shape that form is:
//
//	a square form        the keyline, 2.5 units in from every edge, 19×19
//	a round or curved    the allowance, 13 units across
//	a diagonal form      20 units, 2 in from every edge
//
// The first two are MEASURED off the organization's macOS reference. The
// third is not, and this package says so wherever it is spent.
//
// THE KEYLINE IS MEASURED at 1x off the toolbar bands, where a symbol
// standing alone in a band is the platform drawing this set's own subject at
// its own size. Finder's tag covers 19 × 19 px (finder-window-light.png,
// x 873-891, y 17-35) and its group pull-down 18 × 18 (x 769-786, y 17-34);
// Voice Memos' sidebar toggle covers 19.31 × 15.12 (voicememos-window.png,
// x 106-125, y 18-33). All three stand in controls 36 px tall. 19 is the
// reading the set is drawn to, and 24 dp is where it comes out: one unit to
// one device pixel, so a square mark drawn to the keyline covers the
// platform's own 19 px there.
//
// THE ROUND AND CURVED ALLOWANCE IS MEASURED the same way, off the one round
// form the reference holds standing alone in a band: the magnifier's lens in
// finder-window-light.png (x 965-980, y 18-34). Its outer diameter is
// 13.06 px — the leading outer edge at x 965.02 and the trailing one at
// 978.08 on the lens's own centre row, the top at y 18.85 and the foot at
// 31.90 down its own centre column — against the tag's 19 px square in the
// same band, so the lens is 0.687 of the keyline. The two field glyphs agree
// at their own size: Mail's and Voice Memos' search fields each draw a lens
// 10.29 px across outside in a glyph 12.33 px wide, which against the toolbar
// symbol's own 15.75 px width is 13.14 units. 13 is the reading, and the set
// draws its round forms to it — the search mark's lens and the refresh mark's
// ring. It replaces an unmeasured 20 the set carried before, which stood
// seven units over the platform's own lens.
//
// THE DIAGONAL ALLOWANCE IS NOT MEASURED. No stored capture holds a bare
// diagonal band standing alone in a toolbar band; Notes' compose covers
// 17 × 17, the nearest reading, and it is a figure rather than a band. The
// allowance stays 20 units, the check mark is all that takes it, and the
// capture that would settle it is on the reference's list.
//
// The one stored capture of a CLEAR control does not settle it, and what it
// holds settles the clear mark instead. MEASURED at 1x off
// voicememos-multi-folder-search-2026-09-18.png, a least-squares fit of the
// figure to the capture's own coverage over the mark's whole block (rms 0.007
// of a pixel per pixel): the platform draws its clear control at x 996-1009
// as a FILLED DISC 13.04 px across with the field's own fill showing through
// a cross knocked out of it, the cross's arms 1.18 px across perpendicular
// and their centre lines 7.00 px tip to tip at 45 degrees. That is a hole in
// a solid figure and not a band standing on its own, so it gives the diagonal
// allowance nothing — but the DISC is a round form at 13.04, which is the
// round and curved allowance above read a second time, off a second
// application, at a second place. The set draws that control: clear.svg is
// the disc and the knockout, and the bare cross it carried before is gone.
//
// # What a form is measured against, per place
//
// 19 IS THE KEYLINE OF A SYMBOL STANDING ON ITS OWN — the tag in a toolbar
// band, the marks a sidebar row is drawn with. A mark drawn INSIDE a control
// is that control's own measurement instead, and the set draws each at what
// its own capture reads: the pull-down's glyph at 18, the pop-up's chevron
// pair at 8 by 11, its single chevron at 8 by 5, the history chevron at 8 by
// 14. The density heights work the same way — the platform draws a control in
// a toolbar band taller than one in a dialog, and neither reading corrects the
// other — and a mark is no different. What a form is drawn to is the place it
// stands in, and the keyline is one of those places rather than the answer for
// all of them.
//
// 24 is chosen for the sizes the library actually draws icons at: 16, 20 and
// 24 dp — the control's content box at each density, plus the top of the
// range. It maps one unit to one device pixel at the largest of those, and it
// is the grid the platforms publish their own drawings on, so a mark can be
// traced against a reference without rescaling arithmetic in between.
//
// No grid is pixel-exact at all three sizes — 16, 20 and 24 share only the
// factor 4 — so the honest statement is which lines land where. At 1 device
// pixel per dp, a coordinate falls on a whole pixel at 24 dp on every whole
// unit, at 16 dp on every multiple of 1.5, and at 20 dp only on multiples of
// 1.2.
//
// WHAT THE MEASURED BAND COSTS THAT. The band is a DEVICE width (below), so
// it covers a whole device pixel at every one of the three sizes and what
// decides a landing is where its leading edge stands and nothing else: the
// band holds one when it starts on a pixel boundary or no more than its own
// width less one below the next, which is 0.40 px at 24 dp, 0.40 at 20 and
// 0.32 at 16. The band the set drew until CG5.11 was 1.4 units at every size,
// which is 0.93 px at 16 dp, and it held a whole pixel nowhere at that size at
// all. A band's placement still stands on the WHOLE unit wherever the
// measurement leaves it free — the plus's two bars run 10.6 to 12, the
// folder's flap 8.6 to 10, the document's fold upright 11 to 12.4 — and every
// file records what its measured placements reach where they land nothing.
//
// THE KEYLINE LANDS A PIXEL AT ONE SIZE OF THE THREE, and that is the price
// of the two measurements together. 19 units is odd against a 24-unit box, so
// a centred square form stands at 2.5 and 21.5, and a band from either edge
// holds a whole device pixel at 20 dp — where the widening carries its
// leading edge down onto 1.967 — and at neither of the other two, reaching
// 0.793 of its best pixel at 16 dp and 0.900 at 24. The 18-unit keyline and
// the 1.5 band this set carried before landed whole pixels on both counts,
// and they drew a square symbol a unit under the platform's largest reading
// at a weight a sixth over its own. Measured beats published, and what is
// lost is recorded here and in every file the keyline and the band reach.
//
// # The band
//
// ONE MEASURED WIDTH IN DEVICE PIXELS, and every mark in the set draws it,
// spent perpendicular to whatever the band runs along — the axis, a diagonal
// or a curve alike. It is a device width and not a share of the mark: the
// platform holds one weight across symbols four times each other's size, so a
// mark drawn smaller takes the same band and not a thinner one.
//
//	below 20 dp   1.32 px   the search field's magnifier, a symbol 12.33 px
//	                        across, against the 12.67 a square mark covers
//	                        at 16 dp
//	from 20 dp    1.40 px   Mail's compose, a symbol 16 px across, against
//	                        the 15.83 a square mark covers at 20 dp
//	at 24 dp      1.4 units the same 1.40 px, where one unit is one device
//	                        pixel and the grid's own number and the device
//	                        width are one figure
//
// MEASURED at 1x off the organization's macOS reference, each symbol read
// against its own drawn plateau rather than against a colour name. The
// magnifier standing in a search field draws a ring 1.32 px thick in both the
// captures that hold one — 1.319 and 1.311 across Mail's own lens
// (mail-window.png, x 878-890) and 1.327 and 1.318 across Voice Memos'
// (voicememos-window.png, x 657-669), read on each lens's own equator, with
// the same figure down its centre column. Mail's compose measures 1.400 and
// 1.395 (x 415-430); Voice Memos' sidebar rectangle 1.391, 1.391, 1.392 and
// 1.402 over its four walls (x 106-125); Finder's view-pop-up list bar 1.354
// over all three of its bars (x 704-720); the pop-up's chevron pair 1.36 to
// 1.44 and its magnifier's circle 1.48.
//
// THE PLATFORM'S BAND IS NEAR FIXED ACROSS SIZES, which is the reading this
// rule stands on. From an 8 by 5 chevron to a 19 px square the band runs 1.32
// to 1.54 px — the narrowest reading 0.86 of the widest — over a range of
// symbol sizes a band drawn as a share of the mark would have spanned by a
// factor of four. So the set draws the device width each size's own capture
// gives. It is STEPPED and not run as a line between the captured sizes: a
// square mark covers 12.67 px at 16 dp against the field magnifier's own
// 12.33 and 15.83 at 20 dp against the compose symbol's own 16, so each of
// the set's sizes has a captured symbol standing at its own size and nothing
// has to be interpolated. Nothing is captured between 12.33 px and 16, and a
// mark drawn at a size no capture holds is on the reference's list.
//
// The widening is spent as an OFFSET of half its own width on every edge the
// drawing lays down, so a band keeps the centreline the grid puts it on and
// grows symmetrically about it — what a heavier stroke in the same box is.
// Two consequences are recorded rather than hidden: a square form's covered
// extent grows with the band, from the keyline's 19 units to 19 units and the
// widening (13.05 px at 16 dp, 16.07 at 20, and the measured 19.00 at 24, the
// last two the platform's own 16 and 19 to a sixteenth of a pixel); and the
// air between two bands narrows by the same amount, which the folder's flap
// records below.
//
// The set drew three weights before this and the marks came out uneven — 1.5
// on the axis, 2 on a 45-degree band as optical compensation, and a rendered
// spread of 1.15 to 2.13 px across the set at one size, where the platform
// holds one weight. One measured number is what closes it, and what a mark's
// own capture reads where it differs is stated in that mark's file: the
// history chevron's 1.53 perpendicular, the folder's 1.37 to 1.50, the
// document's 13-wide page against a 19-unit keyline, and the folder's flap
// gap.
//
// THE FOLDER'S FLAP GAP is the fourth of those, and it is the air between two
// bands rather than a band. Its capture holds the flap 1.24 px clear of the
// body's inner top face over a mean band of 1.42
// (voicememos-multi-folder-2026-09-18.png), which is 1.22 units against the
// set's own band. A band widened to a device width takes that air with it: at
// 16 dp the gap comes to 0.41 px and the two bands share a device pixel, so
// the flap and the body's top face read as one thickened band there. No
// stored capture holds the folder BELOW 24 dp, and it is not redrawn without
// one — the gap is a measurement. The capture is on the reference's list.
//
// WHAT ONE BAND COSTS THE DIAGONAL, and why the compensation went. A band at
// 45 degrees crosses a pixel corner to corner, so it covers less of one than
// an axis-aligned band of the same width. At the set's own widths it reaches
// 1.000 of the pixel it runs through at 24 dp and at 20, and 0.996 at 16,
// against the axis-aligned band's 1.000 at all three. The compensation was
// measured against an axis-aligned band that COVERED A WHOLE PIXEL at every
// size, and the two now arrive within half a per cent of each other. A
// heavier diagonal buys nothing but a mark heavier than the platform's, which
// is the defect it was drawn to avoid.
//
// WHAT THE DEVICE WIDTH BOUGHT THE SMALL SIZES. The band covers a whole
// device pixel at 16 and 20 dp as well as at 24, so a mark arrives at the
// control's own colour wherever a band lands one and not only where two bands
// cross or a figure is solid. The set drew 1.4 units at every size until
// CG5.11: at 16 dp that is 0.93 px, under a device pixel, and a mark drawn
// entirely on the diagonal never reached its own colour anywhere at that
// size. Each file states what its own bands land and reach at each of the
// three sizes.
//
// THE SECOND BAND is 0.93 units, and it is measured rather than derived:
// the three list lines inside the sidebar mark's leading column carry 2.80
// units of coverage between them over a length of 2.53 (voicememos-window.png,
// the sidebar toggle, x 108-111, y 21-31), which is 0.93 units each, and their
// centres stand on a period of 2.20. It is what the platform gives an element
// that says "a list lives here" beside a figure that says what the control is.
//
// It falls under the band at every size, taking the same widening: 1.01 px at
// 16 dp, 1.01 at 20 and 0.93 at 24, against the band's own 1.32, 1.40 and
// 1.40. Where the sidebar's three list lines stand it holds a whole device
// pixel at no size. That is the point of it rather than a miss. A band that cannot fill a device pixel is drawn at part coverage, and
// part coverage of the control's own colour is exactly what a line carried
// beside the figure reads as. The set drew this before as the band itself
// under a fill-opacity, which is the same effect asked for twice: a weight the
// platform does not draw, faded by a number nothing measured. What the
// platform draws is a thinner line, and a thinner line is what is drawn here.
//
// Every mark's file states its own bands the same way: where each leading
// edge stands, how thick it is, which of the three sizes it lands a whole
// device pixel at, and what it reaches where it lands none. The thickness is
// the set's own per-size device width; where a mark's own capture reads
// otherwise, its file states the exception. A mark drawn
// entirely on the diagonal or the curve states that the rule reaches none of
// its edges and what it has instead. The package's tests walk the whole set,
// render every mark at 16, 20 and 24 and read each stated band's coverage off
// the render against the arithmetic its own numbers give, so a file's claim
// and its drawing cannot part company.
//
// A mark that needs emphasis gets it from what it draws, not from a thicker
// line. The second band is not a lighter version of the band to
// reach for: it is what a list line, a rule inside a pane and nothing else is.
//
// A MARK THAT LAYS DOWN NO BAND TAKES NO WIDENING. The widening exists to
// hold a band to a device width, and the clear mark has none: it is a solid
// disc with a cross knocked out of it, and both the disc's extent and the
// cross's are measured figures rather than a weight. Offsetting its edges
// would thicken the one while closing the other — at 16 dp the cross's arms
// would come to 0.31 px against the 0.79 the grid gives them — so the mark
// scales with the grid and nothing else. It is the only mark in the set drawn
// that way, and the package's own rule is one predicate: bandless.
//
// # A mark standing in chrome
//
// A mark that is the whole label of a control in a toolbar is drawn at 24 —
// the top of the range above — and the grid is why. MEASURED at 1x off the
// toolbar bands in the organization's macOS reference: a square symbol's
// covered extent is 18 × 18 px (Finder's group pull-down) and 19 × 19 (its
// tag), and a round or diagonal one 16 × 17 (its magnifier) and 17 × 17
// (Notes' compose), in controls 36 px tall. A square form drawn to this set's
// 19-unit keyline at 24 is 19 px, which is the platform's own larger square
// reading to the pixel — and 24 is the size at which every whole unit of the
// grid lands on a whole device pixel.
//
// The round and curved allowance is measured off the same band: the
// magnifier's lens is 13.06 px across outside there, so a round form drawn to
// the set's 13 units at 24 dp covers the platform's own lens to a sixteenth
// of a pixel. The whole magnifier, handle included, comes out 16.2 units
// against the capture's 15.75 by 15.99. The clear mark's disc is the same
// allowance drawn solid, and the platform's own clear control measures 13.04
// px across. The DIAGONAL allowance is the one number here still unmeasured:
// 20 units at 24 dp is 20 px where the nearest diagonal figure the reference
// holds, Notes' compose, covers 17 × 17, so the checkmark stands wider than
// the platform's own. The capture that would settle it is on the reference's
// list.
//
// The weight is the set's one measured band, which at 24 dp is 1.4 units and
// 1.40 px at once — the 1.35 to 1.48 px those same symbols measure, read
// against each symbol's own drawn plateau. components/internal/control carries
// the size as ChromeMarkDp, with the readings.
//
// # Drawing a mark
//
// Author outlines, not strokes. Gio's clip.Stroke exposes neither line cap nor
// line join, so the backend renders every stroked path butt-capped and mitred
// whatever the file asks for. Draw the band as a closed contour instead —
// outer contour one direction, inner contour the other — and the caps and
// joins become geometry that comes out as drawn.
//
// Winding is non-zero and nothing else. Gio's outline fills by the non-zero
// rule and has no even-odd mode, so a hole has to be wound against the contour
// that contains it. The sidebar's pane shows the pattern: the outer rounded
// rectangle runs clockwise and the inner one against it, in one path.
//
// Give every contour a solid fill (fill="#000000"). The value is ignored — the
// control's colour is applied at paint time — but a path with no fill at all
// is skipped, so the attribute has to be there.
//
// A faint element — a hairline inside a pane — can be authored with
// fill-opacity. Per-path opacity survives into the cached drawing and
// modulates the control's colour rather than replacing it, so a faint element
// stays faint in every theme. Marks are otherwise monochrome: gradients are
// painted as flat coverage, because a mark that carries its own colours cannot
// take the control's.
//
// # Colour and cost
//
// A mark's ops are built once per name and pixel size and recorded into a
// macro the set holds; every later frame at that size replays the macro.
// Colour is deliberately outside the macro — the painter emits a colour
// operation and then the geometry — so the colour is not part of the cache
// key and a control animating its foreground costs no rebuild. The painter
// leaves that colour selected when it returns, exactly as paint.Fill does.
//
// Marks are parsed on first use, once per set, and painters are cheap enough
// to build per frame: a painter is a lookup and a closure over the set's
// shared cache, so a call site does not have to hold one. The cache is guarded
// by a mutex, and the parsed drawings never leave the package, so nothing
// outside can be caught mid-resize by another goroutine.
package icons

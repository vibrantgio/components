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
// A name says what the control is, never what the picture contains: sidebar,
// disclosure, history-back — not panel-with-lines, triangle, left-chevron. The
// drawing behind a name is free to change, and does change between platforms;
// the name is the part call sites store, so it has to survive that. Names are
// lowercase ASCII with words joined by "-", the qualifier last
// (history-back, history-forward), and they are stable once published.
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
// Every mark is drawn on a 24×24 grid — viewBox "0 0 24 24" — and a square
// form is drawn to the keyline 3 units in from every edge, an 18×18 box. A
// round or diagonal form may reach 2 units, a 20×20 box, because a circle
// inside the same bounds as a square reads smaller than it; that is optical
// compensation and it is the only reason to cross the keyline.
//
// 24 is chosen for the sizes the library actually draws icons at: 16, 20 and
// 24 dp — the control's content box at each density, plus the top of the
// range. It maps one unit to one device pixel at the largest of those, it
// divides evenly enough to put the keyline and the band on whole pixels at the
// smallest (below), and it is the grid the platforms publish their own
// drawings on, so a mark can be traced against a reference without rescaling
// arithmetic in between.
//
// No grid is pixel-exact at all three sizes — 16, 20 and 24 share only the
// factor 4 — so the honest statement is which lines land where. At 1 device
// pixel per dp, a coordinate falls on a whole pixel at 24 dp on every whole
// unit, at 16 dp on every multiple of 1.5, and at 20 dp only on multiples of
// 1.2. A coordinate whole at both 16 and 24 dp is therefore a multiple of 3,
// and one whole at all three is a multiple of 6. So author on the 1.5 sub-grid,
// prefer multiples of 3, put a drawing's dominant structure on multiples of 6
// where the picture allows, and check the mark rendered at all three sizes
// rather than trusting the arithmetic.
//
// # The stroke
//
// One weight for the whole set, written as two measures: a band 1.5 units
// thick where its edges are axis-aligned, and 2 units where they run
// diagonally. The two are the same weight on screen, which is the point. One
// number is not, and the set was drawn with one number first: the marks came
// out uneven, and the paragraphs below are what the pixels said about it.
//
// The floor fixes the axis-aligned number. Below one device pixel an
// antialiased line is drawn as grey rather than as the control's colour, so the
// mark quietly loses contrast against the label beside it; 1.25 units would be
// 0.83 px at the smallest size the library draws, and that is the failure. 1.5
// is the thinnest weight that never falls under a device pixel across the
// range: at 1 px per dp it is 1.0 px at 16 dp, 1.25 px at 20 dp and 1.5 px at
// 24 dp, and double that at 2 px per dp.
//
// Heavier buys nothing there. 1.5 units already covers whole pixels and already
// comes out at the control's colour undiluted; 2 units would be 1.33 px at
// 16 dp, off the pixel grid the keyline and the band are chosen together to
// land on (below), for ink that is already full. And the reading that a heavier
// band would sit in a bolder register than the platform's own marks does not
// survive being measured: the platform's axis-aligned band is 1.26 px at 16 pt
// against this set's 1.0, so the set is the lighter of the two and not the
// heavier. What settles a measure is how much of a pixel the band covers, not
// how its number compares to somebody else's.
//
// The keyline and the band are chosen together, and that is what makes the
// small sizes crisp. A band running from unit 3 to unit 4.5 covers device
// pixels 2 to 3 at 16 dp — one whole pixel of ink, landed on the pixel grid,
// at the size where half a pixel of error is the largest share of the mark —
// and pixels 4 to 6 at 16 dp on a 2 px/dp display. At 24 dp it covers 3 to 4.5,
// and 6 to 9 at 2 px/dp, exact again. Only 20 dp lands off the grid, at 2.5 to
// 3.75; the alternative would be a band of 1.2 units, the only width that is
// whole-pixel there, and it falls to 0.8 px at 16 dp and breaks the floor. So
// the set is crisp at two of its three sizes and soft at the third, which is
// the best one geometry can do for all three — and it is why a mark's straight
// edges sit on the keyline or on the 1.5 sub-grid rather than a fraction in
// from it.
//
// A diagonal edge cannot land on the grid at all, and that is why it takes a
// measure of its own. A band at 45 degrees crosses a pixel corner to corner,
// so it covers a whole one only from √2 px across; at 1.5 units it is 1.0 px
// at 16 dp and covers 91% of the pixel it runs through. That is the number the
// geometry gives, and it is not the number the eye gets: the backend
// composites in linear light, where 91% of a pixel's area comes out at 67% of
// the ink on screen. Beside an axis-aligned band at 100% the mark reads grey.
//
// 2 units is the measure that closes it. The band is 1.33 px at 16 dp, 1.67 at
// 20 and 2.0 at 24; it covers better than 99.7% of a pixel at all three, and it
// comes out at 96 to 99% of the ink — level with the axis-aligned band to an
// eye, and level with the platform's own chevrons rendered at the same sizes,
// which measure 94 to 100% at 16 pt and 100% at 20 and 24. Under 2 the mark is
// still the lighter sibling: 1.75 units comes out at 80 to 86%. Over 2 nothing
// is left to buy — coverage is already whole, the ink stops rising, and the
// drawing outgrows the 20-unit diagonal allowance, which a 2-unit band on a
// centre line from 3 to 21 fills exactly.
//
// Compensating the diagonal is not this set's invention; only the size of the
// compensation is. The platform draws the same pair heavier on the diagonal —
// its sidebar's band measures 1.26 px at 16 pt and 1.89 at 24, its chevron's
// 1.44 and 2.18, about a seventh more. This set needs a third more, because a
// backend compositing in linear light punishes a part-covered pixel harder
// than one compositing in the encoded space: the same 91% of a pixel's area is
// 67% of the ink here and 91% of it there. So the ratio is not portable, and a
// measure taken off another set's drawing is not either. Render the mark and
// read the pixels.
//
// Every mark uses one of these two numbers, and which one is the mark's edges
// to decide, never its importance. A mark that needs emphasis gets it from
// what it draws, not from a thicker line.
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
// A secondary element, the faint hairline inside a pane, is authored with
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

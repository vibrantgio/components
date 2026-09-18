package icons_test

import (
	"math"
	"testing"

	"gioui.org/layout"

	"github.com/vibrantgio/components/icons"
)

// The set's grid, as icons' package documentation states it: a 24-unit
// square, a square keyline of 19 units, and a band that is a DEVICE width
// rather than a unit one — 1.32 px below 20 dp, 1.40 from 20 dp up, and the
// grid's own 1.4 units at 24 dp, where the two are the same number. The
// second band is 0.93 units, and it takes the same widening as the first,
// because the widening is an offset of every edge the drawing lays down.
//
// A band holds a whole device pixel inside it when its own span contains one,
// which at these widths is a leading edge on a pixel boundary or no more than
// the band's own width less one below the next: 0.40 px at 24 dp, 0.40 at 20
// and 0.32 at 16. The set drew 1.4 units at every size until CG5.11 and
// landed nothing at 16 dp at all.
const (
	gridUnits = 24.0
	// bandUnits is the band at 24 dp, where one grid unit is one device
	// pixel, and the width a band entry carries when it states none.
	bandUnits = 1.4
	// bandPxFrom20 and bandPxBelow20 are the device widths the set draws its
	// band at below 24 dp: MEASURED off Mail's compose at 16 px covered and
	// Voice Memos' sidebar toggle at 19.31, and off the search field's
	// magnifier at 12.33.
	bandPxFrom20  = 1.40
	bandPxBelow20 = 1.32
	// secondBand is the measured weight the sidebar mark's list lines
	// take, stated by the entries that carry it.
	secondBand = 0.93
	// A band's edges land on eighths of a pixel at these sizes and the
	// rasterizer's own rounding is well inside that, which is the slack every
	// reading here is held to.
	bandTolerance = 0.03
)

// bandPx is the device width the set draws its band at for a mark rendered at
// px device pixels, one pixel to the dp. It is the package's own rule
// restated, so a change to either has to move the other.
func bandPx(px int) float64 {
	switch {
	case px >= gridUnits:
		return bandUnits * float64(px) / gridUnits
	case px >= 20:
		return bandPxFrom20
	default:
		return bandPxBelow20
	}
}

// widen is how much wider than the grid's own 1.4 units the band comes out at
// px device pixels. The set spends half of it on every edge the drawing lays
// down, so a band's leading edge stands that much below where the grid puts
// it and its trailing edge that much above.
func widen(px int) float64 {
	return bandPx(px) - bandUnits*float64(px)/gridUnits
}

// markPx are the sizes the library draws icons at, and the three the grid is
// chosen for.
var markPx = []int{16, 20, 24}

// axis says which way a band runs, and so which line it is read off.
type axis uint8

const (
	// across: the band's length runs across the mark, so a COLUMN crosses it.
	across axis = iota
	// down: the band's length runs down the mark, so a ROW crosses it.
	down
)

// band is one of a mark's axis-aligned bands as its file states it: which way
// it runs, where its leading edge stands in grid units, how thick it is, how
// far it runs, and where on the other axis it is read. A zero width is the
// band the figure itself is drawn at.
//
// The run matters because a band widened to a device width can share a device
// pixel with the band beside it: the folder's flap and the body's top face
// stand 1.2 units apart, which is under a pixel at 16 dp once both have
// grown. So a line is read against every band the mark draws that REACHES
// that line, and the run is what says which those are.
type band struct {
	what     string
	axis     axis
	lead     float64
	width    float64
	from, to float64
	along    float64
	// drawnIn is which of the file's paths lays the band down. Two bands in
	// one path fill as one shape and a device pixel they share holds their
	// sum; two in different paths are painted one over the other, and a
	// shared pixel holds a + b - ab instead. The folder's flap and its body
	// are two paths and they share a pixel at 16 dp, so the difference is
	// read rather than assumed.
	drawnIn int
}

// thick reports the band's weight: what it states, or the band the figure itself is drawn at.
func (b band) thick() float64 {
	if b.width == 0 {
		return bandUnits
	}
	return b.width
}

// reaches reports whether the band runs across the line at u.
func (b band) reaches(u float64) bool { return u >= b.from && u <= b.to }

// markBands is the whole set, mark by mark: every band whose edges run along
// the grid, with the leading edge and the weight its file states. A mark drawn
// entirely on the diagonal or the curve carries no entry in the slice — the
// rule has no edge to reach — and its file says so.
//
// The keys are registry keys, so a platform's own drawing is held to its own
// row: the two sidebar drawings are one figure and both are read.
var markBands = map[string][]band{
	// Two bars, each 10.6 to 12 — the trailing edge on the whole unit 12 —
	// inside arms that run the keyline from 2.5 to 21.5. plus.svg states both.
	"plus": {
		{"the crossbar", across, 10.6, 0, 2.5, 21.5, 6, 0},
		{"the upright", down, 10.6, 0, 2.5, 21.5, 6, 1},
	},
	// The pane measured off voicememos-window.png: the seam on the measured
	// one to 2.05, and the top, the foot and the two sides on the keyline.
	// sidebar.svg records all five.
	"sidebar": {
		{"the leading side", down, 2.5, 0, 4.5, 19.5, 12, 0},
		{"the seam", down, 8.75, 0, 5.9, 18.1, 12, 1},
		{"the trailing side", down, 20.1, 0, 4.5, 19.5, 12, 0},
		{"the pane's top", across, 4.5, 0, 2.5, 21.5, 15, 0},
		{"the pane's foot", across, 18.1, 0, 2.5, 21.5, 15, 0},
	},
	// The same pane, with the three list lines macOS adds inside the leading
	// column: the second band, measured at 0.93 on a period of 2.20.
	// sidebar.darwin.svg records all eight.
	"sidebar@darwin": {
		{"the leading side", down, 2.5, 0, 4.5, 19.5, 15, 0},
		{"the seam", down, 8.75, 0, 5.9, 18.1, 15, 1},
		{"the trailing side", down, 20.1, 0, 4.5, 19.5, 15, 0},
		{"the pane's top", across, 4.5, 0, 2.5, 21.5, 15, 0},
		{"the pane's foot", across, 18.1, 0, 2.5, 21.5, 15, 0},
		{"the first list line", across, 7.985, secondBand, 5.15, 7.68, 6.5, 2},
		{"the second list line", across, 10.185, secondBand, 5.15, 7.68, 6.5, 2},
		{"the third list line", across, 12.385, secondBand, 5.15, 7.68, 6.5, 2},
	},
	// The folder measured off mail-window.png: the tab's 1.5-unit rise is an
	// extent the capture gives rather than a band, and it meets the body's own
	// top band, so a column through the tab reads the two as one run.
	// open-folder.svg records all six.
	"open-folder": {
		{"the tab's top", across, 4.5, 1.5, 2.5, 9.5, 6, 0},
		{"the body's top", across, 6, 0, 2.5, 21.5, 12, 1},
		{"the flap", across, 9.6, 0, 3.9, 20.1, 12, 2},
		{"the body's foot", across, 18.1, 0, 2.5, 21.5, 12, 1},
		{"the leading side", down, 2.5, 0, 6, 19.5, 15, 1},
		{"the trailing side", down, 20.1, 0, 6, 19.5, 15, 1},
	},
	// The sidebar's own folder, measured off
	// voicememos-multi-folder-2026-09-18.png. Its flap stands 1.2 units below
	// the body's inner top face where the open folder's stands 2.2, which is
	// the capture's own 1.24 px of air: the two share a device pixel at 16 dp
	// and the set redraws neither without a capture at that size.
	// folder.svg records all six.
	"folder": {
		{"the tab's top", across, 4.5, 1.5, 2.5, 9, 6, 0},
		{"the body's top", across, 6, 0, 2.5, 21.5, 12, 1},
		{"the flap", across, 8.6, 0, 5.15, 18.85, 12, 2},
		{"the body's foot", across, 18.1, 0, 2.5, 21.5, 12, 1},
		{"the leading side", down, 2.5, 0, 6, 19.5, 15, 1},
		{"the trailing side", down, 20.1, 0, 6, 19.5, 15, 1},
	},
	// The page measured off finder-window-untinted-dark.png. The cut itself
	// runs at 45 degrees and the rule reaches it no more than it reaches the
	// chevron. document.svg records all six bands and the cut.
	"document": {
		{"the page's top", across, 2.5, 0, 5.5, 12.5, 9, 0},
		{"the page's foot", across, 20.1, 0, 5.5, 18.5, 9, 0},
		{"the fold's crossbar", across, 9.5, 0, 11, 18.5, 15, 2},
		{"the leading side", down, 5.5, 0, 2.5, 21.5, 15, 0},
		{"the trailing side", down, 17.1, 0, 9.08, 21.5, 15, 0},
		{"the fold's upright", down, 11, 0, 3.9, 9.5, 6, 1},
	},
	// Drawn on the diagonal or the curve throughout: no edge runs along the
	// grid, so the rule reaches none of them and each file says so, with what
	// it has instead — the set's one band spent perpendicular, whichever way
	// the edge runs.
	"check":           nil,
	"chevron":         nil,
	"chevron-pair":    nil,
	"clear":           nil,
	"disclosure":      nil,
	"history-back":    nil,
	"history-forward": nil,
	"refresh":         nil,
	"search":          nil,
}

// TestEveryBandLandsWhereItsFileSaysItDoes walks the whole set and reads every
// band its file states off a render of the mark, at each of the three sizes
// the library draws icons at.
//
// The reading is the pane's: coverage linearised before it is read, because
// Gio mixes the mark into the surface in linear light and the capture stores
// the result encoded, so the stored byte is not the share of the pixel the
// mark took. The organization's macOS reference records the same rule for
// reading a component's own capture.
//
// What is checked is the drawing against the arithmetic its own numbers give,
// pixel by pixel: a band running from lo to hi covers exactly the overlap of
// [lo,hi] with each device pixel it crosses, so where the arithmetic says a
// whole pixel falls inside the band that pixel has to read full coverage on
// the render, and where it says 0.667 the render has to read 0.667. That
// holds a keyline that lands at one size only and a second band that lands
// at none to the same statement as a band sitting square on the sub-grid.
func TestEveryBandLandsWhereItsFileSaysItDoes(t *testing.T) {
	read := map[string]bool{}
	for _, goos := range []string{"windows", "darwin"} {
		set := icons.New(goos)
		for _, name := range set.Names() {
			key, ok := set.Resolve(name)
			if !ok {
				t.Errorf("%s: the set lists a name it cannot resolve", name)
				continue
			}
			if read[key] {
				continue
			}
			read[key] = true
			bands, stated := markBands[key]
			if !stated {
				t.Errorf("%s: the set carries a mark whose bands no entry states; a mark that lands nothing says so with an empty one", key)
				continue
			}
			mark := set.Mark(name)
			if mark == nil {
				t.Errorf("%s: no painter", key)
				continue
			}
			for _, px := range markPx {
				img := shoot(t, px, func(gtx layout.Context) { mark(gtx, px, black) })
				cov := coverage(img)
				for i := range bands {
					checkLanding(t, key, bands, i, px, cov)
				}
			}
		}
	}
	for key := range markBands {
		if !read[key] {
			t.Errorf("%s: a stated mark the set does not carry", key)
		}
	}
}

// checkLanding reads one band off one render and holds every device pixel it
// crosses to the share of it the mark's own numbers cover — its own span, and
// the span of every other band the mark draws that reaches the same line.
func checkLanding(t *testing.T, key string, bands []band, i, px int, cov plane) {
	t.Helper()
	b := bands[i]
	scale, w := float64(px)/gridUnits, widen(px)
	lo, hi := b.span(scale, w)
	line := cov.col(at(px, b.along/gridUnits))
	if b.axis == down {
		line = cov.row(at(px, b.along/gridUnits))
	}
	onLine := map[int][][2]float64{}
	for _, o := range bands {
		if o.axis == b.axis && o.reaches(b.along) {
			l, h := o.span(scale, w)
			onLine[o.drawnIn] = append(onLine[o.drawnIn], [2]float64{l, h})
		}
	}
	for p := int(math.Floor(lo)); p < int(math.Ceil(hi)); p++ {
		if p < 0 || p >= len(line) {
			t.Errorf("%s at %d px: %s crosses pixel %d, off the mark's own square", key, px, b.what, p)
			continue
		}
		want := covered(onLine, p)
		if got := line[p]; math.Abs(got-want) > bandTolerance {
			t.Errorf("%s at %d px: %s covers pixel %d to %.3f, want the %.3f its own %.3f-to-%.3f span gives",
				key, px, b.what, p, got, want, lo, hi)
		}
	}
}

// span is where the band's two edges stand on the render, in device pixels:
// the grid's own placement with half the widening spent outward on each.
func (b band) span(scale, w float64) (lo, hi float64) {
	return b.lead*scale - w/2, (b.lead+b.thick())*scale + w/2
}

// covered is how much of device pixel p the mark's own bands leave covered:
// the spans of one path counted once between them, and one path laid over
// another the way the rasterizer lays it, a + b - ab.
func covered(byPath map[int][][2]float64, p int) float64 {
	clear := 1.0
	for _, spans := range byPath {
		clear *= 1 - fill(spans, p)
	}
	return 1 - clear
}

// fill is how much of device pixel p one path's spans cover between them,
// counting a pixel two of them share once.
func fill(spans [][2]float64, p int) float64 {
	const steps = 512
	var n int
	for i := range steps {
		x := float64(p) + (float64(i)+0.5)/steps
		for _, s := range spans {
			if x >= s[0] && x <= s[1] {
				n++
				break
			}
		}
	}
	return float64(n) / steps
}

// wholePixelIn reports the first whole device pixel lying inside the band that
// runs from lo to hi, both in pixels from the mark's own edge.
func wholePixelIn(lo, hi float64) (int, bool) {
	const slack = 1e-6
	k := int(math.Ceil(lo - slack))
	return k, float64(k+1) <= hi+slack
}

// TestTheBandLandingRuleIsTheArithmetic pins the rule the set is authored by
// against the arithmetic it stands for. A band of w device pixels standing at
// lo holds a whole one inside it when w is at least one and lo either falls
// on a pixel boundary or stands no further than w-1 below the next — which
// for the set's own widths is 0.40 px at 24 dp, 0.40 at 20 and 0.32 at 16.
//
// That a band CAN land one at 16 dp is what the device width bought. The set
// drew 1.4 units at every size until CG5.11, which is 0.93 px at 16 dp, under
// one device pixel, and no band in the set held a whole one anywhere there.
func TestTheBandLandingRuleIsTheArithmetic(t *testing.T) {
	for i := 0; i <= 10*(gridUnits-bandUnits); i++ {
		lead := float64(i) / 10
		for _, px := range markPx {
			lo := lead*float64(px)/gridUnits - widen(px)/2
			_, whole := wholePixelIn(lo, lo+bandPx(px))
			if want := landsByRule(lo, bandPx(px)); whole != want {
				t.Errorf("a band at %.1f at %d px: whole pixel inside = %v, want %v", lead, px, whole, want)
			}
		}
	}
	// The claim the files are written against, size by size: the band covers
	// a whole device pixel at all three, so what decides a landing is where
	// the band's own leading edge stands and nothing else.
	for _, px := range markPx {
		if bandPx(px) < 1 {
			t.Errorf("the band is %.2f px at %d dp, under a whole device pixel", bandPx(px), px)
		}
		var landed int
		for lead := 0.0; lead <= gridUnits-bandUnits; lead++ {
			if landsByRule(lead*float64(px)/gridUnits-widen(px)/2, bandPx(px)) {
				landed++
			}
		}
		if landed == 0 {
			t.Errorf("no whole unit lands a device pixel at %d dp", px)
		}
	}
	// At 24 dp one grid unit is one device pixel and the widening is nothing,
	// so every whole unit lands.
	for lead := 0.0; lead <= gridUnits-bandUnits; lead++ {
		if !landsByRule(lead, bandUnits) {
			t.Errorf("a band at the whole unit %.0f lands nothing at 24 dp", lead)
		}
	}
}

// landsByRule is the closed form of the landing rule: a band of w pixels at lo
// holds a whole device pixel when w covers one and lo leaves room for it.
func landsByRule(lo, w float64) bool {
	const slack = 1e-9
	if w < 1-slack {
		return false
	}
	f := lo - math.Floor(lo)
	if f < slack || 1-f < slack {
		return true
	}
	return 1-f <= w-1+slack
}

// bestPixelIn is the largest share of a device pixel the band from lo to hi
// covers: what it reaches where it lands none.
func bestPixelIn(lo, hi float64) float64 {
	var best float64
	for i := int(math.Floor(lo)); i < int(math.Ceil(hi)); i++ {
		best = max(best, min(hi, float64(i+1))-max(lo, float64(i)))
	}
	return best
}

// TestWhatTheKeylineAndTheSecondBandLand states the two prices the measured
// grid carries, so that a later change to either number has to move this
// claim with it.
//
// The square keyline is 19 units centred in a 24-unit box, which puts a
// square form's own edges at 2.5 and 20.1 — neither on a whole unit, because
// 19 is odd against 24. A band from either of them holds a whole device pixel
// at 20 dp, where the widening carries its leading edge down onto 1.967, and
// at neither of the other two: it reaches 0.793 of its best pixel at 16 dp
// and 0.900 at 24. The set drew 1.4 units at every size until CG5.11 and the
// keyline's band then landed nothing at any of the three.
//
// The second band is 0.93 units, and it takes the same widening as the first,
// so it comes out 1.01 px at 16 and at 20 dp and 0.93 at 24. Where the
// sidebar's three list lines stand it holds a whole device pixel at no size,
// which is the point of it rather than a miss: a line carried beside the
// figure reads as part coverage of the control's own colour.
func TestWhatTheKeylineAndTheSecondBandLand(t *testing.T) {
	const keyline = 19.0
	lead := (gridUnits - keyline) / 2
	reach := map[int]float64{24: 0.900, 20: 1.000, 16: 0.793}
	lands := map[int]bool{24: false, 20: true, 16: false}
	for _, edge := range []float64{lead, gridUnits - lead - bandUnits} {
		for _, px := range markPx {
			lo := edge*float64(px)/gridUnits - widen(px)/2
			hi := lo + bandPx(px)
			if _, whole := wholePixelIn(lo, hi); whole != lands[px] {
				t.Errorf("the keyline's band at %.1f at %d px: whole device pixel inside = %v, want %v", edge, px, whole, lands[px])
			}
			if got := bestPixelIn(lo, hi); math.Abs(got-reach[px]) > bandTolerance {
				t.Errorf("the keyline's band at %.1f at %d px reaches %.3f of its best pixel, want the stated %.3f", edge, px, got, reach[px])
			}
		}
	}
	for _, px := range markPx {
		w := secondBand*float64(px)/gridUnits + widen(px)
		if w >= bandPx(px) {
			t.Errorf("the second band is %.2f px at %d dp, which is the first band's own %.2f or more", w, px, bandPx(px))
		}
		for _, b := range markBands["sidebar@darwin"] {
			if b.width != secondBand {
				continue
			}
			lo := b.lead*float64(px)/gridUnits - widen(px)/2
			if _, whole := wholePixelIn(lo, lo+w); whole {
				t.Errorf("%s holds a whole device pixel at %d dp, where the second band is drawn to hold none", b.what, px)
			}
		}
	}
}

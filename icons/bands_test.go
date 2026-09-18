package icons_test

import (
	"math"
	"testing"

	"gioui.org/layout"

	"github.com/vibrantgio/components/icons"
)

// The set's grid, as icons' package documentation states it: a 24-unit
// square, a square keyline of 19 units, one measured band of 1.4 units and a
// second one of 0.93. A band of 1.4 holds a whole device pixel inside it when
// its own span contains one: at 24 dp that wants a leading edge on a whole
// unit or no more than 0.4 below one, at 20 dp a leading edge whose five
// sixths does the same against 1.167 px, and at 16 dp nothing lands anywhere,
// 0.93 px being under one device pixel. The keyline itself lands nothing at
// any size — 19 is odd against 24 — and so does the second band.
const (
	gridUnits = 24.0
	// band is the set's one measured weight, and the width a band entry
	// carries when it states none.
	bandUnits = 1.4
	// secondBand is the measured weight the sidebar mark's list lines
	// take, stated by the entries that carry it.
	secondBand = 0.93
	// A band's edges land on eighths of a pixel at these sizes and the
	// rasterizer's own rounding is well inside that, which is the slack every
	// reading here is held to.
	bandTolerance = 0.03
)

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
// it runs, where its leading edge stands in grid units, how thick it is, and
// where on the other axis it can be read with nothing else of the drawing on
// the line. A zero width is the band the figure itself is drawn at.
type band struct {
	what  string
	axis  axis
	lead  float64
	width float64
	along float64
}

// thick reports the band's weight: what it states, or the band the figure itself is drawn at.
func (b band) thick() float64 {
	if b.width == 0 {
		return bandUnits
	}
	return b.width
}

// markBands is the whole set, mark by mark: every band whose edges run along
// the grid, with the leading edge and the weight its file states. A mark drawn
// entirely on the diagonal or the curve carries no entry in the slice — the
// rule has no edge to reach — and its file says so.
//
// The keys are registry keys, so a platform's own drawing is held to its own
// row: the two sidebar drawings are one figure and both are read.
var markBands = map[string][]band{
	// Two bars, each 10.6 to 12 — the trailing edge on the whole unit 12,
	// which lands at 20 and 24 dp — inside arms that run the keyline from
	// 2.5 to 21.5. plus.svg states both.
	"plus": {
		{"the crossbar", across, 10.6, 0, 6},
		{"the upright", down, 10.6, 0, 6},
	},
	// The pane measured off voicememos-window.png: the seam on the measured
	// one to 2.05 lands at 24 dp, and the top, the foot and the two sides on
	// the keyline land nothing at all. sidebar.svg records all five.
	"sidebar": {
		{"the leading side", down, 2.5, 0, 12},
		{"the seam", down, 8.75, 0, 12},
		{"the trailing side", down, 20.1, 0, 12},
		{"the pane's top", across, 4.5, 0, 15},
		{"the pane's foot", across, 18.1, 0, 15},
	},
	// The same pane, with the three list lines macOS adds inside the leading
	// column: the second band, measured at 0.93 on a period of 2.20, which
	// holds no whole device pixel at any size. sidebar.darwin.svg records
	// all eight.
	"sidebar@darwin": {
		{"the leading side", down, 2.5, 0, 15},
		{"the seam", down, 8.75, 0, 15},
		{"the trailing side", down, 20.1, 0, 15},
		{"the pane's top", across, 4.5, 0, 15},
		{"the pane's foot", across, 18.1, 0, 15},
		{"the first list line", across, 7.985, secondBand, 6.5},
		{"the second list line", across, 10.185, secondBand, 6.5},
		{"the third list line", across, 12.385, secondBand, 6.5},
	},
	// The folder measured off mail-window.png: the body's top and the flap
	// land at 20 and 24 dp, the tab's 1.5-unit rise is an extent the capture
	// gives rather than a band, and the foot and the two sides land nothing.
	// open-folder.svg records all six.
	"open-folder": {
		{"the tab's top", across, 4.5, 1.5, 6},
		{"the body's top", across, 6, 0, 12},
		{"the flap", across, 9.6, 0, 12},
		{"the body's foot", across, 18.1, 0, 12},
		{"the leading side", down, 2.5, 0, 15},
		{"the trailing side", down, 20.1, 0, 15},
	},
	// The sidebar's own folder, measured off
	// voicememos-multi-folder-2026-09-18.png: the body's top lands at 20 and
	// 24 dp and the flap at 24, the tab's 1.5-unit rise is an extent the
	// capture gives rather than a band, and the foot and the two sides land
	// nothing. folder.svg records all six.
	"folder": {
		{"the tab's top", across, 4.5, 1.5, 6},
		{"the body's top", across, 6, 0, 12},
		{"the flap", across, 8.6, 0, 12},
		{"the body's foot", across, 18.1, 0, 12},
		{"the leading side", down, 2.5, 0, 15},
		{"the trailing side", down, 20.1, 0, 15},
	},
	// The page measured off finder-window-untinted-dark.png: only the fold's
	// two lines land anything, which is what a figure measured on both axes
	// costs on this grid. The cut itself runs at 45 degrees and the rule
	// reaches it no more than it reaches the chevron. document.svg records all
	// six bands and the cut.
	"document": {
		{"the page's top", across, 2.5, 0, 9},
		{"the page's foot", across, 20.1, 0, 9},
		{"the fold's crossbar", across, 9.5, 0, 15},
		{"the leading side", down, 5.5, 0, 15},
		{"the trailing side", down, 17.1, 0, 15},
		{"the fold's upright", down, 11, 0, 6},
	},
	// Drawn on the diagonal or the curve throughout: no edge runs along the
	// grid, so the rule reaches none of them and each file says so, with what
	// it has instead — the set's one band of 1.4 spent perpendicular,
	// whichever way the edge runs.
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
				for _, b := range bands {
					checkLanding(t, key, b, px, cov)
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
// crosses to the share of it the band's own numbers cover.
func checkLanding(t *testing.T, key string, b band, px int, cov plane) {
	t.Helper()
	scale := float64(px) / gridUnits
	lo, hi := b.lead*scale, (b.lead+b.thick())*scale
	line := cov.col(at(px, b.along/gridUnits))
	if b.axis == down {
		line = cov.row(at(px, b.along/gridUnits))
	}
	for i := int(math.Floor(lo)); i < int(math.Ceil(hi)); i++ {
		if i < 0 || i >= len(line) {
			t.Errorf("%s at %d px: %s crosses pixel %d, off the mark's own square", key, px, b.what, i)
			continue
		}
		want := min(hi, float64(i+1)) - max(lo, float64(i))
		if got := line[i]; math.Abs(got-want) > bandTolerance {
			t.Errorf("%s at %d px: %s covers pixel %d to %.3f, want the %.3f its own %.3f-to-%.3f span gives",
				key, px, b.what, i, got, want, lo, hi)
		}
	}
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
// lo holds a whole one inside it when w is at least one and lo either falls on
// a pixel boundary or stands no further than w-1 below the next — which for
// the set's measured 1.4 units is 0.4 of a unit at 24 dp, 0.167 px at 20 and
// nothing at all at 16, where the band is 0.93 px.
func TestTheBandLandingRuleIsTheArithmetic(t *testing.T) {
	for i := 0; i <= 10*(gridUnits-bandUnits); i++ {
		lead := float64(i) / 10
		for _, px := range markPx {
			scale := float64(px) / gridUnits
			lo := lead * scale
			_, whole := wholePixelIn(lo, lo+bandUnits*scale)
			if want := landsByRule(lo, bandUnits*scale); whole != want {
				t.Errorf("a band at %.1f at %d px: whole pixel inside = %v, want %v", lead, px, whole, want)
			}
		}
	}
	// The three claims the files are written against, stated size by size:
	// every whole unit lands at 24 dp, a whole unit lands at 20 dp when it is
	// 0 or 1 past a multiple of six — five sixths of it falling on a pixel
	// boundary or a sixth under one — and nothing lands at 16.
	for lead := 0.0; lead <= gridUnits-bandUnits; lead++ {
		if !landsByRule(lead, bandUnits) {
			t.Errorf("a band at the whole unit %.0f lands nothing at 24 dp", lead)
		}
		lo := lead * 20 / gridUnits
		got := landsByRule(lo, bandUnits*20/gridUnits)
		if want := math.Mod(lead, 6) <= 1; got != want {
			t.Errorf("a band at the whole unit %.0f at 20 dp: lands = %v, want %v", lead, got, want)
		}
	}
	for i := 0; i <= 10*(gridUnits-bandUnits); i++ {
		lo := float64(i) / 10 * 16 / gridUnits
		if landsByRule(lo, bandUnits*16/gridUnits) {
			t.Errorf("a band at %.1f lands a whole device pixel at 16 dp, where 1.4 units is %.2f px", float64(i)/10, bandUnits*16/gridUnits)
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

// TestTheKeylineAndTheSecondBandLandNothing states the two prices the measured
// grid carries, so that a later change to either number has to move this claim
// with it.
//
// The square keyline is 19 units centred in a 24-unit box, which puts a square
// form's own edges at 2.5 and 20.1 — neither on a whole unit, because 19 is
// odd against 24 — so a band of 1.4 from either of them holds a whole device
// pixel at no size at all and reaches 0.900 of its best pixel at 24 dp, 0.917
// at 20 and 0.600 at 16. The second band is 0.93 units, under one device pixel
// at every size the library draws, so it lands one nowhere either.
func TestTheKeylineAndTheSecondBandLandNothing(t *testing.T) {
	const keyline = 19.0
	lead := (gridUnits - keyline) / 2
	reach := map[int]float64{24: 0.900, 20: 0.917, 16: 0.600}
	for _, edge := range []float64{lead, gridUnits - lead - bandUnits} {
		for _, px := range markPx {
			scale := float64(px) / gridUnits
			lo := edge * scale
			if _, whole := wholePixelIn(lo, lo+bandUnits*scale); whole {
				t.Errorf("the keyline's band at %.1f at %d px holds a whole device pixel, and the keyline's odd 19 units cannot", edge, px)
			}
			if got := bestPixelIn(lo, lo+bandUnits*scale); math.Abs(got-reach[px]) > bandTolerance {
				t.Errorf("the keyline's band at %.1f at %d px reaches %.3f of its best pixel, want the stated %.3f", edge, px, got, reach[px])
			}
		}
	}
	for _, px := range markPx {
		if secondBand*float64(px)/gridUnits >= 1 {
			t.Errorf("the second band is %.2f px at %d dp, a whole device pixel or more", secondBand*float64(px)/gridUnits, px)
		}
	}
}

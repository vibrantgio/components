package icons_test

import (
	"math"
	"testing"

	"gioui.org/layout"

	"github.com/vibrantgio/components/icons"
)

// The set's grid, as icons' package documentation states it: a 24-unit
// square, a square keyline of 19 units, an axis-aligned band of 1.5
// units and a second one of 0.93. A band of 1.5 lands a whole device pixel
// inside it at 16, 20 and 24 dp alike when its leading edge is 1.5m with
// m ≡ 0 or 3 (mod 4); from every other multiple of 1.5 it reaches 0.75 of one
// at 20 dp and a whole one at the other two. The keyline itself is off that
// sub-grid — 19 is odd against 24 — so a square form's own edges land at 24 dp
// alone, and the second band lands nothing anywhere, being under one device
// pixel at every size.
const (
	gridUnits = 24.0
	// band is the axis-aligned weight, and the width a band entry
	// carries when it states none.
	bandUnits = 1.5
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
	// Two bars, each 10.5 to 12 — m = 7, which lands at every size — inside
	// arms that run the keyline from 2.5 to 21.5. plus.svg states both.
	"plus": {
		{"the crossbar", across, 10.5, 0, 6},
		{"the upright", down, 10.5, 0, 6},
	},
	// The pane measured off voicememos-window.png: the top and the foot land
	// at every size, and the two sides on the keyline and the seam on the
	// measured one to 2.05 land at 24 dp alone. sidebar.svg records all five.
	"sidebar": {
		{"the leading side", down, 2.5, 0, 12},
		{"the seam", down, 8.75, 0, 12},
		{"the trailing side", down, 20, 0, 12},
		{"the pane's top", across, 4.5, 0, 15},
		{"the pane's foot", across, 18, 0, 15},
	},
	// The same pane, with the three list lines macOS adds inside the leading
	// column: the second band, measured at 0.93 on a period of 2.20, which
	// holds no whole device pixel at any size. sidebar.darwin.svg records
	// all eight.
	"sidebar@darwin": {
		{"the leading side", down, 2.5, 0, 15},
		{"the seam", down, 8.75, 0, 15},
		{"the trailing side", down, 20, 0, 15},
		{"the pane's top", across, 4.5, 0, 15},
		{"the pane's foot", across, 18, 0, 15},
		{"the first list line", across, 7.985, secondBand, 6.5},
		{"the second list line", across, 10.185, secondBand, 6.5},
		{"the third list line", across, 12.385, secondBand, 6.5},
	},
	// The folder measured off mail-window.png: three bands across that land
	// at every size, a flap on the capture's own air, and two sides on the
	// keyline. open-folder.svg records all six.
	"open-folder": {
		{"the tab's top", across, 4.5, 0, 6},
		{"the body's top", across, 6, 0, 12},
		{"the flap", across, 9.75, 0, 12},
		{"the body's foot", across, 18, 0, 12},
		{"the leading side", down, 2.5, 0, 15},
		{"the trailing side", down, 20, 0, 15},
	},
	// The sidebar's own folder, measured off
	// voicememos-multi-folder-2026-09-18.png: the tab's top, the body's top
	// and its foot land at every size, and the flap and the two sides at 24
	// dp alone. folder.svg records all six.
	"folder": {
		{"the tab's top", across, 4.5, 0, 6},
		{"the body's top", across, 6, 0, 12},
		{"the flap", across, 8.75, 0, 12},
		{"the body's foot", across, 18, 0, 12},
		{"the leading side", down, 2.5, 0, 15},
		{"the trailing side", down, 20, 0, 15},
	},
	// The page measured off finder-window-untinted-dark.png: nothing here is
	// on the 1.5 sub-grid, which is what a figure measured on both axes costs
	// on this grid. The cut itself runs at 45 degrees and the rule reaches it
	// no more than it reaches the chevron. document.svg records all six bands
	// and the cut.
	"document": {
		{"the page's top", across, 2.5, 0, 9},
		{"the page's foot", across, 20, 0, 9},
		{"the fold's crossbar", across, 9.5, 0, 15},
		{"the leading side", down, 5.5, 0, 15},
		{"the trailing side", down, 17, 0, 15},
		{"the fold's upright", down, 10.5, 0, 6},
	},
	// Drawn on the diagonal or the curve throughout: no edge runs along the
	// grid, so the rule reaches none of them and each file says so, with what
	// it has instead — the diagonal measure of 2 units for a band at 45
	// degrees, and the 1.5 spent perpendicular for the chevrons,
	// whose arms are steeper.
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
// against the arithmetic it stands for: on this grid a band of 1.5 units
// lands a whole device pixel at 16 and 24 dp wherever its leading edge is a
// multiple of 1.5, and at 20 dp only from 1.5m with m ≡ 0 or 3 (mod 4).
func TestTheBandLandingRuleIsTheArithmetic(t *testing.T) {
	for m := 0; m*3 <= 2*(gridUnits-bandUnits); m++ {
		lead := bandUnits * float64(m)
		for _, px := range markPx {
			scale := float64(px) / gridUnits
			_, whole := wholePixelIn(lead*scale, (lead+bandUnits)*scale)
			want := px != 20 || m%4 == 0 || m%4 == 3
			if whole != want {
				t.Errorf("a band at %.1f (m=%d) at %d px: whole pixel inside = %v, want %v", lead, m, px, whole, want)
			}
		}
	}
}

// TestTheKeylineLandsAtOneSizeAndTheSecondBandAtNone states the two prices the
// measured grid carries, so that a later change to either number has to move
// this claim with it.
//
// The square keyline is 19 units centred in a 24-unit box, which puts a square
// form's own edges at 2.5 and 20 — off the 1.5 sub-grid, because 19 is odd
// against 24 — so each lands a whole device pixel at 24 dp alone. The
// second band is 0.93 units, under one device pixel at every size the
// library draws, so it lands one nowhere.
func TestTheKeylineLandsAtOneSizeAndTheSecondBandAtNone(t *testing.T) {
	const keyline = 19.0
	lead := (gridUnits - keyline) / 2
	for _, edge := range []float64{lead, gridUnits - lead - bandUnits} {
		for _, px := range markPx {
			scale := float64(px) / gridUnits
			_, whole := wholePixelIn(edge*scale, (edge+bandUnits)*scale)
			if want := px == 24; whole != want {
				t.Errorf("the keyline's band at %.1f at %d px: whole pixel inside = %v, want %v", edge, px, whole, want)
			}
		}
	}
	for _, px := range markPx {
		if secondBand*float64(px)/gridUnits >= 1 {
			t.Errorf("the second band is %.2f px at %d dp, a whole device pixel or more", secondBand*float64(px)/gridUnits, px)
		}
	}
}

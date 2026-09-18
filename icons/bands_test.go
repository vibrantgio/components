package icons_test

import (
	"math"
	"testing"

	"gioui.org/layout"

	"github.com/vibrantgio/components/icons"
)

// The set's grid, as icons' package documentation states it: a 24-unit
// square, and an axis-aligned band of 1.5 units. A band lands a whole device
// pixel inside it at 16, 20 and 24 dp alike when its leading edge is 1.5m
// with m ≡ 0 or 3 (mod 4); from every other multiple of 1.5 it reaches 0.75
// of one at 20 dp and a whole one at the other two.
const (
	gridUnits = 24.0
	bandUnits = 1.5
	// A band's edges land on eighths of a pixel at these sizes and the
	// rasterizer's own rounding is well inside that, which is the slack the
	// pane's own reading is held to.
	bandTolerance = 0.03
	// What a band that does not land reaches at 20 dp: 1.25 px of band with
	// neither edge on the grid covers three quarters of its best pixel.
	missedCoverage = 0.75
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
// it runs, where its leading edge stands in grid units, and where on the other
// axis it can be read with nothing else of the drawing on the line.
type band struct {
	what  string
	axis  axis
	lead  float64
	along float64
}

// markBands is the whole set, mark by mark: every band whose edges run along
// the grid, with the leading edge its file states. A mark drawn entirely on
// the diagonal or the curve carries no entry in the slice — the rule has no
// edge to reach — and its file says so.
//
// The keys are registry keys, so a platform's own drawing is held to its own
// row: the two sidebar drawings are one figure and both are read.
var markBands = map[string][]band{
	// Two bars, each 10.5 to 12 — m = 7, which lands. plus.svg states the
	// reading the pair is drawn against.
	"plus": {
		{"the crossbar", across, 10.5, 6},
		{"the upright", down, 10.5, 6},
	},
	// The pane measured off voicememos-window.png: the top and the foot land
	// at every size, and the two sides and the seam keep the measurement and
	// miss at 20. sidebar.svg records all five.
	"sidebar": {
		{"the leading side", down, 3, 12},
		{"the seam", down, 9, 12},
		{"the trailing side", down, 19.5, 12},
		{"the pane's top", across, 4.5, 15},
		{"the pane's foot", across, 18, 15},
	},
	// The same pane: macOS adds the list lines inside the leading column and
	// moves no edge, so the five bands are read again on that drawing.
	"sidebar@darwin": {
		{"the leading side", down, 3, 12},
		{"the seam", down, 9, 12},
		{"the trailing side", down, 19.5, 12},
		{"the pane's top", across, 4.5, 15},
		{"the pane's foot", across, 18, 15},
	},
	// The folder measured off mail-window.png: four bands across that land,
	// and two sides that keep the capture's proportion and miss at 20.
	// open-folder.svg records all six.
	"open-folder": {
		{"the tab's top", across, 4.5, 6},
		{"the body's top", across, 6, 12},
		{"the flap", across, 10.5, 12},
		{"the body's foot", across, 18, 12},
		{"the leading side", down, 3, 15},
		{"the trailing side", down, 19.5, 15},
	},
	// The sidebar's own folder, measured off
	// voicememos-multi-folder-2026-09-18.png: the tab's top and the body's
	// top land, and the flap, the foot and the two sides keep the capture's
	// proportion and miss at 20. folder.svg records all six.
	"folder": {
		{"the tab's top", across, 4.5, 6},
		{"the body's top", across, 6, 12},
		{"the flap", across, 9, 12},
		{"the body's foot", across, 18, 12},
		{"the leading side", down, 3, 15},
		{"the trailing side", down, 19.5, 15},
	},
	// The page measured off finder-window-untinted-dark.png: the two sides
	// and the fold's upright land, and the top, the foot and the fold's
	// crossbar keep the capture's proportion and miss at 20. The cut itself
	// runs at 45 degrees and the rule reaches it no more than it reaches the
	// chevron. document.svg records all five bands and the cut.
	"document": {
		{"the page's top", across, 3, 9},
		{"the page's foot", across, 19.5, 9},
		{"the fold's crossbar", across, 9, 13.5},
		{"the leading side", down, 6, 15},
		{"the trailing side", down, 16.5, 15},
		{"the fold's upright", down, 10.5, 6},
	},
	// Drawn on the diagonal or the curve throughout: no edge runs along the
	// grid, so the rule reaches none of them and each file says so, with what
	// it has instead — the diagonal measure of 2 units, which covers a whole
	// device pixel at all three sizes.
	"check":           nil,
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
// What is checked is the claim, not the arithmetic alone: where the rule says
// a whole device pixel falls inside the band, the pixel the arithmetic names
// has to read full coverage on the rendered mark; where it says none does, no
// pixel of the band's own span may, and the best of them has to reach the
// three quarters the grid's note states.
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

// checkLanding reads one band off one render and holds it to what the rule
// claims for that size.
func checkLanding(t *testing.T, key string, b band, px int, cov plane) {
	t.Helper()
	scale := float64(px) / gridUnits
	lo, hi := b.lead*scale, (b.lead+bandUnits)*scale
	line := cov.col(at(px, b.along/gridUnits))
	if b.axis == down {
		line = cov.row(at(px, b.along/gridUnits))
	}
	if k, whole := wholePixelIn(lo, hi); whole {
		if k < 0 || k >= len(line) {
			t.Errorf("%s at %d px: %s puts its whole pixel at %d, off the mark's own square", key, px, b.what, k)
			return
		}
		if got := line[k]; got < 1-bandTolerance {
			t.Errorf("%s at %d px: %s covers pixel %d to %.3f, want the whole of it — the band lands off the grid", key, px, b.what, k, got)
		}
		return
	}
	best, at := 0.0, -1
	for i := int(math.Floor(lo)); i < int(math.Ceil(hi)); i++ {
		if i < 0 || i >= len(line) {
			continue
		}
		if line[i] > best {
			best, at = line[i], i
		}
	}
	if best >= 1-bandTolerance {
		t.Errorf("%s at %d px: %s covers pixel %d whole (%.3f); the rule says no pixel of it can be", key, px, b.what, at, best)
	}
	if math.Abs(best-missedCoverage) > bandTolerance {
		t.Errorf("%s at %d px: %s reaches %.3f of its best pixel, want the %.2f a band that misses reaches", key, px, b.what, best, missedCoverage)
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

package icons_test

import (
	"math"
	"testing"

	"gioui.org/layout"

	"github.com/vibrantgio/components/icons"
)

// cell is one square of the grid, named, to be read off a render.
type cell struct {
	u, v int
	what string
}

// The badge's own squares. A unit square listed under knocked is covered by
// the plus and nothing else; one listed under solid stands inside the disc
// and clear of all four arms. Both lists are read off folder-plus.svg's own
// numbers — the disc of 4 about (18.8, 8.5), the crossbar 16.3 to 21.3 by
// 7.6 to 9, the upright 18 to 19.4 by 6 to 11 — and every square named is
// wholly inside what it is named for, so what the render owes it is the
// whole pixel and not a share of one.
var (
	knocked = []cell{
		{18, 8, "the crossing"},
		{17, 8, "the crossbar's leading arm"},
		{20, 8, "the crossbar's trailing arm"},
		{18, 6, "the upright's top arm"},
		{18, 10, "the upright's foot"},
	}
	solid = []cell{
		{16, 6, "the disc above the crossbar, leading of the upright"},
		{20, 6, "the disc above the crossbar, trailing of the upright"},
		{16, 10, "the disc below the crossbar, leading of the upright"},
		{20, 10, "the disc below the crossbar, trailing of the upright"},
	}
)

// TestTheBadgesPlusIsKnockedOutOfTheDisc reads the one thing the band-landing
// rule cannot: the plus is a HOLE in the badge rather than a figure drawn
// over it, so the pixels it covers come out at the fill the mark stands on
// while the disc around them comes out at the colour the painter was handed.
//
// A painter is given one colour and the mark is drawn in it, so a mark cannot
// paint the fill it stands on — it does not know it. If the plus were wound
// with the disc instead of against it, or wound as two contours instead of
// one, the figure would still render and the set's other tests would still
// pass: the disc would simply come out solid, or the crossing would. This is
// where that shows.
//
// The two sizes are the ones the badge is snapped for: 24 px is the mark at
// 24 dp on a 1x window and 48 px the same mark on a 2x one, and the band
// takes no widening at either, so every edge of the badge stands exactly
// where the grid puts it.
func TestTheBadgesPlusIsKnockedOutOfTheDisc(t *testing.T) {
	set := icons.New("darwin")
	mark := set.Mark(icons.FolderPlus)
	if mark == nil {
		t.Fatal("no painter for the folder-plus mark")
	}
	for _, px := range []int{24, 48} {
		img := shoot(t, px, func(gtx layout.Context) { mark(gtx, px, black) })
		cov := coverage(img)
		for _, c := range knocked {
			readCell(t, cov, px, c, 0)
		}
		for _, c := range solid {
			readCell(t, cov, px, c, 1)
		}
	}
}

// readCell holds every device pixel of one grid square to the coverage the
// square's own figure owes it.
func readCell(t *testing.T, cov plane, px int, c cell, want float64) {
	t.Helper()
	const tolerance = 0.02
	scale := float64(px) / 24
	x0, x1 := int(math.Round(float64(c.u)*scale)), int(math.Round(float64(c.u+1)*scale))
	y0, y1 := int(math.Round(float64(c.v)*scale)), int(math.Round(float64(c.v+1)*scale))
	for y := y0; y < y1; y++ {
		row := cov.row(y)
		for x := x0; x < x1; x++ {
			if got := row[x]; math.Abs(got-want) > tolerance {
				t.Errorf("folder-plus at %d px: %s covers pixel %d,%d to %.3f, want %.0f",
					px, c.what, x, y, got, want)
			}
		}
	}
}

// TestTheBadgeHoldsItsKnockoutAtEverySize reads the one figure the stated
// band table cannot: a HOLE in a solid figure. That table composites the
// spans it is given as a + b - ab, which can only add coverage, so it can
// state a band and not a knockout; this reads the knockout directly, as the
// coverage the disc is MISSING along a cut through one of the plus's bars.
//
// The badge lays down no band — its disc is a measured extent and its plus a
// measured span knocked out of that disc — so the path takes none of the
// widening the set spends on a band, and each bar stays the grid's own 1.4
// units wide at every size. That is what is read here. Offsetting the
// badge's edges along with the folder's closes the plus instead: the arm
// came out 0.93 px at 20 dp and 0.55 at 16 against the 1.17 and 0.93 the
// grid gives it.
func TestTheBadgeHoldsItsKnockoutAtEverySize(t *testing.T) {
	// folder-plus.svg's own numbers: the disc of 4 about (18.8, 8.5), the
	// upright 18 to 19.4 by 6 to 11 and the crossbar 16.3 to 21.3 by 7.6
	// to 9, so each bar is 1.4 wide and each arm reaches 2.5 from the
	// badge's centre.
	const (
		discRadius = 4.0
		badgeX     = 18.8
		badgeY     = 8.5
		barWidth   = 1.4
		armReach   = 2.5
		// A cut's reading is a sum of whole pixels, each of which the
		// rasterizer rounds to a 255th. This is the slack that leaves.
		knockoutSlack = 0.03
	)
	// armClear is how far from the badge's centre a whole device pixel still
	// stands inside the disc everywhere along a cut through an arm: the
	// disc's half-chord at the arm's own far end.
	armClear := math.Sqrt(discRadius*discRadius - armReach*armReach)

	mark := icons.New("darwin").Mark(icons.FolderPlus)
	if mark == nil {
		t.Fatal("no painter for the folder-plus mark")
	}
	for _, px := range []int{16, 20, 24, 48} {
		scale := float64(px) / 24
		cov := coverage(shoot(t, px, func(gtx layout.Context) { mark(gtx, px, black) }))
		for _, c := range []struct {
			what string
			// column says the cut runs down the mark rather than across it.
			column bool
			// far is the arm's own far end, an arm's reach from the
			// badge's centre: the cut is the first whole device pixel
			// inside the arm from there.
			far float64
			// centre is the badge's centre on the axis the cut runs along.
			centre float64
		}{
			{"the upright's top arm", false, badgeY - armReach, badgeX},
			{"the crossbar's leading arm", true, badgeX - armReach, badgeY},
		} {
			cut := int(math.Ceil(c.far * scale))
			read := cov.row(cut)
			if c.column {
				read = cov.col(cut)
			}
			var deficit float64
			for p := int(math.Ceil((c.centre - armClear) * scale)); p < int(math.Floor((c.centre+armClear)*scale)); p++ {
				deficit += 1 - read[p]
			}
			if want := barWidth * scale; math.Abs(deficit-want) > knockoutSlack {
				t.Errorf("folder-plus at %d px: %s leaves %.3f px of the disc uncovered, want the %.3f its own band of 1.4 units gives",
					px, c.what, deficit, want)
			}
		}
	}
}

// TestTheBadgeKeepsItsClearanceAtEverySize reads the air between the folder's
// outline and the badge's disc, which is what tells the badge from the folder
// it stands on. The outline is cut round the badge rather than painted over
// it, so the cut's own round is the only thing holding that air open.
//
// The reading runs STRAIGHT DOWN from the badge's centre, where the cut's arc
// dips to its lowest and the folder's trailing side stands under it: that is
// the line from the disc's centre through the nearest point of the outline,
// and the air read along it is the air itself rather than a chord of it. It
// is the coverage MISSING between the two, summed over the device pixels of
// the column the centre falls in — a sub-pixel reading, because the air comes
// to about a pixel and no whole pixel of the column need be clear of both.
//
// The window starts under the plus's foot, so that the hole the plus leaves
// in the disc is not read as air, and ends at the first pixel boundary past
// where the outline is owed to stand.
func TestTheBadgeKeepsItsClearanceAtEverySize(t *testing.T) {
	// folder-plus.svg's own numbers: the disc of 4 about (18.8, 8.5), the
	// plus's upright reaching its foot at 11, and the cut a round of 5.4 —
	// the disc and one band — about the same centre.
	const (
		discRadius = 4.0
		badgeX     = 18.8
		badgeY     = 8.5
		plusFoot   = 11.0
		// A reading summed over whole pixels each of which the rasterizer
		// rounds to a 255th. This is the slack that leaves.
		clearSlack = 0.03
	)
	mark := icons.New("darwin").Mark(icons.FolderPlus)
	if mark == nil {
		t.Fatal("no painter for the folder-plus mark")
	}
	for _, px := range []int{16, 20, 24, 48} {
		scale := float64(px) / gridUnits
		// want is the air the mark's own rule owes at this size: the band of
		// air the grid draws between the cut and the disc, and never less
		// than one device pixel.
		want := max(1, bandUnits*scale)
		cov := coverage(shoot(t, px, func(gtx layout.Context) { mark(gtx, px, black) }))
		col := cov.col(int(badgeX * scale))
		from := int(math.Ceil(plusFoot * scale))
		to := int(math.Ceil((badgeY+discRadius)*scale + want))
		var clear float64
		for y := from; y < to; y++ {
			clear += 1 - col[y]
		}
		if clear+clearSlack < want {
			t.Errorf("folder-plus at %d px: the outline stands %.3f px clear of the disc, want the %.3f its own band of air gives, and never under a device pixel",
				px, clear, want)
		}
	}
}

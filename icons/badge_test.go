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
// numbers — the disc of 4 about (19.5, 8.5), the crossbar 17 to 22 by 7.6 to
// 9, the upright 18.6 to 20 by 6 to 11 — and every square named is wholly
// inside what it is named for, so what the render owes it is the whole pixel
// and not a share of one.
var (
	knocked = []cell{
		{19, 8, "the crossing"},
		{17, 8, "the crossbar's leading arm"},
		{20, 8, "the crossbar's trailing arm"},
		{19, 6, "the upright's top arm"},
		{19, 10, "the upright's foot"},
	}
	solid = []cell{
		{17, 6, "the disc above the crossbar, leading of the upright"},
		{21, 6, "the disc above the crossbar, trailing of the upright"},
		{17, 10, "the disc below the crossbar, leading of the upright"},
		{21, 10, "the disc below the crossbar, trailing of the upright"},
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

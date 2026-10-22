package icons_test

import (
	"math"
	"testing"

	"gioui.org/layout"

	"github.com/vibrantgio/components/icons"
)

// square-and-arrow-up.svg's own numbers: the box's top edge and the stubs the
// opening cuts it into, the shaft that passes between them, and the head's
// arm ends standing above that same edge.
const (
	shareTopLead    = 8.0  // the box's top band's leading edge
	shareStubEnd    = 9.2  // where the leading stub is cut
	shareStubStart  = 14.8 // where the trailing stub takes up again
	shareBandBottom = 9.4  // the top band's inner face
	shareShaftLead  = 11.3 // the shaft's leading edge
	shareShaftTrail = 12.7 // its trailing edge
	shareArmEnd     = 5.9  // where the head's two arms are cut
	shareAir        = 2.1  // the air the file draws between any two of them
)

// TestTheShareMarkDrawsOneAirEverywhere holds the file's own numbers to the
// one width: the air across the shaft and the air under the head's arm ends
// are the same gap drawn twice, so a change to either has to move the other.
func TestTheShareMarkDrawsOneAirEverywhere(t *testing.T) {
	gaps := map[string]float64{
		"the leading stub to the shaft":    shareShaftLead - shareStubEnd,
		"the shaft to the trailing stub":   shareStubStart - shareShaftTrail,
		"the head's arm ends to the box":   shareTopLead - shareArmEnd,
		"the shaft inside the two of them": shareShaftTrail - shareShaftLead,
	}
	for what, got := range gaps {
		want := shareAir
		if what == "the shaft inside the two of them" {
			want = bandUnits
		}
		if math.Abs(got-want) > 1e-9 {
			t.Errorf("%s comes to %.3f units, want %.3f", what, got, want)
		}
	}
}

// TestTheShareMarkKeepsItsAirAtEverySize reads the air between the box's top
// edge and the arrow's shaft off a render, at each size the library draws a
// mark at and once more at 2x.
//
// The air is DRAWN rather than opened by the renderer. holdClear opens air
// only between a band and a figure knocked out of a solid one; here it stands
// between two bands of one mark, and every edge of both is offset outward by
// half the widening, so the gap closes by the whole of it at the small sizes.
// The file draws 2.1 units for that reason, and this is the reading that says
// a device pixel is still standing once both bands have grown.
//
// The reading runs along the row through the top band's centre, from a pixel
// inside the stub's straight run to the pixel boundary on the shaft's own
// centre line, which stands inside the shaft at every size read here. What is
// summed is the coverage MISSING over that window: what the stub covers and
// what the shaft covers come off it exactly, so what is left is the air. The
// band is a DEVICE width and its leading edge at 8 lands a whole device pixel
// at 24 dp alone, so the row read crosses it in part at the other sizes, and
// the share of the row the band leaves uncovered there is taken off the
// reading.
func TestTheShareMarkKeepsItsAirAtEverySize(t *testing.T) {
	// A reading summed over whole pixels each of which the rasterizer rounds
	// to a 255th. This is the slack that leaves.
	const airSlack = 0.03
	mark := icons.New("darwin").Mark(icons.SquareAndArrowUp)
	if mark == nil {
		t.Fatal("no painter for the square-and-arrow-up mark")
	}
	for _, px := range []int{16, 20, 24, 48} {
		scale, w := float64(px)/gridUnits, widen(px)
		// want is the air the set's own rule owes at this size: what the file
		// draws, less what both bands take from it, and never less than one
		// device pixel.
		want := max(1, shareAir*scale-w)
		cov := coverage(shoot(t, px, func(gtx layout.Context) { mark(gtx, px, black) }))
		// The row through the band's centre, and the share of it the band
		// covers there: its own span clipped to that row.
		top, bottom := shareTopLead*scale-w/2, shareBandBottom*scale+w/2
		y := int((shareTopLead + shareBandBottom) / 2 * scale)
		band := min(bottom, float64(y+1)) - max(top, float64(y))
		line := cov.row(y)

		centre := int(math.Round((shareShaftLead + shareShaftTrail) / 2 * scale))
		for _, side := range []struct {
			what string
			edge float64 // the stub's own end, in device pixels
			lead bool
		}{
			{"the leading stub", shareStubEnd*scale + w/2, true},
			{"the trailing stub", shareStubStart*scale - w/2, false},
		} {
			from, to := centre, int(math.Ceil(side.edge))
			stub := float64(to) - side.edge
			if side.lead {
				from, to = int(math.Floor(side.edge)), centre
				stub = side.edge - float64(from)
			}
			var clear float64
			for x := from; x < to; x++ {
				clear += 1 - line[x]
			}
			// What the band leaves uncovered in this row over the stub's own
			// run inside the window, which is not air.
			air := clear - (1-band)*stub
			if air+airSlack < want {
				t.Errorf("square-and-arrow-up at %d px: %s stands %.3f px clear of the shaft, want the %.3f its own drawn air gives, and never under a device pixel",
					px, side.what, air, want)
			}
		}
	}
}

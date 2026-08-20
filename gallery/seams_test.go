package main

import (
	"image"
	"testing"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"

	"github.com/vibrantgio/components/gallery/inventory"
	"github.com/vibrantgio/components/golden"
	"github.com/vibrantgio/theme/tokens"
)

// The seam is the run of page ground between what one section draws and the
// heading of the next. It is the section's own margin at both ends — the same
// number above the family and below it — so a column of three dozen families
// reads as one column rather than as three dozen boxes of assorted padding.
//
// A shadow or an antialiased edge lands a few pixels into that run, which is
// what a shadow is for; seamBleed is how much of it may go that way before the
// section is touching what is next to it rather than sitting clear of it.
const (
	seamGap   = int(inventory.SectionPadY)
	seamBleed = 3
)

// seamProbe is the run of page ground laid out under a section so that
// anything the body draws past its own slot has somewhere to land and be
// counted.
const seamProbe = 96

// seamInk is how far a pixel has to sit from the page ground before it counts
// as something drawn. One level is dithering and rounding; three is a mark.
const seamInk = 3

// seamMeasure is one section's seams, in pixels of the captured page.
type seamMeasure struct {
	name string
	// top is the distance from the heading's rule to the first row the body
	// draws in; bottom is the distance from the last row it draws in to the
	// heading of the section below. Zero or less is one section drawn over
	// the next.
	top, bottom int
	// left is the first column the body draws in, measured from the page's
	// own edge.
	left int
	// ink is how tall what the body draws measures, and slot is the run the
	// section asked for.
	ink, slot int
}

// TestSectionSeams measures every seam on the everything page, in both
// schemes, and holds each one to the section margin. It is the standing check
// that no family runs into the heading under it and that none of them sits in
// a hole of its own: both are a slot that disagrees with what the body draws,
// and both are invisible to a test that only asks whether the page renders.
//
// Run it with -v to read the numbers.
func TestSectionSeams(t *testing.T) {
	for _, sc := range schemes() {
		inv := testInventory(t)
		for _, grp := range inv.Groups(sc.colors) {
			for _, s := range grp.Sections {
				m := measureSeam(t, inv, sc.colors, s)
				t.Logf("%-6s %-24s slot %3d  ink %3d  top %3d  bottom %3d  left %3d",
					sc.name, m.name, m.slot, m.ink, m.top, m.bottom, m.left)
				if m.top < seamGap-seamBleed {
					t.Errorf("%s %s: %d px under its heading, want the %d px section margin",
						sc.name, m.name, m.top, seamGap)
				}
				if m.bottom < seamGap-seamBleed {
					t.Errorf("%s %s: %d px above the next heading, want the %d px section margin",
						sc.name, m.name, m.bottom, seamGap)
				}
				// A slot shorter than the body's own height is a body laid
				// out squeezed, which the seams alone would not show: the
				// family shrinks to fit and the margins stay honest.
				if nat := naturalHeight(s); nat < 1<<19 && nat > m.slot {
					t.Errorf("%s %s: slot %d is short of the %d the body measures",
						sc.name, m.name, m.slot, nat)
				}
			}
		}
	}
}

// measureSeam captures one section on its own — the group banner the column
// puts above it, its heading, its slot, and a run of bare ground under it —
// and reads the seams off the pixels rather than off what the layout claimed.
func measureSeam(t *testing.T, inv *inventory.Inventory, c tokens.ColorTokens, s inventory.Section) seamMeasure {
	t.Helper()
	items := inv.GroupItems(c, inventory.Group{Name: "Probe", Sections: []inventory.Section{s}})
	// The probe occupies its run without painting it: the page ground is
	// already under it, and a fill here would cover the very overflow the
	// probe exists to catch.
	items = append(items, func(gtx layout.Context) layout.Dimensions {
		return layout.Dimensions{Size: image.Pt(gtx.Constraints.Max.X, seamProbe)}
	})
	w := inventory.Column(items)
	total := measure(w, pageWidth, 1<<20)
	img := golden.Capture(t, total, ground(c, w))

	// The rows the column laid out above the body: the probe group's banner,
	// then the section's own heading.
	const bannerH, headingH = 44, 32
	bodyTop := bannerH + headingH
	bodyBottom := total.Y - seamProbe

	bg := img.RGBAAt(pageWidth-1, total.Y-1)
	m := seamMeasure{name: s.Name, slot: int(s.Height), top: -1, bottom: -1, left: pageWidth}
	firstInk, lastInk := -1, -1
	for y := bodyTop; y < total.Y; y++ {
		// Walked from the left and stopped at the first mark: the row's
		// leftmost is all the left margin needs, and the rows themselves say
		// where the section's ink starts and ends.
		for x := 0; x < pageWidth; x++ {
			p := img.RGBAAt(x, y)
			if absDiff(p.R, bg.R) < seamInk && absDiff(p.G, bg.G) < seamInk && absDiff(p.B, bg.B) < seamInk {
				continue
			}
			if firstInk < 0 {
				firstInk = y
			}
			lastInk = y
			if x < m.left {
				m.left = x
			}
			break
		}
	}
	if firstInk >= 0 {
		m.top = firstInk - bodyTop
		m.bottom = bodyBottom - 1 - lastInk
		m.ink = lastInk - firstInk + 1
	}
	if m.left == pageWidth {
		m.left = -1
	}
	return m
}

func absDiff(a, b uint8) int {
	if a > b {
		return int(a - b)
	}
	return int(b - a)
}

// naturalHeight is what a section's body measures when nothing bounds it: the
// height its slot has to be for the body to be laid out unsqueezed. A body
// that expands into whatever it is handed returns the bound itself, which is
// how such a body is told apart from one that has a height of its own.
func naturalHeight(s inventory.Section) int {
	var ops op.Ops
	gtx := layout.Context{
		Metric:      unit.Metric{PxPerDp: 1, PxPerSp: 1},
		Constraints: layout.Constraints{Max: image.Pt(pageWidth-2*int(inventory.SectionPadX), 1<<20)},
		Ops:         &ops,
	}
	return s.Body(gtx).Size.Y
}

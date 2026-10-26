package icons_test

import (
	"image"
	"testing"

	"gioui.org/layout"

	"github.com/vibrantgio/components/icons"
)

// TestChevronDownIsChevronRightTurnedAQuarter reads the drawn extent of both
// chevrons off a render at each size the library draws marks at. The
// down-pointing chevron is the right-pointing figure turned a quarter, so its
// extent is the right-pointing one's with width and height swapped. The box's
// centre falls on a pixel corner at all three sizes, so the turn maps the
// pixel grid onto itself and the two extents agree to the pixel.
func TestChevronDownIsChevronRightTurnedAQuarter(t *testing.T) {
	set := icons.New("darwin")
	for _, px := range []int{16, 20, 24} {
		right := drawnExtent(t, set, icons.ChevronRight, px)
		down := drawnExtent(t, set, icons.ChevronDown, px)
		if down.Dx() != right.Dy() || down.Dy() != right.Dx() {
			t.Errorf("at %d dp chevron-down covers %d by %d and chevron-right %d by %d; turned a quarter, chevron-right is %d by %d",
				px, down.Dx(), down.Dy(), right.Dx(), right.Dy(), right.Dy(), right.Dx())
		}
		if down.Dx() <= down.Dy() {
			t.Errorf("at %d dp chevron-down covers %d by %d; a chevron pointing down is wider than it is tall", px, down.Dx(), down.Dy())
		}
	}
}

// drawnExtent is the smallest rectangle that holds every pixel the mark
// paints at px.
func drawnExtent(t *testing.T, set *icons.Set, name icons.Name, px int) image.Rectangle {
	t.Helper()
	mark := set.Mark(name)
	if mark == nil {
		t.Fatalf("no painter for %q", name)
	}
	img := shoot(t, px, func(gtx layout.Context) { mark(gtx, px, black) })
	var r image.Rectangle
	b := img.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			if c := img.RGBAAt(x, y); c.R != 0xff || c.G != 0xff || c.B != 0xff {
				r = r.Union(image.Rect(x, y, x+1, y+1))
			}
		}
	}
	if r.Empty() {
		t.Fatalf("%q at %d dp paints nothing", name, px)
	}
	t.Logf("%q at %d dp covers %v", name, px, r)
	return r
}

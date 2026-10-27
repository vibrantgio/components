package input_test

import (
	"image"
	"image/color"
	"testing"

	golden "github.com/vibrantgio/components/golden"
	"github.com/vibrantgio/components/input"
	vgcolor "github.com/vibrantgio/theme/color"
	"github.com/vibrantgio/theme/tokens"
)

// TestFindBarFieldCountAndClearMarkAreTheMeasuredOnes reads the find bar
// field's count and clear mark off the drawn pixels in both schemes.
//
// MEASURED, textedit-find-match-{light,dark}.png on the field's middle row:
// the count "8" ends at x=391, the disc runs x 396–406 and the field's edge
// column is at x=413. So 4 clear columns stand between the count and the disc,
// and 6 between the disc and the edge. The disc covers 11 columns and 12 rows,
// y 43–54: it is 11 px round, and its centre falls on a row boundary. The count and the disc are the
// platform's placeholder text over the field's fill: #808080 on #ffffff
// light and #9a9a9a on #1e1e1e dark, to one level.
func TestFindBarFieldCountAndClearMarkAreTheMeasuredOnes(t *testing.T) {
	shaper := defaultShaper(t)
	size := image.Pt(300, 22)
	for _, p := range []struct {
		name        string
		col         tokens.PlatformColors
		fill, plate color.NRGBA
	}{
		{"light", tokens.PlatformLight, color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}, color.NRGBA{R: 0x80, G: 0x80, B: 0x80, A: 0xff}},
		{"dark", tokens.PlatformDark, color.NRGBA{R: 0x1e, G: 0x1e, B: 0x1e, A: 0xff}, color.NRGBA{R: 0x9a, G: 0x9a, B: 0x9a, A: 0xff}},
	} {
		t.Run(p.name, func(t *testing.T) {
			if got := vgcolor.Flatten(p.col.PlaceholderText, p.col.TextBackground); !near(got, p.plate, 1) {
				t.Fatalf("the placeholder over the field's fill = %v, want the measured %v", got, p.plate)
			}
			img := golden.Capture(t, size, onChrome(p.col.WindowBackground, input.RenderSearch(
				shaper, "Find", p.col, tokens.Spacing, tokens.Radius,
				tokens.DefaultTypography.SystemFont, tokens.Comfortable,
				input.RenderState{Text: "Gio", Count: "8", Variant: input.FindBar, Surface: p.col.WindowBackground},
			)))
			// The edge is the last column off the fill on the field's middle
			// row. The count and the disc are the columns 20 or more levels
			// off the fill in any row, the threshold the reference reads
			// their bounding boxes by; the edge is fainter than that.
			px := func(x, y int) color.NRGBA {
				c := img.RGBAAt(x, y)
				return color.NRGBA{R: c.R, G: c.G, B: c.B, A: 0xff}
			}
			edge := size.X - 1
			for edge > 0 && near(px(edge, size.Y/2), p.fill, 2) {
				edge--
			}
			off := func(x int) bool {
				for y := 0; y < size.Y; y++ {
					if diff(px(x, y), p.fill) >= 20 {
						return true
					}
				}
				return false
			}
			x := edge - 1
			clearToEdge := 0
			for ; !off(x); x-- {
				clearToEdge++
			}
			discEnd := x
			for off(x) {
				x--
			}
			discStart := x + 1
			gap := 0
			for ; !off(x); x-- {
				gap++
			}
			countEnd := x
			for off(x) {
				x--
			}
			countStart := x + 1
			if clearToEdge != 6 {
				t.Errorf("the clear mark ends %d clear columns from the field's edge, want the measured 6", clearToEdge)
			}
			if w := discEnd - discStart + 1; w != 11 {
				t.Errorf("the clear mark's disc covers %d columns, want the measured 11", w)
			}
			rows := 0
			for y := 0; y < size.Y; y++ {
				for x := discStart; x <= discEnd; x++ {
					if diff(px(x, y), p.fill) >= 20 {
						rows++
						break
					}
				}
			}
			if rows != 12 {
				t.Errorf("the clear mark's disc covers %d rows, want the measured 12", rows)
			}
			if gap != 4 {
				t.Errorf("the count ends %d clear columns before the clear mark, want the measured 4", gap)
			}
			// The plateau is the darkest pixel light and the lightest dark.
			plateau := func(x0, x1 int) color.NRGBA {
				best := p.fill
				for x := x0; x <= x1; x++ {
					for y := 0; y < size.Y; y++ {
						c := img.RGBAAt(x, y)
						g := color.NRGBA{R: c.R, G: c.G, B: c.B, A: 0xff}
						if diff(g, p.fill) > diff(best, p.fill) {
							best = g
						}
					}
				}
				return best
			}
			if got := plateau(discStart, discEnd); !near(got, p.plate, 1) {
				t.Errorf("the clear mark's plateau = %v, want the placeholder's %v", got, p.plate)
			}
			if got := plateau(countStart, countEnd); !near(got, p.plate, 1) {
				t.Errorf("the count's plateau = %v, want the placeholder's %v", got, p.plate)
			}
		})
	}
}

func diff(a, b color.NRGBA) int {
	d := func(x, y uint8) int {
		if x > y {
			return int(x - y)
		}
		return int(y - x)
	}
	return max(d(a.R, b.R), d(a.G, b.G), d(a.B, b.B))
}

func near(a, b color.NRGBA, tol int) bool { return diff(a, b) <= tol }

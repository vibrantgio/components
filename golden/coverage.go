package golden

import (
	"image"
	"image/color"
	"math"
)

// The readers below take a shape's sub-pixel geometry off a capture. They
// live here rather than in a test file because more than one package reads a
// corner the same way and the reading is the same one every time: a capture
// is rendered onto a CLEAR window, so the alpha channel of a premultiplied
// capture IS the rasterizer's coverage, on the full 255 steps and in linear
// light. That is what makes a capture readable to a hundredth of a pixel
// where the stored macOS reference's own boxes, thirteen steps of grey from
// their sheet, are readable to a tenth.
//
// A capture with something painted under the shape carries no coverage at
// all: the window is opaque everywhere and the alpha channel is flat. Lay the
// shape out bare when a corner is to be read.

// CoverageRow answers the coverage of the drawn shape over each of the first
// width columns of row y.
func CoverageRow(img *image.RGBA, y, width int) []float64 {
	c := make([]float64, width)
	for x := range c {
		c[x] = float64(img.RGBAAt(x, y).A) / 255
	}
	return c
}

// LeadingEdge answers the sub-pixel column the covered run in cov begins at:
// the first fully covered column, less the coverage standing to the left of
// it.
func LeadingEdge(cov []float64) (float64, bool) {
	for x, v := range cov {
		if v > 0.995 {
			lead := 0.0
			for _, u := range cov[:x] {
				lead += u
			}
			return float64(x) - lead, true
		}
	}
	return 0, false
}

// TrailingEdge is [LeadingEdge] from the other end.
func TrailingEdge(cov []float64) (float64, bool) {
	for x := len(cov) - 1; x >= 0; x-- {
		if cov[x] > 0.995 {
			trail := 0.0
			for _, u := range cov[x+1:] {
				trail += u
			}
			return float64(x+1) + trail, true
		}
	}
	return 0, false
}

// ArcEdge is where a rounded corner of radius r puts its edge, averaged over
// the part of pixel row y the corner reaches, against a corner whose extremes
// are x0 and y0. A row deeper than the radius sits on the straight side and
// answers x0.
//
// The mean and not the value at the row's middle: a corner's arc turns
// fastest in the row nearest its extreme, which is the row carrying most of
// the signal, and reading it at one height there is wrong by a tenth of a
// pixel.
func ArcEdge(y, x0, y0, r float64) float64 {
	const steps = 400
	sum := 0.0
	for k := range steps {
		d := y + (float64(k)+0.5)/steps - y0
		if d >= r {
			sum += x0
			continue
		}
		sum += x0 + r - math.Sqrt(math.Max(0, r*r-(r-d)*(r-d)))
	}
	return sum / steps
}

// FitCorner fits a circular corner to a profile of per-row sub-pixel edges,
// the corner's extremes pinned at (x0, y0), and answers the radius and the
// fit's rms. The scan runs one to eight pixels at a hundredth of a pixel,
// which covers every corner this design system draws on a control.
func FitCorner(profile map[int]float64, x0, y0 float64) (r, rms float64) {
	rms = math.Inf(1)
	for n := 100; n <= 800; n++ {
		try := float64(n) / 100
		s := 0.0
		for y, e := range profile {
			d := e - ArcEdge(float64(y), x0, y0, try)
			s += d * d
		}
		if got := math.Sqrt(s / float64(len(profile))); got < rms {
			r, rms = try, got
		}
	}
	return r, rms
}

// ColumnCoverageFloor is the coverage from which a column counts as covered
// by text on an opaque capture. MEASURED, save-dialog-light.png, the
// "File Format:" pop-up: its fill begins at x=264, and the title's first
// covered column, x=276, reads #cccccc on the fill #ececec at row 350, 16 %
// of the way to the text's #242424. The column before it reads the fill.
// MEASURED in the library's picker trigger: Roboto at the darker text's stem
// weight draws a 7 % fringe in the column before the title's first 10 %
// column, so the floor reads the same column as the platform's.
const ColumnCoverageFloor = 0.10

// FirstTextColumn answers the first column in [x0, x1) whose text coverage
// reaches [ColumnCoverageFloor] on any row in [y0, y1], on an opaque capture
// where the text stands on fill. The coverage of a pixel is per channel
// (fill − pixel) / (fill − text colour), the mean over the channels in which
// the two differ. The text colour is the pixel in the box farthest from
// fill. It answers false when no column reaches the floor.
func FirstTextColumn(img *image.RGBA, fill color.RGBA, x0, x1, y0, y1 int) (int, bool) {
	dist := func(c color.RGBA) int {
		return abs(int(c.R)-int(fill.R)) + abs(int(c.G)-int(fill.G)) + abs(int(c.B)-int(fill.B))
	}
	text, far := fill, 0
	for y := y0; y <= y1; y++ {
		for x := x0; x < x1; x++ {
			if c := img.RGBAAt(x, y); dist(c) > far {
				text, far = c, dist(c)
			}
		}
	}
	if far == 0 {
		return 0, false
	}
	for x := x0; x < x1; x++ {
		for y := y0; y <= y1; y++ {
			if textCoverage(img.RGBAAt(x, y), fill, text) >= ColumnCoverageFloor {
				return x, true
			}
		}
	}
	return 0, false
}

// textCoverage is the coverage of c between fill and text, as
// [FirstTextColumn] reads it.
func textCoverage(c, fill, text color.RGBA) float64 {
	var sum float64
	n := 0
	for _, ch := range [][3]uint8{{c.R, fill.R, text.R}, {c.G, fill.G, text.G}, {c.B, fill.B, text.B}} {
		if ch[1] == ch[2] {
			continue
		}
		sum += (float64(ch[1]) - float64(ch[0])) / (float64(ch[1]) - float64(ch[2]))
		n++
	}
	if n == 0 {
		return 0
	}
	return sum / float64(n)
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

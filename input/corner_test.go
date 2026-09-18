package input_test

import (
	"image"
	stdcolor "image/color"
	"math"
	"testing"

	golden "github.com/vibrantgio/components/golden"
	"github.com/vibrantgio/components/input"
	"github.com/vibrantgio/components/internal/control"
	"github.com/vibrantgio/theme/tokens"
)

// The glyph's measured 16 dp box or disc, centred in the density's checkbox
// row, at the 1:1 metric golden.Capture renders at.
const (
	glyphSide  = 16
	cornerRows = 8 // the corner's own half-height: every row it can reach into
)

// coverageRow answers the coverage of the drawn glyph over each column of one
// row of a capture.
//
// The capture's background is nothing at all — golden.Capture renders onto a
// clear window — so the alpha channel of a premultiplied capture IS the
// rasterizer's coverage, on the full 255 steps and in linear light. That is
// what makes a capture of the component readable to a hundredth of a pixel
// where the stored reference's own boxes, thirteen steps of grey from their
// sheet, are readable to a tenth.
func coverageRow(img *image.RGBA, y, width int) []float64 {
	c := make([]float64, width)
	for x := range c {
		c[x] = float64(img.RGBAAt(x, y).A) / 255
	}
	return c
}

// leadingEdge answers the sub-pixel column the covered run in cov begins at:
// the first fully covered column, less the coverage standing to the left of
// it. It is the reading CG4.8 made of the sidebar recess's ends, one row at a
// time.
func leadingEdge(cov []float64) (float64, bool) {
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

// trailingEdge is leadingEdge from the other end.
func trailingEdge(cov []float64) (float64, bool) {
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

// arcEdge is where a rounded corner of radius r puts its edge, averaged over
// the part of pixel row y that the corner reaches: the mean of the circle's
// own x over [y, y+1), against a corner whose extremes are x0 and y0. A row
// deeper than the radius sits on the straight side and answers x0.
//
// The mean and not the value at the row's middle: a corner's arc turns fastest
// in the row nearest its extreme, which is the row carrying most of the
// signal, and reading it at one height there is wrong by a tenth of a pixel.
func arcEdge(y, x0, y0, r float64) float64 {
	const steps = 400
	sum := 0.0
	for k := 0; k < steps; k++ {
		d := y + (float64(k)+0.5)/steps - y0
		if d >= r {
			sum += x0
			continue
		}
		sum += x0 + r - math.Sqrt(math.Max(0, r*r-(r-d)*(r-d)))
	}
	return sum / steps
}

// fitCorner fits a circular corner to a profile of per-row edges, the corner's
// extremes pinned at (x0, y0), and answers the radius and the fit's rms. The
// scan is the whole range a 16 dp glyph could carry, at a hundredth of a
// pixel.
func fitCorner(profile map[int]float64, x0, y0 float64) (r, rms float64) {
	rms = math.Inf(1)
	for n := 100; n <= 800; n++ {
		try := float64(n) / 100
		s := 0.0
		for y, e := range profile {
			d := e - arcEdge(float64(y), x0, y0, try)
			s += d * d
		}
		if got := math.Sqrt(s / float64(len(profile))); got < rms {
			r, rms = try, got
		}
	}
	return r, rms
}

// TestTheBoxDrawsTheMeasuredCornerAndEdge reads the checkbox's corner and its
// edge off a capture of the component, in both appearances.
//
// The corner: the per-row coverage of the glyph's leading edge, fitted by a
// circle with the box's own extremes pinned — CG4.8's fit to the sidebar
// recess's ends — must answer the 5 dp checkboxCornerRadius measures off the
// save dialog's switched-off boxes, where the same fit reads 5.04 light (rms
// 0.038 px) and 5.34 dark (rms 0.070 px).
//
// The edge: a run across the box's middle row, and one down its middle column,
// must cross exactly one pixel of the platform's field hairline before the
// box's own interior — the width and colour the save dialog's "Tags:" field
// measures, which is what a control with no capture of its own spends.
func TestTheBoxDrawsTheMeasuredCornerAndEdge(t *testing.T) {
	const size = 44
	row := int(tokens.Comfortable.CheckboxRowHeight)
	lo := (row - glyphSide) / 2
	mid := lo + glyphSide/2

	for _, sc := range switchedOffReadings {
		t.Run(sc.name, func(t *testing.T) {
			img := golden.Capture(t, image.Pt(size, size), input.RenderCheckbox(
				nil,
				sc.platform, tokens.Spacing, tokens.Radius,
				tokens.DefaultTypography.BodyLarge,
				input.CheckboxRenderState{Surface: sc.sheet},
			))
			if img == nil {
				return
			}

			profile := map[int]float64{}
			for y := lo; y < lo+cornerRows; y++ {
				e, ok := leadingEdge(coverageRow(img, y, size))
				if !ok {
					t.Fatalf("row %d of the box carries no fully covered column", y)
				}
				profile[y] = e
			}
			r, rms := fitCorner(profile, float64(lo), float64(lo))
			const want = 5.0
			if math.Abs(r-want) > 0.3 || rms > 0.1 {
				t.Errorf("the box's corner fits r = %.2f (rms %.3f px over %d rows), want the measured %.0f", r, rms, len(profile), want)
			}

			edge, fill := control.Border(sc.platform), control.Fill(sc.platform)
			for _, run := range []struct {
				what string
				at   func(i int) (x, y int)
			}{
				{"across its middle row, from the leading side", func(i int) (int, int) { return lo + i, mid }},
				{"across its middle row, from the trailing side", func(i int) (int, int) { return lo + glyphSide - 1 - i, mid }},
				{"down its middle column, from the top", func(i int) (int, int) { return mid, lo + i }},
				{"down its middle column, from the bottom", func(i int) (int, int) { return mid, lo + glyphSide - 1 - i }},
			} {
				n := 0
				for i := 0; i < glyphSide; i++ {
					x, y := run.at(i)
					if !nearlyEqual(img.RGBAAt(x, y), edge) {
						break
					}
					n++
				}
				if n != 1 {
					t.Errorf("a run %s crosses %d pixels of the field hairline %v, want the measured 1", run.what, n, edge)
				}
				x, y := run.at(1)
				if !nearlyEqual(img.RGBAAt(x, y), fill) {
					t.Errorf("a run %s reads %v one pixel in, want the box's own fill %v", run.what, img.RGBAAt(x, y), fill)
				}
			}
		})
	}
}

// TestTheDiscIsTheMeasuredCircle reads the radio's disc off a capture of the
// component, in both appearances.
//
// The disc is a circle, so the fit is a circle rather than a corner's arc: the
// leading and trailing sub-pixel edge of every row of the glyph, least squares
// against a centre and a radius. It must answer the 8 dp half of the measured
// 16 dp diameter, on the glyph's own centre, where the same fit reads r = 8.17
// (rms 0.084 px) light and 8.12 (rms 0.073 px) dark off System Settings'
// selected radio.
//
// The edge is the box's, read the same way: one pixel of the field hairline
// before the disc's own interior, on the row and the column through its
// centre, where a circle is flat enough to read a width off.
func TestTheDiscIsTheMeasuredCircle(t *testing.T) {
	const size = 44
	row := int(tokens.Comfortable.CheckboxRowHeight)
	lo := (row - glyphSide) / 2
	mid := lo + glyphSide/2

	for _, sc := range switchedOffReadings {
		t.Run(sc.name, func(t *testing.T) {
			img := golden.Capture(t, image.Pt(size, size), input.RenderRadio(
				nil,
				sc.platform, tokens.Spacing, tokens.Radius,
				tokens.DefaultTypography.BodyLarge,
				input.RadioRenderState{Surface: sc.sheet},
			))
			if img == nil {
				return
			}

			var px, py []float64
			for y := lo; y < lo+glyphSide; y++ {
				cov := coverageRow(img, y, size)
				l, ok := leadingEdge(cov)
				if !ok {
					continue
				}
				tr, _ := trailingEdge(cov)
				px = append(px, l, tr)
				py = append(py, float64(y)+0.5, float64(y)+0.5)
			}
			cx, cy, r, rms := fitCircle(px, py)

			const wantR = glyphSide / 2.0
			wantC := float64(lo) + wantR
			if math.Abs(r-wantR) > 0.3 || rms > 0.2 {
				t.Errorf("the disc fits r = %.2f (rms %.3f px over %d edges), want the measured %.0f", r, rms, len(px), wantR)
			}
			if math.Abs(cx-wantC) > 0.15 || math.Abs(cy-wantC) > 0.15 {
				t.Errorf("the disc centres on (%.2f, %.2f), want the glyph's own (%.1f, %.1f)", cx, cy, wantC, wantC)
			}

			edge := control.Border(sc.platform)
			for _, run := range []struct {
				what string
				at   func(i int) (x, y int)
			}{
				{"across its middle row, from the leading side", func(i int) (int, int) { return lo + i, mid }},
				{"down its middle column, from the top", func(i int) (int, int) { return mid, lo + i }},
			} {
				n := 0
				for i := 0; i < glyphSide; i++ {
					x, y := run.at(i)
					if !nearlyEqual(img.RGBAAt(x, y), edge) {
						break
					}
					n++
				}
				if n != 1 {
					t.Errorf("a run %s crosses %d pixels of the field hairline %v, want the measured 1", run.what, n, edge)
				}
			}
		})
	}
}

// fitCircle is the least-squares circle through a set of sub-pixel edge
// points, solved in the linear form the algebraic fit takes: the normal
// equations for (2cx, 2cy, r² − cx² − cy²) against x² + y², by Cramer's rule
// on the 3×3 system. It answers the centre, the radius and the fit's rms.
func fitCircle(px, py []float64) (cx, cy, r, rms float64) {
	n := float64(len(px))
	var sx, sy, sxx, syy, sxy, sxz, syz, sz float64
	for i := range px {
		x, y := px[i], py[i]
		z := x*x + y*y
		sx += x
		sy += y
		sxx += x * x
		syy += y * y
		sxy += x * y
		sxz += x * z
		syz += y * z
		sz += z
	}
	// The 3×3 system, rows over (2cx, 2cy, c):  [sxx sxy sx; sxy syy sy; sx sy n]
	a := [3][4]float64{
		{sxx, sxy, sx, sxz},
		{sxy, syy, sy, syz},
		{sx, sy, n, sz},
	}
	// Gaussian elimination with partial pivoting.
	for i := 0; i < 3; i++ {
		p := i
		for k := i + 1; k < 3; k++ {
			if math.Abs(a[k][i]) > math.Abs(a[p][i]) {
				p = k
			}
		}
		a[i], a[p] = a[p], a[i]
		for k := i + 1; k < 3; k++ {
			f := a[k][i] / a[i][i]
			for j := i; j < 4; j++ {
				a[k][j] -= f * a[i][j]
			}
		}
	}
	var s [3]float64
	for i := 2; i >= 0; i-- {
		v := a[i][3]
		for j := i + 1; j < 3; j++ {
			v -= a[i][j] * s[j]
		}
		s[i] = v / a[i][i]
	}
	cx, cy = s[0]/2, s[1]/2
	r = math.Sqrt(s[2] + cx*cx + cy*cy)
	for i := range px {
		d := math.Hypot(px[i]-cx, py[i]-cy) - r
		rms += d * d
	}
	return cx, cy, r, math.Sqrt(rms / n)
}

// unpremultiplied divides a captured pixel's coverage back out of its colour.
// golden.Capture renders onto a clear window, so a partly covered pixel holds
// the colour drawn scaled by the fraction of the pixel it reached; a reading
// taken anywhere but on a flat side has to divide that fraction out before it
// can be compared with the colour a component was asked to draw.
func unpremultiplied(c stdcolor.RGBA) stdcolor.RGBA {
	if c.A == 0 {
		return stdcolor.RGBA{}
	}
	up := func(v uint8) uint8 {
		n := (int(v)*255 + int(c.A)/2) / int(c.A)
		if n > 255 {
			n = 255
		}
		return uint8(n)
	}
	return stdcolor.RGBA{R: up(c.R), G: up(c.G), B: up(c.B), A: 0xff}
}

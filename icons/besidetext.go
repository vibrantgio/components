package icons

import (
	"math"

	"gioui.org/layout"
)

// BesideText is the platform's measure of a symbol that stands beside text,
// as fractions of the height of that text's capitals. Height is the height
// that the symbol covers. Centre is the distance from the middle of the cap
// band to the middle of what the symbol covers, positive downward.
type BesideText struct {
	Height, Centre float32
}

// besideTextRow is one row of the reference, in points: the text size, the
// measured cap height at that size, the height that the symbol covers, and
// how far the symbol reaches above the cap line and below the baseline.
type besideTextRow struct {
	pt, cap, height, above, below float32
}

// MEASURED, sf-symbols-vs-cap-band.txt in the macOS reference (macOS 26.5.2):
// each SF Symbol beside SF Pro text at the same point size, regular weight,
// read off an 8x supersampled offscreen bitmap. The symbols differ: the
// plus covers 1.14 cap heights and the looking glass 1.39.
//
// The reference also measures the bare cross. The set has no bare cross:
// XmarkCircle is a disc with a cross knocked out of it, a different symbol, so
// it takes the keyline's size.
var besideTextRows = map[Name][]besideTextRow{
	Plus: {
		{11, 7.875, 9.000, 0.625, 0.500},
		{13, 9.250, 10.500, 0.750, 0.500},
		{15, 10.625, 12.125, 0.500, 1.000},
	},
	Check: {
		{11, 7.875, 9.500, 0.625, 1.000},
		{13, 9.250, 11.000, 0.750, 1.000},
		{15, 10.625, 12.500, 0.875, 1.000},
	},
	MagnifyingGlass: {
		{11, 7.875, 11.000, 1.625, 1.500},
		{13, 9.250, 12.875, 2.125, 1.500},
		{15, 10.625, 14.500, 1.875, 2.000},
	},
}

// MeasureBesideText returns the platform's measure of the named symbol beside
// text of sizePt points. ok is false for a mark that the reference does not
// hold.
//
// DERIVED: at a size that the reference does not hold, the fractions of the
// nearest measured size apply, so the symbol scales with the text's cap
// height. A size halfway between two measured sizes takes the smaller.
func MeasureBesideText(name Name, sizePt float32) (m BesideText, ok bool) {
	rows := besideTextRows[name]
	if len(rows) == 0 {
		return BesideText{}, false
	}
	best := rows[0]
	for _, r := range rows[1:] {
		if math.Abs(float64(r.pt-sizePt)) < math.Abs(float64(best.pt-sizePt)) {
			best = r
		}
	}
	return BesideText{
		Height: best.height / best.cap,
		Centre: (best.below - best.above) / 2 / best.cap,
	}, true
}

// The vertical extent of each measured mark's drawing, in grid units. The
// files state them as their contracts: the plus's upright runs the keyline
// from unit 2.5 to 21.5, the check runs from its round cap's top at unit 5.8
// to its turn at 17.99, and the looking glass runs MagnifyingGlassDrawingSize.
// Each extent stands centred in the square to a tenth of a unit.
var drawnExtentUnits = map[Name]float64{
	Plus:            19.0,
	Check:           12.19,
	MagnifyingGlass: MagnifyingGlassDrawingSize * gridUnits,
}

// DrawnHeightPx is the height in device pixels that the named mark covers in
// a square of sizePx in this window. It is the drawing's extent on the grid,
// scaled, and the band's whole widening, half of it past each edge. ok is
// false for a mark whose extent the set does not state.
func DrawnHeightPx(gtx layout.Context, name Name, sizePx int) (float32, bool) {
	units, ok := drawnExtentUnits[name]
	if !ok {
		return 0, false
	}
	w := float64(widening(gtx.Metric.PxPerDp, sizePx)) / 1000
	return float32(units*float64(sizePx)/gridUnits + w), true
}

// WideningPx is how much wider than the grid's band a banded mark's band is
// drawn in a square of sizePx in this window, in device pixels. Half of it
// reaches past each edge of the drawing.
func WideningPx(gtx layout.Context, sizePx int) float32 {
	return float32(widening(gtx.Metric.PxPerDp, sizePx)) / 1000
}

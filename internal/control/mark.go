package control

import (
	"image"
	"image/color"
	"math"

	"gioui.org/f32"
	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
)

// The pop-up mark: two chevrons stacked point to point, the upper pointing up
// and the lower down. It is the one mark the platform puts on a control that
// holds one of several values, and both of this library's pop-up triggers draw
// it — the one on a form and the one in a toolbar.
//
// MEASURED, save-dialog-{light,dark}.png at 1x, both appearances agreeing to
// the pixel: the "File Format:" pop-up runs y 336–359, 24 px tall, and its
// pair spans x 435–442 and y 343–353 — eight columns wide, the upper chevron
// y 343–347 and the lower y 349–353, five rows each, with one clear row
// between them.
//
// The size is FIXED and not a ratio of the control it stands in. The Finder
// toolbar draws the same glyph at the same eight by eleven in a control 36 px
// tall (finder-window-light.png, x 726–733, upper y 21–25, lower y 27–31),
// where the dialog draws it in one 24 px tall: the platform sizes this mark by
// its point size, so a taller control gets the same mark with more room around
// it.
const (
	// MarkWDp is the pair's column and one chevron's full width.
	MarkWDp unit.Dp = 8
	// MarkChevronHDp is one chevron's covered height.
	MarkChevronHDp unit.Dp = 5
	// MarkGapDp is the clear row between the two chevrons.
	MarkGapDp unit.Dp = 1
	// MarkHDp is the pair's covered height, the two chevrons and their gap.
	MarkHDp = 2*MarkChevronHDp + MarkGapDp
)

// MarkStrokeDp is the arm's line weight, perpendicular to the arm.
//
// MEASURED off the same capture: an arm crossing a row covers about 1.85
// columns — the light capture's row y=345 reads 0.02 + 0.65 + 0.99 of a column
// and y=346 reads 0.64 + 0.99 + 0.40 — and the arm runs at 51.3 degrees off
// the horizontal, which is what an eight by five chevron's centreline runs at,
// so perpendicular it is 1.85 × sin 51.3° ≈ 1.44 px. The whole upper chevron
// covers 12.7 px² of the capture, which over a centreline 2 × 4.7 px long is
// 1.36 px of width. 1.5 is the nearest weight this system draws.
//
// It is spent in PIXELS and not through gtx.Dp: a dp rounds to a whole pixel,
// and 1.5 dp rounded at one pixel per dp is a two-pixel arm — a third heavier
// than the platform's, which is what turns two thin strokes into a wedge.
const MarkStrokeDp = 1.5

// DrawMark paints the pop-up mark inside box: the pair spanning box
// horizontally and centred in it vertically, where the centring lands between
// two rows taking the lower one.
//
// MEASURED, the same capture: the pair covers y 343–353 in a control of
// y 336–359 — eleven rows in twenty-four, seven above them and six below,
// which is the exact centre of 6.5 rounded up.
//
// It is STATIC. The platform's pop-up mark says "this control holds one of
// several values" and never "the menu is open" or "it opens upwards"; a pair
// pointing both ways cannot say a direction, which is the whole reason the
// platform draws a pair here and a single chevron on a pull-down.
func DrawMark(gtx layout.Context, box image.Rectangle, col color.NRGBA) {
	w := float32(box.Dx())
	h := float32(gtx.Dp(MarkChevronHDp))
	clear := float32(gtx.Dp(MarkGapDp))
	stroke := MarkStrokeDp * gtx.Metric.PxPerDp
	if stroke < 1 {
		stroke = 1
	}

	top := float32(box.Min.Y) + float32(math.Ceil(float64(float32(box.Dy())-(2*h+clear))/2))
	x0 := float32(box.Min.X)

	var p clip.Path
	p.Begin(gtx.Ops)
	chevron(&p, x0, top, w, h, stroke, true)
	chevron(&p, x0, top+h+clear, w, h, stroke, false)
	paint.FillShape(gtx.Ops, col, clip.Outline{Path: p.End()}.Op())
}

// chevron appends one chevron to p as a FILLED outline rather than as a
// stroked polyline: its apex is a miter and its arms end in a cut, which is
// the profile the platform draws. A stroked path is capped and joined round in
// this rasterizer, and a round join at an apex this small is a blob three rows
// deep where the platform's first row carries a third of a pixel of paint.
//
// The arms' centreline runs corner to corner of the covered box — a chevron
// eight wide and five tall puts its arms at 51.3 degrees — and the outline is
// that centreline offset by half the stroke to each side, mitered at the apex
// and cut horizontally at the ends. Solving the three extremes of the offset
// outline against the covered box gives d below, which is the horizontal room
// the half-stroke takes.
func chevron(p *clip.Path, x, y, w, h, stroke float32, up bool) {
	k := 2 * h / w
	d := (stroke / 2) * float32(math.Sqrt(float64(1+k*k))) / k
	apex, base := y, y+h
	if !up {
		apex, base = y+h, y
	}
	p.MoveTo(f32.Pt(x, base))
	p.LineTo(f32.Pt(x+w/2, apex))
	p.LineTo(f32.Pt(x+w, base))
	p.LineTo(f32.Pt(x+w-2*d, base))
	if up {
		p.LineTo(f32.Pt(x+w/2, apex+2*d*k))
	} else {
		p.LineTo(f32.Pt(x+w/2, apex-2*d*k))
	}
	p.LineTo(f32.Pt(x+2*d, base))
	p.Close()
}

// The patch a bordered toolbar control fills while it records a yes, inset
// inside the control's own box.
//
// MEASURED, finder-window-untinted-dark.png and -light.png, the chosen
// segment of Finder's four-segment view control: the patch is 32 by 26 in a
// segment 37 wide and a control 36 tall, so it stands five rows clear of the
// control's own edge above and below and two and a half columns clear at
// either end. Both appearances draw the same 32 by 26. The colour it is
// filled in is tokens.PlatformColors.ToolbarCheckedOverlay over the fill the
// control carries.
const (
	// ToolbarCheckedInsetYDp is what the patch leaves clear above and below.
	ToolbarCheckedInsetYDp unit.Dp = 5

	// ToolbarCheckedInsetXDp is what it leaves clear at either end.
	ToolbarCheckedInsetXDp unit.Dp = 2.5
)

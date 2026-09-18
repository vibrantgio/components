package control

import (
	"image"
	"image/color"
	"math"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"

	"github.com/vibrantgio/components/icons"
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

// MarkStrokeDp is the arm's line weight, perpendicular to the arm: the icon
// set's one measured band, which chevron-pair.svg draws and the design
// bundle's masked SVG repeats.
//
// MEASURED off the same capture: an arm crossing a row covers about 1.85
// columns — the light capture's row y=345 reads 0.02 + 0.65 + 0.99 of a column
// and y=346 reads 0.64 + 0.99 + 0.40 — and the arm runs at 51.3 degrees off
// the horizontal, which is what an eight by five chevron's centreline runs at,
// so perpendicular it is 1.85 × sin 51.3° ≈ 1.44 px. The whole upper chevron
// covers 12.7 px² of the capture, which over a centreline 2 × 4.7 px long is
// 1.36 px of width. 1.4 stands between the two and is the one band the whole
// set draws.
//
// It is spent in PIXELS and not through gtx.Dp: a dp rounds to a whole pixel,
// and 1.4 dp rounded at one pixel per dp is a one-pixel arm at the small end
// and a two-pixel one at the large, which is what turns two thin strokes into
// a wedge.
const MarkStrokeDp = 1.4

// The square the set draws a control's mark inside, and where the pair stands
// in it. The grid components/icons is authored on is 24 units, so a mark drawn
// at 24 dp puts one unit on one device pixel and the pair comes out at the
// measured eight by eleven; chevron-pair.svg stands it at x 8 to 16 and y 6 to
// 17 of that square, a half unit high of centre because eleven is odd against
// a 24-unit box.
const (
	markBoxDp  unit.Dp = 24
	markLeadDp unit.Dp = 8
	markTopDp  unit.Dp = 6
)

// DrawMark paints the pop-up mark inside box: the pair spanning box
// horizontally and centred in it vertically, where the centring lands between
// two rows taking the lower one.
//
// MEASURED, the same capture: the pair covers y 343–353 in a control of
// y 336–359 — eleven rows in twenty-four, seven above them and six below,
// which is the exact centre of 6.5 rounded up.
//
// The figure itself is the icon set's chevron-pair mark, which carries the
// geometry, the profile and the readings both this control and the design
// bundle are drawn from. The set is where a mark lives; this function is only
// where one is placed.
//
// It is STATIC. The platform's pop-up mark says "this control holds one of
// several values" and never "the menu is open" or "it opens upwards"; a pair
// pointing both ways cannot say a direction, which is the whole reason the
// platform draws a pair here and a single chevron on a pull-down.
func DrawMark(gtx layout.Context, box image.Rectangle, col color.NRGBA) {
	mark := icons.Mark(icons.ChevronPair)
	if mark == nil {
		return
	}
	h := gtx.Dp(MarkHDp)
	top := box.Min.Y + int(math.Ceil(float64(box.Dy()-h)/2))
	off := op.Offset(image.Pt(box.Min.X-gtx.Dp(markLeadDp), top-gtx.Dp(markTopDp))).Push(gtx.Ops)
	mark(gtx, gtx.Dp(markBoxDp), col)
	off.Pop()
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

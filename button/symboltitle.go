package button

import (
	"image"
	"image/color"
	"math"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/text"
	"gioui.org/unit"

	"github.com/vibrantgio/components/icons"
	"github.com/vibrantgio/components/internal/focus"
	"github.com/vibrantgio/theme/tokens"
)

// A symbol that stands before a button's text.
//
// MEASURED, voicememos-collection-search-light.png, the find button: a
// 13 × 13 pt magnifier stands before the 13 pt title "Search". Four clear
// columns stand between the symbol's last covered column and the title. The
// symbol's centre row is the button's centre row. The library's find button
// draws that magnifier in an 18 pt square (findButtonMarkDp). The square
// here is that 18 pt for the 13 pt System font role.
//
// The symbols differ in size beside text (icons.MeasureBesideText). The
// looking glass in its 18 pt square beside 13 pt text is the anchor. A
// measured symbol takes the square in which it covers the anchor's drawn
// height times the ratio of the two symbols' platform heights, and it
// stands off the anchor's centre row by the difference of the two centres.
// So the looking glass beside 13 pt text keeps the find button's square.
//
// DERIVED: a symbol that the reference does not measure takes the keyline's
// square, which scales with the title's size, so the find bar's 11 pt title
// takes a 15 pt square. No stored capture holds such a symbol beside a title.
//
// The set draws every square mark to one keyline, 2.5 units clear of each
// edge of its 24-unit grid (see components/icons, the sidebar-left pane). The
// gap is measured from that keyline, so the clear columns between a square
// mark and the text are the measured four.
const (
	symbolPerTitle   = 18.0 / 13.0
	symbolKeyline    = 2.5 / 24.0
	symbolTitleGapDp = unit.Dp(4)
	// symbolAnchorPt is the title size at which the find button's capture
	// measures the looking glass.
	symbolAnchorPt = 13
)

// leadSymbol is where a symbol stands before a title in device pixels: the
// side of the square it is drawn in, the square's leading edge to the
// keyline, the keyline's width, the gap from the keyline to the title, the
// square's offset down from the control's centre row, and the height the
// title group needs for the symbol.
type leadSymbol struct {
	side, lead, span, gap, dy, height int
}

// symbolBefore returns the place of the named symbol before a title set in
// style. An empty name, or a mark that the reference does not measure, takes
// the keyline's square.
func symbolBefore(gtx layout.Context, name icons.Name, style tokens.TextStyle) leadSymbol {
	side := gtx.Sp(unit.Sp(style.Size * symbolPerTitle))
	lead := int(float32(side)*symbolKeyline + 0.5)
	// end is the first column wholly clear of the keyline, so the gap counts
	// clear columns as the reference counts them.
	end := int(math.Ceil(float64(side) * (1 - symbolKeyline)))
	sym := leadSymbol{side: side, lead: lead, span: end - lead, gap: gtx.Dp(symbolTitleGapDp), height: side}
	if m, ok := icons.MeasureBesideText(name, style.Size); ok {
		sym.measured(gtx, name, m, style)
	}
	return sym
}

// measured sizes and places the square of a symbol that the reference
// measures beside text set in style.
func (sym *leadSymbol) measured(gtx layout.Context, name icons.Name, m icons.BesideText, style tokens.TextStyle) {
	anchor, _ := icons.MeasureBesideText(icons.MagnifyingGlass, symbolAnchorPt)
	here, _ := icons.MeasureBesideText(icons.MagnifyingGlass, style.Size)
	anchorSide := gtx.Sp(unit.Sp(symbolAnchorPt * symbolPerTitle))
	anchorHeight, _ := icons.DrawnHeightPx(gtx, icons.MagnifyingGlass, anchorSide)
	// A face's cap height scales with its size, so the ratio of two cap
	// heights is the ratio of the two sizes. The ratio is exactly 1 for the
	// looking glass beside 13 pt text.
	want := anchorHeight * ((m.Height * style.Size) / (anchor.Height * symbolAnchorPt))

	// The drawn height grows with the square, so the first square that
	// reaches want, or the one before it, is the nearest.
	side := 1
	drawn, ok := icons.DrawnHeightPx(gtx, name, side)
	if !ok || !(want > 0) {
		return
	}
	for drawn < want {
		side++
		drawn, _ = icons.DrawnHeightPx(gtx, name, side)
	}
	if prev, _ := icons.DrawnHeightPx(gtx, name, side-1); side > 1 && want-prev < drawn-want {
		side, drawn = side-1, prev
	}

	capPx := style.FaceMetrics().CapHeight * gtx.Metric.PxPerSp
	dy := int(math.Round(float64((m.Centre - here.Centre) * capPx)))

	sym.side = side
	sym.lead = int(float32(side)*symbolKeyline + 0.5)
	// Half the band's widening reaches past the keyline.
	w := icons.WideningPx(gtx, side)
	end := int(math.Ceil(float64(side)*(1-symbolKeyline) + float64(w)/2))
	sym.span = end - sym.lead
	sym.dy = dy
	sym.height = int(math.Ceil(float64(drawn))) + 2*abs(dy)
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

// titleGroup is a button's title as one drawn part: the text, and the symbol
// before it when the button has one. size is the part's size from the
// symbol's keyline to the text's end, and the line box's height or the
// height the symbol needs, whichever is taller.
type titleGroup struct {
	text     op.CallOp
	textSize image.Point
	icon     func(gtx layout.Context, sizePx int, col color.NRGBA)
	sym      leadSymbol
	fg       color.NRGBA
	size     image.Point
}

// newTitleGroup wraps a recorded text of textSize. A nil icon gives the text
// alone, and the group is then exactly the text's size. name is the icon's
// name in the set, or empty for a painter outside it.
func newTitleGroup(gtx layout.Context, call op.CallOp, textSize image.Point, icon func(gtx layout.Context, sizePx int, col color.NRGBA), name icons.Name, style tokens.TextStyle, fg color.NRGBA) titleGroup {
	g := titleGroup{text: call, textSize: textSize, icon: icon, fg: fg, size: textSize}
	if icon != nil {
		g.sym = symbolBefore(gtx, name, style)
		g.size.X += g.sym.span + g.sym.gap
		g.size.Y = max(g.size.Y, g.sym.height)
	}
	return g
}

// add draws the group at x in a control h pixels tall: the text centred on
// the control's centre row, and the symbol's square centred on it too.
func (g titleGroup) add(gtx layout.Context, x, h int) {
	textX := x
	if g.icon != nil {
		textX = x + g.sym.span + g.sym.gap
		off := op.Offset(image.Pt(x-g.sym.lead, (h-g.sym.side)/2+g.sym.dy)).Push(gtx.Ops)
		g.icon(gtx, g.sym.side, g.fg)
		off.Pop()
	}
	off := op.Offset(image.Pt(textX, (h-g.textSize.Y)/2)).Push(gtx.Ops)
	g.text.Add(gtx.Ops)
	off.Pop()
}

// RenderWithSymbol produces a layout.Widget for a button whose title is a
// symbol and text, in an explicit visual state, without event processing or
// rx machinery. The symbol stands before the text, in the button's
// foreground. The parameters are [Render]'s, and symbol names the symbol in
// the icon set, as [Props.Symbol] does. An empty symbol draws exactly what
// [Render] draws. Intended for golden-image testing and static
// demonstrations; production code uses Button with Props.Title and
// Props.Symbol.
func RenderWithSymbol(
	shaper *text.Shaper,
	title string,
	symbol icons.Name,
	p tokens.PlatformColors,
	sp tokens.SpacingScale,
	rad tokens.RadiusScale,
	titleStyle tokens.TextStyle,
	d tokens.Density,
	s RenderState,
) layout.Widget {
	tok := resolvedTokens{platform: p, spacing: sp, radius: rad, title: titleStyle, density: d}
	var icon func(gtx layout.Context, sizePx int, col color.NRGBA)
	if symbol != "" {
		icon = icons.Mark(symbol)
	}
	return func(gtx layout.Context) layout.Dimensions {
		return focus.Around(gtx, func(gtx layout.Context) layout.Dimensions {
			return drawButton(gtx, shaper, title, icon, symbol, tok, s)
		})
	}
}

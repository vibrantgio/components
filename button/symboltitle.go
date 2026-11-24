package button

import (
	"image"
	"image/color"
	"math"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/text"
	"gioui.org/unit"

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
// DERIVED: the square scales with the title's size, so the find bar's 11 pt
// title takes a 15 pt square. No stored capture holds a symbol and a title
// at another size.
//
// The set's square marks cover more than the platform's symbols at this
// size: the plus covers 14 px in the 18 pt square, and the platform's plus
// beside 13 pt text covers 10.5 pt (sf-symbols-vs-cap-band.txt). The set
// draws one keyline for every mark, and the magnifier is the mark measured
// beside a title.
//
// The set draws every square mark to one keyline, 2.5 units clear of each
// edge of its 24-unit grid (see components/icons, the sidebar-left pane). The
// gap is measured from that keyline, so the clear columns between a square
// mark and the text are the measured four.
const (
	symbolPerTitle   = 18.0 / 13.0
	symbolKeyline    = 2.5 / 24.0
	symbolTitleGapDp = unit.Dp(4)
)

// leadSymbol is where a symbol stands before a title in device pixels: the
// side of the square it is drawn in, the square's leading edge to the
// keyline, the keyline's width, and the gap from the keyline to the title.
type leadSymbol struct {
	side, lead, span, gap int
}

// symbolBefore returns the place of a symbol before a title set in style.
func symbolBefore(gtx layout.Context, style tokens.TextStyle) leadSymbol {
	side := gtx.Sp(unit.Sp(style.Size * symbolPerTitle))
	lead := int(float32(side)*symbolKeyline + 0.5)
	// end is the first column wholly clear of the keyline, so the gap counts
	// clear columns as the reference counts them.
	end := int(math.Ceil(float64(side) * (1 - symbolKeyline)))
	return leadSymbol{side: side, lead: lead, span: end - lead, gap: gtx.Dp(symbolTitleGapDp)}
}

// titleGroup is a button's title as one drawn part: the text, and the symbol
// before it when the button has one. size is the part's size from the
// symbol's keyline to the text's end, and the line box's height or the
// symbol's square's height, whichever is taller.
type titleGroup struct {
	text     op.CallOp
	textSize image.Point
	icon     func(gtx layout.Context, sizePx int, col color.NRGBA)
	sym      leadSymbol
	fg       color.NRGBA
	size     image.Point
}

// newTitleGroup wraps a recorded text of textSize. A nil icon gives the text
// alone, and the group is then exactly the text's size.
func newTitleGroup(gtx layout.Context, call op.CallOp, textSize image.Point, icon func(gtx layout.Context, sizePx int, col color.NRGBA), style tokens.TextStyle, fg color.NRGBA) titleGroup {
	g := titleGroup{text: call, textSize: textSize, icon: icon, fg: fg, size: textSize}
	if icon != nil {
		g.sym = symbolBefore(gtx, style)
		g.size.X += g.sym.span + g.sym.gap
		g.size.Y = max(g.size.Y, g.sym.side)
	}
	return g
}

// add draws the group at x in a control h pixels tall: the text centred on
// the control's centre row, and the symbol's square centred on it too.
func (g titleGroup) add(gtx layout.Context, x, h int) {
	textX := x
	if g.icon != nil {
		textX = x + g.sym.span + g.sym.gap
		off := op.Offset(image.Pt(x-g.sym.lead, (h-g.sym.side)/2)).Push(gtx.Ops)
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
// foreground. The parameters are [Render]'s, and icon is the symbol's
// painter, as [Props.Icon] is. A nil icon draws exactly what [Render] draws.
// Intended for golden-image testing and static demonstrations; production
// code uses Button with Props.Title and Props.Icon.
func RenderWithSymbol(
	shaper *text.Shaper,
	title string,
	icon func(gtx layout.Context, sizePx int, col color.NRGBA),
	p tokens.PlatformColors,
	sp tokens.SpacingScale,
	rad tokens.RadiusScale,
	titleStyle tokens.TextStyle,
	d tokens.Density,
	s RenderState,
) layout.Widget {
	tok := resolvedTokens{platform: p, spacing: sp, radius: rad, title: titleStyle, density: d}
	return func(gtx layout.Context) layout.Dimensions {
		return focus.Around(gtx, func(gtx layout.Context) layout.Dimensions {
			return drawButton(gtx, shaper, title, icon, tok, s)
		})
	}
}

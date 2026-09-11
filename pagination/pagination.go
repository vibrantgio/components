// Package pagination provides the pagination control: moving between
// numbered pages of content. It draws a horizontal row of numbered page
// buttons flanked by prev/next chevrons. One cell carries a fill — the page
// the reader is on, in the platform's accent with the text that reads on an
// accent fill — and every other page is a digit standing on whatever the row
// stands on, so the row says "this one" in one place and nowhere else.
//
// Page cells are drawn natively rather than through components/button:
// inside a density-sized ControlHeight square, components/button's
// Comfortable padding would truncate the page digit to a sliver. Drawing the
// cell here — radius.Md corners, centred digit — keeps the visuals aligned
// with components/button while letting every metric follow the density;
// future components/button styling changes must be mirrored here by hand.
//
// Pagination is a callable Go function consuming a components theme
// observable, returning a stream of layout.Widget. Source is intentionally
// short and free of opaque configuration — copy it into your own app and
// modify as needed.
//
// No virtualisation, no ellipsis collapse — every page in [1, PageCount]
// renders.
package pagination

import (
	"image"
	"image/color"
	"strconv"

	"gioui.org/f32"
	"gioui.org/font"
	"gioui.org/io/pointer"
	"gioui.org/io/semantic"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"

	"github.com/reactivego/rx"
	"github.com/vibrantgio/components/icon"
	"github.com/vibrantgio/components/internal/surface"
	complayout "github.com/vibrantgio/components/layout"
	vgcolor "github.com/vibrantgio/theme/color"
	"github.com/vibrantgio/theme/theme"
	"github.com/vibrantgio/theme/tokens"
	"github.com/vibrantgio/theme/typeset"
)

// Props configures a Pagination instance. Page is 1-indexed; values outside
// [1, PageCount] still render but disable both chevrons and no page is
// highlighted as current. PageCount < 1 renders to zero-sized Dimensions.
type Props struct {
	Page      int
	PageCount int
	OnSelect  func(gtx layout.Context, page int)

	// Shaper is an explicit per-instance override of the text shaper. Leave
	// it nil in normal use: the pagination then shapes its page digits with
	// the theme's shaper (Typography.Shaper()), which is built once for the
	// process and shared by every component reading that typography — the
	// cache lives behind the Typography value, so it survives the copy this
	// component's map function makes of it. Set it only when this instance
	// must shape with a different shaper than the theme provides.
	//
	// A shaper is not safe to use from two goroutines; Gio lays every
	// layout.Widget out on the one goroutine that runs the event loop,
	// which is what makes sharing it correct. See theme/tokens.Typography.Shaper.
	Shaper *text.Shaper

	// Surface is the opaque fill the row stands on. Only the current page
	// carries a fill of its own, so every other digit and both chevrons are
	// drawn straight on this, and the platform's label and disabled control
	// text carry a coverage rather than a colour. The zero value — no
	// colour — is the window's own plane.
	Surface color.NRGBA
}

// Pagination returns an rx.Observable[layout.Widget] that emits a new one
// whenever any consumed theme token changes. Click handlers fire
// for the chevrons (when not at the corresponding edge) and for each
// numbered page button; in all cases OnSelect receives the resulting page
// number (1-indexed).
func Pagination(th rx.Observable[theme.Theme], props Props) rx.Observable[layout.Widget] {
	// Flatten the nested theme observables into a concrete snapshot. The
	// typography emission supplies both the LabelLarge text style and the
	// theme's cached shaper: the theme owns the typeface.
	resolved := rx.SwitchMap(th, func(t theme.Theme) rx.Observable[resolvedTokens] {
		return rx.Map(
			rx.CombineLatest5(t.Platform, t.Spacing, t.Radius, t.Typography, t.Density),
			func(n rx.Tuple5[tokens.PlatformColors, tokens.SpacingScale, tokens.RadiusScale, tokens.Typography, tokens.Density]) resolvedTokens {
				typ := n.Fourth
				return resolvedTokens{
					platform: n.First,
					spacing:  n.Second,
					radius:   n.Third,
					label:    typ.LabelLarge,
					density:  n.Fifth,
					shaper:   typ.Shaper(),
				}
			},
		)
	})

	return rx.Defer(func() rx.Observable[layout.Widget] {
		var prevClick, nextClick widget.Clickable
		n := props.PageCount
		if n < 0 {
			n = 0
		}
		pageClicks := make([]widget.Clickable, n)

		return rx.Map(resolved, func(tok resolvedTokens) layout.Widget {
			shaper := props.Shaper
			if shaper == nil {
				shaper = tok.shaper
			}
			return func(gtx layout.Context) layout.Dimensions {
				if props.OnSelect != nil {
					if props.Page > 1 && prevClick.Clicked(gtx) {
						props.OnSelect(gtx, props.Page-1)
					}
					if props.Page < props.PageCount && nextClick.Clicked(gtx) {
						props.OnSelect(gtx, props.Page+1)
					}
					for i := range pageClicks {
						if pageClicks[i].Clicked(gtx) {
							props.OnSelect(gtx, i+1)
						}
					}
				}
				return drawPagination(gtx, shaper, props, &prevClick, &nextClick, pageClicks, tok)
			}
		})
	})
}

// Render produces a layout.Widget for a pagination row with pre-resolved
// tokens. Intended for golden-image testing and static demonstrations;
// production code should use Pagination, which reads both of the
// parameters below off the theme.
//
// label is the LabelLarge role's whole text style — typeface, weight,
// size and line height all reach the shaper — and d is the density the
// row draws at (every cell is a Density.ControlHeight square, and the
// chevron glyph scales with it). Pass
// tokens.DefaultTypography.LabelLarge and tokens.Comfortable for the
// default desktop look.
func Render(
	shaper *text.Shaper,
	props Props,
	p tokens.PlatformColors,
	sp tokens.SpacingScale,
	rad tokens.RadiusScale,
	label tokens.TextStyle,
	d tokens.Density,
) layout.Widget {
	tok := resolvedTokens{platform: p, spacing: sp, radius: rad, label: label, density: d}
	return func(gtx layout.Context) layout.Dimensions {
		return drawPagination(gtx, shaper, props, nil, nil, nil, tok)
	}
}

type resolvedTokens struct {
	platform tokens.PlatformColors
	spacing  tokens.SpacingScale
	radius   tokens.RadiusScale
	label    tokens.TextStyle // the LabelLarge role: typeface, weight, size, line height
	density  tokens.Density   // cell square and chevron glyph source
	shaper   *text.Shaper     // the theme's shaper; nil in the Render path
}

// Cell metrics: every pagination control is a Density.ControlHeight square
// (36 dp Comfortable, 28 dp Compact); the digit centres in the square like
// an icon-button glyph. The chevron glyph takes the icon rule, icon.Size(d)
// = ControlHeight − 2·PaddingY (20/16 dp), matching components icon
// buttons. Cells are adjacent controls separated by S2 gaps, so their hit
// area stays the cell bounds.

func drawPagination(
	gtx layout.Context,
	shaper *text.Shaper,
	props Props,
	prevClick, nextClick *widget.Clickable,
	pageClicks []widget.Clickable,
	tok resolvedTokens,
) layout.Dimensions {
	if props.PageCount < 1 {
		return layout.Dimensions{}
	}

	gap := layout.Rigid(complayout.HSpacer(tok.spacing.S2))
	children := make([]layout.FlexChild, 0, 2*props.PageCount+5)
	standsOn := surface.Or(props.Surface, tok.platform.WindowBackground)
	children = append(children, layout.Rigid(chevronCellWidget(false, prevClick, props.Page > 1, tok, standsOn)))
	children = append(children, gap)
	for i := 1; i <= props.PageCount; i++ {
		click := clickFor(pageClicks, i-1)
		children = append(children, layout.Rigid(pageCellWidget(shaper, i, i == props.Page, click != nil && props.OnSelect != nil, click, tok, standsOn)))
		if i < props.PageCount {
			children = append(children, gap)
		}
	}
	children = append(children, gap)
	children = append(children, layout.Rigid(chevronCellWidget(true, nextClick, props.Page < props.PageCount, tok, standsOn)))
	return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx, children...)
}

func clickFor(clicks []widget.Clickable, i int) *widget.Clickable {
	if i < 0 || i >= len(clicks) {
		return nil
	}
	return &clicks[i]
}

// pageCellColors answers what one page cell is drawn in. The page the reader
// is on is the accent fill with the text that reads on an accent fill; a page
// the control can take the reader to is a link; any other page number is a
// label. A zero fill is no fill at all: only the current page carries one, so
// the rest of the row stands on standsOn, which is what each foreground that
// carries a coverage is flattened onto.
func pageCellColors(current, navigable bool, p tokens.PlatformColors, standsOn color.NRGBA) (fill, fg color.NRGBA) {
	switch {
	case current:
		return p.ControlAccent, vgcolor.Flatten(p.AlternateSelectedControlText, p.ControlAccent)
	case navigable:
		return color.NRGBA{}, vgcolor.Flatten(p.Link, standsOn)
	default:
		return color.NRGBA{}, vgcolor.Flatten(p.Label, standsOn)
	}
}

// pageCellWidget returns a clickable ControlHeight-square cell rendering
// page n natively. navigable says a click on this cell goes to page n, which
// is what makes the digit a link rather than a label.
func pageCellWidget(shaper *text.Shaper, n int, current, navigable bool, click *widget.Clickable, tok resolvedTokens, standsOn color.NRGBA) layout.Widget {
	bg, fg := pageCellColors(current, navigable, tok.platform, standsOn)
	label := strconv.Itoa(n)

	return func(gtx layout.Context) layout.Dimensions {
		side := gtx.Dp(unit.Dp(tok.density.ControlHeight))
		cgtx := gtx
		cgtx.Constraints = layout.Exact(image.Pt(side, side))
		draw := func(gtx layout.Context) layout.Dimensions {
			return drawPageCell(gtx, shaper, label, bg, fg, tok, side)
		}
		if click == nil {
			return draw(cgtx)
		}
		return click.Layout(cgtx, func(gtx layout.Context) layout.Dimensions {
			semantic.LabelOp(label).Add(gtx.Ops)
			semantic.EnabledOp(true).Add(gtx.Ops)
			pointer.CursorPointer.Add(gtx.Ops)
			return draw(gtx)
		})
	}
}

// drawPageCell paints one page-number cell: a side×side rounded square
// (radius.Md, components/button's corner) filled with bg where there is one,
// the digit shaped in the LabelLarge role and centred. The digit is never
// truncated — the square is the control, the digit its glyph, mirroring the
// icon-button rule rather than the text-button padding rule.
func drawPageCell(gtx layout.Context, shaper *text.Shaper, label string, bg, fg color.NRGBA, tok resolvedTokens, side int) layout.Dimensions {
	if bg.A != 0 {
		rad := gtx.Dp(unit.Dp(tok.radius.Md))
		rrect := clip.RRect{Rect: image.Rectangle{Max: image.Pt(side, side)}, SE: rad, SW: rad, NE: rad, NW: rad}
		paint.FillShape(gtx.Ops, bg, rrect.Op(gtx.Ops))
	}

	mColor := op.Record(gtx.Ops)
	paint.ColorOp{Color: fg}.Add(gtx.Ops)
	material := mColor.Stop()

	labelGtx := gtx
	labelGtx.Constraints.Min = image.Point{}
	labelGtx.Constraints.Max = image.Pt(side, side)

	// Shape with the LabelLarge role's typeface, weight, size and line
	// height. Zero fields (the Render path may pass a size-only style) fall
	// back to the shaper's defaults.
	style := tok.label
	f := typeset.Font(style, font.Normal)
	wl := typeset.Label(style, 1)
	mLabel := op.Record(gtx.Ops)
	labelDims := typeset.Layout(labelGtx, shaper, wl, f, unit.Sp(style.Size), label, material)
	labelCall := mLabel.Stop()

	offX := (side - labelDims.Size.X) / 2
	offY := (side - labelDims.Size.Y) / 2
	if offX < 0 {
		offX = 0
	}
	if offY < 0 {
		offY = 0
	}
	st := op.Offset(image.Pt(offX, offY)).Push(gtx.Ops)
	labelCall.Add(gtx.Ops)
	st.Pop()

	return layout.Dimensions{Size: image.Pt(side, side)}
}

// chevronCellWidget renders a ControlHeight-square chevron cell whose
// glyph takes the icon rule, icon.Size(d). pointsRight selects
// the "next" direction; otherwise the chevron points "prev". A step arrow is
// a drawn glyph rather than words, so it takes the label's strength; at the
// edge it cannot step, and takes the platform's disabled control text and
// registers no click — matching the disabled-control convention used by
// components/button.
func chevronCellWidget(pointsRight bool, click *widget.Clickable, enabled bool, tok resolvedTokens, standsOn color.NRGBA) layout.Widget {
	fg := vgcolor.Flatten(tok.platform.Label, standsOn)
	if !enabled {
		fg = vgcolor.Flatten(tok.platform.DisabledControlText, standsOn)
	}
	return func(gtx layout.Context) layout.Dimensions {
		side := gtx.Dp(unit.Dp(tok.density.ControlHeight))
		sz := gtx.Dp(icon.Size(tok.density))
		cgtx := gtx
		cgtx.Constraints = layout.Exact(image.Pt(side, side))
		draw := func(gtx layout.Context) layout.Dimensions {
			drawChevron(gtx, side/2, side/2, sz, fg, pointsRight)
			return layout.Dimensions{Size: image.Pt(side, side)}
		}
		if click == nil || !enabled {
			return draw(cgtx)
		}
		return click.Layout(cgtx, func(gtx layout.Context) layout.Dimensions {
			semantic.EnabledOp(true).Add(gtx.Ops)
			pointer.CursorPointer.Add(gtx.Ops)
			return draw(gtx)
		})
	}
}

// drawChevron paints a filled triangle centred at (cx, cy) fitting within
// an sz × sz square. pointsRight selects the apex direction along +X (next)
// or -X (prev).
func drawChevron(gtx layout.Context, cx, cy, sz int, col color.NRGBA, pointsRight bool) {
	half := float32(sz) / 2
	quarter := float32(sz) / 4
	fcx := float32(cx)
	fcy := float32(cy)

	var p clip.Path
	p.Begin(gtx.Ops)
	if pointsRight {
		p.MoveTo(f32.Pt(fcx-quarter, fcy-half))
		p.LineTo(f32.Pt(fcx+quarter, fcy))
		p.LineTo(f32.Pt(fcx-quarter, fcy+half))
	} else {
		p.MoveTo(f32.Pt(fcx+quarter, fcy-half))
		p.LineTo(f32.Pt(fcx-quarter, fcy))
		p.LineTo(f32.Pt(fcx+quarter, fcy+half))
	}
	p.Close()
	paint.FillShape(gtx.Ops, col, clip.Outline{Path: p.End()}.Op())
}

package picker

import (
	"image"
	"image/color"

	"gioui.org/io/semantic"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"

	"github.com/reactivego/rx"
	"github.com/vibrantgio/components/icons"
	"github.com/vibrantgio/components/internal/control"
	"github.com/vibrantgio/components/list"
	"github.com/vibrantgio/components/scrollbar"
	"github.com/vibrantgio/mvu"
	vgcolor "github.com/vibrantgio/theme/color"
	"github.com/vibrantgio/theme/theme"
	"github.com/vibrantgio/theme/tokens"
	"github.com/vibrantgio/theme/typeset"
)

// MenuState holds the explicit visual state a static menu render draws in.
// The zero value draws nothing, because a menu with no options is not a
// surface.
//
// Intended for golden-image testing and static rendering; production code
// obtains the selection from the Gio event system via [Menu].
type MenuState struct {
	// Options is the list of selectable items, in the order they are drawn.
	Options []string

	// Selected is the index of the one row the picker is holding: the row the
	// check stands beside, and the row that wears the pill while no row is
	// under the pointer. An index outside Options selects nothing, which is
	// what a picker with no value yet looks like.
	Selected int

	// Hovered is the row the pointer is over, counted from ONE, so that the
	// zero value is no row. A hovered row wears the pill and Selected keeps
	// the check — see [highlightedRow].
	//
	// Selected counts from zero because a picker always holds a value and
	// some row is always it. Hover is the opposite: a menu holds a pointer
	// for a moment and holds none the rest of the time, so the state a
	// caller writes when it has nothing to say has to mean nothing is
	// hovered, and zero is the only value a caller can leave out.
	Hovered int

	// MaxHeight caps the plane the rows are drawn on. The zero value is no
	// cap: the menu draws its full height, which is what a handful of
	// options wants and what runs off the bottom of the window at forty.
	// Above the cap the rows scroll inside it and the plane is the cap
	// exactly — see [MenuProps.MaxHeight] for what the cap costs.
	MaxHeight unit.Dp
}

// MenuProps configures a [Menu] instance.
type MenuProps struct {
	// Options is the list of selectable items.
	Options []string

	// Selected is the initial selected index established on subscribe. A
	// later value does not move a running instance — see the package doc.
	Selected int

	// MaxHeight caps the height of the plane the rows are drawn on. The zero
	// value is no cap and the menu draws every row, which is what it has
	// always done and what a handful of options wants.
	//
	// A capped menu scrolls (components/list) and keeps the selected row in
	// view, and the cap is the whole of what makes a long list usable: a
	// catalogue of forty options is taller than the window it opens in, so
	// uncapped its far end is not merely unscrolled but undrawn.
	//
	// What the cap costs is per-row focus tags for the rows a frame did not
	// lay out — a scrolling viewport has tags for what is on screen, where
	// the uncapped menu has one for every option (see the keyboard-reach
	// note below). It buys back the rows the uncapped menu drew past the
	// bottom edge, which no tag could reach either.
	MaxHeight unit.Dp

	// OnSelect is called with the newly selected index on every selection.
	// This is the FRP path. The gtx argument is the layout.Context active on
	// the frame the selection is processed in, so a consumer may emit
	// mvu.MessageOp{Message: …}.Add(gtx.Ops) from inside it.
	OnSelect func(gtx layout.Context, index int)

	// Message, if non-nil, is emitted as mvu.MessageOp into the frame's ops on
	// every selection — the MVU path, where OnSelect is the FRP one. Both fire
	// when both are set, and they fire from the one place the selection is
	// noticed, so neither can dispatch twice.
	Message any

	// Shaper is an explicit per-instance override of the text shaper. Leave it
	// nil in normal use: the menu then shapes with the theme's shaper
	// (tokens.Typography.Shaper()), built once for the process and shared by
	// every component reading that typography. Set it only when this menu must
	// shape with a different one.
	//
	// A shaper is not safe to use from two goroutines; Gio lays the layout
	// tree out on the one goroutine that runs the event loop, which is what
	// makes sharing it correct. See theme/tokens.Typography.Shaper.
	Shaper *text.Shaper
}

// Menu returns an rx.Observable[layout.Widget] emitting the open surface both
// triggers stand under: the option rows, stacked from the top of the box it is
// handed and sized to the width of that box. The layout.Widget it emits reports the
// rows' own height, so a caller may put it in a popover, under a field or in a
// golden test without the menu assuming where it is.
//
// The selection lives in the rx.Defer scope and survives every theme emission
// for the life of the subscription. Both integration paths are supported:
//   - FRP: set MenuProps.OnSelect.
//   - MVU: set MenuProps.Message; the menu emits mvu.MessageOp on selection.
//
// # Keyboard reach
//
// Uncapped, the menu is not virtualised: it walks every option, so while it
// stands every option row exists in the op tree with its own widget.Clickable
// focus tag, and Tab plus Enter/Space operates all of them. There is no
// inoperable option because there is no offscreen option.
//
// That guarantee is bounded by the option count, and [MenuProps.MaxHeight] is
// where the bound is answered: an uncapped menu draws its full height and runs
// off the window long before it runs out of focus tags, so a catalogue that
// long is drawn where nothing can operate it either. A capped menu is a
// components/list viewport, which reaches rows a different way — one focus tag
// for the whole list, arrow keys and Home/End moving the selection over every
// option including the ones no frame laid out, and the viewport following the
// row that moved. Per-row tags then cover what is on screen, and the list's
// own tag covers the rest.
//
// What the keyboard does NOT draw here is a ring. A list shows its focus as
// the platform does for the place it stands in, and a menu's answer is the
// held row: the pill moves, and nothing is drawn around the surface. So the
// rows are wrapped in no halo — see components/list's Halo, which is where a
// list standing in the content or at the front of a dialog takes one.
func Menu(th rx.Observable[theme.Theme], props MenuProps) rx.Observable[layout.Widget] {
	resolved := menuTokens(th)

	return rx.Defer(func() rx.Observable[layout.Widget] {
		optClicks := make([]widget.Clickable, len(props.Options))
		selected := props.Selected
		// The viewport a capped menu scrolls in. It is seeded with the
		// selection so that the first frame of a long menu shows the row the
		// picker is holding rather than the top of the catalogue.
		rows := list.NewState()
		rows.Select(selected)
		rows.Reveal(selected)

		return rx.Map(resolved, func(tok resolvedTokens) layout.Widget {
			shaper := props.Shaper
			if shaper == nil {
				shaper = tok.shaper
			}

			return func(gtx layout.Context) layout.Dimensions {
				for i := range optClicks {
					for optClicks[i].Clicked(gtx) {
						selected = i
						dispatch(gtx, props.OnSelect, props.Message, i)
					}
				}
				if rows.Selected() != selected {
					rows.Select(selected)
					rows.Reveal(selected)
				}
				dims := layoutMenuLive(gtx, shaper, optClicks, rows, tok, MenuState{
					Options:  props.Options,
					Selected: selected,
					Hovered:  hoveredRow(optClicks),
				}, capPixels(gtx, props.MaxHeight))
				if moved := rows.Selected(); moved >= 0 && moved != selected {
					selected = moved
					dispatch(gtx, props.OnSelect, props.Message, moved)
				}
				return dims
			}
		})
	})
}

// dispatch announces one selection down both integration paths from the one
// place the selection is noticed, so neither can fire twice for one change.
func dispatch(gtx layout.Context, onSelect func(layout.Context, int), message any, index int) {
	if onSelect != nil {
		onSelect(gtx, index)
	}
	if message != nil {
		mvu.MessageOp{Message: message}.Add(gtx.Ops)
	}
}

// hoveredRow reports which row the pointer is over in [MenuState.Hovered]'s
// counting: the first hovered clickable plus one, or zero for none. Rows abut
// without overlapping, so at most one can answer.
func hoveredRow(optClicks []widget.Clickable) int {
	for i := range optClicks {
		if optClicks[i].Hovered() {
			return i + 1
		}
	}
	return 0
}

// capPixels converts a stated [MenuProps.MaxHeight] into the pixel cap the row
// stack takes, where zero is no cap at all.
func capPixels(gtx layout.Context, h unit.Dp) int {
	if h <= 0 {
		return 0
	}
	return gtx.Dp(h)
}

// RenderMenu produces a layout.Widget for the open surface in an explicit
// visual state, without any event processing or rx machinery. Intended for
// golden-image testing and static demonstrations; production code should use
// [Menu], which reads both of the parameters below off the theme.
//
// body is the BodyLarge role's whole text style — typeface, weight, size and
// line height all reach the shaper — and d is the density the rows draw at.
// Pass tokens.DefaultTypography.BodyLarge and tokens.Comfortable for the
// default desktop look.
//
// There is no radius parameter because a row has no corner: the surface is a
// stack of full-width rows, and whatever rounds its outline is the plane the
// caller draws it on.
func RenderMenu(
	shaper *text.Shaper,
	p tokens.PlatformColors,
	sp tokens.SpacingScale,
	body tokens.TextStyle,
	d tokens.Density,
	s MenuState,
) layout.Widget {
	tok := resolvedTokens{platform: p, spacing: sp, body: body, density: d}
	return func(gtx layout.Context) layout.Dimensions {
		return drawMenu(gtx, shaper, tok, s, capPixels(gtx, s.MaxHeight))
	}
}

// menuTokens flattens the theme into the snapshot a menu frame needs: the
// platform's colours, the BodyLarge style its rows are set in, the spacing
// scale, the density, and the theme's cached shaper (the theme owns the
// typeface).
func menuTokens(th rx.Observable[theme.Theme]) rx.Observable[resolvedTokens] {
	return rx.SwitchMap(th, func(t theme.Theme) rx.Observable[resolvedTokens] {
		return rx.Map(
			rx.CombineLatest4(t.Platform, t.Typography, t.Spacing, t.Density),
			func(n rx.Tuple4[tokens.PlatformColors, tokens.Typography, tokens.SpacingScale, tokens.Density]) resolvedTokens {
				typ := n.Second
				return resolvedTokens{
					platform: n.First,
					body:     typ.BodyLarge,
					spacing:  n.Third,
					density:  n.Fourth,
					shaper:   typ.Shaper(),
				}
			},
		)
	})
}

// layoutMenuLive stacks the option rows with their Clickable hit areas.
//
// A row's pointer target is the row: they stack directly against each other,
// so anything added to one would be taken off its neighbour, and the full row
// width is what makes a row easy to land on. What they measure, at 1:1, is
// 28 dp Comfortable and 24 dp Compact — BodyLarge's 24 dp line box plus
// 2×PaddingY, which wins over the ControlHeight floor in both densities.
func layoutMenuLive(gtx layout.Context, shaper *text.Shaper, optClicks []widget.Clickable, rows *list.State, tok resolvedTokens, s MenuState, capPx int) layout.Dimensions {
	return stackRows(gtx, rows, len(s.Options), capPx, tok, func(gtx layout.Context, i int) layout.Dimensions {
		return optClicks[i].Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			semantic.Button.Add(gtx.Ops)
			return drawOptionRow(gtx, shaper, tok, i == s.Selected, i == highlightedRow(s), s.Options[i])
		})
	})
}

// drawMenu stacks the option rows for the pure path.
func drawMenu(gtx layout.Context, shaper *text.Shaper, tok resolvedTokens, s MenuState, capPx int) layout.Dimensions {
	// A viewport with no frames behind it: a static render of a capped menu
	// is its resting state, the rows from the top, because a scroll position
	// is something a menu acquires by being scrolled.
	return stackRows(gtx, nil, len(s.Options), capPx, tok, func(gtx layout.Context, i int) layout.Dimensions {
		return drawOptionRow(gtx, shaper, tok, i == s.Selected, i == highlightedRow(s), s.Options[i])
	})
}

// stackRows draws n option rows down the box it is handed and reports the
// plane they fill, which is the whole of what a menu is.
//
// Uncapped, the rows are stacked directly — the menu is exactly as tall as its
// options and every one of them exists in the op tree with its own focus tag,
// which is the keyboard-reach guarantee [Menu]'s doc makes. Capped, they are
// laid out in a components/list viewport of that height instead, so the plane
// is the cap and the rows move under it; rows outside the viewport are not
// laid out, which is what virtualising means and what the cap trades for
// reaching a catalogue longer than the window.
//
// The capped viewport is drawn through [list.LayoutSelectableScrollbar] with
// [list.Overlay]: the same coupling components/gallery/inventory's list block
// already draws its own bar through. The bar is the rows' own colours, and
// scrollbar.Style.Layout draws nothing when the
// viewport shows the whole of the content, so a capped menu whose rows all
// fit is exactly as bare as an uncapped one; only a cap that actually cuts the
// catalogue short earns the bar.
//
// rows may be nil, and is where there are no frames to keep a position across.
func stackRows(gtx layout.Context, rows *list.State, n, capPx int, tok resolvedTokens, row func(gtx layout.Context, i int) layout.Dimensions) layout.Dimensions {
	if n == 0 {
		// An empty menu is not an empty plane, it is nothing at all.
		return layout.Dimensions{}
	}
	fieldW := gtx.Constraints.Max.X

	if capPx <= 0 {
		totalH := 0
		for i := 0; i < n; i++ {
			off := op.Offset(image.Pt(0, totalH)).Push(gtx.Ops)
			rowGtx := gtx
			rowGtx.Constraints = layout.Constraints{
				Min: image.Pt(0, 0),
				Max: image.Pt(fieldW, gtx.Constraints.Max.Y),
			}
			rowDims := row(rowGtx, i)
			off.Pop()
			totalH += rowDims.Size.Y
		}
		return layout.Dimensions{Size: image.Pt(fieldW, totalH)}
	}

	if rows == nil {
		rows = list.NewState()
	}
	// The cap is a height in pixels and not a share of the box the menu was
	// offered: a field's menu is drawn outside the trigger's own row, so the
	// box the rows were handed says nothing about the room the menu has.
	viewGtx := gtx
	viewGtx.Constraints = layout.Constraints{
		Min: image.Pt(fieldW, 0),
		Max: image.Pt(fieldW, capPx),
	}
	// The list's items are the row indices, because what a row draws is a
	// function of where it sits in the menu and not of its label alone —
	// two providers may well offer the same model name.
	idx := make([]int, n)
	for i := range idx {
		idx[i] = i
	}
	bar := scrollbar.FromTokens(tok.platform, tok.platform.WindowBackground)
	return list.LayoutSelectableScrollbar(viewGtx, rows, bar, list.Overlay, idx, func(gtx layout.Context, i int, _ bool) layout.Dimensions {
		return row(gtx, i)
	})
}

// rowHeights measures every option row on its own, into a recording nothing
// adds, so a caller fitting the plane to the available room can end the plane
// on a row's edge instead of through one row's letters. The rows are measured
// at the width the menu will draw at, which is what a row's height is a
// function of once a label is long enough to wrap.
func rowHeights(gtx layout.Context, shaper *text.Shaper, tok resolvedTokens, s MenuState) []int {
	hs := make([]int, len(s.Options))
	measure := op.Record(gtx.Ops)
	rowGtx := gtx
	rowGtx.Constraints = layout.Constraints{
		Min: image.Point{},
		Max: image.Pt(gtx.Constraints.Max.X, gtx.Constraints.Max.Y),
	}
	for i := range s.Options {
		hs[i] = drawOptionRow(rowGtx, shaper, tok, i == s.Selected, i == s.Selected, s.Options[i]).Size.Y
	}
	measure.Stop()
	return hs
}

// The menu's selection pill, and the mark that stands in it.
//
// NOT MEASURED. No stored capture in reference/macos holds an open menu at
// all, so neither the pill's inset nor its corner nor the box the mark is
// drawn in is read off this platform. Each is named from the one drawing the
// reference does hold of a pill inset inside a rail — the sidebar's selected
// row, measured off voicememos-sidebar-{light,dark}.png as inset 10 and
// cornered at 8 — and the capture that would settle a menu's own is on the
// reference's list.
const (
	// selectionInsetDp is what the pill leaves clear at either end of the
	// row, the sidebar pill's own inset from its rail.
	selectionInsetDp = unit.Dp(10)

	// selectionRadiusDp is the pill's corner, the sidebar pill's own.
	selectionRadiusDp = unit.Dp(8)

	// markBoxDp is the square the check is drawn in: the smallest of the
	// three sizes components/icons is drawn at, which is the one that stands
	// beside a line of body text rather than alone in a control.
	markBoxDp = unit.Dp(16)
)

// optionRowColors returns an option row's fill and the foreground that reads
// on it. A surface decides what can be read on it, so a row's two colours are
// never picked apart: they are returned as a pair.
//
// THE MENU'S OWN PLANE. A row that is neither held nor under the pointer is
// it: the platform's window background, which is the fill of every floating
// surface on this platform, under the platform's label flattened onto it.
//
// THE HIGHLIGHTED ROW is the pill: the platform's selection over a floating
// surface, under the foreground it names for text standing on a fill the
// accent paints. No name in the recorded set is a menu's own selection, so the
// pill takes the one selection the reference measures inside a chrome region
// rather than in a content list — SidebarSelection — and the capture that
// would settle a menu's own is on the reference's list.
func optionRowColors(p tokens.PlatformColors, highlighted bool) (fill, foreground color.NRGBA) {
	if highlighted {
		return p.SidebarSelection, p.AlternateSelectedControlText
	}
	return p.WindowBackground, vgcolor.Flatten(p.Label, p.WindowBackground)
}

// highlightedRow reports which row wears the pill: the row under the pointer
// where there is one, and the row the picker is holding where there is not.
//
// The POINTER WINS. The pill says where the choice would land if the press
// came now, and that is the row under the pointer; what the picker is holding
// is said by the check instead, which never moves. Marking both would put two
// pills on one menu, which is the platform drawing two answers to one
// question.
func highlightedRow(s MenuState) int {
	if s.Hovered > 0 {
		return s.Hovered - 1
	}
	return s.Selected
}

// rowColumns reports where the two things a row draws stand, as insets from
// the plane's own leading edge: the box the check occupies and the column the
// label starts on.
//
// Both are spent from the PILL rather than from the plane, because the pill is
// what a row is read inside: the check stands at the pill's own leading edge,
// and the label follows it after the text field's leading inset
// ([control.TextLeadDp]). They do not move with the state — a row that carries
// no check leaves the column clear, so the labels of a menu stand in one line
// whichever row the picker is holding.
func rowColumns(gtx layout.Context) (markLead, markBox, labelLead int) {
	markLead = gtx.Dp(selectionInsetDp)
	markBox = gtx.Dp(markBoxDp)
	return markLead, markBox, markLead + markBox + gtx.Dp(control.TextLeadDp)
}

// drawOptionRow renders a single option row: the plane it stands on, the pill
// where the row is highlighted, the check where the row is the one the picker
// is holding, and the label.
//
// current says the row is the one the picker holds and is what the check
// answers to; highlighted says the row wears the pill. They are two questions
// — see [highlightedRow] — and a row may answer either, both or neither.
func drawOptionRow(gtx layout.Context, shaper *text.Shaper, tok resolvedTokens, current, highlighted bool, label string) layout.Dimensions {
	// An option row draws max(the density's control height, its line box plus
	// twice the density's vertical padding), the sizing rule every stacked row
	// in the system takes.
	markLead, markBox, lead := rowColumns(gtx)
	trail := gtx.Dp(selectionInsetDp) + gtx.Dp(control.TextTrailDp)
	padV := gtx.Dp(unit.Dp(tok.density.PaddingY))
	f, wl, textSize := bodyLabel(tok)
	minH := gtx.Dp(unit.Dp(tok.density.ControlHeight))
	fieldW := gtx.Constraints.Max.X

	bg, textCol := optionRowColors(tok.platform, highlighted)
	innerW := fieldW - lead - trail
	if innerW < 1 {
		innerW = 1
	}
	innerGtx := gtx
	innerGtx.Constraints = layout.Constraints{
		Min: image.Pt(0, 0),
		Max: image.Pt(innerW, gtx.Constraints.Max.Y),
	}

	mTextCol := op.Record(gtx.Ops)
	paint.ColorOp{Color: textCol}.Add(gtx.Ops)
	textMat := mTextCol.Stop()

	mLabel := op.Record(gtx.Ops)
	labelDims := typeset.Layout(innerGtx, shaper, wl, f, textSize, label, textMat)
	labelCall := mLabel.Stop()

	rowH := labelDims.Size.Y + 2*padV
	if rowH < minH {
		rowH = minH
	}
	rowSize := image.Pt(fieldW, rowH)

	// The plane first, the whole width of the row: the pill is inset inside
	// it, so what stands at either end of the row is the menu's own fill.
	paint.FillShape(gtx.Ops, tok.platform.WindowBackground, clip.Rect{Max: rowSize}.Op())
	if highlighted {
		paintSelection(gtx, rowSize, bg)
	}

	if current {
		if mark := icons.Mark(icons.Check); mark != nil {
			st := op.Offset(image.Pt(markLead, (rowH-markBox)/2)).Push(gtx.Ops)
			mark(gtx, markBox, textCol)
			st.Pop()
		}
	}

	offY := (rowH - labelDims.Size.Y) / 2
	st := op.Offset(image.Pt(lead, offY)).Push(gtx.Ops)
	labelCall.Add(gtx.Ops)
	st.Pop()

	return layout.Dimensions{Size: rowSize}
}

// paintSelection fills the highlighted row's pill at the current offset:
// [selectionInsetDp] in from either end of the row, cornered at
// [selectionRadiusDp], the row's full height. It is the drawing
// patterns/sidebar makes for a selected rail row, at the numbers that pattern
// measures, because until a capture holds an open menu it is the one pill this
// platform is recorded drawing.
func paintSelection(gtx layout.Context, size image.Point, fill color.NRGBA) {
	inset := gtx.Dp(selectionInsetDp)
	if 2*inset >= size.X {
		inset = 0
	}
	bounds := image.Rect(inset, 0, size.X-inset, size.Y)
	if bounds.Dx() <= 0 || bounds.Dy() <= 0 {
		return
	}
	r := gtx.Dp(selectionRadiusDp)
	if half := min(bounds.Dx(), bounds.Dy()) / 2; r > half {
		r = half
	}
	paint.FillShape(gtx.Ops, fill, clip.RRect{Rect: bounds, NE: r, NW: r, SE: r, SW: r}.Op(gtx.Ops))
}

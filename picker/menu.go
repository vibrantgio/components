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
	"github.com/vibrantgio/components/list"
	"github.com/vibrantgio/components/scrollbar"
	"github.com/vibrantgio/mvu"
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

	// Selected is the index of the one row drawn on the accent plane. An
	// index outside Options selects nothing, which is what a picker with no
	// value yet looks like.
	Selected int

	// Hovered is the row the pointer is over, counted from ONE, so that the
	// zero value is no row.
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
// focus tag, and Tab plus Enter/Space reaches all of them. There is no
// unreachable option because there is no offscreen option.
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
					Options:   props.Options,
					Selected:  selected,
					Hovered:   hoveredRow(optClicks),
					MaxHeight: props.MaxHeight,
				})
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
	colors tokens.ColorTokens,
	sp tokens.SpacingScale,
	body tokens.TextStyle,
	d tokens.Density,
	s MenuState,
) layout.Widget {
	tok := resolvedTokens{color: colors, spacing: sp, body: body, density: d}
	return func(gtx layout.Context) layout.Dimensions {
		return drawMenu(gtx, shaper, tok, s)
	}
}

// menuTokens flattens the theme into the snapshot a menu frame needs: the
// palette, the BodyLarge role its rows are set in, the spacing scale, the
// density, and the theme's cached shaper (the theme owns the typeface).
func menuTokens(th rx.Observable[theme.Theme]) rx.Observable[resolvedTokens] {
	return rx.SwitchMap(th, func(t theme.Theme) rx.Observable[resolvedTokens] {
		return rx.Map(
			rx.CombineLatest4(t.Color, t.Typography, t.Spacing, t.Density),
			func(n rx.Tuple4[tokens.ColorTokens, tokens.Typography, tokens.SpacingScale, tokens.Density]) resolvedTokens {
				typ := n.Second
				return resolvedTokens{
					color:   n.First,
					body:    typ.BodyLarge,
					spacing: n.Third,
					density: n.Fourth,
					shaper:  typ.Shaper(),
				}
			},
		)
	})
}

// layoutMenuLive stacks the option rows with their Clickable hit areas.
//
// The rows are not extended to the density's minimum pointer target: they
// stack directly against each other, so a ≥44 dp slop per row would overlap
// the neighbouring rows' targets. They rely on their full-row width instead.
// What they measure, at 1:1, is 40 dp Comfortable and 36 dp Compact —
// BodyLarge's 24 dp line box plus 2×PaddingY, which wins over the ControlHeight
// floor in both densities — so both clear WCAG 2.5.8 Target Size (Minimum),
// the 24 dp AA criterion these rows are held to. See tokens.MinHitTarget for
// why 2.5.5's 44 dp is not that criterion.
func layoutMenuLive(gtx layout.Context, shaper *text.Shaper, optClicks []widget.Clickable, rows *list.State, tok resolvedTokens, s MenuState) layout.Dimensions {
	return stackRows(gtx, rows, len(s.Options), s.MaxHeight, tok, func(gtx layout.Context, i int) layout.Dimensions {
		return optClicks[i].Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			semantic.Button.Add(gtx.Ops)
			return drawOptionRow(gtx, shaper, tok, i == s.Selected, i+1 == s.Hovered, s.Options[i])
		})
	})
}

// drawMenu stacks the option rows for the pure path.
func drawMenu(gtx layout.Context, shaper *text.Shaper, tok resolvedTokens, s MenuState) layout.Dimensions {
	// A viewport with no frames behind it: a static render of a capped menu
	// is its resting state, the rows from the top, because a scroll position
	// is something a menu acquires by being scrolled.
	return stackRows(gtx, nil, len(s.Options), s.MaxHeight, tok, func(gtx layout.Context, i int) layout.Dimensions {
		return drawOptionRow(gtx, shaper, tok, i == s.Selected, i+1 == s.Hovered, s.Options[i])
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
// already draws its own bar through. The bar is scrollbar.FromTokens(tok.color)
// — the rows' own palette — and scrollbar.Style.Layout draws nothing when the
// viewport shows the whole of the content, so a capped menu whose rows all
// fit is exactly as bare as an uncapped one; only a cap that actually cuts the
// catalogue short earns the bar.
//
// rows may be nil, and is where there are no frames to keep a position across.
func stackRows(gtx layout.Context, rows *list.State, n int, maxH unit.Dp, tok resolvedTokens, row func(gtx layout.Context, i int) layout.Dimensions) layout.Dimensions {
	if n == 0 {
		// An empty menu is not an empty plane, it is nothing at all.
		return layout.Dimensions{}
	}
	fieldW := gtx.Constraints.Max.X

	if maxH <= 0 {
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

	capPx := gtx.Dp(maxH)
	if capPx < 1 {
		capPx = 1
	}
	if rows == nil {
		rows = list.NewState()
	}
	// The cap is the caller's number and not a share of the box the menu was
	// offered: a field's menu is drawn outside the trigger's own row and the
	// height it was handed says nothing about the room the menu has.
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
	bar := scrollbar.FromTokens(tok.color)
	return list.LayoutSelectableScrollbar(viewGtx, rows, bar, list.Overlay, idx, func(gtx layout.Context, i int, _ bool) layout.Dimensions {
		return row(gtx, i)
	})
}

// optionRowColors returns an option row's fill and the foreground that reads
// on it, chosen together. A surface decides what can be read on it, so a row's two
// colours are never picked apart: they are returned as a pair and measured as
// a pair (TestMenuOptionRowContrast).
//
// THE MENU'S OWN PLANE. A resting row is it. The open menu is a floating
// transient overlay — an unscrimmed, shadowless plane like patterns/popover —
// so its rows fill at level 3, the top of the elevation,
// asked of the palette rather than of a ramp index. The scheme's body text
// reads on that fill at 18.58:1 light and 8.01:1 dark.
//
// THE SELECTED ROW is the accent, which is the one thing on the menu that is
// not neutral and the one row that is not a choice but the answer. The fill is
// the step of the accent ramp nearest its mid-value step that reaches WCAG
// 1.4.3's 4.5:1 against the menu's own plane, and the foreground is the
// neutral step that reaches the same floor against that fill — each side asked
// of the ramp against the surface it actually meets, neither named. Aiming the
// fill at the 3:1 non-text floor instead is not enough by half: it answers a mid-tone the
// neutral ramp cannot carry text on at all, 4.27:1 at its best, which is the
// same wall a mid-grey highlight runs into. Held at the text floor the pairing
// measures 6.72:1 selected-against-menu and 4.53:1 foreground-on-selected in the
// light scheme, 4.58 and 4.58 in the dark, and over a ten-seed sweep of both
// schemes and both contrast variants no pairing falls under 4.56 and 4.50.
//
// A neutral state walk on the menu's own surface cannot serve for either
// coloured row, and this is where that is settled once. A mid-grey surface is
// precisely where no neutral foreground reaches the text floor, and a walk
// whose surface flips with the scheme while its foreground does not measures
// 1.75:1 in the
// dark scheme: light text on a light-grey highlight. The neutral ramp's 900
// end is the DARK end in one scheme and the light end in the other, so "one
// step further along the ramp" is not one direction.
//
// THE HOVERED ROW is the accent again, a step less pronounced: the role's tonal
// container, its hue held at one measured chroma and one measured depth. It
// is the same colour family as the selection and nowhere near its weight, so
// the pointer says "here" without ever being mistaken for the answer — the
// two are 4.53:1 apart in light and 6.66:1 in dark. Body text reads on it at
// 12.53:1 and 11.64:1, worst 11.65 over the sweep. The container carries no
// contrast floor of its own because hover is not a mark: it says nothing the
// reader cannot already see from where the pointer is, and it is gone the
// moment the pointer is.
//
// Selection wins over hover, the way a press wins over a hover elsewhere in
// this library: the selected row is already the row the menu is pointing at,
// and a transient state fill has nothing to add to a standing answer.
func optionRowColors(c tokens.ColorTokens, selected, hovered bool) (fill, foreground color.NRGBA) {
	plane := c.SurfaceAt(tokens.Level3)
	switch {
	case selected:
		f := c.MarkOn(tokens.RolePrimary, plane, tokens.TextFloor)
		return f, c.MarkOn(tokens.RoleNeutral, f, tokens.TextFloor)
	case hovered:
		return c.StatusContainer(tokens.RolePrimary), c.Text
	}
	return plane, c.Text
}

// drawOptionRow renders a single option row.
func drawOptionRow(gtx layout.Context, shaper *text.Shaper, tok resolvedTokens, selected, hovered bool, label string) layout.Dimensions {
	// Option rows are list rows — row height = Density.ControlHeight exactly
	// (components/list's RowHeight rule: 36 dp Comfortable, 28 dp Compact) —
	// with the same static S3 side padding the field trigger takes.
	padH := gtx.Dp(unit.Dp(tok.spacing.S3))
	padV := gtx.Dp(unit.Dp(tok.density.PaddingY))
	f, wl, textSize := bodyLabel(tok)
	minH := gtx.Dp(unit.Dp(tok.density.ControlHeight))
	fieldW := gtx.Constraints.Max.X

	bg, textCol := optionRowColors(tok.color, selected, hovered)
	innerW := fieldW - 2*padH
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

	paint.FillShape(gtx.Ops, bg, clip.Rect{Max: rowSize}.Op())

	offY := (rowH - labelDims.Size.Y) / 2
	st := op.Offset(image.Pt(padH, offY)).Push(gtx.Ops)
	labelCall.Add(gtx.Ops)
	st.Pop()

	return layout.Dimensions{Size: rowSize}
}

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

	// Selected is the index of the one row drawn on the inverted plane. An
	// index outside Options selects nothing, which is what a picker with no
	// value yet looks like.
	Selected int
}

// MenuProps configures a [Menu] instance.
type MenuProps struct {
	// Options is the list of selectable items.
	Options []string

	// Selected is the initial selected index established on subscribe. A
	// later value does not move a running instance — see the package doc.
	Selected int

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
	// A shaper is not safe to use from two goroutines; Gio lays the widget
	// forest out on the one goroutine that runs the event loop, which is what
	// makes sharing it correct. See theme/tokens.Typography.Shaper.
	Shaper *text.Shaper
}

// Menu returns an rx.Observable[layout.Widget] emitting the open surface both
// triggers stand under: the option rows, stacked from the top of the box it is
// handed and sized to the width of that box. The widget it emits reports the
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
// The menu is not virtualised: it walks every option, so while it stands every
// option row exists in the op tree with its own widget.Clickable focus tag,
// and Tab plus Enter/Space reaches all of them. There is no unreachable option
// because there is no offscreen option — unlike a virtualised region, which
// reaches only the rows a frame laid out.
//
// Two things follow. First, that guarantee is bounded by the option count: the
// menu draws its full height and would run off the window before it ran out of
// focus tags, so an options list long enough to need virtualising must move to
// components/list's LayoutSelectable rather than grow per-row tags. Second,
// Tab-per-option is not the menu behaviour a listbox implies — arrow keys
// should move a highlight within the open menu and Escape should close it.
// That is a real gap, and it is a menu-semantics gap rather than a
// virtualisation one.
func Menu(th rx.Observable[theme.Theme], props MenuProps) rx.Observable[layout.Widget] {
	resolved := menuTokens(th)

	return rx.Defer(func() rx.Observable[layout.Widget] {
		optClicks := make([]widget.Clickable, len(props.Options))
		selected := props.Selected

		return rx.Map(resolved, func(tok resolvedTokens) layout.Widget {
			shaper := props.Shaper
			if shaper == nil {
				shaper = tok.shaper
			}

			return func(gtx layout.Context) layout.Dimensions {
				for i := range optClicks {
					for optClicks[i].Clicked(gtx) {
						selected = i
						if props.OnSelect != nil {
							props.OnSelect(gtx, i)
						}
						if props.Message != nil {
							mvu.MessageOp{Message: props.Message}.Add(gtx.Ops)
						}
					}
				}
				return layoutMenuLive(gtx, shaper, optClicks, tok, MenuState{
					Options:  props.Options,
					Selected: selected,
				})
			}
		})
	})
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
// The rows are NOT extended to the density's minimum pointer target: they
// stack directly against each other, so a ≥44 dp slop per row would overlap
// the neighbouring rows' targets. They rely on their full-row width instead.
// What they measure, at 1:1, is 40 dp Comfortable and 36 dp Compact —
// BodyLarge's 24 dp line box plus 2×PaddingY, which wins over the ControlHeight
// floor in both densities — so both clear WCAG 2.5.8 Target Size (Minimum),
// the 24 dp AA criterion these rows are held to. See tokens.MinHitTarget for
// why 2.5.5's 44 dp is not that criterion.
func layoutMenuLive(gtx layout.Context, shaper *text.Shaper, optClicks []widget.Clickable, tok resolvedTokens, s MenuState) layout.Dimensions {
	if len(s.Options) == 0 {
		return layout.Dimensions{}
	}
	fieldW := gtx.Constraints.Max.X
	totalH := 0
	for i := range optClicks {
		off := op.Offset(image.Pt(0, totalH)).Push(gtx.Ops)
		optGtx := gtx
		optGtx.Constraints = layout.Constraints{
			Min: image.Pt(0, 0),
			Max: image.Pt(fieldW, gtx.Constraints.Max.Y),
		}
		idx := i
		label := s.Options[idx]
		optDims := optClicks[idx].Layout(optGtx, func(gtx layout.Context) layout.Dimensions {
			semantic.Button.Add(gtx.Ops)
			return drawOptionRow(gtx, shaper, tok, idx == s.Selected, label)
		})
		off.Pop()
		totalH += optDims.Size.Y
	}
	return layout.Dimensions{Size: image.Pt(fieldW, totalH)}
}

// drawMenu stacks the option rows for the pure path.
func drawMenu(gtx layout.Context, shaper *text.Shaper, tok resolvedTokens, s MenuState) layout.Dimensions {
	if len(s.Options) == 0 {
		return layout.Dimensions{}
	}
	fieldW := gtx.Constraints.Max.X
	totalH := 0
	for i, opt := range s.Options {
		off := op.Offset(image.Pt(0, totalH)).Push(gtx.Ops)
		optGtx := gtx
		optGtx.Constraints = layout.Constraints{
			Min: image.Pt(0, 0),
			Max: image.Pt(fieldW, gtx.Constraints.Max.Y),
		}
		optDims := drawOptionRow(optGtx, shaper, tok, i == s.Selected, opt)
		off.Pop()
		totalH += optDims.Size.Y
	}
	return layout.Dimensions{Size: image.Pt(fieldW, totalH)}
}

// optionRowColors returns an option row's fill and the ink that reads on it,
// chosen together. A ground decides what can be read on it, so a row's two
// colours are never picked apart: they are returned as a pair and measured as
// a pair (TestMenuOptionRowContrast).
//
// An unselected row is the menu's own plane. The open menu is a floating
// transient overlay — an unscrimmed, shadowless plane like patterns/popover —
// so its rows fill at level 3 on the elevation ladder, the top of the ladder,
// asked of the palette rather than of a ramp index. The scheme's body text
// reads on that fill at 9.16:1 light and 8.01:1 dark.
//
// A selected row is the menu's one inverted plane: the theme's inverse pair, a
// surface built from the counterpart scheme carrying the ink the theme derives
// to read on it — 13.71:1 light and 15.16:1 dark, the counterpart scheme's own
// reading pair.
//
// A neutral state walk on the menu's own ground cannot serve, and that is the
// constraint the inverse pair exists for. A mid-grey ground is precisely where
// no neutral ink can reach WCAG 1.4.3's 4.5:1 for text — the whole ramp tops
// out at 4.27:1 over the light scheme's Neutral 600 — and a walk whose ground
// flips with the scheme while its ink does not measures 1.75:1 in the dark
// scheme: light text on a light-grey highlight. The inverse pair keeps the
// direction that walk had in each scheme — a selected row is darker than the
// menu in a light scheme and lighter in a dark one — separates from the menu
// fill by 7.85:1 light and 7.58:1 dark, well past 1.4.11's 3:1 for a non-text
// indicator, and carries an ink that reads on it in both.
func optionRowColors(c tokens.ColorTokens, selected bool) (fill, ink color.NRGBA) {
	if selected {
		return c.InverseSurface, c.OnInverseSurface
	}
	return c.SurfaceAt(tokens.Level3), c.Text
}

// drawOptionRow renders a single option row.
func drawOptionRow(gtx layout.Context, shaper *text.Shaper, tok resolvedTokens, selected bool, label string) layout.Dimensions {
	// Option rows are list rows — row height = Density.ControlHeight exactly
	// (components/list's RowHeight rule: 36 dp Comfortable, 28 dp Compact) —
	// with the same static S3 side padding the field trigger takes.
	padH := gtx.Dp(unit.Dp(tok.spacing.S3))
	padV := gtx.Dp(unit.Dp(tok.density.PaddingY))
	f, wl, textSize := bodyLabel(tok)
	minH := gtx.Dp(unit.Dp(tok.density.ControlHeight))
	fieldW := gtx.Constraints.Max.X

	bg, textCol := optionRowColors(tok.color, selected)
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

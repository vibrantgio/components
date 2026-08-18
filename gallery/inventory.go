// The gallery's inventory: every published family, built once from static
// state so that one page can show them all at once.
//
// The per-family pages beside this file each drive a live widget and read the
// theme's own default light tokens. The inventory does neither: every section
// here is a pure function of the colour tokens it is handed, so the same code
// draws the whole surface in either scheme and a test can capture a section
// without a window. What the sections do keep across frames — scroll
// positions, the parsed reading sample, the icon set — hangs off the
// [inventory] value rather than off a page.
package main

import (
	"fmt"
	"image"
	"image/color"
	"runtime"

	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"

	"github.com/vibrantgio/components/icon"
	"github.com/vibrantgio/components/icons"
	"github.com/vibrantgio/components/input"
	complayout "github.com/vibrantgio/components/layout"
	"github.com/vibrantgio/components/list"
	"github.com/vibrantgio/components/richtext"
	"github.com/vibrantgio/components/scrollarea"
	"github.com/vibrantgio/components/scrollbar"
	"github.com/vibrantgio/markdown"
	"github.com/vibrantgio/theme/tokens"

	"github.com/vibrantgio/components/button"
	ivgraster "github.com/vibrantgio/ivg/raster/gio"
)

// section is one labelled block of the inventory: a heading and the widget
// that shows the family under it.
type section struct {
	// name is the golden-image name for this section, unique across the
	// whole inventory.
	name string
	// title is the heading shown above the body.
	title string
	// body draws the family. It is bounded by its own slot, never by the
	// page: several patterns expand into whatever constraints they are
	// given, and an unbounded one would take the column with it.
	body layout.Widget
	// height is the slot the body is laid out in, in dp.
	height unit.Dp
}

// group collects the sections that come from one module.
type group struct {
	name     string
	sections []section
}

// inventory owns everything the static sections need to survive across frames
// and builds the sections from it.
//
// The reading sample is parsed once here and never per frame: the document
// keys its per-block interaction state on block pointers, so re-parsing would
// silently orphan every scroll and hover position it holds.
type inventory struct {
	shaper *text.Shaper

	rows    []string
	bars    []string
	listSt  *list.State
	barList layout.List
	barSt   *scrollbar.State
	areaSt  *scrollarea.State

	marks *icons.Set
	reg   *icon.Registry
	// ivg caches the rendered vector icon per ink colour. The icon carries
	// its own palette, which on a dark ground would be a black disc on
	// black, so each scheme gets it recoloured — and rasterising is not
	// something to redo every frame.
	ivg map[color.NRGBA]layout.Widget

	doc *markdown.Document
}

func newInventory(shaper *text.Shaper) *inventory {
	inv := &inventory{
		shaper:  shaper,
		listSt:  list.NewState(),
		barList: layout.List{Axis: layout.Vertical},
		barSt:   scrollbar.NewState(),
		areaSt:  scrollarea.NewState(),
		marks:   icons.New(runtime.GOOS),
		reg:     icon.New(),
		doc:     markdown.NewDocument(markdown.Parse([]byte(readingSample))),
	}
	inv.rows = make([]string, 40)
	inv.bars = make([]string, 40)
	for i := range inv.rows {
		inv.rows[i] = fmt.Sprintf("Row %d — a virtual list lays out only what shows", i+1)
		inv.bars[i] = fmt.Sprintf("Line %d of %d — content the bar beside it measures", i+1, len(inv.rows))
	}
	inv.reg.Register("info", icon.FromIVG(actionInfoIVG))
	inv.ivg = map[color.NRGBA]layout.Widget{}
	return inv
}

// vectorIcon returns the registered vector icon drawn in ink.
func (inv *inventory) vectorIcon(ink color.NRGBA) layout.Widget {
	if w, ok := inv.ivg[ink]; ok {
		return w
	}
	blank := func(gtx layout.Context) layout.Dimensions {
		return layout.Dimensions{Size: image.Pt(gtx.Dp(40), gtx.Dp(40))}
	}
	w := blank
	if ic, ok := inv.reg.Icon("info"); ok {
		if built, err := ivgraster.Widget(ic.IVG(), 40, 40, ivgraster.WithColors(ink)); err == nil {
			w = built
		}
	}
	inv.ivg[ink] = w
	return w
}

// groups returns the whole inventory in the given scheme, in the order the
// everything page shows it: what a theme is made of first, then the widgets
// built on it, then the compositions, then prose.
func (inv *inventory) groups(c tokens.ColorTokens) []group {
	return []group{
		{name: "Foundations", sections: inv.foundations(c)},
		{name: "Components", sections: inv.components(c)},
		{name: "Patterns", sections: inv.patterns(c)},
		{name: "Markdown", sections: inv.reading(c)},
	}
}

// ── Foundations ───────────────────────────────────────────────────────────────

func (inv *inventory) foundations(c tokens.ColorTokens) []section {
	return []section{
		{
			name: "foundations-roles", title: "Palette — the scheme's semantic roles", height: 76,
			body: inv.roleSwatches(c),
		},
		{
			name: "foundations-ramps", title: "Palette — the functional ramps, nine steps each", height: 190,
			body: inv.rampSwatches(c),
		},
		{
			name: "foundations-type", title: "Typography — every role a surface reads in", height: 460,
			body: inv.typeScale(c),
		},
	}
}

func (inv *inventory) roleSwatches(c tokens.ColorTokens) layout.Widget {
	type swatch struct {
		name  string
		fill  color.NRGBA
		on    color.NRGBA
		label string
	}
	sw := []swatch{
		{"Primary", c.Primary, c.OnPrimary, "Aa"},
		{"Secondary", c.Secondary, c.OnSecondary, "Aa"},
		{"Tertiary", c.Tertiary, c.OnTertiary, "Aa"},
		{"Success", c.Success, c.OnSuccess, "Aa"},
		{"Warning", c.Warning, c.OnWarning, "Aa"},
		{"Error", c.Error, c.OnError, "Aa"},
		{"Background", c.Background, c.Text, "Aa"},
		{"Surface", c.Surface, c.Text, "Aa"},
		{"Divider", c.Divider, c.Text, "Aa"},
		{"Inverse", c.InverseSurface, c.OnInverseSurface, "Aa"},
	}
	return func(gtx layout.Context) layout.Dimensions {
		cs := make([]layout.FlexChild, 0, 2*len(sw))
		for i, s := range sw {
			s := s
			if i > 0 {
				cs = append(cs, layout.Rigid(complayout.HSpacer(8)))
			}
			cs = append(cs, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						sz := image.Pt(gtx.Dp(56), gtx.Dp(40))
						paint.FillShape(gtx.Ops, s.fill, clip.Rect{Max: sz}.Op())
						swatchBorder(gtx, c.Ramps.Neutral.Step(400), sz, 1)
						gtx.Constraints = layout.Exact(sz)
						return complayout.InsetXY(8, 10).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							return labelAt(gtx, inv.shaper, s.label, s.on, 13, font.Font{})
						})
					}),
					layout.Rigid(complayout.VSpacer(6)),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return labelAt(gtx, inv.shaper, s.name, c.Text, 11, font.Font{})
					}),
				)
			}))
		}
		return layout.Flex{}.Layout(gtx, cs...)
	}
}

func (inv *inventory) rampSwatches(c tokens.ColorTokens) layout.Widget {
	ramps := []struct {
		name string
		ramp tokens.Ramp
	}{
		{"Neutral", c.Ramps.Neutral},
		{"Primary", c.Ramps.Primary},
		{"Secondary", c.Ramps.Secondary},
		{"Tertiary", c.Ramps.Tertiary},
		{"Success", c.Ramps.Success},
		{"Warning", c.Ramps.Warning},
		{"Error", c.Ramps.Error},
	}
	return func(gtx layout.Context) layout.Dimensions {
		cs := make([]layout.FlexChild, 0, 2*len(ramps))
		for i, r := range ramps {
			r := r
			if i > 0 {
				cs = append(cs, layout.Rigid(complayout.VSpacer(6)))
			}
			cs = append(cs, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						gtx.Constraints.Min.X = gtx.Dp(80)
						gtx.Constraints.Max.X = gtx.Dp(80)
						return labelAt(gtx, inv.shaper, r.name, c.Text, 12, font.Font{})
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						w := gtx.Dp(40)
						h := gtx.Dp(22)
						for n := 0; n < 9; n++ {
							off := op.Offset(image.Pt(n*w, 0)).Push(gtx.Ops)
							paint.FillShape(gtx.Ops, r.ramp.Step((n+1)*100),
								clip.Rect{Max: image.Pt(w, h)}.Op())
							// Every step gets its own hairline: the first one
							// or two sit within a shade of the page itself,
							// and unbordered they read as a ramp that starts
							// short rather than as steps that are nearly the
							// ground.
							swatchBorder(gtx, c.Ramps.Neutral.Step(400), image.Pt(w, h), 1)
							off.Pop()
						}
						return layout.Dimensions{Size: image.Pt(9*w, h)}
					}),
				)
			}))
		}
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx, cs...)
	}
}

func (inv *inventory) typeScale(c tokens.ColorTokens) layout.Widget {
	typo := tokens.DefaultTypography
	// The whole ladder, not a sample of it: a role that is not on the page
	// is a role nobody judges the theme on.
	roles := []struct {
		name  string
		style tokens.TextStyle
	}{
		{"DisplayLarge", typo.DisplayLarge},
		{"DisplayMedium", typo.DisplayMedium},
		{"DisplaySmall", typo.DisplaySmall},
		{"HeadlineLarge", typo.HeadlineLarge},
		{"HeadlineMedium", typo.HeadlineMedium},
		{"HeadlineSmall", typo.HeadlineSmall},
		{"TitleLarge", typo.TitleLarge},
		{"TitleMedium", typo.TitleMedium},
		{"TitleSmall", typo.TitleSmall},
		{"BodyLarge", typo.BodyLarge},
		{"BodyMedium", typo.BodyMedium},
		{"BodySmall", typo.BodySmall},
		{"LabelLarge", typo.LabelLarge},
		{"LabelMedium", typo.LabelMedium},
		{"LabelSmall", typo.LabelSmall},
		{"Code", typo.Code},
	}
	return func(gtx layout.Context) layout.Dimensions {
		cs := make([]layout.FlexChild, 0, len(roles))
		for _, r := range roles {
			r := r
			cs = append(cs, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Alignment: layout.Baseline}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						gtx.Constraints.Min.X = gtx.Dp(170)
						gtx.Constraints.Max.X = gtx.Dp(170)
						return labelAt(gtx, inv.shaper,
							fmt.Sprintf("%s · %gsp", r.name, r.style.Size),
							c.Ramps.Neutral.Step(600), 11, font.Font{})
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						// The token scale numbers weights the way CSS does —
						// 400 regular, 700 bold — and gio's is centred on
						// zero, so the two have to be converted and not
						// assigned: gio reads a raw 500 as Black.
						f := font.Font{
							Typeface: font.Typeface(r.style.Typeface),
							Weight:   tokens.FontWeight(r.style.Weight),
						}
						return labelAt(gtx, inv.shaper, "Vibrant Gio", c.Text, unit.Sp(r.style.Size), f)
					}),
				)
			}))
		}
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx, cs...)
	}
}

// ── Components ────────────────────────────────────────────────────────────────

func (inv *inventory) components(c tokens.ColorTokens) []section {
	return []section{
		{name: "components-button", title: "Button — rest, hover, focus, press, disabled", height: 72,
			body: inv.buttonRow(c)},
		{name: "components-textfield", title: "Text field — rest, focused, disabled", height: 80,
			body: inv.textFieldRow(c)},
		{name: "components-checkbox", title: "Checkbox and radio — unset, set, focused, disabled", height: 72,
			body: inv.toggleRow(c)},
		{name: "components-dropdown", title: "Dropdown — closed, focused, open, disabled", height: 190,
			body: inv.dropdownRow(c)},
		{name: "components-list", title: "List — a virtual list with its scrollbar in the gutter", height: 180,
			body: inv.listBlock(c)},
		{name: "components-scrollbar", title: "Scrollbar — a standalone bar beside its content", height: 180,
			body: inv.scrollbarBlock(c)},
		{name: "components-scrollarea", title: "Scroll area — the edge dissolves while content is hidden past it", height: 76,
			body: inv.scrollAreaBlock(c)},
		{name: "components-richtext", title: "Rich text — weight, style, face, colour, size and links in one paragraph", height: 130,
			body: inv.richtextBlock(c)},
		{name: "components-icon", title: "Icon — a vector icon and the platform control marks", height: 76,
			body: inv.iconBlock(c)},
		{name: "components-layout", title: "Layout — rows, columns, spacers and insets", height: 120,
			body: inv.layoutBlock(c)},
	}
}

func (inv *inventory) buttonRow(c tokens.ColorTokens) layout.Widget {
	states := []struct {
		label string
		st    button.RenderState
	}{
		{"Rest", button.RenderState{}},
		{"Hover", button.RenderState{Hovered: true}},
		{"Focus", button.RenderState{Focused: true}},
		{"Press", button.RenderState{Pressed: true}},
		{"Disabled", button.RenderState{Disabled: true}},
	}
	return func(gtx layout.Context) layout.Dimensions {
		cs := make([]layout.FlexChild, 0, 2*len(states))
		for i, s := range states {
			s := s
			if i > 0 {
				cs = append(cs, layout.Rigid(complayout.HSpacer(12)))
			}
			cs = append(cs, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				gtx.Constraints.Min.X = gtx.Dp(120)
				gtx.Constraints.Max.X = gtx.Dp(120)
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
					layout.Rigid(button.Render(inv.shaper, s.label, c, tokens.Spacing, tokens.Radius,
						tokens.DefaultTypography.LabelLarge, tokens.Comfortable, s.st)),
				)
			}))
		}
		return layout.Flex{}.Layout(gtx, cs...)
	}
}

func (inv *inventory) textFieldRow(c tokens.ColorTokens) layout.Widget {
	states := []struct {
		label string
		st    input.RenderState
	}{
		{"Rest", input.RenderState{}},
		{"Focused", input.RenderState{Focused: true}},
		{"Disabled", input.RenderState{Disabled: true}},
	}
	return func(gtx layout.Context) layout.Dimensions {
		cs := make([]layout.FlexChild, 0, 2*len(states))
		for i, s := range states {
			s := s
			if i > 0 {
				cs = append(cs, layout.Rigid(complayout.HSpacer(16)))
			}
			cs = append(cs, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				gtx.Constraints.Min.X = gtx.Dp(200)
				gtx.Constraints.Max.X = gtx.Dp(200)
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return labelAt(gtx, inv.shaper, s.label, c.Ramps.Neutral.Step(600), 11, font.Font{})
					}),
					layout.Rigid(complayout.VSpacer(6)),
					layout.Rigid(input.Render(inv.shaper, "Placeholder…", c, tokens.Spacing, tokens.Radius,
						tokens.DefaultTypography.BodyLarge, tokens.Comfortable, s.st)),
				)
			}))
		}
		return layout.Flex{}.Layout(gtx, cs...)
	}
}

func (inv *inventory) toggleRow(c tokens.ColorTokens) layout.Widget {
	cells := []struct {
		label string
		w     layout.Widget
	}{
		{"Unchecked", input.RenderCheckbox(c, tokens.Spacing, tokens.Radius, input.CheckboxRenderState{})},
		{"Checked", input.RenderCheckbox(c, tokens.Spacing, tokens.Radius, input.CheckboxRenderState{Checked: true})},
		{"Focused", input.RenderCheckbox(c, tokens.Spacing, tokens.Radius, input.CheckboxRenderState{Focused: true})},
		{"Disabled", input.RenderCheckbox(c, tokens.Spacing, tokens.Radius, input.CheckboxRenderState{Disabled: true})},
		{"Unselected", input.RenderRadio(c, tokens.Spacing, tokens.Radius, input.RadioRenderState{})},
		{"Selected", input.RenderRadio(c, tokens.Spacing, tokens.Radius, input.RadioRenderState{Selected: true})},
		{"Focused", input.RenderRadio(c, tokens.Spacing, tokens.Radius, input.RadioRenderState{Focused: true})},
		{"Disabled", input.RenderRadio(c, tokens.Spacing, tokens.Radius, input.RadioRenderState{Disabled: true})},
	}
	return func(gtx layout.Context) layout.Dimensions {
		cs := make([]layout.FlexChild, 0, 2*len(cells))
		for i, cell := range cells {
			cell := cell
			if i > 0 {
				cs = append(cs, layout.Rigid(complayout.HSpacer(16)))
			}
			cs = append(cs, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				gtx.Constraints.Min.X = gtx.Dp(72)
				gtx.Constraints.Max.X = gtx.Dp(72)
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
					layout.Rigid(cell.w),
					layout.Rigid(complayout.VSpacer(6)),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return labelAt(gtx, inv.shaper, cell.label, c.Ramps.Neutral.Step(600), 11, font.Font{})
					}),
				)
			}))
		}
		return layout.Flex{}.Layout(gtx, cs...)
	}
}

func (inv *inventory) dropdownRow(c tokens.ColorTokens) layout.Widget {
	opts := []string{"Apple", "Banana", "Cherry"}
	states := []struct {
		label string
		st    input.DropdownRenderState
	}{
		{"Closed", input.DropdownRenderState{Options: opts}},
		{"Focused", input.DropdownRenderState{Options: opts, Focused: true}},
		{"Open", input.DropdownRenderState{Options: opts, Open: true, Selected: 1}},
		{"Disabled", input.DropdownRenderState{Options: opts, Disabled: true}},
	}
	return func(gtx layout.Context) layout.Dimensions {
		cs := make([]layout.FlexChild, 0, 2*len(states))
		for i, s := range states {
			s := s
			if i > 0 {
				cs = append(cs, layout.Rigid(complayout.HSpacer(16)))
			}
			cs = append(cs, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				gtx.Constraints.Min.X = gtx.Dp(160)
				gtx.Constraints.Max.X = gtx.Dp(160)
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return labelAt(gtx, inv.shaper, s.label, c.Ramps.Neutral.Step(600), 11, font.Font{})
					}),
					layout.Rigid(complayout.VSpacer(6)),
					layout.Rigid(input.RenderDropdown(inv.shaper, c, tokens.Spacing, tokens.Radius,
						tokens.DefaultTypography.BodyLarge, tokens.Comfortable, s.st)),
				)
			}))
		}
		return layout.Flex{}.Layout(gtx, cs...)
	}
}

// rowHeight is what one row of the list and scrollbar sections measures. The
// sections are sized at an exact multiple of it: a virtual list clips whatever
// row the viewport ends inside, and a row cut through the middle of its
// letters reads as a drawing fault rather than as more content below.
const rowHeight = unit.Dp(36)

// textRow draws s centred in a row of exactly [rowHeight].
func (inv *inventory) textRow(gtx layout.Context, s string, c tokens.ColorTokens) layout.Dimensions {
	h := gtx.Dp(rowHeight)
	gtx.Constraints.Min.Y, gtx.Constraints.Max.Y = h, h
	complayout.InsetXY(0, 9).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return labelAt(gtx, inv.shaper, s, c.Text, 14, font.Font{})
	})
	return layout.Dimensions{Size: image.Pt(gtx.Constraints.Max.X, h)}
}

func (inv *inventory) listBlock(c tokens.ColorTokens) layout.Widget {
	bar := scrollbar.FromTokens(c)
	return func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Max.X = min(gtx.Constraints.Max.X, gtx.Dp(520))
		gtx.Constraints.Min = gtx.Constraints.Max
		return list.LayoutScrollbar(gtx, inv.listSt, bar, list.Occupy, inv.rows,
			func(gtx layout.Context, item string) layout.Dimensions {
				return inv.textRow(gtx, item, c)
			})
	}
}

func (inv *inventory) scrollbarBlock(c tokens.ColorTokens) layout.Widget {
	style := scrollbar.FromTokens(c)
	return func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Max.X = min(gtx.Constraints.Max.X, gtx.Dp(520))
		gtx.Constraints.Min = gtx.Constraints.Max
		h := gtx.Constraints.Max.Y
		barW := gtx.Dp(style.Width())
		// The list is laid out first so the bar reads this frame's position
		// rather than lagging one behind it.
		return layout.Flex{}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				gtx.Constraints.Max.X -= barW
				gtx.Constraints.Min = gtx.Constraints.Max
				return inv.barList.Layout(gtx, len(inv.bars), func(gtx layout.Context, i int) layout.Dimensions {
					return inv.textRow(gtx, inv.bars[i], c)
				})
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				start, end := scrollbar.FromListPosition(inv.barList.Position, len(inv.bars), h)
				dims := style.Layout(gtx, inv.barSt, layout.Vertical, start, end)
				if d := inv.barSt.ScrollDistance(); d != 0 {
					inv.barList.ScrollBy(d * float32(len(inv.bars)))
				}
				return dims
			}),
		)
	}
}

func (inv *inventory) scrollAreaBlock(c tokens.ColorTokens) layout.Widget {
	style := scrollarea.FromTokens(c)
	bar := scrollbar.FromTokens(c)
	return func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Max.X = min(gtx.Constraints.Max.X, gtx.Dp(520))
		// The area rests mid-content rather than at its start, so both
		// edges dissolve and the treatment is a treatment rather than one
		// hard cut and one gradient.
		if inv.areaSt.Overflows() && inv.areaSt.Offset() == 0 {
			inv.areaSt.SetOffset(inv.areaSt.MaxOffset() / 2)
		}
		return style.LayoutScrollbar(gtx, inv.areaSt, bar, func(gtx layout.Context) layout.Dimensions {
			const pitch, mark = 26, 18
			width := gtx.Dp(1100)
			for x := 0; x < width; x += gtx.Dp(pitch) {
				paint.FillShape(gtx.Ops, c.Primary,
					clip.Rect(image.Rect(x, gtx.Dp(8), min(x+gtx.Dp(mark), width), gtx.Dp(48))).Op())
			}
			return layout.Dimensions{Size: image.Pt(width, gtx.Dp(56))}
		})
	}
}

func (inv *inventory) richtextBlock(c tokens.ColorTokens) layout.Widget {
	style := richtext.FromTokens(c, tokens.DefaultTypography.BodyLarge)
	spans := []richtext.SpanStyle{
		{Content: "Rich text lays out "},
		{Content: "bold", Weight: font.Bold},
		{Content: ", "},
		{Content: "italic", Style: font.Italic},
		{Content: ", "},
		{Content: "monospace", Typeface: "Roboto Mono"},
		{Content: ", "},
		{Content: "coloured", Color: c.Error},
		{Content: " and "},
		{Content: "resized", Size: 22},
		{Content: " runs in one wrapped paragraph, with links to "},
		{Content: "gioui.org", URL: "https://gioui.org"},
		{Content: " and the "},
		{Content: "design system", URL: "https://github.com/vibrantgio"},
		{Content: "."},
	}
	return func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Max.X = min(gtx.Constraints.Max.X, gtx.Dp(520))
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(richtext.Render(inv.shaper, style, spans, richtext.Idle())),
			layout.Rigid(complayout.VSpacer(10)),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return labelAt(gtx, inv.shaper, "Link states: idle, hovered, focused.", c.Ramps.Neutral.Step(600), 11, font.Font{})
			}),
			layout.Rigid(complayout.VSpacer(4)),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				short := []richtext.SpanStyle{
					{Content: "Read the "},
					{Content: "documentation", URL: "https://gioui.org/doc"},
					{Content: " — hovered."},
				}
				return richtext.Render(inv.shaper, style, short,
					richtext.RenderState{HoveredLink: 0, FocusedLink: richtext.NoLink})(gtx)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				short := []richtext.SpanStyle{
					{Content: "Read the "},
					{Content: "documentation", URL: "https://gioui.org/doc"},
					{Content: " — focused."},
				}
				return richtext.Render(inv.shaper, style, short,
					richtext.RenderState{HoveredLink: richtext.NoLink, FocusedLink: 0})(gtx)
			}),
		)
	}
}

func (inv *inventory) iconBlock(c tokens.ColorTokens) layout.Widget {
	marks := []struct {
		name  string
		which icons.Name
	}{
		{"sidebar", icons.Sidebar},
		{"disclosure", icons.Disclosure},
		{"back", icons.HistoryBack},
		{"forward", icons.HistoryForward},
	}
	return func(gtx layout.Context) layout.Dimensions {
		cell := func(name string, w layout.Widget) layout.FlexChild {
			return layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				gtx.Constraints.Min.X = gtx.Dp(84)
				gtx.Constraints.Max.X = gtx.Dp(84)
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
					layout.Rigid(w),
					layout.Rigid(complayout.VSpacer(8)),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return labelAt(gtx, inv.shaper, name, c.Ramps.Neutral.Step(600), 11, font.Font{})
					}),
				)
			})
		}
		cs := []layout.FlexChild{cell("vector icon", inv.vectorIcon(c.Text))}
		for _, m := range marks {
			m := m
			cs = append(cs, cell(m.name, func(gtx layout.Context) layout.Dimensions {
				size := gtx.Dp(20)
				off := op.Offset(image.Pt(0, gtx.Dp(10))).Push(gtx.Ops)
				inv.marks.Mark(m.which)(gtx, size, c.Text)
				off.Pop()
				return layout.Dimensions{Size: image.Pt(size, gtx.Dp(40))}
			}))
		}
		return layout.Flex{}.Layout(gtx, cs...)
	}
}

func (inv *inventory) layoutBlock(c tokens.ColorTokens) layout.Widget {
	box := func(fill color.NRGBA, dp float32) layout.Widget {
		return func(gtx layout.Context) layout.Dimensions {
			sz := image.Pt(gtx.Dp(unit.Dp(dp)), gtx.Dp(unit.Dp(dp)))
			paint.FillShape(gtx.Ops, fill, clip.Rect{Max: sz}.Op())
			return layout.Dimensions{Size: sz}
		}
	}
	// Each primitive is captioned with the call that made it. Four clusters
	// of coloured squares with nothing under them say only that something
	// was drawn.
	cluster := func(caption string, w layout.Widget) layout.FlexChild {
		return layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
				layout.Rigid(w),
				layout.Rigid(complayout.VSpacer(8)),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return labelAt(gtx, inv.shaper, caption, c.Ramps.Neutral.Step(600), 11, font.Font{})
				}),
			)
		})
	}
	return func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Alignment: layout.End}.Layout(gtx,
			cluster("Row + HSpacer(8)", func(gtx layout.Context) layout.Dimensions {
				return complayout.Row(gtx,
					box(c.Primary, 40), complayout.HSpacer(8),
					box(c.Secondary, 40), complayout.HSpacer(8),
					box(c.Tertiary, 40),
				)
			}),
			layout.Rigid(complayout.HSpacer(32)),
			cluster("Col + VSpacer(6)", func(gtx layout.Context) layout.Dimensions {
				return complayout.Col(gtx,
					box(c.Success, 24), complayout.VSpacer(6),
					box(c.Warning, 24), complayout.VSpacer(6),
					box(c.Error, 24),
				)
			}),
			layout.Rigid(complayout.HSpacer(32)),
			cluster("Inset(16)", func(gtx layout.Context) layout.Dimensions {
				return complayout.Inset(16).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return box(c.Ramps.Primary.Step(400), 40)(gtx)
				})
			}),
			layout.Rigid(complayout.HSpacer(16)),
			cluster("InsetXY(24, 8)", func(gtx layout.Context) layout.Dimensions {
				return complayout.InsetXY(24, 8).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return box(c.Ramps.Secondary.Step(400), 40)(gtx)
				})
			}),
		)
	}
}

// ── Shared drawing helpers ────────────────────────────────────────────────────

// labelAt draws a one-line label with the gallery's shaper. It is the free
// function behind [gallery.label], so that a section can be drawn without a
// gallery — a golden test has no window and no page.
func labelAt(gtx layout.Context, shaper *text.Shaper, s string, col color.NRGBA, size unit.Sp, f font.Font) layout.Dimensions {
	m := op.Record(gtx.Ops)
	paint.ColorOp{Color: col}.Add(gtx.Ops)
	mat := m.Stop()
	lbl := widget.Label{MaxLines: 1}
	return lbl.Layout(gtx, shaper, f, size, s, mat)
}

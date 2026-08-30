// Package inventory draws every published family of the design system —
// foundations, components, patterns and a prose sample — as one long column
// of labelled sections.
//
// It exists to be looked at whole, because that is the only way a theme can
// be judged: a seed that flatters a button in isolation can still leave the
// tag row muddy against the card it sits on, and nothing but the two side by
// side will say so.
//
// Every section is a pure function of the [tokens.ColorTokens] it is handed.
// Nothing here reads a default palette, so the same code draws the whole
// surface in either scheme, a caller can push a generated palette through it
// and see the result on the next frame, and a test can capture a section
// without a window. What the sections keep across frames — scroll positions,
// the parsed reading sample, the rasterised icon — hangs off the [Inventory]
// value, which is built once and outlives any number of palettes.
package inventory

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
	"github.com/vibrantgio/markdown/highlight"
	"github.com/vibrantgio/theme/tokens"

	"github.com/vibrantgio/components/button"
	"github.com/vibrantgio/components/chip"
	ivgraster "github.com/vibrantgio/ivg/raster/gio"
)

// Section is one labelled block of the inventory: a heading and the widget
// that shows the family under it.
type Section struct {
	// Name identifies the section, uniquely across the whole inventory. It
	// is what a stored image of the section is filed under.
	Name string
	// Title is the heading shown above the body.
	Title string
	// Body draws the family. It is bounded by its own slot, never by the
	// page: several patterns expand into whatever constraints they are
	// given, and an unbounded one would take the column with it.
	Body layout.Widget
	// Height is the slot the body is laid out in, in dp. It is what the
	// body measures on its own, so that the row's margin is the whole of
	// the distance between one family and the next: a slot cut short runs
	// the family into the heading below it, and a slot left long opens a
	// hole under it that no other section has. A body that expands into
	// whatever it is handed measures nothing of its own, and its slot is
	// the size the family is worth showing at.
	Height unit.Dp
}

// Group collects the sections that come from one module.
type Group struct {
	Name     string
	Sections []Section
}

// Inventory owns everything the sections need to survive across frames and
// builds the sections from it. Build one and keep it: a palette change is a
// new set of section values, never a new Inventory.
//
// The reading sample is parsed once here and never per frame — and never per
// palette. The document is content, not colour: its style comes in at layout
// time, so a new palette re-styles the parsed form rather than re-reading the
// source. Re-parsing would also silently orphan every scroll and hover
// position the document holds, which it keys on block pointers.
type Inventory struct {
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
	// black, so each ink gets it recoloured — and rasterising is not
	// something to redo every frame. Keying on the ink rather than clearing
	// the cache is what lets a palette change cost one raster instead of one
	// per frame afterwards.
	ivg map[color.NRGBA]layout.Widget

	doc  *markdown.Document
	code *markdown.Document

	// The syntax palettes the code sections are drawn in — one per
	// appearance. Which member reaches the fence is the appearance's to say;
	// see Inventory.wear.
	codeBases highlight.BasePair

	// typo is the type roles the reading and code sections draw through.
	// Empty is DefaultTypography. A caller that names a code face calls
	// SetTypography; the parsed documents stay.
	typo tokens.Typography

	// The dismissible chips' close targets, so the specimens are real ones
	// the pointer and the keyboard can reach rather than drawings of a
	// mark. Their clicks are drained and dropped: an inventory that let a
	// specimen dismiss itself would leave a hole where the family it
	// demonstrates used to be.
	tagDismiss [2]widget.Clickable
}

// SetTypography names the type roles the reading and code sections draw
// through. The parsed documents stay; only Code and the extra faces a
// named code face appends change at the next layout. The empty value is
// DefaultTypography.
func (inv *Inventory) SetTypography(t tokens.Typography) { inv.typo = t }

// SetShaper replaces the shaper the inventory draws with, without
// rebuilding the parsed documents. A code-face change needs the matching
// collection; the documents do not.
func (inv *Inventory) SetShaper(s *text.Shaper) {
	if s != nil {
		inv.shaper = s
	}
}

func (inv *Inventory) typography() tokens.Typography {
	if inv.typo.Code.Typeface == "" {
		return tokens.DefaultTypography
	}
	return inv.typo
}

// New builds the inventory, with the platform control marks the host draws.
func New(shaper *text.Shaper) *Inventory { return NewForOS(shaper, runtime.GOOS) }

// NewForOS is New with the platform control marks pinned to goos rather than
// taken from the host. The marks are per-platform by design — a sidebar mark
// is drawn the way its platform draws it — so a capture meant to come out the
// same bytes on every machine has to name the platform it is of.
func NewForOS(shaper *text.Shaper, goos string) *Inventory {
	inv := &Inventory{
		shaper:  shaper,
		typo:    tokens.DefaultTypography,
		listSt:  list.NewState(),
		barList: layout.List{Axis: layout.Vertical},
		barSt:   scrollbar.NewState(),
		areaSt:  scrollarea.NewState(),
		marks:   icons.New(goos),
		reg:     icon.New(),
		doc:     markdown.NewDocument(markdown.Parse([]byte(readingSample))),
		code:    markdown.NewDocument(markdown.Parse([]byte(codeSample))),
	}
	inv.rows = make([]string, 40)
	inv.bars = make([]string, 40)
	for i := range inv.rows {
		inv.rows[i] = fmt.Sprintf("Row %d — a virtual list lays out only what shows", i+1)
		inv.bars[i] = fmt.Sprintf("Line %d of %d — content the bar beside it measures", i+1, len(inv.rows))
	}
	inv.reg.Register("info", icon.FromIVG(ActionInfoIVG))
	inv.ivg = map[color.NRGBA]layout.Widget{}
	return inv
}

// vectorIcon returns the registered vector icon drawn in ink.
func (inv *Inventory) vectorIcon(ink color.NRGBA) layout.Widget {
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

// Groups returns the whole inventory in the given scheme, in the order the
// column shows it: what a theme is made of first, then the widgets built on
// it, then the compositions, then prose.
func (inv *Inventory) Groups(c tokens.ColorTokens) []Group {
	return []Group{
		{Name: "Foundations", Sections: inv.Foundations(c)},
		{Name: "Components", Sections: inv.Components(c)},
		{Name: "Patterns", Sections: inv.Patterns(c)},
		{Name: "Markdown", Sections: inv.Reading(c)},
	}
}

// ── Foundations ───────────────────────────────────────────────────────────────

// Foundations returns the sections a theme is made of: the semantic roles,
// the functional ramps and the whole type ladder.
func (inv *Inventory) Foundations(c tokens.ColorTokens) []Section {
	return []Section{
		{
			Name: "foundations-roles", Title: "Palette — the scheme's semantic roles", Height: 60,
			Body: inv.roleSwatches(c),
		},
		{
			Name: "foundations-ramps", Title: "Palette — the functional ramps, nine steps each", Height: 218,
			Body: inv.rampSwatches(c),
		},
		{
			Name: "foundations-type", Title: "Typography — every role a surface reads in", Height: 442,
			Body: inv.typeScale(c),
		},
	}
}

func (inv *Inventory) roleSwatches(c tokens.ColorTokens) layout.Widget {
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
		{"Info", c.Info, c.OnInfo, "Aa"},
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
							return LabelAt(gtx, inv.shaper, s.label, s.on, 13, font.Font{})
						})
					}),
					layout.Rigid(complayout.VSpacer(6)),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return LabelAt(gtx, inv.shaper, s.name, c.Text, 11, font.Font{})
					}),
				)
			}))
		}
		return layout.Flex{}.Layout(gtx, cs...)
	}
}

func (inv *Inventory) rampSwatches(c tokens.ColorTokens) layout.Widget {
	ramps := []struct {
		name string
		ramp tokens.Ramp
	}{
		{"Neutral", c.Ramps.Neutral},
		{"Primary", c.Ramps.Primary},
		{"Secondary", c.Ramps.Secondary},
		{"Tertiary", c.Ramps.Tertiary},
		{"Info", c.Ramps.Info},
		{"Success", c.Ramps.Success},
		{"Warning", c.Ramps.Warning},
		{"Error", c.Ramps.Error},
	}
	return func(gtx layout.Context) layout.Dimensions {
		cs := make([]layout.FlexChild, 0, 2*len(ramps))
		for i, r := range ramps {
			if i > 0 {
				cs = append(cs, layout.Rigid(complayout.VSpacer(6)))
			}
			cs = append(cs, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						gtx.Constraints.Min.X = gtx.Dp(80)
						gtx.Constraints.Max.X = gtx.Dp(80)
						return LabelAt(gtx, inv.shaper, r.name, c.Text, 12, font.Font{})
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

func (inv *Inventory) typeScale(c tokens.ColorTokens) layout.Widget {
	typo := inv.typography()
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
			cs = append(cs, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Alignment: layout.Baseline}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						gtx.Constraints.Min.X = gtx.Dp(170)
						gtx.Constraints.Max.X = gtx.Dp(170)
						return LabelAt(gtx, inv.shaper,
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
						return LabelAt(gtx, inv.shaper, "Vibrant Gio", c.Text, unit.Sp(r.style.Size), f)
					}),
				)
			}))
		}
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx, cs...)
	}
}

// ── Components ────────────────────────────────────────────────────────────────

// Components returns one section per component family, each showing the
// family in every state it has.
func (inv *Inventory) Components(c tokens.ColorTokens) []Section {
	return []Section{
		{Name: "components-button", Title: "Button — rest, hover, focus, press, disabled", Height: 36,
			Body: inv.buttonRow(c)},
		{Name: "components-button-pinned", Title: "Button — the register's own fill, and one pinned from outside the palette", Height: 36,
			Body: inv.pinnedButtonRow(c)},
		{Name: "components-chip", Title: "Chip — the clickable face, the pop-up anchor and the badge, on three storeys", Height: chipBlockH,
			Body: inv.chipBlock(c)},
		{Name: "components-textfield", Title: "Text field — rest, focused, disabled", Height: 60,
			Body: inv.textFieldRow(c)},
		{Name: "components-checkbox", Title: "Checkbox and radio — unset, set, focused, disabled", Height: 56,
			Body: inv.toggleRow(c)},
		{Name: "components-dropdown", Title: "Dropdown — closed, focused, open, disabled", Height: 180,
			Body: inv.dropdownRow(c)},
		{Name: "components-list", Title: "List — a virtual list with its scrollbar in the gutter", Height: 180,
			Body: inv.listBlock(c)},
		{Name: "components-scrollbar", Title: "Scrollbar — a standalone bar beside its content", Height: 180,
			Body: inv.scrollbarBlock(c)},
		{Name: "components-scrollarea", Title: "Scroll area — the edge dissolves while content is hidden past it", Height: 56,
			Body: inv.scrollAreaBlock(c)},
		{Name: "components-richtext", Title: "Rich text — weight, style, face, colour, size and links in one paragraph", Height: 127,
			Body: inv.richtextBlock(c)},
		{Name: "components-icon", Title: "Icon — a vector icon and the platform control marks", Height: 62,
			Body: inv.iconBlock(c)},
		{Name: "components-layout", Title: "Layout — rows, columns, spacers and insets", Height: 106,
			Body: inv.layoutBlock(c)},
	}
}

// buttonCell is one button in a row, its label doubling as the button's own
// text — which is what lets a row of them be read without a caption under
// each.
type buttonCell struct {
	label string
	st    button.RenderState
}

// ButtonCellW is the width one cell of a button row is laid out at, and
// ButtonCellGap the space between two of them. Stated because a test that
// looks at one cell of a row has to know where that cell begins.
const (
	ButtonCellW   unit.Dp = 120
	ButtonCellGap unit.Dp = 12
)

func (inv *Inventory) buttonRow(c tokens.ColorTokens) layout.Widget {
	return inv.buttonCells(c, []buttonCell{
		{"Rest", button.RenderState{}},
		{"Hover", button.RenderState{Hovered: true}},
		{"Focus", button.RenderState{Focused: true}},
		{"Press", button.RenderState{Pressed: true}},
		{"Disabled", button.RenderState{Disabled: true}},
	})
}

func (inv *Inventory) buttonCells(c tokens.ColorTokens, cells []buttonCell) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		cs := make([]layout.FlexChild, 0, 2*len(cells))
		for i, s := range cells {
			if i > 0 {
				cs = append(cs, layout.Rigid(complayout.HSpacer(float32(ButtonCellGap))))
			}
			cs = append(cs, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				gtx.Constraints.Min.X = gtx.Dp(ButtonCellW)
				gtx.Constraints.Max.X = gtx.Dp(ButtonCellW)
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
					layout.Rigid(button.Render(inv.shaper, s.label, c, tokens.Spacing, tokens.Radius,
						tokens.DefaultTypography.LabelLarge, tokens.Comfortable, s.st)),
				)
			}))
		}
		return layout.Flex{}.Layout(gtx, cs...)
	}
}

// PinnedFill and PinnedInk are the pair the pinned specimen wears: a fixed
// red, and the ink that reads over it. They are ordinary colour values and
// not tokens, which is the whole of what this row has to say — an action
// whose colour is chosen by its meaning rather than by the palette hands the
// button its fill, and the button wears it in both schemes while everything
// around it inverts. They are exported so the assertion that this row holds
// still can name the very colour it is looking for.
var (
	PinnedFill = color.NRGBA{R: 0xb3, G: 0x26, B: 0x1e, A: 0xff}
	PinnedInk  = color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}
)

// pinnedButtonRow puts the filled register's own pair beside a pinned one, at
// rest, so the two can be read against each other in one glance and in both
// schemes: the left cell is the palette's answer and moves with it, the right
// cell is the caller's and does not. Only the colours differ — the pinned
// button keeps the register's hover, press, focus and disabled treatments,
// which the button package's own goldens carry state by state.
func (inv *Inventory) pinnedButtonRow(c tokens.ColorTokens) layout.Widget {
	return inv.buttonCells(c, []buttonCell{
		{"Filled", button.RenderState{}},
		{"Pinned", button.RenderState{Fill: PinnedFill, OnFill: PinnedInk}},
	})
}

// The chip section's measurements. The pill is the density's control height,
// and the storey it stands on has to show all round it — a chip captured flush
// with the edge of its ground is a chip nobody can judge the rim of, which is
// the one thing the light scheme has to carry the pill with. So each storey is
// drawn as a panel with the chips inset inside it, and the section is exactly
// three of those plus the air between them.
const (
	chipPanelPadX unit.Dp = 16
	chipPanelPadY unit.Dp = 12
	chipRowGap    unit.Dp = 16
	chipChipGap   unit.Dp = 12
	chipCaptionW  unit.Dp = 108
)

// The section's own height, derived from the pill rather than chosen: the
// density says how tall a control is, and a slot written as a number would
// have to be re-guessed the day that changed.
var (
	chipPanelH = unit.Dp(tokens.Comfortable.ControlHeight) + 2*chipPanelPadY
	chipBlockH = 3*chipPanelH + 2*chipRowGap
)

// chipStoreys are the grounds the section shows the chip on, in the order the
// ladder stacks them: the paper a page is written on, a card raised over it,
// and a dialog floating above that. Three rather than one because the chip's
// whole colour model is relative — every colour it draws is derived from the
// storey it was handed — so a specimen on one ground says nothing about what
// the component does on another.
var chipStoreys = []struct {
	name   string
	ground tokens.ElevationLevel
}{
	{"On the paper", tokens.Level0},
	{"On a card", tokens.Level1},
	{"In a dialog", tokens.Level2},
}

// chipBlock draws all three faces of the chip on each of the three storeys:
// the clickable pill in the states the pointer puts it in, then the pop-up
// anchor, then the badge — the same geometry with no state walk, no ring and
// no cursor — closing each row.
//
// The three stand side by side because the section's whole job is telling them
// apart. The anchor is the one that differs in shape: the button's rounded
// rectangle rather than the pill, with the chevron pair the component draws
// itself. The badge carries no mark at all — the faces share a geometry on
// purpose and must not share a disclosure chevron, because a mark that
// promises a menu on the face that opens none is the one way to make them
// genuinely confusable.
func (inv *Inventory) chipBlock(c tokens.ColorTokens) layout.Widget {
	mark := chip.Glyph(inv.marks.Mark(icons.Disclosure))
	states := []struct {
		label string
		st    chip.RenderState
	}{
		{"Rest", chip.RenderState{}},
		{"Hover", chip.RenderState{Hovered: true}},
		{"Press", chip.RenderState{Pressed: true}},
		{"Focus", chip.RenderState{Focused: true}},
	}
	row := func(storey struct {
		name   string
		ground tokens.ElevationLevel
	}) layout.Widget {
		cs := make([]layout.FlexChild, 0, 2*len(states)+2)
		for _, s := range states {
			if len(cs) > 0 {
				cs = append(cs, layout.Rigid(complayout.HSpacer(float32(chipChipGap))))
			}
			st := s.st
			st.Ground = storey.ground
			cs = append(cs, layout.Rigid(chip.Render(inv.shaper, s.label, mark, c,
				tokens.Spacing, tokens.Radius, tokens.DefaultTypography.LabelLarge,
				tokens.Comfortable, st)))
		}
		cs = append(cs,
			layout.Rigid(complayout.HSpacer(24)),
			layout.Rigid(chip.RenderAnchor(inv.shaper, "Anchor", c,
				tokens.Spacing, tokens.Radius, tokens.DefaultTypography.LabelLarge,
				tokens.Comfortable, chip.RenderState{Ground: storey.ground})),
			layout.Rigid(complayout.HSpacer(float32(chipChipGap))),
			layout.Rigid(chip.RenderBadge(inv.shaper, "Badge", nil, c,
				tokens.Spacing, tokens.Radius, tokens.DefaultTypography.LabelLarge,
				tokens.Comfortable, storey.ground)),
		)
		// The caption stands INSIDE the band rather than beside it. A label
		// naming a ground while sitting on a different one is a label about
		// the row and not about the surface — a reviewer handed the section
		// read the three storeys as one loose row plus two table stripes for
		// exactly that reason.
		band := func(gtx layout.Context) layout.Dimensions {
			return complayout.InsetXY(float32(chipPanelPadX), float32(chipPanelPadY)).Layout(gtx,
				func(gtx layout.Context) layout.Dimensions {
					row := append([]layout.FlexChild{
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							gtx.Constraints.Min.X = gtx.Dp(chipCaptionW)
							gtx.Constraints.Max.X = gtx.Dp(chipCaptionW)
							return LabelAt(gtx, inv.shaper, storey.name, c.Ramps.Neutral.Step(600), 11, font.Font{})
						}),
					}, cs...)
					return layout.Flex{Alignment: layout.Middle}.Layout(gtx, row...)
				})
		}
		return storeyPanel(c.SurfaceAt(storey.ground), band)
	}
	return func(gtx layout.Context) layout.Dimensions {
		cs := make([]layout.FlexChild, 0, 2*len(chipStoreys))
		for i, storey := range chipStoreys {
			if i > 0 {
				cs = append(cs, layout.Rigid(complayout.VSpacer(float32(chipRowGap))))
			}
			cs = append(cs, layout.Rigid(row(storey)))
		}
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx, cs...)
	}
}

// storeyPanel draws content over a ground of its own, sized to what the
// content measured. The fill is painted after the content is recorded and
// replayed over it, because the panel's size is the content's and there is no
// way to know it before laying the content out.
func storeyPanel(fill color.NRGBA, content layout.Widget) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		m := op.Record(gtx.Ops)
		dims := content(gtx)
		call := m.Stop()
		paint.FillShape(gtx.Ops, fill, clip.Rect{Max: dims.Size}.Op())
		call.Add(gtx.Ops)
		return dims
	}
}

func (inv *Inventory) textFieldRow(c tokens.ColorTokens) layout.Widget {
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
			if i > 0 {
				cs = append(cs, layout.Rigid(complayout.HSpacer(16)))
			}
			cs = append(cs, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				gtx.Constraints.Min.X = gtx.Dp(200)
				gtx.Constraints.Max.X = gtx.Dp(200)
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return LabelAt(gtx, inv.shaper, s.label, c.Ramps.Neutral.Step(600), 11, font.Font{})
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

func (inv *Inventory) toggleRow(c tokens.ColorTokens) layout.Widget {
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
						return LabelAt(gtx, inv.shaper, cell.label, c.Ramps.Neutral.Step(600), 11, font.Font{})
					}),
				)
			}))
		}
		return layout.Flex{}.Layout(gtx, cs...)
	}
}

func (inv *Inventory) dropdownRow(c tokens.ColorTokens) layout.Widget {
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
			if i > 0 {
				cs = append(cs, layout.Rigid(complayout.HSpacer(16)))
			}
			cs = append(cs, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				gtx.Constraints.Min.X = gtx.Dp(160)
				gtx.Constraints.Max.X = gtx.Dp(160)
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return LabelAt(gtx, inv.shaper, s.label, c.Ramps.Neutral.Step(600), 11, font.Font{})
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
func (inv *Inventory) textRow(gtx layout.Context, s string, c tokens.ColorTokens) layout.Dimensions {
	h := gtx.Dp(rowHeight)
	gtx.Constraints.Min.Y, gtx.Constraints.Max.Y = h, h
	complayout.InsetXY(0, 9).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return LabelAt(gtx, inv.shaper, s, c.Text, 14, font.Font{})
	})
	return layout.Dimensions{Size: image.Pt(gtx.Constraints.Max.X, h)}
}

func (inv *Inventory) listBlock(c tokens.ColorTokens) layout.Widget {
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

func (inv *Inventory) scrollbarBlock(c tokens.ColorTokens) layout.Widget {
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

func (inv *Inventory) scrollAreaBlock(c tokens.ColorTokens) layout.Widget {
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

func (inv *Inventory) richtextBlock(c tokens.ColorTokens) layout.Widget {
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
				return LabelAt(gtx, inv.shaper, "Link states: idle, hovered, focused.", c.Ramps.Neutral.Step(600), 11, font.Font{})
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

func (inv *Inventory) iconBlock(c tokens.ColorTokens) layout.Widget {
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
						return LabelAt(gtx, inv.shaper, name, c.Ramps.Neutral.Step(600), 11, font.Font{})
					}),
				)
			})
		}
		cs := []layout.FlexChild{cell("vector icon", inv.vectorIcon(c.Text))}
		for _, m := range marks {
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

func (inv *Inventory) layoutBlock(c tokens.ColorTokens) layout.Widget {
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
					return LabelAt(gtx, inv.shaper, caption, c.Ramps.Neutral.Step(600), 11, font.Font{})
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

// ActionInfoIVG is the vector icon the icon section draws — the Material
// action-info glyph, in IVG. It is exported because a surface showing the
// icon family close up wants the same glyph the inventory shows, and two
// copies of one blob would be two things to keep in step.
var ActionInfoIVG = []byte{
	0x89, 0x49, 0x56, 0x47, 0x02, 0x0a, 0x00, 0x50, 0x50, 0xb0, 0xb0, 0xc0,
	0x80, 0x58, 0xa0, 0xf5, 0x74, 0x58, 0x58, 0xf5, 0x74, 0x58, 0x80, 0x91,
	0xf5, 0x88, 0xa8, 0xa8, 0xa8, 0xa8, 0x0d, 0x77, 0xa8, 0x58, 0x80, 0x0d,
	0x8b, 0x58, 0x80, 0x58, 0xe3, 0x84, 0xbc, 0xe7, 0x78, 0xe8, 0x7c, 0xe7,
	0x88, 0xe9, 0x98, 0xe3, 0x80, 0x60, 0xe7, 0x78, 0xe9, 0x78, 0xe7, 0x88,
	0xe9, 0x88, 0xe1,
}

// LabelAt draws a one-line label. It takes its shaper rather than owning one,
// so a section can be drawn with whatever shaper the surface around it uses —
// including none of a window's, which is how a capture is made.
func LabelAt(gtx layout.Context, shaper *text.Shaper, s string, col color.NRGBA, size unit.Sp, f font.Font) layout.Dimensions {
	m := op.Record(gtx.Ops)
	paint.ColorOp{Color: col}.Add(gtx.Ops)
	mat := m.Stop()
	lbl := widget.Label{MaxLines: 1}
	return lbl.Layout(gtx, shaper, f, size, s, mat)
}

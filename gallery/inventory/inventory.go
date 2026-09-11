// Package inventory draws every published family of the design system —
// foundations, components, patterns and a prose sample — as one long column
// of labelled sections.
//
// It exists to be looked at whole, because that is the only way a theme can
// be judged: a fill that flatters a button in isolation can still leave the
// tag row muddy against the card it sits on, and nothing but the two side by
// side will say so.
//
// Every section is a pure function of the [tokens.PlatformColors] it is
// handed. Nothing here reads a default set, so the same code draws the whole
// surface in either appearance, a caller can push another set through it and
// see the result on the next frame, and a test can capture a section without
// a window. What the sections keep across frames — scroll positions, the
// parsed reading sample, the rasterised icon — hangs off the [Inventory]
// value, which is built once and outlives any number of sets.
package inventory

import (
	"fmt"
	"image"
	"image/color"
	"reflect"
	"runtime"

	"gioui.org/f32"
	"gioui.org/font"
	"gioui.org/io/semantic"
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
	"github.com/vibrantgio/components/pagination"
	"github.com/vibrantgio/components/paragraph"
	"github.com/vibrantgio/components/picker"
	"github.com/vibrantgio/components/scrollarea"
	"github.com/vibrantgio/components/scrollbar"
	"github.com/vibrantgio/components/toast"
	"github.com/vibrantgio/components/tooltip"
	"github.com/vibrantgio/markdown"
	"github.com/vibrantgio/markdown/highlight"
	vgcolor "github.com/vibrantgio/theme/color"
	"github.com/vibrantgio/theme/tokens"

	"github.com/vibrantgio/components/alert"
	"github.com/vibrantgio/components/badge"
	"github.com/vibrantgio/components/breadcrumb"
	"github.com/vibrantgio/components/button"
	"github.com/vibrantgio/components/chip"
	ivgraster "github.com/vibrantgio/ivg/raster/gio"
)

// Section is one labelled block of the inventory: a heading and the
// layout.Widget that shows the family under it.
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
// builds the sections from it. Build one and keep it: a change of set is a
// new slice of section values, never a new Inventory.
//
// The reading sample is parsed once here and never per frame — and never per
// set. The document is content, not colour: its style comes in at layout
// time, so a new set re-styles the parsed form rather than re-reading the
// source. Re-parsing would also silently orphan every scroll and hover
// position the document holds, which it keys on block pointers.
type Inventory struct {
	shaper *text.Shaper

	rows    []string
	bars    []string
	listSt  *list.State
	barList layout.List
	barSt   *scrollbar.State
	// The search specimen keeps its own list and bar so that scrolling the
	// plain scrollbar above it does not move it: the two sections stand side
	// by side in one column and answer different questions.
	foundList layout.List
	foundSt   *scrollbar.State
	areaSt    *scrollarea.State

	marks *icons.Set
	reg   *icon.Registry
	// ivg caches the rendered vector icon per foreground colour. The icon
	// carries its own palette, which on a dark surface would be a black disc
	// on black, so each colour gets it recoloured — and rasterising is not
	// something to redo every frame. Keying on the colour rather than clearing
	// the cache is what lets a change of set cost one raster instead of one
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

	// The dismissible badges' close targets, so the specimens are real ones
	// the pointer and the keyboard can reach rather than drawings of a
	// mark. Their clicks are drained and dropped: an inventory that let a
	// specimen dismiss itself would leave a hole where the family it
	// demonstrates used to be.
	badgeDismiss [2]widget.Clickable
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
		shaper:    shaper,
		typo:      tokens.DefaultTypography,
		listSt:    list.NewState(),
		barList:   layout.List{Axis: layout.Vertical},
		foundList: layout.List{Axis: layout.Vertical},
		foundSt:   scrollbar.NewState(),
		barSt:     scrollbar.NewState(),
		areaSt:    scrollarea.NewState(),
		marks:     icons.New(goos),
		reg:       icon.New(),
		doc:       markdown.NewDocument(markdown.Parse([]byte(readingSample))),
		code:      markdown.NewDocument(markdown.Parse([]byte(codeSample))),
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

// vectorIcon returns the registered vector icon drawn in `foreground`.
func (inv *Inventory) vectorIcon(foreground color.NRGBA) layout.Widget {
	if w, ok := inv.ivg[foreground]; ok {
		return w
	}
	blank := func(gtx layout.Context) layout.Dimensions {
		return layout.Dimensions{Size: image.Pt(gtx.Dp(40), gtx.Dp(40))}
	}
	w := blank
	if ic, ok := inv.reg.Icon("info"); ok {
		if built, err := ivgraster.Widget(ic.IVG(), 40, 40, ivgraster.WithColors(foreground)); err == nil {
			w = built
		}
	}
	inv.ivg[foreground] = w
	return w
}

// Groups returns the whole inventory in the given scheme, in the order the
// column shows it: what a theme is made of first, then the components built on
// it, then the compositions, then prose.
func (inv *Inventory) Groups(c tokens.PlatformColors) []Group {
	return []Group{
		{Name: "Foundations", Sections: inv.Foundations(c)},
		{Name: "Components", Sections: inv.Components(c)},
		{Name: "Patterns", Sections: inv.Patterns(c)},
		{Name: "Markdown", Sections: inv.Reading(c)},
	}
}

// ── The surfaces the column draws on ──────────────────────────────────────────

// SectionSurface is the opaque fill every section body is laid out on: the
// platform's content plane, which is what this column is a page of. It is
// stated rather than left implicit because the platform's labels, seams and
// overlays carry a coverage and not a colour, so anything drawn here has to
// name what it is composited onto.
func SectionSurface(c tokens.PlatformColors) color.NRGBA { return c.ControlBackground }

// ChromeSurface is the fill the column's own chrome rows carry — a group's
// banner, a section's heading, the closing line: the platform's chrome
// material, which every sidebar, toolbar and status bar on this platform
// wears. On macOS 26 it is the content's fill exactly in the light
// appearance, so those rows are told apart by their seams there and by the
// fill in the dark one.
func ChromeSurface(c tokens.PlatformColors) color.NRGBA { return c.SidebarMaterial }

// sectionText and sectionMuted are the two foregrounds a section body writes in: the
// platform's label and its secondary label, flattened onto the fill the
// section stands on in encoded sRGB, which is where the platform composites
// a coverage.
func sectionText(c tokens.PlatformColors) color.NRGBA {
	return vgcolor.Flatten(c.Label, SectionSurface(c))
}

func sectionMuted(c tokens.PlatformColors) color.NRGBA {
	return vgcolor.Flatten(c.SecondaryLabel, SectionSurface(c))
}

// ── Foundations ───────────────────────────────────────────────────────────────

// Foundations returns the sections a theme is made of: the platform's colour
// set, and the whole type scale.
func (inv *Inventory) Foundations(c tokens.PlatformColors) []Section {
	return []Section{
		{
			Name:   "foundations-platform",
			Title:  "Palette — the platform's colour set, every name in both appearances",
			Height: platformBlockH,
			Body:   inv.platformSet(c),
		},
		{
			Name: "foundations-type", Title: "Typography — every role a surface reads in", Height: 442,
			Body: inv.typeScale(c),
		},
	}
}

// PlatformRow is one field of the platform's colour set: the name AppKit
// knows it by, in the set's own Go casing, and the value it carries in each
// appearance.
type PlatformRow struct {
	Name  string
	Light color.NRGBA
	Dark  color.NRGBA
}

// PlatformRows returns the platform's colour set field by field, in the order
// the set declares them.
//
// The fields are read by reflection rather than listed, so a name added to
// the set appears on this page without an edit here. A name the gallery
// forgot to list would be a name nobody ever judges, which is the one failure
// a page like this cannot afford.
func PlatformRows() []PlatformRow {
	t := reflect.TypeOf(tokens.PlatformColors{})
	light := reflect.ValueOf(tokens.PlatformLight)
	dark := reflect.ValueOf(tokens.PlatformDark)
	rows := make([]PlatformRow, 0, t.NumField())
	for i := range t.NumField() {
		rows = append(rows, PlatformRow{
			Name:  t.Field(i).Name,
			Light: light.Field(i).Interface().(color.NRGBA),
			Dark:  dark.Field(i).Interface().(color.NRGBA),
		})
	}
	return rows
}

// Hex writes a colour the way the reference records it: six digits, and the
// coverage after them when the value carries one. A name that is a coverage
// rather than a colour — a label, a seam, an overlay — reads correctly only
// over what is beneath it, and the page says so by showing the coverage
// rather than hiding it in a swatch.
func Hex(v color.NRGBA) string {
	out := fmt.Sprintf("#%02x%02x%02x", v.R, v.G, v.B)
	if v.A != 0xff {
		out += fmt.Sprintf(" · %d%%", int(float64(v.A)/255*100+0.5))
	}
	return out
}

// The platform-set section's measurements. The name column is as wide as the
// longest field name in the set, the two appearance columns hold a swatch and
// the value beside it, and the row pitch is the caption's line box.
const (
	platformNameW  unit.Dp = 226
	platformValueW unit.Dp = 136
	platformSwatch unit.Dp = 22
	platformRowH   unit.Dp = 22
	platformColGap unit.Dp = 16
)

// platformBlockH is the section's slot: a heading row and one row per field
// of the set, derived from the set rather than written down, so a field added
// to it does not have to be counted by hand.
var platformBlockH = unit.Dp(len(PlatformRows())+1) * platformRowH

// platformSet draws the platform's colour set: every name the set carries, in
// the order it carries them, with the value each name has in both
// appearances.
//
// Both appearances on one row rather than one page per scheme. The set IS the
// pair — a name whose light and dark values cannot be read against each other
// is a name nobody can judge — and the alpha-carrying names in particular are
// only legible as a pair: labelColor is black at 0.85 in one and white at
// 0.85 in the other, which a page showing one appearance at a time never
// says.
//
// A coverage is shown over ITS OWN appearance's content plane, not over the
// surface this section happens to stand on: white at 0.85 is what a dark
// label is, and over the light page it would read as very nearly nothing.
// So the two columns carry the same two swatches in either appearance, and
// the value beside each is the recorded one, coverage and all.
func (inv *Inventory) platformSet(c tokens.PlatformColors) layout.Widget {
	rows := PlatformRows()
	head := sectionMuted(c)
	body := sectionText(c)
	edge := vgcolor.Flatten(c.Separator, SectionSurface(c))
	return func(gtx layout.Context) layout.Dimensions {
		cell := func(label string, w unit.Dp, col color.NRGBA) layout.FlexChild {
			return layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				gtx.Constraints.Min.X = gtx.Dp(w)
				gtx.Constraints.Max.X = gtx.Dp(w)
				return LabelAt(gtx, inv.shaper, label, col, 11, font.Font{})
			})
		}
		swatchCell := func(v, plane color.NRGBA) layout.FlexChild {
			return layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				sz := image.Pt(gtx.Dp(platformSwatch), gtx.Dp(platformSwatch)-2)
				paint.FillShape(gtx.Ops, vgcolor.Flatten(v, plane), clip.Rect{Max: sz}.Op())
				swatchBorder(gtx, edge, sz, 1)
				return layout.Dimensions{Size: sz}
			})
		}
		line := func(children ...layout.FlexChild) layout.FlexChild {
			return layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				gtx.Constraints.Min.Y = gtx.Dp(platformRowH)
				return layout.Flex{Alignment: layout.Middle}.Layout(gtx, children...)
			})
		}
		gap := func() layout.FlexChild { return layout.Rigid(complayout.HSpacer(float32(platformColGap))) }

		cs := make([]layout.FlexChild, 0, len(rows)+1)
		cs = append(cs, line(
			cell("Platform name", platformNameW, head), gap(),
			cell("", platformSwatch, head), gap(),
			cell("Light", platformValueW, head), gap(),
			cell("", platformSwatch, head), gap(),
			cell("Dark", platformValueW, head),
		))
		for _, r := range rows {
			r := r
			cs = append(cs, line(
				cell(r.Name, platformNameW, body), gap(),
				swatchCell(r.Light, tokens.PlatformLight.ControlBackground), gap(),
				cell(Hex(r.Light), platformValueW, body), gap(),
				swatchCell(r.Dark, tokens.PlatformDark.ControlBackground), gap(),
				cell(Hex(r.Dark), platformValueW, body),
			))
		}
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx, cs...)
	}
}

func (inv *Inventory) typeScale(c tokens.PlatformColors) layout.Widget {
	typo := inv.typography()
	// The whole scale, not a sample of it: a role that is not on the page
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
							sectionMuted(c), 11, font.Font{})
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
						return LabelAt(gtx, inv.shaper, "Vibrant Gio", sectionText(c), unit.Sp(r.style.Size), f)
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
func (inv *Inventory) Components(c tokens.PlatformColors) []Section {
	return []Section{
		{Name: "components-button", Title: "Button — rest, hover, focus, press, disabled", Height: 36,
			Body: inv.buttonRow(c)},
		{Name: "components-button-emphasis", Title: "Button — the three emphases at rest, and the icon-only face", Height: 36,
			Body: inv.emphasisButtonRow(c)},
		{Name: "components-button-pinned", Title: "Button — the theme's own fill, and one pinned from outside the set", Height: 36,
			Body: inv.pinnedButtonRow(c)},
		{Name: "components-chip", Title: "Chip — the four purposes on three surfaces, then rest, hover, press and focus", Height: chipBlockH,
			Body: inv.chipBlock(c)},
		{Name: "components-badge", Title: "Badge — the five statuses, the three utterances, the disc, and the close mark", Height: badgeBlockH,
			Body: inv.badgeBlock(c)},
		{Name: "components-alert", Title: "Alert — info, success, warning, error", Height: 248,
			Body: inv.alerts(c)},
		// The toast is drawn here as the signal alone. The cast shadow that
		// says it floats belongs to whatever places it, so it shows up in
		// the notifications specimen and not in this one.
		{Name: "components-toast", Title: "Toast — the transient message at every status role, over the surface it floats on", Height: 192,
			Body: inv.toasts(c)},
		// The slot stands the trigger in the middle and hangs the bubble
		// above it, so it has to hold the trigger's whole square plus what
		// hangs: a slot cut to the bubble alone shears it off at the band's
		// edge.
		{Name: "components-tooltip", Title: "Tooltip — shown above its trigger", Height: 96,
			Body: inv.tooltip(c)},
		{Name: "components-textfield", Title: "Text field — rest, focused, disabled", Height: 60,
			Body: inv.textFieldRow(c)},
		{Name: "components-searchfield", Title: "Search field — at rest, and holding a query with its clear mark", Height: 60,
			Body: inv.searchFieldRow(c)},
		{Name: "components-checkbox", Title: "Checkbox and radio — unset, set, focused, disabled", Height: 56,
			Body: inv.toggleRow(c)},
		{Name: "components-picker", Title: "Picker — the field closed, focused, open under its menu and disabled, then the chrome toolbar", Height: 180,
			Body: inv.pickerRow(c)},
		{Name: "components-breadcrumb", Title: "Breadcrumb — a trail back to the root", Height: 20,
			Body: inv.breadcrumb(c)},
		{Name: "components-pagination", Title: "Pagination — page four of nine", Height: 36,
			Body: inv.pagination(c)},
		{Name: "components-list", Title: "List — a virtual list with its scrollbar in the gutter", Height: 180,
			Body: inv.listBlock(c)},
		{Name: "components-scrollbar", Title: "Scrollbar — a standalone bar beside its content", Height: 180,
			Body: inv.scrollbarBlock(c)},
		{Name: "components-scrollbar-search", Title: "Scrollbar — while a search is on, where the matches lie in the content, the current one stronger", Height: 180,
			Body: inv.scrollbarSearchBlock(c)},
		{Name: "components-scrollarea", Title: "Scroll area — the edge dissolves while content is hidden past it", Height: 56,
			Body: inv.scrollAreaBlock(c)},
		{Name: "components-paragraph", Title: "Paragraph — weight, style, face, colour, size and links in one run of text", Height: 151,
			Body: inv.paragraphBlock(c)},
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
	// icon, when set, makes the cell the icon-only face: the glyph the button
	// draws in place of a label. Such a cell is laid out at the square the
	// density gives it rather than at the row's cell width — an icon button
	// stretched to a text cell's width is a text button with no text in it,
	// which is the one thing this face is not.
	icon func(gtx layout.Context, sizePx int, col color.NRGBA)
}

// ButtonCellW is the width one cell of a button row is laid out at, and
// ButtonCellGap the space between two of them. Stated because a test that
// looks at one cell of a row has to know where that cell begins.
const (
	ButtonCellW   unit.Dp = 120
	ButtonCellGap unit.Dp = 12
)

func (inv *Inventory) buttonRow(c tokens.PlatformColors) layout.Widget {
	return inv.buttonCells(c, []buttonCell{
		{label: "Rest", st: button.RenderState{}},
		{label: "Hover", st: button.RenderState{Hovered: true}},
		{label: "Focus", st: button.RenderState{Focused: true}},
		{label: "Press", st: button.RenderState{Pressed: true}},
		{label: "Disabled", st: button.RenderState{Disabled: true}},
	})
}

// emphasisButtonRow puts the three emphases side by side at rest, and the
// icon-only face after them.
//
// The three are shown at rest and at rest only. Emphasis is a question of
// prominence: how strongly a button sits on the page when nobody is touching
// it — the most pronounced one a surface is about, the middle one beside it,
// the least pronounced one that must be present without competing — and that
// is a judgement made on three still buttons next to each other. The state
// walk is the row above, which Filled already carries for all three.
//
// The name of each is the button's label, the way the state row's labels are
// its states: a caption under a button that already says "Tonal" is the same
// word twice.
//
// The icon face closes the row rather than standing in a section of its own,
// because it is the same button with a glyph where the label was — same
// emphasis, same corner, same target — and the one thing worth seeing about it
// is how its square sits beside the rectangles it is cut from. It is drawn at
// Filled emphasis so the square itself is visible; the ghost cell to its left
// already shows what a button with no fill at rest looks like.
func (inv *Inventory) emphasisButtonRow(c tokens.PlatformColors) layout.Widget {
	return inv.buttonCells(c, []buttonCell{
		{label: "Filled", st: button.RenderState{Emphasis: button.Filled}},
		{label: "Tonal", st: button.RenderState{Emphasis: button.Tonal}},
		{label: "Ghost", st: button.RenderState{Emphasis: button.Ghost}},
		{icon: inv.marks.Mark(icons.Sidebar), st: button.RenderState{Emphasis: button.Filled}},
	})
}

func (inv *Inventory) buttonCells(c tokens.PlatformColors, cells []buttonCell) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		cs := make([]layout.FlexChild, 0, 2*len(cells))
		for i, s := range cells {
			if i > 0 {
				cs = append(cs, layout.Rigid(complayout.HSpacer(float32(ButtonCellGap))))
			}
			if s.icon != nil {
				cs = append(cs, layout.Rigid(button.RenderIcon(s.icon, c, tokens.Spacing,
					tokens.Radius, tokens.Comfortable, s.st)))
				continue
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

// PinnedFill and PinnedForeground are the pair the pinned specimen wears: a fixed
// red, and the foreground that reads over it. They are ordinary colour values
// and not tokens, which is the whole of what this row has to say — an action
// whose colour is chosen by its meaning rather than by the set hands the
// button its fill, and it wears that colour in both schemes while everything
// around it inverts. They are exported so the assertion that this row holds
// still can name the very colour it is looking for.
var (
	PinnedFill       = color.NRGBA{R: 0xb3, G: 0x26, B: 0x1e, A: 0xff}
	PinnedForeground = color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}
)

// pinnedButtonRow puts the theme's own Filled pair beside a pinned one, at
// rest, so the two can be read against each other in one glance and in both
// appearances: the left cell is the set's answer and follows it, the right
// cell is the caller's and does not. Only the colours differ — the pinned
// button keeps Filled emphasis' hover, press, focus and disabled treatments,
// which the button package's own goldens carry state by state.
func (inv *Inventory) pinnedButtonRow(c tokens.PlatformColors) layout.Widget {
	return inv.buttonCells(c, []buttonCell{
		{label: "Filled", st: button.RenderState{}},
		{label: "Pinned", st: button.RenderState{Fill: PinnedFill, OnFill: PinnedForeground}},
	})
}

// The chip section's measurements. A resting chip is an outline and no fill,
// and the surface it stands on has to show all round it — a chip captured
// flush with the edge of that surface is a chip nobody can judge the rim of,
// which is the whole of what the light scheme has to carry it with. So each
// surface is drawn as a panel with the chips inset inside it, and the state
// rows stand on the page below them.
const (
	chipPanelPadX unit.Dp = 16
	chipPanelPadY unit.Dp = 12
	chipRowGap    unit.Dp = 16
	chipChipGap   unit.Dp = 12
	chipCaptionW  unit.Dp = 108
)

// The section's own height, derived from the chip rather than chosen: the
// density says how tall a chip is, and a slot written as a number would have
// to be re-guessed the day that changed.
var (
	chipH      = unit.Dp(tokens.Comfortable.ChipHeight())
	chipPanelH = chipH + 2*chipPanelPadY
	chipBlockH = 3*chipPanelH + 2*chipH + 4*chipRowGap
)

// chipSurfaces are the fills the section shows the chip on: the content a
// page is written on, the platform's grouped box, and the chrome material a
// sidebar or a toolbar wears. Three rather than one because the chip's rim,
// its focus ring and its press tint each carry a coverage rather than a
// colour, so each lands as whatever it is composited onto and a specimen on
// one fill says nothing about the others.
//
// On macOS 26 the content and the chrome material are one white in the light
// appearance, so two of the three panels read as one there; in the dark
// appearance the three are #1e1e1e, #2a3034 and #232a2e. That is the
// platform's own answer and not a fault of the specimen.
func chipSurfaces(c tokens.PlatformColors) []struct {
	name string
	fill color.NRGBA
} {
	return []struct {
		name string
		fill color.NRGBA
	}{
		{"On the content", c.ControlBackground},
		{"On a card", c.CardFill},
		{"In the chrome", c.SidebarMaterial},
	}
}

// chipPurposes are the four purposes a chip can be given, each drawn doing its
// own job and labelled as a caller would really label it: the assist chip
// offering an action behind a sign for it, the filter chip twice because
// selection is the one state only it has, the input chip with the avatar slot
// and the dismiss mark it always carries, and the suggestion chip as words
// alone. Placeholder labels would leave a reader unable to tell which of the
// four to reach for.
var chipPurposes = []struct {
	label    string
	purpose  chip.Purpose
	icon     chip.Glyph
	selected bool
}{
	{"Set reminder", chip.Assist, chipPlus, false},
	{"Unread", chip.Filter, nil, false},
	{"Starred", chip.Filter, nil, true},
	{"Olivia Barnes", chip.Input, chipAvatar, false},
	{"What's due today?", chip.Suggestion, nil, false},
}

// chipPlus is the sign the assist specimen leads with, drawn as a vector
// rather than rasterised from a font or an SVG so the stored images hold
// still. Its arms span the box the chip reserves edge to edge and it is
// stroked at the label's own stem width, which is what the Glyph contract asks
// now that the box is the label's cap band.
//
// The stem is taken in pixels WITHOUT rounding to a whole one, which is the
// difference between this sign and the marks the chip draws itself. Rounded to
// 2 px this sign lands on the pixel grid at full strength while the check and
// the cross beside it, being diagonals, spread the same weight over three
// columns — three marks in one row at three apparent weights. The measured
// relation is one weight for all of them, so all of them take the number
// unrounded.
func chipPlus(gtx layout.Context, sizePx int, col color.NRGBA) {
	w := float32(sizePx)
	stroke := chip.MarkStrokeDp(tokens.DefaultTypography.LabelLarge) * gtx.Metric.PxPerDp
	if stroke < 1 {
		stroke = 1
	}
	var p clip.Path
	p.Begin(gtx.Ops)
	p.MoveTo(f32.Pt(w*0.5, 0))
	p.LineTo(f32.Pt(w*0.5, w))
	p.MoveTo(f32.Pt(0, w*0.5))
	p.LineTo(f32.Pt(w, w*0.5))
	paint.FillShape(gtx.Ops, col, clip.Stroke{Path: p.End(), Width: stroke}.Op())
}

// chipAvatar is the picture the input specimen carries in its leading slot: a
// ring with a figure inside it, the stand-in every address field draws where a
// photograph has not loaded. The chip clips this box round, so the ring is
// drawn half a stroke inside the box and the figure is kept clear of the
// boundary the clip will cut.
func chipAvatar(gtx layout.Context, sizePx int, col color.NRGBA) {
	w := float32(sizePx)
	stroke := float32(gtx.Dp(unit.Dp(1.5)))
	if stroke < 1 {
		stroke = 1
	}
	inset := int(stroke/2 + 0.5)
	ring := clip.Ellipse{Min: image.Pt(inset, inset), Max: image.Pt(sizePx-inset, sizePx-inset)}
	paint.FillShape(gtx.Ops, col, clip.Stroke{Path: ring.Path(gtx.Ops), Width: stroke}.Op())
	head := clip.Ellipse{
		Min: image.Pt(int(w*0.36), int(w*0.22)),
		Max: image.Pt(int(w*0.64), int(w*0.50)),
	}
	paint.FillShape(gtx.Ops, col, head.Op(gtx.Ops))
	shoulders := clip.Ellipse{
		Min: image.Pt(int(w*0.24), int(w*0.60)),
		Max: image.Pt(int(w*0.76), int(w*0.96)),
	}
	paint.FillShape(gtx.Ops, col, shoulders.Op(gtx.Ops))
}

// chipBlock shows the four purposes in one row and the states under them: the
// purposes once per surface, then the same chip through what the pointer and
// the keyboard put it in, unselected and selected.
//
// The purposes are drawn once per surface, because the chip's rim, ring and
// press tint each carry a coverage and land as whatever they are composited
// onto, so a specimen on one fill says nothing about the others. The state rows below stand on the page: what they
// ask a reader to judge — whether the body that arrives under the pointer
// still holds its label, and whether the focus ring reads as the edge — is the
// same question on every surface, and asking it three times would bury the
// two rows that are not the same. Both rests are there because the two walk from
// different places: an unselected chip walks from the surface it stands on, a
// selected one from the container it wears.
func (inv *Inventory) chipBlock(c tokens.PlatformColors) layout.Widget {
	specimen := func(label string, purpose chip.Purpose, icon chip.Glyph, st chip.RenderState) layout.Widget {
		return chip.Render(inv.shaper, label, purpose, icon, c,
			tokens.Spacing, tokens.Radius, tokens.DefaultTypography.LabelLarge,
			tokens.Comfortable, st)
	}
	// The state rows label each chip with the state it is in, so the row reads
	// without a caption under every cell.
	states := []struct {
		label string
		st    chip.RenderState
	}{
		{"Rest", chip.RenderState{}},
		{"Hover", chip.RenderState{Hovered: true}},
		{"Press", chip.RenderState{Pressed: true}},
		{"Focus", chip.RenderState{Focused: true}},
	}
	stateRow := func(selected bool) []layout.Widget {
		cells := make([]layout.Widget, 0, len(states))
		for _, s := range states {
			st := s.st
			st.Selected = selected
			cells = append(cells, specimen(s.label, chip.Filter, nil, st))
		}
		return cells
	}
	rows := []struct {
		caption string
		cells   []layout.Widget
	}{
		{"Unselected", stateRow(false)},
		{"Selected", stateRow(true)},
	}
	// The caption stands inside the band rather than beside it. A label naming
	// a surface while sitting on a different one is a label about the row and
	// not about the surface.
	panel := func(lv struct {
		name string
		fill color.NRGBA
	}) layout.Widget {
		cells := make([]layout.Widget, 0, len(chipPurposes))
		for _, p := range chipPurposes {
			cells = append(cells, specimen(p.label, p.purpose, p.icon,
				chip.RenderState{Surface: lv.fill, Selected: p.selected}))
		}
		band := func(gtx layout.Context) layout.Dimensions {
			return complayout.InsetXY(float32(chipPanelPadX), float32(chipPanelPadY)).Layout(gtx,
				chipLine(inv, c, lv.name, cells))
		}
		return surfacePanel(lv.fill, band)
	}
	surfaces := chipSurfaces(c)
	return func(gtx layout.Context) layout.Dimensions {
		cs := make([]layout.FlexChild, 0, 2*(len(surfaces)+len(rows)))
		for _, lv := range surfaces {
			if len(cs) > 0 {
				cs = append(cs, layout.Rigid(complayout.VSpacer(float32(chipRowGap))))
			}
			cs = append(cs, layout.Rigid(panel(lv)))
		}
		for _, r := range rows {
			line := chipLine(inv, c, r.caption, r.cells)
			cs = append(cs, layout.Rigid(complayout.VSpacer(float32(chipRowGap))))
			cs = append(cs, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				// Indented by the panel's own padding so every caption in the
				// section starts at one x, panel or page.
				return complayout.InsetXY(float32(chipPanelPadX), 0).Layout(gtx, line)
			}))
		}
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx, cs...)
	}
}

// chipLine lays a captioned row of chips out: the caption in a fixed column so
// every row in the section starts at one x, then the cells across the
// section's own gap.
func chipLine(inv *Inventory, c tokens.PlatformColors, caption string, cells []layout.Widget) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		cs := make([]layout.FlexChild, 0, 2*len(cells)+1)
		cs = append(cs, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			gtx.Constraints.Min.X = gtx.Dp(chipCaptionW)
			gtx.Constraints.Max.X = gtx.Dp(chipCaptionW)
			return LabelAt(gtx, inv.shaper, caption, sectionMuted(c), 11, font.Font{})
		}))
		for i, cell := range cells {
			if i > 0 {
				cs = append(cs, layout.Rigid(complayout.HSpacer(float32(chipChipGap))))
			}
			cs = append(cs, layout.Rigid(cell))
		}
		return layout.Flex{Alignment: layout.Middle}.Layout(gtx, cs...)
	}
}

// surfacePanel draws content over a fill of its own, sized to what the
// content measured. The fill is painted after the content is recorded and
// replayed over it, because the panel's size is the content's and there is no
// way to know it before laying the content out.
func surfacePanel(fill color.NRGBA, content layout.Widget) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		m := op.Record(gtx.Ops)
		dims := content(gtx)
		call := m.Stop()
		paint.FillShape(gtx.Ops, fill, clip.Rect{Max: dims.Size}.Op())
		call.Add(gtx.Ops)
		return dims
	}
}

// The badge section's measurements. A badge pads its own content but nothing
// outside itself, so every number here belongs to the section rather than to
// the component: what separates two badges, what separates the rows, how much
// room the row's caption is given, how far in from the section's own margin
// the rows start, and the air the block keeps above the first row and below
// the last.
const (
	badgeRowGap   unit.Dp = 14
	badgeGap      unit.Dp = 20
	badgeCaptionW unit.Dp = 108
	badgeIndent   unit.Dp = 16
	badgeBandPad  unit.Dp = 10
)

// The section's own height, derived from the badge rather than chosen: the
// type role's line box is the whole of a badge's height, and a slot written as
// a number would have to be re-guessed the day the type scale moved.
var (
	badgeLineBox = unit.Dp(badge.Style(tokens.DefaultTypography, tokens.Comfortable).LineHeight)
	badgeBlockH  = 5*badgeLineBox + 4*badgeRowGap + 2*badgeBandPad
)

// The badge section shows the vocabulary on the same three fills the chip
// section does — see [chipSurfaces]. Three rather than one because a bare
// badge wears no fill of its own: its sign, its close mark and that mark's
// hover and press overlays all carry a coverage and land as whatever they
// are composited onto, so a specimen on one fill says nothing about the
// others.

// badgeCheck is the verdict sign the badge specimens draw, as a vector rather
// than a font or SVG rasterisation so the stored images hold still. Its stroke
// spans most of the box it is handed and is centred on it, which is what the
// Glyph contract asks: the badge reserves the box, and a sign that under-fills
// it reads as a gap in the line.
func badgeCheck(gtx layout.Context, sizePx int, col color.NRGBA) {
	w := float32(sizePx)
	var p clip.Path
	p.Begin(gtx.Ops)
	p.MoveTo(f32.Pt(w*0.16, w*0.52))
	p.LineTo(f32.Pt(w*0.42, w*0.76))
	p.LineTo(f32.Pt(w*0.84, w*0.24))
	paint.FillShape(gtx.Ops, col, clip.Stroke{Path: p.End(), Width: badgeGlyphStroke(gtx)}.Op())
}

// badgeCross is the check's opposite, the second sign the disc row needs: the
// two rows below the vocabulary are the obligation drawn, that a set of glyph
// badges differs in shape and not in hue alone.
//
// Its arms span the same 0.16 to 0.84 of the box the check does, so the two
// rows are one family: two signs that filled their boxes to different extents
// would read as two icon sets rather than as one shape varying.
func badgeCross(gtx layout.Context, sizePx int, col color.NRGBA) {
	w := float32(sizePx)
	var p clip.Path
	p.Begin(gtx.Ops)
	p.MoveTo(f32.Pt(w*0.16, w*0.16))
	p.LineTo(f32.Pt(w*0.84, w*0.84))
	p.MoveTo(f32.Pt(w*0.84, w*0.16))
	p.LineTo(f32.Pt(w*0.16, w*0.84))
	paint.FillShape(gtx.Ops, col, clip.Stroke{Path: p.End(), Width: badgeGlyphStroke(gtx)}.Op())
}

// badgeGlyphStroke is the verdict signs' stroke width in pixels, floored at
// one: a sub-pixel stroke leaves a smear rather than a sign.
func badgeGlyphStroke(gtx layout.Context) float32 {
	if s := float32(gtx.Dp(unit.Dp(1.5))); s >= 1 {
		return s
	}
	return 1
}

// badgeStyle is the type role every badge on the page is set in. The whole
// inventory is drawn at the comfortable density, so the badge's own density
// question has one answer here and is asked in one place.
func (inv *Inventory) badgeStyle() tokens.TextStyle {
	return badge.Style(inv.typography(), tokens.Comfortable)
}

// badgeBlock shows the vocabulary in one column: the four statuses and
// Neutral as the words they name, then the three utterances a badge can make,
// then the same five as discs under two signs, then the close mark through
// the states the pointer puts it in.
//
// Every row stands on the page, and no row is repeated on a second surface.
// On this platform a badge with a label wears its status's own colour, which
// is one value whatever is beneath it, and a bare sign is that colour read
// for the surface rather than derived from it — so a second panel of the same
// five comes out the same byte and promises an adaptation that is not there.
// A sheet labelling three rows with three surfaces and drawing one row three
// times is worse than one that draws the row once (fresh-eyes review,
// CE2.4b).
func (inv *Inventory) badgeBlock(c tokens.PlatformColors) layout.Widget {
	style := inv.badgeStyle()
	statuses := []struct {
		label  string
		status badge.Status
	}{
		{"Neutral", badge.Neutral},
		{"Success", badge.Success},
		{"Warning", badge.Warning},
		{"Error", badge.Error},
		{"Info", badge.Info},
	}
	plain := func(label string, glyph badge.Glyph, status badge.Status) layout.Widget {
		return badge.Render(inv.shaper, label, glyph, status, c, tokens.Spacing, tokens.Radius, style,
			badge.RenderState{})
	}
	// A disc is a glyph badge asked to stand on its status's fill instead of
	// bare. The fill is the same one the vocabulary panels above show on three
	// surfaces, so the disc rows are drawn on the page like the other
	// structure rows: what they ask a reader to judge is the shape, not a
	// derivation the panels already answer.
	discs := func(glyph badge.Glyph) []layout.Widget {
		cells := make([]layout.Widget, 0, len(statuses))
		for _, bs := range statuses {
			cells = append(cells, badge.Render(inv.shaper, "", glyph, bs.status, c,
				tokens.Spacing, tokens.Radius, style, badge.RenderState{Disc: true}))
		}
		return cells
	}

	// Each row varies one thing and each row varies a DIFFERENT thing, which
	// is what makes the dials readable as separate ones: hue across the
	// statuses, utterance across the row under them at one hue, hue again
	// across the two disc rows at one sign each. Drawing every row at one
	// status would leave a reader unable to tell whether the utterances
	// belong to that status or to the component.
	//
	// Exactly three cells stand in the utterance row, because there are
	// exactly three utterances. A sign set beside a word is a composition of
	// two of them and would read as a fourth. The disc is not a fourth
	// utterance either: it is the glyph utterance given the fill a word wears.
	rows := []struct {
		caption string
		cells   []layout.Widget
	}{
		{caption: "Statuses", cells: func() []layout.Widget {
			cells := make([]layout.Widget, 0, len(statuses))
			for _, bs := range statuses {
				cells = append(cells, plain(bs.label, nil, bs.status))
			}
			return cells
		}()},
		{caption: "Utterances", cells: []layout.Widget{
			plain("Popular", nil, badge.Success),
			plain("128", nil, badge.Success),
			plain("", badgeCheck, badge.Success),
		}},
		// Two sign rows rather than one, and the same five statuses across
		// both: the disc puts a field of the status's hue behind the sign
		// without making hue enough on its own, so a reader has to be able to
		// read the pair down the column where only the shape changed.
		{caption: "Disc, a check", cells: discs(badgeCheck)},
		{caption: "Disc, a cross", cells: discs(badgeCross)},
		{caption: "Dismissible", cells: []layout.Widget{
			// Real targets rather than drawings of a mark: the specimen is one
			// a pointer can reach. Their clicks are drained and dropped — an
			// inventory that let a specimen dismiss itself would leave a hole
			// where the family it demonstrates used to be.
			badge.RenderDismissible(inv.shaper, "Filtered by owner", nil, badge.Info,
				&inv.badgeDismiss[0], c, tokens.Spacing, tokens.Radius, style, badge.RenderState{}),
			badge.RenderDismissible(inv.shaper, "Hover", nil, badge.Info,
				&inv.badgeDismiss[1], c, tokens.Spacing, tokens.Radius, style,
				badge.RenderState{DismissHovered: true}),
			badge.RenderDismissible(inv.shaper, "Press", nil, badge.Info,
				nil, c, tokens.Spacing, tokens.Radius, style,
				badge.RenderState{DismissPressed: true}),
		}},
	}
	return func(gtx layout.Context) layout.Dimensions {
		cs := make([]layout.FlexChild, 0, 2*len(rows))
		for _, r := range rows {
			line := badgeLine(inv, c, r.caption, r.cells)
			if len(cs) > 0 {
				cs = append(cs, layout.Rigid(complayout.VSpacer(float32(badgeRowGap))))
			}
			cs = append(cs, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				// Indented so every caption in the section starts at one x.
				return complayout.InsetXY(float32(badgeIndent), 0).Layout(gtx, line)
			}))
		}
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx, cs...)
	}
}

// badgeLine lays a captioned row of specimens out: the caption in a fixed
// column so every row's badges start at one x, then the cells across the
// section's own gap.
func badgeLine(inv *Inventory, c tokens.PlatformColors, caption string, cells []layout.Widget) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		cs := make([]layout.FlexChild, 0, 2*len(cells)+1)
		cs = append(cs, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			gtx.Constraints.Min.X = gtx.Dp(badgeCaptionW)
			gtx.Constraints.Max.X = gtx.Dp(badgeCaptionW)
			return LabelAt(gtx, inv.shaper, caption, sectionMuted(c), 11, font.Font{})
		}))
		for i, cell := range cells {
			if i > 0 {
				cs = append(cs, layout.Rigid(complayout.HSpacer(float32(badgeGap))))
			}
			cs = append(cs, layout.Rigid(cell))
		}
		return layout.Flex{Alignment: layout.Middle}.Layout(gtx, cs...)
	}
}

func (inv *Inventory) alerts(c tokens.PlatformColors) layout.Widget {
	statuses := []struct {
		title  string
		status alert.Status
	}{
		{"Deploy finished", alert.Info},
		{"All checks passed", alert.Success},
		{"Two goldens are stale", alert.Warning},
		{"The build could not start", alert.Error},
	}
	return func(gtx layout.Context) layout.Dimensions {
		cs := make([]layout.FlexChild, 0, 2*len(statuses))
		for i, st := range statuses {
			st := st
			if i > 0 {
				cs = append(cs, layout.Rigid(complayout.VSpacer(8)))
			}
			cs = append(cs, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				gtx.Constraints.Max.X = min(gtx.Constraints.Max.X, gtx.Dp(520))
				gtx.Constraints.Max.Y = gtx.Dp(56)
				gtx.Constraints.Min = gtx.Constraints.Max
				return alert.Render(inv.shaper, alert.Props{
					Status: st.status,
					Title:  st.title,
					Shaper: inv.shaper,
				}, c, tokens.Spacing, tokens.Radius, tokens.DefaultTypography.TitleMedium)(gtx)
			}))
		}
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx, cs...)
	}
}

// toasts draws one toast at every status. All four carry the same fill — the
// window's own plane, which is what every floating surface on this platform is
// filled with — so the leading edge is the only thing between them.
//
// They stand on a band of the platform's box fill rather than on the section's
// own plane. A toast IS the window's plane, and the content it floats over is
// that plane too, so a specimen laid straight on the page has no body at all:
// a coloured stripe and some text on nothing, in both appearances. The band is
// what a toast floats over standing in for itself; the shadow that does the
// same job in a running window belongs to whatever places the toast, so it
// shows in the notifications specimen and not in this one.
func (inv *Inventory) toasts(c tokens.PlatformColors) layout.Widget {
	statuses := []struct {
		status toast.Status
		text   string
	}{
		{toast.Info, "Info — the theme was reloaded."},
		{toast.Success, "Success — the theme was saved."},
		{toast.Warning, "Warning — contrast is below target."},
		{toast.Error, "Error — that image could not be read."},
	}
	column := func(gtx layout.Context) layout.Dimensions {
		cs := make([]layout.FlexChild, 0, 2*len(statuses))
		for i, st := range statuses {
			st := st
			if i > 0 {
				cs = append(cs, layout.Rigid(complayout.VSpacer(8)))
			}
			cs = append(cs, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				gtx.Constraints.Max.X = min(gtx.Constraints.Max.X, gtx.Dp(toast.WidthDp))
				gtx.Constraints.Max.Y = gtx.Dp(toast.MinHeightDp)
				gtx.Constraints.Min = image.Point{}
				return toast.Render(inv.shaper, toast.Props{
					Status: st.status,
					Text:   st.text,
					Shaper: inv.shaper,
				}, c, tokens.Spacing, tokens.Radius, tokens.DefaultTypography.LabelMedium)(gtx)
			}))
		}
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx, cs...)
	}
	return func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Max.X = min(gtx.Constraints.Max.X, gtx.Dp(toastBandW))
		return surfacePanel(c.CardFill, func(gtx layout.Context) layout.Dimensions {
			return complayout.InsetXY(float32(toastBandPad), float32(toastBandPad)).Layout(gtx, column)
		})(gtx)
	}
}

// The toast band's measurements: how wide the surface under the toasts runs
// and how much of it shows around them.
const (
	toastBandW   unit.Dp = toast.WidthDp + 2*toastBandPad
	toastBandPad unit.Dp = 12
)

func (inv *Inventory) textFieldRow(c tokens.PlatformColors) layout.Widget {
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
						return LabelAt(gtx, inv.shaper, s.label, sectionMuted(c), 11, font.Font{})
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

// searchFieldRow is the search field in the two states that separate it from
// the text field above it: empty, where the looking glass is all there is to
// see, and holding a query, where the clear mark has appeared beside it.
func (inv *Inventory) searchFieldRow(c tokens.PlatformColors) layout.Widget {
	states := []struct {
		label string
		st    input.RenderState
	}{
		{"Rest", input.RenderState{}},
		{"Typed", input.RenderState{Text: "meeting notes"}},
		{"Focused", input.RenderState{Focused: true, Text: "meeting notes"}},
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
						return LabelAt(gtx, inv.shaper, s.label, sectionMuted(c), 11, font.Font{})
					}),
					layout.Rigid(complayout.VSpacer(6)),
					layout.Rigid(input.RenderSearch(inv.shaper, "Search", c, tokens.Spacing, tokens.Radius,
						tokens.DefaultTypography.BodyLarge, tokens.Comfortable, s.st)),
				)
			}))
		}
		return layout.Flex{}.Layout(gtx, cs...)
	}
}

func (inv *Inventory) toggleRow(c tokens.PlatformColors) layout.Widget {
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
						return LabelAt(gtx, inv.shaper, cell.label, sectionMuted(c), 11, font.Font{})
					}),
				)
			}))
		}
		return layout.Flex{}.Layout(gtx, cs...)
	}
}

// The picker section's measurements.
const (
	// pickerFieldW is the width one field cell is laid out at. The field is a
	// form control that fills the width it is given, so the cell has to state
	// one — and it also fixes the menu, which the open field drops at its own
	// width.
	pickerFieldW unit.Dp = 160
	// pickerCellGap is the air between two cells of the row.
	pickerCellGap = 16
	// pickerCaptionGap is the drop from a cell's caption to the specimen
	// under it.
	pickerCaptionGap = 6
)

// pickerRow shows the picker's two triggers side by side: the form-variant
// field in each of its states, then the chrome-variant toolbar at rest.
//
// Both stand in one section because they are one component — the same
// pick-one-from-many drawn for a form and for the window's chrome — and telling
// the two variants apart is what a reader comes to this row for. The open
// field carries the third piece with it: the menu it floats beneath itself is
// the shared surface, so the section shows that surface without spending a
// cell on it.
//
// The toolbar trigger is sized by its value rather than pinned to the field's cell
// width. It is the platform's pop-up control, which is as wide as what it
// says; stretched to a form field's width it would be reporting a geometry
// the component does not have.
func (inv *Inventory) pickerRow(c tokens.PlatformColors) layout.Widget {
	opts := []string{"Apple", "Banana", "Cherry"}
	fields := []struct {
		label string
		st    picker.FieldState
	}{
		{"Closed", picker.FieldState{Options: opts}},
		{"Focused", picker.FieldState{Options: opts, Focused: true}},
		{"Open", picker.FieldState{Options: opts, Open: true, Selected: 1}},
		{"Disabled", picker.FieldState{Options: opts, Disabled: true}},
	}
	cell := func(label string, body layout.Widget) layout.Widget {
		return func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return LabelAt(gtx, inv.shaper, label, sectionMuted(c), 11, font.Font{})
				}),
				layout.Rigid(complayout.VSpacer(pickerCaptionGap)),
				layout.Rigid(body),
			)
		}
	}
	return func(gtx layout.Context) layout.Dimensions {
		cs := make([]layout.FlexChild, 0, 2*len(fields)+2)
		for _, f := range fields {
			if len(cs) > 0 {
				cs = append(cs, layout.Rigid(complayout.HSpacer(pickerCellGap)))
			}
			body := picker.RenderField(inv.shaper, c, tokens.Spacing, tokens.Radius,
				tokens.DefaultTypography.BodyLarge, tokens.Comfortable, f.st)
			if f.st.Open {
				body = inv.padByMenu(c, f.st, body)
			}
			w := cell(f.label, body)
			cs = append(cs, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				gtx.Constraints.Min.X = gtx.Dp(pickerFieldW)
				gtx.Constraints.Max.X = gtx.Dp(pickerFieldW)
				return w(gtx)
			}))
		}
		return layout.Flex{}.Layout(gtx, append(cs,
			layout.Rigid(complayout.HSpacer(pickerCellGap)),
			layout.Rigid(cell("Toolbar", picker.RenderToolbar(inv.shaper, opts[0], c,
				c.SidebarMaterial, tokens.Spacing, tokens.Radius,
				tokens.DefaultTypography.LabelLarge, tokens.Comfortable,
				picker.ToolbarState{}))))...)
	}
}

// padByMenu reports the open field's cell as the trigger PLUS the menu the
// field floats under it, so the whole open surface stands inside this
// section's own slot.
//
// An open field reports its trigger and nothing else — the menu is a floating
// surface, deferred to the end of the frame and bounded by the window — so a
// cell cut to what the field reports would let the menu paint across the
// section below it, which is another family's row. The padding is the menu's
// measured height rather than a number, so the cell follows the density and
// the option list.
func (inv *Inventory) padByMenu(c tokens.PlatformColors, st picker.FieldState, body layout.Widget) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		dims := body(gtx)
		measure := op.Record(gtx.Ops)
		menu := picker.RenderMenu(inv.shaper, c, tokens.Spacing,
			tokens.DefaultTypography.BodyLarge, tokens.Comfortable,
			picker.MenuState{Options: st.Options, Selected: st.Selected})(gtx)
		measure.Stop()
		dims.Size.Y += menu.Size.Y
		return dims
	}
}

// rowHeight is what one row of the list and scrollbar sections measures. The
// sections are sized at an exact multiple of it: a virtual list clips whatever
// row the viewport ends inside, and a row cut through the middle of its
// letters reads as a drawing fault rather than as more content below.
const rowHeight = unit.Dp(36)

// textRow draws s centred in a row of exactly [rowHeight].
func (inv *Inventory) textRow(gtx layout.Context, s string, c tokens.PlatformColors) layout.Dimensions {
	h := gtx.Dp(rowHeight)
	gtx.Constraints.Min.Y, gtx.Constraints.Max.Y = h, h
	complayout.InsetXY(0, 9).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return LabelAt(gtx, inv.shaper, s, sectionText(c), 14, font.Font{})
	})
	return layout.Dimensions{Size: image.Pt(gtx.Constraints.Max.X, h)}
}

func (inv *Inventory) listBlock(c tokens.PlatformColors) layout.Widget {
	bar := scrollbar.FromTokens(c, SectionSurface(c))
	return func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Max.X = min(gtx.Constraints.Max.X, gtx.Dp(520))
		gtx.Constraints.Min = gtx.Constraints.Max
		return list.LayoutScrollbar(gtx, inv.listSt, bar, list.Occupy, inv.rows,
			func(gtx layout.Context, item string) layout.Dimensions {
				return inv.textRow(gtx, item, c)
			})
	}
}

func (inv *Inventory) scrollbarBlock(c tokens.PlatformColors) layout.Widget {
	style := scrollbar.FromTokens(c, SectionSurface(c))
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

// scrollbarSearchBlock is the bar of the section above it with a search on:
// the same content, and the places of five matches spread through the whole
// of it, the one the reader is on mid-track. The thumb is parked at the
// start, so the pair the section is for is visible at once — a match the
// thumb sits on is painted over it, and a match far below the viewport is
// painted all the same.
func (inv *Inventory) scrollbarSearchBlock(c tokens.PlatformColors) layout.Widget {
	style := scrollbar.FromTokens(c, SectionSurface(c))
	style.Matches = []float32{0.04, 0.23, 0.5, 0.71, 0.96}
	style.Current = 2
	return func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Max.X = min(gtx.Constraints.Max.X, gtx.Dp(520))
		gtx.Constraints.Min = gtx.Constraints.Max
		h := gtx.Constraints.Max.Y
		barW := gtx.Dp(style.Width())
		return layout.Flex{}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				gtx.Constraints.Max.X -= barW
				gtx.Constraints.Min = gtx.Constraints.Max
				return inv.foundList.Layout(gtx, len(inv.bars), func(gtx layout.Context, i int) layout.Dimensions {
					return inv.textRow(gtx, inv.bars[i], c)
				})
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				start, end := scrollbar.FromListPosition(inv.foundList.Position, len(inv.bars), h)
				dims := style.Layout(gtx, inv.foundSt, layout.Vertical, start, end)
				if d := inv.foundSt.ScrollDistance(); d != 0 {
					inv.foundList.ScrollBy(d * float32(len(inv.bars)))
				}
				return dims
			}),
		)
	}
}

func (inv *Inventory) scrollAreaBlock(c tokens.PlatformColors) layout.Widget {
	style := scrollarea.FromTokens(c)
	bar := scrollbar.FromTokens(c, SectionSurface(c))
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
				paint.FillShape(gtx.Ops, c.SystemBlue,
					clip.Rect(image.Rect(x, gtx.Dp(8), min(x+gtx.Dp(mark), width), gtx.Dp(48))).Op())
			}
			return layout.Dimensions{Size: image.Pt(width, gtx.Dp(56))}
		})
	}
}

func (inv *Inventory) paragraphBlock(c tokens.PlatformColors) layout.Widget {
	style := paragraph.FromTokens(c, tokens.DefaultTypography.BodyLarge, SectionSurface(c))
	spans := []paragraph.SpanStyle{
		{Content: "A paragraph lays out "},
		{Content: "bold", Weight: font.Bold},
		{Content: ", "},
		{Content: "italic", Style: font.Italic},
		{Content: ", "},
		{Content: "monospace", Typeface: "Roboto Mono"},
		{Content: ", "},
		{Content: "coloured", Color: c.SystemRed},
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
			layout.Rigid(paragraph.Render(inv.shaper, style, spans, paragraph.Idle())),
			layout.Rigid(complayout.VSpacer(10)),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return LabelAt(gtx, inv.shaper, "Link states: idle, hovered, focused.", sectionMuted(c), 11, font.Font{})
			}),
			layout.Rigid(complayout.VSpacer(4)),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				short := []paragraph.SpanStyle{
					{Content: "Read the "},
					{Content: "documentation", URL: "https://gioui.org/doc"},
					{Content: " — hovered."},
				}
				return paragraph.Render(inv.shaper, style, short,
					paragraph.RenderState{HoveredLink: 0, FocusedLink: paragraph.NoLink})(gtx)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				short := []paragraph.SpanStyle{
					{Content: "Read the "},
					{Content: "documentation", URL: "https://gioui.org/doc"},
					{Content: " — focused."},
				}
				return paragraph.Render(inv.shaper, style, short,
					paragraph.RenderState{HoveredLink: paragraph.NoLink, FocusedLink: 0})(gtx)
			}),
		)
	}
}

func (inv *Inventory) iconBlock(c tokens.PlatformColors) layout.Widget {
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
						return LabelAt(gtx, inv.shaper, name, sectionMuted(c), 11, font.Font{})
					}),
				)
			})
		}
		cs := []layout.FlexChild{cell("vector icon", inv.vectorIcon(sectionText(c)))}
		for _, m := range marks {
			cs = append(cs, cell(m.name, func(gtx layout.Context) layout.Dimensions {
				size := gtx.Dp(20)
				off := op.Offset(image.Pt(0, gtx.Dp(10))).Push(gtx.Ops)
				inv.marks.Mark(m.which)(gtx, size, sectionText(c))
				off.Pop()
				return layout.Dimensions{Size: image.Pt(size, gtx.Dp(40))}
			}))
		}
		return layout.Flex{}.Layout(gtx, cs...)
	}
}

func (inv *Inventory) layoutBlock(c tokens.PlatformColors) layout.Widget {
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
					return LabelAt(gtx, inv.shaper, caption, sectionMuted(c), 11, font.Font{})
				}),
			)
		})
	}
	return func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Alignment: layout.End}.Layout(gtx,
			cluster("Row + HSpacer(8)", func(gtx layout.Context) layout.Dimensions {
				return complayout.Row(gtx,
					box(c.SystemBlue, 40), complayout.HSpacer(8),
					box(c.SystemIndigo, 40), complayout.HSpacer(8),
					box(c.SystemTeal, 40),
				)
			}),
			layout.Rigid(complayout.HSpacer(32)),
			cluster("Col + VSpacer(6)", func(gtx layout.Context) layout.Dimensions {
				return complayout.Col(gtx,
					box(c.SystemGreen, 24), complayout.VSpacer(6),
					box(c.SystemOrange, 24), complayout.VSpacer(6),
					box(c.SystemRed, 24),
				)
			}),
			layout.Rigid(complayout.HSpacer(32)),
			cluster("Inset(16)", func(gtx layout.Context) layout.Dimensions {
				return complayout.Inset(16).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return box(c.SystemPurple, 40)(gtx)
				})
			}),
			layout.Rigid(complayout.HSpacer(16)),
			cluster("InsetXY(24, 8)", func(gtx layout.Context) layout.Dimensions {
				return complayout.InsetXY(24, 8).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return box(c.SystemPink, 40)(gtx)
				})
			}),
		)
	}
}

func (inv *Inventory) tooltip(c tokens.PlatformColors) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Max.X = min(gtx.Constraints.Max.X, gtx.Dp(320))
		gtx.Constraints.Min = gtx.Constraints.Max
		props := tooltip.Props{
			Text:      specimenName,
			Trigger:   inv.specimenControl(c),
			Placement: tooltip.Top,
			Shaper:    inv.shaper,
		}
		return tooltip.Render(inv.shaper, props, true, c, tokens.Spacing, tokens.Radius,
			tokens.DefaultTypography.LabelSmall)(gtx)
	}
}

func (inv *Inventory) breadcrumb(c tokens.PlatformColors) layout.Widget {
	props := breadcrumb.Props{
		Items: []breadcrumb.Item{
			{Label: "Design system"},
			{Label: "Components"},
			{Label: "Breadcrumb"},
		},
		Shaper: inv.shaper,
	}
	return breadcrumb.Render(inv.shaper, props, c, tokens.Spacing, tokens.DefaultTypography.TitleSmall)
}

func (inv *Inventory) pagination(c tokens.PlatformColors) layout.Widget {
	props := pagination.Props{Page: 4, PageCount: 9, Shaper: inv.shaper}
	return pagination.Render(inv.shaper, props, c, tokens.Spacing, tokens.Radius,
		tokens.DefaultTypography.LabelLarge, tokens.Comfortable)
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

// specimenName is the accessible name the trigger and the anchor both carry,
// and the words the tooltip shows. One string for both halves: an icon-only
// control has no label to fall back on, so the name a reader hovers for and
// the name a screen reader announces are the same name or they disagree.
const specimenName = "Show the sidebar"

// specimenControl is the trigger the tooltip cell and the anchor the popover
// cell hang their surface off: one Ghost icon-only button, identical in both,
// so the two cells differ by what is attached and not by what it attaches to.
//
// Drawn hovered. That is the state each cell is frozen in — a tooltip stands
// only while the pointer rests on its trigger — and it is what gives the
// popover's beak a fill to seat on: Ghost emphasis draws none at rest, and an
// apex aimed at a square with no fill points at empty air.
//
// The name is emitted as a semantic description because the label is empty by
// construction, and an icon-only button has no label to fall back on.
func (inv *Inventory) specimenControl(c tokens.PlatformColors) layout.Widget {
	w := button.RenderIcon(inv.marks.Mark(icons.Sidebar), c, tokens.Spacing,
		tokens.Radius, tokens.Comfortable,
		button.RenderState{Emphasis: button.Ghost, Hovered: true})
	return func(gtx layout.Context) layout.Dimensions {
		semantic.ClassOp(semantic.Button).Add(gtx.Ops)
		semantic.DescriptionOp(specimenName).Add(gtx.Ops)
		return w(gtx)
	}
}

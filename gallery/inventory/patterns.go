// The pattern half of the inventory: the compositions built on top of the
// components, each drawn from static state in a bounded slot.
//
// Every pattern here has a live twin that takes an observable theme and
// returns an observable layout.Widget. The gallery deliberately uses the
// static twin instead: it performs no input handling and schedules no
// invalidation, so a page can show all eighteen at once without eighteen event
// loops, and a golden test can capture one without a window.
package inventory

import (
	"image"
	"image/color"

	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"

	"github.com/vibrantgio/components/badge"
	"github.com/vibrantgio/components/button"
	complayout "github.com/vibrantgio/components/layout"
	"github.com/vibrantgio/components/toast"
	"github.com/vibrantgio/patterns/accordion"
	"github.com/vibrantgio/patterns/breadcrumb"
	"github.com/vibrantgio/patterns/card"
	"github.com/vibrantgio/patterns/feature"
	"github.com/vibrantgio/patterns/group"
	"github.com/vibrantgio/patterns/hero"
	"github.com/vibrantgio/patterns/modal"
	"github.com/vibrantgio/patterns/navbar"
	"github.com/vibrantgio/patterns/notifications"
	"github.com/vibrantgio/patterns/pagination"
	"github.com/vibrantgio/patterns/pane"
	"github.com/vibrantgio/patterns/popover"
	"github.com/vibrantgio/patterns/pricing"
	"github.com/vibrantgio/patterns/shell"
	patsidebar "github.com/vibrantgio/patterns/sidebar"
	"github.com/vibrantgio/patterns/table"
	"github.com/vibrantgio/patterns/tabs"
	"github.com/vibrantgio/patterns/testimonial"
	"github.com/vibrantgio/theme/tokens"
)

// Patterns returns one section per composition, each drawn from static state
// in a slot of its own.
func (inv *Inventory) Patterns(c tokens.ColorTokens) []Section {
	return []Section{
		// The column's slot is the four toasts and the gaps between them,
		// plus the reach of the cast shadow under the last of them: a slot
		// cut to the toasts alone would leave the shadow to fall across the
		// heading of the section below.
		{Name: "patterns-notifications", Title: "Notifications — the column, one toast at every status role", Height: 177,
			Body: inv.notifications(c)},
		{Name: "patterns-card", Title: "Card — one thing singled out, raised on the page it stands on", Height: 150,
			Body: inv.cards(c)},
		{Name: "patterns-group", Title: "Group — the page divided, a hairline at the surface's own level", Height: 150,
			Body: inv.groups(c)},
		{Name: "patterns-accordion", Title: "Accordion — one section open, the rest closed", Height: 240,
			Body: inv.accordion(c)},
		{Name: "patterns-tabs", Title: "Tabs — the second tab selected", Height: 130,
			Body: inv.tabs(c)},
		{Name: "patterns-breadcrumb", Title: "Breadcrumb — a trail back to the root", Height: 20,
			Body: inv.breadcrumb(c)},
		{Name: "patterns-pagination", Title: "Pagination — page four of nine", Height: 36,
			Body: inv.pagination(c)},
		{Name: "patterns-navbar", Title: "Navbar — brand, links and actions", Height: 52,
			Body: inv.navbar(c)},
		{Name: "patterns-sidebar", Title: "Sidebar — expanded beside its collapsed rail", Height: 210,
			Body: inv.sidebar(c)},
		{Name: "patterns-pane", Title: "Pane — chrome set in from the window's edges, with the backdrop on every side of it", Height: paneSpecimenH,
			Body: inv.pane(c)},
		{Name: "patterns-table", Title: "Table — sortable columns, sorted ascending on the first", Height: 176,
			Body: inv.table(c)},
		{Name: "patterns-modal", Title: "Modal — a decision answered from its footer, and a panel closed from the mark at its corner", Height: 260,
			Body: inv.modal(c)},
		// The slot stands the control in the middle and hangs the panel off
		// it, so it has to hold the control's whole square plus what hangs:
		// a slot cut to the panel alone shears the surface off at the
		// band's edge.
		{Name: "patterns-popover", Title: "Popover — a floating panel tied to its anchor", Height: 190,
			Body: inv.popover(c)},
		{Name: "patterns-hero", Title: "Hero — eyebrow, headline, subtitle and a pair of calls to action", Height: 208,
			Body: inv.hero(c)},
		{Name: "patterns-feature", Title: "Feature grid — three columns of icon, title and body", Height: 168,
			Body: inv.feature(c)},
		{Name: "patterns-pricing", Title: "Pricing — three tier groups, the middle one the recommended card", Height: 300,
			Body: inv.pricing(c)},
		{Name: "patterns-testimonial", Title: "Testimonial — a quote with its attribution", Height: 212,
			Body: inv.testimonial(c)},
		{Name: "patterns-shell", Title: "Shell — the three-column frame: sidebar, content, aside", Height: 300,
			Body: inv.shell(c)},
	}
}

// ── The content the patterns are filled with ──────────────────────────────────

// prose returns a layout.Widget that draws a few lines of body text, so a
// pattern's content slot holds something with a shape rather than a
// placeholder block.
func (inv *Inventory) prose(c tokens.ColorTokens, lines ...string) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		cs := make([]layout.FlexChild, 0, 2*len(lines))
		for i, line := range lines {
			line := line
			if i > 0 {
				cs = append(cs, layout.Rigid(complayout.VSpacer(4)))
			}
			cs = append(cs, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return LabelAt(gtx, inv.shaper, line, c.Text, 13, font.Font{})
			}))
		}
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx, cs...)
	}
}

// sized pins a layout.Widget to a width, which is what a pattern's action slot
// needs from a caller handing it a bare button.
func sized(w unit.Dp, child layout.Widget) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Min.X = gtx.Dp(w)
		gtx.Constraints.Max.X = gtx.Dp(w)
		return child(gtx)
	}
}

// dot returns a round icon slot in the given colour, the stand-in a pattern's
// Icon field takes when the gallery has no picture to put there.
func dot(fill color.NRGBA, size unit.Dp) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		d := gtx.Dp(size)
		r := clip.RRect{Rect: image.Rect(0, 0, d, d), SE: d / 2, SW: d / 2, NW: d / 2, NE: d / 2}
		paint.FillShape(gtx.Ops, fill, r.Op(gtx.Ops))
		return layout.Dimensions{Size: image.Pt(d, d)}
	}
}

// ── The patterns ──────────────────────────────────────────────────────────────

// notifications draws the column with one toast at every status role. A
// notification with a zero At does no fading, so the column stands still
// without a timer driving it — which is how the pattern's own stored images
// are made.
func (inv *Inventory) notifications(c tokens.ColorTokens) layout.Widget {
	items := []notifications.Notification{
		{ID: 1, Role: toast.Info, Text: "Info — the theme was reloaded."},
		{ID: 2, Role: toast.Success, Text: "Success — the seed was saved."},
		{ID: 3, Role: toast.Warning, Text: "Warning — contrast is below target."},
		{ID: 4, Role: toast.Error, Text: "Error — that image could not be read."},
	}
	return func(gtx layout.Context) layout.Dimensions {
		// The column gathers in a corner of the frame it is handed, one edge
		// margin in from it. A section's slot is not that frame: its own
		// margin already holds the specimen that distance off the page, so a
		// frame the size of the slot would indent the toasts by two margins
		// and drop them the same distance below the heading.
		//
		// So the frame is handed out one edge margin past the slot on the
		// leading and top sides, and the column drawn back into it. The
		// corner the toasts gather in is then the slot's own corner, and the
		// specimen lines up with the ones above and below it.
		edge := gtx.Dp(unit.Dp(tokens.Spacing.S4))
		defer op.Offset(image.Pt(-edge, -edge)).Push(gtx.Ops).Pop()
		gtx.Constraints.Max = gtx.Constraints.Max.Add(image.Pt(2*edge, 2*edge))
		gtx.Constraints.Min = gtx.Constraints.Max
		return notifications.Render(inv.shaper, notifications.Props{
			Position: notifications.TopLeft,
			Shaper:   inv.shaper,
		}, items, c, tokens.Spacing, tokens.Radius, tokens.DefaultTypography.LabelMedium)(gtx)
	}
}

func (inv *Inventory) cards(c tokens.ColorTokens) layout.Widget {
	header := func(gtx layout.Context) layout.Dimensions {
		return LabelAt(gtx, inv.shaper, "Recommended", c.Text, 15, font.Font{Weight: font.Bold})
	}
	body := inv.prose(c,
		"A card is raised one step",
		"on the surface it is in, and",
		"the raise is what singles it out.",
	)
	return func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Max.X = min(gtx.Constraints.Max.X, gtx.Dp(260))
		gtx.Constraints.Min = gtx.Constraints.Max
		return card.Render(card.Props{
			Header: header,
			Body:   body,
			// The badge's fill is derived against the card's own level
			// rather than the page's: the card stands at level 1, and the
			// developer's word about a card is a badge it carries.
			Footer: badge.Render(inv.shaper, "Popular", nil, badge.Neutral, c, tokens.Spacing,
				tokens.Radius, inv.badgeStyle(), badge.RenderState{Level: tokens.Level1}),
		}, c, tokens.Spacing, tokens.Radius)(gtx)
	}
}

// groups is the card's twin specimen: the same box at the same size, drawn
// as the other answer to the same question. The group takes the surface it
// is in, so nothing in the tile says where it is but the hairline — and it
// holds two things rather than one, because what a group is for is
// gathering related components, and a specimen holding one would not show
// the gap between them.
func (inv *Inventory) groups(c tokens.ColorTokens) layout.Widget {
	first := inv.prose(c,
		"A group draws a hairline at",
		"the level of the surface it is",
		"in, and raises nothing.",
	)
	second := inv.prose(c,
		"It holds related components.",
	)
	return func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Max.X = min(gtx.Constraints.Max.X, gtx.Dp(260))
		gtx.Constraints.Min = gtx.Constraints.Max
		return group.Render(inv.shaper, group.Props{
			Label:   "Density",
			Content: []layout.Widget{first, second},
		}, c, tokens.Spacing, tokens.Radius, tokens.DefaultTypography.LabelLarge)(gtx)
	}
}

func (inv *Inventory) accordion(c tokens.ColorTokens) layout.Widget {
	props := accordion.Props{
		Sections: []accordion.Section{
			{Title: "What the gallery shows", Body: inv.prose(c, "Every published family, in the current scheme.")},
			{Title: "How a section is bounded", Body: inv.prose(c, "Each one is laid out in a slot of its own.")},
			{Title: "Why the static twin", Body: inv.prose(c, "It handles no input and needs no event loop.")},
		},
		Shaper: inv.shaper,
	}
	open := map[int]bool{0: true}
	return func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Max.X = min(gtx.Constraints.Max.X, gtx.Dp(460))
		return accordion.Render(inv.shaper, props, open, c, tokens.Spacing,
			tokens.DefaultTypography.LabelLarge)(gtx)
	}
}

func (inv *Inventory) tabs(c tokens.ColorTokens) layout.Widget {
	props := tabs.Props{
		Tabs: []tabs.Tab{
			{Label: "Overview", Content: inv.prose(c, "The first tab's content.")},
			{Label: "Tokens", Content: inv.prose(c, "The selected tab's content shows below the strip.")},
			{Label: "History", Content: inv.prose(c, "The third tab's content.")},
		},
		Shaper: inv.shaper,
		// A specimen lifted off the page, like the table beside it: the
		// section body under it is the Background pin, so a panel taking the
		// pattern's default level would dissolve into the page and leave a
		// strip floating on nothing. On Level1 the panel keeps the Surface it
		// has always drawn and the strip stands one step over it.
		Level: tokens.Level1,
	}
	return func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Max.X = min(gtx.Constraints.Max.X, gtx.Dp(460))
		return tabs.Render(inv.shaper, props, 1, c, tokens.Spacing,
			tokens.DefaultTypography.LabelLarge, tokens.Comfortable)(gtx)
	}
}

func (inv *Inventory) breadcrumb(c tokens.ColorTokens) layout.Widget {
	props := breadcrumb.Props{
		Items: []breadcrumb.Item{
			{Label: "Design system"},
			{Label: "Patterns"},
			{Label: "Breadcrumb"},
		},
		Shaper: inv.shaper,
	}
	return breadcrumb.Render(inv.shaper, props, c, tokens.Spacing, tokens.DefaultTypography.TitleSmall)
}

func (inv *Inventory) pagination(c tokens.ColorTokens) layout.Widget {
	props := pagination.Props{Page: 4, PageCount: 9, Shaper: inv.shaper}
	return pagination.Render(inv.shaper, props, c, tokens.Spacing, tokens.Radius,
		tokens.DefaultTypography.LabelLarge, tokens.Comfortable)
}

func (inv *Inventory) navbarProps(c tokens.ColorTokens) navbar.Props {
	return navbar.Props{
		Brand: inv.prose(c, "Vibrant Gio"),
		Links: []navbar.Link{
			{Label: "Gallery", Active: true},
			{Label: "Tokens"},
			{Label: "Patterns"},
		},
		Actions: []layout.Widget{
			// The bar fills at the chrome level, and a badge with no fill of
			// its own is derived against whatever it stands on.
			badge.Render(inv.shaper, "v1", nil, badge.Neutral, c, tokens.Spacing,
				tokens.Radius, inv.badgeStyle(), badge.RenderState{Level: tokens.LevelChrome}),
		},
		Shaper: inv.shaper,
	}
}

func (inv *Inventory) navbar(c tokens.ColorTokens) layout.Widget {
	props := inv.navbarProps(c)
	return func(gtx layout.Context) layout.Dimensions {
		// The bar fills the height it is given, so it is pinned to the
		// density's own bar height rather than to the section's slot.
		gtx.Constraints.Max.Y = gtx.Dp(unit.Dp(tokens.Comfortable.ControlHeight + 2*tokens.Comfortable.PaddingY))
		gtx.Constraints.Min.Y = gtx.Constraints.Max.Y
		return navbar.Render(inv.shaper, props, c, tokens.Spacing,
			tokens.DefaultTypography.LabelLarge, tokens.Comfortable)(gtx)
	}
}

func (inv *Inventory) sidebarProps(c tokens.ColorTokens) patsidebar.Props {
	return patsidebar.Props{
		Items: []patsidebar.Item{
			{Icon: dot(c.Primary, 16), Label: "Everything", Active: true},
			{Icon: dot(c.Secondary, 16), Label: "Components"},
			{Icon: dot(c.Tertiary, 16), Label: "Patterns"},
			{Icon: dot(c.Success, 16), Label: "Markdown"},
		},
		Shaper: inv.shaper,
	}
}

func (inv *Inventory) sidebar(c tokens.ColorTokens) layout.Widget {
	props := inv.sidebarProps(c)
	return func(gtx layout.Context) layout.Dimensions {
		one := func(collapsed bool) layout.Widget {
			return patsidebar.Render(inv.shaper, props, collapsed, c, tokens.Spacing,
				tokens.DefaultTypography.LabelLarge, tokens.Comfortable)
		}
		return layout.Flex{}.Layout(gtx,
			layout.Rigid(one(false)),
			layout.Rigid(complayout.HSpacer(24)),
			layout.Rigid(one(true)),
		)
	}
}

// The pane specimen's measurements.
//
// The pane is chrome inside a window, not a component on a page: the inset,
// the corner radius and the internal hairline only say what they say when the
// backdrop shows around them. So the specimen draws a window of its own
// inside the slot, and the pane stands in that.
//
// The window needs no outline of its own: it paints the backdrop, which is a
// tint darker than everything the section row around it is painted in, so the
// darker rectangle is the window's edge.
const (
	paneSpecimenW unit.Dp = 560
	paneSpecimenH unit.Dp = 200
	// paneColumnW is the width the pane is asked for — a chrome column beside
	// a document rather than half the window, which is what the pattern's own
	// stored images show it at.
	paneColumnW unit.Dp = 168
	// paneGutter is the air between the pane's trailing edge and the document
	// that reflows beside it.
	paneGutter unit.Dp = 16
)

// pane draws the pane beside the content it stands over: the backdrop
// visible on every side of it, the rounded corners and the hairline just
// inside its own edge, and a document column starting where the pane stops.
//
// The content beside it is what makes the specimen a specimen. A pane alone
// in a box shows a rounded rectangle; a pane with a document reflowed against
// it shows the one thing the pattern is for — that the pane is an object
// standing on the window's own plane rather than an edge of it.
func (inv *Inventory) pane(c tokens.ColorTokens) layout.Widget {
	contents := func(gtx layout.Context) layout.Dimensions {
		// The strip at the top of the pane is the window buttons' band. This
		// specimen draws no window buttons — they belong to the window and
		// not to the pattern — so the pane's own content starts under it,
		// which is where a caller's content starts too.
		inset := gtx.Dp(12)
		defer op.Offset(image.Pt(inset, gtx.Dp(pane.StripDp))).Push(gtx.Ops).Pop()
		gtx.Constraints.Max.X -= 2 * inset
		return inv.prose(c,
			"Navigator",
			"",
			"Chrome the reader can",
			"send away, and what it",
			"stood over reflows.",
		)(gtx)
	}
	return func(gtx layout.Context) layout.Dimensions {
		// The specimen is a window, so it is drawn at the size a window would
		// show the pattern at — narrowed only when the column itself is
		// narrower than that, the way every other bounded specimen here is.
		size := image.Pt(min(gtx.Constraints.Max.X, gtx.Dp(paneSpecimenW)), gtx.Dp(paneSpecimenH))
		gtx.Constraints = layout.Exact(size)
		// The window's plane, which is what an inset pane stands on: nothing
		// is drawn at the backdrop and it shows wherever nothing stands.
		paint.FillShape(gtx.Ops, c.SurfaceAt(tokens.LevelBackdrop), clip.Rect{Max: size}.Op())

		b := pane.Bounds(gtx, size, paneColumnW, false)
		pane.Layout(gtx, c, b, contents)

		// The document, starting one gutter past the pane's trailing edge —
		// the reflow the pattern's hidden state completes by starting it at
		// the window's own edge instead.
		doc := gtx
		docW := max(0, size.X-b.Max.X-gtx.Dp(paneGutter+pane.MarginDp))
		doc.Constraints = layout.Exact(image.Pt(docW, size.Y-2*gtx.Dp(pane.MarginDp)))
		off := op.Offset(image.Pt(b.Max.X+gtx.Dp(paneGutter), gtx.Dp(pane.MarginDp))).Push(gtx.Ops)
		inv.prose(c,
			"The document beside it",
			"",
			"A pane stands one margin inside the",
			"window's leading, top and bottom edges,",
			"with the backdrop showing round it.",
			"",
			"It is at the chrome level, a step darker",
			"than the content: a pane is read through",
			"its edges and not through its lightness.",
		)(doc)
		off.Pop()

		return layout.Dimensions{Size: size}
	}
}

// tableRow is the shape the table section's rows take. A table is generic over
// its row type, so the gallery has to name one.
type tableRow struct {
	family string
	kind   string
	count  string
}

func (inv *Inventory) table(c tokens.ColorTokens) layout.Widget {
	cell := func(s string) layout.Widget {
		return table.RenderTextCell(inv.shaper, c, tokens.DefaultTypography.BodyMedium, s)
	}
	columns := []table.Column[tableRow]{
		{Header: "Family", Width: 160, Sortable: true, Cell: func(r tableRow) layout.Widget { return cell(r.family) }},
		{Header: "Kind", Width: 120, Sortable: true, Cell: func(r tableRow) layout.Widget { return cell(r.kind) }},
		{Header: "Sections", Width: 100, Cell: func(r tableRow) layout.Widget { return cell(r.count) }},
	}
	rows := []tableRow{
		{"Button", "Component", "1"},
		{"Card", "Pattern", "3"},
		{"Markdown", "Document", "9"},
		{"Toast", "Component", "4"},
	}
	return func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Max.X = min(gtx.Constraints.Max.X, gtx.Dp(400))
		return table.Render(inv.shaper, columns, rows, table.Sort{Column: 0, Asc: true},
			c, tokens.Spacing, tokens.DefaultTypography.LabelLarge, tokens.Comfortable)(gtx)
	}
}

// modalGap is the run of page surface between the two dialogs. It is wide
// enough that the two scrims read as two windows rather than as one scrim
// with two dialogs standing on it, and no wider: the pair is one specimen.
const modalGap = 24

func (inv *Inventory) modal(c tokens.ColorTokens) layout.Widget {
	// Two archetypes, side by side, because they are told apart by their
	// affordances and one of them alone shows only half of that. The
	// decision answers from its footer and carries no mark at its corner;
	// its scrim absorbs a stray click without acting on it, because
	// dismissal is one of the answers. The panel is a place, and every cheap
	// way out of it is offered at once: the mark, the key and the scrim.
	//
	// Neither is a modal you cannot leave from the keyboard — both bind the
	// dismissing key, which is why neither body below tells a reader to
	// reach for the pointer. Which archetype a caller gets is derived from
	// whether it states a decision, so the pair also shows that the mark and
	// the footer are not two checkboxes that could both be ticked.
	decision := modal.Props{
		Title: "Discard this theme?",
		Body: inv.prose(c,
			"The seed you extracted has not been saved.",
			"Discarding restores the default theme.",
		),
		Decision: &modal.Decision{Destructive: true},
		// The footer buttons are the caller's own layout.Widget values on both
		// the live and the static path, so a static dialog hands them over
		// already rendered rather than expecting the pattern to invent them.
		Actions: []layout.Widget{
			sized(96, button.Render(inv.shaper, "Cancel", c, tokens.Spacing, tokens.Radius,
				tokens.DefaultTypography.LabelLarge, tokens.Comfortable,
				button.RenderState{Emphasis: button.Tonal})),
			sized(96, button.Render(inv.shaper, "Discard", c, tokens.Spacing, tokens.Radius,
				tokens.DefaultTypography.LabelLarge, tokens.Comfortable, button.RenderState{})),
		},
		Shaper: inv.shaper,
	}
	// No Decision and no Actions: the panel's changes apply as they are
	// made, which is what leaves it nothing to ask and nothing to put in a
	// footer, and what lets it be left at any moment.
	//
	// Its body says that and stops. A line telling the reader which corner to
	// click would be a specimen's copy carrying its own layout: the mark
	// moves for a mirrored reading direction and the sentence would not, and
	// the panel has two other ways out that no corner names.
	panel := modal.Props{
		Title: "Theme settings",
		Body: inv.prose(c,
			"Every change here applies as you make it.",
			"Nothing to confirm, so there is no footer.",
		),
		Shaper: inv.shaper,
	}
	return func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{}.Layout(gtx,
			layout.Flexed(1, inv.modalScrim(c, decision)),
			layout.Rigid(complayout.HSpacer(modalGap)),
			layout.Flexed(1, inv.modalScrim(c, panel)),
		)
	}
}

// modalScrim draws one dialog over a scrim of its own.
//
// The scrim covers whatever box it is handed, and the box is a whole half of
// the section: a scrim that stops short of the bottom reads as a stray
// rectangle, not as a window under a dialog. A flexed child is handed no
// height of its own, so the height is taken here.
func (inv *Inventory) modalScrim(c tokens.ColorTokens, props modal.Props) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Min = gtx.Constraints.Max
		return modal.Render(inv.shaper, props, true, c, tokens.Spacing, tokens.Radius,
			tokens.DefaultTypography.TitleMedium, tokens.Comfortable)(gtx)
	}
}

func (inv *Inventory) popover(c tokens.ColorTokens) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Max.X = min(gtx.Constraints.Max.X, gtx.Dp(320))
		gtx.Constraints.Min = gtx.Constraints.Max
		props := popover.Props{
			Anchor: inv.specimenControl(c),
			// The prose says the placement this specimen is drawn at, not the
			// contract's four: a cell that says one thing and shows another
			// is the confusion these two cells exist to settle.
			Content:   inv.prose(c, "A popover holds content", "below what opened it."),
			Placement: popover.Bottom,
		}
		return popover.Render(props, true, c, tokens.Spacing, tokens.Radius)(gtx)
	}
}

func (inv *Inventory) hero(c tokens.ColorTokens) layout.Widget {
	props := hero.Props{
		Eyebrow:      "The design system",
		Title:        "Judge a theme whole",
		Subtitle:     "Every family on one page, re-rendered on the theme you are trying.",
		PrimaryCTA:   &hero.CTA{Label: "Try a seed"},
		SecondaryCTA: &hero.CTA{Label: "Read the docs"},
		Shaper:       inv.shaper,
	}
	return func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Max.X = min(gtx.Constraints.Max.X, gtx.Dp(640))
		return hero.Render(inv.shaper, props, c, tokens.Spacing, tokens.Radius,
			tokens.DefaultTypography, tokens.Comfortable)(gtx)
	}
}

func (inv *Inventory) feature(c tokens.ColorTokens) layout.Widget {
	props := feature.Props{
		Columns: 3,
		Items: []feature.Item{
			{Icon: dot(c.Primary, 24), Title: "One scale", Body: "Every ramp is generated from a single seed."},
			{Icon: dot(c.Secondary, 24), Title: "Two schemes", Body: "Light and dark come out of the same derivation."},
			{Icon: dot(c.Tertiary, 24), Title: "Measured contrast", Body: "Every reading pair is checked, not guessed."},
		},
		Shaper: inv.shaper,
	}
	return func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Max.X = min(gtx.Constraints.Max.X, gtx.Dp(640))
		return feature.Render(inv.shaper, props, c, tokens.Spacing, tokens.DefaultTypography)(gtx)
	}
}

func (inv *Inventory) pricing(c tokens.ColorTokens) layout.Widget {
	props := pricing.Props{
		Tiers: []pricing.Tier{
			{Name: "Sketch", Price: "Free", Cadence: "forever",
				Features: []string{"One seed", "Both schemes"}, CTA: &pricing.CTA{Label: "Start"}},
			{Name: "Studio", Price: "$12", Cadence: "per month", Recommended: true,
				Features: []string{"Unlimited seeds", "Both schemes", "Export"}, CTA: &pricing.CTA{Label: "Choose"}},
			{Name: "Team", Price: "$40", Cadence: "per month",
				Features: []string{"Everything in Studio", "Shared themes"}, CTA: &pricing.CTA{Label: "Contact"}},
		},
		Shaper: inv.shaper,
	}
	return func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Max.X = min(gtx.Constraints.Max.X, gtx.Dp(640))
		return pricing.Render(inv.shaper, props, c, tokens.Spacing, tokens.Radius,
			tokens.DefaultTypography, tokens.Comfortable)(gtx)
	}
}

func (inv *Inventory) testimonial(c tokens.ColorTokens) layout.Widget {
	props := testimonial.Props{
		Variant: testimonial.Single,
		Items: []testimonial.Item{
			{
				Quote:        "Seeing the whole inventory at once is what made the grey cast obvious.",
				AuthorName:   "A reviewer",
				AuthorRole:   "Fresh eyes",
				AuthorAvatar: dot(c.Primary, 40),
			},
		},
		Shaper: inv.shaper,
	}
	return func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Max.X = min(gtx.Constraints.Max.X, gtx.Dp(560))
		return testimonial.Render(inv.shaper, props, c, tokens.Spacing, tokens.Radius,
			tokens.DefaultTypography)(gtx)
	}
}

// shell draws the three-column frame with its aside occupied, which is the
// only place the aside's own frame shows.
func (inv *Inventory) shell(c tokens.ColorTokens) layout.Widget {
	props := shell.Props{
		Layout: shell.ThreeColumn,
		Navbar: inv.navbarProps(c),
		Main: func(gtx layout.Context) layout.Dimensions {
			return complayout.Inset(16).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return inv.prose(c,
					"The main column.",
					"",
					"A shell frames a window: a sidebar",
					"on the leading edge, a navbar across",
					"the top, an aside on the trailing edge,",
					"and a footer under all of it.",
				)(gtx)
			})
		},
		Footer: func(gtx layout.Context) layout.Dimensions {
			return complayout.InsetXY(16, 8).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return LabelAt(gtx, inv.shaper, "Footer — status and counts", c.Ramps.Neutral.Step(600), 11, font.Font{})
			})
		},
	}
	sidebarW := patsidebar.Render(inv.shaper, inv.sidebarProps(c), false, c, tokens.Spacing,
		tokens.DefaultTypography.LabelLarge, tokens.Comfortable)
	asideW := func(gtx layout.Context) layout.Dimensions {
		// An aside is an inspector, which is a chrome region: it
		// fills at the chrome level, the same level the frame around it
		// paints, rather than at the c.Surface ramp alias.
		paint.FillShape(gtx.Ops, c.SurfaceAt(tokens.LevelChrome), clip.Rect{Max: gtx.Constraints.Max}.Op())
		return complayout.Inset(12).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return inv.prose(c, "Aside", "", "Inspector, outline", "or details.")(gtx)
		})
	}
	return func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Max.X = min(gtx.Constraints.Max.X, gtx.Dp(720))
		gtx.Constraints.Min = gtx.Constraints.Max
		return shell.RenderThreeColumn(inv.shaper, props, sidebarW, asideW, c, tokens.Spacing,
			tokens.DefaultTypography.LabelLarge, tokens.Comfortable, 180)(gtx)
	}
}

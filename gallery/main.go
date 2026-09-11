// Package main is the Components gallery — one page per Phase 1 component.
// Every variant and every a11y mode is visible; interactions are live.
//
// Run: go run github.com/vibrantgio/components/gallery
package main

import (
	"fmt"
	"image"
	"image/color"
	"log"
	"os"
	"sync"
	"time"

	"gioui.org/app"
	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"

	"github.com/reactivego/rx"
	"github.com/vibrantgio/components/button"
	"github.com/vibrantgio/components/chip"
	"github.com/vibrantgio/components/gallery/inventory"
	"github.com/vibrantgio/components/initial"
	"github.com/vibrantgio/components/input"
	complayout "github.com/vibrantgio/components/layout"
	"github.com/vibrantgio/components/list"
	"github.com/vibrantgio/components/paragraph"
	"github.com/vibrantgio/components/scrollbar"
	"github.com/vibrantgio/mvu/stream"
	"github.com/vibrantgio/theme/a11y"
	vgcolor "github.com/vibrantgio/theme/color"
	"github.com/vibrantgio/theme/theme"
	"github.com/vibrantgio/theme/tokens"

	"github.com/vibrantgio/components/icon"
	"github.com/vibrantgio/components/icons"
	"github.com/vibrantgio/effects/springbutton"
	ivgraster "github.com/vibrantgio/ivg/raster/gio"
)

// pageNames is the sidebar, in order. Everything comes first because it is the
// page the whole surface is judged on; the per-family pages that follow are
// for close-up work on one component at a time.
var pageNames = []string{
	"Everything",
	"Button", "Chip", "Inputs", "List", "Paragraph", "Icon", "Layout", "A11y", "Initial", "Stream",
	"Patterns", "Markdown",
}

const (
	pageEverything int = iota
	pageButton
	pageChip
	pageInputs
	pageList
	pageParagraph
	pageIcon
	pageLayout
	pageA11y
	pageInitial
	pageStream
	pagePatterns
	pageMarkdown
)

// The halves of the scheme control, in the order they are laid out.
const (
	schemeLightSegment = iota
	schemeDarkSegment
)

type gallery struct {
	win    *app.Window
	shaper *text.Shaper
	page   int
	nav    []widget.Clickable

	// The whole published surface, built once from static state. The
	// everything page and the two inventory pages draw it; the per-family
	// pages do not.
	inv  *inventory.Inventory
	dark bool
	// One per segment of the scheme control, indexed by schemeLightSegment
	// and schemeDarkSegment: each half is its own target, so a press names
	// the scheme under it.
	schemeBtn [2]widget.Clickable

	// Interactive components obtained via rx.First()
	btnLive       layout.Widget
	btnCompare    layout.Widget
	springBtnLive layout.Widget
	tfLive        layout.Widget
	sfLive        layout.Widget
	cbLive        layout.Widget
	rbALive       layout.Widget
	rbBLive       layout.Widget
	ddLive        layout.Widget

	// Chip page: one live chip per surface.
	chipLive   []layout.Widget
	chipClicks []int

	// Button page
	btnClicks        int
	btnCompareClicks int
	springBtnClicks  int

	// Scroll state — one per page, allocated once so scroll position survives frames.
	scrollSt []*list.State

	// List page: one state per LayoutScrollbar variant so the two lists
	// scroll independently.
	listSt        *list.State
	listOverlaySt *list.State
	listItems     []string

	// Scrollbar demo (List page): a tall fake-content column laid out with a
	// raw layout.List (its Position is readable, unlike list.State's) plus a
	// standalone scrollbar driven by FromListPosition fractions.
	sbList  layout.List
	sbState *scrollbar.State
	sbItems []string

	// Paragraph page: persistent link-interaction state plus the themed style
	// whose OnLinkClick records the last activated URL.
	paraState   *paragraph.State
	paraStyle   paragraph.Style
	paraLastURL string
	paraClicks  int

	// Icon page
	iconReg   *icon.Registry
	ivgWidget layout.Widget

	// A11y page
	a11ySub rx.Subscription
	prefsMu sync.Mutex
	prefs   a11y.A11yPrefs

	// Initial page
	initVal initial.Value[time.Time]

	// Stream page
	streamObserver rx.Observer[string]
	streamSub      rx.Subscription
	streamMu       sync.Mutex
	streamMsg      string
	streamSend     widget.Clickable
}

func main() {
	go func() {
		w := new(app.Window)
		w.Option(
			app.Title("Components Gallery"),
			app.Size(unit.Dp(900), unit.Dp(700)),
		)
		if err := run(w); err != nil {
			log.Fatal(err)
		}
		os.Exit(0)
	}()
	app.Main()
}

func run(w *app.Window) error {
	// The theme's cached Roboto shaper (theme tokens.DefaultTypography),
	// shared by every static Render* variant and label on the pages.
	shaper := tokens.DefaultTypography.Shaper()
	g := newGallery(w, shaper)
	defer g.cleanup()

	var ops op.Ops
	for {
		e := w.Event()
		switch e := e.(type) {
		case app.DestroyEvent:
			return e.Err
		case app.FrameEvent:
			gtx := app.NewContext(&ops, e)
			g.frame(gtx)
			e.Frame(gtx.Ops)
		}
	}
}

func newGallery(w *app.Window, shaper *text.Shaper) *gallery {
	g := &gallery{win: w, shaper: shaper}
	g.nav = make([]widget.Clickable, len(pageNames))
	g.scrollSt = make([]*list.State, len(pageNames))
	for i := range g.scrollSt {
		g.scrollSt[i] = list.NewState()
	}
	g.inv = inventory.New(shaper)

	// Static theme observable — emits once synchronously, so First() returns immediately.
	th := rx.Of(theme.Default())

	var err error
	g.btnLive, err = button.Button(th, button.Props{
		Label:   "Click me",
		OnClick: func(_ layout.Context) { g.btnClicks++; w.Invalidate() },
	}).First()
	if err != nil {
		log.Printf("button: %v", err)
	}

	g.btnCompare, err = button.Button(th, button.Props{
		Label:   "Click me",
		OnClick: func(_ layout.Context) { g.btnCompareClicks++; w.Invalidate() },
	}).First()
	if err != nil {
		log.Printf("button-compare: %v", err)
	}

	g.springBtnLive, err = springbutton.SpringButton(th, button.Props{
		Label:   "Click me",
		OnClick: func(_ layout.Context) { g.springBtnClicks++; w.Invalidate() },
	}, springbutton.Options{}).First()
	if err != nil {
		log.Printf("springbutton: %v", err)
	}

	// One live chip per surface. Each is a real component with its own
	// clickable, so the page answers the pointer and the Tab key rather than
	// showing a drawing of a chip that does.
	g.chipLive = make([]layout.Widget, len(chipSpecimens))
	g.chipClicks = make([]int, len(chipSpecimens))
	for i, lv := range chipSpecimens {
		i, lv := i, lv
		g.chipLive[i], err = chip.Chip(th, chip.Props{
			Label:       lv.label,
			Icon:        chip.Glyph(icons.Mark(icons.Disclosure)),
			Description: lv.desc,
			Surface:     lv.fill,
			OnClick:     func(_ layout.Context) { g.chipClicks[i]++; w.Invalidate() },
		}).First()
		if err != nil {
			log.Printf("chip %s: %v", lv.label, err)
		}
	}

	g.tfLive, err = input.TextField(th, input.TextFieldProps{
		Placeholder: "Type here…",
		Shaper:      shaper,
	}).First()
	if err != nil {
		log.Printf("textfield: %v", err)
	}

	g.sfLive, err = input.SearchField(th, input.SearchFieldProps{
		Placeholder: "Search",
	}).First()
	if err != nil {
		log.Printf("searchfield: %v", err)
	}

	g.cbLive, err = input.Checkbox(th, input.CheckboxProps{
		Description: "Accept terms",
	}).First()
	if err != nil {
		log.Printf("checkbox: %v", err)
	}

	g.rbALive, err = input.Radio(th, input.RadioProps{
		Description: "Option A",
		Selected:    true,
	}).First()
	if err != nil {
		log.Printf("radio A: %v", err)
	}

	g.rbBLive, err = input.Radio(th, input.RadioProps{
		Description: "Option B",
	}).First()
	if err != nil {
		log.Printf("radio B: %v", err)
	}

	g.ddLive, err = input.Dropdown(th, input.DropdownProps{
		Description: "Choose fruit",
		Options:     []string{"Apple", "Banana", "Cherry", "Date"},
		Shaper:      shaper,
	}).First()
	if err != nil {
		log.Printf("dropdown: %v", err)
	}

	// List demo.
	g.listSt = list.NewState()
	g.listOverlaySt = list.NewState()
	g.listItems = make([]string, 50)
	for i := range g.listItems {
		g.listItems[i] = fmt.Sprintf("Item %d — virtual scrolling: only visible rows are laid out", i+1)
	}

	// Scrollbar demo: tall fake content plus a standalone bar.
	g.sbList = layout.List{Axis: layout.Vertical}
	g.sbState = scrollbar.NewState()
	g.sbItems = make([]string, 100)
	for i := range g.sbItems {
		g.sbItems[i] = fmt.Sprintf("Fake content row %d of %d", i+1, len(g.sbItems))
	}

	// Paragraph: live link state; OnLinkClick carries gtx.
	g.paraState = paragraph.NewState()
	g.paraStyle = paragraph.FromTokens(tokens.PlatformLight, tokens.DefaultTypography.BodyLarge, pagePlane)
	g.paraStyle.OnLinkClick = func(_ layout.Context, url string) {
		g.paraLastURL = url
		g.paraClicks++
		w.Invalidate()
	}

	// Icon registry — register the IVG icon and obtain a render layout.Widget
	// from it.
	g.iconReg = icon.New()
	g.iconReg.Register("info", icon.FromIVG(inventory.ActionInfoIVG))
	if ic, ok := g.iconReg.Icon("info"); ok {
		g.ivgWidget, err = ivgraster.Widget(ic.IVG(), 64, 64)
		if err != nil {
			log.Printf("ivg: %v", err)
		}
	}
	if g.ivgWidget == nil {
		g.ivgWidget = func(gtx layout.Context) layout.Dimensions {
			return layout.Dimensions{Size: image.Pt(64, 64)}
		}
	}

	// A11y: live OS preference polling on Goroutine scheduler.
	g.a11ySub = a11y.Live(2*time.Second).Subscribe(rx.GoroutineContext(), func(p a11y.A11yPrefs, err error, done bool) {
		if !done {
			g.prefsMu.Lock()
			g.prefs = p
			g.prefsMu.Unlock()
			w.Invalidate()
		}
	})

	// Stream: mvu/stream.Value[string] for the producer/consumer demo.
	var streamObs rx.Observable[string]
	g.streamObserver, streamObs = stream.Value("")
	g.streamSub = streamObs.Subscribe(rx.GoroutineContext(), func(msg string, err error, done bool) {
		if !done {
			g.streamMu.Lock()
			g.streamMsg = msg
			g.streamMu.Unlock()
			w.Invalidate()
		}
	})

	return g
}

func (g *gallery) cleanup() {
	if g.a11ySub != nil {
		g.a11ySub.Unsubscribe()
	}
	if g.streamSub != nil {
		g.streamSub.Unsubscribe()
	}
}

// ── Frame layout ──────────────────────────────────────────────────────────────

func (g *gallery) frame(gtx layout.Context) layout.Dimensions {
	paint.FillShape(gtx.Ops, g.chrome().WindowBackground, clip.Rect{Max: gtx.Constraints.Max}.Op())
	return layout.Flex{}.Layout(gtx,
		layout.Rigid(g.sidebar),
		layout.Flexed(1, g.content),
	)
}

func (g *gallery) sidebar(gtx layout.Context) layout.Dimensions {
	const sideW = unit.Dp(150)
	w := gtx.Dp(sideW)
	gtx.Constraints = layout.Exact(image.Pt(w, gtx.Constraints.Max.Y))
	c := g.chrome()

	// The gallery's own rail is a sidebar, so it wears the platform's chrome
	// material — which in the light appearance is the window's own plane
	// exactly, and is told apart from it by the seam alone.
	paint.FillShape(gtx.Ops, c.SidebarMaterial, clip.Rect{Max: gtx.Constraints.Max}.Op())
	paint.FillShape(gtx.Ops, vgcolor.Flatten(c.Separator, c.SidebarMaterial),
		clip.Rect(image.Rect(w-1, 0, w, gtx.Constraints.Max.Y)).Op())

	cs := make([]layout.FlexChild, 0, 1+len(pageNames))
	cs = append(cs, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
		return complayout.Inset(16).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return g.label(gtx, "Components Gallery", vgcolor.Flatten(c.Label, c.SidebarMaterial), unit.Sp(13), font.Font{Weight: font.Bold})
		})
	}))
	for i, name := range pageNames {
		i, name := i, name
		cs = append(cs, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if g.nav[i].Clicked(gtx) {
				g.page = i
			}
			active := g.page == i
			// A sidebar row on this platform is the emphasized selection
			// fill under the foreground the platform pairs with it; an
			// unselected row carries no fill and the sidebar's own label.
			fg := vgcolor.Flatten(c.Label, c.SidebarMaterial)
			if active {
				fg = c.AlternateSelectedControlText
			}
			return g.nav[i].Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				sz := image.Pt(gtx.Constraints.Max.X, gtx.Dp(unit.Dp(40)))
				// Only the selected entry carries a fill of its own, so an
				// unselected entry lets the rail's own fill show through.
				if active {
					paint.FillShape(gtx.Ops, c.SelectedContentBackground, clip.Rect{Max: sz}.Op())
				}
				return complayout.InsetXY(16, 10).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return g.label(gtx, name, fg, unit.Sp(14), font.Font{})
				})
			})
		}))
	}
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx, cs...)
}

func (g *gallery) content(gtx layout.Context) layout.Dimensions {
	paint.FillShape(gtx.Ops, g.chrome().WindowBackground, clip.Rect{Max: gtx.Constraints.Max}.Op())
	switch g.page {
	case pageEverything:
		return g.pageEverything(gtx)
	case pagePatterns:
		return g.pagePatterns(gtx)
	case pageMarkdown:
		return g.pageMarkdown(gtx)
	case pageButton:
		return g.pageButton(gtx)
	case pageChip:
		return g.pageChip(gtx)
	case pageInputs:
		return g.pageInputs(gtx)
	case pageList:
		return g.pageList(gtx)
	case pageParagraph:
		return g.pageParagraph(gtx)
	case pageIcon:
		return g.pageIcon(gtx)
	case pageLayout:
		return g.pageLayout(gtx)
	case pageA11y:
		return g.pageA11y(gtx)
	case pageInitial:
		return g.pageInitial(gtx)
	case pageStream:
		return g.pageStream(gtx)
	}
	return layout.Dimensions{}
}

// ── Button page ───────────────────────────────────────────────────────────────

func (g *gallery) pageButton(gtx layout.Context) layout.Dimensions {
	return g.scrollPage(gtx, g.scrollSt[pageButton], func(gtx layout.Context) layout.Dimensions {
		cs := []layout.FlexChild{g.sectionHeader("Button — variant grid")}
		cs = append(cs, g.buttonVariantRows()...)
		cs = append(cs,
			g.sectionHeader("Button — live interactive"),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return complayout.Inset(24).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							gtx.Constraints.Max.X = gtx.Dp(unit.Dp(200))
							gtx.Constraints.Min.X = gtx.Dp(unit.Dp(200))
							if g.btnLive != nil {
								return g.btnLive(gtx)
							}
							return layout.Dimensions{}
						}),
						layout.Rigid(complayout.HSpacer(16)),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return g.label(gtx, fmt.Sprintf("Clicks: %d", g.btnClicks),
								pageText, unit.Sp(14), font.Font{})
						}),
					)
				})
			}),
			g.sectionHeader("Button — static components.Button vs effects.SpringButton (press to see physics)"),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return complayout.Inset(24).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									gtx.Constraints.Min.X = gtx.Dp(unit.Dp(80))
									gtx.Constraints.Max.X = gtx.Dp(unit.Dp(80))
									return g.label(gtx, "Static", pageMuted, unit.Sp(13), font.Font{})
								}),
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									gtx.Constraints.Max.X = gtx.Dp(unit.Dp(200))
									gtx.Constraints.Min.X = gtx.Dp(unit.Dp(200))
									if g.btnCompare != nil {
										return g.btnCompare(gtx)
									}
									return layout.Dimensions{}
								}),
								layout.Rigid(complayout.HSpacer(16)),
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									return g.label(gtx, fmt.Sprintf("Clicks: %d", g.btnCompareClicks),
										pageText, unit.Sp(14), font.Font{})
								}),
							)
						}),
						layout.Rigid(complayout.VSpacer(16)),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									gtx.Constraints.Min.X = gtx.Dp(unit.Dp(80))
									gtx.Constraints.Max.X = gtx.Dp(unit.Dp(80))
									return g.label(gtx, "Spring", pageMuted, unit.Sp(13), font.Font{})
								}),
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									gtx.Constraints.Max.X = gtx.Dp(unit.Dp(200))
									gtx.Constraints.Min.X = gtx.Dp(unit.Dp(200))
									if g.springBtnLive != nil {
										return g.springBtnLive(gtx)
									}
									return layout.Dimensions{}
								}),
								layout.Rigid(complayout.HSpacer(16)),
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									return g.label(gtx, fmt.Sprintf("Clicks: %d", g.springBtnClicks),
										pageText, unit.Sp(14), font.Font{})
								}),
							)
						}),
						layout.Rigid(complayout.VSpacer(12)),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return g.label(gtx,
								"effects.SpringButton(theme, button.Props{...}, springbutton.Options{}) — DESIGN §Phase 3 — Composition mechanism.",
								pageMuted, unit.Sp(13), font.Font{})
						}),
					)
				})
			}),
		)
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx, cs...)
	})
}

func (g *gallery) buttonVariantRows() []layout.FlexChild {
	type row struct {
		label  string
		state  button.RenderState
		colors tokens.PlatformColors
		rowBg  color.NRGBA
	}
	rows := []row{
		{"Normal (light)", button.RenderState{}, tokens.PlatformLight, tokens.PlatformLight.WindowBackground},
		{"Hovered (light)", button.RenderState{Hovered: true}, tokens.PlatformLight, tokens.PlatformLight.WindowBackground},
		{"Focused (light)", button.RenderState{Focused: true}, tokens.PlatformLight, tokens.PlatformLight.WindowBackground},
		{"Pressed (light)", button.RenderState{Pressed: true}, tokens.PlatformLight, tokens.PlatformLight.WindowBackground},
		{"Disabled (light)", button.RenderState{Disabled: true}, tokens.PlatformLight, tokens.PlatformLight.WindowBackground},
		{"Normal (dark)", button.RenderState{}, tokens.PlatformDark, tokens.PlatformDark.WindowBackground},
		{"Focused (dark)", button.RenderState{Focused: true}, tokens.PlatformDark, tokens.PlatformDark.WindowBackground},
		{"Pressed (dark)", button.RenderState{Pressed: true}, tokens.PlatformDark, tokens.PlatformDark.WindowBackground},
		{"Disabled (dark)", button.RenderState{Disabled: true}, tokens.PlatformDark, tokens.PlatformDark.WindowBackground},
	}
	cs := make([]layout.FlexChild, len(rows))
	for i, r := range rows {
		r := r
		w := button.Render(g.shaper, "Button", r.colors, tokens.Spacing, tokens.Radius, tokens.DefaultTypography.LabelLarge, tokens.Comfortable, r.state)
		cs[i] = layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return g.variantRow(gtx, r.label, r.rowBg, r.colors.Text, w)
		})
	}
	return cs
}

// ── Chip page ─────────────────────────────────────────────────────────────────

// chipSpecimens are the fills the chip page puts a live chip on: the content
// plane, the platform's grouped box, and the chrome material. The chip's rim,
// its focus ring and its press tint each carry a coverage rather than a
// colour, so each lands as whatever it is composited onto and one specimen on
// one fill demonstrates nothing about the component. Three do. The label on
// each is a summary rather than a verb, which is the whole of what separates a
// chip from a button: what a pane is showing, what a list is filtered by,
// which model a conversation is on.
var chipSpecimens = []struct {
	label string
	desc  string
	fill  color.NRGBA
	title string
}{
	{"Claude Opus 5", "Choose a model", tokens.PlatformLight.ControlBackground, "The content plane"},
	{"main", "Switch branch", tokens.PlatformLight.CardFill, "The platform's box"},
	{"3 filters", "Edit filters", tokens.PlatformLight.SidebarMaterial, "The chrome material"},
}

func (g *gallery) pageChip(gtx layout.Context) layout.Dimensions {
	if len(g.chipLive) != len(chipSpecimens) {
		// The live components are built against a window; a gallery assembled
		// without one draws nothing here rather than indexing past its state.
		return layout.Dimensions{}
	}
	return g.scrollPage(gtx, g.scrollSt[pageChip], func(gtx layout.Context) layout.Dimensions {
		c := tokens.PlatformLight
		cs := []layout.FlexChild{g.sectionHeader("Chip — live, on each surface (click it, or Tab to it and press Space)")}
		for i, lv := range chipSpecimens {
			i, lv := i, lv
			cs = append(cs, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return complayout.InsetXY(24, 12).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							gtx.Constraints.Min.X = gtx.Dp(unit.Dp(180))
							gtx.Constraints.Max.X = gtx.Dp(unit.Dp(180))
							return g.label(gtx, lv.title, pageMuted, unit.Sp(12), font.Font{})
						}),
						layout.Rigid(g.surfacePanel(lv.fill, func(gtx layout.Context) layout.Dimensions {
							return complayout.Inset(12).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								if g.chipLive[i] == nil {
									return layout.Dimensions{}
								}
								return g.chipLive[i](gtx)
							})
						})),
						layout.Rigid(complayout.HSpacer(24)),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return g.label(gtx, fmt.Sprintf("Clicks: %d", g.chipClicks[i]),
								c.Text, unit.Sp(14), font.Font{})
						}),
					)
				})
			}))
		}
		cs = append(cs, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return complayout.InsetXY(24, 12).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return g.label(gtx,
					"The chip lays the platform's press overlay over its own fill and wears the focus ring in place of its rim.",
					pageMuted, unit.Sp(13), font.Font{})
			})
		}))
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx, cs...)
	})
}

// surfacePanel draws content over a fill of its own, sized to what the
// content measured — the fill goes down after the content is recorded,
// because the panel's size is the content's and nothing knows it sooner.
func (g *gallery) surfacePanel(fill color.NRGBA, content layout.Widget) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		m := op.Record(gtx.Ops)
		dims := content(gtx)
		call := m.Stop()
		paint.FillShape(gtx.Ops, fill, clip.Rect{Max: dims.Size}.Op())
		call.Add(gtx.Ops)
		return dims
	}
}

// ── Inputs page ───────────────────────────────────────────────────────────────

func (g *gallery) pageInputs(gtx layout.Context) layout.Dimensions {
	return g.scrollPage(gtx, g.scrollSt[pageInputs], func(gtx layout.Context) layout.Dimensions {
		cs := []layout.FlexChild{g.sectionHeader("TextField — variants")}
		cs = append(cs, g.textFieldVariantRows()...)
		cs = append(cs,
			g.sectionHeader("TextField — live"),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return complayout.Inset(24).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					gtx.Constraints.Max.X = gtx.Dp(unit.Dp(300))
					gtx.Constraints.Min.X = gtx.Dp(unit.Dp(300))
					if g.tfLive != nil {
						return g.tfLive(gtx)
					}
					return layout.Dimensions{}
				})
			}),
		)
		cs = append(cs, g.sectionHeader("SearchField — variants"))
		cs = append(cs, g.searchFieldVariantRows()...)
		cs = append(cs,
			g.sectionHeader("SearchField — live (type, then press the clear mark)"),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return complayout.Inset(24).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					gtx.Constraints.Max.X = gtx.Dp(unit.Dp(300))
					gtx.Constraints.Min.X = gtx.Dp(unit.Dp(300))
					if g.sfLive != nil {
						return g.sfLive(gtx)
					}
					return layout.Dimensions{}
				})
			}),
		)
		cs = append(cs, g.sectionHeader("Checkbox — variants"))
		cs = append(cs, g.checkboxVariantRows()...)
		cs = append(cs,
			g.sectionHeader("Checkbox — live"),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return complayout.Inset(24).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					if g.cbLive != nil {
						return g.cbLive(gtx)
					}
					return layout.Dimensions{}
				})
			}),
		)
		cs = append(cs, g.sectionHeader("Radio — variants"))
		cs = append(cs, g.radioVariantRows()...)
		cs = append(cs,
			g.sectionHeader("Radio — live (two options, independent state)"),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return complayout.Inset(24).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							if g.rbALive != nil {
								return g.rbALive(gtx)
							}
							return layout.Dimensions{}
						}),
						layout.Rigid(complayout.HSpacer(8)),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return g.label(gtx, "Option A", pageText, unit.Sp(14), font.Font{})
						}),
						layout.Rigid(complayout.HSpacer(32)),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							if g.rbBLive != nil {
								return g.rbBLive(gtx)
							}
							return layout.Dimensions{}
						}),
						layout.Rigid(complayout.HSpacer(8)),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return g.label(gtx, "Option B", pageText, unit.Sp(14), font.Font{})
						}),
					)
				})
			}),
		)
		cs = append(cs, g.sectionHeader("Dropdown — variants"))
		cs = append(cs, g.dropdownVariantRows()...)
		cs = append(cs,
			g.sectionHeader("Dropdown — live"),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return complayout.Inset(24).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					gtx.Constraints.Max.X = gtx.Dp(unit.Dp(300))
					gtx.Constraints.Min.X = gtx.Dp(unit.Dp(300))
					if g.ddLive != nil {
						return g.ddLive(gtx)
					}
					return layout.Dimensions{}
				})
			}),
		)
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx, cs...)
	})
}

func (g *gallery) textFieldVariantRows() []layout.FlexChild {
	type row struct {
		label  string
		state  input.RenderState
		colors tokens.PlatformColors
	}
	rows := []row{
		{"Normal (light)", input.RenderState{}, tokens.PlatformLight},
		{"Focused (light)", input.RenderState{Focused: true}, tokens.PlatformLight},
		{"Disabled (light)", input.RenderState{Disabled: true}, tokens.PlatformLight},
		{"Normal (dark)", input.RenderState{}, tokens.PlatformDark},
		{"Focused (dark)", input.RenderState{Focused: true}, tokens.PlatformDark},
		{"Disabled (dark)", input.RenderState{Disabled: true}, tokens.PlatformDark},
	}
	cs := make([]layout.FlexChild, len(rows))
	for i, r := range rows {
		r := r
		w := input.Render(g.shaper, "Placeholder…", r.colors, tokens.Spacing, tokens.Radius, tokens.DefaultTypography.BodyLarge, tokens.Comfortable, r.state)
		cs[i] = layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return g.variantRow(gtx, r.label, r.colors.WindowBackground, vgcolor.Flatten(r.colors.Label, r.colors.WindowBackground), w)
		})
	}
	return cs
}

func (g *gallery) searchFieldVariantRows() []layout.FlexChild {
	type row struct {
		label  string
		state  input.RenderState
		colors tokens.PlatformColors
	}
	rows := []row{
		{"Rest (light)", input.RenderState{}, tokens.PlatformLight},
		{"Typed (light)", input.RenderState{Text: "meeting notes"}, tokens.PlatformLight},
		{"Focused (light)", input.RenderState{Focused: true, Text: "meeting notes"}, tokens.PlatformLight},
		{"Rest (dark)", input.RenderState{}, tokens.PlatformDark},
		{"Typed (dark)", input.RenderState{Text: "meeting notes"}, tokens.PlatformDark},
		{"Disabled (dark)", input.RenderState{Disabled: true}, tokens.PlatformDark},
	}
	cs := make([]layout.FlexChild, len(rows))
	for i, r := range rows {
		r := r
		w := input.RenderSearch(g.shaper, "Search", r.colors, tokens.Spacing, tokens.Radius, tokens.DefaultTypography.BodyLarge, tokens.Comfortable, r.state)
		cs[i] = layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return g.variantRow(gtx, r.label, r.colors.WindowBackground, vgcolor.Flatten(r.colors.Label, r.colors.WindowBackground), w)
		})
	}
	return cs
}

func (g *gallery) checkboxVariantRows() []layout.FlexChild {
	type row struct {
		label  string
		state  input.CheckboxRenderState
		colors tokens.PlatformColors
	}
	rows := []row{
		{"Unchecked (light)", input.CheckboxRenderState{}, tokens.PlatformLight},
		{"Checked (light)", input.CheckboxRenderState{Checked: true}, tokens.PlatformLight},
		{"Focused (light)", input.CheckboxRenderState{Focused: true}, tokens.PlatformLight},
		{"Disabled (light)", input.CheckboxRenderState{Disabled: true}, tokens.PlatformLight},
		{"Unchecked (dark)", input.CheckboxRenderState{}, tokens.PlatformDark},
		{"Checked (dark)", input.CheckboxRenderState{Checked: true}, tokens.PlatformDark},
	}
	cs := make([]layout.FlexChild, len(rows))
	for i, r := range rows {
		r := r
		w := input.RenderCheckbox(r.colors, tokens.Spacing, tokens.Radius, r.state)
		cs[i] = layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return g.variantRow(gtx, r.label, r.colors.WindowBackground, vgcolor.Flatten(r.colors.Label, r.colors.WindowBackground), w)
		})
	}
	return cs
}

func (g *gallery) radioVariantRows() []layout.FlexChild {
	type row struct {
		label  string
		state  input.RadioRenderState
		colors tokens.PlatformColors
	}
	rows := []row{
		{"Unselected (light)", input.RadioRenderState{}, tokens.PlatformLight},
		{"Selected (light)", input.RadioRenderState{Selected: true}, tokens.PlatformLight},
		{"Focused (light)", input.RadioRenderState{Focused: true}, tokens.PlatformLight},
		{"Disabled (light)", input.RadioRenderState{Disabled: true}, tokens.PlatformLight},
		{"Unselected (dark)", input.RadioRenderState{}, tokens.PlatformDark},
		{"Selected (dark)", input.RadioRenderState{Selected: true}, tokens.PlatformDark},
	}
	cs := make([]layout.FlexChild, len(rows))
	for i, r := range rows {
		r := r
		w := input.RenderRadio(r.colors, tokens.Spacing, tokens.Radius, r.state)
		cs[i] = layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return g.variantRow(gtx, r.label, r.colors.WindowBackground, vgcolor.Flatten(r.colors.Label, r.colors.WindowBackground), w)
		})
	}
	return cs
}

func (g *gallery) dropdownVariantRows() []layout.FlexChild {
	opts := []string{"Apple", "Banana", "Cherry"}
	type row struct {
		label  string
		state  input.DropdownRenderState
		colors tokens.PlatformColors
	}
	rows := []row{
		{"Closed (light)", input.DropdownRenderState{Options: opts, Selected: 0}, tokens.PlatformLight},
		{"Focused (light)", input.DropdownRenderState{Options: opts, Focused: true}, tokens.PlatformLight},
		{"Open (light)", input.DropdownRenderState{Options: opts, Open: true, Selected: 1}, tokens.PlatformLight},
		{"Disabled (light)", input.DropdownRenderState{Options: opts, Disabled: true}, tokens.PlatformLight},
		{"Closed (dark)", input.DropdownRenderState{Options: opts}, tokens.PlatformDark},
		{"Open (dark)", input.DropdownRenderState{Options: opts, Open: true, Selected: 0}, tokens.PlatformDark},
	}
	cs := make([]layout.FlexChild, len(rows))
	for i, r := range rows {
		r := r
		w := input.RenderDropdown(g.shaper, r.colors, tokens.Spacing, tokens.Radius, tokens.DefaultTypography.BodyLarge, tokens.Comfortable, r.state)
		cs[i] = layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return g.variantRow(gtx, r.label, r.colors.WindowBackground, vgcolor.Flatten(r.colors.Label, r.colors.WindowBackground), w)
		})
	}
	return cs
}

// ── List page ─────────────────────────────────────────────────────────────────

func (g *gallery) pageList(gtx layout.Context) layout.Dimensions {
	return g.scrollPage(gtx, g.scrollSt[pageList], func(gtx layout.Context) layout.Dimensions {
		cs := []layout.FlexChild{
			g.sectionHeader(fmt.Sprintf("List — %d items, LayoutScrollbar: Occupy (left) vs Overlay (right)", len(g.listItems))),
			layout.Rigid(g.listScrollbarDemo),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return complayout.InsetXY(24, 8).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return g.label(gtx,
						"list.LayoutScrollbar(..., Occupy, ...) reserves a gutter; Overlay floats the bar over the rows. Wheel, thumb-drag, and track-click all scroll.",
						pageMuted, unit.Sp(13), font.Font{})
				})
			}),
			g.sectionHeader("Scrollbar — standalone bar beside tall fake content (drag the thumb, click the track, hover)"),
			layout.Rigid(g.scrollbarDemo),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return complayout.InsetXY(24, 8).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return g.label(gtx,
						"scrollbar.FromTokens(...).Layout with fractions from scrollbar.FromListPosition; drags feed back via ScrollDistance.",
						pageMuted, unit.Sp(13), font.Font{})
				})
			}),
		}
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx, cs...)
	})
}

// listScrollbarDemo lays out the 50-item list twice, side by side, via
// list.LayoutScrollbar: the left column anchors the bar with Occupy (a
// reserved gutter narrows the rows), the right with Overlay (the bar floats
// over full-width rows). Each column has its own list.State so they scroll
// independently.
func (g *gallery) listScrollbarDemo(gtx layout.Context) layout.Dimensions {
	return complayout.InsetXY(24, 12).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		h := gtx.Dp(unit.Dp(400))
		gtx.Constraints.Max.Y = h
		bar := scrollbar.FromTokens(tokens.PlatformLight, pagePlane)
		row := func(gtx layout.Context, item string) layout.Dimensions {
			return complayout.InsetXY(0, 10).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return g.label(gtx, item, pageText, unit.Sp(14), font.Font{})
			})
		}
		column := func(title string, st *list.State, anchor list.Anchor) layout.Widget {
			return func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return g.label(gtx, title, pageMuted, unit.Sp(13), font.Font{Weight: font.Bold})
					}),
					layout.Rigid(complayout.VSpacer(8)),
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
						gtx.Constraints.Min = gtx.Constraints.Max
						return list.LayoutScrollbar(gtx, st, bar, anchor, g.listItems, row)
					}),
				)
			}
		}
		return layout.Flex{}.Layout(gtx,
			layout.Flexed(0.5, column("Occupy — gutter reserved", g.listSt, list.Occupy)),
			layout.Rigid(complayout.HSpacer(24)),
			layout.Flexed(0.5, column("Overlay — bar floats over rows", g.listOverlaySt, list.Overlay)),
		)
	})
}

// scrollbarDemo lays out a 300dp-tall fake-content column with a standalone
// scrollbar beside it. The bar's fractions come from FromListPosition over the
// raw layout.List position, and scrollbar drags and track clicks are fed back
// into the list via ScrollDistance.
func (g *gallery) scrollbarDemo(gtx layout.Context) layout.Dimensions {
	return complayout.InsetXY(24, 12).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		h := gtx.Dp(unit.Dp(300))
		gtx.Constraints.Max.Y = h
		gtx.Constraints.Min.Y = h
		style := scrollbar.FromTokens(tokens.PlatformLight, pagePlane)
		barW := gtx.Dp(style.Width())
		// Both children are Rigid so the list lays out first: the bar then
		// reads this frame's scroll position instead of lagging one frame.
		return layout.Flex{}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				gtx.Constraints.Max.X -= barW
				gtx.Constraints.Min = gtx.Constraints.Max
				return g.sbList.Layout(gtx, len(g.sbItems), func(gtx layout.Context, i int) layout.Dimensions {
					return complayout.InsetXY(0, 8).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return g.label(gtx, g.sbItems[i], pageText, unit.Sp(14), font.Font{})
					})
				})
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				start, end := scrollbar.FromListPosition(g.sbList.Position, len(g.sbItems), h)
				dims := style.Layout(gtx, g.sbState, layout.Vertical, start, end)
				if d := g.sbState.ScrollDistance(); d != 0 {
					g.sbList.ScrollBy(d * float32(len(g.sbItems)))
					g.win.Invalidate()
				}
				return dims
			}),
		)
	})
}

// ── Paragraph page ────────────────────────────────────────────────────────────

// paragraphSpans is the demo paragraph exercising the span model: regular,
// bold, italic, monospace, explicit colour, explicit size, and two links.
func paragraphSpans() []paragraph.SpanStyle {
	return []paragraph.SpanStyle{
		{Content: "Paragraph lays out "},
		{Content: "bold", Weight: font.Bold},
		{Content: ", "},
		{Content: "italic", Style: font.Italic},
		{Content: ", "},
		{Content: "monospace", Typeface: "Go Mono, monospace"},
		{Content: ", "},
		{Content: "coloured", Color: tokens.PlatformLight.SystemRed},
		{Content: ", and "},
		{Content: "resized", Size: 22},
		{Content: " spans in one wrapped paragraph, with inline links to "},
		{Content: "gioui.org", URL: "https://gioui.org"},
		{Content: " and the "},
		{Content: "vibrantgio design system", URL: "https://github.com/vibrantgio"},
		{Content: "."},
	}
}

func (g *gallery) pageParagraph(gtx layout.Context) layout.Dimensions {
	return g.scrollPage(gtx, g.scrollSt[pageParagraph], func(gtx layout.Context) layout.Dimensions {
		staticStyle := paragraph.FromTokens(tokens.PlatformLight, tokens.DefaultTypography.BodyLarge, pagePlane)
		linkStates := []struct {
			label string
			state paragraph.RenderState
		}{
			{"Idle", paragraph.Idle()},
			{"Hovered (link 0)", paragraph.RenderState{HoveredLink: 0, FocusedLink: paragraph.NoLink}},
			{"Focused (link 0)", paragraph.RenderState{HoveredLink: paragraph.NoLink, FocusedLink: 0}},
		}

		cs := []layout.FlexChild{
			g.sectionHeader("Paragraph — mixed spans (bold / italic / mono / colour / size / links)"),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return complayout.Inset(24).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					gtx.Constraints.Max.X = gtx.Dp(unit.Dp(520))
					return paragraph.Render(g.shaper, staticStyle, paragraphSpans(), paragraph.Idle())(gtx)
				})
			}),
			g.sectionHeader("Paragraph — link states (static RenderState)"),
		}
		for _, ls := range linkStates {
			ls := ls
			cs = append(cs, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return complayout.InsetXY(24, 10).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							gtx.Constraints.Min.X = gtx.Dp(unit.Dp(160))
							gtx.Constraints.Max.X = gtx.Dp(unit.Dp(160))
							return g.label(gtx, ls.label, pageMuted, unit.Sp(13), font.Font{})
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							gtx.Constraints.Max.X = gtx.Dp(unit.Dp(400))
							spans := []paragraph.SpanStyle{
								{Content: "Read the "},
								{Content: "documentation", URL: "https://gioui.org/doc"},
								{Content: " for details."},
							}
							return paragraph.Render(g.shaper, staticStyle, spans, ls.state)(gtx)
						}),
					)
				})
			}))
		}
		cs = append(cs,
			g.sectionHeader("Paragraph — live links (hover for cursor, Tab to focus, click or Space/Enter to activate)"),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return complayout.Inset(24).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							gtx.Constraints.Max.X = gtx.Dp(unit.Dp(520))
							return paragraph.Layout(gtx, g.paraState, g.shaper, g.paraStyle, paragraphSpans())
						}),
						layout.Rigid(complayout.VSpacer(16)),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							status := "Activated: (none yet — click a link above)"
							if g.paraLastURL != "" {
								status = fmt.Sprintf("Activated %d×, last: %s", g.paraClicks, g.paraLastURL)
							}
							return g.label(gtx, status, pageText, unit.Sp(14), font.Font{})
						}),
						layout.Rigid(complayout.VSpacer(8)),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return g.label(gtx,
								"paragraph.Layout(gtx, state, shaper, style, spans) — OnLinkClick(gtx, url) carries gtx per GX.8.",
								pageMuted, unit.Sp(13), font.Font{})
						}),
					)
				})
			}),
		)
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx, cs...)
	})
}

// ── Icon page ─────────────────────────────────────────────────────────────────

func (g *gallery) pageIcon(gtx layout.Context) layout.Dimensions {
	return g.scrollPage(gtx, g.scrollSt[pageIcon], func(gtx layout.Context) layout.Dimensions {
		cs := []layout.FlexChild{
			g.sectionHeader("Icon — IVG render (material action-info, 64×64 px)"),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return complayout.Inset(24).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							if g.ivgWidget != nil {
								return g.ivgWidget(gtx)
							}
							return layout.Dimensions{Size: image.Pt(64, 64)}
						}),
						layout.Rigid(complayout.HSpacer(16)),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return g.label(gtx, "icon.FromIVG(data) + ivg/raster/gio", pageText, unit.Sp(13), font.Font{})
						}),
					)
				})
			}),
			g.sectionHeader("Icon — Registry (live lookup)"),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return complayout.Inset(24).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					_, registered := g.iconReg.Icon("info")
					status := fmt.Sprintf(`Registry has "info": %v  (kind=IVG, from icon.FromIVG)`, registered)
					return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return g.label(gtx, status, pageText, unit.Sp(14), font.Font{})
						}),
						layout.Rigid(complayout.VSpacer(8)),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return g.label(gtx, "SVG icons: icon.FromSVG(parsed *svg.Icon) for the KindSVG path.", pageMuted, unit.Sp(13), font.Font{})
						}),
					)
				})
			}),
		}
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx, cs...)
	})
}

// ── Layout page ───────────────────────────────────────────────────────────────

func (g *gallery) pageLayout(gtx layout.Context) layout.Dimensions {
	red := color.NRGBA{R: 0xef, G: 0x44, B: 0x44, A: 0xff}
	green := color.NRGBA{R: 0x22, G: 0xc5, B: 0x5e, A: 0xff}
	blue := color.NRGBA{R: 0x3b, G: 0x82, B: 0xf6, A: 0xff}
	purple := color.NRGBA{R: 0xa8, G: 0x5e, B: 0xf7, A: 0xff}
	orange := color.NRGBA{R: 0xf9, G: 0x73, B: 0x16, A: 0xff}

	box := func(c color.NRGBA, dp float32) layout.Widget {
		return func(gtx layout.Context) layout.Dimensions {
			sz := image.Pt(gtx.Dp(unit.Dp(dp)), gtx.Dp(unit.Dp(dp)))
			paint.FillShape(gtx.Ops, c, clip.Rect{Max: sz}.Op())
			return layout.Dimensions{Size: sz}
		}
	}

	return g.scrollPage(gtx, g.scrollSt[pageLayout], func(gtx layout.Context) layout.Dimensions {
		cs := []layout.FlexChild{
			g.sectionHeader("Layout — Row (horizontal flex with HSpacer gaps)"),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return complayout.Inset(24).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return complayout.Row(gtx,
						box(red, 48), complayout.HSpacer(8),
						box(green, 48), complayout.HSpacer(8),
						box(blue, 48),
					)
				})
			}),
			g.sectionHeader("Layout — Col (vertical flex with VSpacer gaps)"),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return complayout.Inset(24).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return complayout.Col(gtx,
						box(red, 40), complayout.VSpacer(8),
						box(green, 40), complayout.VSpacer(8),
						box(blue, 40),
					)
				})
			}),
			g.sectionHeader("Layout — Inset (uniform 24dp left) and InsetXY (32dp×16dp right)"),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return complayout.Inset(24).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return complayout.Inset(24).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								return box(purple, 40)(gtx)
							})
						}),
						layout.Rigid(complayout.HSpacer(16)),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return complayout.InsetXY(32, 16).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								return box(orange, 40)(gtx)
							})
						}),
					)
				})
			}),
		}
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx, cs...)
	})
}

// ── A11y page ─────────────────────────────────────────────────────────────────

func (g *gallery) pageA11y(gtx layout.Context) layout.Dimensions {
	g.prefsMu.Lock()
	prefs := g.prefs
	g.prefsMu.Unlock()

	// The platform set with every foreground driven to full strength: the
	// accent, the label, the control text and the seam opaque black on the
	// white planes the light appearance already carries. It is what a caller
	// may hand a component, not a set this library ships.
	hc := tokens.PlatformLight
	black := color.NRGBA{A: 0xff}
	hc.ControlAccent, hc.Label, hc.Text, hc.ControlText, hc.Separator = black, black, black, black, black

	return g.scrollPage(gtx, g.scrollSt[pageA11y], func(gtx layout.Context) layout.Dimensions {
		cs := []layout.FlexChild{
			g.sectionHeader("A11y — live OS accessibility preferences (polled every 2s)"),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return complayout.Inset(24).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return g.prefRow(gtx, "ReduceMotion", prefs.ReduceMotion)
						}),
						layout.Rigid(complayout.VSpacer(8)),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return g.prefRow(gtx, "HighContrast", prefs.HighContrast)
						}),
						layout.Rigid(complayout.VSpacer(8)),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return g.prefRow(gtx, "IncreaseTextSize", prefs.IncreaseTextSize)
						}),
						layout.Rigid(complayout.VSpacer(12)),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return g.label(gtx, "Toggle Reduce Motion in System Settings > Accessibility > Display.", pageMuted, unit.Sp(13), font.Font{})
						}),
					)
				})
			}),
			g.sectionHeader("A11y — high contrast: the platform set with every foreground at full strength"),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return complayout.Inset(24).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							gtx.Constraints.Max.X = gtx.Dp(unit.Dp(200))
							gtx.Constraints.Min.X = gtx.Dp(unit.Dp(200))
							return button.Render(g.shaper, "High Contrast", hc, tokens.Spacing, tokens.Radius, tokens.DefaultTypography.LabelLarge, tokens.Comfortable, button.RenderState{})(gtx)
						}),
						layout.Rigid(complayout.HSpacer(16)),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return g.label(gtx, "The platform set, every foreground at full strength.", pageText, unit.Sp(13), font.Font{})
						}),
					)
				})
			}),
			g.sectionHeader("A11y — reduced motion: disable animations"),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return complayout.Inset(24).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					status := "animations enabled"
					if prefs.ReduceMotion {
						status = "reduced motion: skip animations"
					}
					return g.label(gtx, status, pageText, unit.Sp(14), font.Font{})
				})
			}),
		}
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx, cs...)
	})
}

func (g *gallery) prefRow(gtx layout.Context, name string, value bool) layout.Dimensions {
	indicator, col := "◯ off", pageMuted
	if value {
		indicator, col = "● on", tokens.PlatformLight.ControlAccent
	}
	return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			gtx.Constraints.Min.X = gtx.Dp(unit.Dp(200))
			return g.label(gtx, name, pageText, unit.Sp(14), font.Font{})
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return g.label(gtx, indicator, col, unit.Sp(14), font.Font{Weight: font.Bold})
		}),
	)
}

// ── Initial page ──────────────────────────────────────────────────────────────

func (g *gallery) pageInitial(gtx layout.Context) layout.Dimensions {
	firstFrame := g.initVal.GetOrSet(time.Now)

	return g.scrollPage(gtx, g.scrollSt[pageInitial], func(gtx layout.Context) layout.Dimensions {
		cs := []layout.FlexChild{
			g.sectionHeader("Initial — first-frame value, set once and stable"),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return complayout.Inset(24).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return g.label(gtx,
								fmt.Sprintf("Gallery opened at: %s", firstFrame.Format("15:04:05.000")),
								pageText, unit.Sp(14), font.Font{})
						}),
						layout.Rigid(complayout.VSpacer(8)),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return g.label(gtx,
								"initial.Value[T].GetOrSet(fn) calls fn once on the first invocation",
								pageMuted, unit.Sp(13), font.Font{})
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return g.label(gtx,
								"and returns the cached result on every subsequent frame.",
								pageMuted, unit.Sp(13), font.Font{})
						}),
						layout.Rigid(complayout.VSpacer(8)),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return g.label(gtx,
								"Use inside rx.Defer closures to replace ad-hoc -1 sentinels.",
								pageMuted, unit.Sp(13), font.Font{})
						}),
					)
				})
			}),
		}
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx, cs...)
	})
}

// ── Stream page ─────────────────────────────────────────────────────────

func (g *gallery) pageStream(gtx layout.Context) layout.Dimensions {
	if g.streamSend.Clicked(gtx) {
		g.streamObserver.Next(fmt.Sprintf("ping at %s", time.Now().Format("15:04:05.000")))
	}
	g.streamMu.Lock()
	msg := g.streamMsg
	g.streamMu.Unlock()

	received := msg
	if received == "" {
		received = "(none yet — click Send ping)"
	}

	return g.scrollPage(gtx, g.scrollSt[pageStream], func(gtx layout.Context) layout.Dimensions {
		cs := []layout.FlexChild{
			g.sectionHeader("Stream — mvu/stream.Value[string] producer/consumer"),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return complayout.Inset(24).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return g.streamSend.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								return button.Render(g.shaper, "Send ping", tokens.PlatformLight, tokens.Spacing, tokens.Radius, tokens.DefaultTypography.LabelLarge, tokens.Comfortable, button.RenderState{})(gtx)
							})
						}),
						layout.Rigid(complayout.VSpacer(16)),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return g.label(gtx, "Consumer received: "+received,
								pageText, unit.Sp(14), font.Font{})
						}),
						layout.Rigid(complayout.VSpacer(16)),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return g.label(gtx,
								"observer.Next(msg) sends via mvu/stream.Value[string](\"\") — ADR-008's",
								pageMuted, unit.Sp(13), font.Font{})
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return g.label(gtx,
								"Consumer goroutine calls w.Invalidate() to schedule the next frame.",
								pageMuted, unit.Sp(13), font.Font{})
						}),
					)
				})
			}),
		}
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx, cs...)
	})
}

// ── Shared helpers ────────────────────────────────────────────────────────────

func (g *gallery) scrollPage(gtx layout.Context, st *list.State, body func(layout.Context) layout.Dimensions) layout.Dimensions {
	items := []layout.Widget{body}
	return list.Layout(gtx, st, items, func(gtx layout.Context, w layout.Widget) layout.Dimensions {
		return w(gtx)
	})
}

func (g *gallery) sectionHeader(title string) layout.FlexChild {
	return layout.Rigid(func(gtx layout.Context) layout.Dimensions {
		bg := tokens.PlatformLight.CardFill
		h := gtx.Dp(unit.Dp(36))
		paint.FillShape(gtx.Ops, bg, clip.Rect{Max: image.Pt(gtx.Constraints.Max.X, h)}.Op())
		return complayout.InsetXY(24, 8).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return g.label(gtx, title, pageText, unit.Sp(13), font.Font{Weight: font.Bold})
		})
	})
}

func (g *gallery) variantRow(gtx layout.Context, lbl string, bg, fg color.NRGBA, w layout.Widget) layout.Dimensions {
	paint.FillShape(gtx.Ops, bg, clip.Rect{Max: image.Pt(gtx.Constraints.Max.X, gtx.Dp(unit.Dp(64)))}.Op())
	return complayout.InsetXY(24, 10).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				gtx.Constraints.Min.X = gtx.Dp(unit.Dp(200))
				gtx.Constraints.Max.X = gtx.Dp(unit.Dp(200))
				return g.label(gtx, lbl, fg, unit.Sp(13), font.Font{})
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				gtx.Constraints.Min.X = gtx.Dp(unit.Dp(200))
				gtx.Constraints.Max.X = gtx.Dp(unit.Dp(200))
				return w(gtx)
			}),
		)
	})
}

// The fill the per-family pages stand on and the two foregrounds they write
// in. Those pages draw the light appearance whatever the everything page's
// control says, so the surface is one value and the platform's label
// coverages are flattened onto it once rather than per frame.
var (
	pagePlane = tokens.PlatformLight.ControlBackground
	pageText  = vgcolor.Flatten(tokens.PlatformLight.Label, pagePlane)
	pageMuted = vgcolor.Flatten(tokens.PlatformLight.SecondaryLabel, pagePlane)
)

func (g *gallery) label(gtx layout.Context, s string, col color.NRGBA, size unit.Sp, f font.Font) layout.Dimensions {
	m := op.Record(gtx.Ops)
	paint.ColorOp{Color: col}.Add(gtx.Ops)
	mat := m.Stop()
	lbl := widget.Label{MaxLines: 1}
	return lbl.Layout(gtx, g.shaper, f, size, s, mat)
}

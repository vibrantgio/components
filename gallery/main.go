// Package main is the Prism gallery — one page per Phase 1 component.
// Every variant and every a11y mode is visible; interactions are live.
//
// Run: go run github.com/vibrantgio/prism/gallery
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
	"github.com/vibrantgio/prism/a11y"
	"github.com/vibrantgio/prism/button"
	"github.com/vibrantgio/prism/coordination"
	"github.com/vibrantgio/prism/initial"
	"github.com/vibrantgio/prism/input"
	prismlayout "github.com/vibrantgio/prism/layout"
	"github.com/vibrantgio/prism/list"
	"github.com/vibrantgio/prism/richtext"
	"github.com/vibrantgio/prism/scrollbar"
	"github.com/vibrantgio/prism/theme"
	"github.com/vibrantgio/prism/tokens"

	ivgraster "github.com/vibrantgio/ivg/raster/gio"
	"github.com/vibrantgio/prism/icon"
	"github.com/vibrantgio/pulse/springbutton"
	spectrumtokens "github.com/vibrantgio/spectrum/tokens"
)

var pageNames = []string{
	"Button", "Inputs", "List", "Richtext", "Icon", "Layout", "A11y", "Initial", "Coordination",
}

const (
	pageButton int = iota
	pageInputs
	pageList
	pageRichtext
	pageIcon
	pageLayout
	pageA11y
	pageInitial
	pageCoord
)

// actionInfoIVG is the material action-info icon in IVG format.
var actionInfoIVG = []byte{
	0x89, 0x49, 0x56, 0x47, 0x02, 0x0a, 0x00, 0x50, 0x50, 0xb0, 0xb0, 0xc0,
	0x80, 0x58, 0xa0, 0xf5, 0x74, 0x58, 0x58, 0xf5, 0x74, 0x58, 0x80, 0x91,
	0xf5, 0x88, 0xa8, 0xa8, 0xa8, 0xa8, 0x0d, 0x77, 0xa8, 0x58, 0x80, 0x0d,
	0x8b, 0x58, 0x80, 0x58, 0xe3, 0x84, 0xbc, 0xe7, 0x78, 0xe8, 0x7c, 0xe7,
	0x88, 0xe9, 0x98, 0xe3, 0x80, 0x60, 0xe7, 0x78, 0xe9, 0x78, 0xe7, 0x88,
	0xe9, 0x88, 0xe1,
}

type gallery struct {
	win    *app.Window
	shaper *text.Shaper
	page   int
	nav    [9]widget.Clickable

	// Interactive widgets obtained via rx.First()
	btnLive       layout.Widget
	btnCompare    layout.Widget
	springBtnLive layout.Widget
	tfLive        layout.Widget
	cbLive        layout.Widget
	rbALive       layout.Widget
	rbBLive       layout.Widget
	ddLive        layout.Widget

	// Button page
	btnClicks        int
	btnCompareClicks int
	springBtnClicks  int

	// Scroll state — one per page, allocated once so scroll position survives frames.
	scrollSt [9]*list.State

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

	// Richtext page: persistent link-interaction state plus the themed style
	// whose OnLinkClick records the last activated URL.
	rtState   *richtext.State
	rtStyle   richtext.Style
	rtLastURL string
	rtClicks  int

	// Icon page
	iconReg   *icon.Registry
	ivgWidget layout.Widget

	// A11y page
	a11ySub rx.Subscription
	prefsMu sync.Mutex
	prefs   a11y.A11yPrefs

	// Initial page
	initVal initial.Value[time.Time]

	// Coordination page
	coordObserver rx.Observer[string]
	coordSub      rx.Subscription
	coordMu       sync.Mutex
	coordMsg      string
	coordSend     widget.Clickable
}

func main() {
	go func() {
		w := new(app.Window)
		w.Option(
			app.Title("Prism Gallery"),
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
	// The theme's cached Roboto shaper (spectrum tokens.DefaultTypography),
	// shared by every static Render* variant and label on the pages.
	shaper := spectrumtokens.DefaultTypography.Shaper()
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
	for i := range g.scrollSt {
		g.scrollSt[i] = list.NewState()
	}

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

	g.tfLive, err = input.TextField(th, input.TextFieldProps{
		Placeholder: "Type here…",
		Shaper:      shaper,
	}).First()
	if err != nil {
		log.Printf("textfield: %v", err)
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

	// Richtext: live link state; OnLinkClick carries gtx per GX.8.
	g.rtState = richtext.NewState()
	g.rtStyle = richtext.FromTokens(tokens.DefaultLight, tokens.DefaultTypeScale)
	g.rtStyle.OnLinkClick = func(_ layout.Context, url string) {
		g.rtLastURL = url
		g.rtClicks++
		w.Invalidate()
	}

	// Icon registry — register the IVG icon and obtain a render widget from it.
	g.iconReg = icon.New()
	g.iconReg.Register("info", icon.FromIVG(actionInfoIVG))
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

	// Coordination: Subject[string] for producer/consumer demo.
	var coordObs rx.Observable[string]
	g.coordObserver, coordObs = coordination.Subject[string](coordination.BufCapSignal)
	g.coordSub = coordObs.Subscribe(rx.GoroutineContext(), func(msg string, err error, done bool) {
		if !done {
			g.coordMu.Lock()
			g.coordMsg = msg
			g.coordMu.Unlock()
			w.Invalidate()
		}
	})

	return g
}

func (g *gallery) cleanup() {
	if g.a11ySub != nil {
		g.a11ySub.Unsubscribe()
	}
	if g.coordSub != nil {
		g.coordSub.Unsubscribe()
	}
}

// ── Frame layout ──────────────────────────────────────────────────────────────

func (g *gallery) frame(gtx layout.Context) layout.Dimensions {
	paint.FillShape(gtx.Ops, tokens.DefaultLight.Background, clip.Rect{Max: gtx.Constraints.Max}.Op())
	return layout.Flex{}.Layout(gtx,
		layout.Rigid(g.sidebar),
		layout.Flexed(1, g.content),
	)
}

func (g *gallery) sidebar(gtx layout.Context) layout.Dimensions {
	const sideW = unit.Dp(150)
	w := gtx.Dp(sideW)
	gtx.Constraints = layout.Exact(image.Pt(w, gtx.Constraints.Max.Y))

	paint.FillShape(gtx.Ops, color.NRGBA{R: 0xf1, G: 0xf5, B: 0xf9, A: 0xff}, clip.Rect{Max: gtx.Constraints.Max}.Op())

	cs := make([]layout.FlexChild, 0, 1+len(pageNames))
	cs = append(cs, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
		return prismlayout.Inset(16).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return g.label(gtx, "Prism Gallery", tokens.DefaultLight.Text, unit.Sp(13), font.Font{Weight: font.Bold})
		})
	}))
	for i, name := range pageNames {
		i, name := i, name
		cs = append(cs, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if g.nav[i].Clicked(gtx) {
				g.page = i
			}
			active := g.page == i
			bg := tokens.DefaultLight.Background
			fg := tokens.DefaultLight.Text
			if active {
				bg = tokens.DefaultLight.Primary
				fg = tokens.DefaultLight.OnPrimary
			}
			return g.nav[i].Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				sz := image.Pt(gtx.Constraints.Max.X, gtx.Dp(unit.Dp(40)))
				paint.FillShape(gtx.Ops, bg, clip.Rect{Max: sz}.Op())
				return prismlayout.InsetXY(16, 10).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return g.label(gtx, name, fg, unit.Sp(14), font.Font{})
				})
			})
		}))
	}
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx, cs...)
}

func (g *gallery) content(gtx layout.Context) layout.Dimensions {
	paint.FillShape(gtx.Ops, tokens.DefaultLight.Background, clip.Rect{Max: gtx.Constraints.Max}.Op())
	switch g.page {
	case pageButton:
		return g.pageButton(gtx)
	case pageInputs:
		return g.pageInputs(gtx)
	case pageList:
		return g.pageList(gtx)
	case pageRichtext:
		return g.pageRichtext(gtx)
	case pageIcon:
		return g.pageIcon(gtx)
	case pageLayout:
		return g.pageLayout(gtx)
	case pageA11y:
		return g.pageA11y(gtx)
	case pageInitial:
		return g.pageInitial(gtx)
	case pageCoord:
		return g.pageCoord(gtx)
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
				return prismlayout.Inset(24).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							gtx.Constraints.Max.X = gtx.Dp(unit.Dp(200))
							gtx.Constraints.Min.X = gtx.Dp(unit.Dp(200))
							if g.btnLive != nil {
								return g.btnLive(gtx)
							}
							return layout.Dimensions{}
						}),
						layout.Rigid(prismlayout.HSpacer(16)),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return g.label(gtx, fmt.Sprintf("Clicks: %d", g.btnClicks),
								tokens.DefaultLight.Text, unit.Sp(14), font.Font{})
						}),
					)
				})
			}),
			g.sectionHeader("Button — static prism.Button vs pulse.SpringButton (press to see physics)"),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return prismlayout.Inset(24).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									gtx.Constraints.Min.X = gtx.Dp(unit.Dp(80))
									gtx.Constraints.Max.X = gtx.Dp(unit.Dp(80))
									return g.label(gtx, "Static", tokens.DefaultLight.Secondary, unit.Sp(13), font.Font{})
								}),
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									gtx.Constraints.Max.X = gtx.Dp(unit.Dp(200))
									gtx.Constraints.Min.X = gtx.Dp(unit.Dp(200))
									if g.btnCompare != nil {
										return g.btnCompare(gtx)
									}
									return layout.Dimensions{}
								}),
								layout.Rigid(prismlayout.HSpacer(16)),
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									return g.label(gtx, fmt.Sprintf("Clicks: %d", g.btnCompareClicks),
										tokens.DefaultLight.Text, unit.Sp(14), font.Font{})
								}),
							)
						}),
						layout.Rigid(prismlayout.VSpacer(16)),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									gtx.Constraints.Min.X = gtx.Dp(unit.Dp(80))
									gtx.Constraints.Max.X = gtx.Dp(unit.Dp(80))
									return g.label(gtx, "Spring", tokens.DefaultLight.Secondary, unit.Sp(13), font.Font{})
								}),
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									gtx.Constraints.Max.X = gtx.Dp(unit.Dp(200))
									gtx.Constraints.Min.X = gtx.Dp(unit.Dp(200))
									if g.springBtnLive != nil {
										return g.springBtnLive(gtx)
									}
									return layout.Dimensions{}
								}),
								layout.Rigid(prismlayout.HSpacer(16)),
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									return g.label(gtx, fmt.Sprintf("Clicks: %d", g.springBtnClicks),
										tokens.DefaultLight.Text, unit.Sp(14), font.Font{})
								}),
							)
						}),
						layout.Rigid(prismlayout.VSpacer(12)),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return g.label(gtx,
								"pulse.SpringButton(theme, button.Props{...}, springbutton.Options{}) — DESIGN §Phase 3 — Composition mechanism.",
								tokens.DefaultLight.Secondary, unit.Sp(13), font.Font{})
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
		colors tokens.ColorTokens
		rowBg  color.NRGBA
	}
	rows := []row{
		{"Normal (light)", button.RenderState{}, tokens.DefaultLight, tokens.DefaultLight.Background},
		{"Hovered (light)", button.RenderState{Hovered: true}, tokens.DefaultLight, tokens.DefaultLight.Background},
		{"Focused (light)", button.RenderState{Focused: true}, tokens.DefaultLight, tokens.DefaultLight.Background},
		{"Pressed (light)", button.RenderState{Pressed: true}, tokens.DefaultLight, tokens.DefaultLight.Background},
		{"Disabled (light)", button.RenderState{Disabled: true}, tokens.DefaultLight, tokens.DefaultLight.Background},
		{"Normal (dark)", button.RenderState{}, tokens.DefaultDark, tokens.DefaultDark.Background},
		{"Focused (dark)", button.RenderState{Focused: true}, tokens.DefaultDark, tokens.DefaultDark.Background},
		{"Pressed (dark)", button.RenderState{Pressed: true}, tokens.DefaultDark, tokens.DefaultDark.Background},
		{"Disabled (dark)", button.RenderState{Disabled: true}, tokens.DefaultDark, tokens.DefaultDark.Background},
	}
	cs := make([]layout.FlexChild, len(rows))
	for i, r := range rows {
		r := r
		w := button.Render(g.shaper, "Button", r.colors, tokens.Spacing, tokens.Radius, tokens.DefaultTypeScale, r.state)
		cs[i] = layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return g.variantRow(gtx, r.label, r.rowBg, r.colors.Text, w)
		})
	}
	return cs
}

// ── Inputs page ───────────────────────────────────────────────────────────────

func (g *gallery) pageInputs(gtx layout.Context) layout.Dimensions {
	return g.scrollPage(gtx, g.scrollSt[pageInputs], func(gtx layout.Context) layout.Dimensions {
		cs := []layout.FlexChild{g.sectionHeader("TextField — variants")}
		cs = append(cs, g.textFieldVariantRows()...)
		cs = append(cs,
			g.sectionHeader("TextField — live"),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return prismlayout.Inset(24).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					gtx.Constraints.Max.X = gtx.Dp(unit.Dp(300))
					gtx.Constraints.Min.X = gtx.Dp(unit.Dp(300))
					if g.tfLive != nil {
						return g.tfLive(gtx)
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
				return prismlayout.Inset(24).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
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
				return prismlayout.Inset(24).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							if g.rbALive != nil {
								return g.rbALive(gtx)
							}
							return layout.Dimensions{}
						}),
						layout.Rigid(prismlayout.HSpacer(8)),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return g.label(gtx, "Option A", tokens.DefaultLight.Text, unit.Sp(14), font.Font{})
						}),
						layout.Rigid(prismlayout.HSpacer(32)),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							if g.rbBLive != nil {
								return g.rbBLive(gtx)
							}
							return layout.Dimensions{}
						}),
						layout.Rigid(prismlayout.HSpacer(8)),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return g.label(gtx, "Option B", tokens.DefaultLight.Text, unit.Sp(14), font.Font{})
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
				return prismlayout.Inset(24).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
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
		colors tokens.ColorTokens
	}
	rows := []row{
		{"Normal (light)", input.RenderState{}, tokens.DefaultLight},
		{"Focused (light)", input.RenderState{Focused: true}, tokens.DefaultLight},
		{"Disabled (light)", input.RenderState{Disabled: true}, tokens.DefaultLight},
		{"Normal (dark)", input.RenderState{}, tokens.DefaultDark},
		{"Focused (dark)", input.RenderState{Focused: true}, tokens.DefaultDark},
		{"Disabled (dark)", input.RenderState{Disabled: true}, tokens.DefaultDark},
	}
	cs := make([]layout.FlexChild, len(rows))
	for i, r := range rows {
		r := r
		w := input.Render(g.shaper, "Placeholder…", r.colors, tokens.Spacing, tokens.Radius, tokens.DefaultTypeScale, r.state)
		cs[i] = layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return g.variantRow(gtx, r.label, r.colors.Background, r.colors.Text, w)
		})
	}
	return cs
}

func (g *gallery) checkboxVariantRows() []layout.FlexChild {
	type row struct {
		label  string
		state  input.CheckboxRenderState
		colors tokens.ColorTokens
	}
	rows := []row{
		{"Unchecked (light)", input.CheckboxRenderState{}, tokens.DefaultLight},
		{"Checked (light)", input.CheckboxRenderState{Checked: true}, tokens.DefaultLight},
		{"Focused (light)", input.CheckboxRenderState{Focused: true}, tokens.DefaultLight},
		{"Disabled (light)", input.CheckboxRenderState{Disabled: true}, tokens.DefaultLight},
		{"Unchecked (dark)", input.CheckboxRenderState{}, tokens.DefaultDark},
		{"Checked (dark)", input.CheckboxRenderState{Checked: true}, tokens.DefaultDark},
	}
	cs := make([]layout.FlexChild, len(rows))
	for i, r := range rows {
		r := r
		w := input.RenderCheckbox(r.colors, tokens.Spacing, tokens.Radius, r.state)
		cs[i] = layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return g.variantRow(gtx, r.label, r.colors.Background, r.colors.Text, w)
		})
	}
	return cs
}

func (g *gallery) radioVariantRows() []layout.FlexChild {
	type row struct {
		label  string
		state  input.RadioRenderState
		colors tokens.ColorTokens
	}
	rows := []row{
		{"Unselected (light)", input.RadioRenderState{}, tokens.DefaultLight},
		{"Selected (light)", input.RadioRenderState{Selected: true}, tokens.DefaultLight},
		{"Focused (light)", input.RadioRenderState{Focused: true}, tokens.DefaultLight},
		{"Disabled (light)", input.RadioRenderState{Disabled: true}, tokens.DefaultLight},
		{"Unselected (dark)", input.RadioRenderState{}, tokens.DefaultDark},
		{"Selected (dark)", input.RadioRenderState{Selected: true}, tokens.DefaultDark},
	}
	cs := make([]layout.FlexChild, len(rows))
	for i, r := range rows {
		r := r
		w := input.RenderRadio(r.colors, tokens.Spacing, tokens.Radius, r.state)
		cs[i] = layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return g.variantRow(gtx, r.label, r.colors.Background, r.colors.Text, w)
		})
	}
	return cs
}

func (g *gallery) dropdownVariantRows() []layout.FlexChild {
	opts := []string{"Apple", "Banana", "Cherry"}
	type row struct {
		label  string
		state  input.DropdownRenderState
		colors tokens.ColorTokens
	}
	rows := []row{
		{"Closed (light)", input.DropdownRenderState{Options: opts, Selected: 0}, tokens.DefaultLight},
		{"Focused (light)", input.DropdownRenderState{Options: opts, Focused: true}, tokens.DefaultLight},
		{"Open (light)", input.DropdownRenderState{Options: opts, Open: true, Selected: 1}, tokens.DefaultLight},
		{"Disabled (light)", input.DropdownRenderState{Options: opts, Disabled: true}, tokens.DefaultLight},
		{"Closed (dark)", input.DropdownRenderState{Options: opts}, tokens.DefaultDark},
		{"Open (dark)", input.DropdownRenderState{Options: opts, Open: true, Selected: 0}, tokens.DefaultDark},
	}
	cs := make([]layout.FlexChild, len(rows))
	for i, r := range rows {
		r := r
		w := input.RenderDropdown(g.shaper, r.colors, tokens.Spacing, tokens.Radius, tokens.DefaultTypeScale, r.state)
		cs[i] = layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return g.variantRow(gtx, r.label, r.colors.Background, r.colors.Text, w)
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
				return prismlayout.InsetXY(24, 8).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return g.label(gtx,
						"list.LayoutScrollbar(..., Occupy, ...) reserves a gutter; Overlay floats the bar over the rows. Wheel, thumb-drag, and track-click all scroll.",
						tokens.DefaultLight.Secondary, unit.Sp(13), font.Font{})
				})
			}),
			g.sectionHeader("Scrollbar — standalone bar beside tall fake content (drag the thumb, click the track, hover)"),
			layout.Rigid(g.scrollbarDemo),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return prismlayout.InsetXY(24, 8).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return g.label(gtx,
						"scrollbar.FromTokens(...).Layout with fractions from scrollbar.FromListPosition; drags feed back via ScrollDistance.",
						tokens.DefaultLight.Secondary, unit.Sp(13), font.Font{})
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
	return prismlayout.InsetXY(24, 12).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		h := gtx.Dp(unit.Dp(400))
		gtx.Constraints.Max.Y = h
		bar := scrollbar.FromTokens(tokens.DefaultLight)
		row := func(gtx layout.Context, item string) layout.Dimensions {
			return prismlayout.InsetXY(0, 10).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return g.label(gtx, item, tokens.DefaultLight.Text, unit.Sp(14), font.Font{})
			})
		}
		column := func(title string, st *list.State, anchor list.Anchor) layout.Widget {
			return func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return g.label(gtx, title, tokens.DefaultLight.Secondary, unit.Sp(13), font.Font{Weight: font.Bold})
					}),
					layout.Rigid(prismlayout.VSpacer(8)),
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
						gtx.Constraints.Min = gtx.Constraints.Max
						return list.LayoutScrollbar(gtx, st, bar, anchor, g.listItems, row)
					}),
				)
			}
		}
		return layout.Flex{}.Layout(gtx,
			layout.Flexed(0.5, column("Occupy — gutter reserved", g.listSt, list.Occupy)),
			layout.Rigid(prismlayout.HSpacer(24)),
			layout.Flexed(0.5, column("Overlay — bar floats over rows", g.listOverlaySt, list.Overlay)),
		)
	})
}

// scrollbarDemo lays out a 300dp-tall fake-content column with a standalone
// scrollbar beside it. The bar's fractions come from FromListPosition over the
// raw layout.List position, and scrollbar drags and track clicks are fed back
// into the list via ScrollDistance.
func (g *gallery) scrollbarDemo(gtx layout.Context) layout.Dimensions {
	return prismlayout.InsetXY(24, 12).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		h := gtx.Dp(unit.Dp(300))
		gtx.Constraints.Max.Y = h
		gtx.Constraints.Min.Y = h
		style := scrollbar.FromTokens(tokens.DefaultLight)
		barW := gtx.Dp(style.Width())
		// Both children are Rigid so the list lays out first: the bar then
		// reads this frame's scroll position instead of lagging one frame.
		return layout.Flex{}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				gtx.Constraints.Max.X -= barW
				gtx.Constraints.Min = gtx.Constraints.Max
				return g.sbList.Layout(gtx, len(g.sbItems), func(gtx layout.Context, i int) layout.Dimensions {
					return prismlayout.InsetXY(0, 8).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return g.label(gtx, g.sbItems[i], tokens.DefaultLight.Text, unit.Sp(14), font.Font{})
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

// ── Richtext page ─────────────────────────────────────────────────────────────

// richtextSpans is the demo paragraph exercising the span model: regular,
// bold, italic, monospace, explicit colour, explicit size, and two links.
func richtextSpans() []richtext.SpanStyle {
	return []richtext.SpanStyle{
		{Content: "Richtext lays out "},
		{Content: "bold", Weight: font.Bold},
		{Content: ", "},
		{Content: "italic", Style: font.Italic},
		{Content: ", "},
		{Content: "monospace", Typeface: "Go Mono, monospace"},
		{Content: ", "},
		{Content: "coloured", Color: tokens.Red.C600},
		{Content: ", and "},
		{Content: "resized", Size: 22},
		{Content: " spans in one wrapped paragraph, with inline links to "},
		{Content: "gioui.org", URL: "https://gioui.org"},
		{Content: " and the "},
		{Content: "vibrantgio design system", URL: "https://github.com/vibrantgio"},
		{Content: "."},
	}
}

func (g *gallery) pageRichtext(gtx layout.Context) layout.Dimensions {
	return g.scrollPage(gtx, g.scrollSt[pageRichtext], func(gtx layout.Context) layout.Dimensions {
		staticStyle := richtext.FromTokens(tokens.DefaultLight, tokens.DefaultTypeScale)
		linkStates := []struct {
			label string
			state richtext.RenderState
		}{
			{"Idle", richtext.Idle()},
			{"Hovered (link 0)", richtext.RenderState{HoveredLink: 0, FocusedLink: richtext.NoLink}},
			{"Focused (link 0)", richtext.RenderState{HoveredLink: richtext.NoLink, FocusedLink: 0}},
		}

		cs := []layout.FlexChild{
			g.sectionHeader("Richtext — mixed spans (bold / italic / mono / colour / size / links)"),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return prismlayout.Inset(24).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					gtx.Constraints.Max.X = gtx.Dp(unit.Dp(520))
					return richtext.Render(g.shaper, staticStyle, richtextSpans(), richtext.Idle())(gtx)
				})
			}),
			g.sectionHeader("Richtext — link states (static RenderState)"),
		}
		for _, ls := range linkStates {
			ls := ls
			cs = append(cs, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return prismlayout.InsetXY(24, 10).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							gtx.Constraints.Min.X = gtx.Dp(unit.Dp(160))
							gtx.Constraints.Max.X = gtx.Dp(unit.Dp(160))
							return g.label(gtx, ls.label, tokens.DefaultLight.Secondary, unit.Sp(13), font.Font{})
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							gtx.Constraints.Max.X = gtx.Dp(unit.Dp(400))
							spans := []richtext.SpanStyle{
								{Content: "Read the "},
								{Content: "documentation", URL: "https://gioui.org/doc"},
								{Content: " for details."},
							}
							return richtext.Render(g.shaper, staticStyle, spans, ls.state)(gtx)
						}),
					)
				})
			}))
		}
		cs = append(cs,
			g.sectionHeader("Richtext — live links (hover for cursor, Tab to focus, click or Space/Enter to activate)"),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return prismlayout.Inset(24).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							gtx.Constraints.Max.X = gtx.Dp(unit.Dp(520))
							return richtext.Layout(gtx, g.rtState, g.shaper, g.rtStyle, richtextSpans())
						}),
						layout.Rigid(prismlayout.VSpacer(16)),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							status := "Activated: (none yet — click a link above)"
							if g.rtLastURL != "" {
								status = fmt.Sprintf("Activated %d×, last: %s", g.rtClicks, g.rtLastURL)
							}
							return g.label(gtx, status, tokens.DefaultLight.Text, unit.Sp(14), font.Font{})
						}),
						layout.Rigid(prismlayout.VSpacer(8)),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return g.label(gtx,
								"richtext.Layout(gtx, state, shaper, style, spans) — OnLinkClick(gtx, url) carries gtx per GX.8.",
								tokens.DefaultLight.Secondary, unit.Sp(13), font.Font{})
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
				return prismlayout.Inset(24).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							if g.ivgWidget != nil {
								return g.ivgWidget(gtx)
							}
							return layout.Dimensions{Size: image.Pt(64, 64)}
						}),
						layout.Rigid(prismlayout.HSpacer(16)),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return g.label(gtx, "icon.FromIVG(data) + ivgraster.Widget", tokens.DefaultLight.Text, unit.Sp(13), font.Font{})
						}),
					)
				})
			}),
			g.sectionHeader("Icon — Registry (live lookup)"),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return prismlayout.Inset(24).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					_, registered := g.iconReg.Icon("info")
					status := fmt.Sprintf(`Registry has "info": %v  (kind=IVG, from icon.FromIVG)`, registered)
					return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return g.label(gtx, status, tokens.DefaultLight.Text, unit.Sp(14), font.Font{})
						}),
						layout.Rigid(prismlayout.VSpacer(8)),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return g.label(gtx, "SVG icons: icon.FromSVG(parsed *svg.Icon) for the KindSVG path.", tokens.DefaultLight.Secondary, unit.Sp(13), font.Font{})
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
				return prismlayout.Inset(24).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return prismlayout.Row(gtx,
						box(red, 48), prismlayout.HSpacer(8),
						box(green, 48), prismlayout.HSpacer(8),
						box(blue, 48),
					)
				})
			}),
			g.sectionHeader("Layout — Col (vertical flex with VSpacer gaps)"),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return prismlayout.Inset(24).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return prismlayout.Col(gtx,
						box(red, 40), prismlayout.VSpacer(8),
						box(green, 40), prismlayout.VSpacer(8),
						box(blue, 40),
					)
				})
			}),
			g.sectionHeader("Layout — Inset (uniform 24dp left) and InsetXY (32dp×16dp right)"),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return prismlayout.Inset(24).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return prismlayout.Inset(24).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								return box(purple, 40)(gtx)
							})
						}),
						layout.Rigid(prismlayout.HSpacer(16)),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return prismlayout.InsetXY(32, 16).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
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

	hc := tokens.ColorTokens{
		Background: color.NRGBA{0xff, 0xff, 0xff, 0xff},
		Text:       color.NRGBA{0x00, 0x00, 0x00, 0xff},
		Primary:    color.NRGBA{0x00, 0x00, 0x00, 0xff},
		OnPrimary:  color.NRGBA{0xff, 0xff, 0xff, 0xff},
		Surface:    color.NRGBA{0xff, 0xff, 0xff, 0xff},
		Divider:    color.NRGBA{0x00, 0x00, 0x00, 0xff},
	}

	return g.scrollPage(gtx, g.scrollSt[pageA11y], func(gtx layout.Context) layout.Dimensions {
		cs := []layout.FlexChild{
			g.sectionHeader("A11y — live OS accessibility preferences (polled every 2s)"),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return prismlayout.Inset(24).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return g.prefRow(gtx, "ReduceMotion", prefs.ReduceMotion)
						}),
						layout.Rigid(prismlayout.VSpacer(8)),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return g.prefRow(gtx, "HighContrast", prefs.HighContrast)
						}),
						layout.Rigid(prismlayout.VSpacer(8)),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return g.prefRow(gtx, "IncreaseTextSize", prefs.IncreaseTextSize)
						}),
						layout.Rigid(prismlayout.VSpacer(12)),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return g.label(gtx, "Toggle Reduce Motion in System Settings > Accessibility > Display.", tokens.DefaultLight.Secondary, unit.Sp(13), font.Font{})
						}),
					)
				})
			}),
			g.sectionHeader("A11y — high-contrast mode: swapped ColorTokens"),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return prismlayout.Inset(24).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							gtx.Constraints.Max.X = gtx.Dp(unit.Dp(200))
							gtx.Constraints.Min.X = gtx.Dp(unit.Dp(200))
							return button.Render(g.shaper, "High Contrast", hc, tokens.Spacing, tokens.Radius, tokens.DefaultTypeScale, button.RenderState{})(gtx)
						}),
						layout.Rigid(prismlayout.HSpacer(16)),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return g.label(gtx, "ColorTokens with maximum contrast ratios.", tokens.DefaultLight.Text, unit.Sp(13), font.Font{})
						}),
					)
				})
			}),
			g.sectionHeader("A11y — reduced motion: disable animations"),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return prismlayout.Inset(24).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					status := "animations enabled"
					if prefs.ReduceMotion {
						status = "reduced motion: skip animations"
					}
					return g.label(gtx, status, tokens.DefaultLight.Text, unit.Sp(14), font.Font{})
				})
			}),
		}
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx, cs...)
	})
}

func (g *gallery) prefRow(gtx layout.Context, name string, value bool) layout.Dimensions {
	indicator, col := "◯ off", tokens.DefaultLight.Secondary
	if value {
		indicator, col = "● on", tokens.DefaultLight.Primary
	}
	return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			gtx.Constraints.Min.X = gtx.Dp(unit.Dp(200))
			return g.label(gtx, name, tokens.DefaultLight.Text, unit.Sp(14), font.Font{})
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
				return prismlayout.Inset(24).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return g.label(gtx,
								fmt.Sprintf("Gallery opened at: %s", firstFrame.Format("15:04:05.000")),
								tokens.DefaultLight.Text, unit.Sp(14), font.Font{})
						}),
						layout.Rigid(prismlayout.VSpacer(8)),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return g.label(gtx,
								"initial.Value[T].GetOrSet(fn) calls fn once on the first invocation",
								tokens.DefaultLight.Secondary, unit.Sp(13), font.Font{})
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return g.label(gtx,
								"and returns the cached result on every subsequent frame.",
								tokens.DefaultLight.Secondary, unit.Sp(13), font.Font{})
						}),
						layout.Rigid(prismlayout.VSpacer(8)),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return g.label(gtx,
								"Use inside rx.Defer closures to replace ad-hoc -1 sentinels.",
								tokens.DefaultLight.Secondary, unit.Sp(13), font.Font{})
						}),
					)
				})
			}),
		}
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx, cs...)
	})
}

// ── Coordination page ─────────────────────────────────────────────────────────

func (g *gallery) pageCoord(gtx layout.Context) layout.Dimensions {
	if g.coordSend.Clicked(gtx) {
		g.coordObserver.Next(fmt.Sprintf("ping at %s", time.Now().Format("15:04:05.000")))
	}
	g.coordMu.Lock()
	msg := g.coordMsg
	g.coordMu.Unlock()

	received := msg
	if received == "" {
		received = "(none yet — click Send ping)"
	}

	return g.scrollPage(gtx, g.scrollSt[pageCoord], func(gtx layout.Context) layout.Dimensions {
		cs := []layout.FlexChild{
			g.sectionHeader("Coordination — Subject[string] producer/consumer"),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return prismlayout.Inset(24).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return g.coordSend.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								return button.Render(g.shaper, "Send ping", tokens.DefaultLight, tokens.Spacing, tokens.Radius, tokens.DefaultTypeScale, button.RenderState{})(gtx)
							})
						}),
						layout.Rigid(prismlayout.VSpacer(16)),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return g.label(gtx, "Consumer received: "+received,
								tokens.DefaultLight.Text, unit.Sp(14), font.Font{})
						}),
						layout.Rigid(prismlayout.VSpacer(16)),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return g.label(gtx,
								"observer.Next(msg) sends via coordination.Subject[string](BufCapSignal).",
								tokens.DefaultLight.Secondary, unit.Sp(13), font.Font{})
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return g.label(gtx,
								"Consumer goroutine calls w.Invalidate() to schedule the next frame.",
								tokens.DefaultLight.Secondary, unit.Sp(13), font.Font{})
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
		bg := color.NRGBA{R: 0xe2, G: 0xe8, B: 0xf0, A: 0xff}
		h := gtx.Dp(unit.Dp(36))
		paint.FillShape(gtx.Ops, bg, clip.Rect{Max: image.Pt(gtx.Constraints.Max.X, h)}.Op())
		return prismlayout.InsetXY(24, 8).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return g.label(gtx, title, tokens.DefaultLight.Text, unit.Sp(13), font.Font{Weight: font.Bold})
		})
	})
}

func (g *gallery) variantRow(gtx layout.Context, lbl string, bg, fg color.NRGBA, w layout.Widget) layout.Dimensions {
	paint.FillShape(gtx.Ops, bg, clip.Rect{Max: image.Pt(gtx.Constraints.Max.X, gtx.Dp(unit.Dp(64)))}.Op())
	return prismlayout.InsetXY(24, 10).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
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

func (g *gallery) label(gtx layout.Context, s string, col color.NRGBA, size unit.Sp, f font.Font) layout.Dimensions {
	m := op.Record(gtx.Ops)
	paint.ColorOp{Color: col}.Add(gtx.Ops)
	mat := m.Stop()
	lbl := widget.Label{MaxLines: 1}
	return lbl.Layout(gtx, g.shaper, f, size, s, mat)
}

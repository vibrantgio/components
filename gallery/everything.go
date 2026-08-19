// The everything page: the whole inventory in one scrollable column.
//
// The per-family pages beside it are for close-up work on one widget at a
// time — they carry the live interactions, the variant grids and the running
// commentary. This page carries none of that. It exists to be looked at
// whole, because that is the only way a theme can be judged: a seed that
// flatters a button in isolation can still leave the tag row muddy against
// the card it sits on, and nothing but the two side by side will say so.
//
// The sections themselves live in the inventory package, which draws them
// from the colour tokens it is handed and nothing else. What is here is the
// page around them: the ground, the viewport, and the banner with the switch
// that redraws the lot in the other scheme.
package main

import (
	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"

	"github.com/vibrantgio/components/gallery/inventory"
	complayout "github.com/vibrantgio/components/layout"
	"github.com/vibrantgio/components/list"
	"github.com/vibrantgio/theme/tokens"
)

// colors returns the scheme the everything page and the two inventory pages
// are drawn in. The per-family pages predate the switch and stay on the light
// scheme; theirs is close-up work on one widget, and the whole-surface
// judgment this switch serves is what the everything page is for.
func (g *gallery) colors() tokens.ColorTokens {
	if g.dark {
		return tokens.DefaultDark
	}
	return tokens.DefaultLight
}

// chrome returns the tokens the window's own furniture — the ground under a
// page and the sidebar beside it — is drawn in. It follows the page: an
// inventory page takes the scheme its switch is set to, and a per-family
// page, which draws light-scheme widgets whatever the switch says, keeps the
// chrome on the light scheme with them.
func (g *gallery) chrome() tokens.ColorTokens {
	switch g.page {
	case pageEverything, pagePatterns, pageMarkdown:
		return g.colors()
	}
	return tokens.DefaultLight
}

// pageEverything lays out every group of the inventory, one after the other,
// in the current scheme.
func (g *gallery) pageEverything(gtx layout.Context) layout.Dimensions {
	c := g.colors()
	items := []layout.Widget{g.pageBanner(c,
		"Everything",
		"Every published family in one column, in the current theme.")}
	items = append(items, g.inv.Items(c)...)
	return g.scrollItems(gtx, g.scrollSt[pageEverything], c, items)
}

func (g *gallery) pagePatterns(gtx layout.Context) layout.Dimensions {
	c := g.colors()
	items := []layout.Widget{g.pageBanner(c,
		"Patterns",
		"The compositions, drawn from static state in bounded slots.")}
	items = append(items, g.inv.GroupItems(c, inventory.Group{Name: "Patterns", Sections: g.inv.Patterns(c)})...)
	return g.scrollItems(gtx, g.scrollSt[pagePatterns], c, items)
}

func (g *gallery) pageMarkdown(gtx layout.Context) layout.Dimensions {
	c := g.colors()
	items := []layout.Widget{g.pageBanner(c,
		"Markdown",
		"A syntax specimen and a reading sample, laid out by the renderer itself.")}
	items = append(items, g.inv.GroupItems(c, inventory.Group{Name: "Markdown", Sections: g.inv.Reading(c)})...)
	return g.scrollItems(gtx, g.scrollSt[pageMarkdown], c, items)
}

// scrollItems shows items in one scrolling column on the scheme's ground.
// Only the rows that show are laid out, which is what keeps a page of three
// dozen sections cheap.
func (g *gallery) scrollItems(gtx layout.Context, st *list.State, c tokens.ColorTokens, items []layout.Widget) layout.Dimensions {
	paint.FillShape(gtx.Ops, c.Background, clip.Rect{Max: gtx.Constraints.Max}.Op())
	return list.Layout(gtx, st, items, func(gtx layout.Context, w layout.Widget) layout.Dimensions {
		return w(gtx)
	})
}

// pageBanner is the row at the top of an inventory page: what the page is,
// and the switch that redraws it in the other scheme.
func (g *gallery) pageBanner(c tokens.ColorTokens, title, subtitle string) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		if g.schemeBtn.Clicked(gtx) {
			g.dark = !g.dark
		}
		return complayout.InsetXY(24, 20).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return inventory.LabelAt(gtx, g.shaper, title, c.Text, 22, font.Font{Weight: font.Bold})
						}),
						layout.Rigid(complayout.VSpacer(6)),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return inventory.LabelAt(gtx, g.shaper, subtitle, c.Ramps.Neutral.Step(600), 13, font.Font{})
						}),
					)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return g.schemeBtn.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return inventory.SchemeSwitch(g.shaper, c, g.dark)(gtx)
					})
				}),
			)
		})
	}
}

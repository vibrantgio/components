// The everything page: the whole inventory in one scrollable column.
//
// The per-family pages beside it are for close-up work on one widget at a
// time — they carry the live interactions, the variant grids and the running
// commentary. This page carries none of that. It exists to be looked at
// whole, because that is the only way a theme can be judged: a seed that
// flatters a button in isolation can still leave the tag row muddy against
// the card it sits on, and nothing but the two side by side will say so.
package main

import (
	"fmt"
	"image"
	"image/color"

	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"

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

func (g *gallery) schemeName() string {
	if g.dark {
		return "Dark"
	}
	return "Light"
}

// pageEverythingBody lays out every group of the inventory, one after the
// other, in the current scheme.
func (g *gallery) pageEverything(gtx layout.Context) layout.Dimensions {
	c := g.colors()
	items := []layout.Widget{g.pageBanner(c,
		"Everything",
		"Every published family in one column, in the current theme.")}
	groups := g.inv.groups(c)
	total := 0
	for _, grp := range groups {
		total += len(grp.sections)
		items = append(items, sectionItems(g.inv, c, grp)...)
	}
	items = append(items, pageEnd(g.shaper, c, total))
	return g.scrollItems(gtx, g.scrollSt[pageEverything], c, items)
}

func (g *gallery) pagePatterns(gtx layout.Context) layout.Dimensions {
	c := g.colors()
	items := []layout.Widget{g.pageBanner(c,
		"Patterns",
		"The compositions, drawn from static state in bounded slots.")}
	items = append(items, sectionItems(g.inv, c, group{name: "Patterns", sections: g.inv.patterns(c)})...)
	return g.scrollItems(gtx, g.scrollSt[pagePatterns], c, items)
}

func (g *gallery) pageMarkdown(gtx layout.Context) layout.Dimensions {
	c := g.colors()
	items := []layout.Widget{g.pageBanner(c,
		"Markdown",
		"A reading sample laid out by the renderer itself.")}
	items = append(items, sectionItems(g.inv, c, group{name: "Markdown", sections: g.inv.reading(c)})...)
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
							return labelAt(gtx, g.shaper, title, c.Text, 22, font.Font{Weight: font.Bold})
						}),
						layout.Rigid(complayout.VSpacer(6)),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return labelAt(gtx, g.shaper, subtitle, c.Ramps.Neutral.Step(600), 13, font.Font{})
						}),
					)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return g.schemeBtn.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return schemeSwitch(g.shaper, c, g.schemeName())(gtx)
					})
				}),
			)
		})
	}
}

// schemeSwitch draws the light/dark switch as a pill carrying the scheme it
// will show next, so what the control does is readable without pressing it.
func schemeSwitch(shaper *text.Shaper, c tokens.ColorTokens, current string) layout.Widget {
	next := "Dark"
	if current == "Dark" {
		next = "Light"
	}
	return func(gtx layout.Context) layout.Dimensions {
		w, h := gtx.Dp(150), gtx.Dp(34)
		r := clip.RRect{Rect: image.Rect(0, 0, w, h), SE: h / 2, SW: h / 2, NW: h / 2, NE: h / 2}
		paint.FillShape(gtx.Ops, c.Primary, r.Op(gtx.Ops))
		gtx.Constraints = layout.Exact(image.Pt(w, h))
		return complayout.InsetXY(16, 9).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return labelAt(gtx, shaper, current+" — show "+next, c.OnPrimary, 13, font.Font{})
		})
	}
}

// pageEnd closes the column. A page this tall that simply stops reads as a
// render that gave out; a line saying how much of the surface has just gone
// past says it ended on purpose.
func pageEnd(shaper *text.Shaper, c tokens.ColorTokens, sections int) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		h := gtx.Dp(64)
		sz := image.Pt(gtx.Constraints.Max.X, h)
		paint.FillShape(gtx.Ops, c.Ramps.Neutral.Step(200), clip.Rect{Max: sz}.Op())
		paint.FillShape(gtx.Ops, c.Divider, clip.Rect(image.Rect(0, 0, sz.X, 1)).Op())
		gtx.Constraints = layout.Exact(sz)
		return complayout.InsetXY(24, 24).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return labelAt(gtx, shaper,
				fmt.Sprintf("End of the inventory — %d sections in the current theme.", sections),
				c.Ramps.Neutral.Step(600), 12, font.Font{})
		})
	}
}

// sectionItems turns one group into the rows the page's list shows: a banner
// for the group, then a header and a bounded body for each section.
func sectionItems(inv *inventory, c tokens.ColorTokens, grp group) []layout.Widget {
	items := make([]layout.Widget, 0, 1+2*len(grp.sections))
	items = append(items, groupBanner(inv.shaper, c, grp.name))
	for _, s := range grp.sections {
		s := s
		items = append(items, sectionHeaderRow(inv.shaper, c, s.title), sectionBody(c, s))
	}
	return items
}

// groupBanner separates one module's families from the next.
func groupBanner(shaper *text.Shaper, c tokens.ColorTokens, name string) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		h := gtx.Dp(44)
		sz := image.Pt(gtx.Constraints.Max.X, h)
		paint.FillShape(gtx.Ops, c.Primary, clip.Rect{Max: sz}.Op())
		gtx.Constraints = layout.Exact(sz)
		return complayout.InsetXY(24, 13).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return labelAt(gtx, shaper, name, c.OnPrimary, 15, font.Font{Weight: font.Bold})
		})
	}
}

// sectionHeaderRow labels one family. Every section on the page carries one:
// an unlabelled swatch is a puzzle, not an inventory.
func sectionHeaderRow(shaper *text.Shaper, c tokens.ColorTokens, title string) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		h := gtx.Dp(32)
		sz := image.Pt(gtx.Constraints.Max.X, h)
		paint.FillShape(gtx.Ops, c.Ramps.Neutral.Step(200), clip.Rect{Max: sz}.Op())
		paint.FillShape(gtx.Ops, c.Divider,
			clip.Rect(image.Rect(0, h-1, sz.X, h)).Op())
		gtx.Constraints = layout.Exact(sz)
		return complayout.InsetXY(24, 8).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return labelAt(gtx, shaper, title, c.Text, 13, font.Font{Weight: font.Bold})
		})
	}
}

// sectionBody lays a family out in a slot of its own. The height is the
// section's, not the content's: several patterns expand into whatever
// constraints they are handed, and one of those left unbounded would swallow
// the rest of the column.
func sectionBody(c tokens.ColorTokens, s section) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		h := gtx.Dp(s.height) + gtx.Dp(40)
		full := image.Pt(gtx.Constraints.Max.X, h)
		paint.FillShape(gtx.Ops, c.Background, clip.Rect{Max: full}.Op())
		gtx.Constraints = layout.Exact(full)
		return complayout.InsetXY(24, 20).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			gtx.Constraints.Max.Y = gtx.Dp(s.height)
			gtx.Constraints.Min.Y = 0
			gtx.Constraints.Min.X = 0
			s.body(gtx)
			return layout.Dimensions{Size: image.Pt(gtx.Constraints.Max.X, gtx.Dp(s.height))}
		})
	}
}

// column stacks items top to bottom at the width it is given. It is the page
// with no viewport in front of it — what the everything page would be if it
// were printed rather than scrolled.
func column(items []layout.Widget) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		cs := make([]layout.FlexChild, len(items))
		for i, w := range items {
			cs[i] = layout.Rigid(w)
		}
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx, cs...)
	}
}

// swatchBorder is the hairline a flat swatch needs to be visible against a
// ground of nearly its own colour.
func swatchBorder(gtx layout.Context, col color.NRGBA, size image.Point, width unit.Dp) {
	w := gtx.Dp(width)
	paint.FillShape(gtx.Ops, col, clip.Rect(image.Rect(0, 0, size.X, w)).Op())
	paint.FillShape(gtx.Ops, col, clip.Rect(image.Rect(0, size.Y-w, size.X, size.Y)).Op())
	paint.FillShape(gtx.Ops, col, clip.Rect(image.Rect(0, 0, w, size.Y)).Op())
	paint.FillShape(gtx.Ops, col, clip.Rect(image.Rect(size.X-w, 0, size.X, size.Y)).Op())
}

// The column: how the sections are stacked, banded and closed off, and the
// switch that redraws the whole of it in the other scheme.
//
// A section on its own is a widget. What makes the inventory readable is the
// furniture around it — the group banner that says which module the next run
// of families comes from, the header that names each one, and the bounded
// slot each body is laid out in.
package inventory

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
	"github.com/vibrantgio/theme/tokens"
)

// Items returns the whole inventory as the rows of one column, in the given
// scheme: every group banded and labelled, and a closing line under the last
// of them. The rows are ordinary widgets, so a caller can put them in a
// scrolling list, print them into one tall column, or take a slice.
//
// The rows are a function of the palette alone, so re-theming is calling this
// again with other tokens. Nothing that survives across frames is rebuilt by
// doing so.
func (inv *Inventory) Items(c tokens.ColorTokens) []layout.Widget {
	groups := inv.Groups(c)
	items := make([]layout.Widget, 0, 8*len(groups))
	total := 0
	for _, grp := range groups {
		total += len(grp.Sections)
		items = append(items, inv.GroupItems(c, grp)...)
	}
	return append(items, inv.PageEnd(c, total))
}

// GroupItems turns one group into the rows a column shows: a banner for the
// group, then a header and a bounded body for each section.
func (inv *Inventory) GroupItems(c tokens.ColorTokens, grp Group) []layout.Widget {
	items := make([]layout.Widget, 0, 1+2*len(grp.Sections))
	items = append(items, groupBanner(inv.shaper, c, grp.Name))
	for _, s := range grp.Sections {
		s := s
		items = append(items, sectionHeaderRow(inv.shaper, c, s.Title), sectionBody(c, s))
	}
	return items
}

// PageEnd closes the column. A column this tall that simply stops reads as a
// render that gave out; a line saying how much of the surface has just gone
// past says it ended on purpose.
func (inv *Inventory) PageEnd(c tokens.ColorTokens, sections int) layout.Widget {
	return pageEnd(inv.shaper, c, sections)
}

func pageEnd(shaper *text.Shaper, c tokens.ColorTokens, sections int) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		h := gtx.Dp(64)
		sz := image.Pt(gtx.Constraints.Max.X, h)
		paint.FillShape(gtx.Ops, c.Ramps.Neutral.Step(200), clip.Rect{Max: sz}.Op())
		paint.FillShape(gtx.Ops, c.Divider, clip.Rect(image.Rect(0, 0, sz.X, 1)).Op())
		gtx.Constraints = layout.Exact(sz)
		return complayout.InsetXY(24, 24).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return LabelAt(gtx, shaper,
				fmt.Sprintf("End of the inventory — %d sections in the current theme.", sections),
				c.Ramps.Neutral.Step(600), 12, font.Font{})
		})
	}
}

// groupBanner separates one module's families from the next.
func groupBanner(shaper *text.Shaper, c tokens.ColorTokens, name string) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		h := gtx.Dp(44)
		sz := image.Pt(gtx.Constraints.Max.X, h)
		paint.FillShape(gtx.Ops, c.Primary, clip.Rect{Max: sz}.Op())
		gtx.Constraints = layout.Exact(sz)
		return complayout.InsetXY(24, 13).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return LabelAt(gtx, shaper, name, c.OnPrimary, 15, font.Font{Weight: font.Bold})
		})
	}
}

// sectionHeaderRow labels one family. Every section carries one: an
// unlabelled swatch is a puzzle, not an inventory.
func sectionHeaderRow(shaper *text.Shaper, c tokens.ColorTokens, title string) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		h := gtx.Dp(32)
		sz := image.Pt(gtx.Constraints.Max.X, h)
		paint.FillShape(gtx.Ops, c.Ramps.Neutral.Step(200), clip.Rect{Max: sz}.Op())
		paint.FillShape(gtx.Ops, c.Divider,
			clip.Rect(image.Rect(0, h-1, sz.X, h)).Op())
		gtx.Constraints = layout.Exact(sz)
		return complayout.InsetXY(24, 8).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return LabelAt(gtx, shaper, title, c.Text, 13, font.Font{Weight: font.Bold})
		})
	}
}

// sectionBody lays a family out in a slot of its own. The height is the
// section's, not the content's: several patterns expand into whatever
// constraints they are handed, and one of those left unbounded would swallow
// the rest of the column.
func sectionBody(c tokens.ColorTokens, s Section) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		h := gtx.Dp(s.Height) + gtx.Dp(40)
		full := image.Pt(gtx.Constraints.Max.X, h)
		paint.FillShape(gtx.Ops, c.Background, clip.Rect{Max: full}.Op())
		gtx.Constraints = layout.Exact(full)
		return complayout.InsetXY(24, 20).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			gtx.Constraints.Max.Y = gtx.Dp(s.Height)
			gtx.Constraints.Min.Y = 0
			gtx.Constraints.Min.X = 0
			s.Body(gtx)
			return layout.Dimensions{Size: image.Pt(gtx.Constraints.Max.X, gtx.Dp(s.Height))}
		})
	}
}

// Column stacks items top to bottom at the width it is given. It is the
// inventory with no viewport in front of it — what the column would be if it
// were printed rather than scrolled.
func Column(items []layout.Widget) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		cs := make([]layout.FlexChild, len(items))
		for i, w := range items {
			cs[i] = layout.Rigid(w)
		}
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx, cs...)
	}
}

// SchemeSwitch draws the light/dark switch as a pill carrying the scheme it
// will show next, so what the control does is readable without pressing it.
// It paints and measures only; the press belongs to whatever surface puts it
// on screen.
func SchemeSwitch(shaper *text.Shaper, c tokens.ColorTokens, dark bool) layout.Widget {
	current, next := "Light", "Dark"
	if dark {
		current, next = "Dark", "Light"
	}
	return func(gtx layout.Context) layout.Dimensions {
		w, h := gtx.Dp(150), gtx.Dp(34)
		r := clip.RRect{Rect: image.Rect(0, 0, w, h), SE: h / 2, SW: h / 2, NW: h / 2, NE: h / 2}
		paint.FillShape(gtx.Ops, c.Primary, r.Op(gtx.Ops))
		gtx.Constraints = layout.Exact(image.Pt(w, h))
		return complayout.InsetXY(16, 9).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return LabelAt(gtx, shaper, current+" — show "+next, c.OnPrimary, 13, font.Font{})
		})
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

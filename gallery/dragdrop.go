// The drag and drop page: two lists side by side, and a row moves to
// another place in its own list or into the other list by drag and drop.

package main

import (
	"image"
	"slices"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"

	"github.com/vibrantgio/components/dragdrop"
	complayout "github.com/vibrantgio/components/layout"
	vgcolor "github.com/vibrantgio/theme/color"
	"github.com/vibrantgio/theme/tokens"
	"github.com/vibrantgio/theme/typeset"
)

// The page's own measurements. They belong to the page, not to a
// component.
const (
	dragPageInset unit.Dp = 24  // the margin around the two lists
	dragListW     unit.Dp = 280 // the width of one list
	dragListGap   unit.Dp = 24  // the space between the two lists
	dragTitleGap  unit.Dp = 8   // the space between a list's title and its box
	dragBoxPad    unit.Dp = 4   // the space inside a box above the first row
	dragRowInsetX unit.Dp = 10  // the space before a row's text
	// dragBoxRows is the number of rows a box is tall. It is one more
	// than the items of both lists, so the place after the last item
	// always has a row of its own.
	dragBoxRows = 9
)

// dragRowH is the height of one row: the platform's content list row.
var dragRowH = unit.Dp(tokens.Comfortable.ContentListRowHeight)

// dragTitles are the titles of the two lists, in sequence order.
var dragTitles = [2]string{"Inbox", "Today"}

// dragDemo is the page's state: the drag area and the two lists it moves
// rows between.
type dragDemo struct {
	st    dragdrop.State
	lists [2][]string
}

func newDragDemo() *dragDemo {
	return &dragDemo{lists: [2][]string{
		{
			"Write the release notes",
			"Answer the support mail",
			"Book the train",
			"Review the pull request",
			"Order new toner",
			"Plan the team lunch",
		},
		{
			"Call the dentist",
			"Water the plants",
		},
	}}
}

// apply moves the row of m: it removes the row at From and inserts it in
// the destination list at the index the row has after the move.
func (d *dragDemo) apply(m dragdrop.Move) {
	from := d.lists[m.From.Sequence]
	row := from[m.From.Index]
	d.lists[m.From.Sequence] = slices.Delete(from, m.From.Index, m.From.Index+1)
	to := d.lists[m.To.Sequence]
	d.lists[m.To.Sequence] = slices.Insert(to, m.Index(), row)
}

// pageDragDrop lays out the page in the current scheme.
func (g *gallery) pageDragDrop(gtx layout.Context) layout.Dimensions {
	c := g.colors()
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(g.pageBanner(c, "Drag and drop",
			"Drag a row to another place in its list or into the other list. Escape keeps it where it was.")),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return complayout.Inset(float32(dragPageInset)).Layout(gtx, g.dragLists(c))
		}),
	)
}

// dragLists lays out the two lists inside one drag area, and applies the
// moves of the rows that landed.
func (g *gallery) dragLists(c tokens.PlatformColors) layout.Widget {
	d := g.drag
	style := dragdrop.FromTokens(c)
	return func(gtx layout.Context) layout.Dimensions {
		listW := gtx.Dp(dragListW)
		titleH := g.dragTitleHeight(gtx, c)
		boxH := 2*gtx.Dp(dragBoxPad) + dragBoxRows*gtx.Dp(dragRowH)
		size := image.Pt(2*listW+gtx.Dp(dragListGap), titleH+gtx.Dp(dragTitleGap)+boxH)
		// The drag area takes the maximum constraints as its own size.
		gtx.Constraints = layout.Exact(size)
		dims := d.st.Layout(gtx, style, g.dragImage(c), func(gtx layout.Context) layout.Dimensions {
			for seq := range d.lists {
				x := seq * (listW + gtx.Dp(dragListGap))
				off := op.Offset(image.Pt(x, 0)).Push(gtx.Ops)
				g.dragList(gtx, c, seq, listW, titleH, boxH)
				off.Pop()
			}
			return layout.Dimensions{Size: size}
		})
		if moves := d.st.Update(); len(moves) > 0 {
			for _, m := range moves {
				d.apply(m)
			}
			// This frame drew the rows before the move.
			gtx.Execute(op.InvalidateCmd{})
		}
		return dims
	}
}

// dragTitleHeight is the height of a list's title line.
func (g *gallery) dragTitleHeight(gtx layout.Context, c tokens.PlatformColors) int {
	m := op.Record(gtx.Ops)
	h := g.label(gtx, dragTitles[0], vgcolor.Flatten(c.Label, c.WindowBackground), bold(roles.Body)).Size.Y
	m.Stop()
	return h
}

// dragList lays out list seq: its title over a box of rows on the
// content fill.
func (g *gallery) dragList(gtx layout.Context, c tokens.PlatformColors, seq, w, titleH, boxH int) {
	d := g.drag
	g.label(gtx, dragTitles[seq], vgcolor.Flatten(c.Label, c.WindowBackground), bold(roles.Body))

	top := titleH + gtx.Dp(dragTitleGap)
	box := op.Offset(image.Pt(0, top)).Push(gtx.Ops)
	defer box.Pop()
	fill := c.ControlBackground
	paint.FillShape(gtx.Ops, fill, clip.Rect{Max: image.Pt(w, boxH)}.Op())
	border := vgcolor.Flatten(c.Separator, fill)
	lw := max(gtx.Dp(1), 1)
	for _, r := range []image.Rectangle{
		image.Rect(0, 0, w, lw),
		image.Rect(0, boxH-lw, w, boxH),
		image.Rect(0, 0, lw, boxH),
		image.Rect(w-lw, 0, w, boxH),
	} {
		paint.FillShape(gtx.Ops, border, clip.Rect(r).Op())
	}
	gtx = typeset.WithFill(gtx, fill)

	// The rows and the line stand inside the border.
	rowW := w - 2*lw
	rowH := gtx.Dp(dragRowH)
	y := gtx.Dp(dragBoxPad)
	for i, row := range d.lists[seq] {
		off := op.Offset(image.Pt(lw, y)).Push(gtx.Ops)
		ig := gtx
		ig.Constraints = layout.Exact(image.Pt(rowW, rowH))
		d.st.Item(ig, dragdrop.Place{Sequence: seq, Index: i}, 0, func(gtx layout.Context) layout.Dimensions {
			return g.dragRow(gtx, c, row)
		})
		off.Pop()
		y += rowH
	}
	off := op.Offset(image.Pt(lw, y)).Push(gtx.Ops)
	eg := gtx
	eg.Constraints = layout.Exact(image.Pt(rowW, boxH-gtx.Dp(dragBoxPad)-y))
	d.st.End(eg, seq, len(d.lists[seq]), 0)
	off.Pop()
}

// dragRow draws one row's text, centred in the row's height.
func (g *gallery) dragRow(gtx layout.Context, c tokens.PlatformColors, s string) layout.Dimensions {
	size := gtx.Constraints.Min
	// The label measures its own line box, so that the text is centred.
	gtx.Constraints.Min = image.Point{}
	m := op.Record(gtx.Ops)
	dims := g.label(gtx, s, vgcolor.Flatten(c.Label, c.ControlBackground), roles.Body)
	call := m.Stop()
	off := op.Offset(image.Pt(gtx.Dp(dragRowInsetX), (size.Y-dims.Size.Y)/2)).Push(gtx.Ops)
	call.Add(gtx.Ops)
	off.Pop()
	return layout.Dimensions{Size: size}
}

// dragImage draws the dragged row as it is in its list: the row's text
// with no fill of its own, as the platform draws the image of a dragged
// list row.
func (g *gallery) dragImage(c tokens.PlatformColors) dragdrop.Image {
	return func(gtx layout.Context, from dragdrop.Place) layout.Dimensions {
		return g.dragRow(gtx, c, g.drag.lists[from.Sequence][from.Index])
	}
}

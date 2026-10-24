package tooltip_test

import (
	"fmt"
	"image"
	"image/color"
	"testing"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"

	"github.com/vibrantgio/components/golden"
	"github.com/vibrantgio/components/tooltip"
)

// flexChild is one child of the rows TestFlexOffsetsFollowFlex compares: a
// rectangle of its own size and baseline, at least as long as the flex's
// minimum along the main axis, drawn in its own colour.
type flexChild struct {
	size     image.Point
	baseline int
	colour   color.NRGBA
}

func (fc flexChild) layout(gtx layout.Context) layout.Dimensions {
	sz := fc.size
	sz.X = max(sz.X, gtx.Constraints.Min.X)
	sz.Y = max(sz.Y, gtx.Constraints.Min.Y)
	paint.FillShape(gtx.Ops, fc.colour, clip.Rect{Max: sz}.Op())
	return layout.Dimensions{Size: sz, Baseline: fc.baseline}
}

// TestFlexOffsetsFollowFlex lays rows out with layout.Flex for every
// spacing and alignment, on both axes, with and without a gap and with and
// without a flexed child, each child drawn in its own colour, and reads
// each child's offset off the capture: flexOffsets returns the same offsets
// from the sizes the children reported.
func TestFlexOffsetsFollowFlex(t *testing.T) {
	spacings := []layout.Spacing{layout.SpaceEnd, layout.SpaceStart, layout.SpaceSides,
		layout.SpaceAround, layout.SpaceBetween, layout.SpaceEvenly}
	alignments := []layout.Alignment{layout.Start, layout.End, layout.Middle, layout.Baseline}
	colours := []color.NRGBA{{R: 0xff, A: 0xff}, {G: 0xff, A: 0xff}, {B: 0xff, A: 0xff}}
	for _, axis := range []layout.Axis{layout.Horizontal, layout.Vertical} {
		cell := axis.Convert(image.Pt(150, 60))
		pitch := cell.Add(image.Pt(10, 10))
		type row struct {
			flex    layout.Flex
			flexed  bool
			name    string
			at      image.Point
			dims    []layout.Dimensions
			flexIdx int
		}
		var rows []*row
		for si, sp := range spacings {
			col := 0
			for _, al := range alignments {
				for _, gap := range []int{0, 3} {
					for _, flexed := range []bool{false, true} {
						rows = append(rows, &row{
							flex:   layout.Flex{Axis: axis, Spacing: sp, Alignment: al, Gap: gap},
							flexed: flexed,
							name:   fmt.Sprintf("axis %v spacing %v alignment %v gap %d flexed %v", axis, sp, al, gap, flexed),
							at:     image.Pt(col*pitch.X, si*pitch.Y),
						})
						if axis == layout.Vertical {
							rows[len(rows)-1].at = image.Pt(si*pitch.X, col*pitch.Y)
						}
						col++
					}
				}
			}
		}
		children := []flexChild{
			{size: axis.Convert(image.Pt(20, 10)), baseline: 3, colour: colours[0]},
			{size: axis.Convert(image.Pt(30, 24)), baseline: 5, colour: colours[1]},
			{size: axis.Convert(image.Pt(12, 17)), baseline: 9, colour: colours[2]},
		}
		var extent image.Point
		for _, r := range rows {
			extent.X = max(extent.X, r.at.X+pitch.X)
			extent.Y = max(extent.Y, r.at.Y+pitch.Y)
		}
		img := golden.Capture(t, extent, func(gtx layout.Context) layout.Dimensions {
			paint.FillShape(gtx.Ops, color.NRGBA{A: 0xff}, clip.Rect{Max: extent}.Op())
			for _, r := range rows {
				r.dims = make([]layout.Dimensions, len(children))
				fc := make([]layout.FlexChild, len(children))
				for i, ch := range children {
					w := func(gtx layout.Context) layout.Dimensions {
						r.dims[i] = ch.layout(gtx)
						return r.dims[i]
					}
					if r.flexed && i == 1 {
						fc[i] = layout.Flexed(0.5, w)
					} else {
						fc[i] = layout.Rigid(w)
					}
				}
				st := op.Offset(r.at).Push(gtx.Ops)
				cgtx := gtx
				cgtx.Constraints = layout.Exact(cell)
				r.flex.Layout(cgtx, fc...)
				st.Pop()
			}
			return layout.Dimensions{Size: extent}
		})
		for _, r := range rows {
			want := tooltip.FlexOffsets(r.flex, layout.Exact(cell), r.dims)
			for i, ch := range children {
				got, ok := topLeftOf(img, image.Rectangle{Min: r.at, Max: r.at.Add(cell)}, ch.colour)
				if !ok {
					t.Errorf("%s: child %d was not drawn", r.name, i)
					continue
				}
				if got.Sub(r.at) != want[i] {
					t.Errorf("%s: child %d at %v in layout.Flex, flexOffsets says %v", r.name, i, got.Sub(r.at), want[i])
				}
			}
		}
	}
}

// topLeftOf returns the top-left corner of the pixels of colour c inside r.
func topLeftOf(img *image.RGBA, r image.Rectangle, c color.NRGBA) (image.Point, bool) {
	want := color.RGBA{R: c.R, G: c.G, B: c.B, A: c.A}
	corner := r.Max
	found := false
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			if img.RGBAAt(x, y) == want {
				corner = image.Pt(min(corner.X, x), min(corner.Y, y))
				found = true
			}
		}
	}
	return corner, found
}

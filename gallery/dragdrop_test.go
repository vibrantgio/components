package main

import (
	"image"
	"image/color"
	"slices"
	"testing"
	"time"

	"gioui.org/f32"
	"gioui.org/io/input"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"

	"github.com/vibrantgio/components/dragdrop"
	"github.com/vibrantgio/components/golden"
	"github.com/vibrantgio/theme/tokens"
)

// dragPageSize is the size the page is captured at. It holds the banner
// and the two lists with the page's margin around them.
var dragPageSize = image.Pt(680, 460)

// dragWindow drives the drag and drop page through a router, as a window
// does.
type dragWindow struct {
	r   input.Router
	g   *gallery
	w   layout.Widget
	now time.Time
}

func (d *dragWindow) frame() {
	d.now = d.now.Add(time.Second / 60)
	gtx := layout.Context{
		Ops:         new(op.Ops),
		Constraints: layout.Exact(dragPageSize),
		Metric:      unit.Metric{PxPerDp: 1, PxPerSp: 1},
		Source:      d.r.Source(),
		Now:         d.now,
	}
	d.w(gtx)
	d.r.Frame(gtx.Ops)
}

// pointer queues one pointer event and lays out a frame. A Drag is a
// Move with the button down, as a window delivers it.
func (d *dragWindow) pointer(kind pointer.Kind, p image.Point) {
	e := pointer.Event{Kind: kind, Source: pointer.Mouse, Position: f32.Pt(float32(p.X), float32(p.Y))}
	if kind == pointer.Press || kind == pointer.Drag {
		e.Buttons = pointer.ButtonPrimary
	}
	if kind == pointer.Drag {
		e.Kind = pointer.Move
	}
	d.r.Queue(e)
	d.frame()
}

// dragRowsOrigin returns the top left corner of the first row of the
// first list, in the page's frame.
func dragRowsOrigin(g *gallery, c tokens.PlatformColors) image.Point {
	gtx := layout.Context{
		Ops:         new(op.Ops),
		Constraints: layout.Constraints{Max: image.Pt(dragPageSize.X, 1<<20)},
		Metric:      unit.Metric{PxPerDp: 1, PxPerSp: 1},
	}
	banner := g.pageBanner(c, "", "")(gtx).Size.Y
	title := g.dragTitleHeight(gtx, c)
	inset := gtx.Dp(dragPageInset)
	return image.Pt(inset, banner+inset+title+gtx.Dp(dragTitleGap)+gtx.Dp(dragBoxPad))
}

// strongest returns the largest distance of a pixel in r from fill, as
// the sum of the three channel differences.
func strongest(img *image.RGBA, r image.Rectangle, fill color.NRGBA) int {
	best := 0
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			px := img.RGBAAt(x, y)
			d := absDiff(px.R, fill.R) + absDiff(px.G, fill.G) + absDiff(px.B, fill.B)
			best = max(best, d)
		}
	}
	return best
}

func sameRGB(a color.RGBA, b color.NRGBA) bool {
	return a.R == b.R && a.G == b.G && a.B == b.B
}

// TestDragBetweenListsGoldens drags the second row of the first list
// over the second list, and stores the page while the drag is in
// progress: the row's image follows the pointer, and the line marks the
// place after the last row of the second list. The release then moves
// the row there.
func TestDragBetweenListsGoldens(t *testing.T) {
	for _, sc := range schemes() {
		t.Run(sc.name, func(t *testing.T) {
			g := testGallery(t)
			g.dark = sc.name == "dark"
			d := &dragWindow{g: g, w: onBackground(sc.colors, g.pageDragDrop), now: time.Unix(0, 0)}
			d.frame()

			o := dragRowsOrigin(g, sc.colors)
			rowH := int(dragRowH)
			x1 := o.X + int(dragListW+dragListGap)
			// The drop is one row below the last row of the second list,
			// so the image covers no other row and no part of the line.
			start := image.Pt(o.X+60, o.Y+rowH+rowH/2)
			end := image.Pt(x1+60, o.Y+3*rowH+rowH/2)
			d.pointer(pointer.Move, start)
			d.pointer(pointer.Press, start)
			const steps = 4
			for i := 1; i <= steps; i++ {
				p := start.Add(end.Sub(start).Mul(i).Div(steps))
				d.pointer(pointer.Drag, p)
			}
			if !g.drag.st.Dragging() {
				t.Fatal("no drag is in progress after the pointer moved")
			}

			img := golden.Capture(t, dragPageSize, d.w)
			line := dragdrop.FromTokens(sc.colors).Line
			if px := img.RGBAAt(x1+200, o.Y+2*rowH); !sameRGB(px, line) {
				t.Errorf("line pixel = %v, want the line colour %v", px, line)
			}
			// The image's text is translucent: its strongest pixel stands
			// less far from the fill than the strongest pixel of the same
			// text in the row it came from.
			fill := sc.colors.ControlBackground
			row := strongest(img, image.Rect(o.X+1, o.Y+rowH, o.X+int(dragListW)-1, o.Y+2*rowH), fill)
			drag := strongest(img, image.Rect(x1+1, o.Y+3*rowH, x1+int(dragListW)-1, o.Y+4*rowH), fill)
			if row == 0 || drag == 0 || drag*10 >= row*9 {
				t.Errorf("text contrast in the image = %d, in the row = %d, want the image below nine tenths of the row", drag, row)
			}
			golden.Compare(t, "dragdrop-"+sc.name, img)

			d.pointer(pointer.Release, end)
			wantInbox := []string{"Write the release notes", "Book the train", "Review the pull request", "Order new toner", "Plan the team lunch"}
			wantToday := []string{"Call the dentist", "Water the plants", "Answer the support mail"}
			if !slices.Equal(g.drag.lists[0], wantInbox) || !slices.Equal(g.drag.lists[1], wantToday) {
				t.Errorf("lists after the drop = %q and %q, want %q and %q", g.drag.lists[0], g.drag.lists[1], wantInbox, wantToday)
			}
		})
	}
}

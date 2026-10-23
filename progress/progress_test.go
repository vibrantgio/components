package progress_test

import (
	"image"
	"image/color"
	"testing"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"

	"github.com/vibrantgio/components/golden"
	"github.com/vibrantgio/components/progress"
	"github.com/vibrantgio/theme/tokens"
)

var schemes = []struct {
	name string
	c    tokens.PlatformColors
}{
	{"light", tokens.PlatformLight},
	{"dark", tokens.PlatformDark},
}

// differs answers whether a pixel lies more than 8 levels off the fill on
// any channel, the reference's threshold for reading the indicator's extent.
func differs(px color.RGBA, fill color.NRGBA) bool {
	d := func(a, b uint8) bool { return int(a)-int(b) > 8 || int(b)-int(a) > 8 }
	return d(px.R, fill.R) || d(px.G, fill.G) || d(px.B, fill.B)
}

// painted is the bounding box of the pixels in r that differ from fill.
func painted(img *image.RGBA, r image.Rectangle, fill color.NRGBA) image.Rectangle {
	var box image.Rectangle
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			if differs(img.RGBAAt(x, y), fill) {
				box = box.Union(image.Rect(x, y, x+1, y+1))
			}
		}
	}
	return box
}

func onFill(fill color.NRGBA, at image.Point, w layout.Widget) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		paint.FillShape(gtx.Ops, fill, clip.Rect{Max: gtx.Constraints.Max}.Op())
		off := op.Offset(at).Push(gtx.Ops)
		w(gtx)
		off.Pop()
		return layout.Dimensions{Size: gtx.Constraints.Max}
	}
}

// TestTheIndicatorIsDrawnAtTheMeasuredSize holds the wheel to the
// reference's extent, 16 × 16 px at 1x, at every phase and in both schemes,
// and holds its darkest stroke to the stroke the phase names.
func TestTheIndicatorIsDrawnAtTheMeasuredSize(t *testing.T) {
	for _, sc := range schemes {
		for phase := range progress.Strokes {
			fill := sc.c.CardFill
			p := progress.Props{Platform: sc.c, Fill: fill, Phase: phase}
			img := golden.Capture(t, image.Pt(32, 32), onFill(fill, image.Pt(8, 8), func(gtx layout.Context) layout.Dimensions {
				return progress.Indicator(gtx, p)
			}))
			if got, want := painted(img, img.Bounds(), fill), image.Rect(8, 8, 24, 24); got != want {
				t.Errorf("%s phase %d: the indicator paints %v, want the measured 16 × 16 at %v", sc.name, phase, got, want)
			}
			// The top stroke is columns 15–16, rows 8–12; the trailing
			// one is rows 15–16, columns 19–23. Phase 0 makes the top
			// stroke darkest, phase 2 the trailing one.
			top, trailing := img.RGBAAt(15, 10), img.RGBAAt(21, 15)
			head := progress.StrokeColor(sc.c, fill, 0)
			switch phase {
			case 0:
				if !near(top, head) {
					t.Errorf("%s phase 0: the top stroke reads %v, want the darkest stroke %v", sc.name, top, head)
				}
			case 2:
				if !near(trailing, head) {
					t.Errorf("%s phase 2: the trailing stroke reads %v, want the darkest stroke %v", sc.name, trailing, head)
				}
			}
		}
	}
}

func near(px color.RGBA, c color.NRGBA) bool {
	d := func(a, b uint8) bool { return int(a)-int(b) <= 2 && int(b)-int(a) <= 2 }
	return d(px.R, c.R) && d(px.G, c.G) && d(px.B, c.B)
}

// TestTheStrokesSpanTheMeasuredGreys holds the darkest and the lightest
// stroke to the reference's readings over the System Settings row's fill.
func TestTheStrokesSpanTheMeasuredGreys(t *testing.T) {
	for _, tc := range []struct {
		name           string
		c              tokens.PlatformColors
		fill           color.NRGBA
		head, lightest uint8
	}{
		{"light", tokens.PlatformLight, color.NRGBA{0xf7, 0xf7, 0xf7, 0xff}, 66, 227},
		{"dark", tokens.PlatformDark, color.NRGBA{0x25, 0x25, 0x25, 0xff}, 158, 53},
	} {
		head := progress.StrokeColor(tc.c, tc.fill, 0).R
		tail := progress.StrokeColor(tc.c, tc.fill, progress.Strokes-1).R
		if d := int(head) - int(tc.head); d < -6 || d > 6 {
			t.Errorf("%s: the darkest stroke reads %d, want the measured %d within 6", tc.name, head, tc.head)
		}
		if d := int(tail) - int(tc.lightest); d < -6 || d > 6 {
			t.Errorf("%s: the lightest stroke reads %d, want the measured %d within 6", tc.name, tail, tc.lightest)
		}
	}
}

// specimen is what the stored images hold: the wheel alone at three phases,
// the row shape on the platform's card fill with the measured 10 px padding,
// and the region shape in a frame of its own.
func specimen(sc tokens.PlatformColors) layout.Widget {
	sh := tokens.DefaultTypography.DeterministicShaper()
	page := sc.WindowBackground
	return func(gtx layout.Context) layout.Dimensions {
		paint.FillShape(gtx.Ops, page, clip.Rect{Max: gtx.Constraints.Max}.Op())
		for i, phase := range []int{0, 3, 6} {
			off := op.Offset(image.Pt(12+i*28, 12)).Push(gtx.Ops)
			progress.Indicator(gtx, progress.Props{Platform: sc, Fill: page, Phase: phase})
			off.Pop()
		}
		row := image.Rect(12, 40, 312, 76)
		off := op.Offset(row.Min).Push(gtx.Ops)
		paint.FillShape(gtx.Ops, sc.CardFill, clip.Rect{Max: row.Size()}.Op())
		inner := op.Offset(image.Pt(10, 0)).Push(gtx.Ops)
		rgtx := gtx
		rgtx.Constraints = layout.Exact(image.Pt(row.Dx()-20, row.Dy()))
		progress.Row(rgtx, progress.Props{Platform: sc, Fill: sc.CardFill, Text: "Checking for updates…", Shaper: sh})
		inner.Pop()
		off.Pop()
		off = op.Offset(image.Pt(12, 88)).Push(gtx.Ops)
		fgtx := gtx
		fgtx.Constraints = layout.Exact(image.Pt(300, 96))
		progress.Region(fgtx, progress.Props{Platform: sc, Fill: page, Text: "Reading the memory slots…", Shaper: sh})
		off.Pop()
		return layout.Dimensions{Size: gtx.Constraints.Max}
	}
}

// TestProgressGolden stores the specimen per scheme.
func TestProgressGolden(t *testing.T) {
	for _, sc := range schemes {
		golden.Render(t, "progress-"+sc.name, image.Pt(324, 196), specimen(sc.c))
	}
}

// TestTheRegionStandsTheIndicatorAboveTheText holds the region shape: the
// indicator is centred across the frame, the text's glyphs begin under it
// past the gap, and the group's middle is the frame's middle.
func TestTheRegionStandsTheIndicatorAboveTheText(t *testing.T) {
	sh := tokens.DefaultTypography.DeterministicShaper()
	for _, sc := range schemes {
		fill := sc.c.WindowBackground
		frame := image.Pt(300, 120)
		img := golden.Capture(t, frame, onFill(fill, image.Point{}, func(gtx layout.Context) layout.Dimensions {
			return progress.Region(gtx, progress.Props{Platform: sc.c, Fill: fill, Text: "Reading the memory slots…", Shaper: sh})
		}))
		group := painted(img, img.Bounds(), fill)
		ind := painted(img, image.Rect(0, group.Min.Y, frame.X, group.Min.Y+16), fill)
		if ind.Dx() != 16 || ind.Min.X != (frame.X-16)/2 {
			t.Errorf("%s: the indicator paints %v, want 16 px centred across %d", sc.name, ind, frame.X)
		}
		txt := painted(img, image.Rect(0, ind.Max.Y, frame.X, frame.Y), fill)
		gap := txt.Min.Y - ind.Max.Y
		if g := int(progress.Gap); gap < g || gap > g+6 {
			t.Errorf("%s: the text's glyphs begin %d px under the indicator, want the gap %d and the line's own room above its glyphs, at most %d", sc.name, gap, g, g+6)
		}
		if d := (group.Min.Y + group.Max.Y) - frame.Y; d < -6 || d > 6 {
			t.Errorf("%s: the group %v centres at %.1f down, want the frame's %.1f", sc.name, group, float64(group.Min.Y+group.Max.Y)/2, float64(frame.Y)/2)
		}
		t.Logf("%s: indicator %v, text %v, gap %d", sc.name, ind, txt, gap)
	}
}

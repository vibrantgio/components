package button_test

import (
	"image"
	"image/color"
	"testing"

	"gioui.org/f32"
	gioinput "gioui.org/io/input"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"

	"github.com/reactivego/rx"
	"github.com/vibrantgio/components/button"
	golden "github.com/vibrantgio/components/golden"
	"github.com/vibrantgio/theme/theme"
	"github.com/vibrantgio/theme/tokens"
)

// bandFill is the fixed band's fill in these tests: a value the find
// button's face and shadow never draw, so the band's top row reads off it.
var bandFill = color.NRGBA{R: 0x20, G: 0x80, B: 0x20, A: 0xff}

// paintAll fills the whole of its constraints with col.
func paintAll(col color.NRGBA) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		paint.FillShape(gtx.Ops, col, clip.Rect{Max: gtx.Constraints.Max}.Op())
		return layout.Dimensions{Size: gtx.Constraints.Max}
	}
}

// fixedBand is a band of height h across the width it is given.
func fixedBand(h int) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		size := image.Pt(gtx.Constraints.Max.X, h)
		paint.FillShape(gtx.Ops, bandFill, clip.Rect{Max: size}.Op())
		return layout.Dimensions{Size: size}
	}
}

// stand is a stand-in for the find button: an opaque block of the measured
// 84 by 24 in a value the content and the band never draw, so its box reads
// off the frame exactly.
var standFill = color.NRGBA{R: 0xc0, G: 0x30, B: 0x30, A: 0xff}

func stand(gtx layout.Context) layout.Dimensions {
	size := image.Pt(84, 24)
	paint.FillShape(gtx.Ops, standFill, clip.Rect{Max: size}.Op())
	return layout.Dimensions{Size: size}
}

// placeFindButton lays the stand-in over a content column of size, with the
// band when band is non-nil, and answers the frame and the stand-in's box.
func placeFindButton(t *testing.T, size image.Point, band layout.Widget) (*image.RGBA, image.Rectangle) {
	t.Helper()
	c := tokens.PlatformLight
	img := golden.Capture(t, size, button.FindButtonOver(paintAll(c.WindowBackground), band, stand))
	box := image.Rectangle{Min: size, Max: image.Pt(-1, -1)}
	for y := 0; y < size.Y; y++ {
		for x := 0; x < size.X; x++ {
			if pixel(img, x, y) == standFill {
				box.Min.X, box.Min.Y = min(box.Min.X, x), min(box.Min.Y, y)
				box.Max.X, box.Max.Y = max(box.Max.X, x+1), max(box.Max.Y, y+1)
			}
		}
	}
	if box.Empty() {
		t.Fatal("the find button is nowhere in the frame")
	}
	return img, box
}

// trackFill is the band's track in these tests, another value the find
// button never draws.
var trackFill = color.NRGBA{R: 0x20, G: 0x20, B: 0x80, A: 0xff}

// trackDp is the height of the band's track in these tests.
//
// MEASURED, voicememos-collection-search-light.png: the waveform's track runs
// y 1312-1355.
const trackDp = 44

// trackedBand is a band that keeps the find button's room above its track:
// fill over the room and the track under it, across the width it is given.
func trackedBand(fill color.NRGBA) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		room := gtx.Dp(button.FindButtonRoomDp)
		size := image.Pt(gtx.Constraints.Max.X, room+gtx.Dp(trackDp))
		paint.FillShape(gtx.Ops, fill, clip.Rect{Max: image.Pt(size.X, room)}.Op())
		paint.FillShape(gtx.Ops, trackFill, clip.Rect{Min: image.Pt(0, room), Max: size}.Op())
		return layout.Dimensions{Size: size}
	}
}

// TestTheFindButtonStandsInTheBand places the find button over a content
// column with a fixed band at its foot. The scrolling region ends at the
// band's top. The find button stands in the band, the measured 10 below the
// region's foot and the measured 35 above the track, and it is centred on the
// column with the odd pixel to the trailing side.
//
// MEASURED, voicememos-collection-search-light.png: the shadow above the find
// button stops at y=1243, the find button's box runs y 1253-1276, and the
// waveform's track starts at y=1312. Its box runs x 1013-1097 in a column
// that runs x 377-1732, 636 columns leading it and 635 trailing it.
func TestTheFindButtonStandsInTheBand(t *testing.T) {
	for _, width := range []int{400, 401} {
		size := image.Pt(width, 300)
		img, box := placeFindButton(t, size, trackedBand(bandFill))
		foot := size.Y - int(button.FindButtonRoomDp) - trackDp
		trackTop := foot + int(button.FindButtonRoomDp)
		if got := pixel(img, 5, foot); got != bandFill {
			t.Fatalf("width %d: the band's top row reads %v, want the band", width, got)
		}
		if got := pixel(img, 5, foot-1); got == bandFill {
			t.Fatalf("width %d: the band starts above its own height", width)
		}
		if got := pixel(img, 5, trackTop); got != trackFill {
			t.Fatalf("width %d: the track's top row reads %v, want the track", width, got)
		}
		if gap := box.Min.Y - foot; gap != 10 {
			t.Errorf("width %d: %d clear rows between the region's foot and the find button, want the measured 10", width, gap)
		}
		if gap := trackTop - box.Max.Y; gap != 35 {
			t.Errorf("width %d: %d clear rows between the find button and the track, want the measured 35", width, gap)
		}
		lead, trail := box.Min.X, width-box.Max.X
		if lead-trail != 0 && lead-trail != 1 {
			t.Errorf("width %d: %d columns lead the find button and %d trail it, want it centred with the odd column leading it", width, lead, trail)
		}
	}
}

// TestTheFindButtonsShadowStaysInTheBand draws the find button in a band
// whose fill is the content's. The row above the band's top is the content's
// own value at every column, and in the light scheme the shadow runs in the
// band beside the find button.
//
// MEASURED, voicememos-collection-search-light.png: y=1242 is #ffffff at
// every column, the row above the region's foot. Across y=1265 the shadow
// reads 7/255 at 1 px out beside the find button.
func TestTheFindButtonsShadowStaysInTheBand(t *testing.T) {
	for _, sc := range []struct {
		name string
		c    tokens.PlatformColors
	}{{"light", tokens.PlatformLight}, {"dark", tokens.PlatformDark}} {
		c := sc.c
		find := button.RenderFindButton(defaultShaper(t), "", c, tokens.DefaultTypography.SystemFont, button.RenderState{})
		size := image.Pt(400, 300)
		img := golden.Capture(t, size, button.FindButtonOver(paintAll(c.WindowBackground), trackedBand(c.WindowBackground), find))
		foot := size.Y - int(button.FindButtonRoomDp) - trackDp
		for x := 0; x < size.X; x++ {
			if got := pixel(img, x, foot-1); got != c.WindowBackground {
				t.Fatalf("%s: (%d,%d), the region's last row, reads %v, want the content's %v", sc.name, x, foot-1, got, c.WindowBackground)
			}
		}
		if sc.name != "light" {
			// The dark shadow is 1/255 deep below the find button and
			// does not show beside it.
			continue
		}
		middle := foot + 10 + 12
		lead := -1
		for x := 0; x < size.X; x++ {
			if pixel(img, x, middle) != c.WindowBackground {
				lead = x
				break
			}
		}
		// The find button is at most 84 wide and centred, so it starts at
		// column 158 or later. A shadow beside it starts columns before.
		if lead < 0 || lead > 150 {
			t.Errorf("%s: the first column off the content's value on the find button's middle row is %d, want the shadow from before column 150", sc.name, lead)
		}
	}
}

// TestTheFindButtonStandsAboveTheColumnsFoot places the find button over a
// content column that keeps no band: its bottom stands the measured 35 above
// the column's foot.
func TestTheFindButtonStandsAboveTheColumnsFoot(t *testing.T) {
	size := image.Pt(400, 300)
	_, box := placeFindButton(t, size, nil)
	if gap := size.Y - box.Max.Y; gap != 35 {
		t.Errorf("%d clear rows between the find button and the column's foot, want the measured 35", gap)
	}
}

// TestNoFindButtonLeavesTheContentWhole lays the column out with no find
// button: every pixel is the content's or the band's.
func TestNoFindButtonLeavesTheContentWhole(t *testing.T) {
	c := tokens.PlatformLight
	size := image.Pt(200, 200)
	img := golden.Capture(t, size, button.FindButtonOver(paintAll(c.WindowBackground), fixedBand(40), nil))
	for y := 0; y < size.Y; y++ {
		want := c.WindowBackground
		if y >= size.Y-40 {
			want = bandFill
		}
		for x := 0; x < size.X; x++ {
			if got := pixel(img, x, y); got != want {
				t.Fatalf("(%d,%d) reads %v with no find button, want %v", x, y, got, want)
			}
		}
	}
}

// TestTheFindButtonTakesThePointerInTheBand presses the live find button
// where it stands in the band of a content column that takes presses itself:
// the find button acts and the content does not.
func TestTheFindButtonTakesThePointerInTheBand(t *testing.T) {
	var pressed int
	find := materialize(t, button.FindButton(rx.Of(theme.Default()), button.FindButtonProps{
		OnClick: func(layout.Context) { pressed++ },
	}))
	var content widget.Clickable
	column := func(gtx layout.Context) layout.Dimensions {
		return content.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.Dimensions{Size: gtx.Constraints.Max}
		})
	}
	w := button.FindButtonOver(column, trackedBand(bandFill), find)
	r := new(gioinput.Router)
	ops := new(op.Ops)
	drive := func() {
		ops.Reset()
		gtx := layout.Context{
			Metric:      unit.Metric{PxPerDp: 1, PxPerSp: 1},
			Constraints: layout.Exact(image.Pt(400, 300)),
			Ops:         ops,
			Source:      r.Source(),
		}
		w(gtx)
		r.Frame(ops)
	}
	drive()
	// The find button's centre: centred on the column, its 24 rows starting
	// 10 below the band's top.
	foot := 300 - int(button.FindButtonRoomDp) - trackDp
	pos := f32.Pt(200, float32(foot+10+12))
	r.Queue(
		pointer.Event{Kind: pointer.Press, Position: pos, Buttons: pointer.ButtonPrimary, Source: pointer.Mouse},
		pointer.Event{Kind: pointer.Release, Position: pos, Buttons: pointer.ButtonPrimary, Source: pointer.Mouse},
	)
	drive()
	if content.Clicked(layout.Context{Source: r.Source()}) {
		t.Error("the content took the press that the find button stands over")
	}
	drive()
	if pressed != 1 {
		t.Errorf("a press on the find button over the content called OnClick %d times, want 1", pressed)
	}
}

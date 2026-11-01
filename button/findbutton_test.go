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

	"github.com/reactivego/rx"
	"github.com/vibrantgio/components/button"
	golden "github.com/vibrantgio/components/golden"
	"github.com/vibrantgio/components/icons"
	"github.com/vibrantgio/theme/theme"
	"github.com/vibrantgio/theme/tokens"
)

// findButtonAt is where the find button's box starts in each capture below:
// far enough in that its shadow lands inside the frame.
var findButtonAt = image.Pt(40, 40)

// captureFindButton draws the find button at rest over the window's own
// fill, the content it stands over, and answers the frame and the box.
func captureFindButton(t *testing.T, c tokens.PlatformColors, s button.RenderState) (*image.RGBA, image.Rectangle) {
	t.Helper()
	w := button.RenderFindButton(defaultShaper(t), "", c, tokens.DefaultTypography.SystemFont, s)
	var dims layout.Dimensions
	img := golden.Capture(t, image.Pt(170, 110), func(gtx layout.Context) layout.Dimensions {
		paint.FillShape(gtx.Ops, c.WindowBackground, clip.Rect{Max: gtx.Constraints.Max}.Op())
		defer op.Offset(findButtonAt).Push(gtx.Ops).Pop()
		gtx.Constraints.Min = image.Point{}
		dims = w(gtx)
		return dims
	})
	return img, image.Rectangle{Min: findButtonAt, Max: findButtonAt.Add(dims.Size)}
}

// covered answers the first and last columns and rows inside r where a
// pixel differs from fill by more than 2 in any channel: the extent of a
// symbol or a title drawn on that fill, as the search field's tests read it.
func covered(img *image.RGBA, r image.Rectangle, fill color.NRGBA) (x0, x1, y0, y1 int) {
	x0, y0, x1, y1 = r.Max.X, r.Max.Y, -1, -1
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			p := pixel(img, x, y)
			if absDiff(p.R, fill.R) > 2 || absDiff(p.G, fill.G) > 2 || absDiff(p.B, fill.B) > 2 {
				x0, x1, y0, y1 = min(x0, x), max(x1, x), min(y0, y), max(y1, y)
			}
		}
	}
	return
}

func absDiff(a, b uint8) int {
	if a > b {
		return int(a - b)
	}
	return int(b - a)
}

// TestTheFindButtonIsTheMeasuredButton reads the find button off the drawn
// pixels in the light scheme and holds it to the reference.
//
// MEASURED, voicememos-collection-search-light.png, a 1x capture: the button
// is 84 × 24. Its fill reads #fdfdfd and its edge #ffffff. The magnifier
// covers 13 × 13 from 13 columns in and 6 rows down. The title's first
// covered column stands 4 clear columns past the magnifier's last, and 13
// clear columns trail the title's last. The shadow under the button is the
// toolbar control's to within 1/255 at every row.
func TestTheFindButtonIsTheMeasuredButton(t *testing.T) {
	c := tokens.PlatformLight
	img, box := captureFindButton(t, c, button.RenderState{})
	if box.Dy() != 24 || box.Dx() < 82 || box.Dx() > 84 {
		t.Errorf("the find button is %v, want 24 tall and within 2 of the measured 84 wide", box.Size())
	}
	if got, want := pixel(img, box.Min.X+20, box.Min.Y+3), (color.NRGBA{R: 0xfd, G: 0xfd, B: 0xfd, A: 0xff}); got != want {
		t.Errorf("the fill = %v, want the measured %v", got, want)
	}
	if got, want := pixel(img, box.Min.X+40, box.Min.Y), (color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}); got != want {
		t.Errorf("the top edge = %v, want the measured %v", got, want)
	}

	mark := image.Rect(box.Min.X+8, box.Min.Y+4, box.Min.X+28, box.Max.Y-4)
	x0, x1, y0, y1 := covered(img, mark, c.FindButtonFill)
	if x0-box.Min.X != 13 || x1-x0+1 != 13 {
		t.Errorf("the magnifier covers columns %d-%d, want 13 wide from column 13", x0-box.Min.X, x1-box.Min.X)
	}
	if y0-box.Min.Y != 6 || y1-y0+1 != 13 {
		t.Errorf("the magnifier covers rows %d-%d, want 13 tall from row 6", y0-box.Min.Y, y1-box.Min.Y)
	}

	title := image.Rect(x1+1, box.Min.Y+4, box.Max.X-8, box.Max.Y-4)
	t0, t1, _, _ := covered(img, title, c.FindButtonFill)
	if gap := t0 - x1 - 1; gap < 3 || gap > 4 {
		t.Errorf("%d clear columns between the magnifier and the title, want the measured 4, or 3 where the library's S reaches past its box", gap)
	}
	if trail := box.Max.X - 1 - t1; trail != 13 {
		t.Errorf("%d clear columns trail the title, want the measured 13", trail)
	}

	// The shadow is the toolbar control's: the same ramp under the find
	// button as under a bordered toolbar control on the same content.
	chrome := button.RenderBordered(icons.Mark(icons.MagnifyingGlass), c, tokens.Radius, tokens.Comfortable,
		button.RenderState{Variant: button.Chrome})
	var cdims layout.Dimensions
	cimg := golden.Capture(t, image.Pt(170, 130), func(gtx layout.Context) layout.Dimensions {
		paint.FillShape(gtx.Ops, c.WindowBackground, clip.Rect{Max: gtx.Constraints.Max}.Op())
		defer op.Offset(findButtonAt).Push(gtx.Ops).Pop()
		gtx.Constraints.Min = image.Point{}
		cdims = chrome(gtx)
		return cdims
	})
	cmid, mid := findButtonAt.X+cdims.Size.X/2, box.Min.X+box.Dx()/2
	for d := range 30 {
		got := pixel(img, mid, box.Max.Y+d).G
		want := pixel(cimg, cmid, findButtonAt.Y+cdims.Size.Y+d).G
		if absDiff(got, want) > 1 {
			t.Errorf("%d px under the find button the content reads %d, and %d under the toolbar control", d+1, got, want)
		}
	}
}

// TestTheFindButtonTakesTheToolbarControlsDarkFace holds the dark scheme to
// the toolbar control's own values, because the reference holds no dark
// reading of the find button.
func TestTheFindButtonTakesTheToolbarControlsDarkFace(t *testing.T) {
	c := tokens.PlatformDark
	if c.FindButtonFill != c.ToolbarControlFill || c.FindButtonRim != c.ToolbarControlRim || c.FindButtonForeground != c.ToolbarLabel {
		t.Fatal("the dark find button does not take the toolbar control's fill, rim and foreground")
	}
	img, box := captureFindButton(t, c, button.RenderState{})
	if box.Dy() != 24 || box.Dx() < 82 || box.Dx() > 84 {
		t.Fatalf("the find button is %v, want the light scheme's size", box.Size())
	}
	if got := pixel(img, box.Min.X+20, box.Min.Y+3); got != c.FindButtonFill {
		t.Errorf("the fill = %v, want %v", got, c.FindButtonFill)
	}
	if got := pixel(img, box.Min.X+40, box.Min.Y); got != c.FindButtonRim {
		t.Errorf("the top edge = %v, want %v", got, c.FindButtonRim)
	}
}

// TestFindButtonGolden stores the find button over the content in both
// schemes.
func TestFindButtonGolden(t *testing.T) {
	for _, sc := range []struct {
		name string
		c    tokens.PlatformColors
	}{{"light", tokens.PlatformLight}, {"dark", tokens.PlatformDark}} {
		t.Run(sc.name, func(t *testing.T) {
			img, _ := captureFindButton(t, sc.c, button.RenderState{})
			golden.Compare(t, "find-button-"+sc.name, img)
		})
	}
}

// TestTheFindButtonActsOnPress presses the live find button: each press
// calls OnClick once and posts nothing else.
func TestTheFindButtonActsOnPress(t *testing.T) {
	var pressed int
	w := materialize(t, button.FindButton(rx.Of(theme.Default()), button.FindButtonProps{
		OnClick: func(layout.Context) { pressed++ },
	}))
	r := new(gioinput.Router)
	ops := new(op.Ops)
	var dims layout.Dimensions
	drive := func() {
		ops.Reset()
		gtx := layout.Context{
			Metric:      unit.Metric{PxPerDp: 1, PxPerSp: 1},
			Constraints: layout.Constraints{Max: image.Pt(200, 100)},
			Ops:         ops,
			Source:      r.Source(),
		}
		dims = w(gtx)
		r.Frame(ops)
	}
	drive()
	if dims.Size.Y != 24 {
		t.Fatalf("the live find button is %v, want 24 tall", dims.Size)
	}
	pos := f32.Pt(40, 12)
	r.Queue(
		pointer.Event{Kind: pointer.Press, Position: pos, Buttons: pointer.ButtonPrimary, Source: pointer.Mouse},
		pointer.Event{Kind: pointer.Release, Position: pos, Buttons: pointer.ButtonPrimary, Source: pointer.Mouse},
	)
	drive()
	drive()
	if pressed != 1 {
		t.Errorf("one press called OnClick %d times, want 1", pressed)
	}
}

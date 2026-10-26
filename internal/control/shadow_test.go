package control_test

import (
	"image"
	"image/color"
	"testing"

	"gioui.org/layout"
	"gioui.org/op/paint"

	"github.com/vibrantgio/components/golden"
	"github.com/vibrantgio/components/internal/control"
	"github.com/vibrantgio/theme/tokens"
)

// drawFinderPopUpShadow paints the light toolbar control's shadow at the
// geometry of the view pop-up in finder-window-light.png: x 694-742, y 8-43,
// rounded to half its height, on the band's flat #ffffff.
func drawFinderPopUpShadow(gtx layout.Context) layout.Dimensions {
	paint.Fill(gtx.Ops, color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff})
	box := image.Rect(694, 8, 743, 44)
	control.DrawToolbarShadow(gtx, box, box.Dy()/2, tokens.PlatformLight.ToolbarControlShadow)
	return layout.Dimensions{Size: gtx.Constraints.Max}
}

// The corners of the shadow are rounded on the corner circle's centre. The
// reference reads the pop-up's diagonals in finder-window-light.png and works
// out three pixels: one step out from the lower leading corner reads 7, eight
// steps out reads 4, and one step out from the upper leading corner reads 3,
// in 255ths below the band's #ffffff.
func TestTheToolbarShadowCornerReadsThePlatformsDiagonal(t *testing.T) {
	img := golden.Capture(t, image.Pt(780, 100), drawFinderPopUpShadow)
	for _, c := range []struct {
		at   image.Point
		want int
	}{
		{image.Pt(693, 44), 7},
		{image.Pt(686, 51), 4},
		{image.Pt(693, 7), 3},
	} {
		got := 255 - int(img.RGBAAt(c.at.X, c.at.Y).R)
		if d := got - c.want; d < -1 || d > 1 {
			t.Errorf("pixel %v is %d 255ths below the band, the platform reads %d", c.at, got, c.want)
		}
	}
}

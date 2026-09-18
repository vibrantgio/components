package button_test

import (
	"image"
	"image/color"
	"testing"

	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"

	"github.com/vibrantgio/components/button"
	golden "github.com/vibrantgio/components/golden"
	vgcolor "github.com/vibrantgio/theme/color"
	"github.com/vibrantgio/theme/tokens"
)

// segmentPair lays a two-segment control out on the chrome material and
// answers the image and the box it reported.
func segmentPair(t *testing.T, p tokens.PlatformColors, segs []button.ChromeSegment, size image.Point) (*image.RGBA, image.Point) {
	t.Helper()
	var got image.Point
	img := golden.Capture(t, size, onChrome(p, func(gtx layout.Context) layout.Dimensions {
		gtx.Metric = unit.Metric{PxPerDp: 1, PxPerSp: 1}
		d := button.ChromeSegments(gtx, p, tokens.Comfortable, segs)
		got = d.Size
		return d
	}))
	return img, got
}

// TestTheSegmentedPairIsTheMeasuredControl reads Finder's back/forward pair
// back off the drawing.
//
// MEASURED at 1x, finder-window-untinted-dark.png: the control spans x
// 326–398 and y 8–43 — 73 by 36 — with its rim at both ends, and the seam
// is the single column at x=362 running y 16–35. So two segments of the
// chrome variant's own width with one hairline between them, and the seam
// leaves eight rows clear at the top and eight at the foot.
// finder-window-light.png draws the same pair at the same columns, its
// seam #f2f2f2 over the control's #ffffff against the dark capture's
// #3a3a3a over its #262626.
func TestTheSegmentedPairIsTheMeasuredControl(t *testing.T) {
	size := image.Pt(160, 60)
	for _, tc := range []struct {
		name string
		p    tokens.PlatformColors
	}{
		{"light", tokens.PlatformLight},
		{"dark", tokens.PlatformDark},
	} {
		t.Run(tc.name, func(t *testing.T) {
			segs := []button.ChromeSegment{{}, {}}
			img, box := segmentPair(t, tc.p, segs, size)

			h := int(tokens.Comfortable.ToolbarControlHeight)
			seg := 38 // the chrome variant's own width: a 24 dp mark with 7 clear a side
			if want := (image.Pt(2*seg+1, h)); box != want {
				t.Fatalf("the pair reports %v, want two segments and a hairline at the toolbar control's height, %v", box, want)
			}

			fill := tc.p.ToolbarControlFill
			// The control's OWN seam, measured: the separator over the fill
			// falls twelve of 255 short of it in the light appearance.
			rule := tc.p.ToolbarControlSeam
			if flat := vgcolor.Flatten(tc.p.Separator, fill); flat == rule {
				t.Errorf("the separator over the fill %v lands the measured seam; the value is carried because it does not", flat)
			}
			is := func(x, y int, want color.NRGBA) bool {
				c := img.RGBAAt(x, y)
				return c.R == want.R && c.G == want.G && c.B == want.B
			}
			// The seam: one column at the segments' boundary, running the
			// control's middle rows with the measured clearance at each end.
			clear := int(button.SegmentSeamClearDp)
			for y := 0; y < h; y++ {
				inside := y >= clear && y < h-clear
				if got := is(seg, y, rule); got != inside {
					t.Errorf("the seam at y=%d draws %v; it runs the control's rows %d to %d and no further",
						y, img.RGBAAt(seg, y), clear, h-clear-1)
					break
				}
			}
			// Either side of it is the control's own fill: the pair is one
			// box divided, not two boxes side by side.
			for _, x := range []int{seg - 1, seg + 1} {
				if !is(x, h/2, fill) {
					t.Errorf("the column beside the seam at x=%d draws %v, want the control's own fill %v",
						x, img.RGBAAt(x, h/2), fill)
				}
			}
			// And the fill runs unbroken across the boundary's own rows
			// above and below the seam, which is what says one capsule.
			if !is(seg, clear-1, fill) || !is(seg, h-clear, fill) {
				t.Error("the seam runs to the control's top or foot; the platform leaves both clear")
			}
		})
	}
}

// TestASwitchedOffSegmentFadesItsSymbolAlone requires the end of the stack to
// read as it does on the platform: the symbol drops to the platform's
// disabled control text and the control's own box does not move.
func TestASwitchedOffSegmentFadesItsSymbolAlone(t *testing.T) {
	p := tokens.PlatformLight
	size := image.Pt(160, 60)
	// A plain filled square standing in for a symbol: what is read back is
	// the colour the segment hands its drawing, not the drawing.
	mark := func(gtx layout.Context, sizePx int, col color.NRGBA) {
		paint.FillShape(gtx.Ops, col, clip.Rect{Max: image.Pt(sizePx, sizePx)}.Op())
	}
	live := []button.ChromeSegment{{Icon: mark}, {Icon: mark}}
	dead := []button.ChromeSegment{{Icon: mark}, {Icon: mark, State: button.RenderState{Disabled: true}}}

	liveImg, liveBox := segmentPair(t, p, live, size)
	deadImg, deadBox := segmentPair(t, p, dead, size)
	if liveBox != deadBox {
		t.Errorf("the pair measures %v with both halves live and %v with one switched off; the stack's end moves nothing", liveBox, deadBox)
	}
	h := int(tokens.Comfortable.ToolbarControlHeight)
	at := image.Pt(38+1+19, h/2)
	fill := p.ToolbarControlFill
	want := vgcolor.Flatten(p.DisabledControlText, fill)
	if got := deadImg.RGBAAt(at.X, at.Y); got.R != want.R || got.G != want.G || got.B != want.B {
		t.Errorf("the switched-off segment's symbol reads %v, want the platform's disabled control text over the fill, %v", got, want)
	}
	if liveImg.RGBAAt(at.X, at.Y) == deadImg.RGBAAt(at.X, at.Y) {
		t.Error("a switched-off segment draws its symbol exactly as a live one does")
	}
}

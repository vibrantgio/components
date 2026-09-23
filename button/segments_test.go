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
		// The band hands the control room, not a width: a segmented control
		// asked for a width divides it, and this pair is read at its own.
		gtx.Constraints.Min = image.Point{}
		d := button.ChromeSegments(gtx, defaultShaper(t), p, tokens.DefaultTypography.LabelLarge, tokens.Comfortable, segs)
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

// segmentRow lays a segmented control out at a stated width — the shape a
// form's own column asks for — and answers the image and the box it reported.
func segmentRow(t *testing.T, p tokens.PlatformColors, segs []button.ChromeSegment, size image.Point, width int) (*image.RGBA, image.Point) {
	t.Helper()
	var got image.Point
	img := golden.Capture(t, size, onChrome(p, func(gtx layout.Context) layout.Dimensions {
		gtx.Metric = unit.Metric{PxPerDp: 1, PxPerSp: 1}
		gtx.Constraints.Min = image.Pt(width, 0)
		d := button.ChromeSegments(gtx, defaultShaper(t), p, tokens.DefaultTypography.LabelLarge, tokens.Comfortable, segs)
		got = d.Size
		return d
	}))
	return img, got
}

// TestAWordedControlDividesTheWidthItIsAskedFor pins the one measured
// property a worded segmented control has: every segment of one control is
// the same width. MEASURED, finder-window-untinted-dark.png: the four-segment
// view control spans x 796–943, 148 px over four segments, 37 each.
//
// The word's own inset is NOT measured — not one control in the five stored
// toolbar bands carries a word — so what is read back here is the division
// and the seams, not a padding.
func TestAWordedControlDividesTheWidthItIsAskedFor(t *testing.T) {
	p := tokens.PlatformLight
	segs := []button.ChromeSegment{
		{Label: "OpenAI"}, {Label: "xAI"}, {Label: "OpenRouter"}, {Label: "Groq"},
	}
	const width = 360
	img, box := segmentRow(t, p, segs, image.Pt(420, 60), width)
	if box.X != width {
		t.Fatalf("the control measures %d wide, want the %d its column asked for", box.X, width)
	}
	h := int(tokens.Comfortable.ToolbarControlHeight)
	if box.Y != h {
		t.Errorf("the control measures %d tall, want the toolbar control's own %d", box.Y, h)
	}
	// The three seams stand at the quarter columns, one pixel each, and the
	// four segments they part are equal to the pixel.
	rule := p.ToolbarControlSeam
	var seams []int
	for x := 1; x < width-1; x++ {
		c := img.RGBAAt(x, h/2)
		if c.R == rule.R && c.G == rule.G && c.B == rule.B {
			seams = append(seams, x)
		}
	}
	if len(seams) != 3 {
		t.Fatalf("the control draws %d seam columns at its middle row, want three", len(seams))
	}
	widths := []int{seams[0], seams[1] - seams[0] - 1, seams[2] - seams[1] - 1, width - seams[2] - 1}
	for i, w := range widths {
		if w < widths[0]-1 || w > widths[0]+1 {
			t.Errorf("segment %d measures %d wide against the first's %d; one control's segments are equal", i, w, widths[0])
		}
	}
}

// TestTheChosenSegmentWearsTheMeasuredPatch reads the on-state off the
// drawing. MEASURED, finder-window-untinted-dark.png: the chosen segment of
// the four-segment view control carries a 32 by 26 patch inside its 37 by 36
// box, five rows clear above and below, at
// tokens.PlatformColors.ToolbarCheckedOverlay over the control's own fill.
func TestTheChosenSegmentWearsTheMeasuredPatch(t *testing.T) {
	for _, tc := range []struct {
		name string
		p    tokens.PlatformColors
	}{
		{"light", tokens.PlatformLight},
		{"dark", tokens.PlatformDark},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := tc.p
			segs := []button.ChromeSegment{
				{Label: "One"},
				{Label: "Two", State: button.RenderState{Checked: true}},
			}
			const width = 200
			img, _ := segmentRow(t, p, segs, image.Pt(260, 60), width)
			h := int(tokens.Comfortable.ToolbarControlHeight)
			fill := p.ToolbarControlFill
			want := vgcolor.Flatten(p.ToolbarCheckedOverlay, fill)
			is := func(x, y int, c color.NRGBA) bool {
				g := img.RGBAAt(x, y)
				return g.R == c.R && g.G == c.G && g.B == c.B
			}
			// Read clear of the word each segment centres: the chosen
			// half carries the patch, the unchosen half the bare fill.
			onPatch, onFill := width/2+8, 10
			if !is(onPatch, h/2, want) {
				t.Errorf("the chosen segment reads %v inside its patch, want the measured %v", img.RGBAAt(onPatch, h/2), want)
			}
			if !is(onFill, h/2, fill) {
				t.Errorf("the unchosen segment reads %v, want the control's own fill %v", img.RGBAAt(onFill, h/2), fill)
			}
			// Five rows clear of the control's box above and below: the
			// patch is inside the segment, not the segment.
			if is(onPatch, 1, want) || is(onPatch, h-2, want) {
				t.Error("the chosen segment's patch runs to the control's own top or foot; the platform leaves five rows clear")
			}
		})
	}
}

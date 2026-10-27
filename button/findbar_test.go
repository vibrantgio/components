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
	"github.com/vibrantgio/theme/theme"
	"github.com/vibrantgio/theme/tokens"
)

// findBarSchemes holds the reference rows of "What the platform's find bar
// measures": the bar's fill and the find bar button's fill, per scheme.
var findBarSchemes = []struct {
	name      string
	colors    tokens.PlatformColors
	bar, fill color.NRGBA
}{
	{"light", tokens.PlatformLight, color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}, color.NRGBA{R: 0xec, G: 0xec, B: 0xec, A: 0xff}},
	{"dark", tokens.PlatformDark, color.NRGBA{R: 0x23, G: 0x29, B: 0x2e, A: 0xff}, color.NRGBA{R: 0x33, G: 0x39, B: 0x3e, A: 0xff}},
}

// onBar stands w on the bar's fill, 10 px in from the frame's edges.
func onBar(bar color.NRGBA, w layout.Widget) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		paint.FillShape(gtx.Ops, bar, clip.Rect{Max: gtx.Constraints.Max}.Op())
		defer op.Offset(image.Pt(10, 10)).Push(gtx.Ops).Pop()
		gtx.Constraints.Min = image.Point{}
		return w(gtx)
	}
}

func pixel(img *image.RGBA, x, y int) color.NRGBA {
	r, g, b, a := img.At(x, y).RGBA()
	return color.NRGBA{R: uint8(r >> 8), G: uint8(g >> 8), B: uint8(b >> 8), A: uint8(a >> 8)}
}

// TestFindBarButtonIsTheMeasuredButton reads Done off the drawn pixels in
// both schemes: 20 px tall, the measured fill from its first row to its last
// with no edge row, nothing cast on the bar around it, and a corner of about
// 5 px.
//
// MEASURED, textedit-find-empty-{light,dark}.png: Done is 48 by 20, fill
// #ececec light and #33393e dark, no edge. The top row's coverage at the
// leading end leaves 3.1 px missing, against 3.05 for a 5 px circle and 3.85
// for 6 px.
func TestFindBarButtonIsTheMeasuredButton(t *testing.T) {
	shaper := defaultShaper(t)
	for _, sc := range findBarSchemes {
		t.Run(sc.name, func(t *testing.T) {
			var dims layout.Dimensions
			w := button.Render(shaper, "Done", sc.colors, tokens.Spacing, tokens.Radius, tokens.DefaultTypography.SmallSystemFont, tokens.Comfortable,
				button.RenderState{Variant: button.FindBar, Surface: sc.bar})
			img := golden.Capture(t, image.Pt(100, 40), onBar(sc.bar, func(gtx layout.Context) layout.Dimensions {
				dims = w(gtx)
				return dims
			}))
			if dims.Size.Y != 20 {
				t.Errorf("height = %d, want the measured 20", dims.Size.Y)
			}
			x := 10 + 6 // past the corner, short of the title
			for y := 10; y < 10+dims.Size.Y; y++ {
				if got := pixel(img, x, y); got != sc.fill {
					t.Errorf("row %d at x=%d = %v, want the measured fill %v: no edge row", y-10, x-10, got, sc.fill)
				}
			}
			for _, y := range []int{9, 10 + dims.Size.Y} {
				if got := pixel(img, x, y); got != sc.bar {
					t.Errorf("the bar at y=%d = %v, want %v: the button casts nothing", y-10, got, sc.bar)
				}
			}
			missing := 0.0
			for dx := 0; dx < dims.Size.X/2; dx++ {
				g := float64(pixel(img, 10+dx, 10).G)
				missing += (g - float64(sc.fill.G)) / (float64(sc.bar.G) - float64(sc.fill.G))
			}
			if missing < 2.6 || missing > 3.5 {
				t.Errorf("the top row misses %.2f px of fill at the leading corner, want about 3.05, a 5 px corner", missing)
			}
		})
	}
}

// TestFindBarStepsActOnPressAndKeepNoChoice presses each segment of the find
// bar's previous and next control: each press calls its own callback once,
// and after it both segments draw the same resting fill, so nothing records
// which one was pressed. While HasQuery answers false, a press acts on
// nothing.
func TestFindBarStepsActOnPressAndKeepNoChoice(t *testing.T) {
	var prev, next int
	query := true
	w := materialize(t, button.FindBarSteps(rx.Of(theme.Default()), button.FindBarStepsProps{
		PreviousDescription: "Previous",
		NextDescription:     "Next",
		HasQuery:            func() bool { return query },
		OnPrevious:          func(layout.Context) { prev++ },
		OnNext:              func(layout.Context) { next++ },
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
	press := func(x float32) {
		pos := f32.Pt(x, 10)
		r.Queue(
			pointer.Event{Kind: pointer.Press, Position: pos, Buttons: pointer.ButtonPrimary, Source: pointer.Mouse},
			pointer.Event{Kind: pointer.Release, Position: pos, Buttons: pointer.ButtonPrimary, Source: pointer.Mouse},
		)
		drive()
		drive()
	}
	drive()
	if dims.Size != image.Pt(33, 20) {
		t.Fatalf("the control is %v, want the measured 33 by 20", dims.Size)
	}
	press(28)
	if next != 1 || prev != 0 {
		t.Fatalf("pressing next: next %d, previous %d, want 1 and 0", next, prev)
	}
	press(4)
	if next != 1 || prev != 1 {
		t.Fatalf("pressing previous: next %d, previous %d, want 1 and 1", next, prev)
	}

	p := tokens.PlatformLight
	img := golden.Capture(t, image.Pt(60, 40), onBar(p.WindowBackground, func(gtx layout.Context) layout.Dimensions {
		return button.BorderedSegments(gtx, nil, p, tokens.Radius, tokens.TextStyle{}, tokens.Comfortable, []button.BorderedSegment{
			{State: button.RenderState{Variant: button.FindBar, Checked: true}},
			{State: button.RenderState{Variant: button.FindBar}},
		})
	}))
	if a, b := pixel(img, 10+8, 10+3), pixel(img, 10+25, 10+3); a != b || a != p.FindBarButtonFill {
		t.Errorf("a segment stated checked reads %v and the other %v; the find bar control keeps no choice, both read %v", a, b, p.FindBarButtonFill)
	}

	query = false
	drive()
	press(28)
	press(4)
	if next != 1 || prev != 1 {
		t.Errorf("with no query the presses acted: next %d, previous %d, want 1 and 1 still", next, prev)
	}
}

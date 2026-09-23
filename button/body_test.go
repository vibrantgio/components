package button_test

import (
	"image"
	stdcolor "image/color"
	"math"
	"testing"

	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"

	"github.com/vibrantgio/components/button"
	golden "github.com/vibrantgio/components/golden"
	"github.com/vibrantgio/components/internal/control"
	"github.com/vibrantgio/theme/tokens"
)

// The body control's measured numbers at the 1:1 metric golden.Capture
// renders at.
// bodyH is the dialog control's height, which is what Place.Body draws a
// bordered control at.
var bodyH = int(tokens.Comfortable.ControlHeight)

const (
	// bodyCorner is the push button's corner. MEASURED,
	// save-dialog-{light,dark}.png: a circular fit to the per-row coverage of
	// "Cancel"'s corners, the box's own extremes pinned, gives r = 6.11 light
	// (rms 0.088 px) and 6.17 dark (rms 0.069), all four corners agreeing to
	// the hundredth. That is tokens.RadiusScale.Md.
	bodyCorner = 6.0

	// bodyMarkTop and bodyMarkBottom are the room the platform leaves above
	// and below a mark inside the control it stands in. MEASURED, the same
	// captures: the "File Format:" pop-up runs y 336-359 and its pair covers
	// y 343-353.
	bodyMarkTop    = int(control.MarkClearTopDp)
	bodyMarkBottom = int(control.MarkClearBottomDp)

	// cornerRows is the corner's own reach: every row the arc can bend in.
	cornerRows = 8
)

// square fills the whole box a mark is handed, so what is read back off a
// capture is the box the control gave the symbol and not a glyph's own
// keyline.
func square(gtx layout.Context, sizePx int, col stdcolor.NRGBA) {
	paint.FillShape(gtx.Ops, col, clip.Rect{Max: image.Pt(sizePx, sizePx)}.Op())
}

// bareBody lays a body control out at the capture's own origin with nothing
// painted under it, which is what leaves the rasterizer's coverage in the
// capture's alpha channel for a corner to be read off.
func bareBody(t *testing.T, p tokens.PlatformColors, icon func(layout.Context, int, stdcolor.NRGBA), size image.Point) (*image.RGBA, image.Point) {
	t.Helper()
	var box image.Point
	img := golden.Capture(t, size, func(gtx layout.Context) layout.Dimensions {
		gtx.Metric = unit.Metric{PxPerDp: 1, PxPerSp: 1}
		gtx.Constraints.Min = image.Point{}
		d := button.RenderChrome(icon, p, tokens.Radius, tokens.Comfortable, button.RenderState{
			Place:   button.Body,
			Surface: p.WindowBackground,
		})(gtx)
		box = d.Size
		return d
	})
	return img, box
}

// bodyReadings are the two appearances every reading below is taken in.
var bodyReadings = []struct {
	name string
	p    tokens.PlatformColors
}{
	{"light", tokens.PlatformLight},
	{"dark", tokens.PlatformDark},
}

// TestABodyControlDrawsTheFormsCorner reads the corner of a bordered control
// standing in a sheet's or a pane's body off a capture, in both appearances.
//
// A body control is the form's shape and not the band's. MEASURED,
// save-dialog-{light,dark}.png: the sheet's "Cancel" fits r = 6.11 light and
// 6.17 dark against a control 24 px tall, where the same sheet's toolbar-band
// neighbours are capsules at half their height. So a control that took the
// body's height and kept the band's capsule drew a 12 px corner where the
// platform draws 6. Read back off the drawing the same fit answers r = 5.97
// at an rms of 0.010 px over eight rows, in both appearances.
func TestABodyControlDrawsTheFormsCorner(t *testing.T) {
	for _, tc := range bodyReadings {
		t.Run(tc.name, func(t *testing.T) {
			img, box := bareBody(t, tc.p, nil, image.Pt(80, 60))
			if img == nil {
				return
			}
			if box.Y != bodyH {
				t.Fatalf("the body control measures %d tall, want the dialog control's own %d", box.Y, bodyH)
			}
			profile := map[int]float64{}
			for y := range cornerRows {
				e, ok := golden.LeadingEdge(golden.CoverageRow(img, y, box.X))
				if !ok {
					t.Fatalf("row %d of the control carries no fully covered column", y)
				}
				profile[y] = e
			}
			r, rms := golden.FitCorner(profile, 0, 0)
			t.Logf("the body control's corner fits r = %.2f, rms %.3f px over %d rows", r, rms, len(profile))
			if math.Abs(r-bodyCorner) > 0.3 || rms > 0.1 {
				t.Errorf("the body control's corner fits r = %.2f (rms %.3f px over %d rows), want the push button's measured %.0f",
					r, rms, len(profile), bodyCorner)
			}
			if half := float64(box.Y) / 2; math.Abs(r-half) < 0.3 {
				t.Errorf("the body control's corner fits the capsule's half-height %.1f; that is the band's shape, not the form's", half)
			}
		})
	}
}

// TestABodyControlsMarkKeepsThePlatformsRoom reads the room a body control
// leaves above and below its mark off a capture, in both appearances.
//
// MEASURED, save-dialog-{light,dark}.png: the "File Format:" pop-up runs
// y 336-359, 24 px tall, and its mark covers y 343-353 — seven rows above it
// and six below, leaving an eleven-row band for an eight by eleven mark. The
// band's own 24 dp mark box in a 24 dp control leaves none of that: the
// symbol runs the control's whole height. Read back off the drawing the mark
// stands seven rows down from the control's top with six clear under it, in
// both appearances.
func TestABodyControlsMarkKeepsThePlatformsRoom(t *testing.T) {
	for _, tc := range bodyReadings {
		t.Run(tc.name, func(t *testing.T) {
			img, box := bareBody(t, tc.p, square, image.Pt(80, 60))
			if img == nil {
				return
			}
			fill := tc.p.PushButtonFill
			mid := box.X / 2
			first, last := -1, -1
			for y := range box.Y {
				if c := img.RGBAAt(mid, y); !nearlyEqual(c, fill) {
					if first < 0 {
						first = y
					}
					last = y
				}
			}
			if first < 0 {
				t.Fatal("the control's middle column carries no mark at all")
			}
			above, below := first, box.Y-1-last
			if above != bodyMarkTop || below != bodyMarkBottom {
				t.Errorf("the mark stands %d rows below the control's top and %d above its foot, want the platform's measured %d and %d",
					above, below, bodyMarkTop, bodyMarkBottom)
			}
			if band := last - first + 1; band != box.Y-bodyMarkTop-bodyMarkBottom {
				t.Errorf("the mark's band is %d rows in a control of %d, want %d", band, box.Y, box.Y-bodyMarkTop-bodyMarkBottom)
			}
		})
	}
}

// TestABodySegmentedControlRoundsItsOuterCornersOnly reads the shape of a
// segmented control standing in a body: the control's outer corners take the
// push button's measured 6 and the seam between two segments stays the
// straight hairline it is in a band, running to neither corner. Read back off
// the drawing the outer corner fits r = 5.97 at an rms of 0.010 px over eight
// rows, in both appearances.
func TestABodySegmentedControlRoundsItsOuterCornersOnly(t *testing.T) {
	for _, tc := range bodyReadings {
		t.Run(tc.name, func(t *testing.T) {
			segs := []button.ChromeSegment{
				{Icon: square, State: button.RenderState{Place: button.Body}},
				{Icon: square, State: button.RenderState{Place: button.Body}},
			}
			var box image.Point
			img := golden.Capture(t, image.Pt(120, 60), func(gtx layout.Context) layout.Dimensions {
				gtx.Metric = unit.Metric{PxPerDp: 1, PxPerSp: 1}
				gtx.Constraints.Min = image.Point{}
				d := button.ChromeSegments(gtx, defaultShaper(t), tc.p, tokens.Radius,
					tokens.DefaultTypography.LabelLarge, tokens.Comfortable, segs)
				box = d.Size
				return d
			})
			if img == nil {
				return
			}
			if box.Y != bodyH {
				t.Fatalf("the body pair measures %d tall, want the dialog control's own %d", box.Y, bodyH)
			}
			profile := map[int]float64{}
			for y := range cornerRows {
				e, ok := golden.LeadingEdge(golden.CoverageRow(img, y, box.X))
				if !ok {
					t.Fatalf("row %d of the control carries no fully covered column", y)
				}
				profile[y] = e
			}
			r, rms := golden.FitCorner(profile, 0, 0)
			t.Logf("the body pair's outer corner fits r = %.2f, rms %.3f px over %d rows", r, rms, len(profile))
			if math.Abs(r-bodyCorner) > 0.3 || rms > 0.1 {
				t.Errorf("the body pair's outer corner fits r = %.2f (rms %.3f px over %d rows), want the push button's measured %.0f",
					r, rms, len(profile), bodyCorner)
			}
			// The seam's own column: the control's shape runs to the top and
			// the foot there, which is what says the inner parting is
			// straight and not a second pair of corners.
			seam := (box.X - 1) / 2
			for _, y := range []int{0, box.Y - 1} {
				if img.RGBAAt(seam, y).A != 0xff {
					t.Errorf("the control's shape is cut away at the seam's column on row %d; only the outer corners round", y)
				}
			}
			clear := int(button.SegmentSeamClearDp)
			rule := tc.p.ToolbarControlSeam
			if got := img.RGBAAt(seam, box.Y/2); !nearlyEqual(got, rule) {
				t.Errorf("the seam's middle row reads %v, want the control's own measured seam %v", got, rule)
			}
			if nearlyEqual(img.RGBAAt(seam, clear-1), rule) {
				t.Errorf("the seam reaches row %d; it leaves the control's measured %d rows clear at each end", clear-1, clear)
			}
		})
	}
}

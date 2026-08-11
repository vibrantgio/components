package hit_test

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

	"github.com/vibrantgio/components/internal/hit"
)

// visualPx is the side of the small drawn control; hitPx is the extended
// pointer floor. The slop is (hitPx-visualPx)/2 = 12 px on each side, so the
// hit rectangle spans -12..32 on both axes around a 0..20 visual.
const (
	visualPx = 20
	hitPx    = 44
)

// square draws a visualPx×visualPx filled square at the origin.
func square(gtx layout.Context) layout.Dimensions {
	sz := image.Pt(visualPx, visualPx)
	paint.FillShape(gtx.Ops, color.NRGBA{R: 0xff, A: 0xff}, clip.Rect{Max: sz}.Op())
	return layout.Dimensions{Size: sz}
}

// drive lays out the extended clickable for one frame, polling for clicks the
// way a live component does (Clicked must run inside the frame that processes
// the pointer events), and delivers queued events.
func drive(r *gioinput.Router, click *widget.Clickable) (layout.Dimensions, bool) {
	ops := new(op.Ops)
	gtx := layout.Context{
		Metric:      unit.Metric{PxPerDp: 1, PxPerSp: 1},
		Constraints: layout.Exact(image.Pt(100, 100)),
		Ops:         ops,
		Source:      r.Source(),
	}
	clicked := click.Clicked(gtx)
	dims := hit.Extend(gtx, hitPx, click.Layout, square)
	r.Frame(ops)
	return dims, clicked
}

// TestExtendReportsVisualDimensions confirms the parent sees the dense visual
// size, not the inflated hit rectangle.
func TestExtendReportsVisualDimensions(t *testing.T) {
	r := new(gioinput.Router)
	var click widget.Clickable
	dims, _ := drive(r, &click)
	if dims.Size != image.Pt(visualPx, visualPx) {
		t.Errorf("Extend dims = %v, want %dx%d (the visual size)", dims.Size, visualPx, visualPx)
	}
}

// TestExtendClickInSlopActivates confirms a press outside the visual bounds
// but inside the extended hit rectangle activates the clickable.
func TestExtendClickInSlopActivates(t *testing.T) {
	r := new(gioinput.Router)
	var click widget.Clickable

	drive(r, &click) // register the input area

	// (30, 30) is outside the 20×20 visual, inside the -12..32 hit rect.
	pos := f32.Pt(30, 30)
	r.Queue(
		pointer.Event{Kind: pointer.Press, Position: pos, Buttons: pointer.ButtonPrimary, Source: pointer.Mouse},
		pointer.Event{Kind: pointer.Release, Position: pos, Buttons: pointer.ButtonPrimary, Source: pointer.Mouse},
	)
	_, clicked := drive(r, &click)
	if !clicked {
		t.Error("click at (30,30) in the hit slop did not activate; pointer target shrank below the floor")
	}
}

// TestExtendClickOutsideHitDoesNothing confirms a press beyond the extended
// hit rectangle does not activate the clickable.
func TestExtendClickOutsideHitDoesNothing(t *testing.T) {
	r := new(gioinput.Router)
	var click widget.Clickable

	drive(r, &click)

	// (40, 40) is outside the -12..32 hit rect.
	pos := f32.Pt(40, 40)
	r.Queue(
		pointer.Event{Kind: pointer.Press, Position: pos, Buttons: pointer.ButtonPrimary, Source: pointer.Mouse},
		pointer.Event{Kind: pointer.Release, Position: pos, Buttons: pointer.ButtonPrimary, Source: pointer.Mouse},
	)
	_, clicked := drive(r, &click)
	if clicked {
		t.Error("click at (40,40) outside the hit rect activated the clickable")
	}
	// A frame later there is still nothing pending.
	if _, clicked = drive(r, &click); clicked {
		t.Error("stray click reported a frame after an outside-hit press")
	}
}

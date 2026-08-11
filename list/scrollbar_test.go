package list

import (
	"image"
	"image/color"
	"testing"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"

	"github.com/vibrantgio/components/scrollbar"
	"github.com/vibrantgio/theme/tokens"
)

const (
	sbRowPx = 30  // fixed row height in pixels
	sbViewW = 200 // viewport width in pixels
	sbViewH = 150 // viewport height in pixels; fits exactly 5 rows
)

func sbTestContext() layout.Context {
	return layout.Context{
		Metric:      unit.Metric{PxPerDp: 1, PxPerSp: 1},
		Constraints: layout.Exact(image.Pt(sbViewW, sbViewH)),
		Ops:         new(op.Ops),
	}
}

func sbItems(n int) []int {
	s := make([]int, n)
	for i := range s {
		s[i] = i
	}
	return s
}

// sbRowFn draws a fixed-height row spanning the full offered width and
// records that width in *seenWidth.
func sbRowFn(seenWidth *int) func(gtx layout.Context, item int) layout.Dimensions {
	return func(gtx layout.Context, item int) layout.Dimensions {
		*seenWidth = gtx.Constraints.Max.X
		size := image.Pt(gtx.Constraints.Max.X, sbRowPx)
		paint.FillShape(gtx.Ops, color.NRGBA{R: 128, G: 128, B: 128, A: 255}, clip.Rect{Max: size}.Op())
		return layout.Dimensions{Size: size}
	}
}

// TestLayoutScrollbarOccupyNarrowsRows asserts that the Occupy anchor
// reserves a gutter of exactly gtx.Dp(bar.Width()) pixels: rows are offered
// the viewport width minus the gutter, while the reported dimensions are
// re-widened to the full viewport width.
func TestLayoutScrollbarOccupyNarrowsRows(t *testing.T) {
	bar := scrollbar.FromTokens(tokens.DefaultLight)
	state := NewState()
	gtx := sbTestContext()
	barWidth := gtx.Dp(bar.Width())
	if barWidth <= 0 {
		t.Fatalf("bar width = %d px; want > 0 for a meaningful test", barWidth)
	}

	var seenWidth int
	dims := LayoutScrollbar(gtx, state, bar, Occupy, sbItems(100), sbRowFn(&seenWidth))

	if want := sbViewW - barWidth; seenWidth != want {
		t.Errorf("Occupy row width = %d; want %d (viewport %d - bar %d)", seenWidth, want, sbViewW, barWidth)
	}
	if dims.Size.X != sbViewW {
		t.Errorf("Occupy reported width = %d; want %d (re-widened by gutter)", dims.Size.X, sbViewW)
	}
	if dims.Size.Y != sbViewH {
		t.Errorf("Occupy reported height = %d; want %d", dims.Size.Y, sbViewH)
	}
}

// TestApplyScrollDeltaAdvancesPosition simulates a scrollbar drag by feeding
// a hand-set distance to the feedback path and asserts the list position
// advances on the next layout.
//
// 100 rows of 30px in a 150px viewport: a delta of 0.5 (half the content)
// should scroll by 50 rows.
func TestApplyScrollDeltaAdvancesPosition(t *testing.T) {
	bar := scrollbar.FromTokens(tokens.DefaultLight)
	state := NewState()
	items := sbItems(100)

	var seenWidth int
	LayoutScrollbar(sbTestContext(), state, bar, Occupy, items, sbRowFn(&seenWidth))
	if got := state.l.Position.First; got != 0 {
		t.Fatalf("Position.First after initial layout = %d; want 0", got)
	}

	// Simulate the drag: half of the total content.
	state.applyScrollDelta(0.5, len(items))

	// ScrollBy takes effect on the next layout.
	LayoutScrollbar(sbTestContext(), state, bar, Occupy, items, sbRowFn(&seenWidth))
	if got := state.l.Position.First; got <= 0 {
		t.Errorf("Position.First after drag = %d; want > 0 (position should advance)", got)
	}

	// A zero delta must not move the list.
	before := state.l.Position.First
	state.applyScrollDelta(0, len(items))
	LayoutScrollbar(sbTestContext(), state, bar, Occupy, items, sbRowFn(&seenWidth))
	if got := state.l.Position.First; got != before {
		t.Errorf("Position.First after zero delta = %d; want %d (unchanged)", got, before)
	}
}

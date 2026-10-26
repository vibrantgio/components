package list

import (
	"image"
	"math"
	"testing"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"

	"github.com/vibrantgio/components/scrollbar"
	"github.com/vibrantgio/theme/tokens"
)

// Rows of unequal height: the first five are 40 px tall and the other fifteen
// 10 px, so the content is 5×40 + 15×10 = 350 px in a 100 px viewport. A
// mean taken over the rows at the start is four times the mean over the rest.
const (
	unequalRows    = 20
	unequalView    = 100
	unequalContent = 5*40 + 15*10
)

func unequalHeight(_ unit.Metric, i int) int {
	if i < 5 {
		return 40
	}
	return 10
}

// layoutUnequal lays the rows out once under a fading bar and returns the
// span the bar reads.
func layoutUnequal(state *State) (start, end float32) {
	gtx := layout.Context{
		Metric:      unit.Metric{PxPerDp: 1, PxPerSp: 1},
		Constraints: layout.Exact(image.Pt(200, unequalView)),
		Ops:         new(op.Ops),
	}
	bar := scrollbar.FromTokens(tokens.PlatformLight, tokens.PlatformLight.ControlBackground)
	LayoutScrollbar(gtx, state, bar, Overlay, sbItems(unequalRows), func(gtx layout.Context, i int) layout.Dimensions {
		return layout.Dimensions{Size: image.Pt(gtx.Constraints.Max.X, unequalHeight(gtx.Metric, i))}
	})
	return state.span(gtx.Metric, unequalView)
}

func near(a, b float32) bool { return math.Abs(float64(a-b)) < 1e-6 }

// TestSpanReadsTheHeightOfRowExactly gives the list the rows' heights. The
// bar's span is then the exact fraction of the content in view: at rest
// before a row past the first screen is laid out, and after a scroll.
func TestSpanReadsTheHeightOfRowExactly(t *testing.T) {
	state := NewState()
	state.SetHeightOfRow(unequalHeight)

	start, end := layoutUnequal(state)
	if !near(start, 0) || !near(end, float32(unequalView)/unequalContent) {
		t.Errorf("at rest the span is (%v, %v); want (0, %v)", start, end, float32(unequalView)/unequalContent)
	}

	for _, scrolled := range []int{30, 130, 250} {
		state.ScrollToStart()
		layoutUnequal(state)
		state.ScrollPixels(scrolled)
		start, end = layoutUnequal(state)
		wantStart := float32(scrolled) / unequalContent
		wantEnd := float32(scrolled+unequalView) / unequalContent
		if !near(start, wantStart) || !near(end, wantEnd) {
			t.Errorf("after %d px the span is (%v, %v); want (%v, %v)", scrolled, start, end, wantStart, wantEnd)
		}
	}
}

// TestSpanRecordsTheHeightsLaidOut gives the list no heights. At rest the
// rows not yet laid out count at the mean of the tall rows in view; once the
// list has laid every row out, the recorded heights replace the mean and the
// span is exact.
func TestSpanRecordsTheHeightsLaidOut(t *testing.T) {
	state := NewState()

	_, end := layoutUnequal(state)
	if exact := float32(unequalView) / unequalContent; end >= exact {
		t.Errorf("at rest with only tall rows laid out the span ends at %v; want under the exact %v", end, exact)
	}

	for range unequalContent / 10 {
		state.ScrollPixels(10)
		layoutUnequal(state)
	}
	state.ScrollToStart()
	start, end := layoutUnequal(state)
	if !near(start, 0) || !near(end, float32(unequalView)/unequalContent) {
		t.Errorf("at rest after every row was laid out the span is (%v, %v); want (0, %v)", start, end, float32(unequalView)/unequalContent)
	}
}

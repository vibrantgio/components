package list_test

import (
	"image"
	"image/color"
	"testing"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"

	golden "github.com/vibrantgio/prism/internal/golden"
	"github.com/vibrantgio/prism/list"
	"github.com/vibrantgio/prism/scrollbar"
	"github.com/vibrantgio/spectrum/tokens"
)

const (
	rowPx = 30  // fixed row height in pixels
	viewW = 200 // viewport width in pixels
	viewH = 150 // viewport height in pixels; fits exactly 5 rows
)

// rowColor maps item index to a unique shade of gray in [15, 243].
// Items 0..19 all produce distinct values, so short/long/scrolled-mid
// goldens are visually distinguishable.
func rowColor(item int) color.NRGBA {
	v := uint8(15 + (item%19)*12)
	return color.NRGBA{R: v, G: v, B: v, A: 0xff}
}

// colorRowFn draws a solid-gray row of fixed pixel height.
func colorRowFn(gtx layout.Context, item int) layout.Dimensions {
	size := image.Pt(gtx.Constraints.Max.X, rowPx)
	paint.FillShape(gtx.Ops, rowColor(item), clip.Rect{Max: size}.Op())
	return layout.Dimensions{Size: size}
}

func makeItems(n int) []int {
	s := make([]int, n)
	for i := range s {
		s[i] = i
	}
	return s
}

// TestListGolden records or diffs three canonical list configurations:
// short (all items visible), long (many items at top), scrolled-mid (scrolled
// into the middle of a list).
func TestListGolden(t *testing.T) {
	size := image.Pt(viewW, viewH)
	cases := []struct {
		name  string
		items []int
		state *list.State
	}{
		{"short", makeItems(3), list.NewState()},
		{"long", makeItems(20), list.NewState()},
		{"scrolled-mid", makeItems(20), list.NewStateAt(8)},
	}
	for _, tc := range cases {
		items := tc.items
		state := tc.state
		t.Run(tc.name, func(t *testing.T) {
			golden.Render(t, tc.name, size, func(gtx layout.Context) layout.Dimensions {
				return list.Layout(gtx, state, items, colorRowFn)
			})
		})
	}
}

// TestListDensityRowGolden records or diffs a list whose rows are the
// standard density-derived height, list.RowHeight (E1.3: row = ControlHeight;
// 36 px Comfortable, 28 px Compact at 1:1 scale). The 150 px viewport shows
// the divergence directly: just over four comfortable rows against just over
// five compact ones.
func TestListDensityRowGolden(t *testing.T) {
	size := image.Pt(viewW, viewH)
	cases := []struct {
		name    string
		density tokens.Density
	}{
		{"rows-comfortable", tokens.Comfortable},
		{"rows-compact", tokens.Compact},
	}
	for _, tc := range cases {
		d := tc.density
		t.Run(tc.name, func(t *testing.T) {
			state := list.NewState()
			items := makeItems(8)
			rowFn := func(gtx layout.Context, item int) layout.Dimensions {
				rowSize := image.Pt(gtx.Constraints.Max.X, gtx.Dp(list.RowHeight(d)))
				paint.FillShape(gtx.Ops, rowColor(item), clip.Rect{Max: rowSize}.Op())
				return layout.Dimensions{Size: rowSize}
			}
			golden.Render(t, tc.name, size, func(gtx layout.Context) layout.Dimensions {
				return list.Layout(gtx, state, items, rowFn)
			})
		})
	}
}

// TestScrollbarGolden records or diffs the two scrollbar anchor modes at the
// same viewport size as the plain list goldens, scrolled into the middle of a
// 20-item list so the thumb sits mid-track:
//
//   - scrollbar-occupy: rows are visibly narrower, the bar sits in its own
//     gutter along the east edge.
//   - scrollbar-overlay: rows span the full viewport width, the bar floats
//     over their trailing edge.
func TestScrollbarGolden(t *testing.T) {
	size := image.Pt(viewW, viewH)
	bar := scrollbar.FromTokens(tokens.DefaultLight)
	cases := []struct {
		name   string
		anchor list.Anchor
	}{
		{"scrollbar-occupy", list.Occupy},
		{"scrollbar-overlay", list.Overlay},
	}
	for _, tc := range cases {
		anchor := tc.anchor
		t.Run(tc.name, func(t *testing.T) {
			items := makeItems(20)
			state := list.NewStateAt(8)
			golden.Render(t, tc.name, size, func(gtx layout.Context) layout.Dimensions {
				return list.LayoutScrollbar(gtx, state, bar, anchor, items, colorRowFn)
			})
		})
	}
}

// TestLayoutScrollbarNonScrollable confirms that when all items fit in the
// viewport (3 rows in a tall viewport) LayoutScrollbar renders no bar: the
// Overlay output is pixel-identical to plain Layout and both anchors report
// the same dimensions as plain Layout.
func TestLayoutScrollbarNonScrollable(t *testing.T) {
	size := image.Pt(viewW, 300) // 3 rows of 30px in a 300px viewport
	bar := scrollbar.FromTokens(tokens.DefaultLight)

	var plainDims layout.Dimensions
	plainImg := golden.Capture(t, size, func(gtx layout.Context) layout.Dimensions {
		plainDims = list.Layout(gtx, list.NewState(), makeItems(3), colorRowFn)
		return plainDims
	})

	var overlayDims layout.Dimensions
	overlayImg := golden.Capture(t, size, func(gtx layout.Context) layout.Dimensions {
		overlayDims = list.LayoutScrollbar(gtx, list.NewState(), bar, list.Overlay, makeItems(3), colorRowFn)
		return overlayDims
	})

	if overlayDims != plainDims {
		t.Errorf("Overlay dims = %v; want %v (same as plain Layout)", overlayDims, plainDims)
	}
	if n := golden.PixelDiff(plainImg, overlayImg); n != 0 {
		t.Errorf("Overlay output differs from plain Layout by %d pixel(s); want 0 (no bar rendered)", n)
	}

	var occupyDims layout.Dimensions
	golden.Capture(t, size, func(gtx layout.Context) layout.Dimensions {
		occupyDims = list.LayoutScrollbar(gtx, list.NewState(), bar, list.Occupy, makeItems(3), colorRowFn)
		return occupyDims
	})
	if occupyDims != plainDims {
		t.Errorf("Occupy dims = %v; want %v (same as plain Layout)", occupyDims, plainDims)
	}
}

// TestLayoutCallsRowFnOnlyForVisibleItems is the counter-based proof that
// Layout calls rowFn O(visible) times rather than O(len(items)).
//
// With viewH=150px and rowPx=30px, exactly 5 rows fit in the viewport.
// Gio may fetch one extra item for look-ahead, so ≤10 calls is the safe bound.
// This is verified against a 1000-item list where O(total) would be 1000 calls.
func TestLayoutCallsRowFnOnlyForVisibleItems(t *testing.T) {
	items := makeItems(1000)
	state := list.NewState()

	var calls int
	var ops op.Ops
	gtx := layout.Context{
		Metric:      unit.Metric{PxPerDp: 1, PxPerSp: 1},
		Constraints: layout.Exact(image.Pt(viewW, viewH)),
		Ops:         &ops,
	}

	list.Layout(gtx, state, items, func(gtx layout.Context, item int) layout.Dimensions {
		calls++
		return colorRowFn(gtx, item)
	})

	const maxVisible = 10 // 5 rows + generous look-ahead buffer; far less than 1000
	if calls > maxVisible {
		t.Errorf("rowFn called %d times for 1000-item list (viewport %dpx, row %dpx); want ≤ %d (O(visible))",
			calls, viewH, rowPx, maxVisible)
	}
	if calls == 0 {
		t.Error("rowFn never called; list should render at least 1 row")
	}
}

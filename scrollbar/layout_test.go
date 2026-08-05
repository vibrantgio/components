package scrollbar

import (
	"image"
	"testing"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"

	"github.com/vibrantgio/spectrum/tokens"
)

// testContext returns a bare layout context at the given size with a
// 1:1 dp-to-px metric.
func testContext(ops *op.Ops, size image.Point) layout.Context {
	return layout.Context{
		Ops:         ops,
		Metric:      unit.Metric{PxPerDp: 1, PxPerSp: 1},
		Constraints: layout.Constraints{Max: size},
	}
}

func TestLayoutScrollableRange(t *testing.T) {
	style := FromTokens(tokens.DefaultLight)
	size := image.Pt(40, 400)

	cases := []struct {
		name string
		axis layout.Axis
		want image.Point
	}{
		// The bar pins itself to the full major axis and Width() (10dp)
		// along the minor axis, regardless of the incoming minor extent.
		{"Vertical", layout.Vertical, image.Pt(10, 400)},
		{"Horizontal", layout.Horizontal, image.Pt(40, 10)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			state := NewState()
			var ops op.Ops
			gtx := testContext(&ops, size)

			dims := style.Layout(gtx, state, tc.axis, 0.25, 0.5)
			if dims.Size != tc.want {
				t.Errorf("Layout(0.25, 0.5) dims = %v, want %v", dims.Size, tc.want)
			}
		})
	}
}

func TestLayoutUnscrollableRange(t *testing.T) {
	style := FromTokens(tokens.DefaultLight)
	state := NewState()
	var ops op.Ops
	gtx := testContext(&ops, image.Pt(40, 400))

	dims := style.Layout(gtx, state, layout.Vertical, 0, 1)
	if dims != (layout.Dimensions{}) {
		t.Errorf("Layout(0, 1) dims = %v, want zero dimensions", dims)
	}
}

package main

import (
	"testing"

	"gioui.org/layout"

	"github.com/vibrantgio/theme/tokens"
)

// TestInputsPageShowsTheBorderlessField asserts that the inputs page holds
// the borderless text field in each state its goldens hold, and that every
// row draws.
func TestInputsPageShowsTheBorderlessField(t *testing.T) {
	g := testGallery(t)
	rows := g.borderlessFieldRows()
	if len(rows) != 7 {
		t.Fatalf("the inputs page holds %d borderless rows, want the 7 golden states", len(rows))
	}
	for i, r := range rows {
		w := func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx, r)
		}
		if h := measure(tokens.PlatformLight, w, pageWidth, 1<<20).Y; h <= 1 {
			t.Errorf("borderless row %d draws %d px tall", i, h)
		}
	}
}

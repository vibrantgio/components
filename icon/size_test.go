package icon_test

import (
	"image"
	"image/color"
	"testing"

	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"

	"github.com/vibrantgio/prism/icon"
	golden "github.com/vibrantgio/prism/internal/golden"
	"github.com/vibrantgio/spectrum/tokens"
)

// TestSizeFollowsDensity pins the content-box rule: the default glyph size is
// ControlHeight − 2·PaddingY — 20 dp Comfortable, 16 dp Compact — so the icon
// scales in lockstep with the control it sits in.
func TestSizeFollowsDensity(t *testing.T) {
	if got := icon.Size(tokens.Comfortable); got != 20 {
		t.Errorf("Size(Comfortable) = %v dp, want 20", got)
	}
	if got := icon.Size(tokens.Compact); got != 16 {
		t.Errorf("Size(Compact) = %v dp, want 16", got)
	}
	if got := icon.DefaultSize; got != icon.Size(tokens.Comfortable) {
		t.Errorf("DefaultSize = %v dp, want Size(Comfortable)", got)
	}
}

// TestIconDensityGolden records or diffs a deterministic glyph stand-in (a
// solid square, no SVG/IVG rasterisation) at each density's default icon
// size: 20 px Comfortable, 16 px Compact at 1:1 scale.
func TestIconDensityGolden(t *testing.T) {
	cases := []struct {
		name    string
		density tokens.Density
	}{
		{"icon-comfortable", tokens.Comfortable},
		{"icon-compact", tokens.Compact},
	}
	for _, tc := range cases {
		d := tc.density
		t.Run(tc.name, func(t *testing.T) {
			golden.Render(t, tc.name, image.Pt(24, 24), func(gtx layout.Context) layout.Dimensions {
				sz := gtx.Dp(icon.Size(d))
				paint.FillShape(gtx.Ops, color.NRGBA{R: 0x33, G: 0x33, B: 0x33, A: 0xff},
					clip.Rect{Max: image.Pt(sz, sz)}.Op())
				return layout.Dimensions{Size: image.Pt(sz, sz)}
			})
		})
	}
}

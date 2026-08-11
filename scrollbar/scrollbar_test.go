package scrollbar

import (
	"image"
	"image/color"
	"testing"

	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"

	golden "github.com/vibrantgio/prism/golden"
	"github.com/vibrantgio/theme/tokens"
)

// TestScrollbarGolden records or diffs canonical vertical scrollbar renders on
// a Surface-filled background: thumb at top, middle and bottom in the light
// scheme, the same middle position in the dark scheme (colours must differ),
// and a near-zero viewport fraction proving the 16dp minimum thumb length.
func TestScrollbarGolden(t *testing.T) {
	size := image.Pt(24, 400)
	cases := []struct {
		name       string
		c          tokens.ColorTokens
		start, end float32
	}{
		{"light-top", tokens.DefaultLight, 0, 0.3},
		{"light-mid", tokens.DefaultLight, 0.35, 0.65},
		{"dark-mid", tokens.DefaultDark, 0.35, 0.65},
		{"light-bottom", tokens.DefaultLight, 0.7, 1.0},
		{"min-thumb", tokens.DefaultLight, 0.5, 0.501},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			state := NewState()
			style := FromTokens(tc.c)
			surface := tc.c.Surface
			start, end := tc.start, tc.end
			golden.Render(t, tc.name, size, func(gtx layout.Context) layout.Dimensions {
				gtx.Metric = unit.Metric{PxPerDp: 1, PxPerSp: 1}
				paint.FillShape(gtx.Ops, surface, clip.Rect{Max: gtx.Constraints.Max}.Op())
				style.Layout(gtx, state, layout.Vertical, start, end)
				return layout.Dimensions{Size: gtx.Constraints.Max}
			})
		})
	}
}

func TestFromTokens(t *testing.T) {
	cases := []struct {
		name string
		c    tokens.ColorTokens
	}{
		{"DefaultLight", tokens.DefaultLight},
		{"DefaultDark", tokens.DefaultDark},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := FromTokens(tc.c)

			wantThumb := tc.c.Ramps.Neutral.Step(700)
			wantThumb.A = 100
			if s.ThumbColor != wantThumb {
				t.Errorf("ThumbColor = %v, want %v", s.ThumbColor, wantThumb)
			}

			wantHover := tc.c.Ramps.Neutral.Step(700)
			wantHover.A = 170
			if s.ThumbHoverColor != wantHover {
				t.Errorf("ThumbHoverColor = %v, want %v", s.ThumbHoverColor, wantHover)
			}

			if s.TrackColor != (color.NRGBA{}) {
				t.Errorf("TrackColor = %v, want transparent zero value", s.TrackColor)
			}

			metrics := []struct {
				name string
				got  unit.Dp
				want unit.Dp
			}{
				{"ThumbMinorWidth", s.ThumbMinorWidth, 6},
				{"TrackPadding", s.TrackPadding, 2},
				{"ThumbCornerRadius", s.ThumbCornerRadius, 3},
				{"ThumbMinLen", s.ThumbMinLen, 16},
				{"Width()", s.Width(), 10},
			}
			for _, m := range metrics {
				if m.got != m.want {
					t.Errorf("%s = %v, want %v", m.name, m.got, m.want)
				}
			}
		})
	}
}

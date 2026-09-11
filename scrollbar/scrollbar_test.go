package scrollbar

import (
	"image"
	"image/color"
	"testing"

	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"

	golden "github.com/vibrantgio/components/golden"
	"github.com/vibrantgio/theme/tokens"
)

// TestScrollbarGolden records or diffs canonical vertical scrollbar renders on
// the content plane the bar rides: thumb at top, middle and bottom in the
// light scheme, the same middle position in the dark scheme (the composite
// must differ), and a near-zero viewport fraction proving the 16dp minimum
// thumb length.
func TestScrollbarGolden(t *testing.T) {
	size := image.Pt(24, 400)
	cases := []struct {
		name       string
		p          tokens.PlatformColors
		start, end float32
	}{
		{"light-top", tokens.PlatformLight, 0, 0.3},
		{"light-mid", tokens.PlatformLight, 0.35, 0.65},
		{"dark-mid", tokens.PlatformDark, 0.35, 0.65},
		{"light-bottom", tokens.PlatformLight, 0.7, 1.0},
		{"min-thumb", tokens.PlatformLight, 0.5, 0.501},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			state := NewState()
			style := FromTokens(tc.p)
			surface := tc.p.ControlBackground
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

// TestFromTokens pins every colour the default bar draws to the platform name
// it is taken from, and the metrics it hands out.
func TestFromTokens(t *testing.T) {
	for _, tc := range []struct {
		name string
		p    tokens.PlatformColors
	}{
		{"PlatformLight", tokens.PlatformLight},
		{"PlatformDark", tokens.PlatformDark},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := FromTokens(tc.p)

			if s.ThumbColor != tc.p.ScrollbarThumb {
				t.Errorf("ThumbColor = %v, want ScrollbarThumb %v", s.ThumbColor, tc.p.ScrollbarThumb)
			}
			if s.ThumbColor.A == 0xff {
				t.Errorf("ThumbColor = %v is opaque; an overlay thumb composites over what it rides", s.ThumbColor)
			}
			if s.TrackColor != (color.NRGBA{}) {
				t.Errorf("TrackColor = %v, want transparent zero value", s.TrackColor)
			}

			wantMatch := tc.p.FindHighlight
			wantMatch.A = matchCoverage
			if s.MatchFill != wantMatch {
				t.Errorf("MatchFill = %v, want FindHighlight at %#x, %v", s.MatchFill, matchCoverage, wantMatch)
			}
			wantCurrent := tc.p.FindHighlight
			wantCurrent.A = currentMatchCoverage
			if s.CurrentMatchFill != wantCurrent {
				t.Errorf("CurrentMatchFill = %v, want FindHighlight at %#x, %v", s.CurrentMatchFill, currentMatchCoverage, wantCurrent)
			}
			if s.MatchFill.A >= s.CurrentMatchFill.A {
				t.Errorf("the current match at %#x is not laid on more strongly than the rest at %#x",
					s.CurrentMatchFill.A, s.MatchFill.A)
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

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
	vgcolor "github.com/vibrantgio/theme/color"
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
			style := FromTokens(tc.p, tc.p.ControlBackground)
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
			s := FromTokens(tc.p, tc.p.ControlBackground)

			rides := tc.p.ControlBackground
			// Each of the three is the platform's name at its coverage,
			// resolved against what the bar rides: the platform composites
			// in encoded sRGB, so the bar paints opaque.
			if want := vgcolor.Flatten(tc.p.ScrollbarThumb, rides); s.ThumbColor != want {
				t.Errorf("ThumbColor = %v, want ScrollbarThumb over the content, %v", s.ThumbColor, want)
			}
			if s.ThumbColor.A != 0xff {
				t.Errorf("ThumbColor = %v carries a coverage; only the fade makes the thumb translucent", s.ThumbColor)
			}
			if s.TrackColor != (color.NRGBA{}) {
				t.Errorf("TrackColor = %v, want transparent zero value", s.TrackColor)
			}

			wantMatch := tc.p.FindHighlight
			wantMatch.A = matchCoverage
			if want := vgcolor.Flatten(wantMatch, rides); s.MatchFill != want {
				t.Errorf("MatchFill = %v, want FindHighlight at %#x over the content, %v", s.MatchFill, matchCoverage, want)
			}
			wantCurrent := tc.p.FindHighlight
			wantCurrent.A = currentMatchCoverage
			if want := vgcolor.Flatten(wantCurrent, rides); s.CurrentMatchFill != want {
				t.Errorf("CurrentMatchFill = %v, want FindHighlight at %#x over the content, %v", s.CurrentMatchFill, currentMatchCoverage, want)
			}
			if matchCoverage >= currentMatchCoverage {
				t.Errorf("the current match at %#x is not laid on more strongly than the rest at %#x",
					currentMatchCoverage, matchCoverage)
			}
			if s.MatchFill == s.CurrentMatchFill {
				t.Errorf("both match fills land on %v; the reader cannot tell the current one", s.MatchFill)
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

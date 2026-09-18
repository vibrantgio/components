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

// TestTheThumbCarriesTheRecordedCoverage reads the thumb off a capture in
// both schemes and requires the pixel to be the recorded ScrollbarThumb
// coverage over the fill the bar rides, and nothing else.
//
// MEASURED, reference/macos/textedit-scrollbar.png: the platform's overlay
// knob reads #9d9fa1 over a #1a2124 track, which is labelColor's white at
// 0.572 — 146 of 255 — reproduced within one 255th on every channel when it
// is flattened in encoded sRGB, the space the platform composites in. That
// coverage is carried in both schemes: the light row is the dark row's under
// labelColor's black, no stored capture holding a light-appearance overlay
// bar.
//
// What the bar then PAINTS is opaque: black at 0.572 over the light content's
// white is 109, and white at the same coverage over the dark content's 30 is
// 159. Both are read back here off the drawn image rather than off the style,
// because a style that resolves correctly and a bar that paints something
// else is the defect a colour assertion cannot see.
func TestTheThumbCarriesTheRecordedCoverage(t *testing.T) {
	const coverage = 0x92 // 146 of 255 — the recorded 0.572
	for _, tc := range []struct {
		name string
		p    tokens.PlatformColors
	}{
		{"light", tokens.PlatformLight},
		{"dark", tokens.PlatformDark},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.p.ScrollbarThumb.A; got != coverage {
				t.Errorf("ScrollbarThumb is laid on at %#x, want the recorded %#x", got, coverage)
			}
			rides := tc.p.ControlBackground
			size := image.Pt(24, 400)
			state := NewState()
			style := FromTokens(tc.p, rides)
			img := golden.Capture(t, size, func(gtx layout.Context) layout.Dimensions {
				gtx.Metric = unit.Metric{PxPerDp: 1, PxPerSp: 1}
				paint.FillShape(gtx.Ops, rides, clip.Rect{Max: gtx.Constraints.Max}.Op())
				style.Layout(gtx, state, layout.Vertical, 0.35, 0.65)
				return layout.Dimensions{Size: gtx.Constraints.Max}
			})
			// The thumb's own middle row and middle column. The bar takes
			// its gutter at the leading end of the minor axis it is given,
			// so the thumb stands one track padding in; the middle column of
			// it is clear of the corners the radius rounds away and of the
			// padding either side, so the pixel read is the fill and never
			// its antialiasing.
			got := img.RGBAAt(int(style.TrackPadding)+int(style.ThumbMinorWidth)/2, size.Y/2)
			want := vgcolor.Flatten(tc.p.ScrollbarThumb, rides)
			if got.R != want.R || got.G != want.G || got.B != want.B || got.A != 0xff {
				t.Errorf("the thumb paints %v, want the recorded coverage over the fill it rides, %v", got, want)
			}
		})
	}
}

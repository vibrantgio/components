package breadcrumb_test

import (
	"image"
	"image/color"
	"testing"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"

	"github.com/vibrantgio/components/breadcrumb"
	"github.com/vibrantgio/components/golden"
	"github.com/vibrantgio/theme/tokens"
)

const (
	frameW, frameH = 320, 32
)

var frameSize = image.Pt(frameW, frameH)

// defaultShaper returns the shaper every golden here draws with: the default
// typography's faces pinned, system fonts off, so the stored images are the
// same on every machine. A golden test pins its faces with
// DeterministicShaper; application code takes the fallback Shaper.
func defaultShaper(t *testing.T) *text.Shaper {
	t.Helper()
	return tokens.DefaultTypography.DeterministicShaper()
}

// scene renders w into a frame-sized constraint over a flat background.
func scene(w layout.Widget, bgColor color.NRGBA) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		paint.FillShape(gtx.Ops, bgColor, clip.Rect{Max: gtx.Constraints.Max}.Op())
		return w(gtx)
	}
}

// trail is the three-segment fixture: a real path, in document order, so the
// last segment is where you are and the two before it are the places back
// up. Titles are ASCII only, so Latin text in Roboto rasterises identically
// on every machine.
func trail() []breadcrumb.Item {
	return []breadcrumb.Item{{Title: "Home"}, {Title: "Design"}, {Title: "Tokens"}}
}

// TestBreadcrumbGolden records or diffs the three Measurable goldens. They
// are what shows the path bar's look: every place in the label colour,
// parted by the set's chevron in the secondary label colour. The single-segment golden is the
// structural assertion ("no chevrons when n == 1").
func TestBreadcrumbGolden(t *testing.T) {
	shaper := defaultShaper(t)
	lightBG := color.NRGBA{R: 240, G: 240, B: 240, A: 255}
	darkBG := color.NRGBA{R: 20, G: 20, B: 20, A: 255}

	threeItems := trail()
	singleItem := []breadcrumb.Item{{Title: "Home"}}

	cases := []struct {
		name     string
		items    []breadcrumb.Item
		platform tokens.PlatformColors
		bg       color.NRGBA
	}{
		{"light-three-segments", threeItems, tokens.PlatformLight, lightBG},
		{"dark-three-segments", threeItems, tokens.PlatformDark, darkBG},
		{"light-single-segment", singleItem, tokens.PlatformLight, lightBG},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			props := breadcrumb.Props{Items: tc.items, Shaper: shaper}
			w := breadcrumb.Render(shaper, props, tc.platform, tokens.Spacing, tokens.DefaultTypography.SystemFont)
			golden.Render(t, tc.name, frameSize, scene(w, tc.bg))
		})
	}
}

// TestBreadcrumbThreeVsSingle confirms a three-segment breadcrumb renders
// differently from a single-segment breadcrumb. Catches regressions where
// the chevron separators silently no-op.
func TestBreadcrumbThreeVsSingle(t *testing.T) {
	shaper := defaultShaper(t)
	bg := color.NRGBA{R: 240, G: 240, B: 240, A: 255}

	render := func(items []breadcrumb.Item) *image.RGBA {
		props := breadcrumb.Props{Items: items, Shaper: shaper}
		w := breadcrumb.Render(shaper, props, tokens.PlatformLight, tokens.Spacing, tokens.DefaultTypography.SystemFont)
		return golden.Capture(t, frameSize, scene(w, bg))
	}

	three := render(trail())
	single := render([]breadcrumb.Item{{Title: "Home"}})
	if n := golden.PixelDiff(three, single); n == 0 {
		t.Errorf("three-segment and single-segment render identically; expected chevrons in three-segment")
	}
}

// TestBreadcrumbLightDarkDiffer confirms swapping the colour token set
// changes the rendered output: the label and the secondary label are both
// a different colour between the two.
func TestBreadcrumbLightDarkDiffer(t *testing.T) {
	shaper := defaultShaper(t)
	bg := color.NRGBA{R: 128, G: 128, B: 128, A: 255}

	items := trail()
	propsL := breadcrumb.Props{Items: items, Shaper: shaper}
	propsD := breadcrumb.Props{Items: items, Shaper: shaper}
	light := breadcrumb.Render(shaper, propsL, tokens.PlatformLight, tokens.Spacing, tokens.DefaultTypography.SystemFont)
	dark := breadcrumb.Render(shaper, propsD, tokens.PlatformDark, tokens.Spacing, tokens.DefaultTypography.SystemFont)

	imgLight := golden.Capture(t, frameSize, scene(light, bg))
	imgDark := golden.Capture(t, frameSize, scene(dark, bg))
	if n := golden.PixelDiff(imgLight, imgDark); n == 0 {
		t.Errorf("light and dark render identically; expected chevron colour differences")
	}
}

// TestChevronSizeIsTheCallersOrTheDefault pins the separator's square and
// the room beside it: a caller that states a square gets it, a caller that
// states nothing gets the default, and the gap each side closes up by what
// the square itself leaves clear, so the figure keeps its eight either way.
//
// The row's width is the measure. What a separator costs the row is the
// square plus its two gaps — c + 2*(8 - c/3), the chevron covering a third
// of its square across — so the arithmetic is stated per square rather than
// derived from a width taken at another one. At 24 dp and up the square
// leaves the whole clearance by itself and the gaps are nothing.
//
// Both shapes are checked. They draw the same row, and a knob that reached
// only one of them would be a knob that works depending on which
// constructor you took.
func TestChevronSizeIsTheCallersOrTheDefault(t *testing.T) {
	shaper := defaultShaper(t)
	items := trail()
	segs := make([]breadcrumb.Segment, len(items))
	for i, it := range items {
		segs[i] = breadcrumb.Segment{Key: it.Title, Title: it.Title}
	}
	loose := layout.Constraints{Max: image.Pt(1<<14, 1<<14)}

	static := func(size unit.Dp) int {
		w := breadcrumb.Render(shaper, breadcrumb.Props{Items: items, Chevron: size},
			tokens.PlatformLight, tokens.Spacing, tokens.DefaultTypography.SystemFont)
		gtx := layout.Context{Metric: unit.Metric{PxPerDp: 1, PxPerSp: 1}, Constraints: loose, Ops: new(op.Ops)}
		return w(gtx).Size.X
	}
	perFrame := func(size unit.Dp) int {
		w := breadcrumb.NewTrail(shaper, breadcrumb.TrailProps{Chevron: size},
			tokens.PlatformLight, tokens.Spacing, tokens.DefaultTypography.SystemFont)
		gtx := layout.Context{Metric: unit.Metric{PxPerDp: 1, PxPerSp: 1}, Constraints: loose, Ops: new(op.Ops)}
		return w(gtx, segs).Size.X
	}

	// What one separator costs the row at 1x, square by square: the square
	// itself plus the two gaps, each 8 less the c/3 the square leaves clear
	// of the figure, rounded to the device as the row lays it out.
	blocks := map[unit.Dp]int{
		breadcrumb.DefaultChevron: 20 + 2*1, // 8 - 6.67 = 1.33 a side
		8:                         8 + 2*5,  // 8 - 2.67 = 5.33 a side
		24:                        24,       // the square leaves the whole 8
	}
	for _, tc := range []struct {
		name  string
		width func(unit.Dp) int
	}{
		{"Render", static},
		{"NewTrail", perFrame},
	} {
		t.Run(tc.name, func(t *testing.T) {
			deflt := tc.width(breadcrumb.DefaultChevron)
			if unstated := tc.width(0); unstated != deflt {
				t.Errorf("an unstated size drew %dpx wide, the default %dpx: zero must take the default",
					unstated, deflt)
			}
			if negative := tc.width(-3); negative != deflt {
				t.Errorf("a negative size drew %dpx wide, the default %dpx: it is no size at all, "+
					"not an invisible separator", negative, deflt)
			}
			// Two separators in a three-segment trail.
			for square, block := range blocks {
				want := deflt + 2*(block-blocks[breadcrumb.DefaultChevron])
				if got := tc.width(square); got != want {
					t.Errorf("at %ddp the row drew %dpx wide, want %dpx: a separator there costs "+
						"%dpx against the default's %dpx, twice over",
						int(square), got, want, block, blocks[breadcrumb.DefaultChevron])
				}
			}
		})
	}
}

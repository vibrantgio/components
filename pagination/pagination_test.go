package pagination_test

import (
	"context"
	"image"
	"image/color"
	"testing"

	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"

	"github.com/reactivego/rx"
	"github.com/vibrantgio/components/golden"
	"github.com/vibrantgio/components/pagination"
	vgcolor "github.com/vibrantgio/theme/color"
	"github.com/vibrantgio/theme/theme"
	"github.com/vibrantgio/theme/tokens"
)

const (
	frameW, frameH = 360, 48
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

// TestPaginationGolden records or diffs the three Measurable goldens.
//
// A zero radius scale (sharp corners) keeps the cell edges deterministic. The
// distinguishing signal across goldens is which cell carries the accent fill:
// page-1-of-5 fills the first cell, page-3-of-5 the third, and the light and
// dark variants swap the platform's two recorded sets.
func TestPaginationGolden(t *testing.T) {
	shaper := defaultShaper(t)
	lightBG := color.NRGBA{R: 240, G: 240, B: 240, A: 255}
	darkBG := color.NRGBA{R: 20, G: 20, B: 20, A: 255}
	sharpRadius := tokens.RadiusScale{}

	cases := []struct {
		name      string
		page      int
		pageCount int
		platform  tokens.PlatformColors
		bg        color.NRGBA
	}{
		{"light-page-1-of-5", 1, 5, tokens.PlatformLight, lightBG},
		{"light-page-3-of-5", 3, 5, tokens.PlatformLight, lightBG},
		{"dark-page-3-of-5", 3, 5, tokens.PlatformDark, darkBG},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			props := pagination.Props{Page: tc.page, PageCount: tc.pageCount, Shaper: shaper}
			w := pagination.Render(shaper, props, tc.platform, tokens.Spacing, sharpRadius, tokens.DefaultTypography.LabelLarge, tokens.Comfortable)
			golden.Render(t, tc.name, frameSize, scene(w, tc.bg))
		})
	}
}

// TestPaginationCurrentPagePositionDiffers confirms that moving the active
// page shifts the filled cell to a different x position. Guards against a
// regression in which all cells render with identical styling.
func TestPaginationCurrentPagePositionDiffers(t *testing.T) {
	shaper := defaultShaper(t)
	bg := color.NRGBA{R: 240, G: 240, B: 240, A: 255}
	sharpRadius := tokens.RadiusScale{}

	render := func(page int) *image.RGBA {
		props := pagination.Props{Page: page, PageCount: 5, Shaper: shaper}
		w := pagination.Render(shaper, props, tokens.PlatformLight, tokens.Spacing, sharpRadius, tokens.DefaultTypography.LabelLarge, tokens.Comfortable)
		return golden.Capture(t, frameSize, scene(w, bg))
	}

	one := render(1)
	three := render(3)
	if n := golden.PixelDiff(one, three); n == 0 {
		t.Error("page-1-of-5 and page-3-of-5 render identically; expected the filled cell to move")
	}
}

// TestPaginationLightDarkDiffer confirms swapping the colour token set
// changes the rendered output.
func TestPaginationLightDarkDiffer(t *testing.T) {
	shaper := defaultShaper(t)
	bg := color.NRGBA{R: 128, G: 128, B: 128, A: 255}
	sharpRadius := tokens.RadiusScale{}

	props := pagination.Props{Page: 3, PageCount: 5, Shaper: shaper}
	light := pagination.Render(shaper, props, tokens.PlatformLight, tokens.Spacing, sharpRadius, tokens.DefaultTypography.LabelLarge, tokens.Comfortable)
	dark := pagination.Render(shaper, props, tokens.PlatformDark, tokens.Spacing, sharpRadius, tokens.DefaultTypography.LabelLarge, tokens.Comfortable)

	imgLight := golden.Capture(t, frameSize, scene(light, bg))
	imgDark := golden.Capture(t, frameSize, scene(dark, bg))
	if n := golden.PixelDiff(imgLight, imgDark); n == 0 {
		t.Error("light and dark render identically; expected the two recorded sets to read apart")
	}
}

// TestTheCurrentPageIsTheAccentAndTheRestAreLabels pins the row's colours by
// name, in both recorded sets. The page the reader is on is the platform's
// accent with the text that reads on an accent fill, and no other page carries
// a fill at all, so the cells beside it are the digit alone over whatever the
// row stands on.
//
// The gates are ranges rather than single pixels because a digit is
// antialiased: every pixel of a cell lies between the colour behind it and the
// colour of the text drawn over it, and the platform's text carries a
// coverage, so the far end of that range is the composite and not the recorded
// value.
func TestTheCurrentPageIsTheAccentAndTheRestAreLabels(t *testing.T) {
	shaper := defaultShaper(t)
	sharpRadius := tokens.RadiusScale{}
	const page, pageCount = 3, 5

	// The cell's own geometry, derived rather than measured: the row is a
	// leading chevron, then one ControlHeight square per page, every pair
	// separated by an S2 gap, laid out middle-aligned in the frame.
	side := int(tokens.Comfortable.ControlHeight)
	gap := int(tokens.Spacing.S2)
	cellAt := func(n int) image.Rectangle {
		x := side + gap + (n-1)*(side+gap)
		y := (frameH - side) / 2
		// Inset by one pixel: the cell's own edge is antialiased against the
		// pixel beside it.
		return image.Rect(x+1, y+1, x+side-1, y+side-1)
	}

	for _, tc := range []struct {
		name string
		p    tokens.PlatformColors
		bg   color.NRGBA
	}{
		{"light", tokens.PlatformLight, color.NRGBA{R: 240, G: 240, B: 240, A: 255}},
		{"dark", tokens.PlatformDark, color.NRGBA{R: 20, G: 20, B: 20, A: 255}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			props := pagination.Props{Page: page, PageCount: pageCount, Shaper: shaper, Surface: tc.bg}
			w := pagination.Render(shaper, props, tc.p, tokens.Spacing, sharpRadius, tokens.DefaultTypography.LabelLarge, tokens.Comfortable)
			img := golden.Capture(t, frameSize, scene(w, tc.bg))

			current := cellAt(page)
			accent := tc.p.ControlAccent
			digit := vgcolor.Flatten(tc.p.AlternateSelectedControlText, accent)
			if out, moved := spanOf(img, current, accent, digit); !moved {
				t.Errorf("the current page cell is the accent %v and nothing else; no pixel carries its digit %v", accent, digit)
			} else if out != nil {
				t.Errorf("pixel %v of the current page cell is %v, which is neither the accent fill %v nor its digit %v nor a blend of the two",
					out.at, out.got, accent, digit)
			}

			resting := cellAt(page - 1)
			label := vgcolor.Flatten(tc.p.Label, tc.bg)
			if out, moved := spanOf(img, resting, tc.bg, label); !moved {
				t.Errorf("the resting page cell carries no digit; want the label %v over the surface %v", label, tc.bg)
			} else if out != nil {
				t.Errorf("pixel %v of the resting page cell is %v, which is neither the surface %v nor the label over it %v nor a blend of the two; a resting page carries no fill",
					out.at, out.got, tc.bg, label)
			}
		})
	}
}

// stray is one pixel that fell outside the span a cell's pixels must lie in.
type stray struct {
	at  image.Point
	got color.NRGBA
}

// spanOf walks r and reports the first pixel outside the per-channel span
// from a to b — the two colours a cell is drawn from, the surface and the text
// composited over it — and whether any pixel left a at all, which is how a
// drawn glyph is told from an empty cell.
func spanOf(img *image.RGBA, r image.Rectangle, a, b color.NRGBA) (*stray, bool) {
	moved := false
	within := func(v, lo, hi uint8) bool {
		if lo > hi {
			lo, hi = hi, lo
		}
		return v >= lo && v <= hi
	}
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			cr, cg, cb, _ := img.At(x, y).RGBA()
			got := color.NRGBA{R: uint8(cr >> 8), G: uint8(cg >> 8), B: uint8(cb >> 8), A: 0xff}
			if !within(got.R, a.R, b.R) || !within(got.G, a.G, b.G) || !within(got.B, a.B, b.B) {
				return &stray{at: image.Pt(x, y), got: got}, moved
			}
			if got != a {
				moved = true
			}
		}
	}
	return nil, moved
}

// liveWidget subscribes to obs and returns its last emitted layout.Widget.
func liveWidget(t *testing.T, obs rx.Observable[layout.Widget]) layout.Widget {
	t.Helper()
	var w layout.Widget
	if err := obs.Subscribe(context.Background(), func(next layout.Widget, _ error, done bool) {
		if !done && next != nil {
			w = next
		}
	}).Wait(); err != nil {
		t.Fatalf("Pagination subscribe: %v", err)
	}
	if w == nil {
		t.Fatal("Pagination did not emit an initial layout.Widget")
	}
	return w
}

// densityTheme returns a theme whose density is d, with sharp corners
// for golden determinism, mirroring components' density tests.
func densityTheme(d tokens.Density) theme.Theme {
	th := theme.Default()
	th.Density = rx.Of(d)
	th.Radius = rx.Of(tokens.RadiusScale{})
	return th
}

// TestPaginationCompactGolden records or diffs the compact-density golden
// through the LIVE pipeline (the static Render path is frozen at
// tokens.Comfortable): the page squares densify to the 28 dp
// ControlHeight and the chevron glyphs to the 16 dp icon size.
func TestPaginationCompactGolden(t *testing.T) {
	lightBG := color.NRGBA{R: 240, G: 240, B: 240, A: 255}
	props := pagination.Props{Page: 3, PageCount: 5, Shaper: defaultShaper(t)}
	w := liveWidget(t, pagination.Pagination(rx.Of(densityTheme(tokens.Compact)), props))
	golden.Render(t, "light-compact-page-3-of-5", frameSize, scene(w, lightBG))
}

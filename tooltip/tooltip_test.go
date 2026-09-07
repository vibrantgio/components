package tooltip_test

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

	"github.com/vibrantgio/components/golden"
	"github.com/vibrantgio/components/tooltip"
	"github.com/vibrantgio/theme/tokens"
)

const (
	frameW, frameH = 320, 240
)

var (
	frameSize = image.Pt(frameW, frameH)
	// Sharp corner radius. Anti-aliased rounded corners vary slightly
	// between GPU contexts, breaking determinism.
	sharpRadius = tokens.RadiusScale{}
)

// defaultShaper returns the shaper every golden here draws with: the default
// typography's faces pinned, system fonts off, so the stored images are the
// same on every machine. A golden test pins its faces with
// DeterministicShaper; application code takes the fallback Shaper.
func defaultShaper(t *testing.T) *text.Shaper {
	t.Helper()
	return tokens.DefaultTypography.DeterministicShaper()
}

// fixedRect is a sharp-edged solid layout.Widget with explicit width and height.
// Used as the Trigger stand-in so the hit rect is predictable and the
// goldens stay deterministic.
func fixedRect(c color.NRGBA, widthDp, heightDp float32) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		size := image.Pt(gtx.Dp(unit.Dp(widthDp)), gtx.Dp(unit.Dp(heightDp)))
		paint.FillShape(gtx.Ops, c, clip.Rect{Max: size}.Op())
		return layout.Dimensions{Size: size}
	}
}

// scene renders w over a flat background sized to the constraints.
func scene(w layout.Widget, bg color.NRGBA) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		paint.FillShape(gtx.Ops, bg, clip.Rect{Max: gtx.Constraints.Max}.Op())
		return w(gtx)
	}
}

// ---- Golden tests ----

// TestTooltipGolden records or diffs the two Measurable goldens —
// light-shown-top and dark-shown-bottom. The trigger is a small solid
// rectangle and the surface contains a short label rendered in
// OnInverseSurface against the InverseSurface-filled bubble. Text is part of
// the component's contract, so every case rasterises real glyphs and must
// pass Props.Shaper to keep the rendered face pinned and deterministic.
func TestTooltipGolden(t *testing.T) {
	shaper := defaultShaper(t)
	trigger := fixedRect(color.NRGBA{R: 80, G: 160, B: 220, A: 255}, 60, 28)

	lightBG := color.NRGBA{R: 240, G: 240, B: 240, A: 255}
	darkBG := color.NRGBA{R: 20, G: 20, B: 20, A: 255}

	cases := []struct {
		name      string
		placement tooltip.Placement
		colors    tokens.ColorTokens
		bg        color.NRGBA
	}{
		{"light-shown-top", tooltip.Top, tokens.DefaultLight, lightBG},
		{"dark-shown-bottom", tooltip.Bottom, tokens.DefaultDark, darkBG},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			props := tooltip.Props{
				Text:      "Save",
				Trigger:   trigger,
				Placement: tc.placement,
				Shaper:    shaper,
			}
			w := tooltip.Render(shaper, props, true, tc.colors, tokens.Spacing, sharpRadius, tokens.DefaultTypography.LabelSmall)
			golden.Render(t, tc.name, frameSize, scene(w, tc.bg))
		})
	}
}

// TestTooltipShownAndHiddenDiffer confirms that flipping the shown flag
// changes the rendered output. Catches regressions where the shown
// branch silently no-ops.
func TestTooltipShownAndHiddenDiffer(t *testing.T) {
	shaper := defaultShaper(t)
	trigger := fixedRect(color.NRGBA{R: 80, G: 160, B: 220, A: 255}, 60, 28)
	bg := color.NRGBA{R: 240, G: 240, B: 240, A: 255}
	props := tooltip.Props{Text: "Save", Trigger: trigger, Placement: tooltip.Top, Shaper: shaper}

	shown := tooltip.Render(shaper, props, true, tokens.DefaultLight, tokens.Spacing, sharpRadius, tokens.DefaultTypography.LabelSmall)
	hidden := tooltip.Render(shaper, props, false, tokens.DefaultLight, tokens.Spacing, sharpRadius, tokens.DefaultTypography.LabelSmall)

	imgShown := golden.Capture(t, frameSize, scene(shown, bg))
	imgHidden := golden.Capture(t, frameSize, scene(hidden, bg))
	if n := golden.PixelDiff(imgShown, imgHidden); n == 0 {
		t.Error("shown and hidden tooltip render identically; expected the bubble + label to appear when shown")
	}
}

// ---- Paint-order tests ----
//
// The defect these cover (feeds, 2026-09-06, on the popover): a floating
// surface drawn inline in its anchor's own paint order is painted over by
// every sibling the window lays out after that anchor's slot. The tooltip
// takes the same idiom, so it takes the same guard.

const (
	// stripH is the early slot's height: the room the tooltip is handed,
	// across the top of the scene, with the covering sibling filling
	// everything below it. The trigger is 60x28 centred in it, so a Bottom
	// annotation stands S1 below the trigger's foot at y=34 — inside the
	// covered band.
	stripH = 40
)

var coverColor = color.NRGBA{R: 255, G: 0, B: 255, A: 255}

// stripScene lays w out in a strip across the top — the early slot — and,
// when covered, paints an opaque sibling over everything below it, the way a
// shell paints its main column after the navigation bar's actions.
func stripScene(w layout.Widget, bg color.NRGBA, covered bool) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		paint.FillShape(gtx.Ops, bg, clip.Rect{Max: gtx.Constraints.Max}.Op())
		strip := gtx
		strip.Constraints = layout.Exact(image.Pt(gtx.Constraints.Max.X, stripH))
		w(strip)
		if covered {
			off := op.Offset(image.Pt(0, stripH)).Push(gtx.Ops)
			paint.FillShape(gtx.Ops, coverColor, clip.Rect{
				Max: image.Pt(gtx.Constraints.Max.X, gtx.Constraints.Max.Y-stripH),
			}.Op())
			off.Pop()
		}
		return layout.Dimensions{Size: gtx.Constraints.Max}
	}
}

// at reads one pixel as an opaque NRGBA, so captures compare against the
// colours they were painted with.
func at(img *image.RGBA, x, y int) color.NRGBA {
	r, g, b, _ := img.At(x, y).RGBA()
	return color.NRGBA{R: uint8(r >> 8), G: uint8(g >> 8), B: uint8(b >> 8), A: 255}
}

// fillBounds is the bounding box of every pixel drawn in c — the annotation's
// own rect, found rather than asserted, so the test carries no arithmetic the
// component could change under it.
func fillBounds(img *image.RGBA, c color.NRGBA) (image.Rectangle, bool) {
	b := image.Rectangle{Min: image.Pt(1<<30, 1<<30)}
	found := false
	for y := img.Bounds().Min.Y; y < img.Bounds().Max.Y; y++ {
		for x := img.Bounds().Min.X; x < img.Bounds().Max.X; x++ {
			if at(img, x, y) != c {
				continue
			}
			found = true
			b.Min.X = min(b.Min.X, x)
			b.Min.Y = min(b.Min.Y, y)
			b.Max.X = max(b.Max.X, x+1)
			b.Max.Y = max(b.Max.Y, y+1)
		}
	}
	return b, found
}

// TestAnnotationIsWholeOverALaterSibling is the paint-order contract: the
// annotation stands above the sibling laid out after the tooltip's slot, so
// the pixels inside it are the same ones it draws with nothing over it at
// all.
func TestAnnotationIsWholeOverALaterSibling(t *testing.T) {
	shaper := defaultShaper(t)
	colors := tokens.DefaultLight
	props := tooltip.Props{
		Text:      "Save",
		Trigger:   fixedRect(color.NRGBA{R: 80, G: 160, B: 220, A: 255}, 60, 28),
		Placement: tooltip.Bottom,
		Shaper:    shaper,
	}
	w := tooltip.Render(shaper, props, true, colors, tokens.Spacing, sharpRadius, tokens.DefaultTypography.LabelSmall)
	bg := color.NRGBA{R: 240, G: 240, B: 240, A: 255}

	bare := golden.Capture(t, frameSize, stripScene(w, bg, false))
	covered := golden.Capture(t, frameSize, stripScene(w, bg, true))

	box, ok := fillBounds(bare, colors.InverseSurface)
	if !ok {
		t.Fatalf("no pixel of the annotation's fill %v was drawn at all", colors.InverseSurface)
	}
	if box.Max.Y <= stripH {
		t.Fatalf("the annotation ends at y=%d, inside the strip; nothing covers it and the test proves nothing", box.Max.Y)
	}
	if got := at(covered, 8, frameH-8); got != coverColor {
		t.Fatalf("the covering sibling did not paint: (8,%d) is %v, want %v", frameH-8, got, coverColor)
	}
	for y := box.Min.Y; y < box.Max.Y; y++ {
		for x := box.Min.X; x < box.Max.X; x++ {
			if a, b := at(bare, x, y), at(covered, x, y); a != b {
				t.Fatalf("(%d,%d) inside the annotation is %v with a later sibling painted and %v without it", x, y, b, a)
			}
		}
	}
}

package tooltip_test

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"strings"
	"testing"

	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"

	"github.com/vibrantgio/components/attachment"
	"github.com/vibrantgio/components/golden"
	"github.com/vibrantgio/components/tooltip"
	vgcolor "github.com/vibrantgio/theme/color"
	"github.com/vibrantgio/theme/tokens"
	"github.com/vibrantgio/theme/typeset"
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
// rectangle and the surface contains short text in the platform's label
// colour on the window's own plane, with the help tag's edge, rim and
// shadow. Text is part
// of the component's contract, so every case rasterises real glyphs and must
// pass Props.Shaper to keep the rendered face pinned and deterministic.
func TestTooltipGolden(t *testing.T) {
	shaper := defaultShaper(t)
	trigger := fixedRect(color.NRGBA{R: 80, G: 160, B: 220, A: 255}, 60, 28)

	lightBG := color.NRGBA{R: 240, G: 240, B: 240, A: 255}
	darkBG := color.NRGBA{R: 20, G: 20, B: 20, A: 255}

	cases := []struct {
		name      string
		placement tooltip.Placement
		colors    tokens.PlatformColors
		bg        color.NRGBA
	}{
		{"light-shown-top", tooltip.Top, tokens.PlatformLight, lightBG},
		{"dark-shown-bottom", tooltip.Bottom, tokens.PlatformDark, darkBG},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			props := tooltip.Props{
				Text:      "Save",
				Trigger:   trigger,
				Placement: tc.placement,
				Shaper:    shaper,
			}
			w := tooltip.Render(shaper, props, true, tc.colors, tokens.Spacing, sharpRadius, tokens.DefaultTypography.SmallSystemFont)
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

	shown := tooltip.Render(shaper, props, true, tokens.PlatformLight, tokens.Spacing, sharpRadius, tokens.DefaultTypography.SmallSystemFont)
	hidden := tooltip.Render(shaper, props, false, tokens.PlatformLight, tokens.Spacing, sharpRadius, tokens.DefaultTypography.SmallSystemFont)

	imgShown := golden.Capture(t, frameSize, scene(shown, bg))
	imgHidden := golden.Capture(t, frameSize, scene(hidden, bg))
	if n := golden.PixelDiff(imgShown, imgHidden); n == 0 {
		t.Error("shown and hidden tooltip render identically; expected the bubble + text to appear when shown")
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
	colors := tokens.PlatformLight
	props := tooltip.Props{
		Text:      "Save",
		Trigger:   fixedRect(color.NRGBA{R: 80, G: 160, B: 220, A: 255}, 60, 28),
		Placement: tooltip.Bottom,
		Shaper:    shaper,
	}
	w := tooltip.Render(shaper, props, true, colors, tokens.Spacing, sharpRadius, tokens.DefaultTypography.SmallSystemFont)
	bg := color.NRGBA{R: 240, G: 240, B: 240, A: 255}

	bare := golden.Capture(t, frameSize, stripScene(w, bg, false))
	covered := golden.Capture(t, frameSize, stripScene(w, bg, true))

	box, ok := fillBounds(bare, colors.WindowBackground)
	if !ok {
		t.Fatalf("no pixel of the annotation's fill %v was drawn at all", colors.WindowBackground)
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

// surfacePixel reports whether the pixel at (x, y) is the annotation's own
// light fill: it differs from the scene with the annotation hidden, and it
// is the fill. The edge and the shadow around the surface are not its
// pixels, so the window can cut the shadow and leave the surface whole.
func surfacePixel(shown, hidden *image.RGBA, x, y int) bool {
	return shown.RGBAAt(x, y) != hidden.RGBAAt(x, y) && at(shown, x, y) == tokens.PlatformLight.WindowBackground
}

// annotationRows returns the rows of img the annotation's surface painted
// on, found by comparing it against the same scene with the annotation
// hidden: every row holding a pixel of the surface's fill.
func annotationRows(shown, hidden *image.RGBA) (top, bottom int) {
	b := shown.Bounds()
	top, bottom = -1, -1
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			if !surfacePixel(shown, hidden, x, y) {
				continue
			}
			if top < 0 {
				top = y
			}
			bottom = y
			break
		}
	}
	return top, bottom
}

// anchored renders a tooltip whose trigger is said to stand at box, with
// nothing drawn for it: the caller's own trigger is already on the page.
func anchored(shaper *text.Shaper, box image.Rectangle, placement tooltip.Placement, shown bool) layout.Widget {
	return tooltip.Render(
		shaper,
		tooltip.Props{
			Text:      "Where this goes",
			Placement: placement,
			Anchor:    func(layout.Context) image.Rectangle { return box },
			Shaper:    shaper,
		},
		shown,
		tokens.PlatformLight,
		tokens.Spacing,
		sharpRadius,
		tokens.DefaultTypography.SmallSystemFont,
	)
}

// TestAnchoredAnnotationStandsOnTheSideWithRoom places the same annotation
// against two anchors in one frame — one with the whole frame below it, one
// hard against the frame's foot — and reads back which side of each the
// annotation landed on. A tooltip asked for the side below takes the side
// above when the side below is too short to hold it, which is what keeps an
// annotation about a word near the foot of a column on the screen.
func TestAnchoredAnnotationStandsOnTheSideWithRoom(t *testing.T) {
	shaper := defaultShaper(t)
	bg := color.NRGBA{R: 245, G: 245, B: 245, A: 255}

	high := image.Rect(40, 20, 140, 36)
	rows := func(box image.Rectangle) (int, int) {
		shown := golden.Capture(t, frameSize, scene(anchored(shaper, box, tooltip.Bottom, true), bg))
		hidden := golden.Capture(t, frameSize, scene(anchored(shaper, box, tooltip.Bottom, false), bg))
		return annotationRows(shown, hidden)
	}

	top, bottom := rows(high)
	if top < 0 {
		t.Fatal("the annotation painted nothing at all")
	}
	if top < high.Max.Y {
		t.Errorf("with the whole frame below it the annotation starts at row %d, above the anchor's foot at %d", top, high.Max.Y)
	}

	// Hard against the frame's foot: the side below holds nothing.
	low := image.Rect(40, frameH-18, 140, frameH-2)
	top, bottom = rows(low)
	if top < 0 {
		t.Fatal("the annotation against the frame's foot painted nothing at all")
	}
	if bottom > low.Min.Y {
		t.Errorf("against the frame's foot the annotation reaches row %d, below the anchor's top at %d; it did not take the side with room", bottom, low.Min.Y)
	}
}

// annotationColumns reads back the leading and trailing column the
// annotation's surface covers, by the same test the rows are read by.
func annotationColumns(shown, hidden *image.RGBA) (int, int) {
	left, right := -1, -1
	b := shown.Bounds()
	for x := b.Min.X; x < b.Max.X; x++ {
		for y := b.Min.Y; y < b.Max.Y; y++ {
			if !surfacePixel(shown, hidden, x, y) {
				continue
			}
			if left < 0 {
				left = x
			}
			right = x
			break
		}
	}
	return left, right
}

// anchoredText renders an anchored annotation saying txt, for the cases that
// ask what a long line does to the surface's own width and place.
func anchoredText(shaper *text.Shaper, box image.Rectangle, txt string, shown bool) layout.Widget {
	return tooltip.Render(
		shaper,
		tooltip.Props{
			Text:      txt,
			Placement: tooltip.Bottom,
			Anchor:    func(layout.Context) image.Rectangle { return box },
			Shaper:    shaper,
		},
		shown,
		tokens.PlatformLight,
		tokens.Spacing,
		sharpRadius,
		tokens.DefaultTypography.SmallSystemFont,
	)
}

// TestTheAnnotationStaysInsideTheFramesSides anchors the same annotation in
// the middle of the frame and hard against each of its sides, and compares
// what was drawn: centred on a trigger near an edge the surface would run
// off it and be cut by the window, so the one that stands is as wide as the
// one in the middle and begins and ends inside the frame. A reader cannot
// read what is outside the window.
func TestTheAnnotationStaysInsideTheFramesSides(t *testing.T) {
	shaper := defaultShaper(t)
	bg := color.NRGBA{R: 245, G: 245, B: 245, A: 255}
	const txt = "https://example.test/a/longer/path"

	span := func(box image.Rectangle) (int, int) {
		shown := golden.Capture(t, frameSize, scene(anchoredText(shaper, box, txt, true), bg))
		hidden := golden.Capture(t, frameSize, scene(anchoredText(shaper, box, txt, false), bg))
		return annotationColumns(shown, hidden)
	}

	midLeft, midRight := span(image.Rect(frameW/2-6, 40, frameW/2+6, 56))
	if midLeft < 0 {
		t.Fatal("the annotation in the middle of the frame painted nothing at all")
	}
	want := midRight - midLeft

	sides := map[string]image.Rectangle{
		"leading":  image.Rect(0, 40, 12, 56),
		"trailing": image.Rect(frameW-12, 40, frameW, 56),
	}
	for name, box := range sides {
		t.Run(name, func(t *testing.T) {
			left, right := span(box)
			if left < 0 {
				t.Fatal("the annotation painted nothing at all")
			}
			if left < 1 || right > frameW-2 {
				t.Errorf("the annotation spans columns %d..%d, against the frame's own edges 0..%d", left, right, frameW-1)
			}
			if got := right - left; got != want {
				t.Errorf("the annotation is %d wide against this side and %d wide in the middle of the frame; it was cut by the window", got, want)
			}
		})
	}
}

// longText is a sentence far wider than MaxTextWidth on one line.
const longText = "Shows the readings the demo source makes up until you quit, and stops them when a real source is chosen in the settings"

// TestTextWiderThanTheTooltipIsSetOnMoreLines anchors an annotation whose
// text is far wider than MaxTextWidth in the middle of the frame and reads
// the surface back: it is no wider than MaxTextWidth and its padding, it is
// taller than one line, and two texts that differ only at their ends draw
// different annotations, so no part of the text was left out.
func TestTextWiderThanTheTooltipIsSetOnMoreLines(t *testing.T) {
	shaper := defaultShaper(t)
	bg := color.NRGBA{R: 245, G: 245, B: 245, A: 255}
	box := image.Rect(frameW/2-6, 20, frameW/2+6, 36)

	surface := func(txt string) (shown, hidden *image.RGBA) {
		return golden.Capture(t, frameSize, scene(anchoredText(shaper, box, txt, true), bg)),
			golden.Capture(t, frameSize, scene(anchoredText(shaper, box, txt, false), bg))
	}
	oneShown, oneHidden := surface("Save")
	oneTop, oneBottom := annotationRows(oneShown, oneHidden)
	shown, hidden := surface(longText)
	left, right := annotationColumns(shown, hidden)
	top, bottom := annotationRows(shown, hidden)
	if left < 0 || oneTop < 0 {
		t.Fatal("the annotation painted nothing at all")
	}

	// The platform's help tag sets its text up to 252 dp, measured off
	// finder-help-tag-view-{light,dark}.png in the macOS reference.
	if tooltip.MaxTextWidth != 252 {
		t.Errorf("MaxTextWidth is %v, not the help tag's measured 252", tooltip.MaxTextWidth)
	}
	// The surface is the text and the padding on both sides, S2 each.
	if got, limit := right-left+1, int(tooltip.MaxTextWidth)+2*int(tokens.Spacing.S2); got > limit {
		t.Errorf("the annotation is %d wide, wider than MaxTextWidth and its padding, %d", got, limit)
	}
	lineBox := int(tokens.DefaultTypography.SmallSystemFont.LineHeight)
	if got, one := bottom-top+1, oneBottom-oneTop+1; got < one+lineBox {
		t.Errorf("the annotation is %d tall against %d for one line; the text was not set on more lines", got, one)
	}
	if strings.Contains(longText, "…") {
		t.Fatal("the test text holds an ellipsis of its own")
	}

	other := strings.TrimSuffix(longText, "settings") + "preferences"
	another, _ := surface(other)
	if golden.PixelDiff(shown, another) == 0 {
		t.Error("two texts that differ only at their ends drew the same annotation; the end was left out")
	}
}

// TestTooltipLongTextGolden stores a tooltip whose text is set on more
// lines, for a reader to look at.
func TestTooltipLongTextGolden(t *testing.T) {
	shaper := defaultShaper(t)
	bg := color.NRGBA{R: 240, G: 240, B: 240, A: 255}
	box := image.Rect(frameW/2-6, 20, frameW/2+6, 36)
	golden.Render(t, "light-long-text-bottom", frameSize, scene(anchoredText(shaper, box, longText, true), bg))
}

// sentence is one line far wider than the small trigger the placement tests
// wrap.
const sentence = "Shows demo readings until you quit"

// wrapped renders a tooltip wrapped around a small trigger the way a caller
// wraps its own control: laid out at at, in the trigger's own size, and told
// the room around it by room, given in the tooltip's own coordinates. A nil
// room reports none.
func wrapped(shaper *text.Shaper, at image.Point, placement tooltip.Placement, room func(layout.Context) image.Rectangle, shown bool) layout.Widget {
	w := tooltip.Render(
		shaper,
		tooltip.Props{
			Text:          sentence,
			Trigger:       fixedRect(color.NRGBA{R: 80, G: 160, B: 220, A: 255}, 24, 16),
			Placement:     placement,
			AvailableRoom: room,
			Shaper:        shaper,
		},
		shown,
		tokens.PlatformLight,
		tokens.Spacing,
		sharpRadius,
		tokens.DefaultTypography.SmallSystemFont,
	)
	return func(gtx layout.Context) layout.Dimensions {
		off := op.Offset(at).Push(gtx.Ops)
		tgtx := gtx
		tgtx.Constraints = layout.Constraints{}
		w(tgtx)
		off.Pop()
		return layout.Dimensions{Size: gtx.Constraints.Max}
	}
}

// TestASentenceWiderThanTheTriggerIsDrawnWhole wraps a trigger 24 dp wide
// in a tooltip laid out in nothing but the trigger's own size, and reads the
// surface back: it is wider than the whole sentence on one line, so nothing
// was cut, whether the window's room is reported or not.
func TestASentenceWiderThanTheTriggerIsDrawnWhole(t *testing.T) {
	shaper := defaultShaper(t)
	bg := color.NRGBA{R: 245, G: 245, B: 245, A: 255}
	at := image.Pt(148, 100)
	whole := typeset.LineWidth(layout.Context{Metric: unit.Metric{PxPerDp: 1, PxPerSp: 1}, Ops: new(op.Ops)}, shaper, sentence, tokens.DefaultTypography.SmallSystemFont, font.Normal)

	rooms := map[string]func(layout.Context) image.Rectangle{
		"window reported": func(layout.Context) image.Rectangle {
			return image.Rectangle{Min: at.Mul(-1), Max: frameSize.Sub(at)}
		},
		"no room reported": nil,
	}
	for name, room := range rooms {
		t.Run(name, func(t *testing.T) {
			shown := golden.Capture(t, frameSize, scene(wrapped(shaper, at, tooltip.Bottom, room, true), bg))
			hidden := golden.Capture(t, frameSize, scene(wrapped(shaper, at, tooltip.Bottom, room, false), bg))
			left, right := annotationColumns(shown, hidden)
			if left < 0 {
				t.Fatal("the annotation painted nothing at all")
			}
			if got := right - left + 1; got <= whole {
				t.Errorf("the annotation is %d wide, no wider than the sentence's own %d on one line; it was cut", got, whole)
			}
			top, _ := annotationRows(shown, hidden)
			if top < at.Y+16 {
				t.Errorf("the annotation starts at row %d, above the trigger's foot at %d; it was asked for below", top, at.Y+16)
			}
		})
	}
}

// TestATooltipNearTheRoomsEdgeTakesTheSideWithRoom wraps a trigger whose
// reported room ends just under it and asks for the side below: the side
// below holds nothing, so the surface is drawn whole above the trigger.
func TestATooltipNearTheRoomsEdgeTakesTheSideWithRoom(t *testing.T) {
	shaper := defaultShaper(t)
	bg := color.NRGBA{R: 245, G: 245, B: 245, A: 255}
	at := image.Pt(148, 120)
	room := func(layout.Context) image.Rectangle {
		return image.Rect(-at.X, -at.Y, frameW-at.X, 16+2)
	}
	shown := golden.Capture(t, frameSize, scene(wrapped(shaper, at, tooltip.Bottom, room, true), bg))
	hidden := golden.Capture(t, frameSize, scene(wrapped(shaper, at, tooltip.Bottom, room, false), bg))
	top, bottom := annotationRows(shown, hidden)
	if top < 0 {
		t.Fatal("the annotation painted nothing at all")
	}
	if bottom >= at.Y {
		t.Errorf("near the foot of its room the annotation reaches row %d, at or below the trigger's top at %d; it did not take the side with room", bottom, at.Y)
	}
	whole := typeset.LineWidth(layout.Context{Metric: unit.Metric{PxPerDp: 1, PxPerSp: 1}, Ops: new(op.Ops)}, shaper, sentence, tokens.DefaultTypography.SmallSystemFont, font.Normal)
	if left, right := annotationColumns(shown, hidden); right-left+1 <= whole {
		t.Errorf("the flipped annotation is %d wide, no wider than the sentence's own %d; it was cut", right-left+1, whole)
	}
}

// placedAfterMeasuring lays a tooltip wrapped around a small trigger out the
// way a row that centres what it holds does: under a context with
// attachment.Surfaces set, recorded, measured, and only then placed against the
// frame's trailing edge at row y. place says whether the row then calls
// attachment.PlaceSurfaces with the frame moved by the offset it placed the
// tooltip at.
func placedAfterMeasuring(shaper *text.Shaper, y int, place, shown bool) layout.Widget {
	w := tooltip.Render(
		shaper,
		tooltip.Props{
			Text:      sentence,
			Trigger:   fixedRect(color.NRGBA{R: 80, G: 160, B: 220, A: 255}, 24, 16),
			Placement: tooltip.Bottom,
			Shaper:    shaper,
		},
		shown,
		tokens.PlatformLight,
		tokens.Spacing,
		sharpRadius,
		tokens.DefaultTypography.SmallSystemFont,
	)
	return func(gtx layout.Context) layout.Dimensions {
		cgtx, surfaces := attachment.WithSurfaces(gtx)
		cgtx.Constraints = layout.Constraints{}
		rec := op.Record(gtx.Ops)
		dims := w(cgtx)
		call := rec.Stop()
		at := image.Pt(gtx.Constraints.Max.X-dims.Size.X, y)
		off := op.Offset(at).Push(gtx.Ops)
		call.Add(gtx.Ops)
		if place {
			attachment.PlaceSurfaces(gtx, surfaces, image.Rectangle{Max: gtx.Constraints.Max}.Sub(at))
		}
		off.Pop()
		return layout.Dimensions{Size: gtx.Constraints.Max}
	}
}

// TestASurfaceRecordedInSurfacesIsPlacedInTheAreaItIsGiven lays a tooltip
// out under a context with attachment.Surfaces set and places it against the
// frame's trailing edge only after measuring it. Until the layout calls
// attachment.PlaceSurfaces nothing is drawn; once it does, with the frame moved
// by the tooltip's offset, the surface is shifted inward from the trailing
// edge and drawn whole.
func TestASurfaceRecordedInSurfacesIsPlacedInTheAreaItIsGiven(t *testing.T) {
	shaper := defaultShaper(t)
	bg := color.NRGBA{R: 245, G: 245, B: 245, A: 255}
	const y = 100
	whole := typeset.LineWidth(layout.Context{Metric: unit.Metric{PxPerDp: 1, PxPerSp: 1}, Ops: new(op.Ops)}, shaper, sentence, tokens.DefaultTypography.SmallSystemFont, font.Normal)

	hidden := golden.Capture(t, frameSize, scene(placedAfterMeasuring(shaper, y, true, false), bg))
	unplaced := golden.Capture(t, frameSize, scene(placedAfterMeasuring(shaper, y, false, true), bg))
	if n := golden.PixelDiff(unplaced, hidden); n != 0 {
		t.Errorf("a tooltip under attachment.Surfaces painted %d pixels before the layout placed its surface, want 0", n)
	}

	shown := golden.Capture(t, frameSize, scene(placedAfterMeasuring(shaper, y, true, true), bg))
	left, right := annotationColumns(shown, hidden)
	if left < 0 {
		t.Fatal("the placed annotation painted nothing at all")
	}
	if got := right - left + 1; got <= whole {
		t.Errorf("the annotation is %d wide, no wider than the sentence's own %d; it was cut at the frame's trailing edge", got, whole)
	}
	if right >= frameW-1 {
		t.Errorf("the annotation reaches column %d, the frame's trailing edge; it was not kept inside the area", right)
	}
	if top, _ := annotationRows(shown, hidden); top < y+16 {
		t.Errorf("the annotation starts at row %d, above the trigger's foot at %d; it was asked for below", top, y+16)
	}
}

// TestTheEdgeRimAndShadowAreTheHelpTags draws a tooltip of several lines in
// both schemes and reads its edge, rim and shadow back. Each pixel is the
// MEASURED coverage of the help tag in theme/tokens, composited in encoded
// sRGB on the surface the tooltip stands on, within 1/255. The tooltip is
// tall, so its sides at mid-height are the curve itself.
func TestTheEdgeRimAndShadowAreTheHelpTags(t *testing.T) {
	shaper := defaultShaper(t)
	box := image.Rect(frameW/2-6, 20, frameW/2+6, 36)
	cases := []struct {
		name     string
		colors   tokens.PlatformColors
		stands   color.NRGBA
		fitBelow float64 // the reference's fit of the shadow one pixel inside its rectangle
	}{
		{"light", tokens.PlatformLight, color.NRGBA{R: 240, G: 240, B: 240, A: 255}, 32.9},
		{"dark", tokens.PlatformDark, color.NRGBA{R: 100, G: 100, B: 100, A: 255}, 41.0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := tc.colors
			w := tooltip.Render(shaper, tooltip.Props{
				Text:      longText,
				Placement: tooltip.Bottom,
				Anchor:    func(layout.Context) image.Rectangle { return box },
				Shaper:    shaper,
				Surface:   tc.stands,
			}, true, p, tokens.Spacing, sharpRadius, tokens.DefaultTypography.SmallSystemFont)
			img := golden.Capture(t, frameSize, scene(w, tc.stands))

			fill := p.WindowBackground
			b, ok := fillBounds(img, fill)
			if !ok {
				t.Fatalf("no pixel of the fill %v was drawn", fill)
			}
			rims := 0
			for _, rim := range p.HelpTagEdge.Rim {
				if rim.A == 0 {
					break
				}
				rims++
			}
			// A rim that lands as the fill itself is inside the fill's
			// bounds already.
			grow := 0
			for i := range rims {
				if vgcolor.Flatten(p.HelpTagEdge.Rim[i], fill) != fill {
					grow = rims - i
					break
				}
			}
			b = b.Inset(-grow)

			a := p.HelpTagShadow.Alpha
			alpha := func(d int) float64 {
				if d < 0 || d >= len(a) {
					return 0
				}
				return float64(a[d].A)
			}
			corner := func(dx, dy float64) float64 { return dx * dy / (2 * alpha(0)) }
			// One pixel inside the shadow's rectangle, under the bottom
			// edge: 2·Alpha[0] − Alpha[1], within 1 of the reference's fit.
			below := 2*alpha(0) - alpha(1)
			if math.Abs(below-tc.fitBelow) > 1 {
				t.Errorf("the shadow under the bottom edge is %.1f, the reference's fit %.1f", below, tc.fitBelow)
			}
			off := int(p.HelpTagShadow.Offset)
			side := float64(p.HelpTagEdge.Side.A)
			cornerEdge := float64(p.HelpTagEdge.Corner.A)
			midX, midY := (b.Min.X+b.Max.X)/2, (b.Min.Y+b.Max.Y)/2
			x0, y0, x1, y1 := b.Min.X, b.Min.Y, b.Max.X, b.Max.Y

			type probe struct {
				what         string
				x, y         int
				shadow, edge float64
			}
			probes := []probe{
				{"top edge", midX, y0 - 1, alpha(off + 1), side},
				{"bottom edge", midX, y1, below, side},
				{"leading edge", x0 - 1, midY, alpha(1), side},
				{"trailing edge", x1, midY, alpha(1), side},
				{"top leading edge corner", x0 - 1, y0 - 1, corner(alpha(1), alpha(off+1)), cornerEdge},
				{"top trailing edge corner", x1, y0 - 1, corner(alpha(1), alpha(off+1)), cornerEdge},
				{"bottom leading edge corner", x0 - 1, y1, corner(alpha(1), below), cornerEdge},
				{"bottom trailing edge corner", x1, y1, corner(alpha(1), below), cornerEdge},
			}
			for d := range len(a) + 1 {
				probes = append(probes,
					probe{fmt.Sprintf("shadow %d below", d), midX, y1 - 1 + off + d, alpha(d), 0},
					probe{fmt.Sprintf("shadow %d out on the leading side", d), x0 - d, midY, alpha(d), 0},
					probe{fmt.Sprintf("shadow %d out on the trailing side", d), x1 - 1 + d, midY, alpha(d), 0},
				)
				if d >= off+2 {
					probes = append(probes, probe{fmt.Sprintf("shadow %d above", d), midX, y0 + off - d, alpha(d), 0})
				}
				if d >= 2 {
					probes = append(probes,
						probe{fmt.Sprintf("shadow corner %d,%d top leading", d, d), x0 - d, y0 + off - d, corner(alpha(d), alpha(d)), 0},
						probe{fmt.Sprintf("shadow corner %d,%d bottom trailing", d, d), x1 - 1 + d, y1 - 1 + off + d, corner(alpha(d), alpha(d)), 0},
					)
				}
			}
			for _, pr := range probes {
				if pr.edge == 0 && (pr.x >= x0-1 && pr.x <= x1 && pr.y >= y0-1 && pr.y <= y1) {
					continue // inside the edge's ring or the bounds
				}
				keep := (1 - pr.shadow/255) * (1 - pr.edge/255)
				want := float64(tc.stands.G) * keep
				got := at(img, pr.x, pr.y)
				for _, ch := range []uint8{got.R, got.G, got.B} {
					if math.Abs(float64(ch)-want) > 1 {
						t.Errorf("%s at (%d,%d) is %v, want %.1f: black at %.1f/255 of shadow and %.0f/255 of edge on %v",
							pr.what, pr.x, pr.y, got, want, pr.shadow, pr.edge, tc.stands)
						break
					}
				}
			}

			for i := range rims {
				want := vgcolor.Flatten(p.HelpTagEdge.Rim[i], fill)
				for _, pt := range []image.Point{{midX, y0 + i}, {x0 + i, midY}, {x1 - 1 - i, midY}, {midX, y1 - 1 - i}} {
					if got := at(img, pt.X, pt.Y); !near(got, want) {
						t.Errorf("rim pixel %d at %v is %v, want %v", i, pt, got, want)
					}
				}
			}
			if got := at(img, x0+rims, midY); !near(got, fill) {
				t.Errorf("the first pixel inside the rim at (%d,%d) is %v, want the fill %v", x0+rims, midY, got, fill)
			}
		})
	}
}

// near reports whether a and b differ by at most 1 in each channel.
func near(a, b color.NRGBA) bool {
	d := func(x, y uint8) bool { return x-y <= 1 || y-x <= 1 }
	return d(a.R, b.R) && d(a.G, b.G) && d(a.B, b.B)
}

package tooltip_test

import (
	"image"
	"image/color"
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
// colour on the window's own plane, inside a separator hairline. Text is part
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

// annotationRows returns the rows of img the annotation painted on, found by
// comparing it against the same scene with the annotation hidden: every row
// holding a pixel the two images differ at.
func annotationRows(shown, hidden *image.RGBA) (top, bottom int) {
	b := shown.Bounds()
	top, bottom = -1, -1
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			if shown.RGBAAt(x, y) == hidden.RGBAAt(x, y) {
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
// annotation covers, by the same difference the rows are read by.
func annotationColumns(shown, hidden *image.RGBA) (int, int) {
	left, right := -1, -1
	b := shown.Bounds()
	for x := b.Min.X; x < b.Max.X; x++ {
		for y := b.Min.Y; y < b.Max.Y; y++ {
			if shown.RGBAAt(x, y) == hidden.RGBAAt(x, y) {
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

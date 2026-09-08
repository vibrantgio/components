package paragraph_test

import (
	"image"
	"image/color"
	"testing"

	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"

	golden "github.com/vibrantgio/components/golden"
	"github.com/vibrantgio/components/paragraph"
	"github.com/vibrantgio/theme/tokens"
)

// fillColor is the field the tests below paint behind a run of glyphs: a
// colour no face and no page in these captures carries, so a pixel wearing it
// was painted by the fill and by nothing else.
var fillColor = color.NRGBA{R: 0x00, G: 0x80, B: 0xff, A: 0xff}

// countFill returns how many pixels of img wear the fill colour, and the box
// around them.
func countFill(img *image.RGBA) (int, image.Rectangle) {
	n := 0
	var box image.Rectangle
	b := img.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			r, g, bl, a := img.At(x, y).RGBA()
			got := color.NRGBA{R: uint8(r >> 8), G: uint8(g >> 8), B: uint8(bl >> 8), A: uint8(a >> 8)}
			if got != fillColor {
				continue
			}
			n++
			p := image.Rect(x, y, x+1, y+1)
			if box.Empty() {
				box = p
			} else {
				box = box.Union(p)
			}
		}
	}
	return n, box
}

// captureFill lays the spans out on the light page and returns the capture
// together with every rectangle the layout reported a fill at.
func captureFill(t *testing.T, spans []paragraph.SpanStyle, width int) (*image.RGBA, []image.Rectangle) {
	t.Helper()
	shaper := defaultShaper(t)
	colors := tokens.DefaultLight
	style := paragraph.FromTokens(colors, tokens.DefaultTypography.BodyLarge)
	var got []image.Rectangle
	style.OnFill = func(span, fill int, r image.Rectangle) { got = append(got, r) }
	img := golden.Capture(t, image.Pt(width, 90), func(gtx layout.Context) layout.Dimensions {
		paint.FillShape(gtx.Ops, colors.Background, clip.Rect{Max: gtx.Constraints.Max}.Op())
		return paragraph.Render(shaper, style, spans, paragraph.Idle())(gtx)
	})
	return img, got
}

// TestSpanFillPaintsBehindItsOwnRun is the fill's contract: the field lands
// where the range says and nowhere else — the words on either side of it keep
// the page.
func TestSpanFillPaintsBehindItsOwnRun(t *testing.T) {
	const content = "The quick brown fox"
	start, end := len("The "), len("The quick")
	spans := []paragraph.SpanStyle{{
		Content: content,
		Fills:   []paragraph.Fill{{Start: start, End: end, Color: fillColor}},
	}}
	img, rects := captureFill(t, spans, 300)
	if len(rects) != 1 {
		t.Fatalf("the layout reported %d fills, want one", len(rects))
	}
	n, box := countFill(img)
	if n == 0 {
		t.Fatal("no pixel wears the fill; the field was not painted")
	}
	if box != rects[0] {
		t.Errorf("the painted field covers %v, but the layout reported %v", box, rects[0])
	}
	if box.Min.X == 0 {
		t.Errorf("the field starts at the paragraph's own left edge (%v); the words before the range should be clear of it", box)
	}
	plain, _ := captureFill(t, []paragraph.SpanStyle{{Content: content}}, 300)
	if unmarked, _ := countFill(plain); unmarked != 0 {
		t.Fatalf("the control capture already wears the fill colour in %d pixels; the test cannot tell the field from the text", unmarked)
	}
}

// TestSpanFillLeavesTheShapingAlone is why the fill is a byte range rather
// than a span of its own: a marked paragraph wraps exactly where the same
// paragraph unmarked does, so marking a word moves no glyph.
func TestSpanFillLeavesTheShapingAlone(t *testing.T) {
	shaper := defaultShaper(t)
	style := paragraph.FromTokens(tokens.DefaultLight, tokens.DefaultTypography.BodyLarge)
	const content = "The quick brown fox jumps over the lazy dog and keeps running"
	measure := func(spans []paragraph.SpanStyle) layout.Dimensions {
		var dims layout.Dimensions
		golden.Capture(t, image.Pt(160, 200), func(gtx layout.Context) layout.Dimensions {
			dims = paragraph.Render(shaper, style, spans, paragraph.Idle())(gtx)
			return dims
		})
		return dims
	}
	plain := measure([]paragraph.SpanStyle{{Content: content}})
	for _, tc := range []struct {
		name       string
		start, end int
	}{
		{"a whole word", len("The "), len("The quick")},
		{"the middle of a word", len("The qu"), len("The quic")},
		{"across the wrap", 0, len(content)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			marked := measure([]paragraph.SpanStyle{{
				Content: content,
				Fills:   []paragraph.Fill{{Start: tc.start, End: tc.end, Color: fillColor}},
			}})
			if marked.Size != plain.Size {
				t.Errorf("the marked paragraph measures %v where the plain one measures %v; the fill moved the wrapping", marked.Size, plain.Size)
			}
		})
	}
}

// TestSpanFillWrapsWithItsRun pins what a fill does when the run it covers
// runs onto the next line: it is painted on every line the run reaches, one
// field per line, and each is reported.
func TestSpanFillWrapsWithItsRun(t *testing.T) {
	const content = "alpha beta gamma delta epsilon zeta eta theta"
	spans := []paragraph.SpanStyle{{
		Content: content,
		Fills:   []paragraph.Fill{{Start: 0, End: len(content), Color: fillColor}},
	}}
	_, rects := captureFill(t, spans, 120)
	if len(rects) < 2 {
		t.Fatalf("a run covering %d lines reported %d fields, want one per line", 2, len(rects))
	}
	for i := 1; i < len(rects); i++ {
		if rects[i].Min.Y < rects[i-1].Max.Y {
			t.Errorf("field %d at %v overlaps the line above it at %v", i, rects[i], rects[i-1])
		}
	}
}

// TestSpanFillKeepsTheChipUnderIt is the order the two are drawn in: a fill
// on a chipped span goes down first, so the chip is still there.
func TestSpanFillKeepsTheChipUnderIt(t *testing.T) {
	chip := paragraph.Chip{Color: color.NRGBA{R: 0xff, G: 0x00, B: 0xff, A: 0xff}, Padding: 4, Radius: 4}
	const content = "code"
	chipped := []paragraph.SpanStyle{{Content: "a ", Fills: nil}, {Content: content, Chip: chip}}
	marked := []paragraph.SpanStyle{{Content: "a "}, {
		Content: content,
		Chip:    chip,
		Fills:   []paragraph.Fill{{Start: 0, End: len(content), Color: color.NRGBA{R: 0, G: 0x80, B: 0xff, A: 0x80}}},
	}}
	plain, _ := captureFill(t, chipped, 200)
	over, _ := captureFill(t, marked, 200)
	if golden.PixelDiff(plain, over) == 0 {
		t.Fatal("marking the chipped span changed nothing")
	}
	count := func(img *image.RGBA) int {
		n := 0
		b := img.Bounds()
		for y := b.Min.Y; y < b.Max.Y; y++ {
			for x := b.Min.X; x < b.Max.X; x++ {
				r, g, bl, _ := img.At(x, y).RGBA()
				if uint8(r>>8) > 0x80 && uint8(g>>8) < 0x60 && uint8(bl>>8) > 0x80 {
					n++
				}
			}
		}
		return n
	}
	if count(over) == 0 {
		t.Error("the chip is gone from under the field; the fill was painted over it, not under it")
	}
}

// TestColourlessFillPaintsNothingAndIsReported holds the probe half of the
// fill's contract: a fill with no colour in it moves no pixel and is still
// reported where it landed, which is how a caller asks where a run of its own
// text is without marking it.
func TestColourlessFillPaintsNothingAndIsReported(t *testing.T) {
	const content = "alpha beta gamma delta"
	plain, none := captureFill(t, []paragraph.SpanStyle{{Content: content}}, 300)
	if len(none) != 0 {
		t.Fatalf("a span with no fills reported %d places", len(none))
	}
	probed, places := captureFill(t, []paragraph.SpanStyle{{
		Content: content,
		Fills:   []paragraph.Fill{{Start: 6, End: 10}},
	}}, 300)
	if len(places) != 1 {
		t.Fatalf("the colourless fill was reported %d times, want once", len(places))
	}
	if places[0].Empty() {
		t.Error("the colourless fill was reported at an empty rectangle")
	}
	if n := golden.PixelDiff(plain, probed); n != 0 {
		t.Errorf("the colourless fill moved %d pixels; it must paint nothing", n)
	}
}

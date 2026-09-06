package richtext_test

import (
	"image"
	"image/color"
	"testing"

	"gioui.org/f32"
	"gioui.org/font"
	gioinput "gioui.org/io/input"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"

	"golang.org/x/image/math/fixed"

	golden "github.com/vibrantgio/components/golden"
	"github.com/vibrantgio/components/internal/focus"
	"github.com/vibrantgio/components/richtext"
	"github.com/vibrantgio/font/notocoloremoji"
	tcolor "github.com/vibrantgio/theme/color"
	"github.com/vibrantgio/theme/tokens"
)

// defaultShaper returns the shaper every golden here draws with: the default
// typography's faces pinned, system fonts off, so the stored images are the
// same on every machine. A golden test pins its faces with
// DeterministicShaper; application code takes the fallback Shaper.
func defaultShaper(t *testing.T) *text.Shaper {
	t.Helper()
	return tokens.DefaultTypography.DeterministicShaper()
}

// mixedSpans is the canonical test paragraph: regular, bold, italic, and
// monospace spans plus one hyperlink (link index 0).
func mixedSpans() []richtext.SpanStyle {
	return []richtext.SpanStyle{
		{Content: "The quick "},
		{Content: "brown", Weight: font.Bold},
		{Content: " fox "},
		{Content: "jumps", Style: font.Italic},
		{Content: " over "},
		{Content: "the lazy dog", Typeface: "Go Mono, monospace"},
		{Content: " via "},
		{Content: "a link", URL: "https://gioui.org"},
		{Content: " home."},
	}
}

// ---- Golden-image tests ----

// TestParagraphGolden records or diffs the mixed-style paragraph (regular,
// bold, italic, mono, link) wrapped over multiple lines, in light and dark
// token themes.
func TestParagraphGolden(t *testing.T) {
	shaper := defaultShaper(t)
	size := image.Pt(300, 100)
	cases := []struct {
		name   string
		colors tokens.ColorTokens
	}{
		{"paragraph-light", tokens.DefaultLight},
		{"paragraph-dark", tokens.DefaultDark},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			style := richtext.FromTokens(tc.colors, tokens.DefaultTypography.BodyLarge)
			w := richtext.Render(shaper, style, mixedSpans(), richtext.Idle())
			golden.Render(t, tc.name, size, w)
		})
	}
}

// TestLinkStateGolden records or diffs the link interaction treatments: the
// paragraph's only link hovered (blended colour) and focused (focus ring).
func TestLinkStateGolden(t *testing.T) {
	shaper := defaultShaper(t)
	size := image.Pt(300, 100)
	style := richtext.FromTokens(tokens.DefaultLight, tokens.DefaultTypography.BodyLarge)
	cases := []struct {
		name  string
		state richtext.RenderState
	}{
		{"link-hovered", richtext.RenderState{HoveredLink: 0, FocusedLink: richtext.NoLink}},
		{"link-focused", richtext.RenderState{HoveredLink: richtext.NoLink, FocusedLink: 0}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := richtext.Render(shaper, style, mixedSpans(), tc.state)
			golden.Render(t, tc.name, size, w)
		})
	}
}

// TestHoveredLinkIsVisuallyDistinct confirms the hovered link treatment
// produces different pixels from the idle paragraph.
func TestHoveredLinkIsVisuallyDistinct(t *testing.T) {
	shaper := defaultShaper(t)
	size := image.Pt(300, 100)
	style := richtext.FromTokens(tokens.DefaultLight, tokens.DefaultTypography.BodyLarge)

	idle := golden.Capture(t, size, richtext.Render(shaper, style, mixedSpans(), richtext.Idle()))
	hovered := golden.Capture(t, size, richtext.Render(shaper, style, mixedSpans(),
		richtext.RenderState{HoveredLink: 0, FocusedLink: richtext.NoLink}))
	if idle == nil || hovered == nil {
		return // headless unavailable; Capture called t.Skip
	}
	if n := golden.PixelDiff(idle, hovered); n == 0 {
		t.Error("hovered and idle paragraphs render identically; expected hover treatment pixels to differ")
	}
}

// TestFocusedLinkIsVisuallyDistinct confirms the focused link renders a
// visible focus ring: different pixels from the idle paragraph.
func TestFocusedLinkIsVisuallyDistinct(t *testing.T) {
	shaper := defaultShaper(t)
	size := image.Pt(300, 100)
	style := richtext.FromTokens(tokens.DefaultLight, tokens.DefaultTypography.BodyLarge)

	idle := golden.Capture(t, size, richtext.Render(shaper, style, mixedSpans(), richtext.Idle()))
	focused := golden.Capture(t, size, richtext.Render(shaper, style, mixedSpans(),
		richtext.RenderState{HoveredLink: richtext.NoLink, FocusedLink: 0}))
	if idle == nil || focused == nil {
		return
	}
	if n := golden.PixelDiff(idle, focused); n == 0 {
		t.Error("focused and idle paragraphs render identically; expected focus ring pixels to differ")
	}
}

// TestStrikethroughIsVisuallyDistinct confirms a strikethrough span renders a
// visible line through the text: different pixels from the same span without
// the decoration.
func TestStrikethroughIsVisuallyDistinct(t *testing.T) {
	shaper := defaultShaper(t)
	size := image.Pt(300, 100)
	style := richtext.FromTokens(tokens.DefaultLight, tokens.DefaultTypography.BodyLarge)

	plain := golden.Capture(t, size, richtext.Render(shaper, style,
		[]richtext.SpanStyle{{Content: "deleted text"}}, richtext.Idle()))
	struck := golden.Capture(t, size, richtext.Render(shaper, style,
		[]richtext.SpanStyle{{Content: "deleted text", Strikethrough: true}}, richtext.Idle()))
	if plain == nil || struck == nil {
		return // headless unavailable; Capture called t.Skip
	}
	if n := golden.PixelDiff(plain, struck); n == 0 {
		t.Error("strikethrough and plain spans render identically; expected line-through pixels to differ")
	}
}

// TestChipIsVisuallyDistinct confirms a chipped span paints its fill:
// different pixels from the same span without one.
func TestChipIsVisuallyDistinct(t *testing.T) {
	shaper := defaultShaper(t)
	size := image.Pt(300, 100)
	colors := tokens.DefaultLight
	style := richtext.FromTokens(colors, tokens.DefaultTypography.BodyLarge)
	chip := richtext.Chip{Color: colors.Ramps.Neutral.Step(200), Padding: 4, Radius: 4}

	plain := golden.Capture(t, size, richtext.Render(shaper, style,
		[]richtext.SpanStyle{{Content: "quoted"}}, richtext.Idle()))
	chipped := golden.Capture(t, size, richtext.Render(shaper, style,
		[]richtext.SpanStyle{{Content: "quoted", Chip: chip}}, richtext.Idle()))
	if plain == nil || chipped == nil {
		return // headless unavailable; Capture called t.Skip
	}
	if n := golden.PixelDiff(plain, chipped); n == 0 {
		t.Error("chipped and plain spans render identically; expected the chip's fill to differ")
	}
}

// ---- Layout tests ----

func measure(shaper *text.Shaper, style richtext.Style, spans []richtext.SpanStyle, maxWidth int) layout.Dimensions {
	var ops op.Ops
	gtx := layout.Context{
		Metric:      unit.Metric{PxPerDp: 1, PxPerSp: 1},
		Constraints: layout.Constraints{Max: image.Pt(maxWidth, 10_000)},
		Ops:         &ops,
	}
	return richtext.Render(shaper, style, spans, richtext.Idle())(gtx)
}

// TestParagraphWraps verifies that narrowing the constraint wraps the spans
// into more lines: the narrow layout must be taller than the wide one, and
// both must respect their max width.
func TestParagraphWraps(t *testing.T) {
	shaper := defaultShaper(t)
	style := richtext.FromTokens(tokens.DefaultLight, tokens.DefaultTypography.BodyLarge)

	wide := measure(shaper, style, mixedSpans(), 600)
	narrow := measure(shaper, style, mixedSpans(), 150)

	if wide.Size.X > 600 || narrow.Size.X > 150 {
		t.Errorf("layout exceeds max width: wide %v (max 600), narrow %v (max 150)", wide.Size, narrow.Size)
	}
	if narrow.Size.Y <= wide.Size.Y {
		t.Errorf("narrow layout height %d not greater than wide %d; spans did not wrap", narrow.Size.Y, wide.Size.Y)
	}
	if wide.Size.Y == 0 || wide.Size.X == 0 {
		t.Errorf("wide layout has empty size %v", wide.Size)
	}
}

// chipPad is the padding every chip test here sets, in dp; the test contexts
// measure at 1 px per dp, so it is also the pixel count expected in a width.
const chipPad = 4

// testChip is the fill the chip tests set their span on.
func testChip() richtext.Chip {
	return richtext.Chip{
		Color:   tokens.DefaultLight.Ramps.Neutral.Step(200),
		Padding: chipPad,
		Radius:  4,
	}
}

// TestChipReservesItsPaddingAndNotTheLine holds a chip to both halves of its
// contract at once. Its padding is real estate: the line grows by it, so the
// words on either side clear the fill rather than running under it. Its fill
// is not: the line stays exactly as tall as the span's own shaped box, so a
// caller can set a word on a chip without anything hung beside the line —
// a list marker, a gutter rule — moving.
func TestChipReservesItsPaddingAndNotTheLine(t *testing.T) {
	shaper := defaultShaper(t)
	style := richtext.FromTokens(tokens.DefaultLight, tokens.DefaultTypography.BodyLarge)

	// Mid-line, with a word on either side to clear: the case the reservation
	// exists for, and the one both flush edges leave alone.
	plain := measure(shaper, style, []richtext.SpanStyle{
		{Content: "call "}, {Content: "quoted"}, {Content: " now"},
	}, 600)
	chipped := measure(shaper, style, []richtext.SpanStyle{
		{Content: "call "}, {Content: "quoted", Chip: testChip()}, {Content: " now"},
	}, 600)

	if want := plain.Size.X + 2*chipPad; chipped.Size.X != want {
		t.Errorf("a chipped span measures %d px wide, want %d (a plain %d plus %d dp of padding on each side)",
			chipped.Size.X, want, plain.Size.X, chipPad)
	}
	if chipped.Size.Y != plain.Size.Y {
		t.Errorf("a chipped span's line measures %d px tall against a plain one's %d; a chip takes the span's own shaped height and cannot stretch the line",
			chipped.Size.Y, plain.Size.Y)
	}
}

// TestChipSpendsNoPaddingAtAFlushEdge pins the two edges where the reservation
// would be the defect: the start of a line, where the padding would set the
// chip's glyphs clear of a margin rather than of a word (and stair-step a list
// whose items open with a chip), and the gap before closing punctuation, where
// it would read as a space nobody typed. What the chip does not spend, the
// line must not count — so each case is measured against the same paragraph
// set without the chip, and the difference is exactly the padding still spent.
func TestChipSpendsNoPaddingAtAFlushEdge(t *testing.T) {
	shaper := defaultShaper(t)
	style := richtext.FromTokens(tokens.DefaultLight, tokens.DefaultTypography.BodyLarge)

	for _, tc := range []struct {
		name  string
		spans []richtext.SpanStyle
		// want is the width the chipped paragraph must exceed the same
		// paragraph unchipped by: the padding the chip still spends.
		want int
	}{
		{
			name:  "opening a line spends only the trailing padding",
			spans: []richtext.SpanStyle{{Content: "quoted"}, {Content: " now"}},
			want:  chipPad,
		},
		{
			name:  "before closing punctuation spends only the leading padding",
			spans: []richtext.SpanStyle{{Content: "call "}, {Content: "quoted"}, {Content: "; now"}},
			want:  chipPad,
		},
		{
			name:  "opening a line and closed by punctuation spends neither",
			spans: []richtext.SpanStyle{{Content: "quoted"}, {Content: ". Now"}},
			want:  0,
		},
		{
			name:  "an opening bracket after a chip is not closing punctuation",
			spans: []richtext.SpanStyle{{Content: "call "}, {Content: "quoted"}, {Content: "(now)"}},
			want:  2 * chipPad,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// The chipped span is the one named "quoted" in each case.
			chipped := make([]richtext.SpanStyle, len(tc.spans))
			copy(chipped, tc.spans)
			for i := range chipped {
				if chipped[i].Content == "quoted" {
					chipped[i].Chip = testChip()
				}
			}
			plain := measure(shaper, style, tc.spans, 600)
			got := measure(shaper, style, chipped, 600)
			if want := plain.Size.X + tc.want; got.Size.X != want {
				t.Errorf("chipped paragraph measures %d px wide, want %d (a plain %d plus %d px of padding spent)",
					got.Size.X, want, plain.Size.X, tc.want)
			}
		})
	}
}

// TestLineInitialChipStartsFlushWithTheMargin is the pixel half of the
// line-start rule: a width can be spent anywhere, so this checks where the
// glyphs actually land. A chipped word opening a line must draw its first dark
// pixel in the same column as the same word unchipped — that column is the
// list's left edge, and a chip that pushed its glyphs a padding right of it is
// exactly the stair-step this rule removes.
func TestLineInitialChipStartsFlushWithTheMargin(t *testing.T) {
	shaper := defaultShaper(t)
	size := image.Pt(300, 60)
	style := richtext.FromTokens(tokens.DefaultLight, tokens.DefaultTypography.BodyLarge)

	// On the theme's own surface: the capture is transparent where nothing is
	// painted, and a threshold on darkness cannot read glyphs against that.
	onBackground := func(spans []richtext.SpanStyle) layout.Widget {
		return func(gtx layout.Context) layout.Dimensions {
			paint.FillShape(gtx.Ops, tokens.DefaultLight.Background,
				clip.Rect{Max: gtx.Constraints.Max}.Op())
			return richtext.Render(shaper, style, spans, richtext.Idle())(gtx)
		}
	}
	plain := golden.Capture(t, size, onBackground(
		[]richtext.SpanStyle{{Content: "quoted word"}}))
	chipped := golden.Capture(t, size, onBackground(
		[]richtext.SpanStyle{{Content: "quoted", Chip: testChip()}, {Content: " word"}}))
	if plain == nil || chipped == nil {
		return // headless unavailable; Capture called t.Skip
	}
	// The chip's fill is a light neutral and the glyphs are near-black, so a
	// midpoint threshold separates glyph from fill.
	if got, want := firstDarkColumn(chipped, 128), firstDarkColumn(plain, 128); got != want {
		t.Errorf("a line-initial chipped word draws from column %d, a plain one from %d; the chip's glyphs must start at the margin, not a padding right of it",
			got, want)
	}
}

// firstDarkColumn returns the leftmost column holding a pixel darker than the
// given luminance, or -1 when the image has none.
func firstDarkColumn(img *image.RGBA, lum float64) int {
	b := img.Bounds()
	for x := b.Min.X; x < b.Max.X; x++ {
		for y := b.Min.Y; y < b.Max.Y; y++ {
			c := img.RGBAAt(x, y)
			if 0.2126*float64(c.R)+0.7152*float64(c.G)+0.0722*float64(c.B) < lum {
				return x - b.Min.X
			}
		}
	}
	return -1
}

// ---- The line box ----

// lineBoxProse repeats one syllable carrying a capital, an x-height letter and
// a descender, so every line it wraps to draws the same band: from the cap tops
// down to the descender's foot. The distance between two such bands is then the
// pitch itself, with nothing about the words left in it.
const lineBoxProse = "Hxg Hxg Hxg Hxg Hxg Hxg Hxg Hxg Hxg Hxg Hxg Hxg Hxg Hxg Hxg Hxg"

// glyphBands returns the vertical extent of every run of rows carrying glyphs,
// in order, as half-open [top, bottom) intervals. A drawn pixel is any pixel
// departing from the corner colour by more than a small luminance threshold.
func glyphBands(img *image.RGBA) [][2]int {
	b := img.Bounds()
	lum := func(x, y int) float64 {
		c := img.RGBAAt(x, y)
		return 0.2126*float64(c.R) + 0.7152*float64(c.G) + 0.0722*float64(c.B)
	}
	bg := lum(b.Max.X-1, b.Max.Y-1)
	var out [][2]int
	top := -1
	for y := b.Min.Y; y < b.Max.Y; y++ {
		drawn := false
		for x := b.Min.X; x < b.Max.X && !drawn; x++ {
			if d := lum(x, y) - bg; d > 24 || d < -24 {
				drawn = true
			}
		}
		switch {
		case drawn && top < 0:
			top = y
		case !drawn && top >= 0:
			out = append(out, [2]int{top, y})
			top = -1
		}
	}
	if top >= 0 {
		out = append(out, [2]int{top, b.Max.Y})
	}
	return out
}

// TestWrappedLinesOccupyTheStylesLineHeight is the contract in one
// measurement, taken off the pixels: the glyphs of one line to the glyphs of the
// next is the style's line height, and a run of n lines is n boxes tall. Both
// halves matter — a paragraph whose lines were spaced right but whose block
// measured wrong would put every following block in the wrong place.
func TestWrappedLinesOccupyTheStylesLineHeight(t *testing.T) {
	shaper := defaultShaper(t)
	style := richtext.FromTokens(tokens.DefaultLight, tokens.DefaultTypography.BodyLarge)
	box := int(style.LineHeight)

	spans := []richtext.SpanStyle{{Content: lineBoxProse}}
	img := golden.Capture(t, image.Pt(120, 200), func(gtx layout.Context) layout.Dimensions {
		paint.FillShape(gtx.Ops, tokens.DefaultLight.Background,
			clip.Rect{Max: gtx.Constraints.Max}.Op())
		return richtext.Render(shaper, style, spans, richtext.Idle())(gtx)
	})
	bands := glyphBands(img)
	if len(bands) < 3 {
		t.Fatalf("scanned %d glyph bands, want at least 3 (one per wrapped line): %v; the probe did not wrap", len(bands), bands)
	}
	for i := 1; i < len(bands); i++ {
		if pitch := bands[i][0] - bands[i-1][0]; pitch != box {
			t.Errorf("line %d draws %d px below line %d, want the style's %d px line height (bands %v)", i, pitch, i-1, box, bands)
		}
	}

	one := measure(shaper, style, []richtext.SpanStyle{{Content: "Hxg"}}, 600)
	if one.Size.Y != box {
		t.Errorf("a single line measures %d px tall, want the %d px line box", one.Size.Y, box)
	}
	three := measure(shaper, style, []richtext.SpanStyle{{Content: "Hxg\nHxg\nHxg"}}, 600)
	if three.Size.Y != 3*box {
		t.Errorf("three lines measure %d px tall, want %d — three whole line boxes", three.Size.Y, 3*box)
	}
}

// TestTheLeadingSplitsAboveAndBelowTheGlyphs holds the line box to the styling
// model the tokens are written in: the space a line has over its own metrics
// is half-leading, split around the glyphs, rather than piled under them. The
// halves are read against the same paragraph laid out with no line height at
// all — the growth at the bottom is what the baseline gains, the growth at the
// top is the rest — so the measurement needs no knowledge of the face.
func TestTheLeadingSplitsAboveAndBelowTheGlyphs(t *testing.T) {
	shaper := defaultShaper(t)
	style := richtext.FromTokens(tokens.DefaultLight, tokens.DefaultTypography.BodyLarge)
	metrics := style
	metrics.LineHeight = 0

	spans := []richtext.SpanStyle{{Content: "Hxg"}}
	natural := measure(shaper, style, spans, 600)
	shaped := measure(shaper, metrics, spans, 600)

	lead := natural.Size.Y - shaped.Size.Y
	if lead <= 1 {
		t.Fatalf("the line box adds %d px over the shaped metrics; the probe has no leading to split", lead)
	}
	below := natural.Baseline - shaped.Baseline
	above := lead - below
	if above < 0 || below < 0 || above-below < -1 || above-below > 1 {
		t.Errorf("a %d px leading landed %d px above the glyphs and %d px below; half-leading splits it evenly, the odd pixel going below", lead, above, below)
	}
}

// TestAMixedSizeSpanKeepsTheLineBox: the box belongs to the paragraph, not to
// the tallest thing on the line. A word quoted into a line in a smaller face
// leaves the line exactly as tall as the prose around it, so nothing hung
// beside that line moves.
func TestAMixedSizeSpanKeepsTheLineBox(t *testing.T) {
	shaper := defaultShaper(t)
	style := richtext.FromTokens(tokens.DefaultLight, tokens.DefaultTypography.BodyLarge)

	plain := measure(shaper, style, []richtext.SpanStyle{{Content: "quoted word here"}}, 600)
	mixed := measure(shaper, style, []richtext.SpanStyle{
		{Content: "quoted "},
		{Content: "word", Size: unit.Sp(tokens.DefaultTypography.Code.Size), Typeface: "Roboto Mono"},
		{Content: " here"},
	}, 600)
	if mixed.Size.Y != plain.Size.Y {
		t.Errorf("a line holding a smaller span measures %d px tall against a plain line's %d; the line box is the paragraph's", mixed.Size.Y, plain.Size.Y)
	}
	if mixed.Baseline != plain.Baseline {
		t.Errorf("a line holding a smaller span baselines %d px above its foot against a plain line's %d; the two must share a baseline", mixed.Baseline, plain.Baseline)
	}
}

// TestAZeroLineHeightKeepsTheShapedMetrics is the escape hatch, pinned: a
// style naming no line height lays out on ascent and descent alone, exactly as
// a paragraph did before the box existed. So does one naming a box shorter
// than the metrics, which has no leading to distribute and must not squeeze
// the lines into an overlap.
func TestAZeroLineHeightKeepsTheShapedMetrics(t *testing.T) {
	shaper := defaultShaper(t)
	style := richtext.FromTokens(tokens.DefaultLight, tokens.DefaultTypography.BodyLarge)
	metrics := style
	metrics.LineHeight = 0

	spans := []richtext.SpanStyle{{Content: "Hxg\nHxg"}}
	shaped := measure(shaper, metrics, spans, 600)
	if shaped.Size.Y >= measure(shaper, style, spans, 600).Size.Y {
		t.Fatalf("the shaped metrics measure %d px for two lines, no less than the line box does; the probe cannot tell the two apart", shaped.Size.Y)
	}
	tight := style
	tight.LineHeight = 1
	if got := measure(shaper, tight, spans, 600); got.Size.Y != shaped.Size.Y || got.Baseline != shaped.Baseline {
		t.Errorf("a 1 sp line box laid two lines out %d px tall with baseline %d, want the shaped %d/%d; a box under the metrics leaves them alone",
			got.Size.Y, got.Baseline, shaped.Size.Y, shaped.Baseline)
	}
}

// TestHardNewlineBreaksLine verifies that a \n inside a span forces a line
// break: the two-line content must be taller than the same content on one
// line.
func TestHardNewlineBreaksLine(t *testing.T) {
	shaper := defaultShaper(t)
	style := richtext.FromTokens(tokens.DefaultLight, tokens.DefaultTypography.BodyLarge)

	oneLine := measure(shaper, style, []richtext.SpanStyle{{Content: "alpha beta"}}, 600)
	twoLines := measure(shaper, style, []richtext.SpanStyle{{Content: "alpha\nbeta"}}, 600)

	if twoLines.Size.Y <= oneLine.Size.Y {
		t.Errorf("newline content height %d not greater than single line %d", twoLines.Size.Y, oneLine.Size.Y)
	}
}

// ---- Interaction tests ----

func driveFrame(w layout.Widget, ops *op.Ops, r *gioinput.Router, size image.Point) layout.Dimensions {
	ops.Reset()
	gtx := layout.Context{
		Metric:      unit.Metric{PxPerDp: 1, PxPerSp: 1},
		Constraints: layout.Constraints{Max: size},
		Ops:         ops,
		Source:      r.Source(),
	}
	dims := w(gtx)
	r.Frame(ops)
	return dims
}

// linkFirstSpans puts the link at the paragraph origin so its interactive
// area is at a known position for synthetic pointer events.
func linkFirstSpans(url string) []richtext.SpanStyle {
	return []richtext.SpanStyle{
		{Content: "click here", URL: url},
		{Content: " for docs."},
	}
}

// TestLinkClickFiresOnLinkClick drives a pointer press+release over the link
// segment and expects OnLinkClick to fire with the link's URL and a live
// layout.Context.
func TestLinkClickFiresOnLinkClick(t *testing.T) {
	shaper := defaultShaper(t)
	const url = "https://example.com/docs"

	var gotURL string
	var gotOps bool
	style := richtext.FromTokens(tokens.DefaultLight, tokens.DefaultTypography.BodyLarge)
	style.OnLinkClick = func(gtx layout.Context, u string) {
		gotURL = u
		gotOps = gtx.Ops != nil
	}

	state := richtext.NewState()
	w := func(gtx layout.Context) layout.Dimensions {
		return richtext.Layout(gtx, state, shaper, style, linkFirstSpans(url))
	}

	r := new(gioinput.Router)
	ops := new(op.Ops)
	size := image.Pt(400, 100)

	// Frame 1 registers the link area; the click then lands inside the
	// link's first segment (origin at 0,0; ~16 px tall at 16 sp).
	driveFrame(w, ops, r, size)
	hit := f32.Pt(6, 8)
	r.Queue(
		pointer.Event{Kind: pointer.Press, Position: hit, Buttons: pointer.ButtonPrimary, Source: pointer.Mouse},
		pointer.Event{Kind: pointer.Release, Position: hit, Source: pointer.Mouse},
	)
	driveFrame(w, ops, r, size)

	if gotURL != url {
		t.Fatalf("OnLinkClick url = %q, want %q", gotURL, url)
	}
	if !gotOps {
		t.Error("OnLinkClick received a layout.Context without Ops; callbacks must carry the live gtx (GX.8)")
	}
}

// TestLinkFocusTraversalAndKeyboardActivation moves focus forward across two
// links (the router-level traversal Gio's window drives from Tab), asserting
// the focus order matches document order, and activates each focused link
// with Enter and Space.
func TestLinkFocusTraversalAndKeyboardActivation(t *testing.T) {
	shaper := defaultShaper(t)

	var clicks []string
	style := richtext.FromTokens(tokens.DefaultLight, tokens.DefaultTypography.BodyLarge)
	style.OnLinkClick = func(_ layout.Context, u string) { clicks = append(clicks, u) }

	spans := []richtext.SpanStyle{
		{Content: "first", URL: "https://a.example"},
		{Content: " and "},
		{Content: "second", URL: "https://b.example"},
	}
	state := richtext.NewState()
	w := func(gtx layout.Context) layout.Dimensions {
		return richtext.Layout(gtx, state, shaper, style, spans)
	}

	r := new(gioinput.Router)
	ops := new(op.Ops)
	size := image.Pt(400, 100)

	probe := func() layout.Context {
		return layout.Context{
			Metric:      unit.Metric{PxPerDp: 1, PxPerSp: 1},
			Constraints: layout.Constraints{Max: size},
			Ops:         new(op.Ops),
			Source:      r.Source(),
		}
	}

	// Frame 1 registers each link as focusable.
	driveFrame(w, ops, r, size)
	if got := state.FocusedLink(probe()); got != richtext.NoLink {
		t.Fatalf("initial FocusedLink = %d, want NoLink", got)
	}

	// Forward traversal reaches link 0, then link 1, in document order.
	r.MoveFocus(key.FocusForward)
	driveFrame(w, ops, r, size)
	if got := state.FocusedLink(probe()); got != 0 {
		t.Fatalf("after first MoveFocus, FocusedLink = %d, want 0", got)
	}

	// Enter activates the focused link.
	r.Queue(
		key.Event{Name: key.NameReturn, State: key.Press},
		key.Event{Name: key.NameReturn, State: key.Release},
	)
	driveFrame(w, ops, r, size)
	if len(clicks) != 1 || clicks[0] != "https://a.example" {
		t.Fatalf("after Enter on link 0, clicks = %v, want [https://a.example]", clicks)
	}

	r.MoveFocus(key.FocusForward)
	driveFrame(w, ops, r, size)
	if got := state.FocusedLink(probe()); got != 1 {
		t.Fatalf("after second MoveFocus, FocusedLink = %d, want 1", got)
	}

	// Space also activates.
	r.Queue(
		key.Event{Name: key.NameSpace, State: key.Press},
		key.Event{Name: key.NameSpace, State: key.Release},
	)
	driveFrame(w, ops, r, size)
	if len(clicks) != 2 || clicks[1] != "https://b.example" {
		t.Fatalf("after Space on link 1, clicks = %v, want [... https://b.example]", clicks)
	}
}

// ---- Token defaults ----

// TestFromTokensDefaults pins the FromTokens contract: body text in Text at
// BodyLarge, links in Primary, and the focus ring in the library's one ring
// colour — the primary step measured against the paragraph surface it is
// drawn on. Both schemes: the step a light scheme walks to and the step a dark
// one walks to are different steps, and a paragraph gets whichever its own tokens
// name.
func TestFromTokensDefaults(t *testing.T) {
	for _, s := range []struct {
		name string
		tok  tokens.ColorTokens
	}{
		{"light", tokens.DefaultLight},
		{"dark", tokens.DefaultDark},
	} {
		st := richtext.FromTokens(s.tok, tokens.DefaultTypography.BodyLarge)
		if st.Color != s.tok.Text {
			t.Errorf("%s: Color = %v, want Text %v", s.name, st.Color, s.tok.Text)
		}
		if st.LinkColor != s.tok.Primary {
			t.Errorf("%s: LinkColor = %v, want Primary %v", s.name, st.LinkColor, s.tok.Primary)
		}
		if want := focus.Ring(s.tok); st.FocusColor != want {
			t.Errorf("%s: FocusColor = %v, want the measured ring %v", s.name, st.FocusColor, want)
		}
		if got := tcolor.ContrastRatio(st.FocusColor, s.tok.Surface); got < focus.Floor {
			t.Errorf("%s: the link ring %v measures %.2f:1 against the paragraph surface %v",
				s.name, st.FocusColor, got, s.tok.Surface)
		}
		if st.Size != unit.Sp(tokens.DefaultTypography.BodyLarge.Size) {
			t.Errorf("%s: Size = %v, want BodyLarge %v", s.name, st.Size, tokens.DefaultTypography.BodyLarge.Size)
		}
	}
}

// ---- Color emoji ----

// emojiShaper is the pinned collection plus Noto Color Emoji. WithEmoji
// does not exist yet; this is the equivalent that already does.
func emojiShaper(t *testing.T) *text.Shaper {
	t.Helper()
	typ := tokens.DefaultTypography.WithFaces(notocoloremoji.FontFace())
	return typ.DeterministicShaper()
}

// resolvedGlyph shapes one rune and reports the font's own glyph ID and the
// face it came from. Glyph ID 0 is .notdef; face 0 is Roboto.
func resolvedGlyph(t *testing.T, shaper *text.Shaper, r rune) (gid uint32, faceIdx int) {
	t.Helper()
	shaper.LayoutString(text.Parameters{
		Font:     font.Font{Typeface: "Roboto"},
		PxPerEm:  fixed.I(16),
		MaxWidth: 1000,
	}, string(r))
	g, ok := shaper.NextGlyph()
	if !ok {
		t.Fatalf("U+%04X %q: shaper produced no glyph at all", r, r)
	}
	return uint32(g.ID), int(uint64(g.ID) >> 48)
}

// TestEmojiResolvesOnAppendedFace pins the collection the painter draws
// from: 😀 is a real glyph on the appended face, "A" stays on Roboto, and
// the same grin on the default pinned shaper is tofu.
func TestEmojiResolvesOnAppendedFace(t *testing.T) {
	emoji := tokens.DefaultTypography.WithFaces(notocoloremoji.FontFace())
	shaper := emoji.DeterministicShaper()
	appended := len(emoji.Faces) - 1

	gid, faceIdx := resolvedGlyph(t, shaper, '😀')
	if gid == 0 {
		t.Fatal("😀 resolved to glyph ID 0 (.notdef) on the appended face")
	}
	if faceIdx != appended {
		t.Errorf("😀 resolved on face %d, want the appended emoji face %d", faceIdx, appended)
	}
	if _, faceIdx := resolvedGlyph(t, shaper, 'A'); faceIdx != 0 {
		t.Errorf("'A' resolved on face %d, want Roboto at 0", faceIdx)
	}

	tofu, _ := resolvedGlyph(t, defaultShaper(t), '😀')
	if tofu != 0 {
		t.Errorf("😀 on DefaultTypography.DeterministicShaper resolved to glyph %d, want 0 (tofu control)", tofu)
	}
}

const emojiInline = "Hi 😀!"

func captureEmojiInline(t *testing.T, shaper *text.Shaper, colors tokens.ColorTokens) *image.RGBA {
	t.Helper()
	style := richtext.FromTokens(colors, tokens.DefaultTypography.BodyLarge)
	size := image.Pt(200, 48)
	return golden.Capture(t, size, func(gtx layout.Context) layout.Dimensions {
		paint.FillShape(gtx.Ops, colors.Background, clip.Rect{Max: gtx.Constraints.Max}.Op())
		return richtext.Render(shaper, style, []richtext.SpanStyle{{Content: emojiInline}}, richtext.Idle())(gtx)
	})
}

// TestEmojiInlinePaintsThePNG is the paint contract: the same span with the
// face is not the tofu capture, and the grin's bounds hold chromatic pixels
// that are not the body colour — the PNG, not a ColorOp-tinted hole.
func TestEmojiInlinePaintsThePNG(t *testing.T) {
	with := emojiShaper(t)
	without := defaultShaper(t)
	style := richtext.FromTokens(tokens.DefaultLight, tokens.DefaultTypography.BodyLarge)

	painted := captureEmojiInline(t, with, tokens.DefaultLight)
	tofu := captureEmojiInline(t, without, tokens.DefaultLight)
	if golden.PixelDiff(painted, tofu) == 0 {
		t.Fatal("Hi 😀! with the emoji face matches the tofu control; Bitmaps did not paint")
	}

	x0, x1 := grinXRange(t, with, int(style.Size))
	if !hasChromaticUnlike(painted, x0, x1, style.Color) {
		t.Errorf("no chromatic pixel unlike the body colour %v in the grin's x-range [%d, %d); the PNG is missing or tinted",
			style.Color, x0, x1)
	}
}

// TestEmojiInlineGolden records the with-face capture in both schemes.
func TestEmojiInlineGolden(t *testing.T) {
	shaper := emojiShaper(t)
	size := image.Pt(200, 48)
	cases := []struct {
		name   string
		colors tokens.ColorTokens
	}{
		{"emoji-inline-light", tokens.DefaultLight},
		{"emoji-inline-dark", tokens.DefaultDark},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			style := richtext.FromTokens(tc.colors, tokens.DefaultTypography.BodyLarge)
			golden.Render(t, tc.name, size, func(gtx layout.Context) layout.Dimensions {
				paint.FillShape(gtx.Ops, tc.colors.Background, clip.Rect{Max: gtx.Constraints.Max}.Op())
				return richtext.Render(shaper, style, []richtext.SpanStyle{{Content: emojiInline}}, richtext.Idle())(gtx)
			})
		})
	}
}

// grinXRange is the horizontal extent of glyphs that did not come from
// Roboto (face 0) when shaping s. Half-leading moves glyphs in y only.
func grinXRange(t *testing.T, shaper *text.Shaper, pxPerEm int) (x0, x1 int) {
	t.Helper()
	shaper.LayoutString(text.Parameters{
		Font:     font.Font{Typeface: "Roboto"},
		PxPerEm:  fixed.I(pxPerEm),
		MaxWidth: 1000,
	}, emojiInline)
	found := false
	for g, ok := shaper.NextGlyph(); ok; g, ok = shaper.NextGlyph() {
		if int(uint64(g.ID)>>48) == 0 {
			continue
		}
		lo := min(g.X.Floor(), g.X.Floor()+g.Bounds.Min.X.Floor())
		hi := max((g.X + g.Advance).Ceil(), g.X.Floor()+g.Bounds.Max.X.Ceil())
		if !found {
			x0, x1, found = lo, hi, true
			continue
		}
		x0 = min(x0, lo)
		x1 = max(x1, hi)
	}
	if !found {
		t.Fatal("Hi 😀! produced no non-Roboto glyph; cannot locate the grin")
	}
	return x0, x1
}

func hasChromaticUnlike(img *image.RGBA, x0, x1 int, body color.NRGBA) bool {
	b := img.Bounds()
	if x0 < b.Min.X {
		x0 = b.Min.X
	}
	if x1 > b.Max.X {
		x1 = b.Max.X
	}
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := x0; x < x1; x++ {
			c := img.RGBAAt(x, y)
			if c.R == body.R && c.G == body.G && c.B == body.B {
				continue
			}
			maxc := max(c.R, max(c.G, c.B))
			minc := min(c.R, min(c.G, c.B))
			if int(maxc)-int(minc) > 40 {
				return true
			}
		}
	}
	return false
}

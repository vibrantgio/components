package badge_test

import (
	"image"
	"image/color"
	"testing"

	"gioui.org/f32"
	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"

	"github.com/vibrantgio/components/badge"
	"github.com/vibrantgio/components/golden"
	"github.com/vibrantgio/theme/tokens"
	"github.com/vibrantgio/theme/typeset"
)

// check is the deterministic sign the badge goldens draw: a tick built from
// two clip.Stroke lines in a sizePx×sizePx box. Being vector rather than font
// or SVG rasterisation, it keeps the stored images stable on every machine.
//
// Its stroke spans most of the box and is centred on it, both deliberately:
// the badge reserves the box, so a sign that under-fills it reads as a gap in
// the line, and one whose stroke is not centred on the box sits off the
// baseline the
// label keeps.
func check(gtx layout.Context, sizePx int, col color.NRGBA) {
	w := float32(sizePx)
	stroke := float32(gtx.Dp(unit.Dp(1.5)))
	if stroke < 1 {
		stroke = 1
	}
	var p clip.Path
	p.Begin(gtx.Ops)
	p.MoveTo(f32.Pt(w*0.16, w*0.52))
	p.LineTo(f32.Pt(w*0.42, w*0.76))
	p.LineTo(f32.Pt(w*0.84, w*0.24))
	paint.FillShape(gtx.Ops, col, clip.Stroke{Path: p.End(), Width: stroke}.Op())
}

// defaultShaper returns the shaper every golden here draws with: the default
// typography's faces pinned, system fonts off, so the stored images are the
// same on every machine. See components/AGENTS.md.
func defaultShaper(t *testing.T) *text.Shaper {
	t.Helper()
	return tokens.DefaultTypography.DeterministicShaper()
}

// goldenInset is the air around the specimen inside its stored image. A badge
// drawn at the image origin has the image edge on two sides, and an image
// framed that way cannot show whether the inline box the badge reported is the
// shape it drew.
const goldenInset = 12

// onPage paints the whole frame in the platform's content fill and draws w
// inset inside it. The page is not decoration: a badge captured over the
// headless window's own clear colour cannot be told from one that painted
// nothing, and the pixel gates below sample the page as well as the badge.
func onPage(p tokens.PlatformColors, w layout.Widget) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		paint.FillShape(gtx.Ops, p.ControlBackground, clip.Rect{Max: gtx.Constraints.Max}.Op())
		return layout.UniformInset(unit.Dp(goldenInset)).Layout(gtx, w)
	}
}

// row lays layout.Widgets out on one line with the S4 stop between them. A badge pads
// its own content but nothing outside itself, so what separates two of them
// belongs to whatever sets them, and a stored image of a row has to say which
// stop it used.
func row(ws ...layout.Widget) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		cs := make([]layout.FlexChild, 0, 2*len(ws))
		for i, w := range ws {
			if i > 0 {
				cs = append(cs, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Dimensions{Size: image.Pt(gtx.Dp(unit.Dp(tokens.Spacing.S4)), 0)}
				}))
			}
			cs = append(cs, layout.Rigid(w))
		}
		return layout.Flex{Alignment: layout.Middle}.Layout(gtx, cs...)
	}
}

// goldenSchemes is the pair every specimen is recorded in: the platform's
// two recorded appearances.
var goldenSchemes = []struct {
	name string
	p    tokens.PlatformColors
}{
	{"light", tokens.PlatformLight},
	{"dark", tokens.PlatformDark},
}

// goldenStatuses is the whole vocabulary, each labelled with its own name so
// the row reads without a caption under it.
var goldenStatuses = []struct {
	label  string
	status badge.Status
}{
	{"Neutral", badge.Neutral},
	{"Success", badge.Success},
	{"Warning", badge.Warning},
	{"Error", badge.Error},
	{"Info", badge.Info},
}

// goldenSize is an image comfortably larger than a row of badges, so the
// stored image carries the page around the words as well as the words.
var goldenSize = image.Pt(420, 44)

// badgeStyle is the type role a Comfortable badge is set in, asked of the component
// rather than named, so a test cannot claim a size the badge does not draw.
func badgeStyle() tokens.TextStyle {
	return badge.Style(tokens.DefaultTypography, tokens.Comfortable)
}

// TestBadgeGolden records the five side by side in both appearances: the
// platform's four status colours and its grey, each filled under white.
// Two images, where there were six — a badge wears the platform's system
// colour whatever it stands on, so there is no surface to vary.
func TestBadgeGolden(t *testing.T) {
	shaper := defaultShaper(t)
	for _, sc := range goldenSchemes {
		name := "badge-" + sc.name
		t.Run(name, func(t *testing.T) {
			ws := make([]layout.Widget, 0, len(goldenStatuses))
			for _, st := range goldenStatuses {
				ws = append(ws, badge.Render(shaper, st.label, nil, st.status,
					sc.p, tokens.Spacing, tokens.Radius, badgeStyle(),
					badge.RenderState{}))
			}
			golden.Render(t, name, goldenSize, onPage(sc.p, row(ws...)))
		})
	}
}

// TestUtterancesGolden records the three things a badge can say, at one
// status, so the images show what a single structure means: a word, a count and
// a sign at the same weight, in the same colour, on the same line.
func TestUtterancesGolden(t *testing.T) {
	shaper := defaultShaper(t)
	for _, sc := range goldenSchemes {
		name := "badge-" + sc.name + "-utterances"
		t.Run(name, func(t *testing.T) {
			ws := []layout.Widget{
				badge.Render(shaper, "Popular", nil, badge.Info,
					sc.p, tokens.Spacing, tokens.Radius, badgeStyle(), badge.RenderState{}),
				badge.Render(shaper, "128", nil, badge.Info,
					sc.p, tokens.Spacing, tokens.Radius, badgeStyle(), badge.RenderState{}),
				badge.Render(shaper, "", check, badge.Info,
					sc.p, tokens.Spacing, tokens.Radius, badgeStyle(), badge.RenderState{}),
				badge.Render(shaper, "Verified", check, badge.Info,
					sc.p, tokens.Spacing, tokens.Radius, badgeStyle(), badge.RenderState{}),
			}
			golden.Render(t, name, goldenSize, onPage(sc.p, row(ws...)))
		})
	}
}

// TestDismissGolden records the close mark through the states the pointer puts
// it in. The mark's region takes the platform's hover overlay and then its
// press overlay, so the three tiles show it coming forward — darkening in the
// light appearance and lightening in the dark one.
func TestDismissGolden(t *testing.T) {
	shaper := defaultShaper(t)
	states := []struct {
		label string
		s     badge.RenderState
	}{
		{"Rest", badge.RenderState{}},
		{"Hover", badge.RenderState{DismissHovered: true}},
		{"Press", badge.RenderState{DismissPressed: true}},
	}
	for _, sc := range goldenSchemes {
		name := "badge-" + sc.name + "-dismiss"
		t.Run(name, func(t *testing.T) {
			ws := make([]layout.Widget, 0, len(states))
			for _, st := range states {
				ws = append(ws, badge.RenderDismissible(shaper, st.label, nil, badge.Neutral,
					nil, sc.p, tokens.Spacing, tokens.Radius, badgeStyle(), st.s))
			}
			golden.Render(t, name, goldenSize, onPage(sc.p, row(ws...)))
		})
	}
}

// TestCompactGolden records the dense badge beside nothing else, because the
// only thing that changes at Compact is the type: LabelSmall rather than
// LabelMedium, at the same line box, which is what "off the control family"
// costs a density switch.
func TestCompactGolden(t *testing.T) {
	shaper := defaultShaper(t)
	style := badge.Style(tokens.DefaultTypography, tokens.Compact)
	ws := make([]layout.Widget, 0, len(goldenStatuses))
	for _, st := range goldenStatuses {
		ws = append(ws, badge.Render(shaper, st.label, nil, st.status,
			tokens.PlatformLight, tokens.Spacing, tokens.Radius, style, badge.RenderState{}))
	}
	golden.Render(t, "badge-light-compact", goldenSize,
		onPage(tokens.PlatformLight, row(ws...)))
}

// measure lays a layout.Widget out at one pixel per dp in a generous box and reports
// what it drew, which is the only honest way to ask a component its height.
func measure(t *testing.T, w layout.Widget) image.Point {
	t.Helper()
	var ops op.Ops
	gtx := layout.Context{
		Metric:      unit.Metric{PxPerDp: 1, PxPerSp: 1},
		Constraints: layout.Constraints{Max: image.Pt(1000, 1000)},
		Ops:         &ops,
	}
	return w(gtx).Size
}

// TestHeightIsTheLineBoxAndNothingElse is the ruling in one assertion: a badge
// is as tall as its type's line box, at every density, whatever it says. There
// is no padding term, no floor and no control height anywhere in the number.
func TestHeightIsTheLineBoxAndNothingElse(t *testing.T) {
	shaper := defaultShaper(t)
	for _, d := range []struct {
		name string
		d    tokens.Density
	}{
		{"comfortable", tokens.Comfortable},
		{"compact", tokens.Compact},
	} {
		style := badge.Style(tokens.DefaultTypography, d.d)
		want := int(style.LineHeight)
		for _, tc := range []struct {
			name  string
			label string
			glyph badge.Glyph
		}{
			{"word", "Popular", nil},
			{"count", "9", nil},
			{"glyph", "", check},
			{"both", "Verified", check},
		} {
			t.Run(d.name+" "+tc.name, func(t *testing.T) {
				got := measure(t, badge.Render(shaper, tc.label, tc.glyph, badge.Neutral,
					tokens.PlatformLight, tokens.Spacing, tokens.Radius, style, badge.RenderState{}))
				if got.Y != want {
					t.Errorf("height = %d dp, want the %g dp line box of the %s role",
						got.Y, style.LineHeight, d.name)
				}
			})
		}
	}
}

// TestABadgeIsLighterThanAnyControl is the other half of the same ruling,
// measured against the family it is off: whatever it says and at whichever
// density, a badge draws well under the control height of the densest mode
// this system has.
func TestABadgeIsLighterThanAnyControl(t *testing.T) {
	shaper := defaultShaper(t)
	for _, d := range []tokens.Density{tokens.Comfortable, tokens.Compact} {
		style := badge.Style(tokens.DefaultTypography, d)
		got := measure(t, badge.Render(shaper, "Deprecated", check, badge.Warning,
			tokens.PlatformLight, tokens.Spacing, tokens.Radius, style, badge.RenderState{}))
		if float32(got.Y) >= tokens.CompactControlHeight {
			t.Errorf("a badge measured %d dp tall against the densest control height %g: a badge is not in the control family",
				got.Y, tokens.CompactControlHeight)
		}
	}
}

// TestTheDensityPicksTheTypeRole pins the table the package doc states. The
// two roles share a line box, so what a density switch moves is the type's
// size and not the badge's height — which is what a component off the control
// family is.
func TestTheDensityPicksTheTypeRole(t *testing.T) {
	typo := tokens.DefaultTypography
	if got := badge.Style(typo, tokens.Comfortable); got != typo.LabelMedium {
		t.Errorf("Comfortable badge style = %+v, want LabelMedium %+v", got, typo.LabelMedium)
	}
	if got := badge.Style(typo, tokens.Compact); got != typo.LabelSmall {
		t.Errorf("Compact badge style = %+v, want LabelSmall %+v", got, typo.LabelSmall)
	}
	if badge.Style(typo, tokens.Comfortable).Size == badge.Style(typo, tokens.Compact).Size {
		t.Error("both densities are set at the same type size: density does not reach the badge")
	}
}

// TestBadgeIsSizedToItsContent: a badge is a run of text and a run of text does
// not stretch. A badge that filled its box would be a banner.
func TestBadgeIsSizedToItsContent(t *testing.T) {
	shaper := defaultShaper(t)
	render := func(label string) layout.Widget {
		return badge.Render(shaper, label, nil, badge.Neutral,
			tokens.PlatformLight, tokens.Spacing, tokens.Radius, badgeStyle(), badge.RenderState{})
	}
	short := measure(t, render("A"))
	long := measure(t, render("A considerably longer statement"))
	if short.X >= long.X {
		t.Errorf("a one-letter badge measured %d dp wide and a long one %d: the badge is not sized to its label",
			short.X, long.X)
	}
	if short.X >= 1000 {
		t.Errorf("badge width %d dp fills the 1000 dp box it was given; a badge is sized to its content", short.X)
	}
}

// TestTheSignCostsTheLineBoxAndOneStop pins the geometry the package doc
// states for the sign: it is the label's own line box, and it leads the label
// across the spacing scale's S1 stop.
func TestTheSignCostsTheLineBoxAndOneStop(t *testing.T) {
	shaper := defaultShaper(t)
	style := badgeStyle()
	render := func(glyph badge.Glyph) layout.Widget {
		return badge.Render(shaper, "Verified", glyph, badge.Neutral,
			tokens.PlatformLight, tokens.Spacing, tokens.Radius, style, badge.RenderState{})
	}
	bare := measure(t, render(nil))
	signed := measure(t, render(check))
	want := int(style.LineHeight) + int(tokens.Spacing.S1)
	if got := signed.X - bare.X; got != want {
		t.Errorf("the sign cost the badge %d dp, want %d (the %g dp line box plus the S1 %g dp gap)",
			got, want, style.LineHeight, tokens.Spacing.S1)
	}
	if bare.Y != signed.Y {
		t.Errorf("a badge with a sign is %d dp tall and one without is %d: the sign must not move the height",
			signed.Y, bare.Y)
	}
}

// TestTheCloseMarkCostsHalfTheLineBoxAndOneStop pins the other end: the mark
// is half the line box, one S1 stop after the label, and it does not make the
// badge taller. What it costs the badge in width is the drawn mark only — the
// 24 dp target under it is slop and belongs to no layout.
func TestTheCloseMarkCostsHalfTheLineBoxAndOneStop(t *testing.T) {
	shaper := defaultShaper(t)
	style := badgeStyle()
	bare := measure(t, badge.Render(shaper, "Filtered", nil, badge.Neutral,
		tokens.PlatformLight, tokens.Spacing, tokens.Radius, style, badge.RenderState{}))
	dismissible := measure(t, badge.RenderDismissible(shaper, "Filtered", nil, badge.Neutral,
		nil, tokens.PlatformLight, tokens.Spacing, tokens.Radius, style, badge.RenderState{}))
	want := int(style.LineHeight)/2 + int(tokens.Spacing.S1)
	if got := dismissible.X - bare.X; got != want {
		t.Errorf("the close mark cost the badge %d dp, want %d (half the %g dp line box plus the S1 %g dp gap)",
			got, want, style.LineHeight, tokens.Spacing.S1)
	}
	if got := dismissible.X - bare.X; got >= badge.CloseHitDp {
		t.Errorf("the close mark cost the badge %d dp, which is the %d dp target or more: the target is slop, not width",
			got, badge.CloseHitDp)
	}
	if bare.Y != dismissible.Y {
		t.Errorf("a dismissible badge is %d dp tall and a plain one is %d: the mark must not move the height",
			dismissible.Y, bare.Y)
	}
}

// TestThePointerIsVisibleOnTheCloseMark is the acknowledgement in pixels: the
// three tiles TestDismissGolden stores must differ from each other, or the one
// thing on a badge that answers a pointer answers it invisibly.
func TestThePointerIsVisibleOnTheCloseMark(t *testing.T) {
	shaper := defaultShaper(t)
	for _, sc := range goldenSchemes {
		frame := func(s badge.RenderState) *image.RGBA {
			w := badge.RenderDismissible(shaper, "Filtered", nil, badge.Neutral,
				nil, sc.p, tokens.Spacing, tokens.Radius, badgeStyle(), s)
			return golden.Capture(t, goldenSize, onPage(sc.p, w))
		}
		rest := frame(badge.RenderState{})
		for _, tc := range []struct {
			name string
			s    badge.RenderState
		}{
			{"hovered", badge.RenderState{DismissHovered: true}},
			{"pressed", badge.RenderState{DismissPressed: true}},
		} {
			if n := golden.PixelDiff(rest, frame(tc.s)); n == 0 {
				t.Errorf("%s: a %s close mark is pixel-identical to a resting one", sc.name, tc.name)
			}
		}
	}
}

// TestTheBodyTakesNoPointerState is the ruling's other side: a badge is read,
// not used, so nothing about its words moves when the pointer is on the mark.
// Only the mark's own pixels may change.
func TestTheBodyTakesNoPointerState(t *testing.T) {
	shaper := defaultShaper(t)
	style := badgeStyle()
	plain := func(s badge.RenderState) *image.RGBA {
		w := badge.Render(shaper, "Filtered", check, badge.Error,
			tokens.PlatformLight, tokens.Spacing, tokens.Radius, style, s)
		return golden.Capture(t, goldenSize, onPage(tokens.PlatformLight, w))
	}
	rest := plain(badge.RenderState{})
	for _, tc := range []struct {
		name string
		s    badge.RenderState
	}{
		{"hovered", badge.RenderState{DismissHovered: true}},
		{"pressed", badge.RenderState{DismissPressed: true}},
	} {
		if n := golden.PixelDiff(rest, plain(tc.s)); n != 0 {
			t.Errorf("a badge with no close mark drew %d pixels differently when %s: the body takes no pointer state",
				n, tc.name)
		}
	}
}

// dimensions is [measure] when the whole answer is wanted rather than the size
// — the baseline included, which is the point of the tests below.
func dimensions(t *testing.T, w layout.Widget) layout.Dimensions {
	t.Helper()
	var ops op.Ops
	gtx := layout.Context{
		Metric:      unit.Metric{PxPerDp: 1, PxPerSp: 1},
		Constraints: layout.Constraints{Max: image.Pt(1000, 1000)},
		Ops:         &ops,
	}
	return w(gtx)
}

// TestTheBadgeReportsItsLabelsBaseline is the fix for a row that could not be
// set on one line: layout.Baseline aligns on Dimensions.Baseline, a badge that
// reports zero there is aligned by its box instead, and a badge whose box is
// the line box while the words beside it are set in a larger role lands a few
// pixels off the line it belongs on.
//
// A glyph badge reports none on purpose. A sign has no baseline to offer, and
// zero is what Gio reads as "align me by my box".
func TestTheBadgeReportsItsLabelsBaseline(t *testing.T) {
	shaper := defaultShaper(t)
	style := badgeStyle()
	worded := dimensions(t, badge.Render(shaper, "Popular", nil, badge.Neutral,
		tokens.PlatformLight, tokens.Spacing, tokens.Radius, style, badge.RenderState{}))
	if worded.Baseline <= 0 {
		t.Errorf("a worded badge reports baseline %d: a row aligned on it has nothing to align on", worded.Baseline)
	}
	if worded.Baseline >= worded.Size.Y {
		t.Errorf("a worded badge %d dp tall reports baseline %d, which is at or above its own top edge",
			worded.Size.Y, worded.Baseline)
	}
	// The baseline is measured up from the bottom, and the label fills the
	// badge's whole height, so the two are the same number by construction —
	// which is the claim: the badge passes on what the shaper told it rather
	// than inventing a line of its own.
	if got, want := worded.Baseline, typesetBaseline(t, shaper, style, "Popular"); got != want {
		t.Errorf("the badge reports baseline %d and its own typesetting reports %d", got, want)
	}
	glyphOnly := dimensions(t, badge.Render(shaper, "", check, badge.Neutral,
		tokens.PlatformLight, tokens.Spacing, tokens.Radius, style, badge.RenderState{}))
	if glyphOnly.Baseline != 0 {
		t.Errorf("a glyph badge reports baseline %d: a sign has none to report", glyphOnly.Baseline)
	}
}

// typesetBaseline is what the label alone reports, laid out the way the badge
// lays it out.
func typesetBaseline(t *testing.T, shaper *text.Shaper, style tokens.TextStyle, label string) int {
	t.Helper()
	var ops op.Ops
	gtx := layout.Context{
		Metric:      unit.Metric{PxPerDp: 1, PxPerSp: 1},
		Constraints: layout.Constraints{Max: image.Pt(1000, 1000)},
		Ops:         &ops,
	}
	return typeset.Layout(gtx, shaper, typeset.Label(style, 1), typeset.Font(style, font.Normal),
		unit.Sp(style.Size), label, op.CallOp{}).Baseline
}

// onCard paints the frame in the platform's box fill and draws w inset in it.
// It is the page the measuring tests use, and it is not the content's: in the
// light appearance the content's fill and the foreground a filled badge knocks
// out are the same white, so a measurement taken over the content cannot tell
// the sign from the page around it. The box fill is neither, in both
// appearances.
func onCard(p tokens.PlatformColors, w layout.Widget) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		paint.FillShape(gtx.Ops, p.CardFill, clip.Rect{Max: gtx.Constraints.Max}.Op())
		return layout.UniformInset(unit.Dp(goldenInset)).Layout(gtx, w)
	}
}

// sameColour reports whether two colours are the same within one step of the
// rounding Gio's paint pipeline does on the way to the framebuffer and back:
// a fill handed in as sRGB is converted to linear light and back, and an
// opaque channel comes out of that round trip up to two 255ths from where it
// went in. Measured on the platform's own system colours — systemBlue's
// #0088ff paints as #0288ff — so a sampler that demanded equality would be
// pinning the round trip rather than the colour.
func sameColour(a, b color.NRGBA) bool {
	const tol = 2
	d := func(x, y uint8) bool {
		if x > y {
			return x-y <= tol
		}
		return y-x <= tol
	}
	return d(a.R, b.R) && d(a.G, b.G) && d(a.B, b.B) && d(a.A, b.A)
}

// badgePixel samples one pixel of a badge captured over the page, addressed
// from the badge's own top-left corner rather than the image's.
func badgePixel(t *testing.T, img *image.RGBA, dx, dy int) color.NRGBA {
	t.Helper()
	c := img.RGBAAt(goldenInset+dx, goldenInset+dy)
	return color.NRGBA{R: c.R, G: c.G, B: c.B, A: c.A}
}

// TestAWordedBadgeWearsItsFill is the structure in pixels: the fill is there,
// it is the platform's system colour for the status, it is inset from the
// label by the padding stop, it stops at the badge's own reported edge, and
// its corner is cut.
//
// Sampled rather than diffed because what is being asserted is which colour
// landed where, and a pixel count cannot say that.
func TestAWordedBadgeWearsItsFill(t *testing.T) {
	shaper := defaultShaper(t)
	style := badgeStyle()
	pad := int(tokens.Spacing.S2)
	for _, sc := range goldenSchemes {
		for _, st := range goldenStatuses {
			w := badge.Render(shaper, st.label, nil, st.status,
				sc.p, tokens.Spacing, tokens.Radius, style, badge.RenderState{})
			size := measure(t, w)
			img := golden.Capture(t, goldenSize, onPage(sc.p, w))
			fill := badge.Fill(sc.p, st.status)
			page := sc.p.ControlBackground
			mid := size.Y / 2

			// Inside the left padding, where only the fill can be.
			for _, dx := range []int{0, pad - 1} {
				if got := badgePixel(t, img, dx, mid); !sameColour(got, fill) {
					t.Errorf("%s %s: the pixel %d in from the badge's left edge is %v, want the fill %v",
						sc.name, st.label, dx, got, fill)
				}
			}
			// Outside it, on both sides, where only the page can be.
			if got := badgePixel(t, img, -1, mid); !sameColour(got, page) {
				t.Errorf("%s %s: the pixel before the badge's left edge is %v, want the page %v — the fill overruns the box the badge reported",
					sc.name, st.label, got, page)
			}
			if got := badgePixel(t, img, size.X, mid); !sameColour(got, page) {
				t.Errorf("%s %s: the pixel after the badge's right edge is %v, want the page %v — the fill overruns the box the badge reported",
					sc.name, st.label, got, page)
			}
			// The corner is cut, which is the silhouette half of telling a
			// badge from a chip: a square fill here would be the other one.
			if got := badgePixel(t, img, 0, 0); sameColour(got, fill) {
				t.Errorf("%s %s: the badge's top-left pixel is the fill — the fill is not rounded",
					sc.name, st.label)
			}
		}
	}
}

// TestAGlyphBadgeStandsBare is the exception the ruling carved out: the
// invariant is that hue is never the badge's only channel, and a sign already
// carries its meaning in its shape, so a glyph badge wears no fill and no
// padding. Its whole box is the page it stands on, plus the sign.
func TestAGlyphBadgeStandsBare(t *testing.T) {
	shaper := defaultShaper(t)
	style := badgeStyle()
	for _, sc := range goldenSchemes {
		w := badge.Render(shaper, "", check, badge.Success,
			sc.p, tokens.Spacing, tokens.Radius, style, badge.RenderState{})
		size := measure(t, w)
		if want := int(style.LineHeight); size.X != want || size.Y != want {
			t.Errorf("%s: a glyph badge measured %v, want the %d dp line box square — a fill or a padding term has crept in",
				sc.name, size, want)
		}
		img := golden.Capture(t, goldenSize, onPage(sc.p, w))
		page := sc.p.ControlBackground
		for _, p := range []image.Point{{X: 0, Y: 0}, {X: size.X - 1, Y: 0}, {X: 0, Y: size.Y - 1}} {
			if got := badgePixel(t, img, p.X, p.Y); !sameColour(got, page) {
				t.Errorf("%s: the glyph badge's corner pixel %v is %v, want the page %v — a bare badge has grown a fill",
					sc.name, p, got, page)
			}
		}
	}
}

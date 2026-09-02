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
// Its ink spans most of the box and is centred on it, both deliberately: the
// badge reserves the box, so a sign that under-fills it reads as a gap in the
// line, and one whose ink is not centred on the box sits off the baseline the
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
// drawn at the canvas origin has the image edge on two sides, and an image
// framed that way cannot show whether the inline box the badge reported is the
// ink it drew.
const goldenInset = 12

// onLevel paints the whole canvas in the fill of the surface the badge stands
// on and draws w inset inside it. That surface is not decoration here: every
// colour a badge resolves is derived against it and against nothing
// else, so a badge captured over the headless window's own clear colour is a
// badge whose derivation cannot be judged.
func onLevel(c tokens.ColorTokens, level tokens.ElevationLevel, w layout.Widget) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		paint.FillShape(gtx.Ops, c.SurfaceAt(level), clip.Rect{Max: gtx.Constraints.Max}.Op())
		return layout.UniformInset(unit.Dp(goldenInset)).Layout(gtx, w)
	}
}

// row lays widgets out on one line with the S4 stop between them. A badge pads
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

// The three surfaces a badge is put on in practice: the content paper, the
// chrome furniture a toolbar band is, and a dialog.
var goldenLevels = []struct {
	name  string
	level tokens.ElevationLevel
}{
	{"paper", tokens.Level0},
	{"floor", tokens.LevelFloor},
	{"dialog", tokens.Level2},
}

var goldenSchemes = []struct {
	name   string
	colors tokens.ColorTokens
}{
	{"light", tokens.DefaultLight},
	{"dark", tokens.DefaultDark},
}

// goldenVariants is the whole vocabulary, each labelled with its own name so
// the row reads without a caption under it.
var goldenVariants = []struct {
	label string
	v     badge.Variant
}{
	{"Neutral", badge.Neutral},
	{"Success", badge.Success},
	{"Warning", badge.Warning},
	{"Error", badge.Error},
	{"Info", badge.Info},
}

// goldenSize is a canvas comfortably larger than a row of badges, so the
// stored image carries the surface around the words as well as the words.
var goldenSize = image.Pt(420, 44)

// badgeStyle is the role a Comfortable badge speaks in, asked of the component
// rather than named, so a test cannot claim a size the badge does not draw.
func badgeStyle() tokens.TextStyle {
	return badge.Style(tokens.DefaultTypography, tokens.Comfortable)
}

// TestBadgeGoldenOnEveryLevel records the five variants side by side, in both
// schemes, on each of the three surfaces. Six images, and between them they
// are the claim the package doc makes: every colour is derived against the
// surface, so the same five words wear five different containers on three.
func TestBadgeGoldenOnEveryLevel(t *testing.T) {
	shaper := defaultShaper(t)
	for _, sc := range goldenSchemes {
		for _, g := range goldenLevels {
			name := "badge-" + sc.name + "-" + g.name
			t.Run(name, func(t *testing.T) {
				ws := make([]layout.Widget, 0, len(goldenVariants))
				for _, va := range goldenVariants {
					ws = append(ws, badge.Render(shaper, va.label, nil, va.v,
						sc.colors, tokens.Spacing, tokens.Radius, badgeStyle(),
						badge.RenderState{Level: g.level}))
				}
				golden.Render(t, name, goldenSize, onLevel(sc.colors, g.level, row(ws...)))
			})
		}
	}
}

// TestUtterancesGolden records the three things a badge can say, in one
// variant, so the images show what a single anatomy means: a word, a count and
// a sign at the same weight, in the same colour, on the same line.
func TestUtterancesGolden(t *testing.T) {
	shaper := defaultShaper(t)
	for _, sc := range goldenSchemes {
		name := "badge-" + sc.name + "-utterances"
		t.Run(name, func(t *testing.T) {
			ws := []layout.Widget{
				badge.Render(shaper, "Popular", nil, badge.Info,
					sc.colors, tokens.Spacing, tokens.Radius, badgeStyle(), badge.RenderState{}),
				badge.Render(shaper, "128", nil, badge.Info,
					sc.colors, tokens.Spacing, tokens.Radius, badgeStyle(), badge.RenderState{}),
				badge.Render(shaper, "", check, badge.Info,
					sc.colors, tokens.Spacing, tokens.Radius, badgeStyle(), badge.RenderState{}),
				badge.Render(shaper, "Verified", check, badge.Info,
					sc.colors, tokens.Spacing, tokens.Radius, badgeStyle(), badge.RenderState{}),
			}
			golden.Render(t, name, goldenSize, onLevel(sc.colors, tokens.Level0, row(ws...)))
		})
	}
}

// TestDismissGolden records the close mark through the states the pointer puts
// it in. The mark walks its own region toward the ramp's 900
// end, so the three tiles show it coming forward — darkening on the light
// paper and lightening on the dark one.
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
					nil, sc.colors, tokens.Spacing, tokens.Radius, badgeStyle(), st.s))
			}
			golden.Render(t, name, goldenSize, onLevel(sc.colors, tokens.Level0, row(ws...)))
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
	ws := make([]layout.Widget, 0, len(goldenVariants))
	for _, va := range goldenVariants {
		ws = append(ws, badge.Render(shaper, va.label, nil, va.v,
			tokens.DefaultLight, tokens.Spacing, tokens.Radius, style, badge.RenderState{}))
	}
	golden.Render(t, "badge-light-compact", goldenSize,
		onLevel(tokens.DefaultLight, tokens.Level0, row(ws...)))
}

// measure lays a widget out at one pixel per dp in a generous box and reports
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
					tokens.DefaultLight, tokens.Spacing, tokens.Radius, style, badge.RenderState{}))
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
			tokens.DefaultLight, tokens.Spacing, tokens.Radius, style, badge.RenderState{}))
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
		t.Error("both densities speak at the same type size: density does not reach the badge")
	}
}

// TestBadgeIsSizedToItsContent: a badge is a run of text and a run of text does
// not stretch. A badge that filled its box would be a banner.
func TestBadgeIsSizedToItsContent(t *testing.T) {
	shaper := defaultShaper(t)
	render := func(label string) layout.Widget {
		return badge.Render(shaper, label, nil, badge.Neutral,
			tokens.DefaultLight, tokens.Spacing, tokens.Radius, badgeStyle(), badge.RenderState{})
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
			tokens.DefaultLight, tokens.Spacing, tokens.Radius, style, badge.RenderState{})
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
		tokens.DefaultLight, tokens.Spacing, tokens.Radius, style, badge.RenderState{}))
	dismissible := measure(t, badge.RenderDismissible(shaper, "Filtered", nil, badge.Neutral,
		nil, tokens.DefaultLight, tokens.Spacing, tokens.Radius, style, badge.RenderState{}))
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
				nil, sc.colors, tokens.Spacing, tokens.Radius, badgeStyle(), s)
			return golden.Capture(t, goldenSize, onLevel(sc.colors, tokens.Level0, w))
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
			tokens.DefaultLight, tokens.Spacing, tokens.Radius, style, s)
		return golden.Capture(t, goldenSize, onLevel(tokens.DefaultLight, tokens.Level0, w))
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
		tokens.DefaultLight, tokens.Spacing, tokens.Radius, style, badge.RenderState{}))
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
		tokens.DefaultLight, tokens.Spacing, tokens.Radius, style, badge.RenderState{}))
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

// badgePixel samples one pixel of a badge captured over a surface, addressed
// from the badge's own top-left corner rather than the image's.
func badgePixel(t *testing.T, img *image.RGBA, dx, dy int) color.NRGBA {
	t.Helper()
	c := img.RGBAAt(goldenInset+dx, goldenInset+dy)
	return color.NRGBA{R: c.R, G: c.G, B: c.B, A: c.A}
}

// TestAWordedBadgeWearsItsContainer is the anatomy in pixels: the fill is
// there, it is the colour the derivation answers with, it is inset from the
// label by the padding stop, it stops at the badge's own reported edge, and
// its corner is cut.
//
// Sampled rather than diffed because what is being asserted is which colour
// landed where, and a pixel count cannot say that.
func TestAWordedBadgeWearsItsContainer(t *testing.T) {
	shaper := defaultShaper(t)
	style := badgeStyle()
	pad := int(tokens.Spacing.S2)
	for _, sc := range goldenSchemes {
		for _, va := range goldenVariants {
			w := badge.Render(shaper, va.label, nil, va.v,
				sc.colors, tokens.Spacing, tokens.Radius, style, badge.RenderState{})
			size := measure(t, w)
			img := golden.Capture(t, goldenSize, onLevel(sc.colors, tokens.Level0, w))
			fill := badge.Fill(sc.colors, va.v, tokens.Level0)
			surface := sc.colors.SurfaceAt(tokens.Level0)
			mid := size.Y / 2

			// Inside the left padding, where only the fill can be.
			for _, dx := range []int{0, pad - 1} {
				if got := badgePixel(t, img, dx, mid); got != fill {
					t.Errorf("%s %s: the pixel %d in from the badge's left edge is %v, want the fill %v",
						sc.name, va.label, dx, got, fill)
				}
			}
			// Outside it, on both sides, where only that surface can be.
			if got := badgePixel(t, img, -1, mid); got != surface {
				t.Errorf("%s %s: the pixel before the badge's left edge is %v, want the surface %v — the fill overruns the box the badge reported",
					sc.name, va.label, got, surface)
			}
			if got := badgePixel(t, img, size.X, mid); got != surface {
				t.Errorf("%s %s: the pixel after the badge's right edge is %v, want the surface %v — the fill overruns the box the badge reported",
					sc.name, va.label, got, surface)
			}
			// The corner is cut, which is the silhouette half of telling a
			// badge from a chip: a square fill here would be the other one.
			if got := badgePixel(t, img, 0, 0); got == fill {
				t.Errorf("%s %s: the badge's top-left pixel is the fill — the container is not rounded",
					sc.name, va.label)
			}
		}
	}
}

// TestAGlyphBadgeStandsBare is the exception the ruling carved out: the
// invariant is that hue is never the badge's only channel, and a sign already
// carries its meaning in its shape, so a glyph badge wears no container and no
// padding. Its whole box is the surface it stands on, plus the sign.
func TestAGlyphBadgeStandsBare(t *testing.T) {
	shaper := defaultShaper(t)
	style := badgeStyle()
	for _, sc := range goldenSchemes {
		w := badge.Render(shaper, "", check, badge.Success,
			sc.colors, tokens.Spacing, tokens.Radius, style, badge.RenderState{})
		size := measure(t, w)
		if want := int(style.LineHeight); size.X != want || size.Y != want {
			t.Errorf("%s: a glyph badge measured %v, want the %d dp line box square — a fill or a padding term has crept in",
				sc.name, size, want)
		}
		img := golden.Capture(t, goldenSize, onLevel(sc.colors, tokens.Level0, w))
		surface := sc.colors.SurfaceAt(tokens.Level0)
		for _, p := range []image.Point{{X: 0, Y: 0}, {X: size.X - 1, Y: 0}, {X: 0, Y: size.Y - 1}} {
			if got := badgePixel(t, img, p.X, p.Y); got != surface {
				t.Errorf("%s: the glyph badge's corner pixel %v is %v, want the surface %v — a bare badge has grown a fill",
					sc.name, p, got, surface)
			}
		}
	}
}

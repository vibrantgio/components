package chip_test

import (
	"image"
	"image/color"
	"testing"

	"gioui.org/f32"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"

	"github.com/vibrantgio/components/chip"
	"github.com/vibrantgio/components/golden"
	"github.com/vibrantgio/theme/tokens"
)

// chevron is the deterministic mark the chip goldens draw: a downward chevron
// built from two clip.Stroke lines in a sizePx×sizePx box. It is the glyph the
// chip this component replaces carried, and being vector rather than font or
// SVG rasterisation it keeps the stored images stable.
//
// Its ink is centred on the box and spans most of it, both deliberately. A
// painter that draws a small figure in the middle of the box it was given
// leaves slack the chip cannot see — the chip reserves the box, so a mark that
// under-fills it reads as extra trailing padding — and one whose ink is not
// centred on the box drops below the label, because the box is what the chip
// centres. The first draft did both and a reviewer handed the rendering caught
// both: "the chevron sits 2px low" and "the right side has visibly more air".
// Neither was the component; both were this fixture, and the contract Glyph
// states — fill the box — is what they were breaking.
func chevron(gtx layout.Context, sizePx int, col color.NRGBA) {
	w := float32(sizePx)
	stroke := float32(gtx.Dp(unit.Dp(1.5)))
	if stroke < 1 {
		stroke = 1
	}
	var p clip.Path
	p.Begin(gtx.Ops)
	p.MoveTo(f32.Pt(w*0.12, w*0.34))
	p.LineTo(f32.Pt(w*0.5, w*0.72))
	p.LineTo(f32.Pt(w*0.88, w*0.34))
	paint.FillShape(gtx.Ops, col, clip.Stroke{Path: p.End(), Width: stroke}.Op())
}

// defaultShaper returns the shaper every golden here draws with: the default
// typography's faces pinned, system fonts off, so the stored images are the
// same on every machine. See components/AGENTS.md.
func defaultShaper(t *testing.T) *text.Shaper {
	t.Helper()
	return tokens.DefaultTypography.DeterministicShaper()
}

// onStorey paints the whole canvas in the fill of the storey the chip is
// standing on and draws w inset inside it, and it has to do both.
//
// The ground, because a chip's whole subject is how it separates from it:
// against the headless window's own clear colour, a correct chip and one that
// resolved its fill from the wrong storey look identical. The inset, because a
// chip drawn at the canvas origin has the host on two sides and the image edge
// on the other two, and an image framed that way cannot show whether anything
// — a ring, a shadow, a stray half-pixel of rim — spills outside the box the
// chip reported. A reviewer handed the first set said so: for a focus-state
// review that is the wrong framing. Every stored image here has ground on all
// four sides.
const goldenInset = 12

func onStorey(c tokens.ColorTokens, level tokens.ElevationLevel, w layout.Widget) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		paint.FillShape(gtx.Ops, c.SurfaceAt(level), clip.Rect{Max: gtx.Constraints.Max}.Op())
		return layout.UniformInset(unit.Dp(goldenInset)).Layout(gtx, w)
	}
}

// The three grounds a chip actually rests on. The ladder has five rungs and
// the contrast sweep walks all of them; these are the three a chip is put on
// in practice — the content paper, the chrome furniture a toolbar band is, and
// a dialog — so they are the three whose pixels are worth storing.
var goldenGrounds = []struct {
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

// goldenSize is a canvas comfortably larger than the chip, so the stored image
// carries the host storey around the pill as well as the pill: the separation
// is the thing under test and it cannot be seen in a crop of the fill.
var goldenSize = image.Pt(220, 60)

// TestChipGoldenOnEveryGround records or diffs the resting chip in both
// schemes on each of the three grounds. Six images, and between them they are
// the claim the package doc makes about the light scheme: the pill is visible
// there because of its rim, not because of its fill.
func TestChipGoldenOnEveryGround(t *testing.T) {
	shaper := defaultShaper(t)
	for _, sc := range goldenSchemes {
		for _, g := range goldenGrounds {
			name := "chip-" + sc.name + "-" + g.name
			t.Run(name, func(t *testing.T) {
				w := chip.Render(shaper, "Claude · Opus 5", chevron,
					sc.colors, tokens.Spacing, tokens.Radius,
					tokens.DefaultTypography.LabelLarge, tokens.Comfortable,
					chip.RenderState{Ground: g.level})
				golden.Render(t, name, goldenSize, onStorey(sc.colors, g.level, w))
			})
		}
	}
}

// TestChipStateGolden records or diffs the interaction states on the paper in
// both schemes. The dark hovered and pressed images are where the walk is
// visible as a walk: three fills a rung apart under one hairline, the rim
// answering a deeper rung as the fill it encloses climbs.
func TestChipStateGolden(t *testing.T) {
	shaper := defaultShaper(t)
	states := []struct {
		name string
		s    chip.RenderState
	}{
		{"hovered", chip.RenderState{Hovered: true}},
		{"pressed", chip.RenderState{Pressed: true}},
		{"focused", chip.RenderState{Focused: true}},
	}
	for _, sc := range goldenSchemes {
		for _, st := range states {
			name := "chip-" + sc.name + "-" + st.name
			t.Run(name, func(t *testing.T) {
				w := chip.Render(shaper, "Claude · Opus 5", chevron,
					sc.colors, tokens.Spacing, tokens.Radius,
					tokens.DefaultTypography.LabelLarge, tokens.Comfortable, st.s)
				golden.Render(t, name, goldenSize, onStorey(sc.colors, tokens.Level0, w))
			})
		}
	}
}

// TestChipCompactGolden records the dense chip: Compact density in LabelMedium,
// which the package doc's table says draws at 28 dp with 12 dp of side padding
// — the hand-rolled pill this component replaces, reproduced out of the tokens.
func TestChipCompactGolden(t *testing.T) {
	shaper := defaultShaper(t)
	w := chip.Render(shaper, "Claude · Opus 5", chevron,
		tokens.DefaultLight, tokens.Spacing, tokens.Radius,
		tokens.DefaultTypography.LabelMedium, tokens.Compact, chip.RenderState{})
	golden.Render(t, "chip-light-compact", goldenSize, onStorey(tokens.DefaultLight, tokens.Level0, w))
}

// measure lays a widget out at one pixel per dp in a generous box and reports
// what it drew, which is the only honest way to ask a component its height:
// the density header's whole point is that ControlHeight is a floor and the
// drawn number is the answer.
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

// TestHeightFollowsTheDensityTable measures the four rows of the package doc's
// table. It is the gate on the one number a reader is most likely to
// re-invent: a chip is as tall as its content box needs and never shorter than
// the density says, which for Compact × LabelLarge means 32 and not 28.
func TestHeightFollowsTheDensityTable(t *testing.T) {
	shaper := defaultShaper(t)
	cases := []struct {
		name  string
		style tokens.TextStyle
		d     tokens.Density
		want  int
	}{
		{"comfortable label-large", tokens.DefaultTypography.LabelLarge, tokens.Comfortable, 36},
		{"compact label-large", tokens.DefaultTypography.LabelLarge, tokens.Compact, 32},
		{"comfortable label-medium", tokens.DefaultTypography.LabelMedium, tokens.Comfortable, 36},
		{"compact label-medium", tokens.DefaultTypography.LabelMedium, tokens.Compact, 28},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := measure(t, chip.Render(shaper, "Model", chevron,
				tokens.DefaultLight, tokens.Spacing, tokens.Radius, tc.style, tc.d,
				chip.RenderState{}))
			if got.Y != tc.want {
				t.Errorf("height = %d dp, want %d — the density table says max(ControlHeight, %g + 2×%g)",
					got.Y, tc.want, tc.style.LineHeight, tc.d.PaddingY)
			}
		})
	}
}

// TestChipIsSizedToItsContent is the other half of what makes a chip not a
// button: a button fills the width it is given and a chip does not. A chip
// that stretched would be a banner, and the difference is invisible in a
// golden recorded at an exact size.
func TestChipIsSizedToItsContent(t *testing.T) {
	shaper := defaultShaper(t)
	short := measure(t, chip.Render(shaper, "A", chevron,
		tokens.DefaultLight, tokens.Spacing, tokens.Radius,
		tokens.DefaultTypography.LabelLarge, tokens.Comfortable, chip.RenderState{}))
	long := measure(t, chip.Render(shaper, "A considerably longer summary", chevron,
		tokens.DefaultLight, tokens.Spacing, tokens.Radius,
		tokens.DefaultTypography.LabelLarge, tokens.Comfortable, chip.RenderState{}))
	if short.X >= long.X {
		t.Errorf("a one-letter chip measured %d dp wide and a long one %d: the chip is not sized to its label",
			short.X, long.X)
	}
	if short.X >= 1000 {
		t.Errorf("chip width %d dp fills the 1000 dp box it was given; a chip is sized to its content", short.X)
	}
}

// TestGlyphCostsTheChipItsOwnWidth pins the geometry the package doc states
// for the mark: the glyph is the label's line box and it is separated from the
// label by the spacing scale's S2 stop, so a chip with a mark is exactly
// LineHeight + S2 wider than the same chip without one.
func TestGlyphCostsTheChipItsOwnWidth(t *testing.T) {
	shaper := defaultShaper(t)
	style := tokens.DefaultTypography.LabelLarge
	bare := measure(t, chip.Render(shaper, "Model", nil,
		tokens.DefaultLight, tokens.Spacing, tokens.Radius, style, tokens.Comfortable,
		chip.RenderState{}))
	marked := measure(t, chip.Render(shaper, "Model", chevron,
		tokens.DefaultLight, tokens.Spacing, tokens.Radius, style, tokens.Comfortable,
		chip.RenderState{}))
	want := int(style.LineHeight) + int(tokens.Spacing.S2)
	if got := marked.X - bare.X; got != want {
		t.Errorf("the mark cost the chip %d dp, want %d (the label's %g dp line box plus the S2 %g dp gap)",
			got, want, style.LineHeight, tokens.Spacing.S2)
	}
	if bare.Y != marked.Y {
		t.Errorf("a chip with a mark is %d dp tall and one without is %d: the mark must not move the height",
			marked.Y, bare.Y)
	}
}

// TestEveryStateIsDrawnApartFromRest is the state walk stated in pixels: a
// chip is clickable and every state a pointer or a keyboard can put it in
// changes what it draws. The stored state tiles show WHAT each one looks like;
// this is the claim that none of them is the resting image.
func TestEveryStateIsDrawnApartFromRest(t *testing.T) {
	shaper := defaultShaper(t)
	frame := func(w layout.Widget) *image.RGBA {
		return golden.Capture(t, goldenSize, onStorey(tokens.DefaultLight, tokens.Level0, w))
	}
	render := func(s chip.RenderState) layout.Widget {
		return chip.Render(shaper, "Model", chevron, tokens.DefaultLight,
			tokens.Spacing, tokens.Radius, tokens.DefaultTypography.LabelLarge,
			tokens.Comfortable, s)
	}
	rest := frame(render(chip.RenderState{}))
	for _, tc := range []struct {
		name string
		s    chip.RenderState
	}{
		{"hovered", chip.RenderState{Hovered: true}},
		{"pressed", chip.RenderState{Pressed: true}},
		{"focused", chip.RenderState{Focused: true}},
	} {
		if n := golden.PixelDiff(rest, frame(render(tc.s))); n == 0 {
			t.Errorf("a %s chip is pixel-identical to a resting one", tc.name)
		}
	}
}

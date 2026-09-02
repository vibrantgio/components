package chip_test

import (
	"image"
	"image/color"
	"math"
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

// chevron is the deterministic mark the chip goldens lead with: a downward
// chevron built from two clip.Stroke lines in a sizePx×sizePx box. Being
// vector rather than font or SVG rasterisation, it keeps the stored images
// stable.
//
// Its ink is centred on the box and spans it edge to edge, both deliberately.
// A painter that draws a small figure in the middle of the box it was given
// leaves slack the chip cannot see — the chip reserves the box, so a mark that
// under-fills it reads as extra padding — and one whose ink is not centred on
// the box drops below the label, because the box is what the chip centres. The
// box is now the label's cap band, so spanning it is what puts the mark on the
// label's own line.
func chevron(gtx layout.Context, sizePx int, col color.NRGBA) {
	w := float32(sizePx)
	// Unrounded, like the marks the chip draws itself: a whole-pixel width on
	// an axis-aligned arm reads heavier than the same width on a diagonal.
	stroke := chip.MarkStrokeDp(tokens.DefaultTypography.LabelLarge) * gtx.Metric.PxPerDp
	if stroke < 1 {
		stroke = 1
	}
	var p clip.Path
	p.Begin(gtx.Ops)
	p.MoveTo(f32.Pt(0, w*0.28))
	p.LineTo(f32.Pt(w*0.5, w*0.78))
	p.LineTo(f32.Pt(w, w*0.28))
	paint.FillShape(gtx.Ops, col, clip.Stroke{Path: p.End(), Width: stroke}.Op())
}

// block fills the whole box it is given, which is what an avatar painter does:
// a picture, not a sign. It is the fixture that shows the corner-full clip an
// Input chip's leading slot applies — a painter drawing square corners into a
// round slot is exactly what that clip is for, and only a painter that fills
// the box can prove the slot is round.
func block(gtx layout.Context, sizePx int, col color.NRGBA) {
	paint.FillShape(gtx.Ops, col, clip.Rect{Max: image.Pt(sizePx, sizePx)}.Op())
}

// defaultShaper returns the shaper every golden here draws with: the default
// typography's faces pinned, system fonts off, so the stored images are the
// same on every machine. See components/AGENTS.md.
func defaultShaper(t *testing.T) *text.Shaper {
	t.Helper()
	return tokens.DefaultTypography.DeterministicShaper()
}

// onLevel paints the whole canvas in the fill of the surface the chip is
// standing on and draws w inset inside it, and it has to do both.
//
// The surface, because a resting chip carries no fill of its own: against the
// headless window's own clear colour, a correct chip and one that resolved its
// body from the wrong level look identical. The inset, because a chip drawn at
// the canvas origin has the host on two sides and the image edge on the other
// two, and an image framed that way cannot show whether anything — a ring, a
// stray half-pixel of outline — spills outside the box the chip reported.
const goldenInset = 12

func onLevel(c tokens.ColorTokens, level tokens.ElevationLevel, w layout.Widget) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		paint.FillShape(gtx.Ops, c.SurfaceAt(level), clip.Rect{Max: gtx.Constraints.Max}.Op())
		return layout.UniformInset(unit.Dp(goldenInset)).Layout(gtx, w)
	}
}

// rowGap is the air between two chips in a stored row, wide enough that the
// chips' own edges are never in doubt for the reader of the image.
const rowGap = 16

// row lays specimens out left to right with rowGap between them.
func row(ws ...layout.Widget) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		cs := make([]layout.FlexChild, 0, 2*len(ws))
		for i, w := range ws {
			if i > 0 {
				cs = append(cs, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Dimensions{Size: image.Pt(gtx.Dp(unit.Dp(rowGap)), 0)}
				}))
			}
			cs = append(cs, layout.Rigid(w))
		}
		return layout.Flex{Alignment: layout.Middle}.Layout(gtx, cs...)
	}
}

// The three surfaces a chip actually rests on. There are five levels and the
// contrast sweep walks all of them; these are the three a chip is put on in
// practice — the content paper, the chrome furniture a toolbar band is, and a
// dialog — so they are the three whose pixels are worth storing.
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

// goldenSize is a canvas comfortably larger than the row it holds, so the
// stored image carries the host surface around the chips as well as the chips:
// the separation is the thing under test and it cannot be seen in a crop.
var goldenSize = image.Pt(660, 60)

// render is the pure path at the comfortable default, which is what every
// golden here draws through.
func render(shaper *text.Shaper, label string, i chip.Purpose, icon chip.Glyph,
	c tokens.ColorTokens, s chip.RenderState,
) layout.Widget {
	return chip.Render(shaper, label, i, icon, c, tokens.Spacing, tokens.Radius,
		tokens.DefaultTypography.LabelLarge, tokens.Comfortable, s)
}

// purposeRow is the structure in one image: every purpose, and for the filter
// both of its two rests. Left to right — assist with a leading mark, a filter
// that is not selected, the same filter selected, an input chip with an avatar
// and its dismiss mark, and a label-only suggestion.
func purposeRow(shaper *text.Shaper, c tokens.ColorTokens, level tokens.ElevationLevel) layout.Widget {
	rest := chip.RenderState{Level: level}
	picked := chip.RenderState{Level: level, Selected: true}
	return row(
		render(shaper, "Assist", chip.Assist, chevron, c, rest),
		render(shaper, "Filter", chip.Filter, nil, c, rest),
		render(shaper, "Filter", chip.Filter, nil, c, picked),
		render(shaper, "Input", chip.Input, block, c, rest),
		render(shaper, "Suggestion", chip.Suggestion, nil, c, rest),
	)
}

// TestChipPurposesOnEveryLevel records or diffs the whole structure in both
// schemes on each of the three surfaces. Six images, and between them they are
// the claim the package doc makes: a resting chip is an outline and colour
// arrives only with selection.
func TestChipPurposesOnEveryLevel(t *testing.T) {
	shaper := defaultShaper(t)
	for _, sc := range goldenSchemes {
		for _, g := range goldenLevels {
			name := "chip-" + sc.name + "-" + g.name
			t.Run(name, func(t *testing.T) {
				golden.Render(t, name, goldenSize,
					onLevel(sc.colors, g.level, purposeRow(shaper, sc.colors, g.level)))
			})
		}
	}
}

// stateRow is one chip in each of the four states the pointer and the keyboard
// put it in, at the given selection.
func stateRow(shaper *text.Shaper, c tokens.ColorTokens, selected bool) layout.Widget {
	ws := make([]layout.Widget, 0, 4)
	for _, st := range []struct {
		label string
		s     chip.RenderState
	}{
		{"Rest", chip.RenderState{}},
		{"Hover", chip.RenderState{Hovered: true}},
		{"Press", chip.RenderState{Pressed: true}},
		{"Focus", chip.RenderState{Focused: true}},
	} {
		s := st.s
		s.Selected = selected
		ws = append(ws, render(shaper, st.label, chip.Filter, nil, c, s))
	}
	return row(ws...)
}

// TestChipStateGolden records or diffs the interaction states on the paper in
// both schemes and at both rests. The unselected rows are where the walk is
// visible as a walk on a chip that has no fill: a body appears under the
// pointer where there was none, and the outline answers a different step as it
// does.
func TestChipStateGolden(t *testing.T) {
	shaper := defaultShaper(t)
	for _, sc := range goldenSchemes {
		for _, sel := range []struct {
			name     string
			selected bool
		}{
			{"states", false},
			{"selected-states", true},
		} {
			name := "chip-" + sc.name + "-" + sel.name
			t.Run(name, func(t *testing.T) {
				golden.Render(t, name, goldenSize,
					onLevel(sc.colors, tokens.Level0, stateRow(shaper, sc.colors, sel.selected)))
			})
		}
	}
}

// TestChipCompactGolden records the dense chip: Compact density, where the
// chip height is 24 dp and the marks do not shrink with it.
func TestChipCompactGolden(t *testing.T) {
	shaper := defaultShaper(t)
	c := tokens.DefaultLight
	w := row(
		chip.Render(shaper, "Assist", chip.Assist, chevron, c, tokens.Spacing, tokens.Radius,
			tokens.DefaultTypography.LabelLarge, tokens.Compact, chip.RenderState{}),
		chip.Render(shaper, "Filter", chip.Filter, nil, c, tokens.Spacing, tokens.Radius,
			tokens.DefaultTypography.LabelLarge, tokens.Compact, chip.RenderState{Selected: true}),
		chip.Render(shaper, "Input", chip.Input, block, c, tokens.Spacing, tokens.Radius,
			tokens.DefaultTypography.LabelLarge, tokens.Compact, chip.RenderState{}),
	)
	golden.Render(t, "chip-light-compact", goldenSize, onLevel(c, tokens.Level0, w))
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

// TestHeightIsTheDensityChipHeight is the gate on the number a reader is most
// likely to re-invent. A chip is off the control family's padding rule: its
// height is tokens.Density.ChipHeight outright — 32 dp Comfortable, 24 dp
// Compact — and neither the type role nor the marks move it, because both fit
// inside that box at every pairing a chip is drawn at.
func TestHeightIsTheDensityChipHeight(t *testing.T) {
	shaper := defaultShaper(t)
	for _, d := range []struct {
		name string
		d    tokens.Density
		want int
	}{
		{"comfortable", tokens.Comfortable, 32},
		{"compact", tokens.Compact, 24},
	} {
		for _, style := range []struct {
			name string
			ts   tokens.TextStyle
		}{
			{"label-large", tokens.DefaultTypography.LabelLarge},
			{"label-medium", tokens.DefaultTypography.LabelMedium},
		} {
			for _, in := range []struct {
				name string
				i    chip.Purpose
				icon chip.Glyph
				s    chip.RenderState
			}{
				{"assist with a mark", chip.Assist, chevron, chip.RenderState{}},
				{"filter selected", chip.Filter, nil, chip.RenderState{Selected: true}},
				{"input with an avatar", chip.Input, block, chip.RenderState{}},
				{"suggestion bare", chip.Suggestion, nil, chip.RenderState{}},
			} {
				name := d.name + " " + style.name + " " + in.name
				t.Run(name, func(t *testing.T) {
					got := measure(t, chip.Render(shaper, "Model", in.i, in.icon,
						tokens.DefaultLight, tokens.Spacing, tokens.Radius, style.ts, d.d, in.s))
					if got.Y != d.want {
						t.Errorf("height = %d dp, want the density's chip height %d (%g − %g)",
							got.Y, d.want, d.d.ControlHeight, tokens.ChipDrop)
					}
				})
			}
		}
	}
}

// TestChipIsSizedToItsContent is what makes a chip not a button: a button fills
// the width it is given and a chip does not. A chip that stretched would be a
// banner, and the difference is invisible in a golden recorded at an exact
// size.
func TestChipIsSizedToItsContent(t *testing.T) {
	shaper := defaultShaper(t)
	short := measure(t, render(shaper, "A", chip.Assist, chevron, tokens.DefaultLight, chip.RenderState{}))
	long := measure(t, render(shaper, "A considerably longer summary", chip.Assist, chevron,
		tokens.DefaultLight, chip.RenderState{}))
	if short.X >= long.X {
		t.Errorf("a one-letter chip measured %d dp wide and a long one %d: the chip is not sized to its label",
			short.X, long.X)
	}
	if short.X >= 1000 {
		t.Errorf("chip width %d dp fills the 1000 dp box it was given; a chip is sized to its content", short.X)
	}
}

// markPx is the square a mark is drawn in at one pixel per dp: the label's cap
// band, which is what the chip reserves for every mark but the avatar.
func markPx() int { return int(math.Round(float64(chip.MarkDp(tokens.DefaultTypography.LabelLarge)))) }

// TestEachMarkCostsItsOwnSlot pins the anatomy in widths, which is the one
// place a slot can be silently dropped or silently doubled: a leading icon
// costs the cap band and the gap after it, an Input chip's avatar costs
// AvatarDp instead because its leading slot is the avatar slot, and the
// dismiss mark costs a second cap band and a second gap.
//
// The avatar also trades the text inset in front of it for its own clearance,
// so the input chip carrying one is narrower than the slot arithmetic alone
// would make it by exactly what that trade saves.
func TestEachMarkCostsItsOwnSlot(t *testing.T) {
	shaper := defaultShaper(t)
	gap := int(tokens.Spacing.S2)
	mark := markPx()
	avatarInset := (int(tokens.Comfortable.ChipHeight()) - chip.AvatarDp) / 2
	textInset := int(tokens.Comfortable.PaddingX)
	width := func(i chip.Purpose, icon chip.Glyph, s chip.RenderState) int {
		return measure(t, render(shaper, "Model", i, icon, tokens.DefaultLight, s)).X
	}
	bare := width(chip.Suggestion, nil, chip.RenderState{})
	for _, tc := range []struct {
		name string
		got  int
		want int
	}{
		{"a leading icon", width(chip.Assist, chevron, chip.RenderState{}) - bare, mark + gap},
		{"a filter's checkmark", width(chip.Filter, nil, chip.RenderState{Selected: true}) - bare, mark + gap},
		{"an input chip's dismiss mark", width(chip.Input, nil, chip.RenderState{}) - bare, mark + gap},
		{"an input chip's avatar and dismiss mark",
			width(chip.Input, block, chip.RenderState{}) - bare,
			chip.AvatarDp + mark + 2*gap - (textInset - avatarInset)},
	} {
		if tc.got != tc.want {
			t.Errorf("%s cost the chip %d dp, want %d", tc.name, tc.got, tc.want)
		}
	}
}

// TestASelectedFilterLeadsWithTheCheckmark is the slot rule where it is easiest
// to get wrong: the checkmark takes the leading slot rather than standing
// beside whatever icon the caller gave, so selecting a filter that already had
// one must not widen it.
func TestASelectedFilterLeadsWithTheCheckmark(t *testing.T) {
	shaper := defaultShaper(t)
	withIcon := measure(t, render(shaper, "Model", chip.Filter, chevron,
		tokens.DefaultLight, chip.RenderState{Selected: true})).X
	withNone := measure(t, render(shaper, "Model", chip.Filter, nil,
		tokens.DefaultLight, chip.RenderState{Selected: true})).X
	if withIcon != withNone {
		t.Errorf("a selected filter with an icon measured %d dp and one without %d: the checkmark takes the slot",
			withIcon, withNone)
	}
}

// TestEveryStateIsDrawnApartFromRest is the state walk stated in pixels: a chip
// is clickable and every state a pointer or a keyboard can put it in changes
// what it draws. The stored tiles show WHAT each one looks like; this is the
// claim that none of them is the resting image — including on a chip whose rest
// has no fill at all, where a walk that did nothing would be invisible rather
// than wrong-looking.
func TestEveryStateIsDrawnApartFromRest(t *testing.T) {
	shaper := defaultShaper(t)
	frame := func(w layout.Widget) *image.RGBA {
		return golden.Capture(t, goldenSize, onLevel(tokens.DefaultLight, tokens.Level0, w))
	}
	for _, sel := range []struct {
		name     string
		selected bool
	}{
		{"unselected", false},
		{"selected", true},
	} {
		draw := func(s chip.RenderState) layout.Widget {
			s.Selected = sel.selected
			return render(shaper, "Model", chip.Filter, nil, tokens.DefaultLight, s)
		}
		rest := frame(draw(chip.RenderState{}))
		for _, tc := range []struct {
			name string
			s    chip.RenderState
		}{
			{"hovered", chip.RenderState{Hovered: true}},
			{"pressed", chip.RenderState{Pressed: true}},
			{"focused", chip.RenderState{Focused: true}},
		} {
			if n := golden.PixelDiff(rest, frame(draw(tc.s))); n == 0 {
				t.Errorf("a %s %s chip is pixel-identical to a resting one", sel.name, tc.name)
			}
		}
	}
}

// TestSelectionIsDrawn is selection stated in pixels rather than in a
// derivation: a selected filter chip and an unselected one cannot look alike,
// because selection is the whole of what a filter chip has to say.
func TestSelectionIsDrawn(t *testing.T) {
	shaper := defaultShaper(t)
	frame := func(selected bool) *image.RGBA {
		return golden.Capture(t, goldenSize, onLevel(tokens.DefaultLight, tokens.Level0,
			render(shaper, "Model", chip.Filter, nil, tokens.DefaultLight,
				chip.RenderState{Selected: selected})))
	}
	if n := golden.PixelDiff(frame(false), frame(true)); n == 0 {
		t.Error("a selected filter chip is pixel-identical to an unselected one")
	}
}

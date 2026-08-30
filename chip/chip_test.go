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
	vgcolor "github.com/vibrantgio/theme/color"
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

// dot is the badge goldens' mark: a filled disc, centred, at half the box. A
// badge is not a disclosure affordance and must not wear a chevron —
// the same reviewer, handed a badge with one, could not tell the two faces
// apart at all. The faces do share a geometry by design; what they must not
// share is a mark that promises something one of them does not do.
func dot(gtx layout.Context, sizePx int, col color.NRGBA) {
	d := sizePx / 2
	off := (sizePx - d) / 2
	paint.FillShape(gtx.Ops, col,
		clip.Ellipse(image.Rect(off, off, off+d, off+d)).Op(gtx.Ops))
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

// TestBadgeGoldenOnEveryGround is the same six for the non-interactive face.
// Its images are stored separately rather than asserted equal to the chip's:
// the two faces share a geometry and are allowed to diverge in treatment, and
// a stored pair is what would show it if they ever did.
func TestBadgeGoldenOnEveryGround(t *testing.T) {
	shaper := defaultShaper(t)
	for _, sc := range goldenSchemes {
		for _, g := range goldenGrounds {
			name := "badge-" + sc.name + "-" + g.name
			t.Run(name, func(t *testing.T) {
				w := chip.RenderBadge(shaper, "3 unread", dot,
					sc.colors, tokens.Spacing, tokens.Radius,
					tokens.DefaultTypography.LabelLarge, tokens.Comfortable,
					g.level)
				golden.Render(t, name, goldenSize, onStorey(sc.colors, g.level, w))
			})
		}
	}
}

// TestAnchorGoldenOnEveryGround is the same six for the pull-down anchor: the
// chip's fill, rim and inks at the button's rounded-rect corner, with the
// chevron the component draws itself. Stored beside the chip's own six rather
// than asserted against them, because the pair of images is what shows the
// corner and the mark are the ONLY things that differ.
func TestAnchorGoldenOnEveryGround(t *testing.T) {
	shaper := defaultShaper(t)
	for _, sc := range goldenSchemes {
		for _, g := range goldenGrounds {
			name := "anchor-" + sc.name + "-" + g.name
			t.Run(name, func(t *testing.T) {
				w := chip.RenderAnchor(shaper, "OpenAI · gpt-5.5",
					sc.colors, tokens.Spacing, tokens.Radius,
					tokens.DefaultTypography.LabelLarge, tokens.Comfortable,
					chip.RenderState{Ground: g.level})
				golden.Render(t, name, goldenSize, onStorey(sc.colors, g.level, w))
			})
		}
	}
}

// TestAnchorStateGolden records the anchor's walk on the paper in both
// schemes. The focused image is the one worth storing twice over: the ring
// replaces the rim at the anchor's corner too, so a ring drawn at the pill's
// Full radius over a rounded-rect fill would show here as a halo that misses
// its corners.
func TestAnchorStateGolden(t *testing.T) {
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
			name := "anchor-" + sc.name + "-" + st.name
			t.Run(name, func(t *testing.T) {
				w := chip.RenderAnchor(shaper, "OpenAI · gpt-5.5",
					sc.colors, tokens.Spacing, tokens.Radius,
					tokens.DefaultTypography.LabelLarge, tokens.Comfortable, st.s)
				golden.Render(t, name, goldenSize, onStorey(sc.colors, tokens.Level0, w))
			})
		}
	}
}

// TestAnchorIsTheChipWithADifferentCornerAndMark holds the seam the anchor
// face was cut at. Everything the chip contributes — the measured fill, the
// two-sided rim, the walked inks, the density's height — is shared code, so
// the two faces may differ in their corner and their mark and must not differ
// anywhere else. The heights are the check that says so without a stored
// image: a face that reached for its own padding or its own line box would
// draw a different box.
func TestAnchorIsTheChipWithADifferentCornerAndMark(t *testing.T) {
	shaper := defaultShaper(t)
	for _, d := range []struct {
		name string
		d    tokens.Density
		ts   tokens.TextStyle
	}{
		{"comfortable", tokens.Comfortable, tokens.DefaultTypography.LabelLarge},
		{"compact", tokens.Compact, tokens.DefaultTypography.LabelMedium},
	} {
		t.Run(d.name, func(t *testing.T) {
			anchor := measure(t, chip.RenderAnchor(shaper, "Model", tokens.DefaultLight,
				tokens.Spacing, tokens.Radius, d.ts, d.d, chip.RenderState{}))
			pill := measure(t, chip.Render(shaper, "Model", chevron, tokens.DefaultLight,
				tokens.Spacing, tokens.Radius, d.ts, d.d, chip.RenderState{}))
			if anchor.Y != pill.Y {
				t.Errorf("anchor height %d, chip height %d: the faces share a geometry and must share this",
					anchor.Y, pill.Y)
			}
			// The mark is the only width difference, and it is the platform's
			// ratio of the control height rather than the label's line box.
			if anchor.X == pill.X {
				t.Errorf("anchor and chip both %d wide: the anchor's mark is the "+
					"control's own ratio and the chip's is the line box, so they cannot match by construction", anchor.X)
			}
		})
	}
}

// TestAnchorMarkIsSteadyAcrossTheWalk is the platform ruling written down
// where a future change cannot quietly undo it: a pull-down anchor's chevron
// says "a menu opens below this" and never "this is open", so nothing about
// the pointer's state may move it. The face offers no open flag to flip — that
// is the structural half — and this is the drawn half: the box the anchor
// reports is the same box in all four states, so the mark neither grows nor
// shifts under the pointer while the fill walks beneath it.
func TestAnchorMarkIsSteadyAcrossTheWalk(t *testing.T) {
	shaper := defaultShaper(t)
	size := func(s chip.RenderState) image.Point {
		return measure(t, chip.RenderAnchor(shaper, "OpenAI · gpt-5.5", tokens.DefaultDark,
			tokens.Spacing, tokens.Radius, tokens.DefaultTypography.LabelLarge,
			tokens.Comfortable, s))
	}
	rest := size(chip.RenderState{})
	for _, tc := range []struct {
		name string
		s    chip.RenderState
	}{
		{"hovered", chip.RenderState{Hovered: true}},
		{"pressed", chip.RenderState{Pressed: true}},
		{"focused", chip.RenderState{Focused: true}},
	} {
		if got := size(tc.s); got != rest {
			t.Errorf("%s anchor measures %v, resting %v: the anchor's mark is fixed and its box must not move",
				tc.name, got, rest)
		}
	}
}

// TestAnchorChevronReachesTheGraphicFloor is the contrast sweep's extension to
// the anchor face, and it is a PIXEL measurement rather than another pass over
// the derivations. chip_contrast_test.go already sweeps the five colours on
// every storey and in every state, and the anchor changes none of them — it
// shares Fill, Rim and Ink with the chip, so that whole sweep covers this face
// as it stands.
//
// What it cannot cover is the mark. The chevron is a 1.5 dp DIAGONAL stroke,
// and a diagonal hairline is antialiased: the colour Ink derives may clear the
// graphic floor while no pixel actually drawn does. The platform has the same
// problem and answers it by drawing diagonals heavier than its axis-aligned
// strokes (ADR-019 measured 1.44 px against 1.26 px at 16 pt), and Gio
// composites in linear light where CoreGraphics composites in encoded sRGB,
// which costs a hairline more ink still. So the claim worth holding is about
// the drawn pixels: somewhere in the mark, in every scheme and every state,
// the chevron reaches the floor it owes.
func TestAnchorChevronReachesTheGraphicFloor(t *testing.T) {
	shaper := defaultShaper(t)
	states := []struct {
		name string
		s    chip.RenderState
	}{
		{"at rest", chip.RenderState{}},
		{"hovered", chip.RenderState{Hovered: true}},
		{"pressed", chip.RenderState{Pressed: true}},
		{"focused", chip.RenderState{Focused: true}},
	}
	for _, sc := range goldenSchemes {
		for _, g := range goldenGrounds {
			for _, st := range states {
				name := sc.name + " " + g.name + " " + st.name
				t.Run(name, func(t *testing.T) {
					s := st.s
					s.Ground = g.level
					w := chip.RenderAnchor(shaper, "OpenAI · gpt-5.5",
						sc.colors, tokens.Spacing, tokens.Radius,
						tokens.DefaultTypography.LabelLarge, tokens.Comfortable, s)
					img := golden.Capture(t, goldenSize, onStorey(sc.colors, g.level, w))
					fill := chip.Fill(sc.colors, g.level, stateOf(s))

					// The mark's own column: the trailing padding's width in
					// from the pill's trailing edge, which is where draw puts
					// it. Measured off the drawn box rather than assumed.
					box := measure(t, w)
					pad := int(tokens.Comfortable.PaddingX)
					x0 := goldenInset + box.X - pad - int(float32(tokens.Comfortable.ControlHeight)*9/29) - 2
					x1 := goldenInset + box.X - pad + 2
					best := 0.0
					for y := goldenInset; y < goldenInset+box.Y; y++ {
						for x := max(x0, 0); x < min(x1, img.Bounds().Dx()); x++ {
							r, gg, b, _ := img.At(x, y).RGBA()
							px := color.NRGBA{R: uint8(r >> 8), G: uint8(gg >> 8), B: uint8(b >> 8), A: 255}
							if v := vgcolor.ContrastRatio(px, fill); v > best {
								best = v
							}
						}
					}
					t.Logf("%s: the mark's heaviest pixel reaches %.2f:1 on the fill", name, best)
					if best < tokens.GraphicFloor {
						t.Errorf("%s: the chevron's heaviest drawn pixel reaches only %.2f:1 on the fill, want at least %.1f:1",
							name, best, tokens.GraphicFloor)
					}
				})
			}
		}
	}
}

// stateOf is RenderState.state() from outside the package: the walk the drawn
// fill took, so the assertion above measures against the fill actually painted
// rather than the resting one.
func stateOf(s chip.RenderState) tokens.State {
	switch {
	case s.Pressed:
		return tokens.StatePressed
	case s.Hovered:
		return tokens.StateHover
	}
	return tokens.StateNormal
}

// TestChipStateGolden records or diffs the interaction states on the paper in
// both schemes. The dark hovered and pressed images are where the package
// doc's rimless case is actually visible: the fill has walked far enough to
// carry its own edge and the hairline is gone.
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

// TestBadgeDrawsNoStateWalkAndNoRing is the difference between the two faces,
// stated in pixels: the badge takes a ground and not a state, so nothing a
// pointer or a keyboard does to a chip has any effect on it. The chip's own
// hovered and focused frames are compared against its resting one in the same
// breath, because "the badge does not move" only means something if the chip
// does.
func TestBadgeDrawsNoStateWalkAndNoRing(t *testing.T) {
	shaper := defaultShaper(t)
	frame := func(w layout.Widget) *image.RGBA {
		return golden.Capture(t, goldenSize, onStorey(tokens.DefaultLight, tokens.Level0, w))
	}
	render := func(s chip.RenderState) layout.Widget {
		return chip.Render(shaper, "Model", chevron, tokens.DefaultLight,
			tokens.Spacing, tokens.Radius, tokens.DefaultTypography.LabelLarge,
			tokens.Comfortable, s)
	}
	badge := frame(chip.RenderBadge(shaper, "Model", chevron, tokens.DefaultLight,
		tokens.Spacing, tokens.Radius, tokens.DefaultTypography.LabelLarge,
		tokens.Comfortable, tokens.Level0))
	rest := frame(render(chip.RenderState{}))
	if n := golden.PixelDiff(badge, rest); n != 0 {
		t.Errorf("the badge differs from a resting chip in %d pixels; the two faces share one geometry", n)
	}
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

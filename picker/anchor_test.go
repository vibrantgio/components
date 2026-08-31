package picker_test

import (
	"image"
	"image/color"
	"testing"

	"gioui.org/f32"
	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"

	"github.com/vibrantgio/components/chip"
	"github.com/vibrantgio/components/golden"
	"github.com/vibrantgio/components/picker"
	vgcolor "github.com/vibrantgio/theme/color"
	"github.com/vibrantgio/theme/tokens"
)

// onStorey paints the whole canvas in the fill of the storey the anchor is
// standing on and draws w inset inside it, and it has to do both.
//
// The ground, because the anchor's whole subject is how it separates from it:
// against the headless window's own clear colour, a correct anchor and one that
// resolved its fill from the wrong storey look identical. The inset, because a
// control drawn at the canvas origin has the host on two sides and the image
// edge on the other two, and an image framed that way cannot show whether
// anything — a ring, a shadow, a stray half-pixel of rim — spills outside the
// box the control reported. Every stored image here has ground on all four
// sides.
const goldenInset = 12

func onStorey(c tokens.ColorTokens, level tokens.ElevationLevel, w layout.Widget) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		paint.FillShape(gtx.Ops, c.SurfaceAt(level), clip.Rect{Max: gtx.Constraints.Max}.Op())
		return layout.UniformInset(unit.Dp(goldenInset)).Layout(gtx, w)
	}
}

// The three grounds a chrome-register trigger actually rests on: the content
// paper, the chrome furniture a toolbar band is, and a dialog.
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

// goldenSize is a canvas comfortably larger than the anchor, so the stored
// image carries the host storey around the control as well as the control:
// the separation is the thing under test and it cannot be seen in a crop of
// the fill.
var goldenSize = image.Pt(220, 60)

// anchorValue is what every stored anchor image says. It is a two-part model
// name because that is the value the chrome register's trigger carries in
// practice, and its width is what the images were recorded at.
const anchorValue = "OpenAI · gpt-5.5"

// anchor is RenderAnchor at the default spacing, radius and comfortable
// density — the resolved tokens every measurement and image below draws with.
func anchor(t *testing.T, c tokens.ColorTokens, s picker.AnchorState) layout.Widget {
	t.Helper()
	return picker.RenderAnchor(defaultShaper(t), anchorValue, c,
		tokens.Spacing, tokens.Radius, tokens.DefaultTypography.LabelLarge,
		tokens.Comfortable, s)
}

// TestAnchorGoldenOnEveryGround records or diffs the resting anchor in both
// schemes on each of the three grounds. Six images, and between them they are
// the claim the chip family's geometry makes about the light scheme: the
// control is visible there because of its rim, not because of its fill.
func TestAnchorGoldenOnEveryGround(t *testing.T) {
	for _, sc := range goldenSchemes {
		for _, g := range goldenGrounds {
			name := "anchor-" + sc.name + "-" + g.name
			t.Run(name, func(t *testing.T) {
				w := anchor(t, sc.colors, picker.AnchorState{Ground: g.level})
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
	states := []struct {
		name string
		s    picker.AnchorState
	}{
		{"hovered", picker.AnchorState{Hovered: true}},
		{"pressed", picker.AnchorState{Pressed: true}},
		{"focused", picker.AnchorState{Focused: true}},
	}
	for _, sc := range goldenSchemes {
		for _, st := range states {
			name := "anchor-" + sc.name + "-" + st.name
			t.Run(name, func(t *testing.T) {
				w := anchor(t, sc.colors, st.s)
				golden.Render(t, name, goldenSize, onStorey(sc.colors, tokens.Level0, w))
			})
		}
	}
}

// TestAnchorIsTheChipWithADifferentCornerAndMark holds the seam the anchor was
// cut at. Everything components/chip contributes — the measured fill, the
// two-sided rim, the walked inks, the density's height — is shared code, so the
// two faces may differ in their corner and their mark and must not differ
// anywhere else. The heights are the check that says so without a stored image:
// a face that reached for its own padding or its own line box would draw a
// different box.
func TestAnchorIsTheChipWithADifferentCornerAndMark(t *testing.T) {
	shaper := defaultShaper(t)
	box := image.Pt(1000, 1000)
	for _, d := range []struct {
		name string
		d    tokens.Density
		ts   tokens.TextStyle
	}{
		{"comfortable", tokens.Comfortable, tokens.DefaultTypography.LabelLarge},
		{"compact", tokens.Compact, tokens.DefaultTypography.LabelMedium},
	} {
		t.Run(d.name, func(t *testing.T) {
			trigger := measure(t, box, picker.RenderAnchor(shaper, "Model", tokens.DefaultLight,
				tokens.Spacing, tokens.Radius, d.ts, d.d, picker.AnchorState{})).Size
			pill := measure(t, box, chip.Render(shaper, "Model", chevron, tokens.DefaultLight,
				tokens.Spacing, tokens.Radius, d.ts, d.d, chip.RenderState{})).Size
			if trigger.Y != pill.Y {
				t.Errorf("anchor height %d, chip height %d: the faces share a geometry and must share this",
					trigger.Y, pill.Y)
			}
			// The mark is the only width difference, and it is the platform's
			// ratio of the control height rather than the label's line box.
			if trigger.X == pill.X {
				t.Errorf("anchor and chip both %d wide: the anchor's mark is the "+
					"control's own ratio and the chip's is the line box, so they cannot match by construction", trigger.X)
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
	box := image.Pt(1000, 1000)
	size := func(s picker.AnchorState) image.Point {
		return measure(t, box, anchor(t, tokens.DefaultDark, s)).Size
	}
	rest := size(picker.AnchorState{})
	for _, tc := range []struct {
		name string
		s    picker.AnchorState
	}{
		{"hovered", picker.AnchorState{Hovered: true}},
		{"pressed", picker.AnchorState{Pressed: true}},
		{"focused", picker.AnchorState{Focused: true}},
	} {
		if got := size(tc.s); got != rest {
			t.Errorf("%s anchor measures %v, resting %v: the anchor's mark is fixed and its box must not move",
				tc.name, got, rest)
		}
	}
}

// TestAnchorChevronReachesTheGraphicFloor is the contrast sweep's extension to
// the anchor, and it is a PIXEL measurement rather than another pass over the
// derivations. components/chip's contrast sweep already walks the five colours
// on every storey and in every state, and the anchor changes none of them — it
// shares Fill, Rim and Ink with the chip, so that whole sweep covers this
// control as it stands.
//
// What it cannot cover is the mark. The chevron is a 1.5 dp DIAGONAL stroke,
// and a diagonal hairline is antialiased: the colour Ink derives may clear the
// graphic floor while no pixel actually drawn does. The platform has the same
// problem and answers it by drawing diagonals heavier than its axis-aligned
// strokes (1.44 px against 1.26 px at 16 pt, measured off the stored macOS
// reference), and Gio composites in linear light where CoreGraphics composites
// in encoded sRGB, which costs a hairline more ink still. So the claim worth
// holding is about the drawn pixels: somewhere in the mark, in every scheme and
// every state, the chevron reaches the floor it owes.
func TestAnchorChevronReachesTheGraphicFloor(t *testing.T) {
	states := []struct {
		name string
		s    picker.AnchorState
	}{
		{"at rest", picker.AnchorState{}},
		{"hovered", picker.AnchorState{Hovered: true}},
		{"pressed", picker.AnchorState{Pressed: true}},
		{"focused", picker.AnchorState{Focused: true}},
	}
	for _, sc := range goldenSchemes {
		for _, g := range goldenGrounds {
			for _, st := range states {
				name := sc.name + " " + g.name + " " + st.name
				t.Run(name, func(t *testing.T) {
					s := st.s
					s.Ground = g.level
					w := anchor(t, sc.colors, s)
					img := golden.Capture(t, goldenSize, onStorey(sc.colors, g.level, w))
					fill := chip.Fill(sc.colors, g.level, stateOf(s))

					// The mark's own column: the trailing padding's width in
					// from the control's trailing edge, which is where Draw
					// puts it. Measured off the drawn box rather than assumed.
					box := measure(t, image.Pt(1000, 1000), w).Size
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

// stateOf is AnchorState's own walk from outside the package, so the assertion
// above measures against the fill actually painted rather than the resting one.
func stateOf(s picker.AnchorState) tokens.State {
	switch {
	case s.Pressed:
		return tokens.StatePressed
	case s.Hovered:
		return tokens.StateHover
	}
	return tokens.StateNormal
}

// chevron is the deterministic mark the chip is measured against here: a
// downward chevron built from two clip.Stroke lines filling a sizePx×sizePx
// box. Being vector rather than font or SVG rasterisation, it costs the chip
// exactly its own line box and nothing that varies by machine.
//
// Its ink is centred on the box and spans most of it, both deliberately: the
// chip reserves the box, so a mark that under-fills it reads as extra trailing
// padding, and one whose ink is not centred on the box drops below the label.
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

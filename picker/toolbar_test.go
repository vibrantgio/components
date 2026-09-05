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

	"github.com/vibrantgio/components/golden"
	"github.com/vibrantgio/components/picker"
	vgcolor "github.com/vibrantgio/theme/color"
	"github.com/vibrantgio/theme/tokens"
)

// onLevel paints the whole frame in the fill of the surface the trigger is
// standing on and draws w inset inside it, and it has to do both.
//
// The host surface, because the trigger's whole subject is how it separates
// from it: against the headless window's own clear colour, a correct trigger and one that
// resolved its fill from the wrong level look identical. The inset, because a
// control drawn at the image origin has the host on two sides and the image
// edge on the other two, and an image framed that way cannot show whether
// anything — a ring, a shadow, a stray half-pixel of rim — spills outside the
// box the control reported. Every stored image here has that surface on all
// four sides.
const goldenInset = 12

func onLevel(c tokens.ColorTokens, level tokens.ElevationLevel, w layout.Widget) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		paint.FillShape(gtx.Ops, c.SurfaceAt(level), clip.Rect{Max: gtx.Constraints.Max}.Op())
		return layout.UniformInset(unit.Dp(goldenInset)).Layout(gtx, w)
	}
}

// The three surfaces a chrome-variant trigger actually rests on: the content
// surface, the chrome level a toolbar band stands at, and a dialog.
var goldenGrounds = []struct {
	name  string
	level tokens.ElevationLevel
}{
	{"paper", tokens.Level0},
	{"chrome", tokens.LevelChrome},
	{"dialog", tokens.Level2},
}

var goldenSchemes = []struct {
	name   string
	colors tokens.ColorTokens
}{
	{"light", tokens.DefaultLight},
	{"dark", tokens.DefaultDark},
}

// goldenSize is an image comfortably larger than the trigger, so the stored
// image carries the host surface around the control as well as the control:
// the separation is the thing under test and it cannot be seen in a crop of
// the fill.
var goldenSize = image.Pt(220, 60)

// toolbarValue is what every stored toolbar image says. It is a two-part model
// name because that is the value the chrome variant's trigger carries in
// practice, and its width is what the images were recorded at.
const toolbarValue = "OpenAI · gpt-5.5"

// toolbar is RenderToolbar at the default spacing, radius and comfortable
// density — the resolved tokens every measurement and image below draws with.
func toolbar(t *testing.T, c tokens.ColorTokens, s picker.ToolbarState) layout.Widget {
	t.Helper()
	return picker.RenderToolbar(defaultShaper(t), toolbarValue, c,
		tokens.Spacing, tokens.Radius, tokens.DefaultTypography.LabelLarge,
		tokens.Comfortable, s)
}

// TestToolbarGoldenOnEveryLevel records or diffs the resting trigger in both
// schemes on each of the three surfaces. Six images, and between them they are
// the claim this geometry makes about the light scheme: the
// control is visible there because of its rim, not because of its fill.
func TestToolbarGoldenOnEveryLevel(t *testing.T) {
	for _, sc := range goldenSchemes {
		for _, g := range goldenGrounds {
			name := "toolbar-" + sc.name + "-" + g.name
			t.Run(name, func(t *testing.T) {
				w := toolbar(t, sc.colors, picker.ToolbarState{Level: g.level})
				golden.Render(t, name, goldenSize, onLevel(sc.colors, g.level, w))
			})
		}
	}
}

// TestToolbarStateGolden records the trigger's walk on the paper in both
// schemes. The focused image is the one worth storing twice over: the ring
// replaces the rim at the trigger's corner too, so a ring drawn at the pill's
// Full radius over a rounded-rect fill would show here as a halo that misses
// its corners.
func TestToolbarStateGolden(t *testing.T) {
	states := []struct {
		name string
		s    picker.ToolbarState
	}{
		{"hovered", picker.ToolbarState{Hovered: true}},
		{"pressed", picker.ToolbarState{Pressed: true}},
		{"focused", picker.ToolbarState{Focused: true}},
	}
	for _, sc := range goldenSchemes {
		for _, st := range states {
			name := "toolbar-" + sc.name + "-" + st.name
			t.Run(name, func(t *testing.T) {
				w := toolbar(t, sc.colors, st.s)
				golden.Render(t, name, goldenSize, onLevel(sc.colors, tokens.Level0, w))
			})
		}
	}
}

// TestToolbarDrawsAtTheDensityTable holds the geometry the trigger takes off the
// tokens rather than off numbers of its own: the height is the density's own
// rule for a control — max(ControlHeight, line box + 2×PaddingY) — so a face
// that reached for its own padding or its own line box would draw a different
// box.
func TestToolbarDrawsAtTheDensityTable(t *testing.T) {
	shaper := defaultShaper(t)
	box := image.Pt(1000, 1000)
	for _, d := range []struct {
		name string
		d    tokens.Density
		ts   tokens.TextStyle
		want int
	}{
		{"comfortable", tokens.Comfortable, tokens.DefaultTypography.LabelLarge, 36},
		{"compact", tokens.Compact, tokens.DefaultTypography.LabelMedium, 28},
	} {
		t.Run(d.name, func(t *testing.T) {
			trigger := measure(t, box, picker.RenderToolbar(shaper, "Model", tokens.DefaultLight,
				tokens.Spacing, tokens.Radius, d.ts, d.d, picker.ToolbarState{})).Size
			if trigger.Y != d.want {
				t.Errorf("toolbar height %d, want %d — max(ControlHeight %g, %g + 2×%g)",
					trigger.Y, d.want, d.d.ControlHeight, d.ts.LineHeight, d.d.PaddingY)
			}
			if trigger.X >= box.X {
				t.Errorf("toolbar measured %d dp wide in a %d dp box: it is sized to its value", trigger.X, box.X)
			}
		})
	}
}

// TestToolbarMarkIsSteadyAcrossTheWalk is the platform ruling written down
// where a future change cannot silently undo it: a pull-down trigger's chevron
// says "a menu opens below this" and never "this is open", so nothing about
// the pointer's state may move it. The face offers no open flag to flip — that
// is the structural half — and this is the drawn half: the box the trigger
// reports is the same box in all four states, so the mark neither grows nor
// shifts under the pointer while the fill walks beneath it.
func TestToolbarMarkIsSteadyAcrossTheWalk(t *testing.T) {
	box := image.Pt(1000, 1000)
	size := func(s picker.ToolbarState) image.Point {
		return measure(t, box, toolbar(t, tokens.DefaultDark, s)).Size
	}
	rest := size(picker.ToolbarState{})
	for _, tc := range []struct {
		name string
		s    picker.ToolbarState
	}{
		{"hovered", picker.ToolbarState{Hovered: true}},
		{"pressed", picker.ToolbarState{Pressed: true}},
		{"focused", picker.ToolbarState{Focused: true}},
	} {
		if got := size(tc.s); got != rest {
			t.Errorf("%s toolbar measures %v, resting %v: the trigger's mark is fixed and its box must not move",
				tc.name, got, rest)
		}
	}
}

// TestToolbarChevronReachesTheGraphicFloor is the contrast sweep's extension to
// the toolbar trigger, and it is a PIXEL measurement rather than another pass over the
// derivations. The geometry's own sweep
// (components/internal/toolbarface) already walks the colours on every level and
// in every state, so the derivations are covered where they live.
//
// What it cannot cover is the mark. The chevron is a 1.5 dp DIAGONAL stroke,
// and a diagonal hairline is antialiased: the colour `Ink` derives may clear the
// graphic floor while no pixel actually drawn does. The platform has the same
// problem and answers it by drawing diagonals heavier than its axis-aligned
// strokes (1.44 px against 1.26 px at 16 pt, measured off the stored macOS
// reference), and Gio composites in linear light where CoreGraphics composites
// in encoded sRGB, which costs a hairline more of its strength still. So the
// claim worth holding is about the drawn pixels: somewhere in the mark, in every scheme and
// every state, the chevron reaches the floor it owes.
func TestToolbarChevronReachesTheGraphicFloor(t *testing.T) {
	states := []struct {
		name string
		s    picker.ToolbarState
	}{
		{"at rest", picker.ToolbarState{}},
		{"hovered", picker.ToolbarState{Hovered: true}},
		{"pressed", picker.ToolbarState{Pressed: true}},
		{"focused", picker.ToolbarState{Focused: true}},
	}
	for _, sc := range goldenSchemes {
		for _, g := range goldenGrounds {
			for _, st := range states {
				name := sc.name + " " + g.name + " " + st.name
				t.Run(name, func(t *testing.T) {
					s := st.s
					s.Level = g.level
					w := toolbar(t, sc.colors, s)
					img := golden.Capture(t, goldenSize, onLevel(sc.colors, g.level, w))
					fill := picker.ToolbarFill(sc.colors, g.level, stateOf(s))

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

// stateOf is ToolbarState's own walk from outside the package, so the assertion
// above measures against the fill actually painted rather than the resting one.
func stateOf(s picker.ToolbarState) tokens.State {
	switch {
	case s.Pressed:
		return tokens.StatePressed
	case s.Hovered:
		return tokens.StateHover
	}
	return tokens.StateNormal
}

// chevron is the deterministic mark the trigger is measured against here: a
// downward chevron built from two clip.Stroke lines filling a sizePx×sizePx
// box. Being vector rather than font or SVG rasterisation, it costs the trigger
// exactly its own line box and nothing that varies by machine.
//
// Its stroke is centred on the box and spans most of it, both deliberately: the
// trigger reserves the box, so a mark that under-fills it reads as extra trailing
// padding, and one whose stroke is not centred on the box drops below the label.
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

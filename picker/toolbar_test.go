package picker_test

import (
	"image"
	"image/color"
	"testing"

	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"

	"github.com/vibrantgio/components/golden"
	"github.com/vibrantgio/components/picker"
	"github.com/vibrantgio/theme/tokens"
)

// onSurface paints the whole frame in the fill of the surface the trigger is
// standing on and draws w inset inside it, and it has to do both.
//
// The host surface, because the trigger tints that surface rather than filling
// itself: at rest the chrome shows through untouched, so against the headless
// window's own clear colour a correct trigger and one that painted a fill of
// its own look identical. The inset, because a control drawn at the image
// origin has the host on two sides and the image edge on the other two, and an
// image framed that way cannot show whether anything — a ring, a shadow, a
// stray half-pixel of rim — spills outside the box the control reported. Every
// stored image here has that surface on all four sides.
const goldenInset = 12

func onSurface(fill color.NRGBA, w layout.Widget) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		paint.FillShape(gtx.Ops, fill, clip.Rect{Max: gtx.Constraints.Max}.Op())
		return layout.UniformInset(unit.Dp(goldenInset)).Layout(gtx, w)
	}
}

// The two surfaces a chrome-variant trigger actually rests on: the chrome band
// a toolbar is, and the content plane a header row inside a pane stands on.
var goldenSurfaces = []struct {
	name string
	fill func(tokens.PlatformColors) color.NRGBA
}{
	{"chrome", func(p tokens.PlatformColors) color.NRGBA { return p.SidebarMaterial }},
	{"content", func(p tokens.PlatformColors) color.NRGBA { return p.ControlBackground }},
}

var goldenSchemes = []struct {
	name string
	p    tokens.PlatformColors
}{
	{"light", tokens.PlatformLight},
	{"dark", tokens.PlatformDark},
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
func toolbar(t *testing.T, p tokens.PlatformColors, s picker.ToolbarState) layout.Widget {
	t.Helper()
	return picker.RenderToolbar(defaultShaper(t), toolbarValue, p,
		tokens.Spacing, tokens.Radius, tokens.DefaultTypography.LabelLarge,
		tokens.Comfortable, s)
}

// TestToolbarGoldenOnEverySurface records or diffs the resting trigger in both
// schemes on each of the two surfaces. Four images, and between them they are
// the claim this geometry makes about the light scheme: the control is visible
// there because of its rim, not because of a fill of its own.
func TestToolbarGoldenOnEverySurface(t *testing.T) {
	for _, sc := range goldenSchemes {
		for _, g := range goldenSurfaces {
			name := "toolbar-" + sc.name + "-" + g.name
			t.Run(name, func(t *testing.T) {
				w := toolbar(t, sc.p, picker.ToolbarState{})
				golden.Render(t, name, goldenSize, onSurface(g.fill(sc.p), w))
			})
		}
	}
}

// TestToolbarStateGolden records the trigger's states on the chrome in both
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
				w := toolbar(t, sc.p, st.s)
				golden.Render(t, name, goldenSize, onSurface(sc.p.SidebarMaterial, w))
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
		{"comfortable", tokens.Comfortable, tokens.DefaultTypography.LabelLarge, 24},
		{"compact", tokens.Compact, tokens.DefaultTypography.LabelMedium, 19},
	} {
		t.Run(d.name, func(t *testing.T) {
			trigger := measure(t, box, picker.RenderToolbar(shaper, "Model", tokens.PlatformLight,
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
		return measure(t, box, toolbar(t, tokens.PlatformDark, s)).Size
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

// TestToolbarFillIsThePlatformsOverlay pins the passthrough by name: a toolbar
// button is the one control on this platform that tints under the pointer, so
// the trigger lays nothing at all over the chrome at rest and the platform's
// own two overlays over it under the pointer and while it is held.
func TestToolbarFillIsThePlatformsOverlay(t *testing.T) {
	for _, sc := range goldenSchemes {
		t.Run(sc.name, func(t *testing.T) {
			for _, tc := range []struct {
				name  string
				state tokens.State
				want  color.NRGBA
			}{
				{"at rest", tokens.StateNormal, color.NRGBA{}},
				{"hovered", tokens.StateHover, sc.p.HoverOverlay},
				{"pressed", tokens.StatePressed, sc.p.PressOverlay},
			} {
				if got := picker.ToolbarFill(sc.p, tc.state); got != tc.want {
					t.Errorf("%s: ToolbarFill = %v, want %v", tc.name, got, tc.want)
				}
			}
		})
	}
}

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
	vgcolor "github.com/vibrantgio/theme/color"
	"github.com/vibrantgio/theme/tokens"
)

// onSurface paints the whole frame in the fill of the surface the trigger is
// standing on and draws w inset inside it, and it has to do both.
//
// The host surface, because what the images are for is the separation between
// the control and the band it stands on: against the headless window's own
// clear colour a control that stood off its band and one that read as part of
// it look alike. The inset, because a control drawn at the image
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

// toolbar is RenderToolbar at the default spacing and comfortable density —
// the resolved tokens every measurement and image below draws with.
func toolbar(t *testing.T, p tokens.PlatformColors, s picker.ToolbarState) layout.Widget {
	t.Helper()
	return picker.RenderToolbar(defaultShaper(t), toolbarValue, p,
		tokens.Spacing, tokens.DefaultTypography.LabelLarge,
		tokens.Comfortable, s)
}

// TestToolbarGoldenOnEverySurface records or diffs the resting trigger in both
// schemes on each of the two surfaces. Four images, and between them they are
// the claim this geometry makes: the control carries a fill of its own that
// stands off the chrome band. On the content plane in the light appearance the
// step goes to nothing — the platform's own toolbar control stands there as
// white on white, told from the band by a shadow this library does not draw —
// and the light images record that rather than inventing an edge.
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

// TestToolbarDrawsAtTheDensityTable holds the geometry the trigger takes off
// the tokens rather than off numbers of its own: the height is the density's
// control height and nothing else, which is what the platform's pop-up
// measures and what the form trigger draws, so a face that reached for its own
// padding or grew to its own line box would draw a different box.
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
				tokens.Spacing, d.ts, d.d, picker.ToolbarState{})).Size
			if trigger.Y != d.want {
				t.Errorf("toolbar height %d, want the density's control height %d",
					trigger.Y, d.want)
			}
			if trigger.X >= box.X {
				t.Errorf("toolbar measured %d dp wide in a %d dp box: it is sized to its value", trigger.X, box.X)
			}
		})
	}
}

// TestToolbarMarkIsSteadyAcrossTheWalk is the platform ruling written down
// where a future change cannot silently undo it: a pop-up trigger's mark says
// the control holds one of several values and never "this is open", so nothing
// about the pointer's state may move it. The face offers no open flag to flip
// — that is the structural half — and this is the drawn half: the box the trigger
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

// TestToolbarFillIsThePlatformsOverlay pins the passthrough by name: the
// trigger draws the platform's measured toolbar control fill at rest, and a
// toolbar button is the one control on this platform that tints under the
// pointer, so the platform's own two overlays go over that fill under the
// pointer and while it is held — each resolved against it, because the
// platform composites a coverage in encoded sRGB and Gio's rasterizer would
// not.
func TestToolbarFillIsThePlatformsOverlay(t *testing.T) {
	for _, sc := range goldenSchemes {
		t.Run(sc.name, func(t *testing.T) {
			fill := sc.p.ToolbarControlFill
			for _, tc := range []struct {
				name  string
				state tokens.State
				want  color.NRGBA
			}{
				{"at rest", tokens.StateNormal, fill},
				{"hovered", tokens.StateHover, vgcolor.Flatten(sc.p.HoverOverlay, fill)},
				{"pressed", tokens.StatePressed, vgcolor.Flatten(sc.p.PressOverlay, fill)},
			} {
				if got := picker.ToolbarFill(sc.p, tc.state); got != tc.want {
					t.Errorf("%s: ToolbarFill = %v, want %v", tc.name, got, tc.want)
				}
			}
		})
	}
}

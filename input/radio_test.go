package input_test

import (
	"image"
	stdcolor "image/color"
	"testing"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/paint"
	"gioui.org/unit"

	"github.com/reactivego/rx"
	golden "github.com/vibrantgio/components/golden"
	"github.com/vibrantgio/components/input"
	"github.com/vibrantgio/components/internal/control"
	vgcolor "github.com/vibrantgio/theme/color"
	"github.com/vibrantgio/theme/tokens"
)

// ---- Golden-image tests ----

// TestRadioGolden records or diffs the four canonical radio button states.
func TestRadioGolden(t *testing.T) {
	size := image.Pt(44, 44)

	cases := []struct {
		name     string
		platform tokens.PlatformColors
		state    input.RadioRenderState
	}{
		{"radio-light-unselected", tokens.PlatformLight, input.RadioRenderState{}},
		{"radio-dark-unselected", tokens.PlatformDark, input.RadioRenderState{}},
		{"radio-light-selected", tokens.PlatformLight, input.RadioRenderState{Selected: true}},
		{"radio-light-focused", tokens.PlatformLight, input.RadioRenderState{Focused: true}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := input.RenderRadio(nil, tc.platform, tokens.Spacing, tokens.DefaultTypography.SystemFont, tc.state)
			golden.Render(t, tc.name, size, w)
		})
	}
}

// ---- Accessibility tests ----

// TestRadioFootprintIsItsMeasuredRow checks the radio's visual footprint is
// the density's measured checkbox row, a square with the 16 dp symbol centred
// in it — the checkbox's measured side length, which the radio's circle
// follows so the two read as one row. That footprint is the pointer target
// too — the button's row, not its circle — the same rule
// TestCheckboxTargetIsItsRow exercises.
func TestRadioFootprintIsItsMeasuredRow(t *testing.T) {
	var ops op.Ops
	gtx := layout.Context{
		Metric:      unit.Metric{PxPerDp: 1, PxPerSp: 1},
		Constraints: layout.Exact(image.Pt(120, 120)),
		Ops:         &ops,
	}

	dims := input.RenderRadio(
		nil,
		tokens.PlatformLight,
		tokens.Spacing,
		tokens.DefaultTypography.SystemFont,
		input.RadioRenderState{},
	)(gtx)

	want := int(tokens.Comfortable.CheckboxRowHeight)
	if dims.Size.X != want || dims.Size.Y != want {
		t.Errorf("radio footprint = %v, want %dx%d px (CheckboxRowHeight square at 1:1 scale)", dims.Size, want, want)
	}
}

// TestRadioCompactGolden records or diffs the radio at tokens.Compact through
// the live pipeline: the 16 dp symbol centred in the Compact control height.
func TestRadioCompactGolden(t *testing.T) {
	w := materialize(t, input.Radio(rx.Of(densityTheme(tokens.Compact)), input.RadioProps{
		Description: "choice",
		Selected:    true,
	}))
	golden.Render(t, "radio-light-compact-selected", image.Pt(44, 44), w)
}

// TestRadioSelectedIsVisuallyDistinct confirms the selected state renders
// differently from the unselected state.
func TestRadioSelectedIsVisuallyDistinct(t *testing.T) {
	size := image.Pt(44, 44)

	imgUnselected := golden.Capture(t, size, input.RenderRadio(
		nil,
		tokens.PlatformLight, tokens.Spacing,
		tokens.DefaultTypography.SystemFont,
		input.RadioRenderState{},
	))
	imgSelected := golden.Capture(t, size, input.RenderRadio(
		nil,
		tokens.PlatformLight, tokens.Spacing,
		tokens.DefaultTypography.SystemFont,
		input.RadioRenderState{Selected: true},
	))

	if imgUnselected == nil || imgSelected == nil {
		return
	}
	if n := golden.PixelDiff(imgUnselected, imgSelected); n == 0 {
		t.Error("selected and unselected radio buttons render identically; expected visual difference")
	}
}

// TestRadioFocusRingIsVisuallyDistinct confirms the focused state renders
// differently from the normal state (focus ring must add pixels).
func TestRadioFocusRingIsVisuallyDistinct(t *testing.T) {
	size := image.Pt(44, 44)

	imgNormal := golden.Capture(t, size, input.RenderRadio(
		nil,
		tokens.PlatformLight, tokens.Spacing,
		tokens.DefaultTypography.SystemFont,
		input.RadioRenderState{},
	))
	imgFocused := golden.Capture(t, size, input.RenderRadio(
		nil,
		tokens.PlatformLight, tokens.Spacing,
		tokens.DefaultTypography.SystemFont,
		input.RadioRenderState{Focused: true},
	))

	if imgNormal == nil || imgFocused == nil {
		return
	}
	if n := golden.PixelDiff(imgNormal, imgFocused); n == 0 {
		t.Error("focused and normal radio buttons render identically; expected focus ring pixels to differ")
	}
}

// TestTheSwitchedOffRadioIsOneFillAndNoEdge reads the switched-off radio off a
// capture of the component, in both appearances, against the same reference
// the box is read against (switchedOffReadings, in checkbox_test.go): the
// Save panel holds no switched-off radio, so the radio takes the checkbox's
// reading, the two standing beside each other in one form.
//
// The edge is read at the widest point of the circle, the one column the
// measured one-pixel hairline occupies there, and it is read unpremultiplied:
// a circle's rim is antialiased by the rasterizer, so that column carries the
// edge's colour at a coverage of its own, and dividing the coverage out is
// what separates the colour drawn from the fraction of the pixel it reached.
func TestTheSwitchedOffRadioIsOneFillAndNoEdge(t *testing.T) {
	const size = 44

	// The symbol's 16 dp circle, centred in the density's checkbox row, at the
	// 1:1 metric golden.Capture renders at.
	circle := 16
	row := int(tokens.Comfortable.CheckboxRowHeight)
	c := row / 2
	rim := c - circle/2

	for _, sc := range switchedOffReadings {
		t.Run(sc.name, func(t *testing.T) {
			want := control.Faded(sc.platform.PushButtonFill, sc.sheet)

			img := golden.Capture(t, image.Pt(size, size), input.RenderRadio(
				nil,
				sc.platform, tokens.Spacing,
				tokens.DefaultTypography.SystemFont,
				input.RadioRenderState{Disabled: true, Surface: sc.sheet},
			))
			if img == nil {
				return
			}
			centre := img.RGBAAt(c, c)
			if !nearlyEqual(centre, want) {
				t.Errorf("a switched-off radio fills %v, want the platform's control fill at the disabled coverage %v", centre, want)
			}
			if got := unpremultiplied(img.RGBAAt(rim, c)); !nearlyEqual(got, want) {
				t.Errorf("a switched-off radio's rim reads %v against its interior's %v; the platform draws no edge on one", got, centre)
			}

			on := golden.Capture(t, image.Pt(size, size), input.RenderRadio(
				nil,
				sc.platform, tokens.Spacing,
				tokens.DefaultTypography.SystemFont,
				input.RadioRenderState{Surface: sc.sheet},
			))
			if on == nil {
				return
			}
			enabled := vgcolor.Flatten(sc.platform.Separator, sc.sheet)
			if enabled == want {
				t.Errorf("the enabled and the switched-off radio both fill %v on %v; the platform draws them apart", want, sc.sheet)
			}
			if got := unpremultiplied(on.RGBAAt(rim, c)); !nearlyEqual(got, enabled) {
				t.Errorf("an enabled radio's rim reads %v against its interior's %v; the platform draws no edge on one", got, enabled)
			}
		})
	}
}

// unselectedReadings are the enabled unselected radio's MEASURED discs, each
// a surface and the disc's fill on it. MEASURED,
// finder-view-options-{light,dark}.png: #e6e6e6 on #ffffff light, #393d3e on
// #232829 dark. MEASURED, system-settings-radios-{light,dark}.png: #dfdfdf on
// the grouped box's #f7f7f7 light, #3f4346 on #2a2f32 dark.
var unselectedReadings = []struct {
	name     string
	platform tokens.PlatformColors
	surface  stdcolor.NRGBA
	disc     stdcolor.NRGBA
}{
	{"light", tokens.PlatformLight, stdcolor.NRGBA{0xff, 0xff, 0xff, 0xff}, stdcolor.NRGBA{0xe6, 0xe6, 0xe6, 0xff}},
	{"dark", tokens.PlatformDark, stdcolor.NRGBA{0x23, 0x28, 0x29, 0xff}, stdcolor.NRGBA{0x39, 0x3d, 0x3e, 0xff}},
	{"light grouped box", tokens.PlatformLight, stdcolor.NRGBA{0xf7, 0xf7, 0xf7, 0xff}, stdcolor.NRGBA{0xdf, 0xdf, 0xdf, 0xff}},
	{"dark grouped box", tokens.PlatformDark, stdcolor.NRGBA{0x2a, 0x2f, 0x32, 0xff}, stdcolor.NRGBA{0x3f, 0x43, 0x46, 0xff}},
}

// TestTheUnselectedDiscIsAStepApartWithNoEdge reads the enabled unselected
// radio off a capture of the component and against the stored reference.
//
// The fill: the platform's separator flattened onto the surface lands on each
// measured disc to the byte, and the disc fills with that. The edge: the
// platform's disc has no ring and no edge, so every pixel of the capture lies
// between the surface and the disc's fill on each channel. A ring or a
// hairline in any other colour puts a pixel outside that range.
func TestTheUnselectedDiscIsAStepApartWithNoEdge(t *testing.T) {
	const size = 44
	row := int(tokens.Comfortable.CheckboxRowHeight)
	c := row / 2

	for _, sc := range unselectedReadings {
		t.Run(sc.name, func(t *testing.T) {
			want := vgcolor.Flatten(sc.platform.Separator, sc.surface)
			if want != sc.disc {
				t.Errorf("the separator over %v lands on %v, want the capture's %v", sc.surface, want, sc.disc)
			}

			disc := input.RenderRadio(
				nil,
				sc.platform, tokens.Spacing,
				tokens.DefaultTypography.SystemFont,
				input.RadioRenderState{Surface: sc.surface},
			)
			img := golden.Capture(t, image.Pt(size, size), func(gtx layout.Context) layout.Dimensions {
				paint.Fill(gtx.Ops, sc.surface)
				return disc(gtx)
			})
			if img == nil {
				return
			}
			centre := img.RGBAAt(c, c)
			if !nearlyEqual(centre, sc.disc) {
				t.Errorf("an unselected radio fills %v, want the capture's %v", centre, sc.disc)
			}
			// MEASURED: down the disc's middle column the platform draws one
			// fill from edge to edge. The first and last pixel of the run
			// carry the rim's coverage, so the run read here is the 14 pixels
			// inside them.
			rim := c - 8
			for i := 1; i < 15; i++ {
				if got := img.RGBAAt(c, rim+i); got != centre {
					t.Errorf("down its middle column, the disc reads %v at row %d against its centre's %v; the platform draws one fill", got, rim+i, centre)
				}
				if got := img.RGBAAt(rim+i, c); got != centre {
					t.Errorf("across its middle row, the disc reads %v at column %d against its centre's %v; the platform draws one fill", got, rim+i, centre)
				}
			}

			between := func(v, a, b uint8) bool {
				lo, hi := int(a), int(b)
				if lo > hi {
					lo, hi = hi, lo
				}
				return int(v) >= lo-1 && int(v) <= hi+1
			}
			edges := 0
			for y := 0; y < size; y++ {
				for x := 0; x < size; x++ {
					px := img.RGBAAt(x, y)
					if !between(px.R, sc.surface.R, sc.disc.R) || !between(px.G, sc.surface.G, sc.disc.G) || !between(px.B, sc.surface.B, sc.disc.B) {
						if edges == 0 {
							t.Errorf("the pixel at (%d, %d) reads %v, outside the range from the surface %v to the disc %v: an edge", x, y, px, sc.surface, sc.disc)
						}
						edges++
					}
				}
			}
			if edges > 0 {
				t.Errorf("%d pixels lie outside the range from the surface to the disc; the platform draws no ring and no edge on an unselected radio", edges)
			}
		})
	}
}

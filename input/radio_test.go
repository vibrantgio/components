package input_test

import (
	"image"
	"testing"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"

	"github.com/reactivego/rx"
	golden "github.com/vibrantgio/components/golden"
	"github.com/vibrantgio/components/input"
	"github.com/vibrantgio/components/internal/control"
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
			w := input.RenderRadio(nil, tc.platform, tokens.Spacing, tokens.DefaultTypography.BodyLarge, tc.state)
			golden.Render(t, tc.name, size, w)
		})
	}
}

// ---- Accessibility tests ----

// TestRadioFootprintIsItsMeasuredRow checks the radio's visual footprint is
// the density's measured checkbox row, a square with the 16 dp glyph centred
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
		tokens.DefaultTypography.BodyLarge,
		input.RadioRenderState{},
	)(gtx)

	want := int(tokens.Comfortable.CheckboxRowHeight)
	if dims.Size.X != want || dims.Size.Y != want {
		t.Errorf("radio footprint = %v, want %dx%d px (CheckboxRowHeight square at 1:1 scale)", dims.Size, want, want)
	}
}

// TestRadioCompactGolden records or diffs the radio at tokens.Compact through
// the live pipeline: the 16 dp glyph centred in the Compact control height.
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
		tokens.DefaultTypography.BodyLarge,
		input.RadioRenderState{},
	))
	imgSelected := golden.Capture(t, size, input.RenderRadio(
		nil,
		tokens.PlatformLight, tokens.Spacing,
		tokens.DefaultTypography.BodyLarge,
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
		tokens.DefaultTypography.BodyLarge,
		input.RadioRenderState{},
	))
	imgFocused := golden.Capture(t, size, input.RenderRadio(
		nil,
		tokens.PlatformLight, tokens.Spacing,
		tokens.DefaultTypography.BodyLarge,
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
// Save dialog holds no switched-off radio, so the radio takes the checkbox's
// reading, the two standing beside each other in one form.
//
// The edge is read at the widest point of the circle, the one column the
// measured one-pixel hairline occupies there, and it is read unpremultiplied:
// a circle's rim is antialiased by the rasterizer, so that column carries the
// edge's colour at a coverage of its own, and dividing the coverage out is
// what separates the colour drawn from the fraction of the pixel it reached.
func TestTheSwitchedOffRadioIsOneFillAndNoEdge(t *testing.T) {
	const size = 44

	// The glyph's 16 dp circle, centred in the density's checkbox row, at the
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
				tokens.DefaultTypography.BodyLarge,
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
				tokens.DefaultTypography.BodyLarge,
				input.RadioRenderState{Surface: sc.sheet},
			))
			if on == nil {
				return
			}
			edge, fill := control.Border(sc.platform), control.Fill(sc.platform)
			if got := unpremultiplied(on.RGBAAt(rim, c)); !nearlyEqual(got, edge) {
				t.Errorf("an enabled radio's rim reads %v, want the field hairline %v the enabled radio draws its edge in", got, edge)
			}
			if got := on.RGBAAt(rim+1, c); !nearlyEqual(got, fill) {
				t.Errorf("an enabled radio reads %v one pixel in from its rim, want its own fill %v: the measured edge is one pixel wide", got, fill)
			}
		})
	}
}

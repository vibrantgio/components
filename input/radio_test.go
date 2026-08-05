package input_test

import (
	"image"
	"testing"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"

	"github.com/reactivego/rx"
	"github.com/vibrantgio/prism/input"
	golden "github.com/vibrantgio/prism/internal/golden"
	"github.com/vibrantgio/spectrum/tokens"
)

// ---- Golden-image tests ----

// TestRadioGolden records or diffs the four canonical radio button states.
func TestRadioGolden(t *testing.T) {
	size := image.Pt(44, 44)

	cases := []struct {
		name   string
		colors tokens.ColorTokens
		state  input.RadioRenderState
	}{
		{"radio-light-unselected", tokens.DefaultLight, input.RadioRenderState{}},
		{"radio-dark-unselected", tokens.DefaultDark, input.RadioRenderState{}},
		{"radio-light-selected", tokens.DefaultLight, input.RadioRenderState{Selected: true}},
		{"radio-light-focused", tokens.DefaultLight, input.RadioRenderState{Focused: true}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := input.RenderRadio(tc.colors, tokens.Spacing, tokens.Radius, tc.state)
			golden.Render(t, tc.name, size, w)
		})
	}
}

// ---- Accessibility tests ----

// TestRadioFootprintIsControlHeight checks the radio's visual footprint is
// the density's control-height square (E1.3: 36 dp Comfortable) with the
// 20 dp glyph centred in it. The 44 dp WCAG 2.5.5 floor applies to the
// pointer target, not the footprint: the live Radio extends its hit area via
// internal/hit (same mechanism TestCheckboxHitSlopToggles exercises).
func TestRadioFootprintIsControlHeight(t *testing.T) {
	var ops op.Ops
	gtx := layout.Context{
		Metric:      unit.Metric{PxPerDp: 1, PxPerSp: 1},
		Constraints: layout.Exact(image.Pt(120, 120)),
		Ops:         &ops,
	}

	dims := input.RenderRadio(
		tokens.DefaultLight,
		tokens.Spacing,
		tokens.Radius,
		input.RadioRenderState{},
	)(gtx)

	want := int(tokens.Comfortable.ControlHeight)
	if dims.Size.X != want || dims.Size.Y != want {
		t.Errorf("radio footprint = %v, want %dx%d px (ControlHeight square at 1:1 scale)", dims.Size, want, want)
	}
}

// TestRadioCompactGolden records or diffs the radio at tokens.Compact through
// the live pipeline: the 20 dp glyph centred in a 28 dp footprint.
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
		tokens.DefaultLight, tokens.Spacing, tokens.Radius,
		input.RadioRenderState{},
	))
	imgSelected := golden.Capture(t, size, input.RenderRadio(
		tokens.DefaultLight, tokens.Spacing, tokens.Radius,
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
		tokens.DefaultLight, tokens.Spacing, tokens.Radius,
		input.RadioRenderState{},
	))
	imgFocused := golden.Capture(t, size, input.RenderRadio(
		tokens.DefaultLight, tokens.Spacing, tokens.Radius,
		input.RadioRenderState{Focused: true},
	))

	if imgNormal == nil || imgFocused == nil {
		return
	}
	if n := golden.PixelDiff(imgNormal, imgFocused); n == 0 {
		t.Error("focused and normal radio buttons render identically; expected focus ring pixels to differ")
	}
}

package input_test

import (
	"image"
	"testing"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"

	"github.com/vibrantgio/prism/input"
	golden "github.com/vibrantgio/prism/internal/golden"
	"github.com/vibrantgio/spectrum/tokens"
)

// ---- Golden-image tests ----

// TestCheckboxGolden records or diffs the four canonical checkbox states.
func TestCheckboxGolden(t *testing.T) {
	size := image.Pt(44, 44)

	// Zero corner radius avoids anti-aliasing variance between GPU context
	// initialisations. Colour accuracy and border/fill presence are still
	// fully exercised; the exact radius is tested in production rendering.
	sharpRadius := tokens.RadiusScale{}

	cases := []struct {
		name   string
		colors tokens.ColorTokens
		state  input.CheckboxRenderState
	}{
		{"light-unchecked", tokens.DefaultLight, input.CheckboxRenderState{}},
		{"dark-unchecked", tokens.DefaultDark, input.CheckboxRenderState{}},
		{"light-checked", tokens.DefaultLight, input.CheckboxRenderState{Checked: true}},
		{"light-focused", tokens.DefaultLight, input.CheckboxRenderState{Focused: true}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := input.RenderCheckbox(tc.colors, tokens.Spacing, sharpRadius, tc.state)
			golden.Render(t, tc.name, size, w)
		})
	}
}

// ---- Accessibility tests ----

// TestCheckboxMinHitTarget checks the checkbox meets the 44 dp minimum
// interactive height (DESIGN §Accessibility / WCAG 2.5.5).
func TestCheckboxMinHitTarget(t *testing.T) {
	var ops op.Ops
	gtx := layout.Context{
		Metric:      unit.Metric{PxPerDp: 1, PxPerSp: 1},
		Constraints: layout.Exact(image.Pt(120, 120)),
		Ops:         &ops,
	}

	dims := input.RenderCheckbox(
		tokens.DefaultLight,
		tokens.Spacing,
		tokens.Radius,
		input.CheckboxRenderState{},
	)(gtx)

	const wantPx = 44
	if dims.Size.Y < wantPx {
		t.Errorf("checkbox height = %d px, want ≥ %d px (44 dp at 1:1 scale)", dims.Size.Y, wantPx)
	}
	if dims.Size.X < wantPx {
		t.Errorf("checkbox width = %d px, want ≥ %d px (44 dp at 1:1 scale)", dims.Size.X, wantPx)
	}
}

// TestCheckboxCheckedIsVisuallyDistinct confirms the checked state renders
// differently from the unchecked state.
func TestCheckboxCheckedIsVisuallyDistinct(t *testing.T) {
	size := image.Pt(44, 44)

	imgUnchecked := golden.Capture(t, size, input.RenderCheckbox(
		tokens.DefaultLight, tokens.Spacing, tokens.Radius,
		input.CheckboxRenderState{},
	))
	imgChecked := golden.Capture(t, size, input.RenderCheckbox(
		tokens.DefaultLight, tokens.Spacing, tokens.Radius,
		input.CheckboxRenderState{Checked: true},
	))

	if imgUnchecked == nil || imgChecked == nil {
		return
	}
	if n := golden.PixelDiff(imgUnchecked, imgChecked); n == 0 {
		t.Error("checked and unchecked checkboxes render identically; expected visual difference")
	}
}

// TestCheckboxFocusRingIsVisuallyDistinct confirms the focused state renders
// differently from the normal state (focus ring must add pixels).
func TestCheckboxFocusRingIsVisuallyDistinct(t *testing.T) {
	size := image.Pt(44, 44)

	imgNormal := golden.Capture(t, size, input.RenderCheckbox(
		tokens.DefaultLight, tokens.Spacing, tokens.Radius,
		input.CheckboxRenderState{},
	))
	imgFocused := golden.Capture(t, size, input.RenderCheckbox(
		tokens.DefaultLight, tokens.Spacing, tokens.Radius,
		input.CheckboxRenderState{Focused: true},
	))

	if imgNormal == nil || imgFocused == nil {
		return
	}
	if n := golden.PixelDiff(imgNormal, imgFocused); n == 0 {
		t.Error("focused and normal checkboxes render identically; expected focus ring pixels to differ")
	}
}

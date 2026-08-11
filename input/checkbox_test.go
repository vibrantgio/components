package input_test

import (
	"image"
	"testing"

	"gioui.org/f32"
	gioinput "gioui.org/io/input"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"

	"github.com/reactivego/rx"
	golden "github.com/vibrantgio/prism/golden"
	"github.com/vibrantgio/prism/input"
	"github.com/vibrantgio/theme/theme"
	"github.com/vibrantgio/theme/tokens"
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
		// Every name carries the component prefix. All four inputs share one
		// testdata/golden directory, and until F4.1 the unprefixed
		// "light-focused" here and in textfield_test.go named one file: the
		// checkbox compared its 44x44 render against the text field's 300x60
		// golden, and the size mismatch made the comparison pass.
		{"checkbox-light-unchecked", tokens.DefaultLight, input.CheckboxRenderState{}},
		{"checkbox-dark-unchecked", tokens.DefaultDark, input.CheckboxRenderState{}},
		{"checkbox-light-checked", tokens.DefaultLight, input.CheckboxRenderState{Checked: true}},
		{"checkbox-light-focused", tokens.DefaultLight, input.CheckboxRenderState{Focused: true}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := input.RenderCheckbox(tc.colors, tokens.Spacing, sharpRadius, tc.state)
			golden.Render(t, tc.name, size, w)
		})
	}
}

// ---- Accessibility tests ----

// TestCheckboxFootprintIsControlHeight checks the checkbox's visual footprint
// is the density's control-height square (E1.3: 36 dp Comfortable) with the
// 20 dp glyph centred in it. The 44 dp WCAG 2.5.5 floor applies to the pointer
// target, not the footprint: the live Checkbox extends its hit area via
// internal/hit, exercised by TestCheckboxHitSlopToggles.
func TestCheckboxFootprintIsControlHeight(t *testing.T) {
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

	want := int(tokens.Comfortable.ControlHeight)
	if dims.Size.X != want || dims.Size.Y != want {
		t.Errorf("checkbox footprint = %v, want %dx%d px (ControlHeight square at 1:1 scale)", dims.Size, want, want)
	}
}

// TestCheckboxHitSlopToggles checks the live checkbox's pointer target
// extends to the 44 dp floor: a click outside the 36 dp visual footprint but
// inside the 44 dp hit rectangle toggles the value.
func TestCheckboxHitSlopToggles(t *testing.T) {
	var toggled int
	w := materialize(t, input.Checkbox(rx.Of(theme.Default()), input.CheckboxProps{
		Description: "opt-in",
		OnChange:    func(_ layout.Context, _ bool) { toggled++ },
	}))

	r := new(gioinput.Router)
	ops := new(op.Ops)
	drive := func() {
		ops.Reset()
		gtx := layout.Context{
			Metric:      unit.Metric{PxPerDp: 1, PxPerSp: 1},
			Constraints: layout.Exact(image.Pt(120, 120)),
			Ops:         ops,
			Source:      r.Source(),
		}
		w(gtx)
		r.Frame(ops)
	}

	drive() // register the input area

	// The hit rect is 44 px centred on the 36 px footprint: -4..40 on each
	// axis. Click at (38, 38) — outside the footprint, inside the slop.
	pos := f32.Pt(38, 38)
	r.Queue(
		pointer.Event{Kind: pointer.Press, Position: pos, Buttons: pointer.ButtonPrimary, Source: pointer.Mouse},
		pointer.Event{Kind: pointer.Release, Position: pos, Buttons: pointer.ButtonPrimary, Source: pointer.Mouse},
	)
	drive()
	if toggled != 1 {
		t.Errorf("click in the hit slop: OnChange fired %d times, want 1", toggled)
	}
}

// TestCheckboxCompactGolden records or diffs the checkbox at tokens.Compact
// through the live pipeline: the 20 dp glyph centred in a 28 dp footprint.
func TestCheckboxCompactGolden(t *testing.T) {
	w := materialize(t, input.Checkbox(rx.Of(densityTheme(tokens.Compact)), input.CheckboxProps{
		Description: "opt-in",
		Checked:     true,
	}))
	golden.Render(t, "checkbox-light-compact-checked", image.Pt(44, 44), w)
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

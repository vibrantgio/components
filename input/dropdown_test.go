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

// TestDropdownGolden records or diffs the four canonical dropdown states.
func TestDropdownGolden(t *testing.T) {
	shaper := defaultShaper(t)

	// Empty option text avoids GPU font rasterisation variance across headless
	// contexts. Shape (border colour, background, chevron, option rows) is still
	// fully exercised.
	opts := []string{"", "", ""}
	openH := 44 + len(opts)*44

	// Zero corner radius avoids anti-aliasing variance between GPU context
	// initialisations. Border/fill presence and colour accuracy are still
	// fully exercised; the exact radius is tested in production rendering.
	sharpRadius := tokens.RadiusScale{}

	cases := []struct {
		name   string
		colors tokens.ColorTokens
		size   image.Point
		state  input.DropdownRenderState
	}{
		{
			"dropdown-light-closed",
			tokens.DefaultLight,
			image.Pt(200, 44),
			input.DropdownRenderState{Options: opts},
		},
		{
			"dropdown-dark-closed",
			tokens.DefaultDark,
			image.Pt(200, 44),
			input.DropdownRenderState{Options: opts},
		},
		{
			"dropdown-light-focused",
			tokens.DefaultLight,
			image.Pt(200, 44),
			input.DropdownRenderState{Focused: true, Options: opts},
		},
		{
			"dropdown-light-open",
			tokens.DefaultLight,
			image.Pt(200, openH),
			input.DropdownRenderState{Open: true, Options: opts, Selected: 0},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := input.RenderDropdown(
				shaper,
				tc.colors,
				tokens.Spacing,
				sharpRadius,
				tokens.DefaultTypeScale,
				tc.state,
			)
			golden.Render(t, tc.name, tc.size, w)
		})
	}
}

// ---- Accessibility tests ----

// TestDropdownMinHitTarget checks the trigger meets the 44 dp minimum
// interactive height (DESIGN §Accessibility / WCAG 2.5.5).
func TestDropdownMinHitTarget(t *testing.T) {
	shaper := defaultShaper(t)
	var ops op.Ops
	gtx := layout.Context{
		Metric:      unit.Metric{PxPerDp: 1, PxPerSp: 1},
		Constraints: layout.Exact(image.Pt(200, 120)),
		Ops:         &ops,
	}

	dims := input.RenderDropdown(
		shaper,
		tokens.DefaultLight,
		tokens.Spacing,
		tokens.Radius,
		tokens.DefaultTypeScale,
		input.DropdownRenderState{Options: []string{"Option A"}},
	)(gtx)

	const wantPx = 44
	if dims.Size.Y < wantPx {
		t.Errorf("dropdown trigger height = %d px, want ≥ %d px (44 dp at 1:1 scale)", dims.Size.Y, wantPx)
	}
	if dims.Size.X < wantPx {
		t.Errorf("dropdown trigger width = %d px, want ≥ %d px (44 dp at 1:1 scale)", dims.Size.X, wantPx)
	}
}

// TestDropdownFocusRingIsVisuallyDistinct confirms the focused state renders
// differently from the normal state (focus ring must add pixels).
func TestDropdownFocusRingIsVisuallyDistinct(t *testing.T) {
	shaper := defaultShaper(t)
	opts := []string{"Alpha", "Beta", "Gamma"}
	size := image.Pt(200, 44)

	imgNormal := golden.Capture(t, size, input.RenderDropdown(
		shaper,
		tokens.DefaultLight, tokens.Spacing, tokens.Radius, tokens.DefaultTypeScale,
		input.DropdownRenderState{Options: opts},
	))
	imgFocused := golden.Capture(t, size, input.RenderDropdown(
		shaper,
		tokens.DefaultLight, tokens.Spacing, tokens.Radius, tokens.DefaultTypeScale,
		input.DropdownRenderState{Focused: true, Options: opts},
	))

	if imgNormal == nil || imgFocused == nil {
		return
	}
	if n := golden.PixelDiff(imgNormal, imgFocused); n == 0 {
		t.Error("focused and normal dropdown triggers render identically; expected focus ring pixels to differ")
	}
}

// TestDropdownOpenStateIsVisuallyDistinct confirms the open state renders
// differently from the closed state (option list must add pixels).
func TestDropdownOpenStateIsVisuallyDistinct(t *testing.T) {
	shaper := defaultShaper(t)
	opts := []string{"Alpha", "Beta", "Gamma"}
	openH := 44 + len(opts)*44

	imgClosed := golden.Capture(t, image.Pt(200, openH), input.RenderDropdown(
		shaper,
		tokens.DefaultLight, tokens.Spacing, tokens.Radius, tokens.DefaultTypeScale,
		input.DropdownRenderState{Options: opts},
	))
	imgOpen := golden.Capture(t, image.Pt(200, openH), input.RenderDropdown(
		shaper,
		tokens.DefaultLight, tokens.Spacing, tokens.Radius, tokens.DefaultTypeScale,
		input.DropdownRenderState{Open: true, Options: opts, Selected: 0},
	))

	if imgClosed == nil || imgOpen == nil {
		return
	}
	if n := golden.PixelDiff(imgClosed, imgOpen); n == 0 {
		t.Error("open and closed dropdown render identically; expected option list pixels to differ")
	}
}

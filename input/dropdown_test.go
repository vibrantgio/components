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

// TestDropdownGolden records or diffs the four canonical dropdown states.
func TestDropdownGolden(t *testing.T) {
	shaper := defaultShaper(t)

	// Empty option text avoids GPU font rasterisation variance across headless
	// contexts. Shape (border colour, background, chevron, option rows) is still
	// fully exercised.
	opts := []string{"", "", ""}
	// Trigger and option rows are each ControlHeight tall (E1.3).
	ctl := int(tokens.Comfortable.ControlHeight)
	openH := ctl + len(opts)*ctl

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

// TestDropdownTriggerHeightIsControlHeight checks the closed trigger draws at
// the density's control height (E1.3: 36 dp Comfortable). The 44 dp WCAG
// 2.5.5 floor applies to the pointer target: the live Dropdown extends the
// trigger's hit area via internal/hit (option rows stack against each other
// and keep their row bounds as their target).
func TestDropdownTriggerHeightIsControlHeight(t *testing.T) {
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

	want := int(tokens.Comfortable.ControlHeight)
	if dims.Size.Y != want {
		t.Errorf("dropdown trigger height = %d px, want %d px (ControlHeight at 1:1 scale)", dims.Size.Y, want)
	}
}

// TestDropdownCompactGolden records or diffs the closed dropdown at
// tokens.Compact through the live pipeline: a 28 dp trigger bar.
func TestDropdownCompactGolden(t *testing.T) {
	w := materialize(t, input.Dropdown(rx.Of(densityTheme(tokens.Compact)), input.DropdownProps{
		Description: "choose",
		Options:     []string{"", "", ""}, // empty labels: no font rasterisation
	}))
	golden.Render(t, "dropdown-light-compact-closed", image.Pt(200, 44), w)
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

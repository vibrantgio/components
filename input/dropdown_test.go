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
	"github.com/vibrantgio/theme/tokens"
)

// ---- Golden-image tests ----

// TestDropdownGolden records or diffs the canonical dropdown states. The open
// menu is recorded in both schemes: its selected row is a per-scheme colour
// pairing (fill and foreground together, see TestDropdownOptionRowContrast), and a
// pairing only one scheme is pictured in is a pairing the other can lose.
func TestDropdownGolden(t *testing.T) {
	shaper := defaultShaper(t)

	// Real option text: DeterministicShaper pins the faces, so Latin glyphs
	// rasterise the same everywhere and the trigger's selected label and the
	// option rows are visible rather than implied.
	opts := []string{"Alpha", "Beta", "Gamma"}
	// Trigger and option rows are each one BodyLarge line box plus the
	// density's vertical padding, floored at ControlHeight. That is 40 dp
	// Comfortable, not the 36 dp floor: sizing this window off ControlHeight
	// alone clips 4 px off the last option row.
	row := int(tokens.DefaultTypography.BodyLarge.LineHeight + 2*tokens.Comfortable.PaddingY)
	if floor := int(tokens.Comfortable.ControlHeight); row < floor {
		row = floor
	}
	// An open dropdown's menu stands OVER its trigger with the held row on
	// the trigger's label, so the plane reaches above the trigger's own top
	// edge: the open frames carry a row of room above the control and the
	// control is laid out into it.
	openTop := row
	openH := openTop + row + len(opts)*row

	// Zero corner radius avoids anti-aliasing variance between GPU context
	// initialisations. Border/fill presence and colour accuracy are still
	// fully exercised; the exact radius is tested in production rendering.
	sharpRadius := tokens.RadiusScale{}

	cases := []struct {
		name     string
		platform tokens.PlatformColors
		size     image.Point
		at       int // where in the frame the control is laid out
		state    input.DropdownRenderState
	}{
		{
			"dropdown-light-closed",
			tokens.PlatformLight,
			image.Pt(200, 44),
			0,
			input.DropdownRenderState{Options: opts},
		},
		{
			"dropdown-dark-closed",
			tokens.PlatformDark,
			image.Pt(200, 44),
			0,
			input.DropdownRenderState{Options: opts},
		},
		{
			"dropdown-light-focused",
			tokens.PlatformLight,
			image.Pt(200, 44),
			0,
			input.DropdownRenderState{Focused: true, Options: opts},
		},
		{
			"dropdown-light-open",
			tokens.PlatformLight,
			image.Pt(200, openH),
			openTop,
			input.DropdownRenderState{Open: true, Options: opts, Selected: 0},
		},
		{
			"dropdown-dark-open",
			tokens.PlatformDark,
			image.Pt(200, openH),
			openTop,
			input.DropdownRenderState{Open: true, Options: opts, Selected: 0},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := input.RenderDropdown(
				shaper,
				tc.platform,
				tokens.Spacing,
				sharpRadius,
				tokens.DefaultTypography.BodyLarge, tokens.Comfortable,
				tc.state,
			)
			at := tc.at
			golden.Render(t, tc.name, tc.size, func(gtx layout.Context) layout.Dimensions {
				off := op.Offset(image.Pt(0, at)).Push(gtx.Ops)
				defer off.Pop()
				w(gtx)
				return layout.Dimensions{Size: gtx.Constraints.Max}
			})
		})
	}
}

// ---- Accessibility tests ----

// TestDropdownTriggerDrawsThePopUpsHeight checks that the deprecated forwarder
// carries components/picker's own answer: the trigger is the platform's pop-up
// button and draws the density's control height, MEASURED at 24 px off the
// save dialog's "File Format:" pop-up. It is not the text field's height —
// that control measures 27 in the same capture — so the forwarder must not be
// asserting the field's arithmetic. The drawn trigger is the pointer target,
// and an option row is its own row, so neither claims a neighbour's pixels.
func TestDropdownTriggerDrawsThePopUpsHeight(t *testing.T) {
	shaper := defaultShaper(t)
	var ops op.Ops
	gtx := layout.Context{
		Metric:      unit.Metric{PxPerDp: 1, PxPerSp: 1},
		Constraints: layout.Exact(image.Pt(200, 120)),
		Ops:         &ops,
	}

	dims := input.RenderDropdown(
		shaper,
		tokens.PlatformLight,
		tokens.Spacing,
		tokens.Radius,
		tokens.DefaultTypography.BodyLarge, tokens.Comfortable,
		input.DropdownRenderState{Options: []string{"Option A"}},
	)(gtx)

	if want := int(tokens.Comfortable.ControlHeight); dims.Size.Y != want {
		t.Errorf("dropdown trigger height = %d px, want the density's control height %d px", dims.Size.Y, want)
	}
}

// TestDropdownCompactGolden records or diffs the closed dropdown at
// tokens.Compact through the live pipeline: the compact control height.
func TestDropdownCompactGolden(t *testing.T) {
	w := materialize(t, input.Dropdown(rx.Of(densityTheme(tokens.Compact)), input.DropdownProps{
		Description: "choose",
		Options:     []string{"Alpha", "Beta", "Gamma"},
		// The live path would otherwise take the theme's fallback Shaper,
		// which resolves against the machine's fonts. A golden pins its faces.
		Shaper: defaultShaper(t),
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
		tokens.PlatformLight, tokens.Spacing, tokens.Radius, tokens.DefaultTypography.BodyLarge, tokens.Comfortable,
		input.DropdownRenderState{Options: opts},
	))
	imgFocused := golden.Capture(t, size, input.RenderDropdown(
		shaper,
		tokens.PlatformLight, tokens.Spacing, tokens.Radius, tokens.DefaultTypography.BodyLarge, tokens.Comfortable,
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
		tokens.PlatformLight, tokens.Spacing, tokens.Radius, tokens.DefaultTypography.BodyLarge, tokens.Comfortable,
		input.DropdownRenderState{Options: opts},
	))
	imgOpen := golden.Capture(t, image.Pt(200, openH), input.RenderDropdown(
		shaper,
		tokens.PlatformLight, tokens.Spacing, tokens.Radius, tokens.DefaultTypography.BodyLarge, tokens.Comfortable,
		input.DropdownRenderState{Open: true, Options: opts, Selected: 0},
	))

	if imgClosed == nil || imgOpen == nil {
		return
	}
	if n := golden.PixelDiff(imgClosed, imgOpen); n == 0 {
		t.Error("open and closed dropdown render identically; expected option list pixels to differ")
	}
}

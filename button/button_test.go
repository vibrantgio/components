package button_test

import (
	"context"
	"image"
	"image/color"
	"testing"

	"gioui.org/f32"
	gioinput "gioui.org/io/input"
	"gioui.org/io/key"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"

	"github.com/reactivego/rx"
	"github.com/vibrantgio/prism/button"
	golden "github.com/vibrantgio/prism/internal/golden"
	"github.com/vibrantgio/spectrum/theme"
	"github.com/vibrantgio/spectrum/tokens"
)

// crossIcon is a deterministic "×" glyph painter — two diagonal clip.Stroke
// lines filling a sizePx×sizePx box — used to exercise the icon-only button.
// Vector strokes (no font/SVG rasterisation) keep golden output stable.
func crossIcon(gtx layout.Context, sizePx int, col color.NRGBA) {
	w := float32(sizePx)
	stroke := float32(gtx.Dp(2))
	if stroke < 1 {
		stroke = 1
	}
	var p clip.Path
	p.Begin(gtx.Ops)
	p.MoveTo(f32.Pt(0, 0))
	p.LineTo(f32.Pt(w, w))
	paint.FillShape(gtx.Ops, col, clip.Stroke{Path: p.End(), Width: stroke}.Op())

	p.Begin(gtx.Ops)
	p.MoveTo(f32.Pt(w, 0))
	p.LineTo(f32.Pt(0, w))
	paint.FillShape(gtx.Ops, col, clip.Stroke{Path: p.End(), Width: stroke}.Op())
}

func defaultShaper(t *testing.T) *text.Shaper {
	t.Helper()
	return tokens.DefaultTypography.Shaper()
}

// ---- Golden-image tests ----

// TestButtonGolden records or diffs the four canonical button states:
// light-normal, dark-normal, light-focused, light-pressed.
func TestButtonGolden(t *testing.T) {
	shaper := defaultShaper(t)
	size := image.Pt(300, 60)

	// Use zero corner radius and empty label to produce fully sharp-edged
	// renders: anti-aliased rounded corners and GPU font rasterisation both
	// vary slightly between GPU context initialisations, breaking determinism.
	// Shape (background colour, focus ring presence) and colour accuracy are
	// still fully exercised; the exact radius and label are tested elsewhere.
	sharpRadius := tokens.RadiusScale{} // all zeros → sharp corners, no AA
	cases := []struct {
		name   string
		colors tokens.ColorTokens
		state  button.RenderState
	}{
		{"light-normal", tokens.DefaultLight, button.RenderState{}},
		{"dark-normal", tokens.DefaultDark, button.RenderState{}},
		{"light-focused", tokens.DefaultLight, button.RenderState{Focused: true}},
		{"light-pressed", tokens.DefaultLight, button.RenderState{Pressed: true}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := button.Render(
				shaper, "",
				tc.colors, tokens.Spacing, sharpRadius, tokens.DefaultTypeScale,
				tc.state,
			)
			golden.Render(t, tc.name, size, w)
		})
	}
}

// ---- Accessibility tests ----

// TestButtonMinHitTarget checks the button meets the 44 dp minimum
// interactive height (DESIGN §Accessibility / WCAG 2.5.5).
func TestButtonMinHitTarget(t *testing.T) {
	shaper := defaultShaper(t)

	var ops op.Ops
	gtx := layout.Context{
		Metric:      unit.Metric{PxPerDp: 1, PxPerSp: 1},
		Constraints: layout.Exact(image.Pt(300, 120)),
		Ops:         &ops,
	}

	dims := button.Render(
		shaper, "OK",
		tokens.DefaultLight, tokens.Spacing, tokens.Radius, tokens.DefaultTypeScale,
		button.RenderState{},
	)(gtx)

	const wantPx = 44 // 44 dp at 1:1 scale
	if dims.Size.Y < wantPx {
		t.Errorf("button height = %d px, want ≥ %d px (44 dp at 1:1 scale)", dims.Size.Y, wantPx)
	}
}

// TestButtonDisabledIsVisuallyDistinct confirms disabled state produces
// different pixels from enabled state.
func TestButtonDisabledIsVisuallyDistinct(t *testing.T) {
	shaper := defaultShaper(t)
	size := image.Pt(300, 60)

	imgEnabled := golden.Capture(t, size, button.Render(
		shaper, "Click me",
		tokens.DefaultLight, tokens.Spacing, tokens.Radius, tokens.DefaultTypeScale,
		button.RenderState{},
	))
	imgDisabled := golden.Capture(t, size, button.Render(
		shaper, "Click me",
		tokens.DefaultLight, tokens.Spacing, tokens.Radius, tokens.DefaultTypeScale,
		button.RenderState{Disabled: true},
	))

	if imgEnabled == nil || imgDisabled == nil {
		return // headless unavailable; Capture called t.Skip
	}
	if n := golden.PixelDiff(imgEnabled, imgDisabled); n == 0 {
		t.Error("disabled and enabled buttons render identically; expected visual difference")
	}
}

// TestButtonFocusRingIsVisuallyDistinct confirms focused state renders
// differently from normal state (the focus ring must add pixels).
func TestButtonFocusRingIsVisuallyDistinct(t *testing.T) {
	shaper := defaultShaper(t)
	size := image.Pt(300, 60)

	imgNormal := golden.Capture(t, size, button.Render(
		shaper, "Click me",
		tokens.DefaultLight, tokens.Spacing, tokens.Radius, tokens.DefaultTypeScale,
		button.RenderState{},
	))
	imgFocused := golden.Capture(t, size, button.Render(
		shaper, "Click me",
		tokens.DefaultLight, tokens.Spacing, tokens.Radius, tokens.DefaultTypeScale,
		button.RenderState{Focused: true},
	))

	if imgNormal == nil || imgFocused == nil {
		return
	}
	if n := golden.PixelDiff(imgNormal, imgFocused); n == 0 {
		t.Error("focused and normal buttons render identically; expected focus ring pixels to differ")
	}
}

// TestButtonPressedIsVisuallyDistinct confirms pressed state renders
// differently from normal state.
func TestButtonPressedIsVisuallyDistinct(t *testing.T) {
	shaper := defaultShaper(t)
	size := image.Pt(300, 60)

	imgNormal := golden.Capture(t, size, button.Render(
		shaper, "Click me",
		tokens.DefaultLight, tokens.Spacing, tokens.Radius, tokens.DefaultTypeScale,
		button.RenderState{},
	))
	imgPressed := golden.Capture(t, size, button.Render(
		shaper, "Click me",
		tokens.DefaultLight, tokens.Spacing, tokens.Radius, tokens.DefaultTypeScale,
		button.RenderState{Pressed: true},
	))

	if imgNormal == nil || imgPressed == nil {
		return
	}
	if n := golden.PixelDiff(imgNormal, imgPressed); n == 0 {
		t.Error("pressed and normal buttons render identically; expected visual difference")
	}
}

// TestButtonHoveredIsVisuallyDistinct confirms hovered state renders
// differently from normal state.
func TestButtonHoveredIsVisuallyDistinct(t *testing.T) {
	shaper := defaultShaper(t)
	size := image.Pt(300, 60)

	imgNormal := golden.Capture(t, size, button.Render(
		shaper, "Click me",
		tokens.DefaultLight, tokens.Spacing, tokens.Radius, tokens.DefaultTypeScale,
		button.RenderState{},
	))
	imgHovered := golden.Capture(t, size, button.Render(
		shaper, "Click me",
		tokens.DefaultLight, tokens.Spacing, tokens.Radius, tokens.DefaultTypeScale,
		button.RenderState{Hovered: true},
	))

	if imgNormal == nil || imgHovered == nil {
		return
	}
	if n := golden.PixelDiff(imgNormal, imgHovered); n == 0 {
		t.Error("hovered and normal buttons render identically; expected visual difference")
	}
}

// ---- Icon-only variant (GX.3) ----

// TestIconButtonGolden records or diffs the icon-only variant in its idle and
// focused states. Zero corner radius keeps edges sharp; the glyph is a
// clip.Stroke "×" so the render is deterministic.
func TestIconButtonGolden(t *testing.T) {
	size := image.Pt(60, 60)
	sharpRadius := tokens.RadiusScale{} // all zeros → sharp corners, no AA
	cases := []struct {
		name   string
		colors tokens.ColorTokens
		state  button.RenderState
	}{
		{"icon-light-normal", tokens.DefaultLight, button.RenderState{}},
		{"icon-light-focused", tokens.DefaultLight, button.RenderState{Focused: true}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := button.RenderIcon(
				crossIcon,
				tc.colors, tokens.Spacing, sharpRadius, tokens.DefaultTypeScale,
				tc.state,
			)
			golden.Render(t, tc.name, size, w)
		})
	}
}

// TestIconButtonMinHitTarget checks the icon-only button is at least the 44 dp
// minimum interactive square (DESIGN §Accessibility / WCAG 2.5.5).
func TestIconButtonMinHitTarget(t *testing.T) {
	var ops op.Ops
	gtx := layout.Context{
		Metric:      unit.Metric{PxPerDp: 1, PxPerSp: 1},
		Constraints: layout.Exact(image.Pt(120, 120)),
		Ops:         &ops,
	}
	dims := button.RenderIcon(
		crossIcon,
		tokens.DefaultLight, tokens.Spacing, tokens.Radius, tokens.DefaultTypeScale,
		button.RenderState{},
	)(gtx)

	const wantPx = 44
	if dims.Size.X < wantPx || dims.Size.Y < wantPx {
		t.Errorf("icon button size = %v, want ≥ %dx%d px (44 dp at 1:1 scale)", dims.Size, wantPx, wantPx)
	}
}

// TestIconButtonFocusRingIsVisuallyDistinct confirms the icon-only button's
// focused state renders differently from idle (the focus ring must add pixels).
func TestIconButtonFocusRingIsVisuallyDistinct(t *testing.T) {
	size := image.Pt(60, 60)

	imgNormal := golden.Capture(t, size, button.RenderIcon(
		crossIcon,
		tokens.DefaultLight, tokens.Spacing, tokens.Radius, tokens.DefaultTypeScale,
		button.RenderState{},
	))
	imgFocused := golden.Capture(t, size, button.RenderIcon(
		crossIcon,
		tokens.DefaultLight, tokens.Spacing, tokens.Radius, tokens.DefaultTypeScale,
		button.RenderState{Focused: true},
	))

	if imgNormal == nil || imgFocused == nil {
		return
	}
	if n := golden.PixelDiff(imgNormal, imgFocused); n == 0 {
		t.Error("focused and idle icon buttons render identically; expected focus ring pixels to differ")
	}
}

// TestButtonInjectedClickableFocusAndActivate proves the caller-owned-clickable
// path (GX.3): a container drives focus to the supplied *widget.Clickable via
// key.FocusCmd, and Space activation flows through it to OnClick. This is the
// mechanism cadence/modal's focus trap relies on for the close button.
func TestButtonInjectedClickableFocusAndActivate(t *testing.T) {
	shaper := defaultShaper(t)
	var clicked int
	var click widget.Clickable

	obs := button.Button(rx.Of(theme.Default()), button.Props{
		Icon:        crossIcon,
		Description: "Close",
		Clickable:   &click,
		OnClick:     func(_ layout.Context) { clicked++ },
		Shaper:      shaper,
	})
	var w layout.Widget
	if err := obs.Subscribe(context.Background(), func(next layout.Widget, _ error, done bool) {
		if !done && next != nil {
			w = next
		}
	}).Wait(); err != nil {
		t.Fatalf("Button subscribe: %v", err)
	}
	if w == nil {
		t.Fatal("Button did not emit a widget")
	}

	r := new(gioinput.Router)
	ops := new(op.Ops)
	size := image.Pt(44, 44)

	drive := func(cw layout.Widget) {
		ops.Reset()
		gtx := layout.Context{
			Metric:      unit.Metric{PxPerDp: 1, PxPerSp: 1},
			Constraints: layout.Exact(size),
			Ops:         ops,
			Source:      r.Source(),
		}
		cw(gtx)
		r.Frame(ops)
	}

	// Frame 1: lay out the button (registers the clickable's focus filter),
	// then a container drives focus to the caller-owned tag.
	focusOnce := true
	composed := func(gtx layout.Context) layout.Dimensions {
		dims := w(gtx)
		if focusOnce {
			gtx.Execute(key.FocusCmd{Tag: &click})
			focusOnce = false
		}
		return dims
	}
	drive(composed)
	// Frame 2: focus is applied.
	drive(w)

	probe := layout.Context{
		Metric:      unit.Metric{PxPerDp: 1, PxPerSp: 1},
		Constraints: layout.Exact(size),
		Ops:         new(op.Ops),
		Source:      r.Source(),
	}
	if !probe.Focused(&click) {
		t.Fatal("injected clickable not focused after key.FocusCmd; container-driven focus failed")
	}

	// Space while focused activates the button → OnClick fires through the
	// caller-owned clickable.
	r.Queue(
		key.Event{Name: key.NameSpace, State: key.Press},
		key.Event{Name: key.NameSpace, State: key.Release},
	)
	drive(w)
	if clicked != 1 {
		t.Errorf("Space activation: OnClick fired %d times, want 1", clicked)
	}
}

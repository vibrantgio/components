package input_test

import (
	"image"
	stdcolor "image/color"
	"testing"

	"gioui.org/f32"
	gioinput "gioui.org/io/input"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"

	"github.com/reactivego/rx"
	golden "github.com/vibrantgio/components/golden"
	"github.com/vibrantgio/components/input"
	"github.com/vibrantgio/components/internal/focus"
	tcolor "github.com/vibrantgio/theme/color"
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

// TestCheckboxChecksAreDrawn is the other half of "visually distinct", and
// the half that was missing. A checked box used to be a plain Primary square:
// distinct from the unchecked one, so the diff test above was satisfied, and
// distinct only in hue — which reads as a swatch, or as the indeterminate
// state, to anyone the hue does not reach. So this asserts the mark itself:
// how much of the checked box reads as check rather than as fill, in both
// schemes.
//
// The count is a band rather than a number because a stroked figure two
// pixels wide is mostly its own anti-aliased edge. At the 1 px/dp the harness
// pins, the mark covers 40 px of the light box and 28 of the dark one; the
// band's job is to fail both a mark that vanished and a mark that swallowed
// the box, not to pin the figure — that is what the goldens are for.
func TestCheckboxChecksAreDrawn(t *testing.T) {
	size := image.Pt(44, 44)
	for _, scheme := range []struct {
		name   string
		colors tokens.ColorTokens
	}{
		{"light", tokens.DefaultLight},
		{"dark", tokens.DefaultDark},
	} {
		c := scheme.colors
		count := func(s input.CheckboxRenderState) int {
			img := golden.Capture(t, size, input.RenderCheckbox(c, tokens.Spacing, tokens.Radius, s))
			if img == nil {
				return -1
			}
			n := 0
			b := img.Bounds()
			for y := b.Min.Y; y < b.Max.Y; y++ {
				for x := b.Min.X; x < b.Max.X; x++ {
					if nearerTo(img.RGBAAt(x, y), c.OnPrimary, c.Primary) {
						n++
					}
				}
			}
			return n
		}

		n := count(input.CheckboxRenderState{Checked: true})
		switch {
		case n < 15:
			t.Errorf("%s: a checked box carries %d pixels of check ink %v over the fill %v — the mark is missing",
				scheme.name, n, c.OnPrimary, c.Primary)
		case n > 120:
			t.Errorf("%s: a checked box carries %d pixels of check ink %v over the fill %v — the mark has taken over the box",
				scheme.name, n, c.OnPrimary, c.Primary)
		default:
			t.Logf("%s: the check covers %d px of the 20 dp box", scheme.name, n)
		}
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

// TestFocusIsVisibleOnEveryControlInEveryState is the walkthrough defect,
// written down. A focused checkbox and a focused radio used to show no ring
// at all, and the code and the eye were both right about why: a ring was
// stroked, in neutral step 500, on the boundary of a box whose border is
// neutral step 500 — 1.00:1 against the very edge it was circling — and the
// box then overdrew its inner half, so the one thing that escaped was a
// single device pixel of grey against grey.
//
// Colour alone would not have fixed it either, and the chosen radio is the
// proof: its edge is already primary, so promoting that edge on focus moves
// primary to a neighbouring rung of primary and a chosen radio looks the same
// focused as unfocused. So the assertion is the one that catches both: the
// ring colour is absent from the resting control and present in the focused
// one — in every state each control has, in both schemes.
func TestFocusIsVisibleOnEveryControlInEveryState(t *testing.T) {
	size := image.Pt(44, 44)
	shaper := defaultShaper(t)

	for _, scheme := range []struct {
		name   string
		colors tokens.ColorTokens
	}{
		{"light", tokens.DefaultLight},
		{"dark", tokens.DefaultDark},
	} {
		c := scheme.colors
		ring := focus.Ring(c, c.Surface)
		if got := tcolor.ContrastRatio(ring, c.Surface); got < focus.Floor {
			t.Errorf("%s: ring %v measures %.2f:1 against the surface it lies on %v",
				scheme.name, ring, got, c.Surface)
		}

		count := func(sz image.Point, w layout.Widget) int {
			img := golden.Capture(t, sz, w)
			n := 0
			b := img.Bounds()
			for y := b.Min.Y; y < b.Max.Y; y++ {
				for x := b.Min.X; x < b.Max.X; x++ {
					if nearlyEqual(img.RGBAAt(x, y), ring) {
						n++
					}
				}
			}
			return n
		}

		for _, control := range []struct {
			name  string
			size  image.Point
			idle  layout.Widget
			focus layout.Widget
		}{
			{"checkbox unchecked", size,
				input.RenderCheckbox(c, tokens.Spacing, tokens.Radius, input.CheckboxRenderState{}),
				input.RenderCheckbox(c, tokens.Spacing, tokens.Radius, input.CheckboxRenderState{Focused: true})},
			{"checkbox checked", size,
				input.RenderCheckbox(c, tokens.Spacing, tokens.Radius, input.CheckboxRenderState{Checked: true}),
				input.RenderCheckbox(c, tokens.Spacing, tokens.Radius, input.CheckboxRenderState{Checked: true, Focused: true})},
			{"radio unselected", size,
				input.RenderRadio(c, tokens.Spacing, tokens.Radius, input.RadioRenderState{}),
				input.RenderRadio(c, tokens.Spacing, tokens.Radius, input.RadioRenderState{Focused: true})},
			{"radio selected", size,
				input.RenderRadio(c, tokens.Spacing, tokens.Radius, input.RadioRenderState{Selected: true}),
				input.RenderRadio(c, tokens.Spacing, tokens.Radius, input.RadioRenderState{Selected: true, Focused: true})},
			{"text field", image.Pt(300, 60),
				input.Render(shaper, "you@example.com", c, tokens.Spacing, tokens.Radius,
					tokens.DefaultTypography.BodyLarge, tokens.Comfortable, input.RenderState{}),
				input.Render(shaper, "you@example.com", c, tokens.Spacing, tokens.Radius,
					tokens.DefaultTypography.BodyLarge, tokens.Comfortable, input.RenderState{Focused: true})},
			{"dropdown trigger", image.Pt(200, 44),
				input.RenderDropdown(shaper, c, tokens.Spacing, tokens.Radius,
					tokens.DefaultTypography.BodyLarge, tokens.Comfortable,
					input.DropdownRenderState{Options: []string{"One", "Two"}}),
				input.RenderDropdown(shaper, c, tokens.Spacing, tokens.Radius,
					tokens.DefaultTypography.BodyLarge, tokens.Comfortable,
					input.DropdownRenderState{Focused: true, Options: []string{"One", "Two"}})},
		} {
			if n := count(control.size, control.idle); n != 0 {
				t.Errorf("%s %s: %d pixels of the ring colour %v with nothing focused",
					scheme.name, control.name, n, ring)
			}
			if n := count(control.size, control.focus); n == 0 {
				t.Errorf("%s %s: focused, and not one pixel of the ring colour %v",
					scheme.name, control.name, ring)
			}
		}
	}
}

// nearlyEqual reports whether a captured pixel is the given token colour, to
// within the three units per channel the GPU's own rounding moves a flat fill
// by. The colours compared against here — the ring's rung and the neutral
// border it replaces — are nowhere near each other, so the slack costs
// nothing.
func nearlyEqual(got stdcolor.RGBA, want stdcolor.NRGBA) bool {
	const slack = 3
	off := func(a, b uint8) bool {
		if a > b {
			return a-b > slack
		}
		return b-a > slack
	}
	return !off(got.R, want.R) && !off(got.G, want.G) && !off(got.B, want.B) && got.A == want.A
}

// nearerTo reports whether a captured pixel lies closer to ink than to fill.
// An anti-aliased figure a couple of pixels wide has an interior of exactly
// its ink and edges of everything between ink and ground, so counting exact
// matches counts the interior only — one pixel, on a dark scheme's mark. The
// question worth asking of a stroked mark is how much of the box reads as
// mark rather than as fill, and that is this: squared distance in RGB, the
// two ends of the blend as the only candidates.
func nearerTo(got stdcolor.RGBA, ink, fill stdcolor.NRGBA) bool {
	d := func(c stdcolor.NRGBA) int {
		dr := int(got.R) - int(c.R)
		dg := int(got.G) - int(c.G)
		db := int(got.B) - int(c.B)
		return dr*dr + dg*dg + db*db
	}
	return got.A == 0xff && d(ink) < d(fill)
}

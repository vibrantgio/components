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
		name     string
		platform tokens.PlatformColors
		state    input.CheckboxRenderState
	}{
		// Every name carries the component prefix: all four inputs share one
		// testdata/golden directory, so an unprefixed name risks one golden
		// file silently serving two different components' renders.
		{"checkbox-light-unchecked", tokens.PlatformLight, input.CheckboxRenderState{}},
		{"checkbox-dark-unchecked", tokens.PlatformDark, input.CheckboxRenderState{}},
		{"checkbox-light-checked", tokens.PlatformLight, input.CheckboxRenderState{Checked: true}},
		{"checkbox-light-focused", tokens.PlatformLight, input.CheckboxRenderState{Focused: true}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := input.RenderCheckbox(tc.platform, tokens.Spacing, sharpRadius, tc.state)
			golden.Render(t, tc.name, size, w)
		})
	}
}

// ---- Accessibility tests ----

// TestCheckboxFootprintIsControlHeight checks the checkbox's visual footprint
// is the density's control-height square with the measured 16 dp glyph centred
// in it. The 44 dp WCAG 2.5.5 floor applies to the pointer target, not the
// footprint: the live Checkbox extends its hit area via internal/hit,
// exercised by TestCheckboxHitSlopToggles.
func TestCheckboxFootprintIsControlHeight(t *testing.T) {
	var ops op.Ops
	gtx := layout.Context{
		Metric:      unit.Metric{PxPerDp: 1, PxPerSp: 1},
		Constraints: layout.Exact(image.Pt(120, 120)),
		Ops:         &ops,
	}

	dims := input.RenderCheckbox(
		tokens.PlatformLight,
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

	// The hit rect is the 44 px floor centred on the ControlHeight footprint,
	// so it reaches (44-ControlHeight)/2 px past each edge. Click one px
	// outside the footprint's far corner — outside the glyph's row, inside
	// the slop.
	pos := f32.Pt(float32(tokens.Comfortable.ControlHeight)+1, float32(tokens.Comfortable.ControlHeight)+1)
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
// through the live pipeline: the 16 dp glyph centred in the Compact control
// height, which is smaller than the glyph's own row at Comfortable.
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
		tokens.PlatformLight, tokens.Spacing, tokens.Radius,
		input.CheckboxRenderState{},
	))
	imgChecked := golden.Capture(t, size, input.RenderCheckbox(
		tokens.PlatformLight, tokens.Spacing, tokens.Radius,
		input.CheckboxRenderState{Checked: true},
	))

	if imgUnchecked == nil || imgChecked == nil {
		return
	}
	if n := golden.PixelDiff(imgUnchecked, imgChecked); n == 0 {
		t.Error("checked and unchecked checkboxes render identically; expected visual difference")
	}
}

// TestCheckboxChecksAreDrawn asserts the check mark itself is present, not
// merely that the checked state differs in colour from the fill it is drawn
// on — a colour-only difference reads as a swatch, or as an indeterminate
// state, to anyone who cannot use hue. It measures how much of the checked
// box reads as mark versus as fill, in both schemes.
//
// The count is a band rather than a number because a stroked figure two
// pixels wide is mostly its own anti-aliased edge. The band's job is to fail
// both a mark that vanished and a mark that swallowed the box, not to pin the
// figure — that is what the goldens are for.
func TestCheckboxChecksAreDrawn(t *testing.T) {
	size := image.Pt(44, 44)
	for _, scheme := range []struct {
		name     string
		platform tokens.PlatformColors
	}{
		{"light", tokens.PlatformLight},
		{"dark", tokens.PlatformDark},
	} {
		p := scheme.platform
		mark, fill := p.AlternateSelectedControlText, p.ControlAccent
		count := func(s input.CheckboxRenderState) int {
			img := golden.Capture(t, size, input.RenderCheckbox(p, tokens.Spacing, tokens.Radius, s))
			if img == nil {
				return -1
			}
			n := 0
			b := img.Bounds()
			for y := b.Min.Y; y < b.Max.Y; y++ {
				for x := b.Min.X; x < b.Max.X; x++ {
					if nearerTo(img.RGBAAt(x, y), mark, fill) {
						n++
					}
				}
			}
			return n
		}

		n := count(input.CheckboxRenderState{Checked: true})
		switch {
		case n < 10:
			t.Errorf("%s: a checked box carries %d pixels of check mark %v over the fill %v — the mark is missing",
				scheme.name, n, mark, fill)
		case n > 90:
			t.Errorf("%s: a checked box carries %d pixels of check mark %v over the fill %v — the mark has taken over the box",
				scheme.name, n, mark, fill)
		default:
			t.Logf("%s: the check covers %d px of the 16 dp box", scheme.name, n)
		}
	}
}

// TestCheckboxFocusRingIsVisuallyDistinct confirms the focused state renders
// differently from the normal state (focus ring must add pixels).
func TestCheckboxFocusRingIsVisuallyDistinct(t *testing.T) {
	size := image.Pt(44, 44)

	imgNormal := golden.Capture(t, size, input.RenderCheckbox(
		tokens.PlatformLight, tokens.Spacing, tokens.Radius,
		input.CheckboxRenderState{},
	))
	imgFocused := golden.Capture(t, size, input.RenderCheckbox(
		tokens.PlatformLight, tokens.Spacing, tokens.Radius,
		input.CheckboxRenderState{Focused: true},
	))

	if imgNormal == nil || imgFocused == nil {
		return
	}
	if n := golden.PixelDiff(imgNormal, imgFocused); n == 0 {
		t.Error("focused and normal checkboxes render identically; expected focus ring pixels to differ")
	}
}

// TestFocusIsVisibleOnEveryControlInEveryState asserts that the ring colour
// is absent from every control's resting state and present in its focused
// state, across every state each control has and in both colour schemes.
// A colour-only signal is not enough on its own: a chosen radio is already
// drawn in the accent, so promoting its own edge on focus would move one blue
// to a neighbouring blue rather than adding a distinguishable mark. The
// assertion instead counts the ring's own colour at the pixel level, in both
// the resting and focused renders.
//
// It counts the colour the ring lands as rather than the value the platform
// publishes. The keyboard focus indicator carries a coverage of its own, and
// focus.Ring resolves it against what lies under the band before anything is
// painted — for a checkbox and a radio, the surface the control stands on,
// which is the window's plane unless the caller says otherwise — so what a
// capture holds is that opaque answer.
func TestFocusIsVisibleOnEveryControlInEveryState(t *testing.T) {
	size := image.Pt(44, 44)
	shaper := defaultShaper(t)

	for _, scheme := range []struct {
		name     string
		platform tokens.PlatformColors
	}{
		{"light", tokens.PlatformLight},
		{"dark", tokens.PlatformDark},
	} {
		p := scheme.platform
		// A control rings on what the band actually lies on: the surface
		// for a glyph whose ring rides in the slack beside it, the
		// control's own fill for a trigger whose edge IS the ring.
		onPlane := focus.Ring(p, p.WindowBackground)
		onTrigger := focus.Ring(p, p.PushButtonFill)

		count := func(sz image.Point, w layout.Widget, ring stdcolor.NRGBA) int {
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
			ring  stdcolor.NRGBA
		}{
			{"checkbox unchecked", size,
				input.RenderCheckbox(p, tokens.Spacing, tokens.Radius, input.CheckboxRenderState{}),
				input.RenderCheckbox(p, tokens.Spacing, tokens.Radius, input.CheckboxRenderState{Focused: true}), onPlane},
			{"checkbox checked", size,
				input.RenderCheckbox(p, tokens.Spacing, tokens.Radius, input.CheckboxRenderState{Checked: true}),
				input.RenderCheckbox(p, tokens.Spacing, tokens.Radius, input.CheckboxRenderState{Checked: true, Focused: true}), onPlane},
			{"radio unselected", size,
				input.RenderRadio(p, tokens.Spacing, tokens.Radius, input.RadioRenderState{}),
				input.RenderRadio(p, tokens.Spacing, tokens.Radius, input.RadioRenderState{Focused: true}), onPlane},
			{"radio selected", size,
				input.RenderRadio(p, tokens.Spacing, tokens.Radius, input.RadioRenderState{Selected: true}),
				input.RenderRadio(p, tokens.Spacing, tokens.Radius, input.RadioRenderState{Selected: true, Focused: true}), onPlane},
			{"text field", image.Pt(300, 60),
				input.Render(shaper, "you@example.com", p, tokens.Spacing, tokens.Radius,
					tokens.DefaultTypography.BodyLarge, tokens.Comfortable, input.RenderState{}),
				input.Render(shaper, "you@example.com", p, tokens.Spacing, tokens.Radius,
					tokens.DefaultTypography.BodyLarge, tokens.Comfortable, input.RenderState{Focused: true}), onPlane},
			{"search field", image.Pt(300, 60),
				input.RenderSearch(shaper, "Search", p, tokens.Spacing, tokens.Radius,
					tokens.DefaultTypography.BodyLarge, tokens.Comfortable, input.RenderState{}),
				input.RenderSearch(shaper, "Search", p, tokens.Spacing, tokens.Radius,
					tokens.DefaultTypography.BodyLarge, tokens.Comfortable, input.RenderState{Focused: true}), onPlane},
			{"dropdown trigger", image.Pt(200, 44),
				input.RenderDropdown(shaper, p, tokens.Spacing, tokens.Radius,
					tokens.DefaultTypography.BodyLarge, tokens.Comfortable,
					input.DropdownRenderState{Options: []string{"One", "Two"}}),
				input.RenderDropdown(shaper, p, tokens.Spacing, tokens.Radius,
					tokens.DefaultTypography.BodyLarge, tokens.Comfortable,
					input.DropdownRenderState{Focused: true, Options: []string{"One", "Two"}}), onTrigger},
		} {
			if n := count(control.size, control.idle, control.ring); n != 0 {
				t.Errorf("%s %s: %d pixels of the ring colour %v with nothing focused",
					scheme.name, control.name, n, control.ring)
			}
			if n := count(control.size, control.focus, control.ring); n == 0 {
				t.Errorf("%s %s: focused, and not one pixel of the ring colour %v",
					scheme.name, control.name, control.ring)
			}
		}
	}
}

// nearlyEqual reports whether a captured pixel carries the given colour, to
// within the three units per channel the GPU's own rounding moves a flat fill
// by. Alpha is not compared: a translucent fill keeps its own coverage in the
// capture while the colour it is checked against is the opaque composite, and
// the colours compared here are nowhere near each other on the three channels
// that are.
func nearlyEqual(got stdcolor.RGBA, want stdcolor.NRGBA) bool {
	const slack = 3
	off := func(a, b uint8) bool {
		if a > b {
			return a-b > slack
		}
		return b-a > slack
	}
	return !off(got.R, want.R) && !off(got.G, want.G) && !off(got.B, want.B)
}

// nearerTo reports whether a captured pixel lies closer to the mark than to
// fill. An anti-aliased figure a couple of pixels wide has an interior of
// exactly its own colour and edges of everything between the two, so counting exact
// matches counts the interior only — one pixel, on a dark scheme's mark. The
// question worth asking of a stroked mark is how much of the box reads as
// mark rather than as fill, and that is this: squared distance in RGB, the
// two ends of the blend as the only candidates.
func nearerTo(got stdcolor.RGBA, mark, fill stdcolor.NRGBA) bool {
	d := func(c stdcolor.NRGBA) int {
		dr := int(got.R) - int(c.R)
		dg := int(got.G) - int(c.G)
		db := int(got.B) - int(c.B)
		return dr*dr + dg*dg + db*db
	}
	return got.A == 0xff && d(mark) < d(fill)
}

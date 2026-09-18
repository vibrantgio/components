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
	"github.com/vibrantgio/components/internal/control"
	"github.com/vibrantgio/components/internal/focus"
	vgcolor "github.com/vibrantgio/theme/color"
	"github.com/vibrantgio/theme/theme"
	"github.com/vibrantgio/theme/tokens"
)

// ---- Golden-image tests ----

// TestCheckboxGolden records or diffs the four canonical checkbox states.
func TestCheckboxGolden(t *testing.T) {
	size := image.Pt(44, 44)

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
			w := input.RenderCheckbox(nil, tc.platform, tokens.Spacing, tokens.DefaultTypography.BodyLarge, tc.state)
			golden.Render(t, tc.name, size, w)
		})
	}
}

// ---- Accessibility tests ----

// TestCheckboxFootprintIsItsMeasuredRow checks the checkbox's visual
// footprint is the density's measured checkbox row, a square with the
// measured 16 dp glyph centred in it. That footprint is the pointer target
// too — the checkbox's row, not its glyph — which TestCheckboxTargetIsItsRow
// exercises.
func TestCheckboxFootprintIsItsMeasuredRow(t *testing.T) {
	var ops op.Ops
	gtx := layout.Context{
		Metric:      unit.Metric{PxPerDp: 1, PxPerSp: 1},
		Constraints: layout.Exact(image.Pt(120, 120)),
		Ops:         &ops,
	}

	dims := input.RenderCheckbox(
		nil,
		tokens.PlatformLight,
		tokens.Spacing,
		tokens.DefaultTypography.BodyLarge,
		input.CheckboxRenderState{},
	)(gtx)

	want := int(tokens.Comfortable.CheckboxRowHeight)
	if dims.Size.X != want || dims.Size.Y != want {
		t.Errorf("checkbox footprint = %v, want %dx%d px (CheckboxRowHeight square at 1:1 scale)", dims.Size, want, want)
	}
}

// TestCheckboxTargetIsItsRow checks the live checkbox's pointer target is the
// footprint the glyph is centred in and not the 16 dp glyph: a click in the
// footprint's corner, clear of the glyph, toggles the value, and a click
// outside the footprint does not.
func TestCheckboxTargetIsItsRow(t *testing.T) {
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

	// The footprint's far corner: inside the row, outside the 16 dp glyph
	// centred in it, which is the only place the two can be told apart.
	pos := f32.Pt(float32(tokens.Comfortable.CheckboxRowHeight)-1, float32(tokens.Comfortable.CheckboxRowHeight)-1)
	r.Queue(
		pointer.Event{Kind: pointer.Press, Position: pos, Buttons: pointer.ButtonPrimary, Source: pointer.Mouse},
		pointer.Event{Kind: pointer.Release, Position: pos, Buttons: pointer.ButtonPrimary, Source: pointer.Mouse},
	)
	drive()
	if toggled != 1 {
		t.Errorf("click in the footprint's corner, clear of the glyph: OnChange fired %d times, want 1", toggled)
	}

	outside := f32.Pt(float32(tokens.Comfortable.CheckboxRowHeight)+4, float32(tokens.Comfortable.CheckboxRowHeight)+4)
	r.Queue(
		pointer.Event{Kind: pointer.Press, Position: outside, Buttons: pointer.ButtonPrimary, Source: pointer.Mouse},
		pointer.Event{Kind: pointer.Release, Position: outside, Buttons: pointer.ButtonPrimary, Source: pointer.Mouse},
	)
	drive()
	if toggled != 1 {
		t.Errorf("click outside the footprint: OnChange fired %d times in total, want 1 — the target is the row", toggled)
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
		nil,
		tokens.PlatformLight, tokens.Spacing,
		tokens.DefaultTypography.BodyLarge,
		input.CheckboxRenderState{},
	))
	imgChecked := golden.Capture(t, size, input.RenderCheckbox(
		nil,
		tokens.PlatformLight, tokens.Spacing,
		tokens.DefaultTypography.BodyLarge,
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
			img := golden.Capture(t, size, input.RenderCheckbox(nil, p, tokens.Spacing, tokens.DefaultTypography.BodyLarge, s))
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
		nil,
		tokens.PlatformLight, tokens.Spacing,
		tokens.DefaultTypography.BodyLarge,
		input.CheckboxRenderState{},
	))
	imgFocused := golden.Capture(t, size, input.RenderCheckbox(
		nil,
		tokens.PlatformLight, tokens.Spacing,
		tokens.DefaultTypography.BodyLarge,
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
				input.RenderCheckbox(shaper, p, tokens.Spacing, tokens.DefaultTypography.BodyLarge, input.CheckboxRenderState{}),
				input.RenderCheckbox(shaper, p, tokens.Spacing, tokens.DefaultTypography.BodyLarge, input.CheckboxRenderState{Focused: true}), onPlane},
			{"checkbox checked", size,
				input.RenderCheckbox(shaper, p, tokens.Spacing, tokens.DefaultTypography.BodyLarge, input.CheckboxRenderState{Checked: true}),
				input.RenderCheckbox(shaper, p, tokens.Spacing, tokens.DefaultTypography.BodyLarge, input.CheckboxRenderState{Checked: true, Focused: true}), onPlane},
			{"radio unselected", size,
				input.RenderRadio(shaper, p, tokens.Spacing, tokens.DefaultTypography.BodyLarge, input.RadioRenderState{}),
				input.RenderRadio(shaper, p, tokens.Spacing, tokens.DefaultTypography.BodyLarge, input.RadioRenderState{Focused: true}), onPlane},
			{"radio selected", size,
				input.RenderRadio(shaper, p, tokens.Spacing, tokens.DefaultTypography.BodyLarge, input.RadioRenderState{Selected: true}),
				input.RenderRadio(shaper, p, tokens.Spacing, tokens.DefaultTypography.BodyLarge, input.RadioRenderState{Selected: true, Focused: true}), onPlane},
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

// switchedOffReadings is the stored reference's switched-off control, one row
// per appearance. MEASURED, save-dialog-light.png and save-dialog-dark.png:
// the two "Options:" checkboxes are the only switched-off controls on the
// sheet and their 16 px boxes read #f2f2f2 and #2e3439, on sheets of #ffffff
// and #232a2f; the enabled "File Format:" pop-up seventeen rows above them
// reads the push button's own #ececec and #333a3f on the same sheet. The
// slack is what the platform's disabled coverage lands over on that enabled
// fill: nothing in light, one 255th on green and blue in dark, which is the
// tolerance the same reference carries for the hover overlay's dark reading.
var switchedOffReadings = []struct {
	name     string
	platform tokens.PlatformColors
	sheet    stdcolor.NRGBA
	box      stdcolor.NRGBA
	slack    int
}{
	{"light", tokens.PlatformLight, stdcolor.NRGBA{0xff, 0xff, 0xff, 0xff}, stdcolor.NRGBA{0xf2, 0xf2, 0xf2, 0xff}, 0},
	{"dark", tokens.PlatformDark, stdcolor.NRGBA{0x23, 0x2a, 0x2f, 0xff}, stdcolor.NRGBA{0x2e, 0x34, 0x39, 0xff}, 1},
}

// off reports how far apart two colours are on their widest channel.
func off(a, b stdcolor.NRGBA) int {
	d := func(x, y uint8) int {
		if x > y {
			return int(x) - int(y)
		}
		return int(y) - int(x)
	}
	n := d(a.R, b.R)
	if v := d(a.G, b.G); v > n {
		n = v
	}
	if v := d(a.B, b.B); v > n {
		n = v
	}
	return n
}

// TestTheSwitchedOffBoxIsOneFillAndNoEdge reads the switched-off checkbox off
// a capture of the component, in both appearances, and against the stored
// reference's pixels.
//
// Two things are read. The fill: the platform's control fill at the measured
// disabled coverage over the surface the box stands on, which is what the
// push button does when it is switched off and what puts the box on the
// reference's #f2f2f2 and #2e3439. And the edge: the platform's switched-off
// box draws no edge column at all — its rim is a one-pixel antialiased ramp
// from the fill to the sheet — so every row and column of the box carries the
// interior's own colour, where the enabled box's first row carries the field
// edge instead.
//
// Every row and column is read at the box's own middle, where the measured
// corner cannot reach and the drawing is flat fill or flat edge.
func TestTheSwitchedOffBoxIsOneFillAndNoEdge(t *testing.T) {
	const size = 44

	// The glyph's measured 16 dp box, centred in the density's checkbox row,
	// at the 1:1 metric golden.Capture renders at.
	const box = 16
	row := int(tokens.Comfortable.CheckboxRowHeight)
	lo := (row - box) / 2
	hi := lo + box - 1
	mid := lo + box/2

	for _, sc := range switchedOffReadings {
		t.Run(sc.name, func(t *testing.T) {
			want := control.Faded(sc.platform.PushButtonFill, sc.sheet)
			if n := off(want, sc.box); n > sc.slack {
				t.Errorf("the platform's control fill at the disabled coverage lands on %v, want the capture's %v within %d", want, sc.box, sc.slack)
			}

			img := golden.Capture(t, image.Pt(size, size), input.RenderCheckbox(
				nil,
				sc.platform, tokens.Spacing,
				tokens.DefaultTypography.BodyLarge,
				input.CheckboxRenderState{Disabled: true, Surface: sc.sheet},
			))
			if img == nil {
				return
			}
			if got := img.RGBAAt(mid, mid); !nearlyEqual(got, want) {
				t.Errorf("a switched-off box fills %v, want the platform's control fill at the disabled coverage %v", got, want)
			}
			centre := img.RGBAAt(mid, mid)
			for _, at := range []struct {
				what string
				x, y int
			}{
				{"first row", mid, lo},
				{"last row", mid, hi},
				{"first column", lo, mid},
				{"last column", hi, mid},
			} {
				if got := img.RGBAAt(at.x, at.y); got != centre {
					t.Errorf("a switched-off box's %s reads %v against its interior's %v; the platform draws no edge on one", at.what, got, centre)
				}
			}

			// The enabled box is what an edge row looks like, and it still
			// draws one.
			on := golden.Capture(t, image.Pt(size, size), input.RenderCheckbox(
				nil,
				sc.platform, tokens.Spacing,
				tokens.DefaultTypography.BodyLarge,
				input.CheckboxRenderState{Surface: sc.sheet},
			))
			if on == nil {
				return
			}
			if on.RGBAAt(mid, lo) == on.RGBAAt(mid, mid) {
				t.Errorf("an enabled box's first row reads its interior's %v; the enabled box keeps its edge", on.RGBAAt(mid, mid))
			}
		})
	}
}

// TestTheSwitchedOffBoxKeepsItsMark asserts that a switched-off box and radio
// still say whether a value is set, and say it in the colour the switched-off
// label beside them is drawn in rather than in the accent the platform draws
// on no switched-off control at all.
//
// No stored capture holds a switched-off checked box or a switched-off
// selected radio, so what is pinned here is the rule and not a pixel: the
// fill does not move between set and unset, the mark does, and it is the
// platform's tertiary label over that fill.
func TestTheSwitchedOffBoxKeepsItsMark(t *testing.T) {
	const size = 44

	for _, sc := range switchedOffReadings {
		p := sc.platform
		fill := control.Faded(p.PushButtonFill, sc.sheet)
		mark := vgcolor.Flatten(p.TertiaryLabel, fill)

		for _, cs := range []struct {
			what  string
			unset layout.Widget
			set   layout.Widget
		}{
			{"box",
				input.RenderCheckbox(nil, p, tokens.Spacing, tokens.DefaultTypography.BodyLarge, input.CheckboxRenderState{Disabled: true, Surface: sc.sheet}),
				input.RenderCheckbox(nil, p, tokens.Spacing, tokens.DefaultTypography.BodyLarge, input.CheckboxRenderState{Disabled: true, Checked: true, Surface: sc.sheet})},
			{"radio",
				input.RenderRadio(nil, p, tokens.Spacing, tokens.DefaultTypography.BodyLarge, input.RadioRenderState{Disabled: true, Surface: sc.sheet}),
				input.RenderRadio(nil, p, tokens.Spacing, tokens.DefaultTypography.BodyLarge, input.RadioRenderState{Disabled: true, Selected: true, Surface: sc.sheet})},
		} {
			unset := golden.Capture(t, image.Pt(size, size), cs.unset)
			set := golden.Capture(t, image.Pt(size, size), cs.set)
			if unset == nil || set == nil {
				return
			}
			if golden.PixelDiff(unset, set) == 0 {
				t.Errorf("%s %s: a switched-off control with a value set is the same pixels as one without; the mark says which", sc.name, cs.what)
			}
			marked, accented := 0, 0
			b := set.Bounds()
			for y := b.Min.Y; y < b.Max.Y; y++ {
				for x := b.Min.X; x < b.Max.X; x++ {
					got := set.RGBAAt(x, y)
					if nearerTo(got, mark, fill) {
						marked++
					}
					if nearlyEqual(got, p.ControlAccent) {
						accented++
					}
				}
			}
			if marked == 0 {
				t.Errorf("%s %s: a switched-off control with a value set carries no pixel of its mark %v over the fill %v", sc.name, cs.what, mark, fill)
			}
			if accented != 0 {
				t.Errorf("%s %s: a switched-off control carries %d pixels of the accent %v; the platform draws none on one", sc.name, cs.what, accented, p.ControlAccent)
			}
		}
	}
}

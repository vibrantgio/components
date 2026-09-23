package input_test

import (
	"context"
	"image"
	stdcolor "image/color"
	"testing"

	"gioui.org/f32"
	"gioui.org/io/event"
	gioinput "gioui.org/io/input"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"

	"github.com/reactivego/rx"
	golden "github.com/vibrantgio/components/golden"
	"github.com/vibrantgio/components/input"
	"github.com/vibrantgio/components/internal/control"
	vgcolor "github.com/vibrantgio/theme/color"
	"github.com/vibrantgio/theme/theme"
	"github.com/vibrantgio/theme/tokens"
)

// defaultShaper returns the shaper every golden here draws with: the default
// typography's faces pinned, system fonts off, so the stored images are the
// same on every machine. A golden test pins its faces with
// DeterministicShaper; application code takes the fallback Shaper.
func defaultShaper(t *testing.T) *text.Shaper {
	t.Helper()
	return tokens.DefaultTypography.DeterministicShaper()
}

// ---- Golden-image tests ----

// TestTextFieldGolden records or diffs the four canonical text field states.
func TestTextFieldGolden(t *testing.T) {
	shaper := defaultShaper(t)
	size := image.Pt(300, 60)

	// Zero corner radius keeps the edges sharp: anti-aliased rounded corners
	// vary between GPU context initialisations. The placeholder is real text —
	// the faces are pinned by DeterministicShaper, so Latin glyphs rasterise
	// identically on every machine and the placeholder's own type role is on
	// screen where a regression in it would show.
	sharpRadius := tokens.RadiusScale{}
	// Disabled is intentionally omitted: semi-transparent disabled colours
	// composite non-deterministically against the headless window background.
	// The disabled visual is tested separately in TestTextFieldDisabledIsVisuallyDistinct.
	cases := []struct {
		name     string
		platform tokens.PlatformColors
		state    input.RenderState
	}{
		// Prefixed per component: see the note in checkbox_test.go. These four
		// share testdata/golden with the checkbox, radio and dropdown cases.
		{"textfield-light-normal", tokens.PlatformLight, input.RenderState{}},
		{"textfield-dark-normal", tokens.PlatformDark, input.RenderState{}},
		{"textfield-light-focused", tokens.PlatformLight, input.RenderState{Focused: true}},
		{"textfield-light-focused-with-text", tokens.PlatformLight, input.RenderState{Focused: true, Text: "hello@example.com"}},
		// A selected run, which the platform draws in a second colour: the
		// first of the two words is selected and the second is not.
		{"textfield-light-selected", tokens.PlatformLight, input.RenderState{Focused: true, Text: selectedRunValue, Selection: [2]int{0, 8}}},
		{"textfield-dark-selected", tokens.PlatformDark, input.RenderState{Focused: true, Text: selectedRunValue, Selection: [2]int{0, 8}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := input.Render(
				shaper, "Email address",
				tc.platform, tokens.Spacing, sharpRadius, tokens.DefaultTypography.BodyLarge, tokens.Comfortable,
				tc.state,
			)
			golden.Render(t, tc.name, size, w)
		})
	}
}

// TestTextFieldLeadingInsetIsMeasured reads off the drawn pixels the number
// that places the text in a field carrying no looking glass: the field's
// inner edge to the first pixel of the text standing in it. The prompt and a
// typed value are read the same way, because the two are drawn at one inset.
//
// MEASURED, save-dialog-{light,dark}.png, the "Save As:" field at 1x: the
// field's box runs x 264–495, the columns the unfocused "Tags:" field below
// it runs, so its fill begins at x=265; the value's first covered pixel
// column is x=272. Seven columns in, in both appearances.
//
// What the field spends is the origin, and the face adds its first glyph's
// left side bearing to it, so the first covered pixel stands one column
// further in than [control.TextLeadDp] — the faces are pinned by
// DeterministicShaper, so that column is the same on every machine. The
// platform's capture carries a bearing of its own, which is why the inset it
// is read from is the measured seven less that bearing: the selection
// standing behind "Untitled" fills from x=271 and the U's first covered
// pixel is at x=272, one column of bearing in the face the platform sets
// the field in.
func TestTextFieldLeadingInsetIsMeasured(t *testing.T) {
	shaper := defaultShaper(t)
	size := image.Pt(300, 40)
	// The field wears its hairline at rest, so its inner edge is one column
	// in from its outer one; the platform's first covered pixel stands seven
	// columns in from that edge.
	const innerEdge, bearing, platform = 1, 1, 7
	for _, tc := range []struct {
		name  string
		state input.RenderState
	}{
		{"prompt", input.RenderState{}},
		{"value", input.RenderState{Text: "Email address"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := tokens.PlatformLight
			w := input.Render(
				shaper, "Email address",
				p, tokens.Spacing, tokens.Radius, tokens.DefaultTypography.BodyLarge, tokens.Comfortable,
				tc.state,
			)
			img := golden.Capture(t, size, onChrome(p.WindowBackground, w))
			// x=200 is clear of the text; rows 6 to 22 hold it and stay clear
			// of the field's corners.
			fill := img.RGBAAt(200, 14)
			first := -1
			for x := 3; x < 200 && first < 0; x++ {
				for y := 6; y <= 22; y++ {
					if img.RGBAAt(x, y) != fill {
						first = x
						break
					}
				}
			}
			if first < 0 {
				t.Fatal("no text found inside the field; this measures nothing")
			}
			if got, want := first-innerEdge, int(control.TextLeadDp)+bearing; got != want {
				t.Errorf("the text's first covered pixel is %d px in from the field's inner edge, want %d: the %d dp origin and the face's %d px side bearing",
					got, want, int(control.TextLeadDp), bearing)
			}
			if want := int(control.TextLeadDp) + bearing; want != platform {
				t.Errorf("the first covered pixel lands %d px in where the platform's lands %d: the origin is the platform's column less the bearing its own first letter carries",
					want, platform)
			}
		})
	}
}

// ---- Accessibility tests ----

// TestTextFieldHeightIsItsLineBoxOverTheFloor checks the drawn field is
// max(FieldHeight, BodyLarge's line box + 2×PaddingY).
//
// The floor is [tokens.Density.FieldHeight] and not
// [tokens.Density.ControlHeight]: the platform draws a field shorter than the
// button standing beside it, so the density carries the two separately and a
// field that took the button's floor would be the wrong height wherever the
// type role left the floor binding.
//
// The drawn field is also the pointer target, verified by
// TestTextFieldTargetIsTheField.
func TestTextFieldHeightIsItsLineBoxOverTheFloor(t *testing.T) {
	shaper := defaultShaper(t)

	var ops op.Ops
	gtx := layout.Context{
		Metric:      unit.Metric{PxPerDp: 1, PxPerSp: 1},
		Constraints: layout.Exact(image.Pt(300, 120)),
		Ops:         &ops,
	}

	dims := input.Render(
		shaper, "Email",
		tokens.PlatformLight, tokens.Spacing, tokens.Radius, tokens.DefaultTypography.BodyLarge, tokens.Comfortable,
		input.RenderState{},
	)(gtx)

	body := tokens.DefaultTypography.BodyLarge
	want := int(body.LineHeight + 2*tokens.Comfortable.PaddingY)
	if floor := int(tokens.Comfortable.FieldHeight); want < floor {
		want = floor
	}
	if dims.Size.Y != want {
		t.Errorf("text field height = %d px, want %d px (BodyLarge line box %v + 2\u00d7PaddingY %v, floored at FieldHeight %v)",
			dims.Size.Y, want, body.LineHeight, tokens.Comfortable.PaddingY, tokens.Comfortable.FieldHeight)
	}
}

// TestTextFieldTargetIsTheField checks the live field's pointer target is the
// drawn field: a press inside it, clear of the text line, focuses the editor.
func TestTextFieldTargetIsTheField(t *testing.T) {
	var tag event.Tag
	w := materialize(t, input.TextField(rx.Of(theme.Default()), input.TextFieldProps{
		Placeholder: "Email",
		FocusTag:    func(tg event.Tag) { tag = tg },
		Shaper:      defaultShaper(t),
	}))

	r := new(gioinput.Router)
	ops := new(op.Ops)
	size := image.Pt(300, 120)

	dims := driveTextFieldFrame(w, ops, r, size)
	fieldH := int(tokens.DefaultTypography.BodyLarge.LineHeight + 2*tokens.Comfortable.PaddingY)
	if floor := int(tokens.Comfortable.FieldHeight); fieldH < floor {
		fieldH = floor
	}
	if dims.Size.Y != fieldH {
		t.Fatalf("field height = %d px, want %d", dims.Size.Y, fieldH)
	}

	// Press a pixel above the field's own bottom edge, below the text line:
	// inside the field's area, outside the editor's own.
	pos := f32.Pt(150, float32(fieldH-1))
	r.Queue(
		pointer.Event{Kind: pointer.Press, Position: pos, Buttons: pointer.ButtonPrimary, Source: pointer.Mouse},
		pointer.Event{Kind: pointer.Release, Position: pos, Buttons: pointer.ButtonPrimary, Source: pointer.Mouse},
	)
	// Frame 2 — the hit area's press turns into a FocusCmd; frame 3 applies it.
	driveTextFieldFrame(w, ops, r, size)
	driveTextFieldFrame(w, ops, r, size)

	probe := layout.Context{
		Metric:      unit.Metric{PxPerDp: 1, PxPerSp: 1},
		Constraints: layout.Exact(size),
		Ops:         new(op.Ops),
		Source:      r.Source(),
	}
	if tag == nil || !probe.Focused(tag) {
		t.Error("press inside the field, below the text line, did not focus the editor")
	}
}

// TestTextFieldCompactGolden records or diffs the field at tokens.Compact
// through the live pipeline.
func TestTextFieldCompactGolden(t *testing.T) {
	w := materialize(t, input.TextField(rx.Of(densityTheme(tokens.Compact)), input.TextFieldProps{
		Placeholder: "Email address",
		// The live path would otherwise take the theme's fallback Shaper,
		// which resolves against the machine's fonts. A golden pins its faces.
		Shaper: defaultShaper(t),
	}))
	golden.Render(t, "textfield-light-compact", image.Pt(300, 60), w)
}

// TestTextFieldEditorTextRestsOnPlaceholderLine pins the field's focus
// transition: the resting placeholder is drawn through typeset.Layout with
// half its line-height deficit above the glyphs, while the live editor is a
// raw widget.Editor whose first line Gio baselines at its own ascent. Unless
// the editor is offset down by that same half-deficit, the visible text rises
// the moment the editor takes over. Goldens cannot see the transition — they
// render one state at a time — so this test measures the glyphs directly: the
// topmost pixel row of the placeholder's glyphs and of the editor's glyphs,
// for the same string at the same size, must be the same row. Both captures
// are unfocused so the border is identical and the text is the only
// difference against a blank field; the editor draws its content whenever it
// is non-empty, so seeding it exercises the exact draw path a focused field
// uses.
func TestTextFieldEditorTextRestsOnPlaceholderLine(t *testing.T) {
	const txt = "Email address"
	size := image.Pt(300, 60)

	for _, d := range []struct {
		name    string
		density tokens.Density
	}{
		{"comfortable", tokens.Comfortable},
		{"compact", tokens.Compact},
	} {
		t.Run(d.name, func(t *testing.T) {
			field := func(props input.TextFieldProps) layout.Widget {
				props.Shaper = defaultShaper(t)
				return materialize(t, input.TextField(rx.Of(densityTheme(d.density)), props))
			}

			// Blank baseline: same field, no placeholder, empty editor —
			// the edge and the fill only.
			imgBlank := golden.Capture(t, size, field(input.TextFieldProps{}))
			// The placeholder: empty, unfocused field showing txt.
			imgPh := golden.Capture(t, size, field(input.TextFieldProps{Placeholder: txt}))
			// The editor: the seed makes the editor non-empty, which hides
			// the placeholder and draws txt through widget.Editor.
			imgEd := golden.Capture(t, size, field(input.TextFieldProps{Placeholder: txt, Seed: txt}))
			if imgBlank == nil || imgPh == nil || imgEd == nil {
				return
			}

			phTop := topDiffRow(imgPh, imgBlank)
			edTop := topDiffRow(imgEd, imgBlank)
			if phTop < 0 {
				t.Fatal("placeholder rendered nothing; the measurement is broken")
			}
			if edTop < 0 {
				t.Fatal("seeded editor rendered nothing; the measurement is broken")
			}
			if phTop != edTop {
				t.Errorf("the text moves when the editor takes over: the placeholder starts at row %d, the editor at row %d", phTop, edTop)
			}
		})
	}
}

// topDiffRow returns the index of the first pixel row where a and b differ,
// or -1 when they are identical. Both images must be the same size.
func topDiffRow(a, b *image.RGBA) int {
	bounds := a.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			if a.RGBAAt(x, y) != b.RGBAAt(x, y) {
				return y
			}
		}
	}
	return -1
}

// TestTextFieldDisabledIsVisuallyDistinct confirms disabled state produces
// different pixels from enabled state.
func TestTextFieldDisabledIsVisuallyDistinct(t *testing.T) {
	shaper := defaultShaper(t)
	size := image.Pt(300, 60)

	imgEnabled := golden.Capture(t, size, input.Render(
		shaper, "Placeholder",
		tokens.PlatformLight, tokens.Spacing, tokens.Radius, tokens.DefaultTypography.BodyLarge, tokens.Comfortable,
		input.RenderState{},
	))
	imgDisabled := golden.Capture(t, size, input.Render(
		shaper, "Placeholder",
		tokens.PlatformLight, tokens.Spacing, tokens.Radius, tokens.DefaultTypography.BodyLarge, tokens.Comfortable,
		input.RenderState{Disabled: true},
	))

	if imgEnabled == nil || imgDisabled == nil {
		return
	}
	if n := golden.PixelDiff(imgEnabled, imgDisabled); n == 0 {
		t.Error("disabled and enabled fields render identically; expected visual difference")
	}
}

// TestTextFieldFocusRingIsVisuallyDistinct confirms focused state renders
// differently from normal state (the focus ring must add pixels).
func TestTextFieldFocusRingIsVisuallyDistinct(t *testing.T) {
	shaper := defaultShaper(t)
	size := image.Pt(300, 60)

	imgNormal := golden.Capture(t, size, input.Render(
		shaper, "Placeholder",
		tokens.PlatformLight, tokens.Spacing, tokens.Radius, tokens.DefaultTypography.BodyLarge, tokens.Comfortable,
		input.RenderState{},
	))
	imgFocused := golden.Capture(t, size, input.Render(
		shaper, "Placeholder",
		tokens.PlatformLight, tokens.Spacing, tokens.Radius, tokens.DefaultTypography.BodyLarge, tokens.Comfortable,
		input.RenderState{Focused: true},
	))

	if imgNormal == nil || imgFocused == nil {
		return
	}
	if n := golden.PixelDiff(imgNormal, imgFocused); n == 0 {
		t.Error("focused and normal fields render identically; expected focus ring pixels to differ")
	}
}

// ---- Behavioural tests ----

// liveTextField subscribes to the TextField observable, drains the trampoline
// scheduler with Wait(), and returns the first emitted layout.Widget. The
// editor referenced by the layout.Widget closure remains valid for the remainder of
// the test because it is captured by the rx.Defer scope.
func liveTextField(t *testing.T, props input.TextFieldProps) layout.Widget {
	t.Helper()
	if props.Shaper == nil {
		props.Shaper = defaultShaper(t)
	}
	obs := input.TextField(rx.Of(theme.Default()), props)
	var w layout.Widget
	if err := obs.Subscribe(context.Background(), func(next layout.Widget, _ error, done bool) {
		if !done && next != nil {
			w = next
		}
	}).Wait(); err != nil {
		t.Fatalf("TextField subscribe: %v", err)
	}
	if w == nil {
		t.Fatal("TextField did not emit an initial layout.Widget")
	}
	return w
}

// driveTextFieldFrame lays out the layout.Widget against a fresh op.Ops + router and
// returns the rendered dimensions. ops is reset before layout.
func driveTextFieldFrame(w layout.Widget, ops *op.Ops, r *gioinput.Router, size image.Point) layout.Dimensions {
	ops.Reset()
	gtx := layout.Context{
		Metric:      unit.Metric{PxPerDp: 1, PxPerSp: 1},
		Constraints: layout.Exact(size),
		Ops:         ops,
		Source:      r.Source(),
	}
	dims := w(gtx)
	r.Frame(ops)
	return dims
}

// TestTextFieldSubmitFiresCallbacksAndClears drives a real key.EditEvent with a
// trailing newline through a focused TextField in submit mode and verifies that
// SubmitMessage and OnSubmit fire with the editor's text and that the editor
// is cleared on the following frame.
func TestTextFieldSubmitFiresCallbacksAndClears(t *testing.T) {
	var (
		gotSubmit  string
		gotMessage string
		gotChanges []string
	)
	props := input.TextFieldProps{
		Submit:        true,
		SubmitMessage: func(s string) any { gotMessage = s; return s },
		OnSubmit:      func(_ layout.Context, s string) { gotSubmit = s },
		OnChange:      func(_ layout.Context, s string) { gotChanges = append(gotChanges, s) },
	}
	w := liveTextField(t, props)

	r := new(gioinput.Router)
	ops := new(op.Ops)
	size := image.Pt(300, 60)

	// Frame 1 — register the editor's pointer/keyboard regions.
	dims := driveTextFieldFrame(w, ops, r, size)

	// Click inside the field; the editor self-focuses on press.
	centre := f32.Pt(float32(dims.Size.X)/2, float32(dims.Size.Y)/2)
	r.Queue(
		pointer.Event{Kind: pointer.Press, Position: centre, Buttons: pointer.ButtonPrimary, Source: pointer.Mouse},
		pointer.Event{Kind: pointer.Release, Position: centre, Buttons: pointer.ButtonPrimary, Source: pointer.Mouse},
	)

	// Frame 2 — editor processes the press and requests focus.
	driveTextFieldFrame(w, ops, r, size)

	// Type "hi\n"; the trailing newline triggers a SubmitEvent because
	// editor.Submit is true.
	r.Queue(key.EditEvent{Range: key.Range{Start: 0, End: 0}, Text: "hi\n"})

	// Frame 3 — editor delivers ChangeEvent + SubmitEvent. Our handler runs
	// SubmitMessage/OnSubmit then clears the editor; the clear in turn marks
	// the buffer changed for the next Update.
	driveTextFieldFrame(w, ops, r, size)

	if gotSubmit != "hi" {
		t.Errorf("OnSubmit got %q, want %q", gotSubmit, "hi")
	}
	if gotMessage != "hi" {
		t.Errorf("SubmitMessage called with %q, want %q", gotMessage, "hi")
	}

	// Frame 4 — the SetText("") performed during submit produces a follow-up
	// ChangeEvent with the (now empty) text. Observers see the field cleared.
	driveTextFieldFrame(w, ops, r, size)

	if len(gotChanges) == 0 {
		t.Fatalf("expected at least one OnChange call, got none")
	}
	if last := gotChanges[len(gotChanges)-1]; last != "" {
		t.Errorf("editor not cleared after submit: last OnChange = %q, want %q", last, "")
	}
}

// TestTextFieldChangeEventStillFiresWithoutSubmit confirms callers without
// Submit: true see ChangeEvent-driven OnChange/Message dispatch. The exported
// MessageOp collector is unreachable from outside mvu, so OnChange (which
// sits on the same code path one branch above the Message dispatch) is used
// as the live proxy.
func TestTextFieldChangeEventStillFiresWithoutSubmit(t *testing.T) {
	var got []string
	props := input.TextFieldProps{
		Message:  "ping",
		OnChange: func(_ layout.Context, s string) { got = append(got, s) },
	}
	w := liveTextField(t, props)

	r := new(gioinput.Router)
	ops := new(op.Ops)
	size := image.Pt(300, 60)

	dims := driveTextFieldFrame(w, ops, r, size)
	centre := f32.Pt(float32(dims.Size.X)/2, float32(dims.Size.Y)/2)
	r.Queue(
		pointer.Event{Kind: pointer.Press, Position: centre, Buttons: pointer.ButtonPrimary, Source: pointer.Mouse},
		pointer.Event{Kind: pointer.Release, Position: centre, Buttons: pointer.ButtonPrimary, Source: pointer.Mouse},
	)
	driveTextFieldFrame(w, ops, r, size)

	r.Queue(key.EditEvent{Range: key.Range{Start: 0, End: 0}, Text: "x"})
	driveTextFieldFrame(w, ops, r, size)

	if len(got) == 0 {
		t.Fatalf("OnChange was not invoked; ChangeEvent path appears broken")
	}
	if got[len(got)-1] != "x" {
		t.Errorf("OnChange got %q, want %q", got[len(got)-1], "x")
	}
}

// TestTextFieldMaskKeepsValue confirms Mask obscures only the on-screen display:
// the unmasked text is still delivered through OnChange (and hence the editor's
// Text()), so a masked secret field reports the real secret to its consumer.
func TestTextFieldMaskKeepsValue(t *testing.T) {
	var got []string
	props := input.TextFieldProps{
		Mask:     '•',
		OnChange: func(_ layout.Context, s string) { got = append(got, s) },
	}
	w := liveTextField(t, props)

	r := new(gioinput.Router)
	ops := new(op.Ops)
	size := image.Pt(300, 60)

	dims := driveTextFieldFrame(w, ops, r, size)
	centre := f32.Pt(float32(dims.Size.X)/2, float32(dims.Size.Y)/2)
	r.Queue(
		pointer.Event{Kind: pointer.Press, Position: centre, Buttons: pointer.ButtonPrimary, Source: pointer.Mouse},
		pointer.Event{Kind: pointer.Release, Position: centre, Buttons: pointer.ButtonPrimary, Source: pointer.Mouse},
	)
	driveTextFieldFrame(w, ops, r, size)

	r.Queue(key.EditEvent{Range: key.Range{Start: 0, End: 0}, Text: "secret"})
	driveTextFieldFrame(w, ops, r, size)

	if len(got) == 0 {
		t.Fatal("OnChange not invoked on masked field")
	}
	if last := got[len(got)-1]; last != "secret" {
		t.Errorf("masked field delivered %q, want unmasked %q", last, "secret")
	}
}

// TestTextFieldSeedPrefillsEditor proves Seed is real editor content — an
// editable value the user can modify — not a placeholder: an untouched
// seeded field submits the seed itself.
func TestTextFieldSeedPrefillsEditor(t *testing.T) {
	var gotSubmit string
	props := input.TextFieldProps{
		Seed:     "hello",
		Submit:   true,
		OnSubmit: func(_ layout.Context, s string) { gotSubmit = s },
	}
	w := liveTextField(t, props)

	r := new(gioinput.Router)
	ops := new(op.Ops)
	size := image.Pt(300, 60)

	// Frame 1 registers the event regions; frame 2 — click focuses the editor.
	dims := driveTextFieldFrame(w, ops, r, size)
	centre := f32.Pt(float32(dims.Size.X)/2, float32(dims.Size.Y)/2)
	r.Queue(
		pointer.Event{Kind: pointer.Press, Position: centre, Buttons: pointer.ButtonPrimary, Source: pointer.Mouse},
		pointer.Event{Kind: pointer.Release, Position: centre, Buttons: pointer.ButtonPrimary, Source: pointer.Mouse},
	)
	driveTextFieldFrame(w, ops, r, size)

	// Submit without typing anything.
	r.Queue(key.EditEvent{Range: key.Range{Start: 0, End: 0}, Text: "\n"})
	driveTextFieldFrame(w, ops, r, size)

	if gotSubmit != "hello" {
		t.Errorf("OnSubmit got %q, want the seed %q (Seed must be editor content, not a placeholder)", gotSubmit, "hello")
	}
}

// TestTextFieldFocusTagExposesEditor proves FocusTag hands out the live
// editor's focus tag: focusing that tag programmatically then typing —
// without any click — reaches the editor (the mechanics patterns/modal's
// Tab cycle and initial focus rely on).
func TestTextFieldFocusTagExposesEditor(t *testing.T) {
	var tag event.Tag
	var gotChanges []string
	props := input.TextFieldProps{
		FocusTag: func(tg event.Tag) { tag = tg },
		OnChange: func(_ layout.Context, s string) { gotChanges = append(gotChanges, s) },
	}
	w := liveTextField(t, props)
	if tag == nil {
		t.Fatal("FocusTag was not called at field creation")
	}

	r := new(gioinput.Router)
	ops := new(op.Ops)
	size := image.Pt(300, 60)

	// Frame 1 registers the editor; then focus its tag programmatically.
	driveTextFieldFrame(w, ops, r, size)
	r.Source().Execute(key.FocusCmd{Tag: tag})
	driveTextFieldFrame(w, ops, r, size)

	// Type without clicking: only a focused editor receives edit events.
	r.Queue(key.EditEvent{Range: key.Range{Start: 0, End: 0}, Text: "ok"})
	driveTextFieldFrame(w, ops, r, size)

	if len(gotChanges) == 0 || gotChanges[len(gotChanges)-1] != "ok" {
		t.Fatalf("OnChange after focusing via FocusTag = %v, want [... ok]", gotChanges)
	}
}

// TestTheIBeamStandsOverTheFieldAndNoFurther reads the shape the frame
// declares over a field and over a list standing under it in the same column.
// The I-beam belongs to the text that can be edited, so it reaches the field's
// own edge and the list beside it keeps the arrow. It is the reading behind
// the report that a rail's list wore the shape of the find field above it.
func TestTheIBeamStandsOverTheFieldAndNoFurther(t *testing.T) {
	field := materialize(t, input.TextField(rx.Of(theme.Default()), input.TextFieldProps{
		Placeholder: "Search",
		Shaper:      defaultShaper(t),
	}))
	var rows int
	size := image.Pt(300, 200)
	r := new(gioinput.Router)
	ops := new(op.Ops)
	var fieldH int
	frame := func() {
		ops.Reset()
		gtx := layout.Context{
			Metric:      unit.Metric{PxPerDp: 1, PxPerSp: 1},
			Constraints: layout.Exact(size),
			Ops:         ops,
			Source:      r.Source(),
		}
		// One column: the field at its own height, the list filling the rest.
		fgtx := gtx
		fgtx.Constraints = layout.Constraints{Max: size}
		fieldH = field(fgtx).Size.Y
		st := op.Offset(image.Pt(0, fieldH)).Push(ops)
		cl := clip.Rect(image.Rect(0, 0, size.X, size.Y-fieldH)).Push(ops)
		event.Op(ops, &rows)
		cl.Pop()
		st.Pop()
		r.Frame(ops)
	}
	frame()
	frame()
	at := func(y int) pointer.Cursor {
		r.Queue(pointer.Event{Kind: pointer.Move, Position: f32.Pt(150, float32(y)), Source: pointer.Mouse})
		frame()
		return r.Cursor()
	}
	if got, want := at(fieldH/2), pointer.CursorText; got != want {
		t.Errorf("pointer over the field = %v; want %v", got, want)
	}
	if got, want := at(fieldH+40), pointer.CursorDefault; got != want {
		t.Errorf("pointer over the list under the field = %v; want %v", got, want)
	}
	if got, want := at(fieldH/2), pointer.CursorText; got != want {
		t.Errorf("pointer back over the field = %v; want %v", got, want)
	}
}

// TestTheValuesPlateauIsTheLabel reads the colour of a typed value off a
// capture of the field in both schemes: the pixel furthest from the field's
// own fill is the one the value was stroked in, because anti-aliasing only
// moves a glyph's pixels toward the surface and never past the colour it was
// drawn with.
//
// The value is the platform's label flattened onto that fill and not the
// opaque text colour. MEASURED, voicememos-multi-folder-search-2026-09-18.png
// at 1x: a typed query standing unselected in the toolbar recess plateaus at
// #232323 over that recess's #e8e8e8 fill, which is the label's black at
// 216/255 over it to the byte, where the opaque text colour would read
// #000000.
func TestTheValuesPlateauIsTheLabel(t *testing.T) {
	shaper := defaultShaper(t)
	size := image.Pt(300, 60)
	for _, tc := range []struct {
		name     string
		platform tokens.PlatformColors
	}{
		{"light", tokens.PlatformLight},
		{"dark", tokens.PlatformDark},
	} {
		t.Run(tc.name, func(t *testing.T) {
			state := input.RenderState{Text: "Untitled"}
			img := golden.Capture(t, size, onSheet(
				control.FieldFill(tc.platform, stdcolor.NRGBA{}),
				input.Render(
					shaper, "Email address",
					tc.platform, tokens.Spacing, tokens.RadiusScale{},
					tokens.DefaultTypography.BodyLarge, tokens.Comfortable, state,
				)))
			if img == nil {
				return
			}
			fill := control.FieldFill(tc.platform, stdcolor.NRGBA{})
			label := vgcolor.Flatten(tc.platform.Label, fill)

			best, bestD := stdcolor.RGBA{}, -1
			for y := 0; y < size.Y; y++ {
				for x := 0; x < size.X; x++ {
					if c := img.RGBAAt(x, y); dist(c, fill) > bestD {
						bestD, best = dist(c, fill), c
					}
				}
			}
			if bestD <= 0 {
				t.Fatal("the field drew no value on its fill")
			}
			if !nearerTo(best, label, tc.platform.Text) {
				t.Errorf("the value's plateau is %v, nearer the opaque text %v than the platform's label over the fill, %v",
					best, tc.platform.Text, label)
			}
			if best.R != label.R || best.G != label.G || best.B != label.B {
				t.Errorf("the value's plateau is %v, want the platform's label over the field's fill, %v", best, label)
			}
		})
	}
}

// selectedRunValue is one word standing twice in a field, the first selected
// and the second not: the same glyphs at the same size, so the two plateaus
// the test reads differ by the colour the field drew them in and nothing else.
const selectedRunValue = "Untitled Untitled"

// selectionFillBox is the box the selection's fill covers in img — the
// columns and rows the field painted SelectedTextBackground across.
func selectionFillBox(img *image.RGBA, size image.Point, fill stdcolor.NRGBA) (image.Rectangle, bool) {
	box, found := image.Rectangle{}, false
	for y := 0; y < size.Y; y++ {
		for x := 0; x < size.X; x++ {
			if dist(img.RGBAAt(x, y), fill) != 0 {
				continue
			}
			if !found {
				box, found = image.Rect(x, y, x+1, y+1), true
				continue
			}
			box = box.Union(image.Rect(x, y, x+1, y+1))
		}
	}
	return box, found
}

// plateau is the pixel in r standing furthest from the colour it was drawn
// over: the run's fully covered pixel, since anti-aliasing only moves a
// glyph's pixels toward what it stands on and never past the colour it was
// drawn with.
func plateau(img *image.RGBA, r image.Rectangle, over stdcolor.NRGBA) (stdcolor.RGBA, int) {
	best, bestD := stdcolor.RGBA{}, -1
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			if c := img.RGBAAt(x, y); dist(c, over) > bestD {
				bestD, best = dist(c, over), c
			}
		}
	}
	return best, bestD
}

// TestTheSelectedRunsPlateauIsTheSelectedText reads the two colours a field's
// value wears when part of it is selected: the selected run over the
// selection's fill, and the run beside it on the same rows, over the field's
// own fill.
//
// A selected run wears the platform's selected text colour over its selected
// text background whatever colour the field's other text wears. MEASURED,
// save-dialog-{light,dark}.png at 1x: the focused "Save As:" field's selected
// "Untitled" plateaus at #000000 light and #ffffff dark, opaque — sixteen
// pixels exactly light and forty-eight exactly dark — standing on the
// selection's own fill, where the label at 216/255 over those two fills would
// land 28 and 29 of 255 off on the first channel. The unselected run is the
// label over the field's fill, which is what the same field's value reads
// unselected (voicememos-multi-folder-search-2026-09-18.png, CG7.4).
func TestTheSelectedRunsPlateauIsTheSelectedText(t *testing.T) {
	shaper := defaultShaper(t)
	size := image.Pt(300, 60)
	for _, tc := range []struct {
		name     string
		platform tokens.PlatformColors
	}{
		{"light", tokens.PlatformLight},
		{"dark", tokens.PlatformDark},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fill := control.FieldFill(tc.platform, stdcolor.NRGBA{})
			img := golden.Capture(t, size, onSheet(fill, input.Render(
				shaper, "Email address",
				tc.platform, tokens.Spacing, tokens.RadiusScale{},
				tokens.DefaultTypography.BodyLarge, tokens.Comfortable,
				input.RenderState{Text: selectedRunValue, Selection: [2]int{0, 8}},
			)))
			if img == nil {
				return
			}
			box, ok := selectionFillBox(img, size, tc.platform.SelectedTextBackground)
			if !ok {
				t.Fatal("the field painted no selection fill; this measures nothing")
			}
			selected, d := plateau(img, box, tc.platform.SelectedTextBackground)
			if d <= 0 {
				t.Fatal("the selection's fill carries no run")
			}
			if want := tc.platform.SelectedText; selected.R != want.R || selected.G != want.G || selected.B != want.B {
				t.Errorf("the selected run's plateau is %v, want the platform's selected text %v; the label over the selection's fill would read %v",
					selected, want, vgcolor.Flatten(tc.platform.Label, tc.platform.SelectedTextBackground))
			}
			// The rest of the line, on the selection's own rows so the
			// field's edge and any band on it are out of the reading.
			beside := image.Rect(box.Max.X, box.Min.Y, size.X, box.Max.Y)
			unselected, d := plateau(img, beside, fill)
			if d <= 0 {
				t.Fatal("no unselected run stands beside the selected one")
			}
			if want := vgcolor.Flatten(tc.platform.Label, fill); unselected.R != want.R || unselected.G != want.G || unselected.B != want.B {
				t.Errorf("the unselected run's plateau is %v, want the platform's label over the field's fill %v", unselected, want)
			}
		})
	}
}

// TestTheLiveFieldsSelectedRunWearsTheSelectedText reads the same two colours
// off the live field, whose run, caret and selection gioui.org/widget's editor
// paints from one paint material: the field draws the selected run again over
// them in the platform's selected text colour, and the caret keeps the colour
// the first pass gave it.
//
// It is read in the light appearance alone, which is the one a live field
// laid out against theme.Default() draws in; the static path above reads both.
func TestTheLiveFieldsSelectedRunWearsTheSelectedText(t *testing.T) {
	var tag event.Tag
	w := liveTextField(t, input.TextFieldProps{
		Placeholder: "Email address",
		Seed:        selectedRunValue,
		FocusTag:    func(tg event.Tag) { tag = tg },
	})
	editor, ok := tag.(*widget.Editor)
	if !ok {
		t.Fatalf("FocusTag handed %T, want the field's editor", tag)
	}

	r := new(gioinput.Router)
	ops := new(op.Ops)
	size := image.Pt(300, 60)

	// Frame 1 registers the editor, then its tag takes focus: an editor paints
	// no selection at all while another holds it.
	driveTextFieldFrame(w, ops, r, size)
	r.Source().Execute(key.FocusCmd{Tag: tag})
	driveTextFieldFrame(w, ops, r, size)
	editor.SetCaret(0, 8)

	p := tokens.PlatformLight
	fill := control.FieldFill(p, p.WindowBackground)
	// The capture frame carries the router's own source, so the editor is
	// still the focused tag while it draws.
	img := golden.Capture(t, size, onSheet(p.WindowBackground, func(gtx layout.Context) layout.Dimensions {
		gtx.Source = r.Source()
		return w(gtx)
	}))
	if img == nil {
		return
	}
	box, ok := selectionFillBox(img, size, p.SelectedTextBackground)
	if !ok {
		t.Fatal("the live field painted no selection fill; this measures nothing")
	}
	selected, d := plateau(img, box, p.SelectedTextBackground)
	if d <= 0 {
		t.Fatal("the selection's fill carries no run")
	}
	if want := p.SelectedText; selected.R != want.R || selected.G != want.G || selected.B != want.B {
		t.Errorf("the live field's selected run plateaus at %v, want the platform's selected text %v", selected, want)
	}
	unselected, d := plateau(img, image.Rect(box.Max.X, box.Min.Y, size.X, box.Max.Y), fill)
	if d <= 0 {
		t.Fatal("no unselected run stands beside the selected one")
	}
	if want := vgcolor.Flatten(p.Label, fill); unselected.R != want.R || unselected.G != want.G || unselected.B != want.B {
		t.Errorf("the live field's unselected run plateaus at %v, want the platform's label over the field's fill %v", unselected, want)
	}
	// The caret stands at the selected run's leading end and keeps the colour
	// the field's other text wears. It is painted over the selection's own
	// leading column — which is why the fill's box begins one column in — so
	// it is read across that column and the two the caret is drawn at.
	// Nothing else there can read it: the run's anti-aliased pixels carry the
	// selection fill's channel order, ascending red to blue, and the label's
	// grey does not.
	caret := vgcolor.Flatten(p.Label, fill)
	band := image.Rect(box.Min.X-2, box.Min.Y, box.Max.X, box.Max.Y)
	found := false
	for y := band.Min.Y; y < band.Max.Y && !found; y++ {
		for x := band.Min.X; x < band.Max.X; x++ {
			if c := img.RGBAAt(x, y); c.R == caret.R && c.G == caret.G && c.B == caret.B {
				found = true
				break
			}
		}
	}
	if !found {
		t.Errorf("the caret standing at the selected run does not read %v; the second pass painted it too", caret)
	}
}

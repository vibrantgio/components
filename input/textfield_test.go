package input_test

import (
	"context"
	"image"
	"testing"

	"gioui.org/f32"
	"gioui.org/io/event"
	gioinput "gioui.org/io/input"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/text"
	"gioui.org/unit"

	"github.com/reactivego/rx"
	golden "github.com/vibrantgio/prism/golden"
	"github.com/vibrantgio/prism/input"
	"github.com/vibrantgio/theme/theme"
	"github.com/vibrantgio/theme/tokens"
)

// defaultShaper returns the shaper every golden here draws with: the default
// typography's faces pinned, system fonts off, so the stored images are the
// same on every machine. A golden test pins its faces with
// DeterministicShaper; application code takes the fallback Shaper. See
// AGENTS.md.
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
	// vary between GPU context initialisations. The placeholder is real text
	// (F4.4) — the faces are pinned by DeterministicShaper, so Latin glyphs
	// rasterise identically on every machine and the placeholder's own type
	// role is now on screen where a regression in it would show.
	sharpRadius := tokens.RadiusScale{}
	// Disabled is intentionally omitted: semi-transparent disabled colours
	// composite non-deterministically against the headless window background.
	// The disabled visual is tested separately in TestTextFieldDisabledIsVisuallyDistinct.
	cases := []struct {
		name   string
		colors tokens.ColorTokens
		state  input.RenderState
	}{
		// Prefixed per component: see the note in checkbox_test.go. These four
		// share testdata/golden with the checkbox, radio and dropdown cases.
		{"textfield-light-normal", tokens.DefaultLight, input.RenderState{}},
		{"textfield-dark-normal", tokens.DefaultDark, input.RenderState{}},
		{"textfield-light-focused", tokens.DefaultLight, input.RenderState{Focused: true}},
		{"textfield-light-focused-with-text", tokens.DefaultLight, input.RenderState{Focused: true, Text: "hello@example.com"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := input.Render(
				shaper, "Email address",
				tc.colors, tokens.Spacing, sharpRadius, tokens.DefaultTypography.BodyLarge, tokens.Comfortable,
				tc.state,
			)
			golden.Render(t, tc.name, size, w)
		})
	}
}

// ---- Accessibility tests ----

// TestTextFieldHeightIsItsLineBoxOverTheFloor checks the drawn field is
// max(ControlHeight, BodyLarge's line box + 2×PaddingY), which is what
// [tokens.Density.ControlHeight] being a floor means at the one control where
// the two readings differ. BodyLarge is a 24 dp line box, so a Comfortable
// field is 24 + 16 = 40 dp, four taller than the 36 dp floor and four taller
// than a Comfortable button in the 20 dp LabelLarge role. That is the type
// roles talking, not a defect: F4.4c made the line height mean the line box,
// and a field set in a larger role is a larger field.
//
// The 44 dp WCAG 2.5.5 floor applies to the pointer target, verified by
// TestTextFieldHitSlopFocusesEditor.
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
		tokens.DefaultLight, tokens.Spacing, tokens.Radius, tokens.DefaultTypography.BodyLarge, tokens.Comfortable,
		input.RenderState{},
	)(gtx)

	body := tokens.DefaultTypography.BodyLarge
	want := int(body.LineHeight + 2*tokens.Comfortable.PaddingY)
	if floor := int(tokens.Comfortable.ControlHeight); want < floor {
		want = floor
	}
	if dims.Size.Y != want {
		t.Errorf("text field height = %d px, want %d px (BodyLarge line box %v + 2\u00d7PaddingY %v, floored at ControlHeight %v)",
			dims.Size.Y, want, body.LineHeight, tokens.Comfortable.PaddingY, tokens.Comfortable.ControlHeight)
	}
}

// TestTextFieldHitSlopFocusesEditor checks the live field's pointer target
// extends to the 44 dp floor: a press below the 36 dp visual field, inside
// the hit slop, focuses the editor.
func TestTextFieldHitSlopFocusesEditor(t *testing.T) {
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
	if dims.Size.Y != fieldH {
		t.Fatalf("field height = %d px, want %d", dims.Size.Y, fieldH)
	}

	// The hit rect is the 44 px floor centred on the 40 px field: -2..42.
	// Press at y=41 — outside the field, inside the slop.
	pos := f32.Pt(150, 41)
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
		t.Error("press in the hit slop below the field did not focus the editor")
	}
}

// TestTextFieldCompactGolden records or diffs the field at tokens.Compact
// through the live pipeline: a 28 dp bar (shadcn h-8 territory).
func TestTextFieldCompactGolden(t *testing.T) {
	w := materialize(t, input.TextField(rx.Of(densityTheme(tokens.Compact)), input.TextFieldProps{
		Placeholder: "Email address",
		// The live path would otherwise take the theme's fallback Shaper,
		// which resolves against the machine's fonts. A golden pins its faces.
		Shaper: defaultShaper(t),
	}))
	golden.Render(t, "textfield-light-compact", image.Pt(300, 60), w)
}

// TestTextFieldDisabledIsVisuallyDistinct confirms disabled state produces
// different pixels from enabled state.
func TestTextFieldDisabledIsVisuallyDistinct(t *testing.T) {
	shaper := defaultShaper(t)
	size := image.Pt(300, 60)

	imgEnabled := golden.Capture(t, size, input.Render(
		shaper, "Placeholder",
		tokens.DefaultLight, tokens.Spacing, tokens.Radius, tokens.DefaultTypography.BodyLarge, tokens.Comfortable,
		input.RenderState{},
	))
	imgDisabled := golden.Capture(t, size, input.Render(
		shaper, "Placeholder",
		tokens.DefaultLight, tokens.Spacing, tokens.Radius, tokens.DefaultTypography.BodyLarge, tokens.Comfortable,
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
		tokens.DefaultLight, tokens.Spacing, tokens.Radius, tokens.DefaultTypography.BodyLarge, tokens.Comfortable,
		input.RenderState{},
	))
	imgFocused := golden.Capture(t, size, input.Render(
		shaper, "Placeholder",
		tokens.DefaultLight, tokens.Spacing, tokens.Radius, tokens.DefaultTypography.BodyLarge, tokens.Comfortable,
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
// editor referenced by the widget closure remains valid for the remainder of
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
		t.Fatal("TextField did not emit an initial widget")
	}
	return w
}

// driveTextFieldFrame lays out the widget against a fresh op.Ops + router and
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
// is cleared on the following frame (Measurable a + b).
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
// Submit: true continue to see ChangeEvent-driven OnChange/Message dispatch
// (Measurable c — no behavioural regression). The exported MessageOp collector
// is unreachable from outside mvu, so OnChange (which sits on the same code
// path one branch above the Message dispatch) is used as the live proxy.
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

	// Frame 1 — register regions; frame 2 — click focuses the editor.
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
// without any click — reaches the editor (the mechanics cadence/modal's
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

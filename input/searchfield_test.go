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
	golden "github.com/vibrantgio/components/golden"
	"github.com/vibrantgio/components/icon"
	"github.com/vibrantgio/components/input"
	"github.com/vibrantgio/theme/theme"
	"github.com/vibrantgio/theme/tokens"
)

// TestSearchFieldGolden records or diffs the search field's three states in
// both schemes: resting with its prompt, holding a query with the clear mark
// beside it, and focused.
//
// The three are what separate this control from the text field it is built
// on — the looking glass is in every one of them, and the clear mark is in
// exactly the one that has something to clear.
func TestSearchFieldGolden(t *testing.T) {
	shaper := defaultShaper(t)
	size := image.Pt(300, 60)

	// Zero corner radius keeps the edges sharp, for the reason
	// TestTextFieldGolden gives.
	sharpRadius := tokens.RadiusScale{}
	cases := []struct {
		name   string
		colors tokens.ColorTokens
		state  input.RenderState
	}{
		{"searchfield-light-normal", tokens.DefaultLight, input.RenderState{}},
		{"searchfield-dark-normal", tokens.DefaultDark, input.RenderState{}},
		{"searchfield-light-typed", tokens.DefaultLight, input.RenderState{Text: "meeting notes"}},
		{"searchfield-dark-typed", tokens.DefaultDark, input.RenderState{Text: "meeting notes"}},
		{"searchfield-light-focused", tokens.DefaultLight, input.RenderState{Focused: true}},
		{"searchfield-dark-focused", tokens.DefaultDark, input.RenderState{Focused: true}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := input.RenderSearch(
				shaper, "Search",
				tc.colors, tokens.Spacing, sharpRadius, tokens.DefaultTypography.BodyLarge, tokens.Comfortable,
				tc.state,
			)
			golden.Render(t, tc.name, size, w)
		})
	}

	// And the compact density, where both marks shrink with the control they
	// stand in: the slot is the density's icon size, so a field that lost
	// height has not kept full-size marks inside it.
	t.Run("searchfield-light-compact", func(t *testing.T) {
		w := input.RenderSearch(
			shaper, "Search",
			tokens.DefaultLight, tokens.Spacing, sharpRadius, tokens.DefaultTypography.BodyLarge, tokens.Compact,
			input.RenderState{Text: "meeting notes"},
		)
		golden.Render(t, "searchfield-light-compact", size, w)
	})
}

// TestSearchFieldReservesBothSlots asserts the structure is paid for in
// layout and not merely painted: a search field spends the same width on its
// marks whether or not the clear mark is in its slot, so the text a reader is
// typing does not reflow the moment the field stops being empty.
//
// It is read off the drawn width the placeholder is given, which is the field
// width less both slots — the same number the editor is laid out in.
func TestSearchFieldReservesBothSlots(t *testing.T) {
	shaper := defaultShaper(t)
	const fieldW = 300

	measure := func(w layout.Widget) image.Point {
		var ops op.Ops
		gtx := layout.Context{
			Ops:         &ops,
			Metric:      unit.Metric{PxPerDp: 1, PxPerSp: 1},
			Constraints: layout.Exact(image.Pt(fieldW, 60)),
		}
		return w(gtx).Size
	}

	empty := measure(input.RenderSearch(shaper, "Search", tokens.DefaultLight, tokens.Spacing,
		tokens.Radius, tokens.DefaultTypography.BodyLarge, tokens.Comfortable, input.RenderState{}))
	typed := measure(input.RenderSearch(shaper, "Search", tokens.DefaultLight, tokens.Spacing,
		tokens.Radius, tokens.DefaultTypography.BodyLarge, tokens.Comfortable, input.RenderState{Text: "q"}))
	plain := measure(input.Render(shaper, "Search", tokens.DefaultLight, tokens.Spacing,
		tokens.Radius, tokens.DefaultTypography.BodyLarge, tokens.Comfortable, input.RenderState{}))

	if empty != typed {
		t.Errorf("an empty search field measured %v and one holding a query measured %v; the slots are reserved, so the two are one size", empty, typed)
	}
	if empty != plain {
		t.Errorf("a search field measured %v and a text field %v; the marks are spent out of the field's own width, not added to it", empty, plain)
	}
}

// TestSearchFieldClearMarkEmptiesTheField drives a press on the clear mark
// through a live search field holding a query, and checks the two things the
// mark promises: the field is empty afterwards, and the empty query is
// reported the way a keystroke reports any other — which is what dismisses a
// consumer's highlight along with the search that caused it.
func TestSearchFieldClearMarkEmptiesTheField(t *testing.T) {
	var changes []string
	type clearMsg struct{}
	w := materialize(t, input.SearchField(rx.Of(theme.Default()), input.SearchFieldProps{
		Placeholder:  "Search",
		Seed:         "meeting notes",
		Shaper:       defaultShaper(t),
		ClearMessage: clearMsg{},
		OnChange:     func(_ layout.Context, s string) { changes = append(changes, s) },
	}))

	r := new(gioinput.Router)
	ops := new(op.Ops)
	size := image.Pt(300, 120)

	dims := driveTextFieldFrame(w, ops, r, size)

	// The mark stands in the trailing slot, one horizontal pad in from the
	// field's own edge and centred on its height.
	pos := f32.Pt(float32(dims.Size.X)-tokens.Spacing.S3-float32(icon.Size(tokens.Comfortable))/2,
		float32(dims.Size.Y)/2)
	r.Queue(
		pointer.Event{Kind: pointer.Press, Position: pos, Buttons: pointer.ButtonPrimary, Source: pointer.Mouse},
		pointer.Event{Kind: pointer.Release, Position: pos, Buttons: pointer.ButtonPrimary, Source: pointer.Mouse},
	)
	driveTextFieldFrame(w, ops, r, size)
	driveTextFieldFrame(w, ops, r, size)

	if len(changes) == 0 {
		t.Fatal("pressing the clear mark reported no change at all")
	}
	if got := changes[len(changes)-1]; got != "" {
		t.Errorf("the clear mark reported %q; it empties the field, so it reports the empty query", got)
	}

	// And the field is empty afterwards, read off the mark itself: a second
	// press has nothing to take back and reports nothing.
	n := len(changes)
	r.Queue(
		pointer.Event{Kind: pointer.Press, Position: pos, Buttons: pointer.ButtonPrimary, Source: pointer.Mouse},
		pointer.Event{Kind: pointer.Release, Position: pos, Buttons: pointer.ButtonPrimary, Source: pointer.Mouse},
	)
	driveTextFieldFrame(w, ops, r, size)
	driveTextFieldFrame(w, ops, r, size)
	if len(changes) != n {
		t.Errorf("a press where the mark stood on an emptied field reported %v; the field was already empty and the mark is not drawn there any more", changes[n:])
	}
}

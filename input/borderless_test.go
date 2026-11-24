package input_test

import (
	"context"
	"image"
	stdcolor "image/color"
	"testing"

	"gioui.org/io/event"
	gioinput "gioui.org/io/input"
	"gioui.org/io/key"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/widget"

	"github.com/reactivego/rx"
	golden "github.com/vibrantgio/components/golden"
	"github.com/vibrantgio/components/input"
	vgcolor "github.com/vibrantgio/theme/color"
	"github.com/vibrantgio/theme/theme"
	"github.com/vibrantgio/theme/tokens"
	"github.com/vibrantgio/theme/typeset"
)

const (
	borderlessShort = "Write the release notes"
	// borderlessLong wraps at the width the tests give the field.
	borderlessLong = "Write the release notes for the next version and send them to the list"
)

var borderlessSize = image.Pt(240, 80)

// loose relaxes the floor the capture hands down, as a card hands its
// title the width alone.
func loose(w layout.Widget) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Min = image.Point{}
		return w(gtx)
	}
}

// liveBorderless subscribes to a borderless field on the light default
// theme and returns the last layout.Widget it emitted.
func liveBorderless(t *testing.T, props input.BorderlessTextFieldProps) layout.Widget {
	t.Helper()
	if props.Shaper == nil {
		props.Shaper = defaultShaper(t)
	}
	var w layout.Widget
	if err := input.BorderlessTextField(rx.Of(theme.Default()), props).Subscribe(context.Background(), func(next layout.Widget, _ error, done bool) {
		if !done && next != nil {
			w = next
		}
	}).Wait(); err != nil {
		t.Fatalf("subscribe: %v", err)
	}
	if w == nil {
		t.Fatal("BorderlessTextField did not emit a layout.Widget")
	}
	return w
}

// borderlessSurface is the fill the tests stand the field on: the content
// fill, a fill that is not the window's own plane.
func borderlessSurface(p tokens.PlatformColors) stdcolor.NRGBA { return p.ControlBackground }

// TestBorderlessTextFieldGolden records or diffs the borderless field's
// static states in both schemes, on the content fill.
func TestBorderlessTextFieldGolden(t *testing.T) {
	shaper := defaultShaper(t)
	cases := []struct {
		name     string
		platform tokens.PlatformColors
		state    input.BorderlessState
	}{
		{"borderless-light-placeholder", tokens.PlatformLight, input.BorderlessState{Focused: true}},
		{"borderless-dark-placeholder", tokens.PlatformDark, input.BorderlessState{Focused: true}},
		{"borderless-light-text", tokens.PlatformLight, input.BorderlessState{Text: borderlessShort}},
		{"borderless-dark-text", tokens.PlatformDark, input.BorderlessState{Text: borderlessShort}},
		{"borderless-light-wrapped", tokens.PlatformLight, input.BorderlessState{Text: borderlessLong}},
		{"borderless-light-selected", tokens.PlatformLight, input.BorderlessState{Focused: true, Text: borderlessShort, Selection: [2]int{0, 5}}},
		{"borderless-dark-selected", tokens.PlatformDark, input.BorderlessState{Focused: true, Text: borderlessShort, Selection: [2]int{0, 5}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := tc.state
			s.Surface = borderlessSurface(tc.platform)
			w := input.RenderBorderless(shaper, "Card title", tc.platform, tokens.DefaultTypography.Body, s)
			golden.Render(t, tc.name, borderlessSize, onSheet(s.Surface, loose(w)))
		})
	}
}

// labelOn draws txt as a label of the Body role in the platform's label
// colour on fill: the text the borderless field edits, once the edit ends.
func labelOn(t *testing.T, p tokens.PlatformColors, fill stdcolor.NRGBA, txt string) layout.Widget {
	shaper := defaultShaper(t)
	return func(gtx layout.Context) layout.Dimensions {
		return typeset.Text(gtx, shaper, txt, tokens.DefaultTypography.Body, vgcolor.Flatten(p.Label, fill), 0)
	}
}

// diffBox is the smallest rectangle that holds every pixel where a and b
// differ, and the number of those pixels.
func diffBox(a, b *image.RGBA) (image.Rectangle, int) {
	var box image.Rectangle
	n := 0
	for y := a.Bounds().Min.Y; y < a.Bounds().Max.Y; y++ {
		for x := a.Bounds().Min.X; x < a.Bounds().Max.X; x++ {
			if a.RGBAAt(x, y) != b.RGBAAt(x, y) {
				box = box.Union(image.Rect(x, y, x+1, y+1))
				n++
			}
		}
	}
	return box, n
}

// drawnBox is the smallest rectangle that holds every pixel of img that is not
// fill.
func drawnBox(img *image.RGBA, fill stdcolor.NRGBA) image.Rectangle {
	var box image.Rectangle
	want := stdcolor.RGBA{R: fill.R, G: fill.G, B: fill.B, A: 0xff}
	for y := img.Bounds().Min.Y; y < img.Bounds().Max.Y; y++ {
		for x := img.Bounds().Min.X; x < img.Bounds().Max.X; x++ {
			if img.RGBAAt(x, y) != want {
				box = box.Union(image.Rect(x, y, x+1, y+1))
			}
		}
	}
	return box
}

// TestBorderlessTextStandsWhereTheLabelStands asserts that the text in the
// field and the same text drawn as a label are the same pixels: nothing
// moves when the edit ends. A title on one line and one that wraps are read
// both.
func TestBorderlessTextStandsWhereTheLabelStands(t *testing.T) {
	p := tokens.PlatformLight
	fill := borderlessSurface(p)
	for _, txt := range []string{borderlessShort, borderlessLong} {
		t.Run(txt[:10], func(t *testing.T) {
			field := liveBorderless(t, input.BorderlessTextFieldProps{
				Placeholder: "Card title",
				Seed:        txt,
				Surface:     borderlessSurface,
			})
			edited := golden.Capture(t, borderlessSize, onSheet(fill, loose(field)))
			after := golden.Capture(t, borderlessSize, onSheet(fill, loose(labelOn(t, p, fill, txt))))
			if edited == nil || after == nil {
				return
			}
			if drawnBox(after, fill).Empty() {
				t.Fatal("the label drew nothing; this measures nothing")
			}
			if box, n := diffBox(edited, after); n > 0 {
				t.Errorf("the field's text and the label differ in %d pixels inside %v", n, box)
			}
		})
	}
}

// TestBorderlessFocusAddsTheCaretAlone asserts that the field holding the
// keyboard draws its text where the label draws it and adds only the caret:
// no ring, no bezel, no fill.
func TestBorderlessFocusAddsTheCaretAlone(t *testing.T) {
	p := tokens.PlatformLight
	fill := borderlessSurface(p)
	var tag event.Tag
	field := liveBorderless(t, input.BorderlessTextFieldProps{
		Placeholder: "Card title",
		Seed:        borderlessShort,
		Surface:     borderlessSurface,
		FocusTag:    func(tg event.Tag) { tag = tg },
	})
	editor := tag.(*widget.Editor)
	r := new(gioinput.Router)
	ops := new(op.Ops)
	driveTextFieldFrame(field, ops, r, borderlessSize)
	r.Source().Execute(key.FocusCmd{Tag: tag})
	driveTextFieldFrame(field, ops, r, borderlessSize)
	n := editor.Len()
	editor.SetCaret(n, n)
	driveTextFieldFrame(field, ops, r, borderlessSize)

	edited := golden.Capture(t, borderlessSize, onSheet(fill, loose(func(gtx layout.Context) layout.Dimensions {
		gtx.Source = r.Source()
		return field(gtx)
	})))
	after := golden.Capture(t, borderlessSize, onSheet(fill, loose(labelOn(t, p, fill, borderlessShort))))
	if edited == nil || after == nil {
		return
	}
	drawn := drawnBox(after, fill)
	box, diff := diffBox(edited, after)
	if diff == 0 {
		t.Fatal("the focused field drew no caret; this measures nothing")
	}
	if box.Min.X < drawn.Max.X-1 || box.Dx() > 3 {
		t.Errorf("focus changed pixels inside %v; only the caret after the text, which ends at x %d, may change", box, drawn.Max.X)
	}
}

// TestBorderlessPlaceholderStaysWhileFocused asserts that the placeholder
// stands in an empty field that holds the keyboard, in the platform's
// placeholder colour, and leaves when the user types.
func TestBorderlessPlaceholderStaysWhileFocused(t *testing.T) {
	p := tokens.PlatformLight
	fill := borderlessSurface(p)
	var tag event.Tag
	field := liveBorderless(t, input.BorderlessTextFieldProps{
		Placeholder: "Card title",
		Surface:     borderlessSurface,
		FocusTag:    func(tg event.Tag) { tag = tg },
	})
	r := new(gioinput.Router)
	ops := new(op.Ops)
	driveTextFieldFrame(field, ops, r, borderlessSize)
	r.Source().Execute(key.FocusCmd{Tag: tag})
	driveTextFieldFrame(field, ops, r, borderlessSize)
	capture := func() *image.RGBA {
		return golden.Capture(t, borderlessSize, onSheet(fill, loose(func(gtx layout.Context) layout.Dimensions {
			gtx.Source = r.Source()
			return field(gtx)
		})))
	}
	focused := capture()
	static := golden.Capture(t, borderlessSize, onSheet(fill, loose(input.RenderBorderless(defaultShaper(t), "Card title", p, tokens.DefaultTypography.Body, input.BorderlessState{Surface: fill}))))
	if focused == nil || static == nil {
		return
	}
	ph := drawnBox(static, fill)
	if ph.Empty() {
		t.Fatal("the static path drew no placeholder; this measures nothing")
	}
	// The caret stands at the field's leading edge, over the placeholder's
	// first column; every pixel past it is the placeholder's.
	if box, n := diffBox(focused, static); n > 0 && box.Min.X > ph.Min.X+2 {
		t.Errorf("the focused empty field differs from the placeholder inside %v", box)
	}

	r.Queue(key.EditEvent{Range: key.Range{Start: 0, End: 0}, Text: "W"})
	driveTextFieldFrame(field, ops, r, borderlessSize)
	typed := capture()
	if typed == nil {
		return
	}
	if drawn := drawnBox(typed, fill); drawn.Max.X >= ph.Max.X {
		t.Errorf("the placeholder stays after a keystroke: drawn reaches x %d, the placeholder's end is x %d", drawn.Max.X, ph.Max.X)
	}
}

// TestBorderlessSubmitKeepsTheText asserts that Return reports the text and
// leaves it in the field, so the text stands where it is until the caller
// ends the edit.
func TestBorderlessSubmitKeepsTheText(t *testing.T) {
	var submitted string
	var tag event.Tag
	field := liveBorderless(t, input.BorderlessTextFieldProps{
		Submit:   true,
		OnSubmit: func(_ layout.Context, s string) { submitted = s },
		FocusTag: func(tg event.Tag) { tag = tg },
	})
	editor := tag.(*widget.Editor)
	r := new(gioinput.Router)
	ops := new(op.Ops)
	driveTextFieldFrame(field, ops, r, borderlessSize)
	r.Source().Execute(key.FocusCmd{Tag: tag})
	driveTextFieldFrame(field, ops, r, borderlessSize)
	r.Queue(key.EditEvent{Range: key.Range{Start: 0, End: 0}, Text: "hi\n"})
	driveTextFieldFrame(field, ops, r, borderlessSize)
	driveTextFieldFrame(field, ops, r, borderlessSize)
	if submitted != "hi" {
		t.Errorf("OnSubmit got %q, want %q", submitted, "hi")
	}
	if got := editor.Text(); got != "hi" {
		t.Errorf("the field holds %q after Return, want %q", got, "hi")
	}
}

// TestBorderlessGrowsWithItsText asserts that the field is as tall as its
// lines, as tall as the label of the same text: one line for a short title,
// more for one that wraps.
func TestBorderlessGrowsWithItsText(t *testing.T) {
	r := new(gioinput.Router)
	ops := new(op.Ops)
	p := tokens.PlatformLight
	fill := borderlessSurface(p)
	for _, txt := range []string{"", borderlessShort, borderlessLong} {
		field := liveBorderless(t, input.BorderlessTextFieldProps{Placeholder: "Card title", Seed: txt})
		got := driveTextFieldFrame(loose(field), ops, r, borderlessSize).Size.Y
		shown := txt
		if shown == "" {
			shown = "Card title"
		}
		want := driveTextFieldFrame(loose(labelOn(t, p, fill, shown)), ops, r, borderlessSize).Size.Y
		if got != want {
			t.Errorf("the field holding %q is %d px tall, want the label's %d", txt, got, want)
		}
	}
	short := driveTextFieldFrame(loose(labelOn(t, p, fill, borderlessShort)), ops, r, borderlessSize).Size.Y
	long := driveTextFieldFrame(loose(labelOn(t, p, fill, borderlessLong)), ops, r, borderlessSize).Size.Y
	if long <= short {
		t.Fatalf("the long title is %d px and the short one %d px tall; the long one does not wrap, so this measures nothing", long, short)
	}
}

// TestBorderlessEmptyFieldKeepsOneLine asserts that an empty field with no
// placeholder is one line tall: the line box of the role it edits. It holds
// for a field that starts empty, for a field whose text the user deleted,
// and for the static path.
func TestBorderlessEmptyFieldKeepsOneLine(t *testing.T) {
	r := new(gioinput.Router)
	ops := new(op.Ops)
	p := tokens.PlatformLight
	want := driveTextFieldFrame(loose(labelOn(t, p, borderlessSurface(p), "W")), ops, r, borderlessSize).Size.Y
	if want <= 0 {
		t.Fatal("a one-line label is 0 px tall; this measures nothing")
	}

	empty := liveBorderless(t, input.BorderlessTextFieldProps{})
	if got := driveTextFieldFrame(loose(empty), ops, r, borderlessSize).Size.Y; got != want {
		t.Errorf("the empty field is %d px tall, want one line's %d", got, want)
	}

	var tag event.Tag
	deleted := liveBorderless(t, input.BorderlessTextFieldProps{
		Seed:     borderlessShort,
		FocusTag: func(tg event.Tag) { tag = tg },
	})
	r = new(gioinput.Router)
	driveTextFieldFrame(loose(deleted), ops, r, borderlessSize)
	r.Source().Execute(key.FocusCmd{Tag: tag})
	driveTextFieldFrame(loose(deleted), ops, r, borderlessSize)
	n := tag.(*widget.Editor).Len()
	r.Queue(key.EditEvent{Range: key.Range{Start: 0, End: n}, Text: ""})
	driveTextFieldFrame(loose(deleted), ops, r, borderlessSize)
	if got := tag.(*widget.Editor).Len(); got != 0 {
		t.Fatalf("the field holds %d runes after the delete; this measures nothing", got)
	}
	if got := driveTextFieldFrame(loose(deleted), ops, r, borderlessSize).Size.Y; got != want {
		t.Errorf("the field is %d px tall after its text is deleted, want one line's %d", got, want)
	}

	static := input.RenderBorderless(defaultShaper(t), "", p, tokens.DefaultTypography.Body, input.BorderlessState{})
	if got := driveTextFieldFrame(loose(static), ops, r, borderlessSize).Size.Y; got != want {
		t.Errorf("the static empty field is %d px tall, want one line's %d", got, want)
	}
}

// TestBorderlessStaticSelectionIsTheLiveOne asserts that the static path
// draws a selected run where the live field draws it: the same fill over
// the same rows and columns, and the same text over it. The live field adds
// its caret at the run's end, and the static path draws no caret.
func TestBorderlessStaticSelectionIsTheLiveOne(t *testing.T) {
	p := tokens.PlatformLight
	fill := borderlessSurface(p)
	var tag event.Tag
	field := liveBorderless(t, input.BorderlessTextFieldProps{
		Seed:     borderlessShort,
		Surface:  borderlessSurface,
		FocusTag: func(tg event.Tag) { tag = tg },
	})
	r := new(gioinput.Router)
	ops := new(op.Ops)
	driveTextFieldFrame(field, ops, r, borderlessSize)
	r.Source().Execute(key.FocusCmd{Tag: tag})
	driveTextFieldFrame(field, ops, r, borderlessSize)
	tag.(*widget.Editor).SetCaret(5, 0)
	driveTextFieldFrame(field, ops, r, borderlessSize)

	live := golden.Capture(t, borderlessSize, onSheet(fill, loose(func(gtx layout.Context) layout.Dimensions {
		gtx.Source = r.Source()
		return field(gtx)
	})))
	static := golden.Capture(t, borderlessSize, onSheet(fill, loose(input.RenderBorderless(defaultShaper(t), "", p, tokens.DefaultTypography.Body,
		input.BorderlessState{Surface: fill, Focused: true, Text: borderlessShort, Selection: [2]int{0, 5}}))))
	if live == nil || static == nil {
		return
	}
	// The selection's fill in the static path: its last column and its rows,
	// read down column 0 and across row 0.
	sel := static.RGBAAt(0, 0)
	if sel == (stdcolor.RGBA{fill.R, fill.G, fill.B, 255}) {
		t.Fatal("the static path drew no selection; this measures nothing")
	}
	end := 0
	for end < borderlessSize.X && static.RGBAAt(end, 0) == sel {
		end++
	}
	box, n := diffBox(live, static)
	if n == 0 {
		t.Fatal("the live field drew no caret; this measures nothing")
	}
	if box.Min.X < end-1 || box.Dx() > 3 {
		t.Errorf("the static selection differs from the live one inside %v; only the caret at the run's end, x %d, may differ", box, end)
	}
}

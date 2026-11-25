package input_test

import (
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
	"gioui.org/widget"

	"github.com/reactivego/rx"

	"github.com/vibrantgio/components/golden"
	"github.com/vibrantgio/components/input"
	"github.com/vibrantgio/theme/theme"
	"github.com/vibrantgio/theme/tokens"
)

// insertionAccent is the colour a live field on the light default theme
// draws its insertion point in: the measured #178bfb, the platform set's
// TextInsertionPoint.
var insertionAccent = tokens.PlatformLight.TextInsertionPoint

// accentBox is the smallest rectangle holding every pixel of img that reads
// c exactly, and the number of those pixels.
func accentBox(img *image.RGBA, c stdcolor.NRGBA) (image.Rectangle, int) {
	var box image.Rectangle
	n := 0
	for y := img.Bounds().Min.Y; y < img.Bounds().Max.Y; y++ {
		for x := img.Bounds().Min.X; x < img.Bounds().Max.X; x++ {
			if p := img.RGBAAt(x, y); p.R == c.R && p.G == c.G && p.B == c.B {
				box = box.Union(image.Rect(x, y, x+1, y+1))
				n++
			}
		}
	}
	return box, n
}

// liveCapture draws w once more on the router's own source, so the field
// still holds the keyboard while it is captured.
func liveCapture(t *testing.T, r *gioinput.Router, size image.Point, sheet stdcolor.NRGBA, w layout.Widget) *image.RGBA {
	t.Helper()
	return golden.Capture(t, size, onSheet(sheet, func(gtx layout.Context) layout.Dimensions {
		gtx.Source = r.Source()
		return w(gtx)
	}))
}

// TestTheInsertionPointIsOneColumnInTheAccent asserts that a bordered and a
// borderless field draw one insertion point of one width whatever they hold:
// one column wide and as tall as the text's line, in the accent, empty, after
// the text and inside it.
func TestTheInsertionPointIsOneColumnInTheAccent(t *testing.T) {
	p := tokens.PlatformLight
	lineH := int(tokens.DefaultTypography.SystemFont.LineHeight)
	bodyH := int(tokens.DefaultTypography.Body.LineHeight)
	type field struct {
		name  string
		make  func(seed string, tag *event.Tag) layout.Widget
		size  image.Point
		sheet stdcolor.NRGBA
		lineH int
	}
	fields := []field{
		{"bordered", func(seed string, tag *event.Tag) layout.Widget {
			return liveTextField(t, input.TextFieldProps{Placeholder: "Email address", Seed: seed, FocusTag: func(tg event.Tag) { *tag = tg }})
		}, image.Pt(300, 60), p.WindowBackground, lineH},
		{"borderless", func(seed string, tag *event.Tag) layout.Widget {
			return loose(liveBorderless(t, input.BorderlessTextFieldProps{Placeholder: "Card title", Seed: seed, Surface: borderlessSurface, FocusTag: func(tg event.Tag) { *tag = tg }}))
		}, borderlessSize, borderlessSurface(p), bodyH},
	}
	cases := []struct {
		name  string
		seed  string
		caret func(n int) int
	}{
		{"empty", "", func(int) int { return 0 }},
		{"after the text", "hello@example", func(n int) int { return n }},
		{"inside the text", "hello@example", func(int) int { return 5 }},
	}
	for _, f := range fields {
		for _, c := range cases {
			t.Run(f.name+" "+c.name, func(t *testing.T) {
				var tag event.Tag
				w := f.make(c.seed, &tag)
				r := new(gioinput.Router)
				ops := new(op.Ops)
				driveTextFieldFrame(w, ops, r, f.size)
				r.Source().Execute(key.FocusCmd{Tag: tag})
				driveTextFieldFrame(w, ops, r, f.size)
				ed := tag.(*widget.Editor)
				at := c.caret(ed.Len())
				ed.SetCaret(at, at)
				img := liveCapture(t, r, f.size, f.sheet, w)
				if img == nil {
					return
				}
				box, n := accentBox(img, insertionAccent)
				if n == 0 {
					t.Fatalf("no pixel reads the accent %v; the field drew no insertion point in it", insertionAccent)
				}
				if box.Dx() != 1 {
					t.Errorf("the insertion point is %d columns wide at %v, want one", box.Dx(), box)
				}
				if box.Dy() != f.lineH {
					t.Errorf("the insertion point is %d rows tall at %v, want the line's %d", box.Dy(), box, f.lineH)
				}
				if n != box.Dx()*box.Dy() {
					t.Errorf("the insertion point covers %d of the %d pixels of %v; it is one solid line", n, box.Dx()*box.Dy(), box)
				}
			})
		}
	}
}

// TestTheInsertionPointFollowsTheAccent asserts that the insertion point is
// drawn in the accent row of the platform set the field is themed with, so a
// themed accent moves it.
func TestTheInsertionPointFollowsTheAccent(t *testing.T) {
	if got, want := insertionAccent, (stdcolor.NRGBA{R: 0x17, G: 0x8b, B: 0xfb, A: 0xff}); got != want {
		t.Fatalf("the light set's accent row for the insertion point is %v, want the measured %v", got, want)
	}
	moved := tokens.PlatformLight.WithAccent(stdcolor.NRGBA{R: 0xff, G: 0x00, B: 0x00, A: 0xff})
	if moved.TextInsertionPoint == tokens.PlatformLight.TextInsertionPoint {
		t.Fatal("WithAccent leaves the row the insertion point reads; a themed accent would not reach it")
	}
}

// insertionSentinel is a platform set whose TextInsertionPoint is a colour
// no other row of the set carries. A field that draws its insertion point in
// any other row, the sidebar's selection among them, draws no pixel of it.
func insertionSentinel() tokens.PlatformColors {
	p := tokens.PlatformLight
	p.TextInsertionPoint = stdcolor.NRGBA{R: 0xd0, G: 0x20, B: 0x90, A: 0xff}
	return p
}

// insertionKind is one kind of text field, drawn live on a platform set with
// its placeholder, and drawn static as focused in an explicit state.
type insertionKind struct {
	name        string
	placeholder string
	size        image.Point
	lineH       int
	sheet       func(p tokens.PlatformColors) stdcolor.NRGBA
	live        func(t *testing.T, p tokens.PlatformColors, seed string, tag *event.Tag) layout.Widget
	static      func(t *testing.T, p tokens.PlatformColors, placeholder, text string, sel [2]int) layout.Widget
}

func insertionKinds() []insertionKind {
	on := func(p tokens.PlatformColors) theme.Theme {
		th := theme.Default()
		th.Platform = rx.Of(p)
		return th
	}
	window := func(p tokens.PlatformColors) stdcolor.NRGBA { return p.WindowBackground }
	sys := tokens.DefaultTypography.SystemFont
	return []insertionKind{
		{"bordered", "Email address", image.Pt(300, 60), int(sys.LineHeight), window,
			func(t *testing.T, p tokens.PlatformColors, seed string, tag *event.Tag) layout.Widget {
				return materialize(t, input.TextField(rx.Of(on(p)), input.TextFieldProps{Placeholder: "Email address", Seed: seed, Shaper: defaultShaper(t), FocusTag: func(tg event.Tag) { *tag = tg }}))
			},
			func(t *testing.T, p tokens.PlatformColors, placeholder, text string, sel [2]int) layout.Widget {
				return input.Render(defaultShaper(t), placeholder, p, tokens.Spacing, tokens.Radius, sys, tokens.Comfortable, input.RenderState{Focused: true, Text: text, Selection: sel})
			}},
		{"search", "Search", image.Pt(300, 60), int(sys.LineHeight), window,
			func(t *testing.T, p tokens.PlatformColors, seed string, tag *event.Tag) layout.Widget {
				return materialize(t, input.SearchField(rx.Of(on(p)), input.SearchFieldProps{Placeholder: "Search", Seed: seed, Shaper: defaultShaper(t), FocusTag: func(tg event.Tag) { *tag = tg }}))
			},
			func(t *testing.T, p tokens.PlatformColors, placeholder, text string, sel [2]int) layout.Widget {
				return input.RenderSearch(defaultShaper(t), placeholder, p, tokens.Spacing, tokens.Radius, sys, tokens.Comfortable, input.RenderState{Focused: true, Text: text, Selection: sel})
			}},
		{"borderless", "Card title", borderlessSize, int(tokens.DefaultTypography.Body.LineHeight), borderlessSurface,
			func(t *testing.T, p tokens.PlatformColors, seed string, tag *event.Tag) layout.Widget {
				return loose(materialize(t, input.BorderlessTextField(rx.Of(on(p)), input.BorderlessTextFieldProps{Placeholder: "Card title", Seed: seed, Shaper: defaultShaper(t), Surface: borderlessSurface, FocusTag: func(tg event.Tag) { *tag = tg }})))
			},
			func(t *testing.T, p tokens.PlatformColors, placeholder, text string, sel [2]int) layout.Widget {
				return loose(input.RenderBorderless(defaultShaper(t), placeholder, p, tokens.DefaultTypography.Body, input.BorderlessState{Surface: borderlessSurface(p), Focused: true, Text: text, Selection: sel}))
			}},
	}
}

// liveInsertionBox draws kind live on p holding seed, gives it the keyboard
// and puts the insertion point at the end of the value, and answers the box
// of the pixels that read p's TextInsertionPoint and their number.
func liveInsertionBox(t *testing.T, kind insertionKind, p tokens.PlatformColors, seed string) (image.Rectangle, int, *image.RGBA) {
	t.Helper()
	var tag event.Tag
	w := kind.live(t, p, seed, &tag)
	r := new(gioinput.Router)
	ops := new(op.Ops)
	driveTextFieldFrame(w, ops, r, kind.size)
	r.Source().Execute(key.FocusCmd{Tag: tag})
	driveTextFieldFrame(w, ops, r, kind.size)
	ed := tag.(*widget.Editor)
	ed.SetCaret(ed.Len(), ed.Len())
	img := liveCapture(t, r, kind.size, kind.sheet(p), w)
	if img == nil {
		return image.Rectangle{}, 0, nil
	}
	box, n := accentBox(img, p.TextInsertionPoint)
	return box, n, img
}

// TestEveryFieldKindDrawsTheInsertionPointInItsOwnColour asserts that the
// bordered, the search and the borderless field draw their insertion point
// in the platform set's TextInsertionPoint and in no other row, live and
// static. The static field drawn as focused puts it where the live field
// puts it with the insertion point at the end of the value: after the last
// character, or where the text begins when the field is empty. A static
// field drawn with its value selected draws none.
func TestEveryFieldKindDrawsTheInsertionPointInItsOwnColour(t *testing.T) {
	p := insertionSentinel()
	for _, kind := range insertionKinds() {
		seeds := []string{"", "hello@example"}
		if kind.name == "borderless" {
			// A borderless field wraps its value: the insertion point stands
			// after the last character, on the last line.
			seeds = append(seeds, "Draft the release notes for the next round of the board")
		}
		for i, seed := range seeds {
			name := kind.name + " empty"
			if i == 1 {
				name = kind.name + " after the value"
			} else if i == 2 {
				name = kind.name + " after a wrapped value"
			}
			t.Run(name, func(t *testing.T) {
				live, n, img := liveInsertionBox(t, kind, p, seed)
				if img == nil {
					return
				}
				if n == 0 {
					t.Fatalf("the live field drew no pixel in TextInsertionPoint %v", p.TextInsertionPoint)
				}
				if live.Dx() != 1 || live.Dy() != kind.lineH || n != live.Dx()*live.Dy() {
					t.Errorf("the live insertion point covers %d pixels of %v, want one solid column %d rows tall", n, live, kind.lineH)
				}
				if i == 2 && live.Min.Y < kind.lineH {
					t.Fatalf("the live insertion point stands at %v, on the first line; the value did not wrap, so this measures nothing", live)
				}
				if _, m := accentBox(img, p.SidebarSelection); m != 0 {
					t.Errorf("%d pixels of the live field read SidebarSelection; the insertion point has its own colour", m)
				}
				static := golden.Capture(t, kind.size, onSheet(kind.sheet(p), kind.static(t, p, kind.placeholder, seed, [2]int{})))
				if static == nil {
					return
				}
				box, m := accentBox(static, p.TextInsertionPoint)
				if m == 0 {
					t.Fatalf("the static field drawn as focused drew no insertion point")
				}
				if box != live || m != n {
					t.Errorf("the static insertion point covers %d pixels of %v; the live field covers %d of %v", m, box, n, live)
				}
			})
		}
		t.Run(kind.name+" selected", func(t *testing.T) {
			img := golden.Capture(t, kind.size, onSheet(kind.sheet(p), kind.static(t, p, kind.placeholder, "hello@example", [2]int{0, 5})))
			if img == nil {
				return
			}
			if _, n := accentBox(img, p.TextInsertionPoint); n != 0 {
				t.Errorf("%d pixels read TextInsertionPoint beside a selected run; a selection draws no insertion point", n)
			}
			if _, ok := selectionFillBox(img, kind.size, p.SelectedTextBackground); !ok {
				t.Error("the static field painted no selection fill; this measures nothing")
			}
		})
	}
}

// TestNoInsertionPointBesideASelection asserts that a field holding a
// selected run draws the run and no insertion point: the selection shows
// where the typed text goes.
func TestNoInsertionPointBesideASelection(t *testing.T) {
	p := tokens.PlatformLight
	var tag event.Tag
	w := liveTextField(t, input.TextFieldProps{Placeholder: "Email address", Seed: "hello@example", FocusTag: func(tg event.Tag) { tag = tg }})
	r := new(gioinput.Router)
	ops := new(op.Ops)
	size := image.Pt(300, 60)
	driveTextFieldFrame(w, ops, r, size)
	r.Source().Execute(key.FocusCmd{Tag: tag})
	driveTextFieldFrame(w, ops, r, size)
	tag.(*widget.Editor).SetCaret(0, 5)
	img := liveCapture(t, r, size, p.WindowBackground, w)
	if img == nil {
		return
	}
	if _, n := accentBox(img, insertionAccent); n != 0 {
		t.Errorf("%d pixels read the accent beside a selected run; a selection draws no insertion point", n)
	}
	if _, ok := selectionFillBox(img, size, p.SelectedTextBackground); !ok {
		t.Error("the field painted no selection fill; this measures nothing")
	}
}

// TestAFieldFocusedByACommandSelectsItsValue asserts that a field that takes
// the keyboard by a command or by Tab has its whole value selected, with the
// insertion point at its end, and that the text typed then replaces the
// value; and that a press in the field's text puts the insertion point where
// the press is and selects nothing.
func TestAFieldFocusedByACommandSelectsItsValue(t *testing.T) {
	const seed = "Draft the notes"
	n := len(seed)
	type maker func(tag *event.Tag, changed *string) layout.Widget
	makers := []struct {
		name string
		make maker
		size image.Point
	}{
		{"bordered", func(tag *event.Tag, changed *string) layout.Widget {
			return liveTextField(t, input.TextFieldProps{Seed: seed, FocusTag: func(tg event.Tag) { *tag = tg }, OnChange: func(_ layout.Context, s string) { *changed = s }})
		}, image.Pt(300, 60)},
		{"borderless", func(tag *event.Tag, changed *string) layout.Widget {
			return loose(liveBorderless(t, input.BorderlessTextFieldProps{Seed: seed, Surface: borderlessSurface, FocusTag: func(tg event.Tag) { *tag = tg }, OnChange: func(_ layout.Context, s string) { *changed = s }}))
		}, borderlessSize},
	}
	for _, m := range makers {
		t.Run(m.name+" by a command", func(t *testing.T) {
			var tag event.Tag
			var changed string
			w := m.make(&tag, &changed)
			r := new(gioinput.Router)
			ops := new(op.Ops)
			driveTextFieldFrame(w, ops, r, m.size)
			r.Source().Execute(key.FocusCmd{Tag: tag})
			driveTextFieldFrame(w, ops, r, m.size)
			ed := tag.(*widget.Editor)
			if start, end := ed.Selection(); start != n || end != 0 {
				t.Fatalf("the field focused by a command selects %d to %d, want the whole value, 0 to %d, the insertion point at its end", end, start, n)
			}
			r.Queue(key.EditEvent{Range: key.Range{Start: 0, End: n}, Text: "Ship"})
			driveTextFieldFrame(w, ops, r, m.size)
			driveTextFieldFrame(w, ops, r, m.size)
			if changed != "Ship" {
				t.Errorf("typing into the selected value gives %q, want %q: the typed text replaces the value", changed, "Ship")
			}
		})
		t.Run(m.name+" by a press in its text", func(t *testing.T) {
			var tag event.Tag
			var changed string
			w := m.make(&tag, &changed)
			r := new(gioinput.Router)
			ops := new(op.Ops)
			driveTextFieldFrame(w, ops, r, m.size)
			at := f32.Pt(20, 8)
			r.Queue(
				pointer.Event{Kind: pointer.Press, Position: at, Buttons: pointer.ButtonPrimary, Source: pointer.Mouse},
				pointer.Event{Kind: pointer.Release, Position: at, Buttons: pointer.ButtonPrimary, Source: pointer.Mouse},
			)
			driveTextFieldFrame(w, ops, r, m.size)
			driveTextFieldFrame(w, ops, r, m.size)
			if !r.Source().Focused(tag) {
				t.Fatal("the press did not give the field the keyboard; this measures nothing")
			}
			ed := tag.(*widget.Editor)
			start, end := ed.Selection()
			if start != end {
				t.Errorf("the field focused by a press selects %d to %d, want no selection", end, start)
			}
			if start == 0 || start == n {
				t.Errorf("the press put the insertion point at %d, want where the press is, inside the value", start)
			}
		})
	}
	t.Run("by Tab", func(t *testing.T) {
		var first, second event.Tag
		a := liveTextField(t, input.TextFieldProps{Seed: "first", FocusTag: func(tg event.Tag) { first = tg }})
		b := liveTextField(t, input.TextFieldProps{Seed: seed, FocusTag: func(tg event.Tag) { second = tg }})
		size := image.Pt(300, 120)
		both := func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					gtx.Constraints.Max.Y = 40
					return a(gtx)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					gtx.Constraints.Max.Y = 40
					return b(gtx)
				}),
			)
		}
		r := new(gioinput.Router)
		ops := new(op.Ops)
		driveTextFieldFrame(both, ops, r, size)
		r.Source().Execute(key.FocusCmd{Tag: first})
		driveTextFieldFrame(both, ops, r, size)
		// Tab is the window moving the keyboard forward.
		r.MoveFocus(key.FocusForward)
		driveTextFieldFrame(both, ops, r, size)
		driveTextFieldFrame(both, ops, r, size)
		if !r.Source().Focused(second) {
			t.Fatal("Tab did not move the keyboard to the second field; this measures nothing")
		}
		if start, end := second.(*widget.Editor).Selection(); start != n || end != 0 {
			t.Errorf("the field Tab gave the keyboard to selects %d to %d, want the whole value, 0 to %d", end, start, n)
		}
	})
}

// TestEveryFieldKindKeepsItsPlaceholderWhileItHoldsTheKeyboard asserts that
// the bordered, the search and the borderless field, live, empty and holding
// the keyboard, draw their placeholder and the insertion point where the text
// begins, as TextEdit's find field is measured. After one typed character the
// field draws no placeholder.
func TestEveryFieldKindKeepsItsPlaceholderWhileItHoldsTheKeyboard(t *testing.T) {
	p := insertionSentinel()
	for _, kind := range insertionKinds() {
		t.Run(kind.name, func(t *testing.T) {
			capture := func(w layout.Widget) *image.RGBA {
				return golden.Capture(t, kind.size, onSheet(kind.sheet(p), w))
			}
			withPh := capture(kind.static(t, p, kind.placeholder, "", [2]int{}))
			bare := capture(kind.static(t, p, "", "", [2]int{}))
			if withPh == nil || bare == nil {
				return
			}
			ph, n := diffBox(withPh, bare)
			if n == 0 {
				t.Fatal("the static field drew no placeholder; this measures nothing")
			}
			// pastColumn counts the pixels of ph right of column x where img
			// differs from want, and where want differs from bare.
			pastColumn := func(img, want *image.RGBA, x int) (differ, drawn int) {
				for py := ph.Min.Y; py < ph.Max.Y; py++ {
					for px := max(x, ph.Min.X); px < ph.Max.X; px++ {
						if img.RGBAAt(px, py) != want.RGBAAt(px, py) {
							differ++
						}
						if want.RGBAAt(px, py) != bare.RGBAAt(px, py) {
							drawn++
						}
					}
				}
				return differ, drawn
			}

			var tag event.Tag
			w := kind.live(t, p, "", &tag)
			r := new(gioinput.Router)
			ops := new(op.Ops)
			driveTextFieldFrame(w, ops, r, kind.size)
			r.Source().Execute(key.FocusCmd{Tag: tag})
			driveTextFieldFrame(w, ops, r, kind.size)
			focused := liveCapture(t, r, kind.size, kind.sheet(p), w)
			if focused == nil {
				return
			}
			in, m := accentBox(focused, p.TextInsertionPoint)
			if m == 0 {
				t.Fatal("the live field drew no insertion point; it does not hold the keyboard")
			}
			if d := in.Min.X - ph.Min.X; d < -2 || d > 2 {
				t.Errorf("the insertion point stands at x %d; the text begins at x %d", in.Min.X, ph.Min.X)
			}
			if differ, drawn := pastColumn(focused, withPh, in.Max.X); drawn == 0 {
				t.Fatalf("the placeholder has no pixel right of the insertion point at x %d; this measures nothing", in.Max.X)
			} else if differ > 0 {
				t.Errorf("%d pixels of the placeholder right of the insertion point differ: the empty field that holds the keyboard hides its placeholder", differ)
			}

			// An edit event leaves the caret where the range began; a typed
			// key moves it past the character.
			r.Queue(key.EditEvent{Range: key.Range{Start: 0, End: 0}, Text: "W"})
			driveTextFieldFrame(w, ops, r, kind.size)
			ed := tag.(*widget.Editor)
			ed.SetCaret(ed.Len(), ed.Len())
			typed := liveCapture(t, r, kind.size, kind.sheet(p), w)
			if typed == nil {
				return
			}
			after, m := accentBox(typed, p.TextInsertionPoint)
			if m == 0 || after.Min.X <= in.Min.X {
				t.Fatalf("the insertion point stands at %v after one typed character; the character did not reach the field", after)
			}
			if _, drawn := pastColumn(bare, withPh, after.Max.X); drawn == 0 {
				t.Fatal("the placeholder has no pixel right of the typed character; this measures nothing")
			}
			if differ, _ := pastColumn(typed, bare, after.Max.X); differ > 0 {
				t.Errorf("%d pixels right of the typed character are not the empty field's; the placeholder stays after one typed character", differ)
			}
		})
	}
}

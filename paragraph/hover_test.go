package paragraph_test

import (
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

	"github.com/vibrantgio/components/paragraph"
	"github.com/vibrantgio/theme/tokens"
)

// hoverSpans is one line carrying two links with prose between them and after
// them, so the pointer has somewhere to be on a link, on another link, and on
// no link at all.
func hoverSpans() []paragraph.SpanStyle {
	return []paragraph.SpanStyle{
		{Content: "first", URL: "https://a.example/one"},
		{Content: " and "},
		{Content: "second", URL: "https://b.example/two"},
		{Content: " after."},
	}
}

// linkRuns reads back the x range of every link in a one-line paragraph from
// the shape the frame declares, so a test points where the faces actually put
// the links rather than at widths it guessed. It drives its own state and
// router and reports no hover of its own.
func linkRuns(shaper *text.Shaper, style paragraph.Style, spans []paragraph.SpanStyle, size image.Point) [][2]int {
	style.OnLinkHover = nil
	state := paragraph.NewState()
	w := func(gtx layout.Context) layout.Dimensions {
		return paragraph.Layout(gtx, state, shaper, style, spans)
	}
	r := new(gioinput.Router)
	ops := new(op.Ops)
	driveFrame(w, ops, r, size)
	driveFrame(w, ops, r, size)
	var runs [][2]int
	on := false
	for x := 0; x < size.X; x++ {
		r.Queue(pointer.Event{Kind: pointer.Move, Position: f32.Pt(float32(x), 8), Source: pointer.Mouse})
		driveFrame(w, ops, r, size)
		switch hand := r.Cursor() == pointer.CursorPointer; {
		case hand && !on:
			runs = append(runs, [2]int{x, x})
		case hand:
			runs[len(runs)-1][1] = x
		}
		on = r.Cursor() == pointer.CursorPointer
	}
	return runs
}

// TestLinkHoverReportsEveryChangeAndNothingElse drives the pointer onto a
// link, across it, straight onto a second link and off the end, counting the
// calls: one per change of the link under the pointer, none on the frames
// between them.
func TestLinkHoverReportsEveryChangeAndNothingElse(t *testing.T) {
	shaper := defaultShaper(t)
	spans := hoverSpans()
	style := paragraph.FromTokens(tokens.PlatformLight, tokens.DefaultTypography.BodyLarge, tokens.PlatformLight.WindowBackground)
	size := image.Pt(400, 100)

	runs := linkRuns(shaper, style, spans, size)
	if len(runs) != 2 {
		t.Fatalf("the line declares %d link runs, want 2: %v", len(runs), runs)
	}
	mid := func(run [2]int) int { return (run[0] + run[1]) / 2 }

	var got []string
	style.OnLinkHover = func(gtx layout.Context, u string) {
		if gtx.Ops == nil {
			t.Error("OnLinkHover received a layout.Context without Ops; callbacks must carry the live gtx")
		}
		got = append(got, u)
	}
	state := paragraph.NewState()
	w := func(gtx layout.Context) layout.Dimensions {
		return paragraph.Layout(gtx, state, shaper, style, spans)
	}
	r := new(gioinput.Router)
	ops := new(op.Ops)
	idle := func() { driveFrame(w, ops, r, size) }
	move := func(x int) {
		r.Queue(pointer.Event{Kind: pointer.Move, Position: f32.Pt(float32(x), 8), Source: pointer.Mouse})
		driveFrame(w, ops, r, size)
	}

	idle()
	idle()
	if len(got) != 0 {
		t.Fatalf("a paragraph the pointer has not reached reported %v", got)
	}

	move(mid(runs[0]))
	if want := []string{"https://a.example/one"}; !equalStrings(got, want) {
		t.Fatalf("entering the first link reported %v, want %v", got, want)
	}

	// Moving within the same link, and frames with no pointer event at all,
	// are not changes.
	move(runs[0][1])
	idle()
	idle()
	if len(got) != 1 {
		t.Fatalf("staying on one link reported %v, want one call", got)
	}

	// Straight from one link onto the other: one call, carrying the new URL.
	move(mid(runs[1]))
	if want := []string{"https://a.example/one", "https://b.example/two"}; !equalStrings(got, want) {
		t.Fatalf("crossing to the second link reported %v, want %v", got, want)
	}

	// Off the end of the links onto the prose: the empty string, once.
	move(runs[1][1] + 2)
	idle()
	want := []string{"https://a.example/one", "https://b.example/two", ""}
	if !equalStrings(got, want) {
		t.Fatalf("leaving the link reported %v, want %v", got, want)
	}
}

// TestLinkHoverIsNilSafe drives a pointer across a link with no hover
// callback set, which is every caller written before there was one.
func TestLinkHoverIsNilSafe(t *testing.T) {
	shaper := defaultShaper(t)
	style := paragraph.FromTokens(tokens.PlatformLight, tokens.DefaultTypography.BodyLarge, tokens.PlatformLight.WindowBackground)
	state := paragraph.NewState()
	spans := hoverSpans()
	w := func(gtx layout.Context) layout.Dimensions {
		return paragraph.Layout(gtx, state, shaper, style, spans)
	}
	r := new(gioinput.Router)
	ops := new(op.Ops)
	size := image.Pt(400, 100)
	driveFrame(w, ops, r, size)
	driveFrame(w, ops, r, size)
	r.Queue(pointer.Event{Kind: pointer.Move, Position: f32.Pt(6, 8), Source: pointer.Mouse})
	driveFrame(w, ops, r, size)
	r.Queue(pointer.Event{Kind: pointer.Move, Position: f32.Pt(390, 8), Source: pointer.Mouse})
	driveFrame(w, ops, r, size)
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// TestHoveredLinkBoxIsTheLinkUnderThePointer drives the pointer onto each
// link in turn and reads the box back: the first link's box holds the x range
// the frame declared for it, the second link's holds its own, and neither is
// reported while the pointer is on the prose.
func TestHoveredLinkBoxIsTheLinkUnderThePointer(t *testing.T) {
	shaper := defaultShaper(t)
	spans := hoverSpans()
	style := paragraph.FromTokens(tokens.PlatformLight, tokens.DefaultTypography.BodyLarge, tokens.PlatformLight.WindowBackground)
	size := image.Pt(400, 100)

	runs := linkRuns(shaper, style, spans, size)
	if len(runs) != 2 {
		t.Fatalf("the line declares %d link runs, want 2: %v", len(runs), runs)
	}

	state := paragraph.NewState()
	w := func(gtx layout.Context) layout.Dimensions {
		return paragraph.Layout(gtx, state, shaper, style, spans)
	}
	r := new(gioinput.Router)
	ops := new(op.Ops)
	move := func(x int) {
		r.Queue(pointer.Event{Kind: pointer.Move, Position: f32.Pt(float32(x), 8), Source: pointer.Mouse})
		driveFrame(w, ops, r, size)
	}

	driveFrame(w, ops, r, size)
	driveFrame(w, ops, r, size)
	if _, ok := state.HoveredLinkBox(); ok {
		t.Fatal("a paragraph the pointer has not reached reports a box")
	}

	for i, run := range runs {
		move((run[0] + run[1]) / 2)
		box, ok := state.HoveredLinkBox()
		if !ok {
			t.Fatalf("link %d is hovered and reports no box", i)
		}
		if box.Empty() {
			t.Fatalf("link %d reports the empty box", i)
		}
		// The declared run is the pointer's own reading of where the link
		// is, so the box has to hold it rather than merely overlap it.
		if box.Min.X > run[0] || box.Max.X < run[1] {
			t.Errorf("link %d reports x range [%d,%d), which does not hold the declared run [%d,%d]",
				i, box.Min.X, box.Max.X, run[0], run[1])
		}
		if box.Min.Y < 0 || box.Max.Y > size.Y {
			t.Errorf("link %d reports y range [%d,%d) outside the paragraph", i, box.Min.Y, box.Max.Y)
		}
	}

	// The two links are different places, and leaving them both reports no
	// box at all.
	move((runs[0][0] + runs[0][1]) / 2)
	first, _ := state.HoveredLinkBox()
	move((runs[1][0] + runs[1][1]) / 2)
	second, _ := state.HoveredLinkBox()
	if first == second {
		t.Errorf("both links report the same box %v", first)
	}
	move(runs[1][1] + 2)
	if box, ok := state.HoveredLinkBox(); ok {
		t.Errorf("the pointer is on the prose and the box %v is still reported", box)
	}
}

// TestHoveredLinkBoxIsTheRunUnderThePointer wraps one link across two lines
// and points at each of them in turn: the box is the run pointed at and not
// the block around both, so an annotation placed against it stands beside the
// line the reader is pointing at.
func TestHoveredLinkBoxIsTheRunUnderThePointer(t *testing.T) {
	shaper := defaultShaper(t)
	spans := []paragraph.SpanStyle{
		{Content: "one two three four five six seven", URL: "https://a.example/long"},
	}
	style := paragraph.FromTokens(tokens.PlatformLight, tokens.DefaultTypography.BodyLarge, tokens.PlatformLight.WindowBackground)
	// Narrow enough that the one link has to be carried onto a second line.
	size := image.Pt(120, 120)

	state := paragraph.NewState()
	var height int
	w := func(gtx layout.Context) layout.Dimensions {
		dims := paragraph.Layout(gtx, state, shaper, style, spans)
		height = dims.Size.Y
		return dims
	}
	r := new(gioinput.Router)
	ops := new(op.Ops)
	move := func(y int) {
		r.Queue(pointer.Event{Kind: pointer.Move, Position: f32.Pt(4, float32(y)), Source: pointer.Mouse})
		driveFrame(w, ops, r, size)
	}
	driveFrame(w, ops, r, size)
	driveFrame(w, ops, r, size)

	move(4)
	first, ok := state.HoveredLinkBox()
	if !ok {
		t.Fatal("the wrapped link is hovered on its first line and reports no box")
	}
	if first.Dy() >= height {
		t.Fatalf("the first line's box is %d tall against a paragraph %d tall; it is not one run", first.Dy(), height)
	}

	move(height - 4)
	second, ok := state.HoveredLinkBox()
	if !ok {
		t.Fatal("the wrapped link is hovered on its second line and reports no box")
	}
	if second == first {
		t.Fatalf("both lines of the link report the same box %v; the box is not the run under the pointer", first)
	}
	if second.Min.Y < first.Max.Y {
		t.Errorf("the second line's box %v begins above the first line's foot at %d", second, first.Max.Y)
	}
}

// TestFocusedLinkBoxSpansEveryLineOfAWrappedLink gives the keyboard to a link
// wrapped across two lines and reads its box: the block around both runs,
// since nothing picks a run when no pointer points at one.
func TestFocusedLinkBoxSpansEveryLineOfAWrappedLink(t *testing.T) {
	shaper := defaultShaper(t)
	spans := []paragraph.SpanStyle{
		{Content: "one two three four five six seven", URL: "https://a.example/long"},
	}
	style := paragraph.FromTokens(tokens.PlatformLight, tokens.DefaultTypography.BodyLarge, tokens.PlatformLight.WindowBackground)
	size := image.Pt(120, 120)

	state := paragraph.NewState()
	var (
		height int
		box    image.Rectangle
		boxed  bool
	)
	w := func(gtx layout.Context) layout.Dimensions {
		dims := paragraph.Layout(gtx, state, shaper, style, spans)
		height = dims.Size.Y
		box, boxed = state.FocusedLinkBox(gtx)
		return dims
	}
	r := new(gioinput.Router)
	ops := new(op.Ops)
	driveFrame(w, ops, r, size)
	driveFrame(w, ops, r, size)
	if boxed {
		t.Fatal("a paragraph the keyboard has not reached reports a focused box")
	}

	r.MoveFocus(key.FocusForward)
	driveFrame(w, ops, r, size)
	if !boxed {
		t.Fatal("the link holds the keyboard and reports no box")
	}
	if box.Dy() >= height {
		t.Fatalf("the focused link's box is %d tall against a paragraph %d tall; it is taller than the link", box.Dy(), height)
	}
	if box.Dy() <= height/3 {
		t.Errorf("the focused link's box is %d tall in a paragraph %d tall, so it holds one line and not every line of the link", box.Dy(), height)
	}
}

// TestTheKeyboardReportsTheLinkItReaches tabs onto each link in turn and off
// the end of them, counting the reports: the address of the link the keyboard
// lands on, and the empty string when it leaves the last one.
func TestTheKeyboardReportsTheLinkItReaches(t *testing.T) {
	shaper := defaultShaper(t)
	spans := hoverSpans()
	style := paragraph.FromTokens(tokens.PlatformLight, tokens.DefaultTypography.BodyLarge, tokens.PlatformLight.WindowBackground)
	size := image.Pt(400, 100)

	var got []string
	style.OnLinkHover = func(_ layout.Context, u string) { got = append(got, u) }
	state := paragraph.NewState()
	// A focusable beside the paragraph, so the keyboard has somewhere to go
	// after the last link instead of wrapping back onto the first.
	var beyond int
	w := func(gtx layout.Context) layout.Dimensions {
		dims := paragraph.Layout(gtx, state, shaper, style, spans)
		for {
			if _, ok := gtx.Event(key.FocusFilter{Target: &beyond}); !ok {
				break
			}
		}
		event.Op(gtx.Ops, &beyond)
		return dims
	}
	r := new(gioinput.Router)
	ops := new(op.Ops)
	driveFrame(w, ops, r, size)
	driveFrame(w, ops, r, size)
	if len(got) != 0 {
		t.Fatalf("a paragraph nothing has reached reported %v", got)
	}

	r.MoveFocus(key.FocusForward)
	driveFrame(w, ops, r, size)
	if want := []string{"https://a.example/one"}; !equalStrings(got, want) {
		t.Fatalf("the keyboard reaching the first link reported %v, want %v", got, want)
	}
	// Frames in which the focus does not move are not changes.
	driveFrame(w, ops, r, size)
	driveFrame(w, ops, r, size)
	if len(got) != 1 {
		t.Fatalf("a link holding the keyboard reported %v, want one call", got)
	}

	r.MoveFocus(key.FocusForward)
	driveFrame(w, ops, r, size)
	if want := []string{"https://a.example/one", "https://b.example/two"}; !equalStrings(got, want) {
		t.Fatalf("the keyboard reaching the second link reported %v, want %v", got, want)
	}

	// Off the last link onto the focusable beyond the paragraph: the
	// keyboard leaves the links and the address goes with it.
	r.MoveFocus(key.FocusForward)
	driveFrame(w, ops, r, size)
	want := []string{"https://a.example/one", "https://b.example/two", ""}
	if !equalStrings(got, want) {
		t.Fatalf("the keyboard leaving the links reported %v, want %v", got, want)
	}
}

// TestThePointerWinsOverTheKeyboard rests the pointer on one link and moves
// the keyboard focus between the two: the address stays the pointer's for as
// long as it is on a link, and the focused link's address is said only once
// the pointer leaves the prose.
func TestThePointerWinsOverTheKeyboard(t *testing.T) {
	shaper := defaultShaper(t)
	spans := hoverSpans()
	style := paragraph.FromTokens(tokens.PlatformLight, tokens.DefaultTypography.BodyLarge, tokens.PlatformLight.WindowBackground)
	size := image.Pt(400, 100)

	runs := linkRuns(shaper, style, spans, size)
	if len(runs) != 2 {
		t.Fatalf("the line declares %d link runs, want 2: %v", len(runs), runs)
	}

	var got []string
	style.OnLinkHover = func(_ layout.Context, u string) { got = append(got, u) }
	state := paragraph.NewState()
	w := func(gtx layout.Context) layout.Dimensions {
		return paragraph.Layout(gtx, state, shaper, style, spans)
	}
	r := new(gioinput.Router)
	ops := new(op.Ops)
	move := func(x int) {
		r.Queue(pointer.Event{Kind: pointer.Move, Position: f32.Pt(float32(x), 8), Source: pointer.Mouse})
		driveFrame(w, ops, r, size)
	}
	driveFrame(w, ops, r, size)
	driveFrame(w, ops, r, size)

	// The pointer on the first link, then the keyboard onto the second.
	move((runs[0][0] + runs[0][1]) / 2)
	r.MoveFocus(key.FocusForward)
	driveFrame(w, ops, r, size)
	r.MoveFocus(key.FocusForward)
	driveFrame(w, ops, r, size)
	if want := []string{"https://a.example/one"}; !equalStrings(got, want) {
		t.Fatalf("with the pointer resting on the first link the keyboard reported %v, want %v", got, want)
	}
	if box, ok := state.HoveredLinkBox(); !ok || box.Min.X > runs[0][0] || box.Max.X < runs[0][1] {
		t.Errorf("the box %v (reported %v) is not the first link's run [%d,%d]", box, ok, runs[0][0], runs[0][1])
	}

	// The pointer leaves the prose and the focused link gets the address.
	move(runs[1][1] + 2)
	want := []string{"https://a.example/one", "https://b.example/two"}
	if !equalStrings(got, want) {
		t.Fatalf("the pointer leaving reported %v, want %v", got, want)
	}
}

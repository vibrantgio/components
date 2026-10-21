package paragraph_test

import (
	"image"
	"testing"

	"gioui.org/f32"
	gioinput "gioui.org/io/input"
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

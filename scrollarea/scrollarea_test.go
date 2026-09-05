package scrollarea

import (
	"image"
	"testing"

	"gioui.org/f32"
	"gioui.org/gesture"
	gioinput "gioui.org/io/input"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/unit"

	"github.com/vibrantgio/components/scrollbar"
	"github.com/vibrantgio/theme/tokens"
)

// testContext returns a bare layout context at the given size with a
// 1:1 dp-to-px metric.
func testContext(ops *op.Ops, size image.Point) layout.Context {
	return layout.Context{
		Ops:         ops,
		Metric:      unit.Metric{PxPerDp: 1, PxPerSp: 1},
		Constraints: layout.Constraints{Max: size},
	}
}

// block returns a layout.Widget of a fixed natural size that ignores its
// constraints, exactly as content which must not reflow does.
func block(size image.Point) layout.Widget {
	return func(layout.Context) layout.Dimensions {
		return layout.Dimensions{Size: size}
	}
}

// TestFitsAndOverflows pins the boundary between the two states a scroll area
// can be in. Content up to the viewport's width reports everything visible,
// nothing to scroll to, and its own width; content past it reports the
// overflow, the viewport's width, and the fraction of itself on screen.
func TestFitsAndOverflows(t *testing.T) {
	style := FromTokens(tokens.DefaultLight)
	viewport := image.Pt(200, 50)

	cases := []struct {
		name      string
		content   int
		overflows bool
		maxOffset int
		width     int
		end       float32
	}{
		{name: "narrower", content: 120, overflows: false, maxOffset: 0, width: 120, end: 1},
		{name: "exact", content: 200, overflows: false, maxOffset: 0, width: 200, end: 1},
		{name: "one past", content: 201, overflows: true, maxOffset: 1, width: 200, end: 200.0 / 201.0},
		{name: "wider", content: 400, overflows: true, maxOffset: 200, width: 200, end: 0.5},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var ops op.Ops
			state := NewState()
			dims := style.Layout(testContext(&ops, viewport), state, block(image.Pt(tc.content, 50)))

			if got := state.Overflows(); got != tc.overflows {
				t.Errorf("Overflows() = %v, want %v", got, tc.overflows)
			}
			if got := state.MaxOffset(); got != tc.maxOffset {
				t.Errorf("MaxOffset() = %d, want %d", got, tc.maxOffset)
			}
			if dims.Size.X != tc.width {
				t.Errorf("width = %d, want %d", dims.Size.X, tc.width)
			}
			if dims.Size.Y != 50 {
				t.Errorf("height = %d, want the content's 50", dims.Size.Y)
			}
			if got := state.Content(); got != tc.content {
				t.Errorf("Content() = %d, want %d", got, tc.content)
			}
			if got := state.Viewport(); got != viewport.X {
				t.Errorf("Viewport() = %d, want %d", got, viewport.X)
			}
			start, end := state.Fractions()
			if start != 0 || end != tc.end {
				t.Errorf("Fractions() = (%v, %v), want (0, %v)", start, end, tc.end)
			}
		})
	}
}

// TestOffsetBounds pins the offset's bounds: it never leaves [0, MaxOffset],
// whichever end it is pushed past and whether it is pushed absolutely or by a
// delta. A viewport that grows until the content fits pulls a far-out offset
// back with it rather than leaving the content scrolled past its own end.
func TestOffsetBounds(t *testing.T) {
	style := FromTokens(tokens.DefaultLight)
	var ops op.Ops
	state := NewState()
	style.Layout(testContext(&ops, image.Pt(200, 50)), state, block(image.Pt(400, 50)))

	cases := []struct {
		name string
		set  func()
		want int
	}{
		{"past the end", func() { state.SetOffset(10_000) }, 200},
		{"before the start", func() { state.SetOffset(-10_000) }, 0},
		{"inside", func() { state.SetOffset(75) }, 75},
		{"at the end exactly", func() { state.SetOffset(200) }, 200},
		{"delta past the end", func() { state.SetOffset(190); state.ScrollBy(50) }, 200},
		{"delta before the start", func() { state.SetOffset(10); state.ScrollBy(-50) }, 0},
		{"delta inside", func() { state.SetOffset(10); state.ScrollBy(50) }, 60},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tc.set()
			if got := state.Offset(); got != tc.want {
				t.Errorf("Offset() = %d, want %d", got, tc.want)
			}
		})
	}

	state.SetOffset(200)
	style.Layout(testContext(&ops, image.Pt(600, 50)), state, block(image.Pt(400, 50)))
	if got := state.Offset(); got != 0 {
		t.Errorf("Offset() = %d after the content came to fit; want 0", got)
	}
}

// TestScrollClaimsHorizontalAxisOnly is the axis-separation proof: a
// horizontal gesture over the area scrolls the area, and a vertical one over
// the same pixels reaches the ancestor that scrolls the column instead. The
// two handlers are stacked exactly as a code block inside a document list is.
func TestScrollClaimsHorizontalAxisOnly(t *testing.T) {
	style := FromTokens(tokens.DefaultLight)
	state := NewState()
	var outer gesture.Scroll
	var outerTotal int

	r := new(gioinput.Router)
	var ops op.Ops
	frame := func() {
		ops.Reset()
		gtx := testContext(&ops, image.Pt(200, 50))
		gtx.Source = r.Source()
		// The ancestor: a vertical scroller covering the whole area, with
		// room to move in both directions.
		outerTotal += outer.Update(gtx.Metric, gtx.Source, gtx.Now, gesture.Vertical,
			pointer.ScrollRange{}, pointer.ScrollRange{Min: -1000, Max: 1000})
		area := clip.Rect{Max: image.Pt(200, 50)}.Push(&ops)
		outer.Add(&ops)
		style.Layout(gtx, state, block(image.Pt(400, 50)))
		area.Pop()
		r.Frame(&ops)
	}
	frame()

	r.Queue(pointer.Event{
		Kind:     pointer.Scroll,
		Position: f32.Pt(100, 25),
		Scroll:   f32.Pt(60, 0),
		Source:   pointer.Mouse,
	})
	frame()
	if state.Offset() != 60 {
		t.Errorf("horizontal gesture left the area at offset %d, want 60", state.Offset())
	}
	if outerTotal != 0 {
		t.Errorf("horizontal gesture moved the ancestor by %d, want 0", outerTotal)
	}

	r.Queue(pointer.Event{
		Kind:     pointer.Scroll,
		Position: f32.Pt(100, 25),
		Scroll:   f32.Pt(0, 40),
		Source:   pointer.Mouse,
	})
	frame()
	if outerTotal != 40 {
		t.Errorf("vertical gesture over the area moved the ancestor by %d, want 40; the area must not claim the vertical axis", outerTotal)
	}
	if state.Offset() != 60 {
		t.Errorf("vertical gesture moved the area to offset %d, want it left at 60", state.Offset())
	}
}

// TestFittingContentIsTransparentToThePointer is the other half of the axis
// contract: an area whose content fits registers no scroll range at all, so
// even a horizontal gesture over it is free to reach an ancestor that can use
// one.
func TestFittingContentIsTransparentToThePointer(t *testing.T) {
	style := FromTokens(tokens.DefaultLight)
	state := NewState()
	var outer gesture.Scroll
	var outerTotal int

	r := new(gioinput.Router)
	var ops op.Ops
	frame := func() {
		ops.Reset()
		gtx := testContext(&ops, image.Pt(200, 50))
		gtx.Source = r.Source()
		outerTotal += outer.Update(gtx.Metric, gtx.Source, gtx.Now, gesture.Horizontal,
			pointer.ScrollRange{Min: -1000, Max: 1000}, pointer.ScrollRange{})
		area := clip.Rect{Max: image.Pt(200, 50)}.Push(&ops)
		outer.Add(&ops)
		style.Layout(gtx, state, block(image.Pt(120, 50)))
		area.Pop()
		r.Frame(&ops)
	}
	frame()

	r.Queue(pointer.Event{
		Kind:     pointer.Scroll,
		Position: f32.Pt(100, 25),
		Scroll:   f32.Pt(60, 0),
		Source:   pointer.Mouse,
	})
	frame()
	if outerTotal != 60 {
		t.Errorf("horizontal gesture over a fitting area moved the ancestor by %d, want 60", outerTotal)
	}
	if state.Offset() != 0 {
		t.Errorf("fitting area scrolled to %d, want 0", state.Offset())
	}
}

// TestScrollbarDragScrolls verifies the bar is wired both ways: it reports
// where the viewport sits, and dragging it moves the viewport. The bar sits
// on the area's trailing edge, so the drag runs along the bottom strip.
func TestScrollbarDragScrolls(t *testing.T) {
	style := FromTokens(tokens.DefaultLight)
	bar := scrollbar.FromTokens(tokens.DefaultLight)
	state := NewState()

	r := new(gioinput.Router)
	var ops op.Ops
	frame := func() {
		ops.Reset()
		gtx := testContext(&ops, image.Pt(200, 50))
		gtx.Source = r.Source()
		style.LayoutScrollbar(gtx, state, bar, block(image.Pt(400, 50)))
		r.Frame(&ops)
	}
	frame()
	if state.Offset() != 0 {
		t.Fatalf("fresh area at offset %d, want 0", state.Offset())
	}

	// Grab the thumb near the leading end of the track and pull it a
	// quarter of the track's length along. The bar reports drags as a
	// fraction of the content, so a quarter of 400 px is 100 px.
	const y = 45
	r.Queue(
		pointer.Event{Kind: pointer.Press, Position: f32.Pt(20, y), Source: pointer.Mouse, Buttons: pointer.ButtonPrimary},
		pointer.Event{Kind: pointer.Move, Position: f32.Pt(20, y), Source: pointer.Mouse, Buttons: pointer.ButtonPrimary},
		pointer.Event{Kind: pointer.Move, Position: f32.Pt(70, y), Source: pointer.Mouse, Buttons: pointer.ButtonPrimary},
	)
	frame()
	if got, want := state.Offset(), 100; got != want {
		t.Errorf("offset %d after dragging the bar a quarter of the track, want %d", got, want)
	}
	if !state.Overflows() {
		t.Error("area stopped reporting overflow while dragging")
	}
}

// TestFadeIsNotLayout pins the fade's opt-out and its cost: either half of
// the style at zero draws no dissolve, and none of the four combinations
// moves anything.
func TestFadeIsNotLayout(t *testing.T) {
	base := FromTokens(tokens.DefaultLight)
	cases := []struct {
		name  string
		style Style
	}{
		{"default", base},
		{"no run", Style{FadeColor: base.FadeColor}},
		{"no colour", Style{Fade: base.Fade}},
		{"zero style", Style{}},
	}
	var want layout.Dimensions
	for i, tc := range cases {
		var ops op.Ops
		state := NewState()
		got := tc.style.Layout(testContext(&ops, image.Pt(200, 50)), state, block(image.Pt(400, 50)))
		if i == 0 {
			want = got
			continue
		}
		if got != want {
			t.Errorf("%s laid out %v, want the default's %v; the fade must not move anything", tc.name, got, want)
		}
	}
}

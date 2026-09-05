package layout

import (
	"image"
	"testing"

	"gioui.org/io/event"
	"gioui.org/io/input"
	"gioui.org/io/key"
	gio "gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
)

// TestFocusGroupTraversal verifies that FocusGroup.Focused() tracks which item
// has focus as the router advances focus with MoveFocus.
func TestFocusGroupTraversal(t *testing.T) {
	var r input.Router
	var g FocusGroup
	g.Grow(3)

	gtx := gio.Context{
		Ops:    new(op.Ops),
		Source: r.Source(),
	}

	// frame processes one layout frame: Update, register the event tags, submit to router.
	frame := func() {
		gtx.Reset()
		g.Update(gtx)
		// Register each item at a distinct non-overlapping position so that
		// spatial MoveFocus (Left/Right) also works; ForwardBackward only needs
		// any registered order.
		for i := 0; i < g.Len(); i++ {
			x := i * 30
			cl := clip.Rect{Min: image.Pt(x, 0), Max: image.Pt(x+20, 20)}.Push(gtx.Ops)
			event.Op(gtx.Ops, g.Tag(i))
			cl.Pop()
		}
		r.Frame(gtx.Ops)
	}

	// No focus yet.
	frame()
	if got := g.Focused(); got != -1 {
		t.Fatalf("initial: Focused() = %d, want -1", got)
	}

	// Explicitly focus item 0.
	gtx.Execute(key.FocusCmd{Tag: g.Tag(0)})
	frame()
	if got := g.Focused(); got != 0 {
		t.Fatalf("after Focus(0): Focused() = %d, want 0", got)
	}

	// Tab (FocusForward) → item 1.
	r.MoveFocus(key.FocusForward)
	frame()
	if got := g.Focused(); got != 1 {
		t.Errorf("after Tab: Focused() = %d, want 1", got)
	}

	// Tab → item 2.
	r.MoveFocus(key.FocusForward)
	frame()
	if got := g.Focused(); got != 2 {
		t.Errorf("after Tab: Focused() = %d, want 2", got)
	}

	// Shift+Tab (FocusBackward) → item 1.
	r.MoveFocus(key.FocusBackward)
	frame()
	if got := g.Focused(); got != 1 {
		t.Errorf("after Shift+Tab: Focused() = %d, want 1", got)
	}

	// Shift+Tab → item 0.
	r.MoveFocus(key.FocusBackward)
	frame()
	if got := g.Focused(); got != 0 {
		t.Errorf("after Shift+Tab: Focused() = %d, want 0", got)
	}
}

// TestFocusGroupFocusedReportsNoneWhenFocusLost verifies that Focused() returns
// -1 after focus leaves the group entirely (e.g. FocusCmd{Tag: nil}).
func TestFocusGroupFocusedReportsNoneWhenFocusLost(t *testing.T) {
	var r input.Router
	var g FocusGroup
	g.Grow(2)

	gtx := gio.Context{
		Ops:    new(op.Ops),
		Source: r.Source(),
	}

	frame := func() {
		gtx.Reset()
		g.Update(gtx)
		for i := 0; i < g.Len(); i++ {
			x := i * 30
			cl := clip.Rect{Min: image.Pt(x, 0), Max: image.Pt(x+20, 20)}.Push(gtx.Ops)
			event.Op(gtx.Ops, g.Tag(i))
			cl.Pop()
		}
		r.Frame(gtx.Ops)
	}

	// Focus item 0.
	gtx.Execute(key.FocusCmd{Tag: g.Tag(0)})
	frame()
	if got := g.Focused(); got != 0 {
		t.Fatalf("after Focus(0): Focused() = %d, want 0", got)
	}

	// Clear focus.
	gtx.Execute(key.FocusCmd{Tag: nil})
	frame()
	if got := g.Focused(); got != -1 {
		t.Errorf("after clear focus: Focused() = %d, want -1", got)
	}
}

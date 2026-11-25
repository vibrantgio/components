package input_test

import (
	"testing"

	"gioui.org/io/event"
	gioinput "gioui.org/io/input"
	"gioui.org/io/key"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/widget"

	"github.com/vibrantgio/components/input"
)

// typingRig drives a focused borderless field and records what it reports.
type typingRig struct {
	t      *testing.T
	field  layout.Widget
	editor *widget.Editor
	r      *gioinput.Router
	ops    *op.Ops
	// left records each LeaveUndo call: true for Redo.
	left []bool
	// state is the last OnTyping state, and reports counts the calls.
	state   [2]bool
	reports int
}

func newTypingRig(t *testing.T, seed string) *typingRig {
	t.Helper()
	rig := &typingRig{t: t, r: new(gioinput.Router), ops: new(op.Ops)}
	var tag event.Tag
	rig.field = liveBorderless(t, input.BorderlessTextFieldProps{
		Seed:      seed,
		FocusTag:  func(tg event.Tag) { tag = tg },
		LeaveUndo: func(_ layout.Context, redo bool) { rig.left = append(rig.left, redo) },
		OnTyping: func(_ layout.Context, undo, redo bool) {
			rig.state = [2]bool{undo, redo}
			rig.reports++
		},
	})
	rig.editor = tag.(*widget.Editor)
	rig.frame()
	rig.r.Source().Execute(key.FocusCmd{Tag: tag})
	rig.frame()
	return rig
}

func (rig *typingRig) frame() { driveTextFieldFrame(rig.field, rig.ops, rig.r, borderlessSize) }

func (rig *typingRig) chord(mods key.Modifiers) {
	rig.r.Queue(key.Event{Name: "Z", Modifiers: mods, State: key.Press})
	rig.frame()
}

func (rig *typingRig) typeAtEnd(s string) {
	n := rig.editor.Len()
	rig.r.Queue(key.EditEvent{Range: key.Range{Start: n, End: n}, Text: s})
	rig.frame()
}

func (rig *typingRig) want(text string, undo, redo bool, left ...bool) {
	rig.t.Helper()
	if got := rig.editor.Text(); got != text {
		rig.t.Errorf("the field holds %q, want %q", got, text)
	}
	if rig.state != [2]bool{undo, redo} {
		rig.t.Errorf("the field reports undo %v and redo %v, want %v and %v", rig.state[0], rig.state[1], undo, redo)
	}
	if len(rig.left) != len(left) {
		rig.t.Fatalf("the field left %v to the window, want %v", rig.left, left)
	}
	for i := range left {
		if rig.left[i] != left[i] {
			rig.t.Errorf("the field left %v to the window, want %v", rig.left, left)
		}
	}
}

// A field whose value the user has not changed holds no typing: Undo and
// Redo go to the window, and the seed stays.
func TestAnUnchangedFieldLeavesUndoToTheWindow(t *testing.T) {
	rig := newTypingRig(t, "Draft the notes")
	rig.chord(key.ModShortcut)
	rig.chord(key.ModShortcut | key.ModShift)
	rig.want("Draft the notes", false, false, false, true)
	if rig.reports != 0 {
		t.Errorf("an unchanged field reported its typing %d times", rig.reports)
	}
}

// An empty field holds no typing.
func TestAnEmptyFieldLeavesUndoToTheWindow(t *testing.T) {
	rig := newTypingRig(t, "")
	rig.chord(key.ModShortcut)
	rig.want("", false, false, false)
}

// Undo reverses the user's typing first. Once the typing is all reversed,
// Undo goes to the window, and Redo makes the typing again.
func TestUndoReversesTypingAndThenLeavesIt(t *testing.T) {
	rig := newTypingRig(t, "Draft")
	rig.typeAtEnd(" the")
	rig.typeAtEnd(" notes")
	rig.want("Draft the notes", true, false)
	rig.chord(key.ModShortcut)
	rig.want("Draft the", true, true)
	if start, end := rig.editor.Selection(); start != 9 || end != 9 {
		t.Errorf("the undo selects %d to %d, want the caret where the removed text was", start, end)
	}
	rig.chord(key.ModShortcut)
	rig.want("Draft", false, true)
	rig.chord(key.ModShortcut)
	rig.want("Draft", false, true, false)
	rig.chord(key.ModShortcut | key.ModShift)
	rig.want("Draft the", true, true, false)
	rig.typeAtEnd("m")
	rig.want("Draft them", true, false, false)
	rig.chord(key.ModShortcut | key.ModShift)
	rig.want("Draft them", true, false, false, true)
}

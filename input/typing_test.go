package input_test

import (
	"io"
	"strings"
	"testing"

	"github.com/reactivego/rx"

	"gioui.org/io/event"
	gioinput "gioui.org/io/input"
	"gioui.org/io/key"
	"gioui.org/io/transfer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/widget"

	"github.com/vibrantgio/components/input"
	"github.com/vibrantgio/theme/theme"
)

// typingHooks are the props every text field kind takes for its typing.
type typingHooks struct {
	seed     string
	focusTag func(event.Tag)
	leave    func(layout.Context, bool)
	onTyping func(layout.Context, input.Typing)
	onChange func(layout.Context, string)
	submit   bool
	// set receives the search field's SetText.
	set func(func(string))
}

// typingKinds builds each text field kind from the same hooks.
var typingKinds = []struct {
	name  string
	field func(t *testing.T, h typingHooks) layout.Widget
}{
	{"borderless", func(t *testing.T, h typingHooks) layout.Widget {
		return liveBorderless(t, input.BorderlessTextFieldProps{
			Seed: h.seed, FocusTag: h.focusTag, LeaveUndo: h.leave, OnTyping: h.onTyping,
			OnChange: h.onChange, Submit: h.submit,
		})
	}},
	{"bordered", func(t *testing.T, h typingHooks) layout.Widget {
		return liveTextField(t, input.TextFieldProps{
			Seed: h.seed, FocusTag: h.focusTag, LeaveUndo: h.leave, OnTyping: h.onTyping,
			OnChange: h.onChange, Submit: h.submit,
		})
	}},
	{"search", func(t *testing.T, h typingHooks) layout.Widget {
		return materialize(t, input.SearchField(rx.Of(theme.Default()), input.SearchFieldProps{
			Seed: h.seed, FocusTag: h.focusTag, LeaveUndo: h.leave, OnTyping: h.onTyping,
			OnChange: h.onChange, SetText: h.set, Shaper: defaultShaper(t),
		}))
	}},
}

// typingRig drives a focused text field and records what it reports.
type typingRig struct {
	t      *testing.T
	field  layout.Widget
	editor *widget.Editor
	r      *gioinput.Router
	ops    *op.Ops
	// left records each LeaveUndo call: true for Redo.
	left []bool
	// state is the last OnTyping state, and reports counts the calls.
	state   input.Typing
	reports int
	// changes counts the OnChange calls.
	changes int
	// set is the search field's SetText.
	set func(string)
}

func newTypingRig(t *testing.T, build func(*testing.T, typingHooks) layout.Widget, seed string, submit bool) *typingRig {
	t.Helper()
	rig := &typingRig{t: t, r: new(gioinput.Router), ops: new(op.Ops)}
	var tag event.Tag
	rig.field = build(t, typingHooks{
		seed:     seed,
		focusTag: func(tg event.Tag) { tag = tg },
		leave:    func(_ layout.Context, redo bool) { rig.left = append(rig.left, redo) },
		onTyping: func(_ layout.Context, s input.Typing) {
			rig.state = s
			rig.reports++
		},
		onChange: func(layout.Context, string) { rig.changes++ },
		submit:   submit,
		set:      func(s func(string)) { rig.set = s },
	})
	rig.editor = tag.(*widget.Editor)
	rig.frame()
	rig.r.Source().Execute(key.FocusCmd{Tag: tag})
	rig.frame()
	rig.changes = 0
	return rig
}

func (rig *typingRig) frame() { driveTextFieldFrame(rig.field, rig.ops, rig.r, borderlessSize) }

func (rig *typingRig) undo() { rig.chord(key.ModShortcut) }
func (rig *typingRig) redo() { rig.chord(key.ModShortcut | key.ModShift) }

func (rig *typingRig) chord(mods key.Modifiers) {
	rig.r.Queue(key.Event{Name: "Z", Modifiers: mods, State: key.Press})
	rig.frame()
}

// typeAtEnd moves the insertion point to the end of the text and types s
// there at once, as a new change.
func (rig *typingRig) typeAtEnd(s string) {
	rig.moveTo(rig.editor.Len())
	n := rig.editor.Len()
	rig.r.Queue(key.EditEvent{Range: key.Range{Start: n, End: n}, Text: s})
	rig.frame()
}

// moveTo puts the insertion point at rune offset at, as a click does.
func (rig *typingRig) moveTo(at int) {
	rig.r.Queue(key.SelectionEvent{Start: at, End: at})
	rig.frame()
}

// typeKeys types s key by key at the insertion point, one frame for each
// character. Each character is a replace of the selection and a selection
// after the character, as the platform's text input gives them.
func (rig *typingRig) typeKeys(s string) {
	for _, r := range s {
		start, end := rig.editor.Selection()
		start = min(start, end)
		rig.r.Queue(
			key.EditEvent{Range: key.Range{Start: start, End: end}, Text: string(r)},
			key.SelectionEvent{Start: start + 1, End: start + 1},
		)
		rig.frame()
	}
}

// press presses the key name with mods.
func (rig *typingRig) press(name key.Name, mods key.Modifiers) {
	rig.r.Queue(key.Event{Name: name, Modifiers: mods, State: key.Press})
	rig.frame()
}

// paste presses Cmd-V and gives the field s from the clipboard.
func (rig *typingRig) paste(s string) {
	rig.press("V", key.ModShortcut)
	rig.r.Queue(transfer.DataEvent{Type: "application/text", Open: func() io.ReadCloser {
		return io.NopCloser(strings.NewReader(s))
	}})
	rig.frame()
}

// focus gives the field the keyboard, or takes it away.
func (rig *typingRig) focus(on bool) {
	var tag event.Tag
	if on {
		tag = rig.editor
	}
	rig.r.Source().Execute(key.FocusCmd{Tag: tag})
	rig.frame()
}

// deleteAll removes the whole text as one change.
func (rig *typingRig) deleteAll() {
	rig.r.Queue(key.EditEvent{Range: key.Range{Start: 0, End: rig.editor.Len()}, Text: ""})
	rig.frame()
}

func (rig *typingRig) want(text string, state input.Typing, left ...bool) {
	rig.t.Helper()
	if got := rig.editor.Text(); got != text {
		rig.t.Errorf("the field holds %q, want %q", got, text)
	}
	if rig.state != state {
		rig.t.Errorf("the field reports %+v, want %+v", rig.state, state)
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

var (
	noTyping   = input.Typing{}
	undoOnly   = input.Typing{Undo: true, RedoEnded: true}
	undoRedo   = input.Typing{Undo: true, Redo: true}
	redoOnly   = input.Typing{Redo: true}
	undoNoRedo = input.Typing{Undo: true}
)

// A field that the user has not typed in holds no typing, whether it is
// empty or holds a value: Undo and Redo go to the window, and the value
// stays.
func TestAFieldWithoutTypingLeavesUndoToTheWindow(t *testing.T) {
	for _, kind := range typingKinds {
		for _, seed := range []string{"Draft the notes", ""} {
			t.Run(kind.name+"/"+seed, func(t *testing.T) {
				rig := newTypingRig(t, kind.field, seed, false)
				rig.undo()
				rig.redo()
				rig.want(seed, noTyping, false, true)
				if rig.reports != 0 {
					t.Errorf("a field without typing reported its typing %d times", rig.reports)
				}
			})
		}
	}
}

// A field that the user made empty holds typing: Undo puts the value back.
func TestAFieldMadeEmptyTakesUndo(t *testing.T) {
	for _, kind := range typingKinds {
		t.Run(kind.name, func(t *testing.T) {
			rig := newTypingRig(t, kind.field, "Draft", false)
			rig.deleteAll()
			rig.want("", undoOnly)
			rig.undo()
			rig.want("Draft", redoOnly)
			if rig.changes != 2 {
				t.Errorf("the deletion and the undo reported %d changes, want 2", rig.changes)
			}
		})
	}
}

// Redo follows the order of the undos. The user types "ab", then "cd",
// and presses Cmd-Z: the field shows "ab", and Redo makes "cd" again.
// New typing ends Redo, in the field and in the window, until an Undo runs.
func TestRedoFollowsTheOrderOfTheUndos(t *testing.T) {
	for _, kind := range typingKinds {
		t.Run(kind.name, func(t *testing.T) {
			rig := newTypingRig(t, kind.field, "", false)
			rig.typeAtEnd("ab")
			rig.typeAtEnd("cd")
			rig.want("abcd", undoOnly)
			rig.undo()
			rig.want("ab", undoRedo)
			if start, end := rig.editor.Selection(); start != 2 || end != 2 {
				t.Errorf("the undo selects %d to %d, want the caret where the removed text was", start, end)
			}
			rig.redo()
			rig.want("abcd", undoNoRedo)
			rig.undo()
			rig.typeAtEnd("x")
			rig.want("abx", undoOnly)
			rig.redo()
			rig.want("abx", undoOnly)
			rig.undo()
			rig.want("ab", undoRedo)
			rig.redo()
			rig.want("abx", undoNoRedo)
		})
	}
}

// Once the typing is all reversed, Undo goes to the window. Redo then
// makes the window's change again first, and the typing after it.
func TestRedoGoesToTheWindowFirstAfterAnUndoOfTheWindow(t *testing.T) {
	for _, kind := range typingKinds {
		t.Run(kind.name, func(t *testing.T) {
			rig := newTypingRig(t, kind.field, "Draft", false)
			rig.typeAtEnd(" the")
			rig.undo()
			rig.want("Draft", redoOnly)
			rig.undo()
			rig.want("Draft", noTyping, false)
			rig.redo()
			rig.want("Draft", redoOnly, false, true)
			rig.redo()
			rig.want("Draft the", undoNoRedo, false, true)
			rig.redo()
			rig.want("Draft the", undoNoRedo, false, true, true)
		})
	}
}

// A submit that clears a field leaves it with no typing.
func TestASubmitLeavesNoTyping(t *testing.T) {
	for _, kind := range typingKinds[1:2] {
		t.Run(kind.name, func(t *testing.T) {
			rig := newTypingRig(t, kind.field, "", true)
			rig.typeAtEnd("Send this")
			rig.typeAtEnd("\n")
			rig.want("", noTyping)
			rig.undo()
			rig.want("", noTyping, false)
		})
	}
}

// A text that the program puts in the search field is no typing, and
// Escape, which the user presses to empty the field, is typing.
func TestTheSearchFieldsProgramTextIsNoTyping(t *testing.T) {
	rig := newTypingRig(t, typingKinds[2].field, "", false)
	rig.typeAtEnd("notes")
	rig.set("meeting")
	rig.frame()
	rig.want("meeting", noTyping)
	rig.r.Queue(key.Event{Name: key.NameEscape, State: key.Press})
	rig.frame()
	rig.want("", undoOnly)
	rig.undo()
	rig.want("meeting", redoOnly)
	rig.undo()
	rig.want("meeting", noTyping, false)
}

// The characters that the user types one after the other are one change:
// Cmd-Z after "abcd" typed key by key removes "abcd".
func TestARunOfTypingIsOneChange(t *testing.T) {
	for _, kind := range typingKinds {
		for _, seed := range []string{"", "Draft "} {
			t.Run(kind.name+"/"+seed, func(t *testing.T) {
				rig := newTypingRig(t, kind.field, seed, false)
				rig.moveTo(len(seed))
				rig.typeKeys("abcd")
				rig.want(seed+"abcd", undoOnly)
				rig.undo()
				rig.want(seed, redoOnly)
				rig.redo()
				rig.want(seed+"abcd", undoNoRedo)
			})
		}
	}
}

// A move of the insertion point starts a new change, also a move that
// comes back to where the typing stopped.
func TestAMoveOfTheInsertionPointStartsANewChange(t *testing.T) {
	for _, kind := range typingKinds {
		t.Run(kind.name, func(t *testing.T) {
			rig := newTypingRig(t, kind.field, "", false)
			rig.typeKeys("ab")
			rig.press(key.NameLeftArrow, 0)
			rig.typeKeys("x")
			rig.press(key.NameLeftArrow, 0)
			rig.press(key.NameRightArrow, 0)
			rig.typeKeys("yz")
			rig.want("axyzb", undoOnly)
			rig.undo()
			rig.want("axb", undoRedo)
			rig.undo()
			rig.want("ab", undoRedo)
			rig.undo()
			rig.want("", redoOnly)
		})
	}
}

// A paste is one change, and so is a cut. Typing after either starts a
// new change.
func TestAPasteAndACutAreOneChangeEach(t *testing.T) {
	for _, kind := range typingKinds {
		t.Run(kind.name, func(t *testing.T) {
			rig := newTypingRig(t, kind.field, "", false)
			rig.typeKeys("ab")
			rig.paste("PQ")
			rig.typeKeys("cd")
			rig.want("abPQcd", undoOnly)
			rig.r.Queue(key.SelectionEvent{Start: 2, End: 4})
			rig.press("X", key.ModShortcut)
			rig.typeKeys("e")
			rig.want("abecd", undoOnly)
			for _, want := range []string{"abcd", "abPQcd", "abPQ", "ab"} {
				rig.undo()
				if got := rig.editor.Text(); got != want {
					t.Fatalf("Undo leaves %q, want %q", got, want)
				}
			}
			rig.undo()
			rig.want("", redoOnly)
		})
	}
}

// A composed character joins the run: the input method replaces its marked
// text before the insertion point, and the commit is one more replace.
func TestAComposedCharacterJoinsTheRun(t *testing.T) {
	for _, kind := range typingKinds {
		t.Run(kind.name, func(t *testing.T) {
			rig := newTypingRig(t, kind.field, "", false)
			rig.typeKeys("ab")
			for _, text := range []string{"´", "é"} {
				rig.r.Queue(
					key.EditEvent{Range: key.Range{Start: 2, End: 2 + len([]rune(rig.editor.Text())) - 2}, Text: text},
					key.SelectionEvent{Start: 3, End: 3},
				)
				rig.frame()
			}
			rig.typeKeys("c")
			rig.want("abéc", undoOnly)
			rig.undo()
			rig.want("", redoOnly)
		})
	}
}

// A Delete key in a run joins the run, and a run that puts the text back
// as it was is no change.
func TestADeleteKeyJoinsTheRun(t *testing.T) {
	for _, kind := range typingKinds {
		t.Run(kind.name, func(t *testing.T) {
			rig := newTypingRig(t, kind.field, "", false)
			rig.typeKeys("abc")
			rig.press(key.NameDeleteBackward, 0)
			rig.typeKeys("d")
			rig.want("abd", undoOnly)
			rig.undo()
			rig.want("", redoOnly)
			rig.redo()
			rig.moveTo(3)
			rig.typeKeys("e")
			rig.press(key.NameDeleteBackward, 0)
			rig.want("abd", undoOnly)
			rig.undo()
			rig.want("", redoOnly)
		})
	}
}

// A field that loses the keyboard drops its typing, for Undo and for Redo.
// When it gets the keyboard again, Cmd-Z and Shift-Cmd-Z go to the window.
func TestAFieldThatLosesTheKeyboardDropsItsTyping(t *testing.T) {
	for _, kind := range typingKinds {
		t.Run(kind.name, func(t *testing.T) {
			rig := newTypingRig(t, kind.field, "", false)
			rig.typeKeys("abc")
			rig.want("abc", undoOnly)
			rig.focus(false)
			rig.want("abc", noTyping)
			rig.focus(true)
			rig.redo()
			rig.undo()
			rig.want("abc", noTyping, true, false)
			rig.typeKeys("d")
			rig.undo()
			rig.want("abc", redoOnly, true, false)
			rig.focus(false)
			rig.focus(true)
			rig.redo()
			rig.want("abc", noTyping, true, false, true)
		})
	}
}

// The search field's clear mark and Escape are one change each.
func TestTheSearchFieldsEscapeIsOneChange(t *testing.T) {
	rig := newTypingRig(t, typingKinds[2].field, "", false)
	rig.typeKeys("ab")
	rig.press(key.NameEscape, 0)
	rig.want("", undoOnly)
	rig.undo()
	rig.want("ab", undoRedo)
	rig.undo()
	rig.want("", redoOnly)
}

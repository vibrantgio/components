package input

import (
	"unicode/utf8"

	"gioui.org/io/key"
	"gioui.org/layout"
	"gioui.org/widget"
)

// typing is the user's typing in one field: each change the user made to
// the field's text, as Undo reverses it and Redo makes it again. The text
// the field was created with is no change. The field keeps its own steps,
// because widget.Editor records its seed as a change and does not tell
// whether it holds a change to reverse.
type typing struct {
	// text is the editor's text after the last step.
	text string
	// undo holds the text before each change Undo can reverse, the last
	// change last. redo holds the text after each reversed change, the
	// last reversed change last.
	undo, redo []string
	// reported is the state last given to the field's OnTyping.
	reported [2]bool
}

// newTyping is the typing of a field whose editor holds text.
func newTyping(text string) *typing { return &typing{text: text} }

// keys reads Cmd-Z and Shift-Cmd-Z, on macOS, or Ctrl-Z and Shift-Ctrl-Z,
// before editor reads them. The editor reads every press of them while it
// holds the keyboard. A press that the field holds a step for reverses or
// makes the step again and reports true. Each other press goes to leave,
// when leave is non-nil.
func (t *typing) keys(gtx layout.Context, editor *widget.Editor, leave func(gtx layout.Context, redo bool)) (changed bool) {
	for {
		e, ok := gtx.Event(key.Filter{Focus: editor, Name: "Z", Required: key.ModShortcut, Optional: key.ModShift})
		if !ok {
			return changed
		}
		ke, ok := e.(key.Event)
		if !ok || ke.State != key.Press {
			continue
		}
		redo := ke.Modifiers.Contain(key.ModShift)
		from, to := &t.undo, &t.redo
		if redo {
			from, to = &t.redo, &t.undo
		}
		if len(*from) == 0 {
			if leave != nil {
				leave(gtx, redo)
			}
			continue
		}
		next := (*from)[len(*from)-1]
		*from = (*from)[:len(*from)-1]
		*to = append(*to, t.text)
		restore(editor, t.text, next)
		t.text = next
		changed = true
	}
}

// changed records a change the editor made to its text. A change that
// leaves the text as it was is no step.
func (t *typing) changed(text string) {
	if text == t.text {
		return
	}
	t.undo = append(t.undo, t.text)
	t.redo = t.redo[:0]
	t.text = text
}

// state reports whether Undo can reverse a change, and whether Redo can
// make a reversed change again.
func (t *typing) state() (undo, redo bool) { return len(t.undo) > 0, len(t.redo) > 0 }

// report gives the state to on when it differs from the state last given.
func (t *typing) report(gtx layout.Context, on func(gtx layout.Context, undo, redo bool)) {
	undo, redo := t.state()
	if [2]bool{undo, redo} == t.reported {
		return
	}
	t.reported = [2]bool{undo, redo}
	if on != nil {
		on(gtx, undo, redo)
	}
}

// restore puts text into editor in place of from and selects the run of
// text that differs from from, as widget.Editor selects the text its own
// undo puts back.
func restore(editor *widget.Editor, from, text string) {
	pre := 0
	for pre < len(from) && pre < len(text) && from[pre] == text[pre] {
		pre++
	}
	for pre > 0 && pre < len(text) && !utf8.RuneStart(text[pre]) {
		pre--
	}
	suf := 0
	for suf < len(from)-pre && suf < len(text)-pre && from[len(from)-1-suf] == text[len(text)-1-suf] {
		suf++
	}
	for suf > 0 && !utf8.RuneStart(text[len(text)-suf]) {
		suf--
	}
	start := utf8.RuneCountInString(text[:pre])
	end := utf8.RuneCountInString(text[:len(text)-suf])
	editor.SetText(text)
	editor.SetCaret(end, start)
}

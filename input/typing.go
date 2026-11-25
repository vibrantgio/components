package input

import (
	"unicode/utf8"

	"gioui.org/io/key"
	"gioui.org/layout"
	"gioui.org/widget"
)

// Typing is what a text field's Undo and Redo keys act on while the field
// holds the keyboard. Every text field kind reports it through OnTyping.
type Typing struct {
	// Undo is true when Undo reverses a change the user made to the text.
	// Else the field gives Undo to the window.
	Undo bool
	// Redo is true when Redo makes a reversed change to the text again.
	Redo bool
	// RedoEnded is true when the user typed after the last Undo. Redo then
	// makes nothing again, in the field or in the window, until an Undo
	// reverses a change.
	RedoEnded bool
}

// typing is the user's typing in one field: each change the user made to
// the field's text, as Undo reverses it and Redo makes it again. The text
// the field was created with, and each text the program puts in the field,
// is no change. The field keeps its own steps, because widget.Editor
// records each text the program puts in it as a change and does not tell
// whether it holds a change to reverse.
//
// One step is one change the editor reports: a typed key, a paste, a cut,
// or a committed input method composition.
type typing struct {
	// text is the editor's text after the last step.
	text string
	// undo holds the text before each change Undo can reverse, the last
	// change last. redo holds the text after each reversed change, the
	// last reversed change last.
	undo, redo []string
	// left counts the Undo keys given to the window after the field's
	// typing was all reversed. Redo gives that many keys to the window
	// first, because Redo makes changes again in the opposite order of the
	// undos.
	left int
	// ended is true when the user typed after the last Undo.
	ended bool
	// reported is the state last given to the field's OnTyping.
	reported Typing
}

// newTyping is the typing of a field whose editor holds text.
func newTyping(text string) *typing { return &typing{text: text} }

// reset makes text the field's value with no typing, for a text the
// program puts in the field. Call it with the text given to the editor's
// SetText.
func (t *typing) reset(text string) {
	t.text, t.undo, t.redo, t.left, t.ended = text, nil, nil, 0, false
}

// keys reads Cmd-Z and Shift-Cmd-Z, on macOS, or Ctrl-Z and Shift-Ctrl-Z,
// before editor reads them. The editor reads every press of them while it
// holds the keyboard. Call keys before the frame's first editor.Update: a
// key that puts a step back makes the editor report a change in that
// Update. Each press that the field holds no step for and that Redo does
// not end goes to leave, when leave is non-nil.
func (t *typing) keys(gtx layout.Context, editor *widget.Editor, leave func(gtx layout.Context, redo bool)) {
	for {
		e, ok := gtx.Event(key.Filter{Focus: editor, Name: "Z", Required: key.ModShortcut, Optional: key.ModShift})
		if !ok {
			return
		}
		ke, ok := e.(key.Event)
		if !ok || ke.State != key.Press {
			continue
		}
		redo := ke.Modifiers.Contain(key.ModShift)
		switch {
		case redo && t.ended:
			continue
		case redo && t.left > 0:
			t.left--
		case redo && len(t.redo) > 0:
			t.step(editor, &t.redo, &t.undo)
			continue
		case !redo && len(t.undo) > 0:
			t.step(editor, &t.undo, &t.redo)
			continue
		case !redo:
			t.ended = false
			if len(t.redo) > 0 {
				t.left++
			}
		}
		if leave != nil {
			leave(gtx, redo)
		}
	}
}

// step puts the last text of from into editor and moves the editor's text
// onto to. An Undo of a step is not a new change, so it lets Redo act
// again.
func (t *typing) step(editor *widget.Editor, from, to *[]string) {
	next := (*from)[len(*from)-1]
	*from = (*from)[:len(*from)-1]
	*to = append(*to, t.text)
	restore(editor, t.text, next)
	t.text = next
	t.ended = false
}

// changed records a change the editor reported. A change that leaves the
// text as it was is no step. A step is a new change: it ends what Redo
// can make again.
func (t *typing) changed(text string) {
	if text == t.text {
		return
	}
	t.undo = append(t.undo, t.text)
	t.redo, t.left, t.ended = t.redo[:0], 0, true
	t.text = text
}

// state reports what Undo and Redo act on.
func (t *typing) state() Typing {
	return Typing{
		Undo:      len(t.undo) > 0,
		Redo:      !t.ended && t.left == 0 && len(t.redo) > 0,
		RedoEnded: t.ended,
	}
}

// report gives the state to on when it differs from the state last given.
func (t *typing) report(gtx layout.Context, on func(gtx layout.Context, typing Typing)) {
	s := t.state()
	if s == t.reported {
		return
	}
	t.reported = s
	if on != nil {
		on(gtx, s)
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

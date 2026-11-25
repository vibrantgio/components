package input

import (
	"io"
	"unicode/utf8"

	"gioui.org/io/key"
	"gioui.org/io/transfer"
	"gioui.org/layout"
	"gioui.org/widget"
)

// Typing is what a text field's Undo and Redo keys act on while the field
// holds the keyboard. Every text field kind reports it through OnTyping.
// A run of typed characters is one change, and a paste, a cut and a clear
// are one change each. A field that does not hold the keyboard holds no
// typing.
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
// The characters that the user types one after the other, with no move of
// the insertion point between them, are one step. A paste, a cut and a
// clear are one step each. The field holds its steps only while it holds
// the keyboard.
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
	// run is true while the last undo step is a run of typed characters
	// that the next typed character joins. at is the insertion point after
	// the run.
	run bool
	at  int
	// start and end are the editor's selection before the next change.
	start, end int
	// reported is the state last given to the field's OnTyping.
	reported Typing
}

// newTyping is the typing of a field whose editor holds text.
func newTyping(text string) *typing { return &typing{text: text} }

// reset makes text the field's value with no typing, for a text the
// program puts in the field and for a field that loses the keyboard. Call
// it with the text given to the editor's SetText.
func (t *typing) reset(text string) {
	t.text, t.undo, t.redo, t.left, t.ended, t.run = text, nil, nil, 0, false, false
}

// keys reads Cmd-Z and Shift-Cmd-Z, on macOS, or Ctrl-Z and Shift-Ctrl-Z,
// before editor reads them. The editor reads every press of them while it
// holds the keyboard. Call keys before the frame's first editor.Update: a
// key that puts a step back makes the editor report a change in that
// Update. Each press that the field holds no step for and that Redo does
// not end goes to leave, when leave is non-nil.
//
// A field that does not hold the keyboard drops its typing, so it holds
// no typing when it gets the keyboard again. A move of the insertion
// point ends the run of typed characters.
func (t *typing) keys(gtx layout.Context, editor *widget.Editor, leave func(gtx layout.Context, redo bool)) {
	if !gtx.Focused(editor) && (len(t.undo) > 0 || len(t.redo) > 0 || t.left > 0 || t.ended || t.run) {
		t.reset(editor.Text())
	}
	t.start, t.end = selection(editor)
	if t.start != t.end || t.start != t.at {
		t.run = false
	}
	t.paste(gtx, editor)
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

// paste takes the text that a paste or a drop gives the editor and puts it
// in, as the editor does, as one change by itself. The editor reports the
// change in its next Update. A paste looks like typed characters in the
// editor's text, so the field takes it before the editor does.
func (t *typing) paste(gtx layout.Context, editor *widget.Editor) {
	for {
		e, ok := gtx.Event(transfer.TargetFilter{Target: editor, Type: "application/text"})
		if !ok {
			return
		}
		de, ok := e.(transfer.DataEvent)
		if !ok {
			continue
		}
		r := de.Open()
		content, err := io.ReadAll(r)
		r.Close()
		if err != nil {
			continue
		}
		editor.Insert(string(content))
		t.alone(editor)
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
	t.ended, t.run = false, false
}

// changed records a change the editor reported. A typed character joins
// the run before it. Each other change is one step by itself.
func (t *typing) changed(editor *widget.Editor) { t.record(editor, true) }

// alone records the editor's text as one change by itself, for a paste and
// a clear. A typed character after it starts a new change.
func (t *typing) alone(editor *widget.Editor) { t.record(editor, false) }

// record records the editor's text as a change. A change that leaves the
// text as it was is no step. A step is a new change: it ends what Redo
// can make again.
//
// A change is typed when it keeps the text after the selection before it,
// and leaves the insertion point after the text it put in. A typed
// character, a Delete key and an input method composition each give such
// a change. A typed change at the insertion point after the run joins the
// run. DERIVED: a Delete key in a run and a committed composition join the
// run, because no typing capture measures how the platform groups them.
//
// A change at a selection is a new step. When it removes the selection
// and puts no text in, it is a cut or a Delete of the selection, and it is
// one change by itself. DERIVED: a Delete of a selection is one change, as
// a cut is.
func (t *typing) record(editor *widget.Editor, typedOK bool) {
	text := editor.Text()
	lo, hi := t.start, t.end
	t.start, t.end = selection(editor)
	if text == t.text {
		return
	}
	typed := typedOK && t.start == t.end && (lo == hi || t.end > lo) && keepsAfter(t.text, text, hi, t.end)
	if !typed || !t.run || lo != hi || lo != t.at {
		t.undo = append(t.undo, t.text)
	}
	t.redo, t.left, t.ended = t.redo[:0], 0, true
	t.text, t.run, t.at = text, typed, t.end
	if n := len(t.undo); t.undo[n-1] == text {
		// A run that puts the text back as it was before the run is no
		// change. A field with no step left holds no typing, so Undo and
		// Redo go to the window.
		t.undo, t.run = t.undo[:n-1], false
		t.ended = t.ended && len(t.undo) > 0
	}
}

// keepsAfter is true when text ends with the runes of was from rune offset
// hi on, and those runes start at rune offset at in text.
func keepsAfter(was, text string, hi, at int) bool {
	w, n := []rune(was), []rune(text)
	if hi > len(w) {
		return false
	}
	rest := w[hi:]
	return at == len(n)-len(rest) && at >= 0 && string(n[at:]) == string(rest)
}

// selection is the editor's selection as rune offsets, the lower offset
// first.
func selection(editor *widget.Editor) (start, end int) {
	start, end = editor.Selection()
	if start > end {
		start, end = end, start
	}
	return start, end
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

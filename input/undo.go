package input

import (
	"maps"

	"gioui.org/layout"
)

// windowUndoKey is the key the window's Undo is set under in
// layout.Context.Values. This file is the one place that names it.
const windowUndoKey = "github.com/vibrantgio/components/input.WindowUndo"

// WindowUndo is the window's Undo and Redo. A text field that holds the
// keyboard gives it each Cmd-Z and Shift-Cmd-Z that the field holds no
// typing for: Ctrl-Z and Shift-Ctrl-Z away from macOS. redo is true for
// Redo. WindowUndo reports true when the Undo reversed a change, or when
// the Redo made a change again. It reports false when it found no such
// change, and then it changes nothing.
type WindowUndo func(gtx layout.Context, redo bool) bool

// WithWindowUndo returns gtx with undo set as the window's Undo and Redo.
// A window that keeps its own changes sets it before it lays out what it
// holds, and each text field under gtx gives undo the keys that it holds
// no typing for. A field under a gtx with no window Undo gives those keys
// to no one: the window then holds no change to reverse.
//
// The field reads the keys before the window does, because Gio gives a key
// event to the first reader whose filter matches. widget.Editor reads both
// keys while it holds the keyboard, so the field reads them first and
// passes on each key that it does not act on. gtx.Values is cloned and not
// written, so the caller's context keeps what it had.
func WithWindowUndo(gtx layout.Context, undo WindowUndo) layout.Context {
	values := maps.Clone(gtx.Values)
	if values == nil {
		values = make(map[string]any, 1)
	}
	values[windowUndoKey] = undo
	gtx.Values = values
	return gtx
}

// windowUndo gives the window's Undo, or its Redo when redo is true, to the
// window that gtx carries. It reports whether the window reversed a change,
// or made one again. A gtx with no window Undo reports false.
func windowUndo(gtx layout.Context, redo bool) bool {
	undo, ok := gtx.Values[windowUndoKey].(WindowUndo)
	return ok && undo != nil && undo(gtx, redo)
}

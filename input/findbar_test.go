package input_test

import (
	"testing"

	"gioui.org/io/event"
	gioinput "gioui.org/io/input"
	"gioui.org/io/key"
	"gioui.org/layout"
	"gioui.org/op"

	"github.com/reactivego/rx"
	"github.com/vibrantgio/components/input"
	"github.com/vibrantgio/theme/theme"
)

// findBarRun lays a search field of one variant out with "Gio" typed into
// it, inside a handler that asks for Escape after the field, and keeps the
// counts the tests read.
type findBarRun struct {
	changes         []string
	next, previous  int
	escapesReceived int
	tag             event.Tag
	w               layout.Widget
	r               *gioinput.Router
	ops             *op.Ops
}

func newFindBarRun(t *testing.T, v input.Variant) *findBarRun {
	t.Helper()
	run := &findBarRun{r: new(gioinput.Router), ops: new(op.Ops)}
	field := materialize(t, input.SearchField(rx.Of(theme.Default()), input.SearchFieldProps{
		Placeholder: "Find",
		Variant:     v,
		Shaper:      defaultShaper(t),
		FocusTag:    func(t event.Tag) { run.tag = t },
		OnChange:    func(_ layout.Context, s string) { run.changes = append(run.changes, s) },
		OnNext:      func(layout.Context) { run.next++ },
		OnPrevious:  func(layout.Context) { run.previous++ },
	}))
	run.w = func(gtx layout.Context) layout.Dimensions {
		dims := field(gtx)
		for {
			ev, ok := gtx.Event(key.Filter{Name: key.NameEscape})
			if !ok {
				break
			}
			if ke, isKey := ev.(key.Event); isKey && ke.State == key.Press {
				run.escapesReceived++
			}
		}
		return dims
	}
	focusSearchField(run.w, run.ops, run.r)
	if !run.r.Source().Focused(run.tag) {
		t.Fatal("pressing in the field did not give it the keyboard")
	}
	run.r.Queue(key.EditEvent{Text: "Gio"})
	run.frames(1)
	if n := len(run.changes); n == 0 || run.changes[n-1] != "Gio" {
		t.Fatalf("typing into the field posted %q, want \"Gio\" last", run.changes)
	}
	return run
}

func (run *findBarRun) frames(n int) {
	for range n {
		driveTextFieldFrame(run.w, run.ops, run.r, escapeTestSize)
	}
}

func (run *findBarRun) last() string { return run.changes[len(run.changes)-1] }

// TestFindBarFieldDoesNotTakeEscape presses Escape in the find bar field
// with text typed: the field keeps its text, and the handler around it
// receives the key, which is how the find closes at once.
func TestFindBarFieldDoesNotTakeEscape(t *testing.T) {
	run := newFindBarRun(t, input.FindBar)
	before := len(run.changes)
	run.r.Queue(key.Event{Name: key.NameEscape, State: key.Press})
	run.frames(2)
	if got := run.changes[before:]; len(got) != 0 {
		t.Errorf("Escape in the find bar field posted %q; the field does not empty on Escape", got)
	}
	if run.last() != "Gio" {
		t.Errorf("the field holds %q after Escape, want \"Gio\"", run.last())
	}
	if run.escapesReceived != 1 {
		t.Errorf("the handler around the field received Escape %d times, want 1: the find bar field does not take the key", run.escapesReceived)
	}
}

// TestFindBarFieldReportsEnterAndShiftEnter presses Return, Shift+Return and
// Enter in the find bar field: each one is reported to the caller, and the
// editor does not take any of them, so the text stays "Gio" on one line.
func TestFindBarFieldReportsEnterAndShiftEnter(t *testing.T) {
	run := newFindBarRun(t, input.FindBar)
	before := len(run.changes)

	run.r.Queue(key.Event{Name: key.NameReturn, State: key.Press})
	run.frames(2)
	if run.next != 1 || run.previous != 0 {
		t.Fatalf("Return reported next %d and previous %d times, want 1 and 0", run.next, run.previous)
	}
	run.r.Queue(key.Event{Name: key.NameReturn, Modifiers: key.ModShift, State: key.Press})
	run.frames(2)
	if run.next != 1 || run.previous != 1 {
		t.Fatalf("Shift+Return reported next %d and previous %d times, want 1 and 1", run.next, run.previous)
	}
	run.r.Queue(key.Event{Name: key.NameEnter, State: key.Press})
	run.frames(2)
	if run.next != 2 {
		t.Errorf("Enter reported next %d times in all, want 2", run.next)
	}
	if got := run.changes[before:]; len(got) != 0 {
		t.Errorf("the keys changed the field's text to %q; the editor does not see them", got)
	}
	if !run.r.Source().Focused(run.tag) {
		t.Error("the keyboard left the field; stepping keeps it in the field")
	}
}

// TestChromeFieldKeepsItsEscape presses Escape in the chrome field with text
// typed: it empties the field and keeps the key, as the platform's field on
// chrome does. Enter is not reported there.
func TestChromeFieldKeepsItsEscape(t *testing.T) {
	run := newFindBarRun(t, input.Chrome)
	run.r.Queue(key.Event{Name: key.NameReturn, State: key.Press})
	run.frames(2)
	if run.next != 0 {
		t.Errorf("Return in the chrome field reported next %d times; only the find bar field reports it", run.next)
	}
	before := len(run.changes)
	run.r.Queue(key.Event{Name: key.NameEscape, State: key.Press})
	run.frames(2)
	if got := run.changes[before:]; len(got) != 1 || got[0] != "" {
		t.Errorf("Escape in the chrome field posted %q; it empties the field once", got)
	}
	if run.escapesReceived != 0 {
		t.Errorf("the handler around the chrome field received Escape %d times; the field takes it while it holds text", run.escapesReceived)
	}
}

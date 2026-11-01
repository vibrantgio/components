package inventory

import (
	"context"
	"image"
	"image/color"
	"slices"
	"sync"
	"testing"
	"time"

	"gioui.org/f32"
	"gioui.org/io/event"
	gioinput "gioui.org/io/input"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"

	"github.com/reactivego/rx"

	"github.com/vibrantgio/components/button"
	"github.com/vibrantgio/components/golden"
	"github.com/vibrantgio/components/input"
	"github.com/vibrantgio/markdown"
	"github.com/vibrantgio/patterns/findbar"
	"github.com/vibrantgio/theme/tokens"
)

// The live window's measures, at one pixel per dp: a toolbar band with the
// search field, the list column under it, and the content column beside the
// list.
var searchWindow = image.Pt(800, 400)

const (
	searchToolbarH = 52
	searchListW    = 220
	searchFieldW   = 200
)

// searchWiring wires a search over a collection as an application does: the
// toolbar search field filters the list column, the find button stands at
// the content column's foot while the field holds a query, and the find
// button and Cmd-F open one find bar with the field's query. The content
// marks the matches of the bar's query while the bar is open.
type searchWiring struct {
	t    *testing.T
	c    tokens.PlatformColors
	r    *gioinput.Router
	ops  *op.Ops
	doc  *markdown.Document
	text *int

	field  layout.Widget
	clear  func()
	toEdit event.Tag
	find   layout.Widget

	// query is the toolbar field's text.
	query string
	// rows are the list column's rows on the last frame.
	rows []string
	// findStood is whether the find button was laid out on the last frame.
	findStood bool

	send     rx.Observer[findbar.State]
	state    findbar.State
	barField event.Tag
	barQuery string

	mu     sync.Mutex
	bar    layout.Widget
	barGen int
}

func newSearchWiring(t *testing.T) *searchWiring {
	t.Helper()
	c := tokens.PlatformLight
	shaper := tokens.DefaultTypography.DeterministicShaper()
	th := rx.Of(golden.StaticTheme(c))
	w := &searchWiring{
		t: t, c: c, r: new(gioinput.Router), ops: new(op.Ops), text: new(int),
		doc: markdown.NewDocument(markdown.Parse([]byte(CollectionSearchText))),
	}
	fill := func(p tokens.PlatformColors) color.NRGBA { return p.TextBackground }

	w.field = materializeWidget(t, input.SearchField(th, input.SearchFieldProps{
		Placeholder: "Search",
		Variant:     input.Chrome,
		Region:      input.Toolbar,
		Surface:     fill,
		Shaper:      shaper,
		FocusTag:    func(tag event.Tag) { w.toEdit = tag },
		Clear:       func(clear func()) { w.clear = clear },
		OnChange:    func(_ layout.Context, q string) { w.query = q },
	}))
	w.find = materializeWidget(t, button.FindButton(th, button.FindButtonProps{
		Surface: fill,
		Shaper:  shaper,
		OnClick: func(layout.Context) { w.openBar() },
	}))

	send, open := rx.Subject[findbar.State](0, 1)
	w.send = send
	sub := findbar.FindBar(th, findbar.Props{
		Open:     open,
		Fill:     fill,
		Content:  func() event.Tag { return w.text },
		OnClose:  func(layout.Context) { w.closeBar() },
		OnChange: func(_ layout.Context, q string) { w.barQuery = q },
		FocusTag: func(tag event.Tag) { w.barField = tag },
	}).Subscribe(rx.GoroutineContext(), func(b layout.Widget, err error, done bool) {
		if !done && err == nil && b != nil {
			w.mu.Lock()
			w.bar = b
			w.barGen++
			w.mu.Unlock()
		}
	})
	t.Cleanup(sub.Unsubscribe)
	w.set(findbar.State{})
	return w
}

// materializeWidget subscribes to a live control whose theme completes, and
// answers the layout.Widget it emitted.
func materializeWidget(t *testing.T, obs rx.Observable[layout.Widget]) layout.Widget {
	t.Helper()
	var w layout.Widget
	if err := obs.Subscribe(context.Background(), func(next layout.Widget, _ error, done bool) {
		if !done && next != nil {
			w = next
		}
	}).Wait(); err != nil {
		t.Fatalf("subscribe: %v", err)
	}
	if w == nil {
		t.Fatal("the control did not emit a layout.Widget")
	}
	return w
}

// openBar is what the find button and Cmd-F do: open the bar with the
// toolbar field's query.
func (w *searchWiring) openBar() {
	w.state = findbar.State{Open: true, Opens: w.state.Opens + 1, Query: w.query}
	w.send.Next(w.state)
}

func (w *searchWiring) closeBar() {
	w.state = findbar.State{Opens: w.state.Opens}
	w.send.Next(w.state)
}

// set hands the bar a state and waits for the bar to emit for it.
func (w *searchWiring) set(st findbar.State) {
	w.t.Helper()
	w.mu.Lock()
	before := w.barGen
	w.mu.Unlock()
	w.state = st
	w.send.Next(st)
	w.waitBar(before)
}

func (w *searchWiring) waitBar(before int) {
	w.t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for {
		w.mu.Lock()
		gen := w.barGen
		w.mu.Unlock()
		if gen > before {
			break
		}
		if time.Now().After(deadline) {
			w.t.Fatal("the bar did not emit for the state")
		}
		time.Sleep(5 * time.Millisecond)
	}
	time.Sleep(20 * time.Millisecond)
	w.frame()
	w.frame()
}

// layout is the window: the toolbar band with the field at its trailing
// end, the list column, and the content column with the bar over it and the
// find button at its foot.
func (w *searchWiring) layout(gtx layout.Context) layout.Dimensions {
	size := gtx.Constraints.Max
	fill := w.c.TextBackground
	paint.FillShape(gtx.Ops, fill, clip.Rect{Max: size}.Op())

	// Cmd-F opens the bar wherever the keyboard is.
	for {
		ev, ok := gtx.Event(key.Filter{Name: "F", Required: key.ModShortcut})
		if !ok {
			break
		}
		if ke, isKey := ev.(key.Event); isKey && ke.State == key.Press {
			w.openBar()
		}
	}

	w.mu.Lock()
	bar := w.bar
	w.mu.Unlock()

	// The content marks the bar's query while the bar is open, and
	// nothing before it opens.
	if w.state.Open {
		w.doc.Find(w.barQuery, 0)
	} else {
		w.doc.ClearFind()
	}

	ftgx := gtx
	ftgx.Constraints = layout.Exact(image.Pt(searchFieldW, searchToolbarH))
	off := op.Offset(image.Pt(size.X-searchFieldW-8, 8)).Push(gtx.Ops)
	w.field(ftgx)
	off.Pop()

	w.rows = CollectionSearchMatches(w.query)

	content := func(gtx layout.Context) layout.Dimensions {
		csize := gtx.Constraints.Max
		defer clip.Rect{Max: csize}.Push(gtx.Ops).Pop()
		event.Op(gtx.Ops, w.text)
		for {
			if _, ok := gtx.Event(key.FocusFilter{Target: w.text}); !ok {
				break
			}
		}
		gtx.Constraints.Min = image.Point{}
		gtx.Constraints.Max.Y = 1 << 16
		w.doc.LayoutColumn(gtx, tokens.DefaultTypography.DeterministicShaper(), markdown.FromTokens(w.c, tokens.DefaultTypography, fill))
		return layout.Dimensions{Size: csize}
	}
	var find layout.Widget
	w.findStood = false
	if w.query != "" {
		find = func(gtx layout.Context) layout.Dimensions {
			w.findStood = true
			return w.find(gtx)
		}
	}
	cgtx := gtx
	cgtx.Constraints = layout.Exact(image.Pt(size.X-searchListW, size.Y-searchToolbarH))
	off = op.Offset(image.Pt(searchListW, searchToolbarH)).Push(gtx.Ops)
	findbar.Over(bar, button.FindButtonOver(content, nil, find))(cgtx)
	off.Pop()
	return layout.Dimensions{Size: size}
}

func (w *searchWiring) gtx() layout.Context {
	return layout.Context{
		Constraints: layout.Exact(searchWindow),
		Metric:      unit.Metric{PxPerDp: 1, PxPerSp: 1},
		Ops:         w.ops,
		Source:      w.r.Source(),
	}
}

func (w *searchWiring) frame() {
	w.ops.Reset()
	w.layout(w.gtx())
	w.r.Frame(w.ops)
}

// settle lays out frames until the bar has acted on what the wiring sent it.
func (w *searchWiring) settle() {
	w.t.Helper()
	w.mu.Lock()
	before := w.barGen
	w.mu.Unlock()
	w.frame()
	w.mu.Lock()
	moved := w.barGen > before
	w.mu.Unlock()
	if moved {
		w.waitBar(before)
		return
	}
	// A request the bar has not answered yet emits on its goroutine.
	deadline := time.Now().Add(200 * time.Millisecond)
	for time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
		w.mu.Lock()
		moved = w.barGen > before
		w.mu.Unlock()
		if moved {
			w.waitBar(before)
			return
		}
	}
	w.frame()
}

// typeQuery gives the toolbar field the keyboard and types q into it.
func (w *searchWiring) typeQuery(q string) {
	w.t.Helper()
	w.r.Source().Execute(key.FocusCmd{Tag: w.toEdit})
	w.frame()
	w.r.Queue(key.EditEvent{Text: q})
	w.frame()
	w.frame()
	if w.query != q {
		w.t.Fatalf("the toolbar field reports %q after typing, want %q", w.query, q)
	}
}

func (w *searchWiring) press(name key.Name, mods key.Modifiers) {
	w.r.Queue(key.Event{Name: name, Modifiers: mods, State: key.Press})
	w.settle()
}

// pressFind clicks the middle of the find button: centred on the content
// column, its 24 rows ending 35 above the column's foot.
func (w *searchWiring) pressFind() {
	w.t.Helper()
	at := f32.Pt(float32(searchListW+(searchWindow.X-searchListW)/2), float32(searchWindow.Y-35-12))
	w.r.Queue(
		pointer.Event{Kind: pointer.Press, Position: at, Source: pointer.Mouse, Buttons: pointer.ButtonPrimary},
		pointer.Event{Kind: pointer.Release, Position: at, Source: pointer.Mouse},
	)
	w.settle()
}

// barEditor answers the find bar's field.
func (w *searchWiring) barEditor() *widget.Editor {
	w.t.Helper()
	ed, ok := w.barField.(*widget.Editor)
	if !ok {
		w.t.Fatalf("the bar's focus tag is %T, want its editor", w.barField)
	}
	return ed
}

// marked counts the pixels of the light highlight fill in the window.
func (w *searchWiring) marked() int {
	w.t.Helper()
	img := golden.Capture(w.t, searchWindow, w.layout)
	n := 0
	b := img.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			r, g, bl, _ := img.At(x, y).RGBA()
			p := w.c.HighlightFill
			if uint8(r>>8) == p.R && uint8(g>>8) == p.G && uint8(bl>>8) == p.B {
				n++
			}
		}
	}
	return n
}

// checkOpen holds the bar to its open state: the bar's field holds the
// toolbar field's query selected whole, and the content marks every match.
func (w *searchWiring) checkOpen(how string) {
	w.t.Helper()
	if !w.state.Open {
		w.t.Fatalf("%s did not open the find bar", how)
	}
	if got := w.barEditor().SelectedText(); got != w.query {
		w.t.Errorf("%s opened the bar with %q selected, want the whole query %q", how, got, w.query)
	}
	if !w.r.Source().Focused(w.barField) {
		w.t.Errorf("%s opened the bar without the keyboard in its field", how)
	}
	if n := len(w.doc.Matches()); n != 3 {
		w.t.Errorf("%s: the content marks %d matches, want the 3 the text holds", how, n)
	}
	if w.marked() == 0 {
		w.t.Errorf("%s: no pixel of the content is the highlight fill", how)
	}
}

// TestTheFindButtonStandsWhileTheFieldHoldsAQuery types a query into the
// toolbar field and empties it again: the find button stands while the field
// holds the query, the bar open or not, and leaves when the field is emptied.
// The list column shows only the rows that hold the query.
func TestTheFindButtonStandsWhileTheFieldHoldsAQuery(t *testing.T) {
	w := newSearchWiring(t)
	w.frame()
	if w.findStood {
		t.Fatal("the find button stands while the toolbar field is empty")
	}
	if !slices.Equal(w.rows, CollectionSearchRows) {
		t.Fatalf("the list column shows %q with an empty field, want every row", w.rows)
	}

	w.typeQuery(CollectionSearchQuery)
	if !w.findStood {
		t.Fatal("the toolbar field holds a query and the find button does not stand")
	}
	want := []string{"Blue hour by the river", "Bluebird notes"}
	if !slices.Equal(w.rows, want) {
		t.Errorf("the list column shows %q, want only the rows that match, %q", w.rows, want)
	}

	w.pressFind()
	if !w.state.Open {
		t.Fatal("the find button did not open the find bar")
	}
	w.frame()
	if !w.findStood {
		t.Error("the find bar is open and the find button left")
	}

	w.clear()
	w.frame()
	w.frame()
	if w.query != "" {
		t.Fatalf("the clear mark left %q in the toolbar field", w.query)
	}
	if w.findStood {
		t.Error("the toolbar field is empty and the find button still stands")
	}
	if !slices.Equal(w.rows, CollectionSearchRows) {
		t.Errorf("the list column shows %q after the field is emptied, want every row", w.rows)
	}
}

// TestTheFindButtonAndCmdFOpenTheSameBar opens the bar with the find button
// and then with Cmd-F. Both open the one bar with the toolbar field's query
// selected whole and every match marked. No match is marked before the bar
// opens.
func TestTheFindButtonAndCmdFOpenTheSameBar(t *testing.T) {
	w := newSearchWiring(t)
	w.typeQuery(CollectionSearchQuery)
	if n := len(w.doc.Matches()); n != 0 {
		t.Errorf("the toolbar field holds a query and the content marks %d matches before the bar opens", n)
	}
	if n := w.marked(); n != 0 {
		t.Errorf("%d pixels of the content are the highlight fill before the bar opens", n)
	}

	w.pressFind()
	w.checkOpen("the find button")
	bar := w.barField

	w.press(key.NameEscape, 0)
	if w.state.Open {
		t.Fatal("Escape did not close the find bar")
	}
	if n := len(w.doc.Matches()); n != 0 {
		t.Errorf("the bar is closed and the content marks %d matches", n)
	}

	w.press("F", key.ModShortcut)
	w.checkOpen("Cmd-F")
	if w.barField != bar {
		t.Error("Cmd-F opened a find bar other than the one the find button opens")
	}
}

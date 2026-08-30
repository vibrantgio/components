package picker_test

import (
	"context"
	"image"
	"testing"

	"gioui.org/f32"
	gioinput "gioui.org/io/input"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"

	"github.com/reactivego/rx"
	"github.com/vibrantgio/components/picker"
	"github.com/vibrantgio/theme/theme"
	"github.com/vibrantgio/theme/tokens"
)

// materialize subscribes to a component observable and returns the widget it
// emitted. A static theme emits once synchronously, so the subscription is
// finished by the time Wait returns.
func materialize(t *testing.T, obs rx.Observable[layout.Widget]) layout.Widget {
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
		t.Fatal("component emitted no widget")
	}
	return w
}

// driver lays a widget out against a router, one frame per call, and hands
// back the dimensions it reported. Clicked has to run inside the frame that
// processes the events, which is why every assertion below drives a frame
// rather than querying the widget.
func driver(w layout.Widget, r *gioinput.Router, size image.Point) func() layout.Dimensions {
	ops := new(op.Ops)
	return func() layout.Dimensions {
		ops.Reset()
		gtx := layout.Context{
			Metric:      unit.Metric{PxPerDp: 1, PxPerSp: 1},
			Constraints: layout.Exact(size),
			Ops:         ops,
			Source:      r.Source(),
		}
		dims := w(gtx)
		r.Frame(ops)
		return dims
	}
}

// click queues a press and a release at pos and drives one frame.
func click(r *gioinput.Router, drive func() layout.Dimensions, pos f32.Point) layout.Dimensions {
	r.Queue(
		pointer.Event{Kind: pointer.Press, Position: pos, Buttons: pointer.ButtonPrimary, Source: pointer.Mouse},
		pointer.Event{Kind: pointer.Release, Position: pos, Buttons: pointer.ButtonPrimary, Source: pointer.Mouse},
	)
	return drive()
}

// liveTheme is the default theme with sharp corners and a pinned shaper, so a
// live widget measures the same on every machine.
func liveTheme() theme.Theme {
	th := theme.Default()
	th.Radius = rx.Of(tokens.RadiusScale{})
	return th
}

// TestFieldOpensItsMenuAndSelectsFromIt walks the form register end to end:
// the trigger toggles the menu, a row picks a value, and picking closes the
// menu again — the whole of what a select does, driven through the Gio event
// system rather than asserted off a render state.
func TestFieldOpensItsMenuAndSelectsFromIt(t *testing.T) {
	var picked []int
	w := materialize(t, picker.Field(rx.Of(liveTheme()), picker.FieldProps{
		Description: "choose",
		Options:     options,
		Shaper:      defaultShaper(t),
		OnSelect:    func(_ layout.Context, i int) { picked = append(picked, i) },
	}))

	r := new(gioinput.Router)
	drive := driver(w, r, image.Pt(200, 400))
	row := rowHeight(tokens.Comfortable)

	if dims := drive(); dims.Size.Y != row {
		t.Fatalf("closed field measured %d px tall, want the trigger's %d px", dims.Size.Y, row)
	}

	// The trigger's own band, well inside it on both axes.
	dims := click(r, drive, f32.Pt(100, float32(row)/2))
	if want := row * (1 + len(options)); dims.Size.Y != want {
		t.Fatalf("after clicking the trigger the field measured %d px tall, want the open %d px", dims.Size.Y, want)
	}

	// The second option's row: rows stack directly under the trigger, each one
	// row tall, so row index i occupies [ (1+i)*row, (2+i)*row ).
	dims = click(r, drive, f32.Pt(100, float32(2*row)+float32(row)/2))
	if len(picked) != 1 || picked[0] != 1 {
		t.Fatalf("OnSelect fired with %v, want exactly one call carrying index 1", picked)
	}
	if dims.Size.Y != row {
		t.Errorf("after picking, the field measured %d px tall, want the closed %d px: choosing closes the menu", dims.Size.Y, row)
	}
}

// TestUpwardFieldSelectsFromTheMenuAboveItsTrigger walks the same path with
// the menu on the other side: opening moves the trigger to the BOTTOM of the
// box, the rows take the space above it, and a click lands on the option that
// is drawn where it was clicked. The direction is a placement and the field
// stays one widget — nothing about picking changes with it.
func TestUpwardFieldSelectsFromTheMenuAboveItsTrigger(t *testing.T) {
	var picked []int
	w := materialize(t, picker.Field(rx.Of(liveTheme()), picker.FieldProps{
		Description: "choose",
		Options:     options,
		Drop:        picker.DropUp,
		Shaper:      defaultShaper(t),
		OnSelect:    func(_ layout.Context, i int) { picked = append(picked, i) },
	}))

	r := new(gioinput.Router)
	drive := driver(w, r, image.Pt(200, 400))
	row := rowHeight(tokens.Comfortable)

	if dims := drive(); dims.Size.Y != row {
		t.Fatalf("closed upward field measured %d px tall, want the trigger's %d px", dims.Size.Y, row)
	}

	// Closed, the trigger is the whole widget and stands at the top.
	dims := click(r, drive, f32.Pt(100, float32(row)/2))
	if want := row * (1 + len(options)); dims.Size.Y != want {
		t.Fatalf("after clicking the trigger the field measured %d px tall, want the open %d px", dims.Size.Y, want)
	}

	// Open, the rows are above: index i occupies [ i*row, (i+1)*row ) and the
	// trigger has moved down to the last band.
	dims = click(r, drive, f32.Pt(100, float32(row)+float32(row)/2))
	if len(picked) != 1 || picked[0] != 1 {
		t.Fatalf("OnSelect fired with %v, want exactly one call carrying index 1", picked)
	}
	if dims.Size.Y != row {
		t.Errorf("after picking, the field measured %d px tall, want the closed %d px: choosing closes the menu", dims.Size.Y, row)
	}
}

// TestFieldTriggerHitsTheFloorBelowItsBar is the pointer-target contract for
// the form register: the widget measures the bar it drew, while what the
// pointer may land on is the density's 44 dp floor centred on it. The click
// below is outside the drawn bar and inside the slop, which is the only place
// the two can be told apart.
func TestFieldTriggerHitsTheFloorBelowItsBar(t *testing.T) {
	w := materialize(t, picker.Field(rx.Of(liveTheme()), picker.FieldProps{
		Options: options,
		Shaper:  defaultShaper(t),
	}))

	r := new(gioinput.Router)
	drive := driver(w, r, image.Pt(200, 400))
	row := rowHeight(tokens.Comfortable)
	drive() // register the input area

	// The hit rect is 44 px centred on the 40 px bar: −2..42 on the y axis.
	dims := click(r, drive, f32.Pt(100, float32(row)+1.5))
	if want := row * (1 + len(options)); dims.Size.Y != want {
		t.Errorf("a click in the slop below the trigger left the field %d px tall, want the open %d px", dims.Size.Y, want)
	}
}

// TestMenuSelectsWithoutATriggerOfItsOwn: the open surface is a component in
// its own right, which is what lets an anchor's caller hand it to a popover.
// It reports the rows it drew and dispatches the index that was clicked.
func TestMenuSelectsWithoutATriggerOfItsOwn(t *testing.T) {
	var picked []int
	w := materialize(t, picker.Menu(rx.Of(liveTheme()), picker.MenuProps{
		Options:  options,
		Shaper:   defaultShaper(t),
		OnSelect: func(_ layout.Context, i int) { picked = append(picked, i) },
	}))

	r := new(gioinput.Router)
	drive := driver(w, r, image.Pt(200, 400))
	row := rowHeight(tokens.Comfortable)

	if dims := drive(); dims.Size.Y != row*len(options) {
		t.Fatalf("menu measured %d px tall, want %d rows of %d px", dims.Size.Y, len(options), row)
	}
	click(r, drive, f32.Pt(100, float32(2*row)+float32(row)/2))
	if len(picked) != 1 || picked[0] != 2 {
		t.Fatalf("OnSelect fired with %v, want exactly one call carrying index 2", picked)
	}
}

// TestAnchorActivatesAndKeepsItsPointerFloor: the chrome register's trigger
// reports the control it drew — it is sized to its value, and a row of chrome
// laid out at 44 dp apiece would be a row of gaps — while the pointer floor is
// centred on it, exactly as components/button and the chip family extend
// theirs.
func TestAnchorActivatesAndKeepsItsPointerFloor(t *testing.T) {
	var clicks int
	w := materialize(t, picker.Anchor(rx.Of(liveTheme()), picker.AnchorProps{
		Value:   "Anthropic · Opus 5",
		Shaper:  defaultShaper(t),
		OnClick: func(_ layout.Context) { clicks++ },
	}))

	r := new(gioinput.Router)
	drive := driver(w, r, image.Pt(400, 120))
	dims := drive()
	if want := int(tokens.Comfortable.ControlHeight); dims.Size.Y != want {
		t.Fatalf("anchor measured %d px tall, want the density's control height %d px", dims.Size.Y, want)
	}
	if dims.Size.X >= 400 {
		t.Fatalf("anchor measured %d px wide at a 400 px constraint: it is sized to its value", dims.Size.X)
	}

	// The hit rect is 44 px centred on the 36 px control: −4..40 on the y axis.
	click(r, drive, f32.Pt(float32(dims.Size.X)/2, 38))
	if clicks != 1 {
		t.Errorf("click in the slop below the anchor: OnClick fired %d times, want 1", clicks)
	}
}

// TestAnchorPinTrailingReportsTheBoxItWasOffered is the placement seam: pinned,
// the anchor draws the same control at the same width and reports the cap it
// was handed, so a caller upstream of a pattern finds the trailing edge where
// it asked for it.
func TestAnchorPinTrailingReportsTheBoxItWasOffered(t *testing.T) {
	box := image.Pt(400, 120)
	free := materialize(t, picker.Anchor(rx.Of(liveTheme()), picker.AnchorProps{
		Value:  "Anthropic · Opus 5",
		Shaper: defaultShaper(t),
	}))
	pinned := materialize(t, picker.Anchor(rx.Of(liveTheme()), picker.AnchorProps{
		Value:  "Anthropic · Opus 5",
		Pin:    picker.PinTrailing,
		Shaper: defaultShaper(t),
	}))

	freeDims := driver(free, new(gioinput.Router), box)()
	pinnedDims := driver(pinned, new(gioinput.Router), box)()
	if freeDims.Size.X >= box.X {
		t.Fatalf("unpinned anchor measured %d px wide, want less than the %d px box", freeDims.Size.X, box.X)
	}
	if pinnedDims.Size.X != box.X {
		t.Errorf("pinned anchor measured %d px wide, want the whole %d px box", pinnedDims.Size.X, box.X)
	}
	if pinnedDims.Size.Y != freeDims.Size.Y {
		t.Errorf("pinning changed the anchor's height, %d px against %d px: a pin is a placement, not a stretch", pinnedDims.Size.Y, freeDims.Size.Y)
	}
}

// TestOpenFieldClosesOnAPressLandingElsewhere is half of what a transient
// overlay owes the window it covers: while the menu stands, the next press
// anywhere but on the field puts it away. The press below lands well under the
// last row, where nothing the field drew is.
func TestOpenFieldClosesOnAPressLandingElsewhere(t *testing.T) {
	var picked []int
	w := materialize(t, picker.Field(rx.Of(liveTheme()), picker.FieldProps{
		Options:  options,
		Shaper:   defaultShaper(t),
		OnSelect: func(_ layout.Context, i int) { picked = append(picked, i) },
	}))

	r := new(gioinput.Router)
	drive := driver(w, r, image.Pt(200, 400))
	row := rowHeight(tokens.Comfortable)

	drive()
	if dims := click(r, drive, f32.Pt(100, float32(row)/2)); dims.Size.Y != row*(1+len(options)) {
		t.Fatalf("after clicking the trigger the field measured %d px tall, want the open %d px", dims.Size.Y, row*(1+len(options)))
	}
	drive() // the absorber registers with the open menu, one frame behind

	dims := click(r, drive, f32.Pt(100, 300))
	if dims.Size.Y != row {
		t.Errorf("after a press below the menu the field measured %d px tall, want the closed %d px", dims.Size.Y, row)
	}
	if len(picked) != 0 {
		t.Errorf("a press that dismissed the menu also selected %v; dismissal is not a choice", picked)
	}
}

// TestEscapeClosesTheMenuAndGoesNoFurther is the other half, and the half a
// dialog around the field depends on: Escape with the menu open closes the
// menu and is CONSUMED, so the dialog that would also have answered it never
// sees the key. Closed, the field asks for nothing and the same key reaches
// the dialog untouched.
//
// The stand-in dialog below asks the way patterns/modal asks — a key.Filter
// for Escape, drained after its content has been laid out — so what is
// measured is the ordering the real one has.
func TestEscapeClosesTheMenuAndGoesNoFurther(t *testing.T) {
	field := materialize(t, picker.Field(rx.Of(liveTheme()), picker.FieldProps{
		Options: options,
		Shaper:  defaultShaper(t),
	}))
	var reachedTheDialog int
	w := func(gtx layout.Context) layout.Dimensions {
		dims := field(gtx)
		for {
			e, ok := gtx.Event(key.Filter{Name: key.NameEscape})
			if !ok {
				break
			}
			if ke, ok := e.(key.Event); ok && ke.State == key.Press {
				reachedTheDialog++
			}
		}
		return dims
	}

	r := new(gioinput.Router)
	drive := driver(w, r, image.Pt(200, 400))
	row := rowHeight(tokens.Comfortable)

	// Closed: the field asks for nothing and Escape is the dialog's.
	drive()
	drive()
	r.Queue(key.Event{Name: key.NameEscape, State: key.Press})
	drive()
	if reachedTheDialog != 1 {
		t.Fatalf("with the menu closed Escape reached the dialog %d times, want 1", reachedTheDialog)
	}

	// Open: the field asks first and takes it.
	if dims := click(r, drive, f32.Pt(100, float32(row)/2)); dims.Size.Y != row*(1+len(options)) {
		t.Fatalf("after clicking the trigger the field measured %d px tall, want the open %d px", dims.Size.Y, row*(1+len(options)))
	}
	drive()
	r.Queue(key.Event{Name: key.NameEscape, State: key.Press})
	dims := drive()
	if dims.Size.Y != row {
		t.Errorf("after Escape the field measured %d px tall, want the closed %d px", dims.Size.Y, row)
	}
	if reachedTheDialog != 1 {
		t.Errorf("Escape reached the dialog %d times, want the 1 it arrived with: an open menu consumes the key", reachedTheDialog)
	}
}

// TestCappedMenuOpensOnTheSelectedRow: a cap makes the menu a viewport, and a
// viewport that opens at the top of a forty-row catalogue hides the answer the
// field is already holding. The row it opens on is the selected one, which the
// pointer can then reach — a click at the leading edge of the capped plane
// picks a row near the selection rather than the first option.
func TestCappedMenuOpensOnTheSelectedRow(t *testing.T) {
	long := make([]string, 40)
	for i := range long {
		long[i] = "Option " + string(rune('A'+i%26))
	}
	var picked []int
	row := rowHeight(tokens.Comfortable)
	w := materialize(t, picker.Field(rx.Of(liveTheme()), picker.FieldProps{
		Options:   long,
		Selected:  30,
		MaxHeight: unit.Dp(row * 5),
		Shaper:    defaultShaper(t),
		OnSelect:  func(_ layout.Context, i int) { picked = append(picked, i) },
	}))

	r := new(gioinput.Router)
	drive := driver(w, r, image.Pt(200, 600))

	drive()
	dims := click(r, drive, f32.Pt(100, float32(row)/2))
	if want := row * 6; dims.Size.Y != want {
		t.Fatalf("the open capped field measured %d px tall, want the trigger plus the cap: %d px", dims.Size.Y, want)
	}

	// The first row of the viewport, which is the selected row's own band
	// once the viewport has been scrolled to it.
	click(r, drive, f32.Pt(100, float32(row)+float32(row)/2))
	if len(picked) != 1 {
		t.Fatalf("clicking the first visible row selected %v, want exactly one option", picked)
	}
	if picked[0] < 26 {
		t.Errorf("clicking the first visible row of a menu holding option 30 selected option %d; the viewport opened at the top instead of on the selection", picked[0])
	}
}

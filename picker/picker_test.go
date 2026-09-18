package picker_test

import (
	"image"
	"image/color"
	"testing"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"

	golden "github.com/vibrantgio/components/golden"
	"github.com/vibrantgio/components/internal/control"
	"github.com/vibrantgio/components/picker"
	"github.com/vibrantgio/theme/tokens"
)

// defaultShaper pins the faces every measurement here draws with, so a
// comparison is of the drawing and not of whatever fonts the machine has.
func defaultShaper(t *testing.T) *text.Shaper {
	t.Helper()
	return tokens.DefaultTypography.DeterministicShaper()
}

// sharpRadius keeps every corner square. Anti-aliased rounded corners vary
// between GPU context initialisations, and every test below compares two
// renderings pixel for pixel.
var sharpRadius = tokens.RadiusScale{}

// rowHeight is the height an option row draws at: one BodyLarge line box plus
// the density's vertical padding, floored at the density's control height.
func rowHeight(d tokens.Density) int {
	h := int(tokens.DefaultTypography.BodyLarge.LineHeight + 2*d.PaddingY)
	if floor := int(d.ControlHeight); h < floor {
		h = floor
	}
	return h
}

// triggerHeight is the height the form trigger draws at: the control height
// and nothing else, because the trigger is the platform's pop-up button and a
// pop-up is not sized by the line box it carries. It is shorter than a menu
// row, which is why the two are named apart here.
func triggerHeight(d tokens.Density) int { return int(d.ControlHeight) }

// measure lays w out at an exact size and reports what it said it used.
func measure(t *testing.T, size image.Point, w layout.Widget) layout.Dimensions {
	t.Helper()
	var ops op.Ops
	return w(layout.Context{
		Metric:      unit.Metric{PxPerDp: 1, PxPerSp: 1},
		Constraints: layout.Exact(size),
		Ops:         &ops,
	})
}

// field is RenderField in the platform's light colours at comfortable density.
func field(t *testing.T, s picker.FieldState) layout.Widget {
	t.Helper()
	return picker.RenderField(defaultShaper(t), tokens.PlatformLight, tokens.Spacing,
		sharpRadius, tokens.DefaultTypography.BodyLarge, tokens.Comfortable, s)
}

// menu is RenderMenu in the same colours at the same density.
func menu(t *testing.T, s picker.MenuState) layout.Widget {
	t.Helper()
	return picker.RenderMenu(defaultShaper(t), tokens.PlatformLight, tokens.Spacing,
		tokens.DefaultTypography.BodyLarge, tokens.Comfortable, s)
}

var options = []string{"Alpha", "Beta", "Gamma"}

// menuTop is where an open field puts its plane, measured from the trigger's
// own top edge: the held row's box centred on the trigger's, with every row
// above it laid out above. It is the arithmetic the field does, spelled once
// here so a composition test can put a standalone menu where the field puts
// one.
func menuTop(d tokens.Density, selected int) int {
	return (triggerHeight(d)-rowHeight(d))/2 - selected*rowHeight(d)
}

// planeInset is how far inside its own box a comparison reads the plane: the
// corner the field cuts the plane to plus the line it draws round it, which is
// where a rounded, antialiased outline stops being the rows' own pixels.
const planeInset = 10

// standsThere reports whether the plane itself covers a pixel of a capture
// painted on [menuCover]. That surface carries no green at all and the shadow
// the plane casts only darkens what it falls on, so any green in a pixel is
// the plane's own fill and nothing else.
func standsThere(img *image.RGBA, x, y int) bool { return px(img, x, y).G > 0 }

// TestFieldTriggerShowsTheValue is the single-choice contract in pixels: the
// closed trigger draws the option Selected names and nothing else about the
// list, so a picker holding the second of three is indistinguishable from one
// whose only option is that same string.
func TestFieldTriggerShowsTheValue(t *testing.T) {
	size := image.Pt(200, 44)
	ofThree := golden.Capture(t, size, field(t, picker.FieldState{Options: options, Selected: 1}))
	ofOne := golden.Capture(t, size, field(t, picker.FieldState{Options: []string{"Beta"}}))
	if n := golden.PixelDiff(ofThree, ofOne); n != 0 {
		t.Errorf("a closed trigger holding %q out of three differs from one holding it alone in %d pixels; the trigger shows the value and nothing else", options[1], n)
	}
}

// TestFieldTriggerDrawsThePopUpsHeight holds the trigger to the control it is
// drawn as. MEASURED, save-dialog-{light,dark}.png at 1x: the "File Format:"
// pop-up runs y 336–359, 24 px, the same number the push button beside it
// draws — a pop-up takes the control height and is not sized by the line box
// it carries, which is what the 27 px text field above it in that capture is.
// So the trigger draws 24 comfortable and 19 compact, the chrome variant's
// number in both cases, and the two variants agree. That drawn bar is the
// pointer target, which picker_live_test.go measures.
func TestFieldTriggerDrawsThePopUpsHeight(t *testing.T) {
	for _, d := range []struct {
		name string
		d    tokens.Density
	}{{"comfortable", tokens.Comfortable}, {"compact", tokens.Compact}} {
		t.Run(d.name, func(t *testing.T) {
			w := picker.RenderField(defaultShaper(t), tokens.PlatformLight, tokens.Spacing,
				tokens.Radius, tokens.DefaultTypography.BodyLarge, d.d,
				picker.FieldState{Options: options})
			dims := measure(t, image.Pt(200, 200), w)
			if want := triggerHeight(d.d); dims.Size.Y != want {
				t.Errorf("trigger height = %d px, want the density's control height %d px", dims.Size.Y, want)
			}
		})
	}
}

// TestOpenFieldStandsTheSharedMenuOverItsTrigger is this platform's pop-up
// behaviour in pixels: the open menu does not drop below the trigger, it
// stands OVER it with the row the picker is holding on the trigger's own
// label, and the rows either side of it are laid out above and below.
//
// What stands there is the SHARED menu — the same surface [picker.Menu] emits
// standing alone — so the interior of the field's plane is a standalone menu
// placed at that offset, pixel for pixel. The comparison is read inside the
// plane's own corner: the field cuts the plane to a corner and draws a line
// round it, and neither is the rows' drawing.
func TestOpenFieldStandsTheSharedMenuOverItsTrigger(t *testing.T) {
	row, trig := rowHeight(tokens.Comfortable), triggerHeight(tokens.Comfortable)
	const sel = 1
	at := row * 3
	planeY := at + menuTop(tokens.Comfortable, sel)
	planeH := row * len(options)
	size := image.Pt(200, at+trig+row*len(options))

	open := golden.Capture(t, size, laidOutAt(at, field(t, picker.FieldState{
		Open: true, Options: options, Selected: sel,
	})))
	standalone := golden.Capture(t, size, laidOutAt(planeY, menu(t, picker.MenuState{
		Options: options, Selected: sel,
	})))

	for y := planeY + planeInset; y < planeY+planeH-planeInset; y++ {
		for x := planeInset; x < size.X-planeInset; x++ {
			if a, b := px(open, x, y), px(standalone, x, y); a != b {
				t.Fatalf("(%d,%d) inside the open field's plane is %v and the shared menu standing there is %v", x, y, a, b)
			}
		}
	}
}

// TestOpenMenuStandsOnTheHeldRow reads the placement off the pixels: the plane
// begins exactly [menuTop] above the trigger's top edge, so the held row
// covers the trigger and the rows above it stand above.
func TestOpenMenuStandsOnTheHeldRow(t *testing.T) {
	row := rowHeight(tokens.Comfortable)
	const sel = 1
	at := row * 3
	planeY := at + menuTop(tokens.Comfortable, sel)
	size := image.Pt(200, at+row*(1+len(options)))

	img := golden.Capture(t, size, onCover(laidOutAt(at, field(t, picker.FieldState{
		Open: true, Options: options, Selected: sel,
	}))))
	if standsThere(img, 100, planeY-1) {
		t.Errorf("y=%d, one pixel above where the plane should start, is the plane's own fill", planeY-1)
	}
	if !standsThere(img, 100, planeY+1) {
		t.Errorf("y=%d, inside the plane's first row, is not the plane's own fill; the menu did not stand where the held row puts it", planeY+1)
	}
	if last := planeY + row*len(options); standsThere(img, 100, last) {
		t.Errorf("y=%d, one pixel past the plane's last row, is the plane's own fill", last)
	}
}

// laidOutAt lays w out y px down the frame and reports the frame, so a capture
// can place a field where a container would.
func laidOutAt(y int, w layout.Widget) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		off := op.Offset(image.Pt(0, y)).Push(gtx.Ops)
		defer off.Pop()
		w(gtx)
		return layout.Dimensions{Size: gtx.Constraints.Max}
	}
}

// onCover paints [menuCover] behind w, the surface [standsThere] reads against.
func onCover(w layout.Widget) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		paint.FillShape(gtx.Ops, menuCover, clip.Rect{Max: gtx.Constraints.Max}.Op())
		return w(gtx)
	}
}

// TestOpenFieldReportsTheTriggerAlone is the floating-surface contract in
// measurements: the menu is deferred to the end of the frame and bounded by
// the window, so it takes no room from the container the trigger stands in.
// An open field reports what the closed one reports, whichever side its Drop
// names, and a container places the two identically.
//
// The second half is why that is not simply a field that stopped drawing: the
// band past the reported box is painted. The menu still paints whole where the
// reported box no longer reaches.
func TestOpenFieldReportsTheTriggerAlone(t *testing.T) {
	row, trig := rowHeight(tokens.Comfortable), triggerHeight(tokens.Comfortable)
	closed := measure(t, image.Pt(200, 400), field(t, picker.FieldState{Options: options}))
	if closed.Size != (image.Pt(200, trig)) {
		t.Fatalf("closed field measured %v, want the trigger's %v", closed.Size, image.Pt(200, trig))
	}
	for _, d := range []struct {
		name string
		drop picker.Drop
	}{{"down", picker.DropDown}, {"up", picker.DropUp}} {
		open := measure(t, image.Pt(200, 400), field(t, picker.FieldState{
			Open: true, Drop: d.drop, Options: options,
		}))
		if open != closed {
			t.Errorf("%s, an open field measured %v against the closed field's %v; the menu floats and asks its container for no room",
				d.name, open, closed)
		}
	}

	at := row * 3
	size := image.Pt(200, at+trig+row*len(options))
	img := golden.Capture(t, size, onCover(laidOutAt(at, field(t, picker.FieldState{
		Open: true, Options: options, Selected: 1,
	}))))
	painted := false
	for y := at + trig; y < size.Y; y++ {
		if standsThere(img, 100, y) {
			painted = true
			break
		}
	}
	if !painted {
		t.Error("nothing at all was drawn past the box the open field reported; a floating menu is not bounded by it")
	}
}

// TestOpenMenusPlaneIsCorneredAndShadowed is what tells the level. The Level
// entry gives a floating surface the window background under the platform's
// shadow, and the window background is what the rows themselves fill: nothing
// but the shadow parts the menu from the window behind it. The corner is the
// plane's own, so the square pixel at its corner is not the plane.
func TestOpenMenusPlaneIsCorneredAndShadowed(t *testing.T) {
	row := rowHeight(tokens.Comfortable)
	const sel = 1
	at := row * 4
	planeY := at + menuTop(tokens.Comfortable, sel)
	size := image.Pt(200, at+row*(2+len(options)))

	img := golden.Capture(t, size, onCover(laidOutAt(at, field(t, picker.FieldState{
		Open: true, Options: options, Selected: sel,
	}))))
	if standsThere(img, 0, planeY) {
		t.Error("the plane's own corner pixel is the plane's fill; the plane is not cornered")
	}
	if !standsThere(img, planeInset, planeY+planeInset) {
		t.Fatal("the plane is not painted where it should be; this measures nothing")
	}
	// Two pixels out from the plane's foot, where nothing of the plane stands
	// and the shadow is at its deepest.
	foot := planeY + row*len(options) + 2
	shadowed, clear := px(img, 100, foot), px(img, 100, size.Y-1)
	if shadowed == clear {
		t.Errorf("y=%d, just past the plane's foot, is %v — the same as the surface at %v; the floating plane casts no shadow",
			foot, shadowed, clear)
	}
	if shadowed.R >= clear.R {
		t.Errorf("y=%d reads %v against the surface's %v; the shadow does not darken what it falls on", foot, shadowed, clear)
	}
}

// TestCappedMenuIsTheCapAndScrolls: a cap is a plane height, so a menu given
// one draws exactly that and no more however many options it holds, while the
// same cap over a short list changes nothing — the rows are still shorter than
// it.
func TestCappedMenuIsTheCapAndScrolls(t *testing.T) {
	row := rowHeight(tokens.Comfortable)
	long := make([]string, 40)
	for i := range long {
		long[i] = "Option " + string(rune('A'+i%26))
	}
	cap := unit.Dp(row * 5)

	capped := measure(t, image.Pt(200, 2000), menu(t, picker.MenuState{Options: long, MaxHeight: cap}))
	if want := row * 5; capped.Size.Y != want {
		t.Errorf("a %d-row menu capped at %d px measured %d px tall, want the cap", len(long), want, capped.Size.Y)
	}
	uncapped := measure(t, image.Pt(200, 2000), menu(t, picker.MenuState{Options: long}))
	if want := row * len(long); uncapped.Size.Y != want {
		t.Errorf("an uncapped %d-row menu measured %d px tall, want every row: %d px", len(long), uncapped.Size.Y, want)
	}
	short := measure(t, image.Pt(200, 2000), menu(t, picker.MenuState{Options: options, MaxHeight: cap}))
	if want := row * len(options); short.Size.Y != want {
		t.Errorf("a %d-row menu under the same cap measured %d px tall, want its own %d px: a cap is a ceiling, not a height",
			len(options), short.Size.Y, want)
	}
}

// TestCappedFieldFloatsExactlyTheCap: the cap reaches the field the same way
// it reaches the menu, so an open field over a catalogue floats a plane
// exactly the cap tall rather than one forty rows tall — and measures its
// trigger either way, because the plane is not the field's to make room for.
//
// A capped plane is a viewport and is placed by its room rather than by the
// held row; told no room, it takes the trigger's own edges, so a DropDown cap
// begins at the trigger's top. The extent is read off the pixels, which is
// where a floating plane's height is visible at all.
func TestCappedFieldFloatsExactlyTheCap(t *testing.T) {
	row, trig := rowHeight(tokens.Comfortable), triggerHeight(tokens.Comfortable)
	long := catalogue(40)
	cap := unit.Dp(row * 5)
	dims := measure(t, image.Pt(200, 2000), field(t, picker.FieldState{
		Open: true, Options: long, MaxHeight: cap,
	}))
	if dims.Size.Y != trig {
		t.Errorf("an open capped field measured %d px tall, want the trigger's %d px", dims.Size.Y, trig)
	}

	at := row
	size := image.Pt(200, at+row*7)
	img := golden.Capture(t, size, onCover(laidOutAt(at, field(t, picker.FieldState{
		Open: true, Options: long, MaxHeight: cap,
	}))))
	if last := at + row*5 - 1; !standsThere(img, 100, last) {
		t.Errorf("y=%d, the last row inside the cap, was left unpainted; the plane is shorter than the cap it was given", last)
	}
	if past := at + row*5; standsThere(img, 100, past) {
		t.Errorf("y=%d, one pixel past the cap, is the plane's own fill; the plane is taller than its cap", past)
	}
}

// TestTriggerDrawsItsPromptApartFromItsValue is the empty-state contract: a
// field holding no value says so, in the prompt's own foreground, and the two prompts
// are two sentences — a field with options and none picked asks the reader to
// choose, and one with no options reports that there is nothing to choose.
// None of the three drawings may be the same image.
func TestTriggerDrawsItsPromptApartFromItsValue(t *testing.T) {
	size := image.Pt(200, 44)
	picked := golden.Capture(t, size, field(t, picker.FieldState{
		Options: options, Selected: 1, Placeholder: "Choose one…", NoOptions: "Nothing to pick",
	}))
	unpicked := golden.Capture(t, size, field(t, picker.FieldState{
		Options: options, Selected: -1, Placeholder: "Choose one…", NoOptions: "Nothing to pick",
	}))
	empty := golden.Capture(t, size, field(t, picker.FieldState{
		Selected: -1, Placeholder: "Choose one…", NoOptions: "Nothing to pick",
	}))
	if n := golden.PixelDiff(picked, unpicked); n == 0 {
		t.Error("a field holding a value draws the same image as one holding none")
	}
	if n := golden.PixelDiff(unpicked, empty); n == 0 {
		t.Error("a field asking the reader to choose draws the same image as one with nothing to choose")
	}

	// The prompt is drawn in the prompt's foreground, not the body's: the same
	// wording as a VALUE is a different image.
	asValue := golden.Capture(t, size, field(t, picker.FieldState{
		Options: []string{"Choose one…"},
	}))
	if n := golden.PixelDiff(unpicked, asValue); n == 0 {
		t.Error("a prompt is drawn in the same colour as a value; an unanswered field reads as answered")
	}
}

// TestTheTriggersMarkIsSteadyWhicheverWayItsMenuOpens is the platform's mark
// written down where a future change cannot silently undo it. A pop-up
// button's mark is a pair of chevrons pointing opposite ways — MEASURED,
// save-dialog-{light,dark}.png, the "File Format:" pop-up's x 435–442, upper
// y 343–347 and lower y 349–353 — and a pair pointing both ways cannot say a
// direction. So a field that drops upwards draws the downward field's trigger
// pixel for pixel, and the reader learns the direction from the menu rather
// than from the closed control.
func TestTheTriggersMarkIsSteadyWhicheverWayItsMenuOpens(t *testing.T) {
	size := image.Pt(200, 44)
	down := golden.Capture(t, size, field(t, picker.FieldState{Options: options, Selected: 1}))
	up := golden.Capture(t, size, field(t, picker.FieldState{Options: options, Selected: 1, Drop: picker.DropUp}))
	if n := golden.PixelDiff(down, up); n != 0 {
		t.Errorf("an upward field's closed trigger differs from a downward one's in %d pixels; the pop-up's mark says nothing about the direction and nothing else on the trigger may either", n)
	}
}

// TestMenuWithNoOptionsIsNoSurface: an empty menu is not an empty plane, it is
// nothing at all, so a caller that opens one with nothing to offer paints no
// overlay over its content.
func TestMenuWithNoOptionsIsNoSurface(t *testing.T) {
	if dims := measure(t, image.Pt(200, 200), menu(t, picker.MenuState{})); dims.Size != (image.Point{}) {
		t.Errorf("menu with no options measured %v, want the zero size", dims.Size)
	}
}

// TestMenuSelectedRowIsDrawnApartFromTheRest: the row the picker is holding
// wears the pill and carries the check, so which row is held changes the
// drawing.
func TestMenuSelectedRowIsDrawnApartFromTheRest(t *testing.T) {
	size := image.Pt(200, rowHeight(tokens.Comfortable)*len(options))
	first := golden.Capture(t, size, menu(t, picker.MenuState{Options: options, Selected: 0}))
	second := golden.Capture(t, size, menu(t, picker.MenuState{Options: options, Selected: 1}))
	none := golden.Capture(t, size, menu(t, picker.MenuState{Options: options, Selected: -1}))
	if n := golden.PixelDiff(first, second); n == 0 {
		t.Error("a menu with its first row selected renders identically to one with its second selected")
	}
	if n := golden.PixelDiff(first, none); n == 0 {
		t.Error("a menu with a selected row renders identically to one with none; an out-of-range index selects nothing")
	}
}

// TestToolbarIsSizedToItsValue: the chrome variant's trigger is a control
// around its value, not a bar across its container — it clamps to what it is
// offered and otherwise reports its own width, at the density's TOOLBAR
// control height.
func TestToolbarIsSizedToItsValue(t *testing.T) {
	w := picker.RenderToolbar(defaultShaper(t), "Anthropic · Opus 5", tokens.PlatformLight,
		tokens.Spacing, tokens.DefaultTypography.LabelLarge,
		tokens.Comfortable, picker.ToolbarState{})
	dims := measure(t, image.Pt(400, 200), w)
	if dims.Size.X >= 400 {
		t.Errorf("toolbar measured %d px wide at a 400 px constraint: a picker's trigger is sized to its value", dims.Size.X)
	}
	if want := int(tokens.Comfortable.ToolbarControlHeight); dims.Size.Y != want {
		t.Errorf("toolbar height = %d px, want the density's toolbar control height %d px", dims.Size.Y, want)
	}
}

// TestMenuMarksTheRowUnderThePointer: the pill says where the choice would
// land if the press came now, so it goes on the row under the pointer and only
// falls back to the held row when the pointer is over none. Two pills on one
// menu would be two answers to one question.
func TestMenuMarksTheRowUnderThePointer(t *testing.T) {
	size := image.Pt(200, rowHeight(tokens.Comfortable)*len(options))
	rest := golden.Capture(t, size, menu(t, picker.MenuState{Options: options, Selected: 0}))
	second := golden.Capture(t, size, menu(t, picker.MenuState{Options: options, Selected: 0, Hovered: 2}))
	third := golden.Capture(t, size, menu(t, picker.MenuState{Options: options, Selected: 0, Hovered: 3}))
	onSelected := golden.Capture(t, size, menu(t, picker.MenuState{Options: options, Selected: 0, Hovered: 1}))

	if n := golden.PixelDiff(rest, second); n == 0 {
		t.Error("a menu with its second row hovered renders identically to one with no row hovered")
	}
	if n := golden.PixelDiff(second, third); n == 0 {
		t.Error("hovering the second row renders identically to hovering the third; hover marks a row, not the menu")
	}
	if n := golden.PixelDiff(rest, onSelected); n != 0 {
		t.Errorf("hovering the held row changed %d pixels; the pill is one drawing and the held row already wears it", n)
	}
}

// TestTheCheckStandsBesideTheHeldRowAndNowhereElse: the check says which row
// the picker is holding, which is a different question from where the pointer
// is. It stands in the pill's own leading column, it moves only when the held
// row moves, and a menu holding nothing draws none at all. Nothing else in the
// row moves with it — the labels stand in one line whichever row is held.
func TestTheCheckStandsBesideTheHeldRowAndNowhereElse(t *testing.T) {
	row := rowHeight(tokens.Comfortable)
	size := image.Pt(200, row*len(options))
	// The check's own column: the pill's inset from the plane, and the box
	// the mark is drawn in.
	const markLead, markBox = 10, 16

	// One pointer position throughout — the third row — so the pill stands in
	// the same place in all three and every difference is the check's.
	held0 := golden.Capture(t, size, menu(t, picker.MenuState{Options: options, Selected: 0, Hovered: 3}))
	held1 := golden.Capture(t, size, menu(t, picker.MenuState{Options: options, Selected: 1, Hovered: 3}))
	none := golden.Capture(t, size, menu(t, picker.MenuState{Options: options, Selected: -1, Hovered: 3}))

	// differs reports whether two captures differ anywhere in the column
	// [from,to) of row r, by more than the one 255th two renderings of the
	// same shape can land apart on: the rasterizer answers a pill's flat
	// interior a level either way depending on what else the frame drew, and
	// a level is below any reading the reference records.
	differs := func(a, b *image.RGBA, r, from, to int) bool {
		apart := func(p, q uint8) bool { d := int(p) - int(q); return d > 1 || d < -1 }
		for x := from; x < to; x++ {
			for y := r * row; y < (r+1)*row; y++ {
				u, v := px(a, x, y), px(b, x, y)
				if apart(u.R, v.R) || apart(u.G, v.G) || apart(u.B, v.B) {
					return true
				}
			}
		}
		return false
	}
	mark := func(a, b *image.RGBA, r int) bool { return differs(a, b, r, markLead, markLead+markBox) }

	if !mark(held0, none, 0) {
		t.Error("the first row's mark column is the same held and unheld; no check is drawn beside the held row")
	}
	if !mark(held1, none, 1) {
		t.Error("the second row's mark column is the same held and unheld")
	}
	if mark(held1, none, 0) {
		t.Error("the first row carries a mark while the picker is holding the second")
	}
	if mark(held0, none, 1) {
		t.Error("the second row carries a mark while the picker is holding the first")
	}
	if mark(held0, none, 2) || mark(held1, none, 2) {
		t.Error("the row under the pointer carries a mark; the pill says where the press would land and the check says what is held")
	}
	// Everything past the mark's own box is one drawing in all three.
	for r := range options {
		if differs(held0, none, r, markLead+markBox, size.X) || differs(held1, none, r, markLead+markBox, size.X) {
			t.Errorf("row %d draws differently past the mark's box; the check moved the label beside it", r)
		}
	}
}

// ---- Paint-order tests ----
//
// The defect these cover (feeds, 2026-09-06, on the popover): a floating
// surface drawn inline in its anchor's own paint order is painted over by
// every sibling the window lays out after that anchor's slot. The field's
// dropped menu is such a surface and takes the same idiom.

// menuCover is the opaque sibling painted after the field, over everything
// below the trigger's own band — the way a shell paints its main column after
// the row the field stands in. It carries no green at all, which is what
// [standsThere] reads it by.
var menuCover = color.NRGBA{R: 255, G: 0, B: 255, A: 255}

// coveredField lays the field out at y and then, when covered, paints
// menuCover over everything below the trigger's band.
func coveredField(w layout.Widget, bg color.NRGBA, y, triggerH int, covered bool) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		paint.FillShape(gtx.Ops, bg, clip.Rect{Max: gtx.Constraints.Max}.Op())
		laidOutAt(y, w)(gtx)
		if covered {
			off := op.Offset(image.Pt(0, y+triggerH)).Push(gtx.Ops)
			paint.FillShape(gtx.Ops, menuCover, clip.Rect{
				Max: image.Pt(gtx.Constraints.Max.X, gtx.Constraints.Max.Y-y-triggerH),
			}.Op())
			off.Pop()
		}
		return layout.Dimensions{Size: gtx.Constraints.Max}
	}
}

// TestOpenMenuIsWholeOverALaterSibling is the paint-order contract: the open
// menu stands above the sibling laid out after the field, so every pixel of
// its plane is the one it draws with nothing over it at all.
func TestOpenMenuIsWholeOverALaterSibling(t *testing.T) {
	row, trig := rowHeight(tokens.Comfortable), triggerHeight(tokens.Comfortable)
	const sel = 1
	at := row * 3
	planeY := at + menuTop(tokens.Comfortable, sel)
	size := image.Pt(200, at+trig+row*len(options)+40)
	w := field(t, picker.FieldState{Open: true, Selected: sel, Options: options})
	bg := color.NRGBA{R: 240, G: 240, B: 240, A: 255}

	bare := golden.Capture(t, size, coveredField(w, bg, at, trig, false))
	covered := golden.Capture(t, size, coveredField(w, bg, at, trig, true))

	if got := px(covered, size.X-1, size.Y-1); got != menuCover {
		t.Fatalf("the covering sibling did not paint: (%d,%d) is %v, want %v", size.X-1, size.Y-1, got, menuCover)
	}
	for y := planeY + planeInset; y < planeY+row*len(options)-planeInset; y++ {
		for x := planeInset; x < size.X-planeInset; x++ {
			if a, b := px(bare, x, y), px(covered, x, y); a != b {
				t.Fatalf("(%d,%d) inside the menu is %v with a later sibling painted and %v without it", x, y, b, a)
			}
		}
	}
}

// px reads one pixel as an opaque NRGBA, so a capture compares against the
// colours it was painted with.
func px(img *image.RGBA, x, y int) color.NRGBA {
	r, g, b, _ := img.At(x, y).RGBA()
	return color.NRGBA{R: uint8(r >> 8), G: uint8(g >> 8), B: uint8(b >> 8), A: 255}
}

// catalogue is an option list longer than any room a test gives it, so what
// bounds the menu is the room and never the list running out.
func catalogue(n int) []string {
	out := make([]string, n)
	for i := range out {
		out[i] = "Option " + string(rune('A'+i%26))
	}
	return out
}

// TestUpwardMenuStaysInTheRoomAbove is the fitting contract on the side the
// caller asked for: a field pinned near the top of a short container, whose
// Drop names the room above it, floats a plane no taller than that room — and
// the caller's own MaxHeight, three times that room, does not buy it back. The
// available room tightens a preference and never loosens it.
//
// The extent is read off the pixels, which is where a floating plane's height
// is visible at all: the field reports its trigger either way.
func TestUpwardMenuStaysInTheRoomAbove(t *testing.T) {
	row := rowHeight(tokens.Comfortable)
	long := catalogue(40)
	above := row * 2
	triggerY := row * 3
	size := image.Pt(200, row*8)

	img := golden.Capture(t, size, onCover(laidOutAt(triggerY, field(t, picker.FieldState{
		Open: true, Drop: picker.DropUp, Options: long, MaxHeight: unit.Dp(row * 6),
		AvailableRoom: func(layout.Context) (int, int) { return above, 0 },
	}))))
	if top := triggerY - above; standsThere(img, 100, top-1) {
		t.Errorf("y=%d, one pixel above the room the container reported, is the plane's own fill; the upward menu stands taller than the room above its trigger", top-1)
	}
	if top := triggerY - above; !standsThere(img, 100, top) {
		t.Errorf("y=%d, the top of the room above the trigger, was left unpainted; the menu is shorter than the room it was given", top)
	}
	if mid := triggerY - 1; !standsThere(img, 100, mid) {
		t.Errorf("y=%d, directly above the trigger, was left unpainted; no menu was floated at all", mid)
	}
}

// TestACappedMenuIsAnchoredByItsDrop is the fitting contract where moving the
// plane cannot help: a catalogue taller than the whole room is capped to the
// room and scrolls inside that cap, and [picker.Drop] says which end of the
// room the cap takes — DropUp the room above the trigger, DropDown the room
// below. The trigger is one drawing either way: the pop-up's mark says no
// direction.
func TestACappedMenuIsAnchoredByItsDrop(t *testing.T) {
	row, trig := rowHeight(tokens.Comfortable), triggerHeight(tokens.Comfortable)
	long := catalogue(40)
	below := row * 6
	triggerY := row
	size := image.Pt(200, row*9)
	room := func(layout.Context) (int, int) { return 0, below }

	for _, tc := range []struct {
		name       string
		drop       picker.Drop
		start, end int  // the plane's own extent in the capture
		headClear  bool // whether the pixel before the plane is clear of the trigger
	}{
		// The room runs from the trigger's top to six rows past its foot;
		// six whole rows is what fits in it. Dropping down, the plane's head
		// stands on the trigger's own last row, which this reads the same as
		// the plane, so only its foot is read there.
		{"up", picker.DropUp, triggerY, triggerY + row*6, true},
		{"down", picker.DropDown, triggerY + trig + below - row*6, triggerY + trig + below, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			img := golden.Capture(t, size, onCover(laidOutAt(triggerY, field(t, picker.FieldState{
				Open: true, Drop: tc.drop, Options: long, AvailableRoom: room,
			}))))
			if tc.headClear && standsThere(img, 100, tc.start-1) {
				t.Errorf("y=%d, one pixel before the plane, is the plane's own fill", tc.start-1)
			}
			if !standsThere(img, 100, tc.start) {
				t.Errorf("y=%d, the plane's first row, was left unpainted", tc.start)
			}
			if !standsThere(img, 100, tc.end-1) {
				t.Errorf("y=%d, the plane's last row, was left unpainted", tc.end-1)
			}
			if standsThere(img, 100, tc.end) {
				t.Errorf("y=%d, one pixel past the plane, is the plane's own fill", tc.end)
			}
			if standsThere(img, 100, triggerY-1) {
				t.Errorf("y=%d, above a trigger with no room above it, is the plane's own fill; the cap took room the container does not have", triggerY-1)
			}
		})
	}

	// The trigger is one drawing whichever end of the room the cap takes. It
	// is read off a CLOSED field, because an open one wears the held drawing.
	up := golden.Capture(t, size, onCover(laidOutAt(triggerY, field(t, picker.FieldState{
		Options: long, Drop: picker.DropUp,
	}))))
	down := golden.Capture(t, size, onCover(laidOutAt(triggerY, field(t, picker.FieldState{
		Options: long, Drop: picker.DropDown,
	}))))
	for y := triggerY; y < triggerY+trig; y++ {
		for x := 0; x < size.X; x++ {
			if a, b := px(up, x, y), px(down, x, y); a != b {
				t.Fatalf("(%d,%d) of an upward field's trigger is %v and a downward one's is %v; the trigger is one drawing whichever end the cap takes", x, y, a, b)
			}
		}
	}
}

// TestTheTriggerStandsOnThePopUpsColumnAndItsRowsOnTheFields reads off the
// drawn pixels the column a picker starts its text on, at the trigger and in
// the menu that trigger drops. They are two columns because they are two
// controls.
//
// THE TRIGGER is the platform's pop-up button. MEASURED,
// save-dialog-{light,dark}.png at 1x: the "File Format:" pop-up's fill runs
// x 264–451 with no edge column of any kind, and the first covered pixel of
// its label is x=276 — twelve columns in from the fill's own edge, five
// further than the field beside it. What is spent is the origin,
// [control.PopupLeadDp]'s eleven, and the face adds its first glyph's left
// side bearing to reach the twelfth.
//
// THE MENU'S ROWS are not pop-ups either, and they are not bare text: a row is
// read inside its pill, and the pill leads with the column the check stands
// in. So a row's label starts at the pill's own inset from the plane — the
// sidebar pill's measured 10 — plus the box the mark is drawn in (16, the
// smallest of the three sizes components/icons is drawn at) plus the text
// field's own leading inset, [control.TextLeadDp]. None of those three is read
// off an open menu, because no stored capture holds one; the capture is on the
// reference's list.
//
// The faces are pinned by DeterministicShaper, so both columns are the same on
// every machine.
func TestTheTriggerStandsOnThePopUpsColumnAndItsRowsOnTheFields(t *testing.T) {
	// The face bears one column on this word, as input's own reading of the
	// same inset does.
	const bearing = 1
	// The pill's inset and the mark's box, which the row's label follows.
	const selectionInset, markBox = 10, 16
	for _, tc := range []struct {
		name      string
		innerEdge int
		want      int
		w         layout.Widget
	}{
		{"trigger", 0, int(control.PopupLeadDp) + bearing,
			field(t, picker.FieldState{Options: []string{"Email address"}})},
		{"row", 0, selectionInset + markBox + int(control.TextLeadDp) + bearing,
			menu(t, picker.MenuState{Options: []string{"Email address"}, Selected: -1})},
	} {
		t.Run(tc.name, func(t *testing.T) {
			size := image.Pt(200, rowHeight(tokens.Comfortable))
			img := golden.Capture(t, size, tc.w)
			// A row clear of the top and bottom edges, and of the corners
			// the trigger rounds, holds the text; x=150 is clear of it and
			// of the trigger's mark.
			fill := img.RGBAAt(150, size.Y/2)
			first := -1
			for x := tc.innerEdge + 1; x < 150 && first < 0; x++ {
				for y := 4; y <= size.Y-5; y++ {
					if img.RGBAAt(x, y) != fill {
						first = x
						break
					}
				}
			}
			if first < 0 {
				t.Fatal("no text found inside the control; this measures nothing")
			}
			if got := first - tc.innerEdge; got != tc.want {
				t.Errorf("the text's first covered pixel is %d px in from the inner edge, want %d: the origin and the face's %d px side bearing",
					got, tc.want, bearing)
			}
		})
	}
}

// TestFieldStateGolden records the form trigger's states in both schemes, on
// the plane a form field stands on.
//
// The trigger is the platform's pop-up button and takes the platform's own
// pointer overlays: MEASURED, control-hover-{light,dark}.png, where the
// Finder toolbar's view pop-up under the pointer reads #f2f2f2 on the white
// band and #384146 on the #242d32 one. No capture holds a pressed pop-up, so
// the held image is the push button's press overlay
// (control-pressed-{light,dark}.png) until one is taken. Switched off, the
// whole control fades toward the plane at the platform's measured disabled
// coverage, which is what parts these four images from one another.
func TestFieldStateGolden(t *testing.T) {
	opts := []string{"Alpha", "Beta", "Gamma"}
	states := []struct {
		name string
		s    picker.FieldState
	}{
		{"rest", picker.FieldState{Options: opts, Selected: 1}},
		{"hovered", picker.FieldState{Options: opts, Selected: 1, Hovered: true}},
		{"pressed", picker.FieldState{Options: opts, Selected: 1, Pressed: true}},
		{"disabled", picker.FieldState{Options: opts, Selected: 1, Disabled: true}},
	}
	for _, sc := range goldenSchemes {
		for _, st := range states {
			name := "field-" + sc.name + "-" + st.name
			t.Run(name, func(t *testing.T) {
				w := picker.RenderField(defaultShaper(t), sc.p, tokens.Spacing,
					sharpRadius, tokens.DefaultTypography.BodyLarge, tokens.Comfortable, st.s)
				golden.Render(t, name, goldenSize, onSurface(sc.p.WindowBackground, w))
			})
		}
	}
}

// TestOpenFieldGolden records the open menu in both schemes: the plane on the
// floating level's own fill under the platform's shadow, cornered, the held
// row wearing the inset pill with its check, and the trigger under it in the
// held drawing. The frame carries room above and below the control, because
// the menu stands over the trigger rather than beside it, and the picture is
// only the picture if the shadow has somewhere to fall.
func TestOpenFieldGolden(t *testing.T) {
	opts := []string{"Alpha", "Beta", "Gamma"}
	row := rowHeight(tokens.Comfortable)
	size := image.Pt(240, row*(2+len(opts)))
	for _, sc := range goldenSchemes {
		name := "field-" + sc.name + "-open"
		t.Run(name, func(t *testing.T) {
			w := picker.RenderField(defaultShaper(t), sc.p, tokens.Spacing,
				sharpRadius, tokens.DefaultTypography.BodyLarge, tokens.Comfortable,
				picker.FieldState{Open: true, Options: opts, Selected: 1})
			golden.Render(t, name, size, onSurface(sc.p.WindowBackground,
				laidOutAt(row*2, inset(20, w))))
		})
	}
}

// inset lays w out w px in from either end of the frame, so a plane that
// reaches the frame's own edges cannot be told from one that was clipped by
// them.
func inset(w int, child layout.Widget) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		off := op.Offset(image.Pt(w, 0)).Push(gtx.Ops)
		defer off.Pop()
		gtx.Constraints.Max.X -= 2 * w
		return child(gtx)
	}
}

// The four state images are four images: a trigger that answered the same
// pixels under the pointer, held, and switched off would be reporting states
// it does not draw.
func TestTheFormTriggersStatesAreApart(t *testing.T) {
	opts := []string{"Alpha", "Beta", "Gamma"}
	frame := func(s picker.FieldState) *image.RGBA {
		return golden.Capture(t, goldenSize, onSurface(tokens.PlatformLight.WindowBackground,
			picker.RenderField(defaultShaper(t), tokens.PlatformLight, tokens.Spacing,
				sharpRadius, tokens.DefaultTypography.BodyLarge, tokens.Comfortable, s)))
	}
	rest := frame(picker.FieldState{Options: opts, Selected: 1})
	if rest == nil {
		return // headless unavailable; Capture called t.Skip
	}
	for _, st := range []struct {
		name string
		s    picker.FieldState
	}{
		{"hovered", picker.FieldState{Options: opts, Selected: 1, Hovered: true}},
		{"pressed", picker.FieldState{Options: opts, Selected: 1, Pressed: true}},
		{"disabled", picker.FieldState{Options: opts, Selected: 1, Disabled: true}},
	} {
		if n := golden.PixelDiff(rest, frame(st.s)); n == 0 {
			t.Errorf("a %s trigger is pixel-identical to a resting one", st.name)
		}
	}
}

// A press wins over a hover on the trigger too: the two overlays are one
// answer and are never laid on each other.
func TestTheTriggersPressWinsOverItsHover(t *testing.T) {
	opts := []string{"Alpha", "Beta", "Gamma"}
	frame := func(s picker.FieldState) *image.RGBA {
		return golden.Capture(t, goldenSize, onSurface(tokens.PlatformLight.WindowBackground,
			picker.RenderField(defaultShaper(t), tokens.PlatformLight, tokens.Spacing,
				sharpRadius, tokens.DefaultTypography.BodyLarge, tokens.Comfortable, s)))
	}
	held := frame(picker.FieldState{Options: opts, Selected: 1, Pressed: true})
	if held == nil {
		return // headless unavailable; Capture called t.Skip
	}
	both := frame(picker.FieldState{Options: opts, Selected: 1, Pressed: true, Hovered: true})
	if n := golden.PixelDiff(held, both); n != 0 {
		t.Errorf("a held trigger under the pointer moved %d pixels off the held one", n)
	}
}

// TestCompactTriggerKeepsItsLabelInside holds the Compact box. BodyLarge's
// line box is 24 dp at every density and the platform's small control is
// 19 px (controls.md's small control rows: the small push button's published
// 19 pt, which is what CompactControlHeight carries — no stored capture holds
// a small pop-up, and one is on the reference's capture list). The role's SIZE
// does not move: its cap band measures 12 px and stands inside 19 with room
// either side. What is cut is the leading the line box carries around that
// band, and the trigger clips what it draws to its own shape, so nothing it
// paints falls outside the box it reports.
func TestCompactTriggerKeepsItsLabelInside(t *testing.T) {
	const pad = 10
	surface := color.NRGBA{R: 0xff, G: 0x00, B: 0x00, A: 0xff}
	p := tokens.PlatformLight
	for _, d := range []struct {
		name string
		d    tokens.Density
	}{{"comfortable", tokens.Comfortable}, {"compact", tokens.Compact}} {
		t.Run(d.name, func(t *testing.T) {
			trigger := picker.RenderField(defaultShaper(t), p, tokens.Spacing,
				sharpRadius, tokens.DefaultTypography.BodyLarge, d.d,
				picker.FieldState{Options: options, Selected: 1})
			h := triggerHeight(d.d)
			img := golden.Capture(t, image.Pt(160+2*pad, h+2*pad), func(gtx layout.Context) layout.Dimensions {
				paint.FillShape(gtx.Ops, surface, clip.Rect{Max: gtx.Constraints.Max}.Op())
				gtx.Constraints = layout.Exact(image.Pt(160, h))
				off := op.Offset(image.Pt(pad, pad)).Push(gtx.Ops)
				dims := trigger(gtx)
				off.Pop()
				return dims
			})
			for y := 0; y < img.Bounds().Dy(); y++ {
				for x := 0; x < img.Bounds().Dx(); x++ {
					if x >= pad && x < pad+160 && y >= pad && y < pad+h {
						continue
					}
					r, g, b, _ := img.At(x, y).RGBA()
					if uint8(r>>8) != surface.R || uint8(g>>8) != surface.G || uint8(b>>8) != surface.B {
						t.Fatalf("the trigger painted (%d,%d) outside the %dx%d box it reports", x, y, 160, h)
					}
				}
			}
		})
	}
}

// TestBothTriggersDrawOneMark holds the two variants to one drawing at one
// size. MEASURED: the Save dialog's pop-up and the Finder toolbar's both draw
// the pair 8 px wide and 11 px tall, in controls 24 px and 36 px tall, so the
// mark is sized by its point size and not by the control it stands in. The
// form trigger's mark and the chrome trigger's are therefore the same pixels,
// and the chrome trigger no longer draws the pull-down's single chevron:
// a picker is single-choice by contract and has no pull-down purpose.
func TestBothTriggersDrawOneMark(t *testing.T) {
	const box = 40
	p := tokens.PlatformLight
	// Both triggers on one fill, so the mark's own antialiasing lands on the
	// same background in both crops: the chrome variant's fill is what the
	// form variant is asked to stand on.
	var drawn image.Point
	var under uint8
	crop := func(w layout.Widget, fill color.NRGBA) image.Image {
		under = fill.R
		return golden.Capture(t, image.Pt(200, box), func(gtx layout.Context) layout.Dimensions {
			paint.FillShape(gtx.Ops, fill, clip.Rect{Max: gtx.Constraints.Max}.Op())
			dims := w(gtx)
			drawn = dims.Size
			return dims
		})
	}
	// The mark is the last thing in the control, so the rightmost columns of
	// what the control drew hold it and nothing else.
	marks := func(img image.Image) image.Rectangle {
		bounds := image.Rectangle{Min: image.Pt(1<<30, 1<<30)}
		for y := 0; y < drawn.Y; y++ {
			for x := drawn.X - 20; x < drawn.X; x++ {
				r, _, _, _ := img.At(x, y).RGBA()
				if uint8(r>>8) < under-40 {
					bounds = bounds.Union(image.Rect(x, y, x+1, y+1))
				}
			}
		}
		return bounds
	}

	fieldW := picker.RenderField(defaultShaper(t), p, tokens.Spacing, sharpRadius,
		tokens.DefaultTypography.BodyLarge, tokens.Comfortable,
		picker.FieldState{Options: options, Selected: 1})
	toolbarW := picker.RenderToolbar(defaultShaper(t), options[1], p,
		tokens.Spacing, tokens.DefaultTypography.LabelLarge, tokens.Comfortable,
		picker.ToolbarState{})

	fb := marks(crop(fieldW, p.PushButtonFill))
	tb := marks(crop(toolbarW, p.ToolbarControlFill))
	if fb.Dx() != tb.Dx() || fb.Dy() != tb.Dy() {
		t.Errorf("the form trigger's mark measures %dx%d and the chrome trigger's %dx%d; one mark, one size",
			fb.Dx(), fb.Dy(), tb.Dx(), tb.Dy())
	}
	if want := int(control.MarkHDp); fb.Dy() != want {
		t.Errorf("the mark measures %d px tall, want the measured %d", fb.Dy(), want)
	}
	if want := int(control.MarkWDp); fb.Dx() != want {
		t.Errorf("the mark measures %d px wide, want the measured %d", fb.Dx(), want)
	}
}

// TestDisabledTriggerFadesTowardItsStatedSurface: a switched-off control fades
// toward what it stands on at the platform's measured coverage, so a caller
// that put the field on a card or a coloured fill has to be able to say so —
// every field in this library states its surface. An unstated surface is the
// window's own plane, which is what the fade landed on before the property
// existed, so the default drawing does not move.
func TestDisabledTriggerFadesTowardItsStatedSurface(t *testing.T) {
	p := tokens.PlatformLight
	size := image.Pt(160, triggerHeight(tokens.Comfortable))
	render := func(s picker.FieldState) *image.RGBA {
		return golden.Capture(t, size, picker.RenderField(defaultShaper(t), p,
			tokens.Spacing, sharpRadius, tokens.DefaultTypography.BodyLarge,
			tokens.Comfortable, s))
	}
	unstated := render(picker.FieldState{Options: options, Selected: 1, Disabled: true})
	onPlane := render(picker.FieldState{Options: options, Selected: 1, Disabled: true, Surface: p.WindowBackground})
	if n := golden.PixelDiff(unstated, onPlane); n != 0 {
		t.Errorf("an unstated surface differs from the window's plane in %d pixels; the plane is what an unstated surface means", n)
	}
	onCard := render(picker.FieldState{Options: options, Selected: 1, Disabled: true, Surface: p.SidebarSelection})
	if n := golden.PixelDiff(unstated, onCard); n == 0 {
		t.Error("a disabled trigger renders the same on the window's plane and on the sidebar's pill; the fade does not reach the surface it stands on")
	}
	enabled := render(picker.FieldState{Options: options, Selected: 1, Surface: p.SidebarSelection})
	if n := golden.PixelDiff(onCard, enabled); n == 0 {
		t.Error("a disabled trigger renders identically to an enabled one on the same surface")
	}
}
